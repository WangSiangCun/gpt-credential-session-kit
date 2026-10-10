package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type lotteryRepoStub struct {
	created *CreateLotteryCampaignInput
	token   string
}

func (s *lotteryRepoStub) CreateCampaign(_ context.Context, _ int64, token string, input CreateLotteryCampaignInput) (*LotteryCampaign, error) {
	s.created = &input
	s.token = token
	return &LotteryCampaign{Token: token, Name: input.Name}, nil
}
func (*lotteryRepoStub) SettleDueCampaigns(context.Context) error { return nil }
func (*lotteryRepoStub) ListCampaigns(context.Context, int, int) ([]LotteryCampaign, int64, error) {
	return nil, 0, nil
}
func (*lotteryRepoStub) GetCampaignByID(context.Context, int64) (*LotteryCampaign, error) {
	return nil, nil
}
func (*lotteryRepoStub) GetCampaignByToken(context.Context, string, int64) (*LotteryCampaign, *LotteryDraw, error) {
	return nil, nil, nil
}
func (*lotteryRepoStub) StopCampaign(context.Context, int64) error   { return nil }
func (*lotteryRepoStub) DeleteCampaign(context.Context, int64) error { return nil }
func (*lotteryRepoStub) ListWinners(context.Context, int64, int, int) ([]LotteryDraw, int64, error) {
	return nil, 0, nil
}
func (*lotteryRepoStub) ListPublicWinners(context.Context, int64, int, int) ([]LotteryDraw, int64, error) {
	return nil, 0, nil
}
func (*lotteryRepoStub) Draw(context.Context, string, int64) (*LotteryDraw, error) {
	return nil, nil
}

func TestLotteryCreateAcceptsRandomBalanceRange(t *testing.T) {
	repo := &lotteryRepoStub{}
	svc := NewLotteryService(repo)
	expires := time.Now().Add(time.Hour)
	_, err := svc.CreateCampaign(context.Background(), 1, CreateLotteryCampaignInput{
		Name:      "范围余额",
		WinRate:   35,
		ExpiresAt: expires,
		Prizes: []CreateLotteryPrizeInput{{
			Name:      "十份随机余额",
			MinAmount: 0,
			MaxAmount: 1,
			Quantity:  10,
		}},
	})
	if err != nil {
		t.Fatalf("CreateCampaign() error = %v", err)
	}
	if repo.created == nil || repo.created.Prizes[0].Quantity != 10 {
		t.Fatal("range prize was not passed to repository")
	}
	if repo.created.Prizes[0].Weight != 1 {
		t.Fatalf("default weight = %d, want 1", repo.created.Prizes[0].Weight)
	}
	if len(repo.token) != 40 {
		t.Fatalf("token length = %d, want 40", len(repo.token))
	}
}

func TestLotteryCreateRejectsInvalidRange(t *testing.T) {
	svc := NewLotteryService(&lotteryRepoStub{})
	_, err := svc.CreateCampaign(context.Background(), 1, CreateLotteryCampaignInput{
		Name:      "错误范围",
		WinRate:   20,
		ExpiresAt: time.Now().Add(time.Hour),
		Prizes: []CreateLotteryPrizeInput{{
			Name:      "错误",
			MinAmount: 2,
			MaxAmount: 1,
			Quantity:  1,
		}},
	})
	if err != ErrLotteryInvalidConfig {
		t.Fatalf("CreateCampaign() error = %v, want ErrLotteryInvalidConfig", err)
	}
}

func TestLotteryCreateRejectsNonFiniteAndOverflowingAmounts(t *testing.T) {
	tests := []struct {
		name string
		min  float64
		max  float64
	}{
		{name: "nan", min: math.NaN(), max: 1},
		{name: "positive infinity", min: 0, max: math.Inf(1)},
		{name: "zero-only range", min: 0, max: 0},
		{name: "database amount overflow", min: 0, max: MaxLotteryPayoutAmount * 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewLotteryService(&lotteryRepoStub{})
			_, err := svc.CreateCampaign(context.Background(), 1, CreateLotteryCampaignInput{
				Name:      "测试活动",
				WinRate:   100,
				ExpiresAt: time.Now().Add(time.Hour),
				Prizes: []CreateLotteryPrizeInput{{
					Name: "余额", MinAmount: tt.min, MaxAmount: tt.max, Quantity: 1, Weight: 1,
				}},
			})
			if err != ErrLotteryInvalidConfig {
				t.Fatalf("CreateCampaign() error = %v, want ErrLotteryInvalidConfig", err)
			}
		})
	}
}

func TestLotteryCreateRegistrationCampaignKeepsScheduleAndWinnerLimit(t *testing.T) {
	repo := &lotteryRepoStub{}
	svc := NewLotteryService(repo)
	starts := time.Now().Add(-time.Hour)
	deadline := starts.Add(2 * time.Hour)
	drawAt := deadline.Add(time.Hour)

	_, err := svc.CreateCampaign(context.Background(), 1, CreateLotteryCampaignInput{
		Name:                 "报名开奖",
		ActivityType:         LotteryActivityTypeRegistration,
		Description:          "报名后统一开奖",
		MinTotalRecharged:    20,
		StartsAt:             &starts,
		RegistrationDeadline: &deadline,
		DrawAt:               &drawAt,
		WinnerLimit:          8,
	})

	require.NoError(t, err)
	require.NotNil(t, repo.created)
	require.Equal(t, LotteryActivityTypeRegistration, repo.created.ActivityType)
	require.Equal(t, 8, repo.created.WinnerLimit)
	require.Equal(t, 20.0, repo.created.MinTotalRecharged)
	require.Empty(t, repo.created.Prizes)
	require.Equal(t, drawAt, repo.created.ExpiresAt)
}

func TestLotteryCreateRegistrationCampaignRejectsInvalidSchedule(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name                 string
		startsAt             *time.Time
		registrationDeadline *time.Time
		drawAt               *time.Time
		winnerLimit          int
	}{
		{
			name:                 "deadline before start",
			startsAt:             lotteryTimePtr(now.Add(time.Hour)),
			registrationDeadline: lotteryTimePtr(now.Add(30 * time.Minute)),
			drawAt:               lotteryTimePtr(now.Add(2 * time.Hour)),
			winnerLimit:          1,
		},
		{
			name:                 "draw before deadline",
			startsAt:             lotteryTimePtr(now),
			registrationDeadline: lotteryTimePtr(now.Add(2 * time.Hour)),
			drawAt:               lotteryTimePtr(now.Add(time.Hour)),
			winnerLimit:          1,
		},
		{
			name:                 "zero winners",
			startsAt:             lotteryTimePtr(now),
			registrationDeadline: lotteryTimePtr(now.Add(time.Hour)),
			drawAt:               lotteryTimePtr(now.Add(2 * time.Hour)),
			winnerLimit:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewLotteryService(&lotteryRepoStub{})
			_, err := svc.CreateCampaign(context.Background(), 1, CreateLotteryCampaignInput{
				Name:                 "错误报名活动",
				ActivityType:         LotteryActivityTypeRegistration,
				StartsAt:             tt.startsAt,
				RegistrationDeadline: tt.registrationDeadline,
				DrawAt:               tt.drawAt,
				WinnerLimit:          tt.winnerLimit,
			})
			require.ErrorIs(t, err, ErrLotteryInvalidConfig)
		})
	}
}

func lotteryTimePtr(value time.Time) *time.Time { return &value }
