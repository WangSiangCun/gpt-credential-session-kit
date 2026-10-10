package accountmarket

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type Service struct {
	db        *sql.DB
	admin     service.AdminService
	tester    accountTester
	oauth     *service.OpenAIOAuthService
	quota     accountMarketQuotaService
	recoverer accountMarketAccountRecoverer
}

type accountTester interface {
	RunTestBackground(context.Context, int64, string) (*service.ScheduledTestResult, error)
}

type accountMarketQuotaService interface {
	QueryUsage(context.Context, int64) (*service.OpenAIQuotaUsage, error)
	CacheResetCreditsSnapshot(context.Context, int64, *service.OpenAIRateLimitResetCredits) error
	CacheCreditsSnapshot(context.Context, int64, *service.OpenAIQuotaUsage) error
	ResetCredit(context.Context, int64) (*service.OpenAIQuotaResetResult, error)
	CachePostResetSnapshot(context.Context, int64, *service.OpenAIQuotaUsage) error
}

type accountMarketAccountRecoverer interface {
	RecoverAccountState(context.Context, int64, service.AccountRecoveryOptions) (*service.SuccessfulTestRecoveryResult, error)
}

// Account-market OAuth probing follows the model accepted by the current
// ChatGPT subscription routing used by this deployment.
const accountMarketOpenAIOAuthTestModel = "gpt-5.6-terra"

func NewService(db *sql.DB, admin service.AdminService, tester *service.AccountTestService, oauth *service.OpenAIOAuthService, quota accountMarketQuotaService, recoverer accountMarketAccountRecoverer) *Service {
	return &Service{db: db, admin: admin, tester: tester, oauth: oauth, quota: quota, recoverer: recoverer}
}

func (s *Service) GenerateOAuthURL(ctx context.Context, proxyID *int64, redirectURI string) (*service.OpenAIAuthURLResult, error) {
	if s.oauth == nil {
		return nil, appError(http.StatusServiceUnavailable, "ACCOUNT_MARKET_OAUTH_UNAVAILABLE", "OAuth 登录暂不可用")
	}
	return s.oauth.GenerateAuthURL(ctx, proxyID, redirectURI, "openai")
}

func (s *Service) ExchangeOAuthCode(ctx context.Context, input *service.OpenAIExchangeCodeInput) (*service.OpenAITokenInfo, error) {
	if s.oauth == nil {
		return nil, appError(http.StatusServiceUnavailable, "ACCOUNT_MARKET_OAUTH_UNAVAILABLE", "OAuth 登录暂不可用")
	}
	return s.oauth.ExchangeCode(ctx, input)
}

func (s *Service) GetConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	var groupID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT enabled, target_group_id, commission_percent,
		       minimum_withdrawal, withdrawal_review_required
		FROM account_market_config WHERE id = 1
	`).Scan(&cfg.Enabled, &groupID, &cfg.CommissionPercent, &cfg.MinimumWithdrawal, &cfg.WithdrawalReviewRequired)
	if err != nil {
		return nil, err
	}
	if groupID.Valid {
		cfg.TargetGroupID = &groupID.Int64
		if group, groupErr := s.admin.GetGroup(ctx, groupID.Int64); groupErr == nil {
			cfg.TargetGroupName = group.Name
		}
	}
	return &cfg, nil
}

func (s *Service) UpdateConfig(ctx context.Context, cfg Config) (*Config, error) {
	if cfg.CommissionPercent < 0 || cfg.CommissionPercent > 100 {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_BAD_COMMISSION", "抽佣比例必须在 0% 到 100% 之间")
	}
	if cfg.MinimumWithdrawal < 0 {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_BAD_MINIMUM", "最低提现金额不能小于 0")
	}
	if cfg.TargetGroupID != nil {
		group, err := s.admin.GetGroup(ctx, *cfg.TargetGroupID)
		if err != nil {
			return nil, err
		}
		if group.Status != "active" {
			return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_GROUP_INACTIVE", "目标分组必须处于启用状态")
		}
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE account_market_config
		SET enabled=$1, target_group_id=$2, commission_percent=$3,
		    minimum_withdrawal=$4, withdrawal_review_required=$5, updated_at=NOW()
		WHERE id=1
	`, cfg.Enabled, cfg.TargetGroupID, cfg.CommissionPercent, cfg.MinimumWithdrawal, cfg.WithdrawalReviewRequired)
	if err != nil {
		return nil, err
	}
	return s.GetConfig(ctx)
}

func (s *Service) ListSelectableProxies(ctx context.Context) ([]ProxyOption, error) {
	proxies, err := s.admin.GetAllProxies(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	result := make([]ProxyOption, 0, len(proxies))
	for i := range proxies {
		proxy := &proxies[i]
		if !proxy.IsActive() || proxy.IsExpired(now) {
			continue
		}
		result = append(result, ProxyOption{
			ID: proxy.ID, Name: proxy.Name, Protocol: proxy.Protocol, ExpiresAt: proxy.ExpiresAt,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *Service) validateSelectedProxy(ctx context.Context, proxyID *int64) error {
	if proxyID == nil {
		return nil
	}
	proxy, err := s.admin.GetProxy(ctx, *proxyID)
	if err != nil {
		return appError(http.StatusBadRequest, "ACCOUNT_MARKET_PROXY_NOT_FOUND", "所选代理不存在，请重新选择")
	}
	if !proxy.IsActive() || proxy.IsExpired(time.Now()) {
		return appError(http.StatusBadRequest, "ACCOUNT_MARKET_PROXY_UNAVAILABLE", "所选代理当前不可用，请重新选择")
	}
	return nil
}

func (s *Service) Upload(ctx context.Context, userID int64, input UploadInput, selectedProxyID *int64) (*UploadResult, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, appError(http.StatusForbidden, "ACCOUNT_MARKET_DISABLED", "账号上传收益功能当前未开启")
	}
	if cfg.TargetGroupID == nil {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_GROUP_NOT_CONFIGURED", "管理员尚未配置上传分组，请联系管理员")
	}
	group, err := s.admin.GetGroup(ctx, *cfg.TargetGroupID)
	if err != nil {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_GROUP_MISSING", "上传分组不存在，请联系管理员")
	}
	if group.Status != "active" {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_GROUP_INACTIVE", "上传分组已停用，请联系管理员")
	}
	input.Platform = strings.TrimSpace(strings.ToLower(input.Platform))
	input.Type = strings.TrimSpace(strings.ToLower(input.Type))
	if input.Platform == "" || input.Type == "" || len(input.Credentials) == 0 {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_ACCOUNT", "账号缺少 platform、type 或 credentials")
	}
	if group.Platform != input.Platform {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_PLATFORM_MISMATCH", fmt.Sprintf("当前上传分组仅支持 %s 账号", group.Platform))
	}
	if input.Type != "oauth" && input.Type != "apikey" && input.Type != "setup-token" {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_TYPE", "仅支持上传独立 OAuth 或 API 凭据账号")
	}
	if s.tester == nil {
		return nil, appError(http.StatusServiceUnavailable, "ACCOUNT_MARKET_TEST_UNAVAILABLE", "账号自动检测暂不可用，请稍后重试")
	}
	if input.Concurrency <= 0 {
		input.Concurrency = 1
	}
	if input.Concurrency > 10000 {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_BAD_CONCURRENCY", "并发容量必须在 1 到 10000 之间")
	}
	if key := contributionForbiddenKey(input.Credentials, input.Extra); key != "" {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_OPERATOR_FIELD", fmt.Sprintf("上传账号不能设置管理员运行字段 %s", key))
	}
	if err := s.validateSelectedProxy(ctx, selectedProxyID); err != nil {
		return nil, err
	}

	email := extractAccountEmail(input.Credentials, input.Extra)
	emailHash := ""
	if email != "" {
		emailHash = digest(strings.ToLower(email))
	}
	fingerprint := credentialFingerprint(input.Platform, input.Type, input.Credentials)
	duplicate, err := s.isDuplicate(ctx, input.Platform, input.Type, email, emailHash, fingerprint)
	if err != nil {
		return nil, err
	}
	if duplicate {
		return nil, appError(http.StatusConflict, "ACCOUNT_MARKET_DUPLICATE", "该账号已经存在，不能重复上传")
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = email
		if input.Name == "" {
			input.Name = input.Platform + "-contributed"
		}
	}

	initiallySchedulable := false
	created, err := s.admin.CreateAccount(ctx, &service.CreateAccountInput{
		Name:        input.Name,
		Notes:       input.Notes,
		Platform:    input.Platform,
		Type:        input.Type,
		Credentials: input.Credentials,
		Extra:       input.Extra,
		// Only the separately validated panel selection may bind an internal proxy.
		// Imported JSON still cannot inject routing or scheduling controls.
		ProxyID:               selectedProxyID,
		Concurrency:           input.Concurrency,
		Priority:              50,
		RateMultiplier:        nil,
		LoadFactor:            nil,
		GroupIDs:              []int64{*cfg.TargetGroupID},
		SkipMixedChannelCheck: true,
		Schedulable:           &initiallySchedulable,
	})
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO account_market_accounts
		(account_id, uploader_user_id, target_group_id, platform, account_type, account_email,
		 account_email_hash, credential_fingerprint, status, activated_at, last_synced_at)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),$8,'pending',NULL,NOW())
	`, created.ID, userID, *cfg.TargetGroupID, input.Platform, input.Type, email, emailHash, fingerprint)
	if err != nil {
		// The account was created only for this upload. Remove it when ownership
		// registration loses a duplicate race, so no orphan enters scheduling.
		if cleanupErr := s.cleanupUpload(created.ID); cleanupErr != nil {
			return nil, fmt.Errorf("register contributed account: %w; cleanup account %d: %v", err, created.ID, cleanupErr)
		}
		if isUniqueViolation(err) {
			return nil, appError(http.StatusConflict, "ACCOUNT_MARKET_DUPLICATE", "该账号已经存在，不能重复上传")
		}
		return nil, err
	}
	testModel := ""
	if input.Platform == "openai" && input.Type == "oauth" {
		testModel = accountMarketOpenAIOAuthTestModel
	}
	testResult, testErr := s.tester.RunTestBackground(ctx, created.ID, testModel)
	if testErr != nil || testResult == nil || testResult.Status != "success" {
		message := "账号连接测试失败"
		if testErr != nil {
			message = testErr.Error()
		} else if testResult != nil && strings.TrimSpace(testResult.ErrorMessage) != "" {
			message = testResult.ErrorMessage
		}
		if cleanupErr := s.cleanupUpload(created.ID); cleanupErr != nil {
			return nil, fmt.Errorf("account test failed: %s; cleanup account %d: %w", message, created.ID, cleanupErr)
		}
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_ACCOUNT_TEST_FAILED", message)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE account_market_accounts SET status='paused', activated_at=NOW(), updated_at=NOW() WHERE account_id=$1 AND status='pending'`, created.ID)
	if err == nil {
		var affected int64
		affected, err = result.RowsAffected()
		if err == nil && affected != 1 {
			err = appError(http.StatusConflict, "ACCOUNT_MARKET_STATE_CONFLICT", "账号状态已发生变化，请刷新后重试")
		}
	}
	if err != nil {
		return nil, errors.Join(err, s.cleanupUpload(created.ID))
	}
	if err := s.SetSchedulable(ctx, userID, created.ID, true); err != nil {
		return nil, errors.Join(err, s.cleanupUpload(created.ID))
	}
	return &UploadResult{
		Success: true, AccountID: created.ID, Name: created.Name, Email: maskEmail(email),
		GroupID: group.ID, GroupName: group.Name, Status: "active",
	}, nil
}

func (s *Service) cleanupUpload(accountID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Keep ownership until account deletion succeeds, so failed cleanup remains
	// visible and retryable rather than leaving a schedulable orphan.
	_, pauseErr := s.admin.SetAccountSchedulable(ctx, accountID, false)
	if err := s.admin.DeleteAccount(ctx, accountID); err != nil {
		return errors.Join(err, pauseErr)
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM account_market_accounts WHERE account_id=$1`, accountID)
	return err
}

func (s *Service) ListAccounts(ctx context.Context, userID int64) ([]AccountView, error) {
	if err := s.SyncEarnings(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT ama.id, ama.account_id, COALESCE(ama.account_email,''), ama.status,
		       ama.target_group_id, ama.created_at, COALESCE(SUM(ame.gross_amount),0)
		FROM account_market_accounts ama
		LEFT JOIN account_market_earnings ame
		  ON ame.account_id=ama.account_id AND ame.uploader_user_id=ama.uploader_user_id
		WHERE ama.uploader_user_id=$1 AND ama.status <> 'removed'
		GROUP BY ama.id
		ORDER BY ama.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]marketAccountRecord, 0)
	for rows.Next() {
		var record marketAccountRecord
		if err := rows.Scan(&record.marketID, &record.accountID, &record.email, &record.marketStatus, &record.groupID, &record.createdAt, &record.earned); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.hydrateAccountViews(ctx, records)
}

type AccountMarketQuotaRefreshResult struct {
	Usage                 *service.OpenAIQuotaUsage
	CachePersisted        bool
	CreditsCachePersisted bool
}

func (s *Service) RefreshOpenAIQuota(ctx context.Context, userID, accountID int64) (*AccountMarketQuotaRefreshResult, error) {
	if s.quota == nil {
		return nil, appError(http.StatusServiceUnavailable, "ACCOUNT_MARKET_QUOTA_UNAVAILABLE", "账号额度查询暂不可用")
	}
	if !s.ownsAccount(ctx, userID, accountID) {
		return nil, appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
	}
	account, err := s.admin.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.Platform != service.PlatformOpenAI || account.Type != service.AccountTypeOAuth {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_QUOTA_UNSUPPORTED", "仅支持 OpenAI OAuth 账号查询额度")
	}
	usage, err := s.quota.QueryUsage(ctx, accountID)
	if err != nil {
		return nil, err
	}
	result := &AccountMarketQuotaRefreshResult{Usage: usage}
	if err := s.quota.CacheCreditsSnapshot(ctx, accountID, usage); err == nil {
		result.CreditsCachePersisted = true
	}
	if err := s.quota.CacheResetCreditsSnapshot(ctx, accountID, usage.RateLimitResetCredits); err == nil {
		result.CachePersisted = true
	}
	service.NotifyOpenAIAutoResetCredit(accountID)
	return result, nil
}

type AccountMarketQuotaResetResult struct {
	Result                *service.OpenAIQuotaResetResult
	Quota                 *service.OpenAIQuotaUsage
	CacheRefreshed        bool
	AccountStateRecovered bool
	WarningCode           string
}

func (s *Service) ResetOpenAIQuota(ctx context.Context, userID, accountID int64) (*AccountMarketQuotaResetResult, error) {
	if s.quota == nil || s.recoverer == nil {
		return nil, appError(http.StatusServiceUnavailable, "ACCOUNT_MARKET_QUOTA_UNAVAILABLE", "账号额度重置暂不可用")
	}
	if !s.ownsAccount(ctx, userID, accountID) {
		return nil, appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
	}
	account, err := s.admin.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.Platform != service.PlatformOpenAI || account.Type != service.AccountTypeOAuth {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_QUOTA_UNSUPPORTED", "仅支持 OpenAI OAuth 账号重置额度")
	}
	reset, err := s.quota.ResetCredit(ctx, accountID)
	if err != nil {
		return nil, err
	}
	post := service.RunOpenAIQuotaResetPostProcess(ctx, accountID, s.quota, s.recoverer, s.admin.GetAccount)
	return &AccountMarketQuotaResetResult{
		Result: reset, Quota: post.Quota, CacheRefreshed: post.CacheRefreshed,
		AccountStateRecovered: post.AccountStateRecovered, WarningCode: post.WarningCode,
	}, nil
}

func (s *Service) ListAdminAccounts(ctx context.Context) ([]AccountView, error) {
	if err := s.syncAllEarnings(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT ama.id, ama.account_id, ama.uploader_user_id, COALESCE(u.email,''),
		       COALESCE(ama.account_email,''), ama.status,
		       ama.target_group_id, ama.created_at, COALESCE(SUM(ame.gross_amount),0)
		FROM account_market_accounts ama
		LEFT JOIN users u ON u.id=ama.uploader_user_id
		LEFT JOIN account_market_earnings ame ON ame.account_id=ama.account_id
		GROUP BY ama.id, u.email ORDER BY ama.created_at DESC LIMIT 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]marketAccountRecord, 0)
	for rows.Next() {
		var record marketAccountRecord
		if err := rows.Scan(
			&record.marketID, &record.accountID, &record.uploaderUserID, &record.uploaderEmail,
			&record.email, &record.marketStatus, &record.groupID, &record.createdAt, &record.earned,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.hydrateAccountViews(ctx, records)
}

type marketAccountRecord struct {
	marketID       int64
	accountID      int64
	uploaderUserID int64
	uploaderEmail  string
	email          string
	marketStatus   string
	groupID        int64
	createdAt      time.Time
	earned         float64
}

func (s *Service) hydrateAccountViews(ctx context.Context, records []marketAccountRecord) ([]AccountView, error) {
	if len(records) == 0 {
		return []AccountView{}, nil
	}
	accountIDs := make([]int64, 0, len(records))
	for _, record := range records {
		accountIDs = append(accountIDs, record.accountID)
	}
	accounts, err := s.admin.GetAccountsByIDs(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	groups, err := s.admin.GetAllGroupsIncludingInactive(ctx)
	if err != nil {
		return nil, err
	}
	proxyIDs := make([]int64, 0)
	seenProxyIDs := make(map[int64]struct{})
	for _, account := range accounts {
		if account == nil || account.ProxyID == nil {
			continue
		}
		if _, exists := seenProxyIDs[*account.ProxyID]; exists {
			continue
		}
		seenProxyIDs[*account.ProxyID] = struct{}{}
		proxyIDs = append(proxyIDs, *account.ProxyID)
	}
	proxies := make([]service.Proxy, 0, len(proxyIDs))
	if len(proxyIDs) > 0 {
		proxies, err = s.admin.GetProxiesByIDs(ctx, proxyIDs)
		if err != nil {
			return nil, err
		}
	}
	accountByID := make(map[int64]*service.Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountByID[account.ID] = account
		}
	}
	groupByID := make(map[int64]*service.Group, len(groups))
	for i := range groups {
		group := &groups[i]
		groupByID[group.ID] = group
	}
	proxyByID := make(map[int64]*service.Proxy, len(proxies))
	for i := range proxies {
		proxy := &proxies[i]
		proxyByID[proxy.ID] = proxy
	}
	result := make([]AccountView, 0, len(records))
	for _, record := range records {
		account := accountByID[record.accountID]
		if account == nil {
			continue
		}
		item := buildAccountView(
			record.marketID, record.email, record.marketStatus, record.createdAt,
			record.earned, account, groupByID[record.groupID],
		)
		if account.ProxyID != nil {
			item.ProxyID = account.ProxyID
			if proxy := proxyByID[*account.ProxyID]; proxy != nil {
				item.ProxyName = proxy.Name
				item.ProxyProtocol = proxy.Protocol
			}
		}
		item.UploaderUserID = record.uploaderUserID
		item.UploaderEmail = record.uploaderEmail
		result = append(result, item)
	}
	return result, nil
}

func (s *Service) SetSchedulable(ctx context.Context, userID, accountID int64, enabled bool) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	unlock, err := s.lockAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}
	defer unlock()
	var ready bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM account_market_accounts WHERE uploader_user_id=$1 AND account_id=$2 AND status IN ('active','paused') AND activated_at IS NOT NULL)`, userID, accountID).Scan(&ready); err != nil {
		return err
	}
	if !ready {
		return appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
	}
	account, err := s.admin.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}
	previous := account.Schedulable
	if _, err := s.admin.SetAccountSchedulable(ctx, accountID, enabled); err != nil {
		return errors.Join(err, s.restoreScheduling(accountID, previous))
	}
	status := "paused"
	if enabled {
		status = "active"
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE account_market_accounts SET status=$1, updated_at=NOW()
		WHERE account_id=$2 AND uploader_user_id=$3 AND status <> 'removed'
	`, status, accountID, userID)
	if err == nil {
		if affected, rowsErr := result.RowsAffected(); rowsErr != nil {
			err = rowsErr
		} else if affected != 1 {
			err = appError(http.StatusConflict, "ACCOUNT_MARKET_STATE_CONFLICT", "账号状态已发生变化，请刷新后重试")
		}
	}
	if err != nil && previous != enabled {
		if restoreErr := s.restoreScheduling(accountID, previous); restoreErr != nil {
			return fmt.Errorf("update contributed account state: %w; restore scheduler state: %v", err, restoreErr)
		}
	}
	return err
}

func (s *Service) restoreScheduling(accountID int64, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.admin.SetAccountSchedulable(ctx, accountID, enabled)
	return err
}

// A short lease serializes panel operations across instances without reserving
// a pool connection while repository calls need another connection. Operations
// time out after 30s; compensation takes at most 20s, below the 2m lease.
func (s *Service) lockAccount(ctx context.Context, userID, accountID int64) (func(), error) {
	token := uuid.NewString()
	var id int64
	err := s.db.QueryRowContext(ctx, `UPDATE account_market_accounts
		SET operation_token=$3, operation_expires_at=NOW()+INTERVAL '2 minutes'
		WHERE account_id=$1 AND uploader_user_id=$2 AND status <> 'removed'
		AND (operation_expires_at IS NULL OR operation_expires_at < NOW())
		RETURNING account_id`, accountID, userID, token).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appError(http.StatusConflict, "ACCOUNT_MARKET_BUSY", "账号正在处理其他操作，请稍后重试")
	}
	if err != nil {
		return nil, err
	}
	return func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.db.ExecContext(releaseCtx, `UPDATE account_market_accounts SET operation_token=NULL, operation_expires_at=NULL WHERE account_id=$1 AND operation_token=$2`, accountID, token)
	}, nil
}

func (s *Service) UpdateAccount(ctx context.Context, userID, accountID int64, input AccountUpdateRequest) (*AccountView, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	unlock, err := s.lockAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if !s.ownsAccount(ctx, userID, accountID) {
		return nil, appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
	}
	if input.Concurrency < 1 || input.Concurrency > 10000 {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_BAD_CONCURRENCY", "并发容量必须在 1 到 10000 之间")
	}
	if _, err := s.admin.UpdateAccount(ctx, accountID, &service.UpdateAccountInput{Concurrency: &input.Concurrency}); err != nil {
		return nil, err
	}
	accounts, err := s.ListAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		if accounts[i].AccountID == accountID {
			return &accounts[i], nil
		}
	}
	return nil, appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
}

func (s *Service) Remove(ctx context.Context, userID, accountID int64) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	unlock, err := s.lockAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}
	defer unlock()
	if !s.ownsAccount(ctx, userID, accountID) {
		return appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
	}
	account, err := s.admin.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}
	wasSchedulable := account.Schedulable
	if wasSchedulable {
		if _, err := s.admin.SetAccountSchedulable(ctx, accountID, false); err != nil {
			return errors.Join(err, s.restoreScheduling(accountID, wasSchedulable))
		}
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE account_market_accounts
		SET status=CASE WHEN activated_at IS NULL THEN 'pending' ELSE 'paused' END, updated_at=NOW()
		WHERE account_id=$1 AND uploader_user_id=$2 AND status <> 'removed'
	`, accountID, userID)
	if err != nil {
		if wasSchedulable {
			err = errors.Join(err, s.restoreScheduling(accountID, true))
		}
		return err
	}
	if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
		err := errors.Join(rowsErr, appError(http.StatusConflict, "ACCOUNT_MARKET_REMOVE_CONFLICT", "账号状态已发生变化，请刷新后重试"))
		if wasSchedulable {
			err = errors.Join(err, s.restoreScheduling(accountID, true))
		}
		return err
	}
	if err := s.admin.DeleteAccount(ctx, accountID); err != nil {
		restoreCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, restoreErr := s.db.ExecContext(restoreCtx, `
			UPDATE account_market_accounts
			SET status=CASE WHEN activated_at IS NULL THEN 'pending' WHEN $3 THEN 'active' ELSE 'paused' END,
			    removed_at=NULL, updated_at=NOW()
			WHERE account_id=$1 AND uploader_user_id=$2 AND status IN ('pending','paused')
		`, accountID, userID, wasSchedulable)
		if wasSchedulable && restoreErr == nil {
			restoreErr = s.restoreScheduling(accountID, true)
		}
		if restoreErr != nil {
			return fmt.Errorf("delete contributed account: %w; restore market state: %v", err, restoreErr)
		}
		return err
	}
	// Only hide ownership after deletion; an interrupted removal stays visible.
	_, err = s.db.ExecContext(ctx, `UPDATE account_market_accounts SET status='removed', removed_at=NOW(), updated_at=NOW() WHERE account_id=$1 AND uploader_user_id=$2`, accountID, userID)
	return err
}

func (s *Service) ExportAccount(ctx context.Context, userID, accountID int64) (*UploadInput, error) {
	if !s.ownsAccount(ctx, userID, accountID) {
		return nil, appError(http.StatusNotFound, "ACCOUNT_MARKET_ACCOUNT_NOT_FOUND", "账号不存在")
	}
	account, err := s.admin.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &UploadInput{
		Name: account.Name, Notes: account.Notes, Platform: account.Platform, Type: account.Type,
		Credentials: account.Credentials, Extra: account.Extra,
		Concurrency: account.Concurrency, Priority: account.Priority,
		RateMultiplier: account.RateMultiplier, LoadFactor: account.LoadFactor,
	}, nil
}

func (s *Service) GetSummary(ctx context.Context, userID int64) (*Summary, error) {
	var out Summary
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN status='available' THEN gross_amount ELSE 0 END),0),
		       COALESCE(SUM(CASE WHEN status='locked' THEN gross_amount ELSE 0 END),0),
		       COALESCE(SUM(CASE WHEN status='withdrawn' THEN gross_amount ELSE 0 END),0),
		       COALESCE(SUM(gross_amount),0)
		FROM account_market_earnings WHERE uploader_user_id=$1
	`, userID).Scan(&out.Available, &out.Locked, &out.Withdrawn, &out.Total)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM account_market_accounts ama
		WHERE uploader_user_id=$1 AND status <> 'removed'
		AND EXISTS (SELECT 1 FROM accounts a WHERE a.id=ama.account_id AND a.deleted_at IS NULL)
	`, userID).Scan(&out.AccountCount)
	return &out, err
}

func (s *Service) SyncEarnings(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO account_market_earnings
		(uploader_user_id, account_id, usage_log_id, gross_amount)
		SELECT ama.uploader_user_id, ul.account_id, ul.id, GREATEST(COALESCE(ul.actual_cost,0),0)
		FROM account_market_accounts ama
		JOIN usage_logs ul
		  ON ul.account_id=ama.account_id AND ul.created_at >= ama.submitted_at
		LEFT JOIN account_market_earnings ame ON ame.usage_log_id=ul.id
		WHERE ama.uploader_user_id=$1 AND ul.actual_cost > 0 AND ame.id IS NULL
		ON CONFLICT (usage_log_id) DO NOTHING
	`, userID)
	return err
}

func (s *Service) syncAllEarnings(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO account_market_earnings
		(uploader_user_id, account_id, usage_log_id, gross_amount)
		SELECT ama.uploader_user_id, ul.account_id, ul.id, GREATEST(COALESCE(ul.actual_cost,0),0)
		FROM account_market_accounts ama
		JOIN usage_logs ul
		  ON ul.account_id=ama.account_id AND ul.created_at >= ama.submitted_at
		LEFT JOIN account_market_earnings ame ON ame.usage_log_id=ul.id
		WHERE ul.actual_cost > 0 AND ame.id IS NULL
		ON CONFLICT (usage_log_id) DO NOTHING
	`)
	return err
}

func (s *Service) isDuplicate(ctx context.Context, platform, accountType, email, emailHash, fingerprint string) (bool, error) {
	var duplicate bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM account_market_accounts
			WHERE (
				($1 <> '' AND platform=$3 AND account_type=$4 AND account_email_hash=$1)
				OR credential_fingerprint=$2
			)
		)
	`, emailHash, fingerprint, platform, accountType).Scan(&duplicate)
	if err != nil || duplicate {
		return duplicate, err
	}
	// Also reject an account already present outside the contribution module.
	// This is an upload-time scan rather than a request-path operation, and it
	// handles email values embedded in JWTs that PostgreSQL cannot compare safely.
	normalized := strings.ToLower(strings.TrimSpace(email))
	rows, err := s.db.QueryContext(ctx, `
		SELECT credentials, extra FROM accounts
		WHERE deleted_at IS NULL AND platform=$1 AND type=$2
	`, platform, accountType)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var credentialsRaw, extraRaw []byte
		if err := rows.Scan(&credentialsRaw, &extraRaw); err != nil {
			return false, err
		}
		var credentials, extra map[string]any
		if json.Unmarshal(credentialsRaw, &credentials) != nil {
			continue
		}
		_ = json.Unmarshal(extraRaw, &extra)
		if credentialFingerprint(platform, accountType, credentials) == fingerprint {
			return true, nil
		}
		if normalized != "" && strings.ToLower(extractAccountEmail(credentials, extra)) == normalized {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Service) ownsAccount(ctx context.Context, userID, accountID int64) bool {
	var exists bool
	_ = s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM account_market_accounts
		WHERE uploader_user_id=$1 AND account_id=$2 AND status <> 'removed')
	`, userID, accountID).Scan(&exists)
	return exists
}

func buildAccountView(marketID int64, email, marketStatus string, createdAt time.Time, earned float64, account *service.Account, group *service.Group) AccountView {
	accountRate := 1.0
	if account.RateMultiplier != nil {
		accountRate = *account.RateMultiplier
	}
	groupID, groupName, groupRate := int64(0), "", 1.0
	if group != nil {
		groupID, groupName, groupRate = group.ID, group.Name, group.RateMultiplier
	}
	quota, resetAt := accountMarketQuota(account)
	return AccountView{
		MarketID: marketID, AccountID: account.ID, Name: account.Name,
		Platform: account.Platform, Type: account.Type, Email: maskEmail(email),
		MarketStatus: marketStatus, AccountStatus: account.Status, Schedulable: account.Schedulable,
		ErrorMessage: account.ErrorMessage, Concurrency: account.Concurrency,
		RateMultiplier: accountRate, GroupID: groupID, GroupName: groupName,
		GroupRateMultiplier: groupRate, Quota: quota, ResetAt: resetAt,
		LastUsedAt: account.LastUsedAt, CreatedAt: createdAt, TotalEarned: earned,
	}
}

func accountMarketQuota(account *service.Account) (any, any) {
	if account == nil {
		return nil, nil
	}
	if quota := firstExtra(account.Extra, "remaining_quota", "quota", "balance"); quota != nil {
		return quota, firstExtra(account.Extra, "reset_at", "resets_at", "codex_reset_at")
	}

	var windows []string
	appendRemaining := func(label, key string, fraction bool) {
		raw, ok := account.Extra[key]
		if !ok {
			return
		}
		used, ok := numericValue(raw)
		if !ok {
			return
		}
		if fraction {
			used *= 100
		}
		remaining := math.Max(0, math.Min(100, 100-used))
		windows = append(windows, fmt.Sprintf("%s %.0f%%", label, remaining))
	}

	switch account.Platform {
	case service.PlatformOpenAI:
		appendRemaining("5h", "codex_5h_used_percent", false)
		appendRemaining("7d", "codex_7d_used_percent", false)
		if len(windows) > 0 {
			return strings.Join(windows, " · "), firstExtra(account.Extra, "codex_5h_reset_at", "codex_7d_reset_at")
		}
	case service.PlatformAnthropic:
		appendRemaining("5h", "session_window_utilization", true)
		appendRemaining("7d", "passive_usage_7d_utilization", true)
		if len(windows) > 0 {
			reset := firstExtra(account.Extra, "passive_usage_7d_reset")
			if account.SessionWindowEnd != nil {
				reset = account.SessionWindowEnd
			}
			return strings.Join(windows, " · "), reset
		}
	}

	if limit := account.GetQuotaLimit(); limit > 0 {
		return math.Max(0, limit-account.GetQuotaUsed()), nil
	}
	return nil, firstExtra(account.Extra, "reset_at", "resets_at", "codex_reset_at")
}

func numericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func firstExtra(extra map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := extra[key]; ok {
			return value
		}
	}
	return nil
}

func appError(status int, reason, message string) error {
	return infraerrors.New(status, reason, message)
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

func roundMoney(value float64) float64 {
	return math.Round(value*1e10) / 1e10
}

func extractAccountEmail(credentials, extra map[string]any) string {
	for _, tokenKey := range []string{"access_token"} {
		if token, ok := credentials[tokenKey].(string); ok {
			if email := emailFromJWT(token); email != "" {
				return email
			}
		}
	}
	keys := []string{"email", "account_email", "user_email", "login_email"}
	for _, source := range []map[string]any{credentials, extra} {
		for _, key := range keys {
			if value, ok := source[key].(string); ok {
				value = strings.TrimSpace(value)
				if strings.Contains(value, "@") {
					return value
				}
			}
		}
	}
	return ""
}

func emailFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	for _, key := range []string{"email", "account_email", "preferred_username"} {
		if value, ok := claims[key].(string); ok {
			value = strings.TrimSpace(value)
			if strings.Contains(value, "@") {
				return value
			}
		}
	}
	if profile, ok := claims["https://api.openai.com/profile"].(map[string]any); ok {
		if email, ok := profile["email"].(string); ok && strings.Contains(email, "@") {
			return strings.TrimSpace(email)
		}
	}
	return ""
}

func credentialFingerprint(platform, accountType string, credentials map[string]any) string {
	if accountType == service.AccountTypeAPIKey {
		// API-key tests do not authenticate the optional OAuth fields in a file.
		key, _ := credentials["api_key"].(string)
		return digest(platform + "\x00" + accountType + "\x00api_key\x00" + strings.TrimSpace(key))
	}
	if identity := credentialStableIdentity(credentials); identity != "" {
		return digest(platform + "\x00" + accountType + "\x00identity\x00" + identity)
	}
	encoded, _ := json.Marshal(credentials)
	return digest(platform + "\x00" + accountType + "\x00" + string(encoded))
}

func credentialStableIdentity(credentials map[string]any) string {
	// Only the credential used by the upstream test may establish identity.
	// A separately supplied id_token or email is not proof of ownership.
	for _, tokenKey := range []string{"access_token"} {
		token, _ := credentials[tokenKey].(string)
		parts := strings.Split(token, ".")
		if len(parts) < 2 {
			continue
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}
		var claims map[string]any
		if json.Unmarshal(payload, &claims) != nil {
			continue
		}
		for _, key := range []string{"sub", "account_id", "chatgpt_account_id", "user_id"} {
			if value, ok := claims[key].(string); ok && strings.TrimSpace(value) != "" {
				return key + ":" + strings.TrimSpace(value)
			}
		}
	}
	for _, key := range []string{"access_token", "api_key", "token"} {
		if value, ok := credentials[key].(string); ok && strings.TrimSpace(value) != "" {
			return key + ":" + strings.TrimSpace(value)
		}
	}
	return ""
}

func contributionForbiddenKey(maps ...map[string]any) string {
	forbidden := map[string]struct{}{
		"base_url": {}, "api_base_urls": {}, "custom_base_url": {},
		"custom_base_url_enabled": {}, "upstream_url": {}, "endpoint_url": {},
		"headers": {}, "custom_headers": {}, "header_overrides": {}, "request_headers": {},
		"proxy_id": {}, "proxy_url": {}, "rate_multiplier": {}, "load_factor": {}, "priority": {},
		"parent_account_id": {}, "agent_private_key": {},
	}
	var walkValue func(any) string
	walk := func(source map[string]any) string { return "" }
	walk = func(source map[string]any) string {
		for key, value := range source {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if _, blocked := forbidden[normalized]; blocked {
				return key
			}
			if found := walkValue(value); found != "" {
				return found
			}
		}
		return ""
	}
	walkValue = func(value any) string {
		switch nested := value.(type) {
		case map[string]any:
			return walk(nested)
		case []any:
			for _, item := range nested {
				if found := walkValue(item); found != "" {
					return found
				}
			}
		}
		return ""
	}
	for _, source := range maps {
		if found := walk(source); found != "" {
			return found
		}
	}
	return ""
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func maskEmail(value string) string {
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		if value == "" {
			return ""
		}
		return "***"
	}
	local := []rune(parts[0])
	masked := "***"
	if len(local) > 2 {
		masked = string(local[0]) + "***" + string(local[len(local)-1])
	}
	return masked + "@" + parts[1]
}

func ParseUpload(data []byte) ([]UploadInput, error) {
	if len(data) == 0 || len(data) > MaxUploadBytes {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_FILE_SIZE", "JSON 文件大小必须在 2 MB 以内")
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_JSON", "JSON 格式不正确")
	}
	var list []UploadInput
	switch value := root.(type) {
	case []any:
		encoded, _ := json.Marshal(value)
		if err := json.Unmarshal(encoded, &list); err != nil {
			return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_JSON", "账号数据格式不正确")
		}
	case map[string]any:
		if accounts, ok := value["accounts"]; ok {
			encoded, _ := json.Marshal(accounts)
			if err := json.Unmarshal(encoded, &list); err != nil {
				return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_JSON", "accounts 数据格式不正确")
			}
		} else {
			encoded, _ := json.Marshal(value)
			var item UploadInput
			if err := json.Unmarshal(encoded, &item); err != nil {
				return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_JSON", "账号数据格式不正确")
			}
			list = []UploadInput{item}
		}
	default:
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_INVALID_JSON", "JSON 根节点必须是账号对象或账号数组")
	}
	if len(list) == 0 || len(list) > MaxBatchSize {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_BATCH_SIZE", "单次需要上传 1 到 20 个账号")
	}
	return list, nil
}
