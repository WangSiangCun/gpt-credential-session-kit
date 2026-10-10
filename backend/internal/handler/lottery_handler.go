package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LotteryHandler struct {
	service *service.LotteryService
}

func NewLotteryHandler(lotteryService *service.LotteryService) *LotteryHandler {
	return &LotteryHandler{service: lotteryService}
}

type publicLotteryPrize struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	MinAmount         float64 `json:"min_amount"`
	MaxAmount         float64 `json:"max_amount"`
	TotalQuantity     int     `json:"total_quantity"`
	RemainingQuantity int     `json:"remaining_quantity"`
}

type publicLotteryCampaign struct {
	ID                   int64                `json:"id"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	ActivityType         string               `json:"activity_type"`
	Status               string               `json:"status"`
	WinRate              float64              `json:"win_rate"`
	MinTotalRecharged    float64              `json:"min_total_recharged,omitempty"`
	StartsAt             time.Time            `json:"starts_at"`
	ExpiresAt            time.Time            `json:"expires_at"`
	RegistrationDeadline *time.Time           `json:"registration_deadline,omitempty"`
	DrawAt               *time.Time           `json:"draw_at,omitempty"`
	WinnerLimit          int                  `json:"winner_limit,omitempty"`
	WinnerCount          int64                `json:"winner_count"`
	Prizes               []publicLotteryPrize `json:"prizes"`
}

type publicLotteryDraw struct {
	ID         int64     `json:"id"`
	Status     string    `json:"status"`
	PrizeName  string    `json:"prize_name,omitempty"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
	NewBalance *float64  `json:"new_balance,omitempty"`
}

type publicLotteryWinner struct {
	ID            int64     `json:"id"`
	MaskedAccount string    `json:"masked_account"`
	PrizeName     string    `json:"prize_name"`
	Amount        float64   `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

func toPublicLotteryCampaign(campaign service.LotteryCampaign) publicLotteryCampaign {
	prizes := make([]publicLotteryPrize, 0, len(campaign.Prizes))
	for _, prize := range campaign.Prizes {
		prizes = append(prizes, publicLotteryPrize{
			ID:                prize.ID,
			Name:              prize.Name,
			MinAmount:         prize.MinAmount,
			MaxAmount:         prize.MaxAmount,
			TotalQuantity:     prize.TotalQuantity,
			RemainingQuantity: prize.RemainingQuantity,
		})
	}
	return publicLotteryCampaign{
		ID:                   campaign.ID,
		Name:                 campaign.Name,
		Description:          campaign.Description,
		ActivityType:         campaign.ActivityType,
		Status:               campaign.Status,
		WinRate:              campaign.WinRate,
		MinTotalRecharged:    campaign.MinTotalRecharged,
		StartsAt:             campaign.StartsAt,
		ExpiresAt:            campaign.ExpiresAt,
		RegistrationDeadline: campaign.RegistrationDeadline,
		DrawAt:               campaign.DrawAt,
		WinnerLimit:          campaign.WinnerLimit,
		WinnerCount:          campaign.WinnerCount,
		Prizes:               prizes,
	}
}

func toPublicLotteryDraw(draw *service.LotteryDraw) *publicLotteryDraw {
	if draw == nil {
		return nil
	}
	return &publicLotteryDraw{
		ID:         draw.ID,
		Status:     draw.Status,
		PrizeName:  draw.PrizeName,
		Amount:     draw.Amount,
		CreatedAt:  draw.CreatedAt,
		NewBalance: draw.NewBalance,
	}
}

func maskLotteryAccount(username, email string) string {
	account := strings.TrimSpace(username)
	if account != "" {
		runes := []rune(account)
		switch len(runes) {
		case 0:
			return "***"
		case 1:
			return "*"
		case 2:
			return string(runes[0]) + "*"
		default:
			return string(runes[0]) + "***" + string(runes[len(runes)-1])
		}
	}

	email = strings.TrimSpace(email)
	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" || domain == "" {
		if email == "" {
			return "***"
		}
		first, _ := utf8.DecodeRuneInString(email)
		return string(first) + "***"
	}
	first, _ := utf8.DecodeRuneInString(local)
	return string(first) + "***@" + domain
}

func (h *LotteryHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.service.ListPublicCampaigns(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	publicItems := make([]publicLotteryCampaign, 0, len(items))
	for _, item := range items {
		publicItems = append(publicItems, toPublicLotteryCampaign(item))
	}
	response.Paginated(c, publicItems, total, page, pageSize)
}

func (h *LotteryHandler) GetCampaign(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录后参与抽奖")
		return
	}
	campaign, draw, err := h.service.GetCampaign(c.Request.Context(), c.Param("token"), subject.UserID)
	if err != nil {
		writeLotteryError(c, err)
		return
	}
	response.Success(c, gin.H{
		"campaign": toPublicLotteryCampaign(*campaign),
		"my_draw":  toPublicLotteryDraw(draw),
	})
}

func (h *LotteryHandler) Winners(c *gin.Context) {
	if _, ok := middleware2.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "请先登录后查看中奖名单")
		return
	}
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || campaignID <= 0 {
		response.NotFound(c, "抽奖活动不存在")
		return
	}
	if _, err := h.service.GetCampaignByID(c.Request.Context(), campaignID); err != nil {
		writeLotteryError(c, err)
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.service.ListPublicWinners(c.Request.Context(), campaignID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	winners := make([]publicLotteryWinner, 0, len(items))
	for _, item := range items {
		winners = append(winners, publicLotteryWinner{
			ID:            item.ID,
			MaskedAccount: maskLotteryAccount(item.Username, item.UserEmail),
			PrizeName:     item.PrizeName,
			Amount:        item.Amount,
			CreatedAt:     item.CreatedAt,
		})
	}
	response.Paginated(c, winners, total, page, pageSize)
}

func (h *LotteryHandler) Draw(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录后参与抽奖")
		return
	}
	draw, err := h.service.Draw(c.Request.Context(), c.Param("token"), subject.UserID)
	if err != nil {
		writeLotteryError(c, err)
		return
	}
	response.Success(c, toPublicLotteryDraw(draw))
}

func writeLotteryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLotteryNotFound):
		response.NotFound(c, "抽奖活动不存在")
	case errors.Is(err, service.ErrLotteryUnavailable):
		response.Error(c, http.StatusConflict, "活动未开始、已结束或奖品已抽完")
	case errors.Is(err, service.ErrLotteryAlreadyDrawn):
		response.Error(c, http.StatusConflict, "你已经参与过本次抽奖")
	case errors.Is(err, service.ErrLotteryRegistrationClosed):
		response.Error(c, http.StatusConflict, "报名已经截止，请等待开奖")
	case errors.Is(err, service.ErrLotteryThresholdNotMet):
		response.Error(c, http.StatusForbidden, "未达到本次活动的累计充值门槛")
	case errors.Is(err, service.ErrLotteryInvalidConfig):
		response.BadRequest(c, "抽奖配置不正确")
	default:
		response.ErrorFrom(c, err)
	}
}
