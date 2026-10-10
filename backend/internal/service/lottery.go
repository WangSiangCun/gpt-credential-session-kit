package service

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLotteryNotFound           = errors.New("lottery not found")
	ErrLotteryUnavailable        = errors.New("lottery is unavailable")
	ErrLotteryAlreadyDrawn       = errors.New("lottery already drawn")
	ErrLotteryRegistrationClosed = errors.New("lottery registration is closed")
	ErrLotteryThresholdNotMet    = errors.New("lottery recharge threshold not met")
	ErrLotteryInvalidConfig      = errors.New("invalid lottery configuration")
	ErrLotteryDeleteDenied       = errors.New("lottery campaign cannot be deleted")
)

const (
	LotteryActivityTypeInstant      = "instant_lottery"
	LotteryActivityTypeRegistration = "registration_lottery"
)

type LotteryPrize struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	MinAmount         float64   `json:"min_amount"`
	MaxAmount         float64   `json:"max_amount"`
	TotalQuantity     int       `json:"total_quantity"`
	RemainingQuantity int       `json:"remaining_quantity"`
	Weight            int       `json:"weight"`
	SortOrder         int       `json:"sort_order"`
	CreatedAt         time.Time `json:"created_at"`
}

type LotteryCampaign struct {
	ID                   int64          `json:"id"`
	Token                string         `json:"token"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	ActivityType         string         `json:"activity_type"`
	Status               string         `json:"status"`
	WinRate              float64        `json:"win_rate"`
	MinTotalRecharged    float64        `json:"min_total_recharged,omitempty"`
	StartsAt             time.Time      `json:"starts_at"`
	ExpiresAt            time.Time      `json:"expires_at"`
	RegistrationDeadline *time.Time     `json:"registration_deadline,omitempty"`
	DrawAt               *time.Time     `json:"draw_at,omitempty"`
	WinnerLimit          int            `json:"winner_limit,omitempty"`
	CreatedBy            int64          `json:"created_by"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	WinnerCount          int64          `json:"winner_count"`
	DrawCount            int64          `json:"draw_count"`
	Prizes               []LotteryPrize `json:"prizes"`
}

type LotteryDraw struct {
	ID            int64     `json:"id"`
	CampaignID    int64     `json:"campaign_id"`
	UserID        int64     `json:"user_id"`
	Status        string    `json:"status"`
	PrizeID       *int64    `json:"prize_id,omitempty"`
	PrizeName     string    `json:"prize_name,omitempty"`
	Amount        float64   `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
	NewBalance    *float64  `json:"new_balance,omitempty"`
	UserEmail     string    `json:"user_email,omitempty"`
	Username      string    `json:"username,omitempty"`
	UserRole      string    `json:"user_role,omitempty"`
	UserStatus    string    `json:"user_status,omitempty"`
	UserBalance   float64   `json:"user_balance,omitempty"`
	UserCreatedAt time.Time `json:"user_created_at,omitempty"`
}

type CreateLotteryPrizeInput struct {
	Name      string  `json:"name"`
	MinAmount float64 `json:"min_amount"`
	MaxAmount float64 `json:"max_amount"`
	Quantity  int     `json:"quantity"`
	Weight    int     `json:"weight"`
}

type CreateLotteryCampaignInput struct {
	Name                 string                    `json:"name"`
	Description          string                    `json:"description"`
	ActivityType         string                    `json:"activity_type"`
	WinRate              float64                   `json:"win_rate"`
	MinTotalRecharged    float64                   `json:"min_total_recharged"`
	StartsAt             *time.Time                `json:"starts_at"`
	ExpiresAt            time.Time                 `json:"expires_at"`
	RegistrationDeadline *time.Time                `json:"registration_deadline"`
	DrawAt               *time.Time                `json:"draw_at"`
	WinnerLimit          int                       `json:"winner_limit"`
	Prizes               []CreateLotteryPrizeInput `json:"prizes"`
}

type LotteryRepository interface {
	CreateCampaign(ctx context.Context, adminID int64, token string, input CreateLotteryCampaignInput) (*LotteryCampaign, error)
	SettleDueCampaigns(ctx context.Context) error
	ListCampaigns(ctx context.Context, page, pageSize int) ([]LotteryCampaign, int64, error)
	GetCampaignByID(ctx context.Context, id int64) (*LotteryCampaign, error)
	GetCampaignByToken(ctx context.Context, token string, userID int64) (*LotteryCampaign, *LotteryDraw, error)
	StopCampaign(ctx context.Context, id int64) error
	DeleteCampaign(ctx context.Context, id int64) error
	ListWinners(ctx context.Context, campaignID int64, page, pageSize int) ([]LotteryDraw, int64, error)
	ListPublicWinners(ctx context.Context, campaignID int64, page, pageSize int) ([]LotteryDraw, int64, error)
	Draw(ctx context.Context, token string, userID int64) (*LotteryDraw, error)
}
