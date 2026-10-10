package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LotteryHandler struct {
	service *service.LotteryService
}

func writeLotteryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLotteryNotFound):
		response.NotFound(c, "抽奖活动不存在")
	case errors.Is(err, service.ErrLotteryUnavailable):
		response.Error(c, http.StatusConflict, "活动未开始、已结束或奖品已抽完")
	case errors.Is(err, service.ErrLotteryAlreadyDrawn):
		response.Error(c, http.StatusConflict, "用户已经参与过本次抽奖")
	case errors.Is(err, service.ErrLotteryRegistrationClosed):
		response.Error(c, http.StatusConflict, "报名已经截止，请等待开奖")
	case errors.Is(err, service.ErrLotteryThresholdNotMet):
		response.Error(c, http.StatusForbidden, "未达到本次活动的累计充值门槛")
	case errors.Is(err, service.ErrLotteryInvalidConfig):
		response.BadRequest(c, "抽奖配置不正确，请检查时间、中奖率和余额范围")
	case errors.Is(err, service.ErrLotteryDeleteDenied):
		response.Error(c, http.StatusConflict, "只能删除已停止、已抽完或已过期的历史活动")
	default:
		response.ErrorFrom(c, err)
	}
}

func NewLotteryHandler(lotteryService *service.LotteryService) *LotteryHandler {
	return &LotteryHandler{service: lotteryService}
}

func (h *LotteryHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "管理员未登录")
		return
	}
	var input service.CreateLotteryCampaignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "抽奖配置格式不正确")
		return
	}
	campaign, err := h.service.CreateCampaign(c.Request.Context(), subject.UserID, input)
	if err != nil {
		writeLotteryError(c, err)
		return
	}
	response.Created(c, campaign)
}

func (h *LotteryHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.service.ListCampaigns(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *LotteryHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "活动 ID 不正确")
		return
	}
	item, err := h.service.GetCampaignByID(c.Request.Context(), id)
	if err != nil {
		writeLotteryError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *LotteryHandler) Stop(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "活动 ID 不正确")
		return
	}
	if err := h.service.StopCampaign(c.Request.Context(), id); err != nil {
		writeLotteryError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "活动已停止"})
}

func (h *LotteryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "活动 ID 不正确")
		return
	}
	if err := h.service.DeleteCampaign(c.Request.Context(), id); err != nil {
		writeLotteryError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "历史活动已删除"})
}

func (h *LotteryHandler) Winners(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "活动 ID 不正确")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.service.ListWinners(c.Request.Context(), id, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}
