package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const defaultIntelligenceTestPrompt = "Generate an SVG of a pelican riding a bicycle."
const (
	defaultIntelligenceTestTimeoutSeconds = 120
	minIntelligenceTestTimeoutSeconds     = 30
	maxIntelligenceTestTimeoutSeconds     = 1800
)

var intelligenceSVGPattern = regexp.MustCompile(`(?is)<svg\b[^>]*>.*?</svg>`)

type intelligenceTestConfig struct {
	Prompt                   string                             `json:"prompt"`
	FrequencyMins            int                                `json:"frequency_mins"`
	TimeoutSeconds           int                                `json:"timeout_seconds"`
	ProxyID                  *int64                             `json:"proxy_id,omitempty"`
	PublicGroupFilterEnabled bool                               `json:"public_group_filter_enabled"`
	PublicGroupIDs           []int64                            `json:"public_group_ids"`
	GroupConfigs             map[string]intelligenceGroupConfig `json:"group_configs"`
	Results                  map[string]IntelligenceTestResult  `json:"results"`
}

type intelligenceGroupConfig struct {
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

type IntelligenceTestResult struct {
	GroupID         int64     `json:"group_id"`
	Model           string    `json:"model,omitempty"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
	SVG             string    `json:"svg"`
	Status          string    `json:"status"`
	Error           string    `json:"error,omitempty"`
	TestedAt        time.Time `json:"tested_at"`
	AccountID       int64     `json:"account_id,omitempty"`
}

type intelligenceTestGroup struct {
	ID                        int64    `json:"id"`
	Name                      string   `json:"name"`
	Platform                  string   `json:"platform"`
	Status                    string   `json:"status"`
	TestModel                 string   `json:"test_model"`
	TestReasoningEffort       string   `json:"test_reasoning_effort"`
	AvailableTestModels       []string `json:"available_test_models"`
	AvailableReasoningEfforts []string `json:"available_reasoning_efforts"`
}

type intelligenceTestProxy struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

// IntelligenceTestHandler owns the small, group-scoped SVG test workflow.
// Results live in the existing settings store so this feature needs no schema migration.
type IntelligenceTestHandler struct {
	settingService   *service.SettingService
	groupRepo        service.GroupRepository
	accountRepo      service.AccountRepository
	accountTestSvc   *service.AccountTestService
	proxyRepo        service.ProxyRepository
	mu               sync.Mutex
	runMu            sync.Mutex
	healthMu         sync.Mutex
	accountCooldowns map[intelligenceTestAccountKey]intelligenceTestCooldown
}

type intelligenceTestAccountKey struct {
	accountID int64
	groupID   int64
	model     string
}

type intelligenceTestCooldown struct {
	until  time.Time
	reason string
}

func NewIntelligenceTestHandler(settingService *service.SettingService, groupRepo service.GroupRepository, accountRepo service.AccountRepository, accountTestSvc *service.AccountTestService, proxyRepo service.ProxyRepository) *IntelligenceTestHandler {
	h := &IntelligenceTestHandler{
		settingService:   settingService,
		groupRepo:        groupRepo,
		accountRepo:      accountRepo,
		accountTestSvc:   accountTestSvc,
		proxyRepo:        proxyRepo,
		accountCooldowns: make(map[intelligenceTestAccountKey]intelligenceTestCooldown),
	}
	go h.runLoop()
	return h
}

func (h *IntelligenceTestHandler) Get(c *gin.Context) {
	cfg, err := h.load(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	groups, err := h.groupRepo.ListActive(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"prompt": cfg.Prompt, "frequency_mins": cfg.FrequencyMins, "timeout_seconds": cfg.TimeoutSeconds, "proxy_id": cfg.ProxyID, "public_group_filter_enabled": cfg.PublicGroupFilterEnabled, "public_group_ids": cfg.PublicGroupIDs, "proxies": h.activeProxies(c.Request.Context()), "groups": h.intelligenceGroups(c.Request.Context(), groups, cfg), "results": cfg.Results})
}

// GetPublic exposes only the latest group results to authenticated users.
// Configuration and manual execution remain administrator-only operations.
func (h *IntelligenceTestHandler) GetPublic(c *gin.Context) {
	cfg, err := h.load(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	groups, err := h.groupRepo.ListActive(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	groups = filterPublicIntelligenceGroups(groups, cfg)
	c.JSON(http.StatusOK, gin.H{"groups": h.intelligenceGroups(c.Request.Context(), groups, cfg), "results": cfg.Results})
}

func filterPublicIntelligenceGroups(groups []service.Group, cfg *intelligenceTestConfig) []service.Group {
	if cfg == nil || !cfg.PublicGroupFilterEnabled {
		return groups
	}
	allowed := make(map[int64]struct{}, len(cfg.PublicGroupIDs))
	for _, id := range cfg.PublicGroupIDs {
		allowed[id] = struct{}{}
	}
	filtered := make([]service.Group, 0, len(cfg.PublicGroupIDs))
	for _, group := range groups {
		if _, ok := allowed[group.ID]; ok {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func (h *IntelligenceTestHandler) intelligenceGroups(ctx context.Context, groups []service.Group, cfg *intelligenceTestConfig) []intelligenceTestGroup {
	out := make([]intelligenceTestGroup, 0, len(groups))
	for _, group := range groups {
		models := h.availableTestModels(ctx, &group)
		options := intelligenceReasoningEffortsForModels(group.Platform, models)
		selected := cfg.GroupConfigs[strconv.FormatInt(group.ID, 10)]
		if !containsString(models, selected.Model) {
			selected.Model = ""
		}
		if !containsString(options, selected.ReasoningEffort) {
			selected.ReasoningEffort = ""
		}
		out = append(out, intelligenceTestGroup{
			ID: group.ID, Name: group.Name, Platform: group.Platform, Status: group.Status,
			TestModel: selected.Model, TestReasoningEffort: selected.ReasoningEffort,
			AvailableTestModels: models, AvailableReasoningEfforts: options,
		})
	}
	return out
}

func (h *IntelligenceTestHandler) availableTestModels(ctx context.Context, group *service.Group) []string {
	if group == nil {
		return nil
	}
	accounts, err := h.accountRepo.ListSchedulableByGroupID(ctx, group.ID)
	if err != nil {
		return nil
	}
	models := make([]string, 0)
	for _, candidate := range group.ModelAllowlist.Models {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.Contains(candidate, "*") || containsString(models, candidate) {
			continue
		}
		for i := range accounts {
			if accounts[i].IsModelSupported(candidate) {
				models = append(models, candidate)
				break
			}
		}
	}
	return models
}

func intelligenceReasoningEfforts(platform string) []string {
	if platform != service.PlatformOpenAI {
		return []string{}
	}
	return []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}
}

func intelligenceReasoningEffortsForModels(platform string, models []string) []string {
	options := make([]string, 0)
	initialized := false
	for _, model := range models {
		modelOptions := intelligenceReasoningEffortsForModel(platform, model)
		if len(modelOptions) == 0 {
			continue
		}
		if !initialized {
			initialized = true
			options = append([]string(nil), modelOptions...)
			continue
		}
		filtered := options[:0]
		for _, option := range options {
			if containsString(modelOptions, option) {
				filtered = append(filtered, option)
			}
		}
		options = filtered
	}
	return options
}

func intelligenceReasoningEffortsForModel(platform, model string) []string {
	if platform != service.PlatformOpenAI || strings.Contains(strings.ToLower(model), "image") {
		return []string{}
	}
	// The upstream rejects minimal for the GPT-6 Astra family. Keep this
	// catalog aligned with the real Responses API instead of letting the UI
	// offer a value that will always return HTTP 400.
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-6-astra") {
		return []string{"none", "low", "medium", "high", "xhigh", "max"}
	}
	return intelligenceReasoningEfforts(platform)
}

func containsString(values []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (h *IntelligenceTestHandler) Update(c *gin.Context) {
	var req struct {
		Prompt                   string                             `json:"prompt"`
		FrequencyMins            int                                `json:"frequency_mins"`
		TimeoutSeconds           int                                `json:"timeout_seconds"`
		ProxyID                  *int64                             `json:"proxy_id"`
		PublicGroupFilterEnabled bool                               `json:"public_group_filter_enabled"`
		PublicGroupIDs           []int64                            `json:"public_group_ids"`
		GroupConfigs             map[string]intelligenceGroupConfig `json:"group_configs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		req.Prompt = defaultIntelligenceTestPrompt
	}
	if utf8.RuneCountInString(req.Prompt) > 4000 {
		response.BadRequest(c, "测试提示词最多支持 4000 个字符")
		return
	}
	if req.FrequencyMins < 0 || req.FrequencyMins > 1440 {
		response.BadRequest(c, "frequency_mins must be between 0 and 1440")
		return
	}
	// Keep older clients compatible when they do not send the new field.
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = defaultIntelligenceTestTimeoutSeconds
	}
	if req.TimeoutSeconds < minIntelligenceTestTimeoutSeconds || req.TimeoutSeconds > maxIntelligenceTestTimeoutSeconds {
		response.BadRequest(c, "timeout_seconds must be between 30 and 1800")
		return
	}
	cfg, err := h.load(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if req.ProxyID != nil {
		proxy, err := h.proxyRepo.GetByID(c.Request.Context(), *req.ProxyID)
		if err != nil || proxy == nil || !proxy.IsActive() {
			response.BadRequest(c, "选择的代理不存在或已停用")
			return
		}
	}
	cfg.Prompt, cfg.FrequencyMins, cfg.TimeoutSeconds, cfg.ProxyID = req.Prompt, req.FrequencyMins, req.TimeoutSeconds, req.ProxyID
	cfg.PublicGroupFilterEnabled, cfg.PublicGroupIDs = req.PublicGroupFilterEnabled, normalizePublicGroupIDs(req.PublicGroupIDs)
	if req.GroupConfigs != nil {
		groups, err := h.groupRepo.ListActive(c.Request.Context())
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		for _, group := range groups {
			key := strconv.FormatInt(group.ID, 10)
			selected, supplied := req.GroupConfigs[key]
			// A temporary scheduling outage must not block unrelated settings.
			if !supplied || selected == cfg.GroupConfigs[key] {
				continue
			}
			models := h.availableTestModels(c.Request.Context(), &group)
			if selected.Model != "" && !containsString(models, selected.Model) {
				response.BadRequest(c, fmt.Sprintf("分组 %s 的测试模型当前没有可调度账号支持", group.Name))
				return
			}
			if selected.ReasoningEffort != "" && !containsString(intelligenceReasoningEffortsForModel(group.Platform, selected.Model), selected.ReasoningEffort) {
				response.BadRequest(c, fmt.Sprintf("分组 %s 不支持该思考强度", group.Name))
				return
			}
		}
		// Merge submitted groups so a single-group save preserves the others.
		for _, group := range groups {
			key := strconv.FormatInt(group.ID, 10)
			if selected, supplied := req.GroupConfigs[key]; supplied {
				cfg.GroupConfigs[key] = selected
			}
		}
	}
	if err := h.save(c.Request.Context(), cfg); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"prompt": cfg.Prompt, "frequency_mins": cfg.FrequencyMins, "timeout_seconds": cfg.TimeoutSeconds, "proxy_id": cfg.ProxyID, "public_group_filter_enabled": cfg.PublicGroupFilterEnabled, "public_group_ids": cfg.PublicGroupIDs, "group_configs": cfg.GroupConfigs})
}

func normalizePublicGroupIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (h *IntelligenceTestHandler) Run(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}
	result, err := h.testGroup(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *IntelligenceTestHandler) load(ctx context.Context) (*intelligenceTestConfig, error) {
	cfg := &intelligenceTestConfig{Prompt: defaultIntelligenceTestPrompt, GroupConfigs: map[string]intelligenceGroupConfig{}, Results: map[string]IntelligenceTestResult{}}
	raw, err := h.settingService.GetRawSetting(ctx, service.SettingKeyIntelligenceTestConfig)
	if err == nil && strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), cfg)
	} else if err != nil && !errors.Is(err, service.ErrSettingNotFound) {
		return nil, err
	}
	if cfg.Prompt == "" {
		cfg.Prompt = defaultIntelligenceTestPrompt
	}
	cfg.TimeoutSeconds = normalizeIntelligenceTestTimeout(cfg.TimeoutSeconds)
	if cfg.Results == nil {
		cfg.Results = map[string]IntelligenceTestResult{}
	}
	if cfg.GroupConfigs == nil {
		cfg.GroupConfigs = map[string]intelligenceGroupConfig{}
	}
	return cfg, nil
}

func normalizeIntelligenceTestTimeout(seconds int) int {
	if seconds == 0 {
		return defaultIntelligenceTestTimeoutSeconds
	}
	if seconds < minIntelligenceTestTimeoutSeconds {
		return minIntelligenceTestTimeoutSeconds
	}
	if seconds > maxIntelligenceTestTimeoutSeconds {
		return maxIntelligenceTestTimeoutSeconds
	}
	return seconds
}

func (h *IntelligenceTestHandler) save(ctx context.Context, cfg *intelligenceTestConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return h.settingService.SetRawSetting(ctx, service.SettingKeyIntelligenceTestConfig, string(b))
}

func (h *IntelligenceTestHandler) testGroup(ctx context.Context, groupID int64) (*IntelligenceTestResult, error) {
	if !h.runMu.TryLock() {
		return nil, errors.New("已有智力测试正在运行，请等待完成后重试")
	}
	defer h.runMu.Unlock()

	accounts, err := h.accountRepo.ListSchedulableByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	result := &IntelligenceTestResult{GroupID: groupID, Status: "error", TestedAt: time.Now()}
	if len(accounts) == 0 {
		result.Error = "该分组没有可调度账号"
		_ = h.persistResultDetached(ctx, result)
		return result, nil
	}
	cfg, err := h.load(ctx)
	if err != nil {
		return nil, err
	}
	group, err := h.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	if cfg.ProxyID != nil {
		proxy, proxyErr := h.proxyRepo.GetByID(ctx, *cfg.ProxyID)
		if proxyErr != nil || proxy == nil || !proxy.IsActive() {
			result.Error = "智力测试代理不存在或已停用"
			_ = h.persistResultDetached(ctx, result)
			return result, nil
		}
		testCtx = service.WithAccountTestProxyURL(testCtx, proxy.URL())
	}
	var lastErr error
	attemptErrors := make([]string, 0, len(accounts))
	skippedCooldowns := 0
	for _, account := range accounts {
		if testCtx.Err() != nil {
			lastErr = testCtx.Err()
			break
		}
		result.AccountID = account.ID
		selected := cfg.GroupConfigs[strconv.FormatInt(groupID, 10)]
		testCtx = service.WithAccountTestReasoningEffort(testCtx, selected.ReasoningEffort)
		modelID := selected.Model
		if modelID != "" && !account.IsModelSupported(modelID) {
			result.Model = modelID
			result.ReasoningEffort = selected.ReasoningEffort
			attemptErrors = append(attemptErrors, fmt.Sprintf("账号 #%d：不支持已选择的测试模型 %s", account.ID, modelID))
			continue
		}
		if modelID == "" {
			modelID = intelligenceTestModelIDForGroup(group, &account)
		}
		if cooldown, ok := h.accountCooldown(account.ID, groupID, modelID); ok {
			skippedCooldowns++
			attemptErrors = append(attemptErrors, fmt.Sprintf("账号 #%d：暂时跳过（%s，预计 %s 后重试）", account.ID, cooldown.reason, cooldown.until.Format("15:04:05")))
			continue
		}
		effort := selected.ReasoningEffort
		if !containsString(intelligenceReasoningEffortsForModel(group.Platform, modelID), effort) {
			effort = ""
		}
		result.Model = modelID
		result.ReasoningEffort = effort
		recorder := httptest.NewRecorder()
		gc, _ := gin.CreateTestContext(recorder)
		requestCtx := service.WithAccountTestReasoningEffort(testCtx, effort)
		gc.Request, _ = http.NewRequestWithContext(requestCtx, http.MethodPost, "/admin/intelligence-tests/run", nil)
		testErr := h.accountTestSvc.TestAccountConnection(gc, account.ID, modelID, intelligenceTestPrompt(cfg.Prompt), service.AccountTestModeDefault)
		if svg := extractIntelligenceSVG(recorder.Body.String()); svg != "" {
			result.SVG, result.Status, result.Error = svg, "success", ""
			_ = h.persistResultDetached(ctx, result)
			return result, nil
		}
		if testErr != nil {
			lastErr = testErr
			reason, cooldown, markAccount := classifyIntelligenceTestError(testErr, cfg.ProxyID != nil)
			if testCtx.Err() != nil {
				reason, _, _ = classifyIntelligenceTestError(testCtx.Err(), false)
				attemptErrors = append(attemptErrors, reason)
				break
			}
			if !markAccount {
				// Shared transport failures and cancellation are not account health failures.
				result.Error = reason
				_ = h.persistResultDetached(ctx, result)
				return result, nil
			}
			if markAccount {
				h.markAccountCooldown(account.ID, groupID, modelID, reason, cooldown)
			}
			attemptErrors = append(attemptErrors, fmt.Sprintf("账号 #%d：%s", account.ID, reason))
		}
		if errors.Is(testCtx.Err(), context.DeadlineExceeded) {
			lastErr = errors.New("测试超时（超过管理员设置的超时时间）")
			break
		}
	}
	if lastErr == nil && skippedCooldowns == len(accounts) {
		result.Error = "该分组测试账号正在短期冷却：\n" + strings.Join(attemptErrors, "\n")
		_ = h.persistResultDetached(ctx, result)
		return result, nil
	}
	if lastErr != nil {
		if len(attemptErrors) > 0 {
			result.Error = strings.Join(attemptErrors, "\n")
		} else {
			result.Error = lastErr.Error()
		}
	} else {
		if len(attemptErrors) > 0 {
			result.Error = strings.Join(attemptErrors, "\n")
		} else {
			result.Error = "测试响应中没有找到 SVG"
		}
	}
	_ = h.persistResultDetached(ctx, result)
	return result, nil
}

func (h *IntelligenceTestHandler) accountCooldown(accountID, groupID int64, model string) (intelligenceTestCooldown, bool) {
	key := intelligenceTestAccountKey{accountID: accountID, groupID: groupID, model: strings.TrimSpace(model)}
	now := time.Now()
	h.healthMu.Lock()
	defer h.healthMu.Unlock()
	cooldown, ok := h.accountCooldowns[key]
	if !ok {
		return intelligenceTestCooldown{}, false
	}
	if !now.Before(cooldown.until) {
		delete(h.accountCooldowns, key)
		return intelligenceTestCooldown{}, false
	}
	return cooldown, true
}

func (h *IntelligenceTestHandler) markAccountCooldown(accountID, groupID int64, model, reason string, duration time.Duration) {
	if duration <= 0 {
		return
	}
	key := intelligenceTestAccountKey{accountID: accountID, groupID: groupID, model: strings.TrimSpace(model)}
	h.healthMu.Lock()
	if h.accountCooldowns == nil {
		h.accountCooldowns = make(map[intelligenceTestAccountKey]intelligenceTestCooldown)
	}
	h.accountCooldowns[key] = intelligenceTestCooldown{until: time.Now().Add(duration), reason: reason}
	h.healthMu.Unlock()
}

// classifyIntelligenceTestError keeps the test-only health memory separate from
// account scheduling. A selected test proxy failing must not quarantine every
// account behind it, while credential/model failures should stop repeated probes.
func classifyIntelligenceTestError(err error, hasTestProxy bool) (string, time.Duration, bool) {
	if err == nil {
		return "测试失败", 0, false
	}
	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)
	if errors.Is(err, context.Canceled) || strings.Contains(lower, "context canceled") {
		return "测试请求已中断，未将账号标记为失效，请重新测试", 0, false
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(lower, "context deadline exceeded") {
		return "测试请求超时，未将账号标记为失效，请检查超时设置或网络后重试", 0, false
	}
	proxyFailure := strings.Contains(lower, "socks") ||
		strings.Contains(lower, "http proxy") ||
		strings.Contains(lower, "proxy") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, " eof") || strings.HasSuffix(lower, "eof")
	if hasTestProxy && proxyFailure {
		return "测试代理连接失败，请检查代理协议、端口或目标白名单", 0, false
	}
	switch {
	case strings.Contains(lower, "401"), strings.Contains(lower, "unauthorized"), strings.Contains(lower, "authentication"):
		return "账号鉴权失败（401），该账号暂时跳过", 15 * time.Minute, true
	case strings.Contains(lower, "403"), strings.Contains(lower, "forbidden"), strings.Contains(lower, "not entitled"):
		return "账号无权使用当前上游或模型（403），该账号暂时跳过", 15 * time.Minute, true
	case strings.Contains(lower, "429"), strings.Contains(lower, "usage_limit_reached"), strings.Contains(lower, "usage limit"):
		return "账号额度已耗尽（429），该账号暂时跳过", 30 * time.Minute, true
	case strings.Contains(lower, "model") && (strings.Contains(lower, "not found") || strings.Contains(lower, "not supported") || strings.Contains(lower, "unavailable") || strings.Contains(lower, "no available channel")):
		return "当前账号或上游没有可用的测试模型，该账号暂时跳过", 10 * time.Minute, true
	case strings.Contains(lower, "502"), strings.Contains(lower, "503"), strings.Contains(lower, "504"), strings.Contains(lower, "upstream"):
		return "上游暂时不可用，该账号暂时跳过", 3 * time.Minute, true
	default:
		return message, 2 * time.Minute, true
	}
}

func intelligenceTestPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		prompt = defaultIntelligenceTestPrompt
	}
	return prompt + "\nReturn only one complete SVG document beginning with <svg> and ending with </svg>; do not add Markdown fences or explanations."
}

// intelligenceTestModelID deliberately ignores legacy group default models:
// those values can describe a model that no account in the group supports.
// Account mappings are the authoritative per-upstream model declaration.
func intelligenceTestModelID(_ *service.Group, account *service.Account) string {
	return intelligenceTestModelIDForGroup(nil, account)
}

// intelligenceTestModelIDForGroup picks a model the group exposes before
// consulting account mappings. Mapping maps often retain historical models;
// selecting their first sorted key can probe a model that the current group
// no longer serves.
func intelligenceTestModelIDForGroup(group *service.Group, account *service.Account) string {
	if account == nil {
		return ""
	}
	candidates := make([]string, 0)
	if group != nil {
		preferred := intelligenceTestPreferredModels(account.Platform)
		allowed := make(map[string]bool, len(group.ModelAllowlist.Models))
		for _, model := range group.ModelAllowlist.Models {
			allowed[strings.TrimSpace(model)] = true
		}
		for _, model := range preferred {
			if allowed[model] {
				candidates = append(candidates, model)
			}
		}
		for _, model := range group.ModelAllowlist.Models {
			if !allowed[strings.TrimSpace(model)] {
				continue
			}
			found := false
			for _, selected := range candidates {
				if selected == strings.TrimSpace(model) {
					found = true
					break
				}
			}
			if !found {
				candidates = append(candidates, model)
			}
		}
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && !strings.Contains(candidate, "*") && account.IsModelSupported(candidate) {
			return candidate
		}
	}
	// Keep a stable platform fallback for groups without an explicit list.
	switch account.Platform {
	case service.PlatformAnthropic:
		candidates = append(candidates, "claude-sonnet-4-6", "claude-sonnet-4-5-20250929")
	case service.PlatformGrok:
		candidates = append(candidates, "grok-4.6", "grok-4.5")
	case service.PlatformOpenAI:
		candidates = append(candidates, "gpt-5.6-terra", "gpt-5.6-sol", "gpt-5.5")
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && account.IsModelSupported(candidate) {
			return candidate
		}
	}
	return ""
}

func intelligenceTestPreferredModels(platform string) []string {
	switch platform {
	case service.PlatformAnthropic:
		return []string{"claude-sonnet-4-6", "claude-sonnet-4-5-20250929", "claude-haiku-4-5-20251001"}
	case service.PlatformGrok:
		return []string{"grok-4.6", "grok-4.5", "grok-4.3"}
	case service.PlatformOpenAI:
		return []string{"gpt-5.6-terra", "gpt-5.6-sol", "gpt-5.5"}
	default:
		return nil
	}
}

// extractIntelligenceSVG decodes the JSON payload carried by the account-test
// SSE stream before looking for SVG. json.Unmarshal restores escaped markup
// such as \u003csvg emitted by encoding/json.
func extractIntelligenceSVG(body string) string {
	var text strings.Builder
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event any
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err == nil {
			collectIntelligenceText(event, &text)
		}
	}
	if svg := intelligenceSVGPattern.FindString(text.String()); svg != "" {
		return svg
	}
	// Some compatible relays put escaped JSON or plain text outside a typed
	// content event. Search the raw stream as a final compatibility fallback.
	decoded := strings.ReplaceAll(strings.ReplaceAll(body, `\u003c`, "<"), `\u003e`, ">")
	return intelligenceSVGPattern.FindString(decoded)
}

func collectIntelligenceText(value any, out *strings.Builder) {
	switch v := value.(type) {
	case string:
		out.WriteString(v)
	case []any:
		for _, item := range v {
			collectIntelligenceText(item, out)
		}
	case map[string]any:
		if eventType, ok := v["type"].(string); ok {
			switch eventType {
			case "status", "test_start", "test_complete", "error":
				return
			}
		}
		for key, item := range v {
			if key == "type" || key == "model" || key == "status" {
				continue
			}
			collectIntelligenceText(item, out)
		}
	}
}

func (h *IntelligenceTestHandler) persistResult(ctx context.Context, result *IntelligenceTestResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	cfg, err := h.load(ctx)
	if err != nil {
		return err
	}
	cfg.Results[strconv.FormatInt(result.GroupID, 10)] = *result
	return h.save(ctx, cfg)
}

func (h *IntelligenceTestHandler) persistResultDetached(ctx context.Context, result *IntelligenceTestResult) error {
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return h.persistResult(persistCtx, result)
}

func (h *IntelligenceTestHandler) activeProxies(ctx context.Context) []intelligenceTestProxy {
	if h.proxyRepo == nil {
		return []intelligenceTestProxy{}
	}
	proxies, err := h.proxyRepo.ListActive(ctx)
	if err != nil {
		return []intelligenceTestProxy{}
	}
	out := make([]intelligenceTestProxy, 0, len(proxies))
	for _, proxy := range proxies {
		out = append(out, intelligenceTestProxy{ID: proxy.ID, Name: proxy.Name, Protocol: proxy.Protocol, Host: proxy.Host, Port: proxy.Port})
	}
	return out
}

func (h *IntelligenceTestHandler) runLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		cfg, err := h.load(ctx)
		if err == nil && cfg.FrequencyMins > 0 {
			groups, groupErr := h.groupRepo.ListActive(ctx)
			if groupErr == nil {
				for _, group := range groups {
					old, ok := cfg.Results[strconv.FormatInt(group.ID, 10)]
					if !ok || time.Since(old.TestedAt) >= time.Duration(cfg.FrequencyMins)*time.Minute {
						// Each group gets its own configured budget, not the metadata lookup deadline.
						runCtx, runCancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds+5)*time.Second)
						_, _ = h.testGroup(runCtx, group.ID)
						runCancel()
					}
				}
			}
		}
		cancel()
	}
}
