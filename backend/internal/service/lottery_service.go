package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math"
	"strings"
	"time"
)

const (
	maxLotteryPrizes       = 1000
	maxLotteryIntegerValue = 2_147_483_647
	MaxLotteryPayoutAmount = 999_999_999_999.99
	MaxLotteryRechargeGate = 999_999_999_999.99
)

type LotteryService struct {
	repo LotteryRepository
}

func NewLotteryService(repo LotteryRepository) *LotteryService {
	return &LotteryService{repo: repo}
}

func (s *LotteryService) CreateCampaign(ctx context.Context, adminID int64, input CreateLotteryCampaignInput) (*LotteryCampaign, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.ActivityType == "" {
		input.ActivityType = LotteryActivityTypeInstant
	}
	if input.Name == "" ||
		(input.ActivityType != LotteryActivityTypeInstant && input.ActivityType != LotteryActivityTypeRegistration) ||
		!isFiniteLotteryNumber(input.MinTotalRecharged) ||
		input.MinTotalRecharged < 0 ||
		input.MinTotalRecharged > MaxLotteryRechargeGate {
		return nil, ErrLotteryInvalidConfig
	}
	now := time.Now()
	if input.StartsAt == nil {
		input.StartsAt = &now
	}
	if input.ActivityType == LotteryActivityTypeRegistration {
		if input.WinnerLimit <= 0 || input.WinnerLimit > maxLotteryIntegerValue ||
			input.RegistrationDeadline == nil || input.DrawAt == nil ||
			!input.RegistrationDeadline.After(*input.StartsAt) ||
			!input.DrawAt.After(*input.RegistrationDeadline) {
			return nil, ErrLotteryInvalidConfig
		}
		if input.ExpiresAt.IsZero() {
			input.ExpiresAt = *input.DrawAt
		}
		if !input.ExpiresAt.Equal(*input.DrawAt) {
			return nil, ErrLotteryInvalidConfig
		}
		input.WinRate = 0
		input.Prizes = nil
	} else {
		if !isFiniteLotteryNumber(input.WinRate) || input.WinRate < 0 || input.WinRate > 100 ||
			len(input.Prizes) == 0 || len(input.Prizes) > maxLotteryPrizes ||
			!input.ExpiresAt.After(*input.StartsAt) {
			return nil, ErrLotteryInvalidConfig
		}
	}
	for i := range input.Prizes {
		p := &input.Prizes[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			p.Name = "随机余额"
		}
		if !isFiniteLotteryNumber(p.MinAmount) ||
			!isFiniteLotteryNumber(p.MaxAmount) ||
			p.MinAmount < 0 ||
			p.MaxAmount < p.MinAmount ||
			p.MaxAmount < 0.01 ||
			p.MaxAmount > MaxLotteryPayoutAmount ||
			p.Quantity <= 0 ||
			p.Quantity > maxLotteryIntegerValue ||
			p.Weight > maxLotteryIntegerValue {
			return nil, ErrLotteryInvalidConfig
		}
		if p.Weight <= 0 {
			p.Weight = 1
		}
	}
	tokenBytes := make([]byte, 20)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	return s.repo.CreateCampaign(ctx, adminID, hex.EncodeToString(tokenBytes), input)
}

func isFiniteLotteryNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (s *LotteryService) ListCampaigns(ctx context.Context, page, pageSize int) ([]LotteryCampaign, int64, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, 0, err
	}
	return s.repo.ListCampaigns(ctx, page, pageSize)
}

func (s *LotteryService) GetCampaignByID(ctx context.Context, id int64) (*LotteryCampaign, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetCampaignByID(ctx, id)
}

func (s *LotteryService) GetCampaign(ctx context.Context, token string, userID int64) (*LotteryCampaign, *LotteryDraw, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, nil, err
	}
	return s.repo.GetCampaignByToken(ctx, strings.TrimSpace(token), userID)
}

func (s *LotteryService) StopCampaign(ctx context.Context, id int64) error {
	return s.repo.StopCampaign(ctx, id)
}

func (s *LotteryService) DeleteCampaign(ctx context.Context, id int64) error {
	return s.repo.DeleteCampaign(ctx, id)
}

func (s *LotteryService) ListPublicCampaigns(ctx context.Context, page, pageSize int) ([]LotteryCampaign, int64, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, 0, err
	}
	items, total, err := s.repo.ListCampaigns(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		detail, detailErr := s.repo.GetCampaignByID(ctx, items[i].ID)
		if detailErr != nil {
			return nil, 0, detailErr
		}
		items[i].Prizes = detail.Prizes
	}
	return items, total, nil
}

func (s *LotteryService) ListWinners(ctx context.Context, campaignID int64, page, pageSize int) ([]LotteryDraw, int64, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, 0, err
	}
	return s.repo.ListWinners(ctx, campaignID, page, pageSize)
}

func (s *LotteryService) ListPublicWinners(ctx context.Context, campaignID int64, page, pageSize int) ([]LotteryDraw, int64, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, 0, err
	}
	return s.repo.ListPublicWinners(ctx, campaignID, page, pageSize)
}

func (s *LotteryService) Draw(ctx context.Context, token string, userID int64) (*LotteryDraw, error) {
	if err := s.repo.SettleDueCampaigns(ctx); err != nil {
		return nil, err
	}
	return s.repo.Draw(ctx, strings.TrimSpace(token), userID)
}
