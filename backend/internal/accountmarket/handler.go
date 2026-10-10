package accountmarket

import (
	"io"
	"net/http"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

type OAuthURLRequest struct {
	ProxyID     *int64 `json:"proxy_id"`
	RedirectURI string `json:"redirect_uri"`
}

type OAuthCodeRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetPublicConfig(c *gin.Context) {
	cfg, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *Handler) ListProxies(c *gin.Context) {
	items, err := h.service.ListSelectableProxies(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) GenerateOAuthURL(c *gin.Context) {
	var req OAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		response.BadRequest(c, "OAuth 参数不正确")
		return
	}
	result, err := h.service.GenerateOAuthURL(c.Request.Context(), req.ProxyID, req.RedirectURI)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ExchangeOAuthCode(c *gin.Context) {
	var req OAuthCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "OAuth 回调参数不完整")
		return
	}
	result, err := h.service.ExchangeOAuthCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{
		SessionID: req.SessionID, Code: req.Code, State: req.State,
		RedirectURI: req.RedirectURI, ProxyID: req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) Upload(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes)
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "上传文件不能超过 2 MB")
		return
	}
	inputs, err := ParseUpload(data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var selectedProxyID *int64
	if raw := c.Query("proxy_id"); raw != "" {
		proxyID, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || proxyID <= 0 {
			response.BadRequest(c, "代理 ID 参数不正确")
			return
		}
		selectedProxyID = &proxyID
	}
	results := make([]UploadResult, 0, len(inputs))
	for index, input := range inputs {
		result, uploadErr := h.service.Upload(c.Request.Context(), subject.UserID, input, selectedProxyID)
		if uploadErr != nil {
			results = append(results, UploadResult{
				Index: index, Success: false, Name: input.Name,
				Error: infraerrors.Message(uploadErr),
			})
			continue
		}
		result.Index = index
		results = append(results, *result)
	}
	response.Success(c, gin.H{"items": results})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	items, err := h.service.ListAccounts(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) RefreshOpenAIQuota(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	accountID, ok := pathID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.RefreshOpenAIQuota(c.Request.Context(), userID, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"user_id": result.Usage.UserID, "account_id": result.Usage.AccountID,
		"email": result.Usage.Email, "plan_type": result.Usage.PlanType,
		"rate_limit": result.Usage.RateLimit, "additional_rate_limits": result.Usage.AdditionalRateLimits,
		"rate_limit_reset_credits": result.Usage.RateLimitResetCredits, "credits": result.Usage.Credits,
		"fetched_at": result.Usage.FetchedAt, "cache_persisted": result.CachePersisted,
		"credits_cache_persisted": result.CreditsCachePersisted,
	})
}

func (h *Handler) ResetOpenAIQuota(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	accountID, ok := pathID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.ResetOpenAIQuota(c.Request.Context(), userID, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"code": result.Result.Code, "windows_reset": result.Result.WindowsReset,
		"quota": result.Quota, "cache_refreshed": result.CacheRefreshed,
		"account_state_recovered": result.AccountStateRecovered, "warning_code": result.WarningCode,
	})
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	accountID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req AccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数不正确")
		return
	}
	item, err := h.service.UpdateAccount(c.Request.Context(), userID, accountID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) SetSchedulable(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	accountID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Enabled *bool `json:"enabled" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		response.BadRequest(c, "enabled 参数不正确")
		return
	}
	if err := h.service.SetSchedulable(c.Request.Context(), userID, accountID, *req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": accountID, "schedulable": *req.Enabled})
}

func (h *Handler) Remove(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	accountID, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Remove(c.Request.Context(), userID, accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": accountID})
}

func (h *Handler) Export(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	accountID, ok := pathID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.ExportAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) Summary(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	summary, err := h.service.GetSummary(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *Handler) ListWithdrawals(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	items, err := h.service.ListWithdrawals(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) RequestWithdrawal(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req WithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写完整的收款信息")
		return
	}
	item, err := h.service.RequestWithdrawal(c.Request.Context(), userID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handler) CancelWithdrawal(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.CancelWithdrawal(c.Request.Context(), userID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) AdminGetConfig(c *gin.Context) {
	h.GetPublicConfig(c)
}

func (h *Handler) AdminUpdateConfig(c *gin.Context) {
	var cfg Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "配置参数不正确")
		return
	}
	updated, err := h.service.UpdateConfig(c.Request.Context(), cfg)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *Handler) AdminListAccounts(c *gin.Context) {
	items, err := h.service.ListAdminAccounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) AdminListWithdrawals(c *gin.Context) {
	items, err := h.service.ListAdminWithdrawals(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) AdminReviewWithdrawal(c *gin.Context) {
	adminID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "审核参数不正确")
		return
	}
	if err := h.service.ReviewWithdrawal(c.Request.Context(), adminID, id, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "status": req.Status})
}

func currentUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func pathID(c *gin.Context, key string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "ID 参数不正确")
		return 0, false
	}
	return id, true
}
