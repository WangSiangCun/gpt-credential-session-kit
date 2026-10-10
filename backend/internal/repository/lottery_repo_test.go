package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newLotteryRepositoryTest(t *testing.T) (*lotteryRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &lotteryRepository{db: db}, mock
}

func expectLotteryCampaignLock(mock sqlmock.Sqlmock, token string, winRate float64) {
	now := time.Now()
	mock.ExpectQuery(`(?s)SELECT id, status, activity_type, win_rate, starts_at, expires_at.*FOR UPDATE`).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "activity_type", "win_rate", "starts_at", "expires_at",
			"min_total_recharged", "registration_deadline", "draw_at", "winner_limit",
		}).AddRow(
			int64(7), "active", service.LotteryActivityTypeInstant, winRate,
			now.Add(-time.Hour), now.Add(time.Hour), 0, nil, nil, 0,
		))
}

func expectNoExistingLotteryDraw(mock sqlmock.Sqlmock, userID int64) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM lottery_draws WHERE campaign_id = $1 AND user_id = $2")).
		WithArgs(int64(7), userID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
}

func expectLotteryRegistrationCampaignLock(
	mock sqlmock.Sqlmock,
	token string,
	startsAt, deadline, drawAt time.Time,
	minTotalRecharged float64,
	winnerLimit int,
) {
	mock.ExpectQuery(`(?s)SELECT id, status, activity_type, win_rate, starts_at, expires_at.*FOR UPDATE`).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "activity_type", "win_rate", "starts_at", "expires_at",
			"min_total_recharged", "registration_deadline", "draw_at", "winner_limit",
		}).AddRow(
			int64(7), "active", service.LotteryActivityTypeRegistration, 0.0,
			startsAt, drawAt, minTotalRecharged, deadline, drawAt, winnerLimit,
		))
}

func expectFixedLotteryPrize(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`(?s)SELECT id, name, min_amount, max_amount, weight.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "min_amount", "max_amount", "weight"}).
			AddRow(int64(5), "固定余额", 1.25, 1.25, int64(1)))
}

func TestLotteryListPublicWinnersFiltersDeletedCampaign(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM lottery_draws d.*JOIN lottery_campaigns c ON c.id = d.campaign_id.*c\.deleted_at IS NULL`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)SELECT d\.id.*FROM lottery_draws d.*JOIN lottery_campaigns c ON c.id = d.campaign_id.*c\.deleted_at IS NULL.*LIMIT \$2 OFFSET \$3`).
		WithArgs(int64(7), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "campaign_id", "user_id", "status", "prize_id", "name", "amount", "created_at",
			"email", "username", "role", "user_status", "balance", "user_created_at",
		}).AddRow(
			int64(21), int64(7), int64(9), "won", int64(5), "余额", 1.25, time.Now(),
			"user@example.com", "用户", "user", "active", 6.25, time.Now(),
		))

	winners, total, err := repo.ListPublicWinners(context.Background(), 7, 1, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, winners, 1)
	require.Equal(t, int64(9), winners[0].UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryDrawRejectsRepeatBeforeRewardMutation(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectBegin()
	expectLotteryCampaignLock(mock, "repeat-token", 100)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM lottery_draws WHERE campaign_id = $1 AND user_id = $2")).
		WithArgs(int64(7), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(12)))
	mock.ExpectRollback()

	draw, err := repo.Draw(context.Background(), "repeat-token", 9)

	require.Nil(t, draw)
	require.ErrorIs(t, err, service.ErrLotteryAlreadyDrawn)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryRegistrationDrawCreatesOneRegistration(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	now := time.Now()
	mock.ExpectBegin()
	expectLotteryRegistrationCampaignLock(mock, "registration-token", now.Add(-time.Hour), now.Add(time.Hour), now.Add(2*time.Hour), 0, 3)
	expectNoExistingLotteryDraw(mock, 9)
	mock.ExpectQuery(`(?s)INSERT INTO lottery_draws.*'registered'.*RETURNING id, created_at`).
		WithArgs(int64(7), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(31), now))
	mock.ExpectCommit()

	draw, err := repo.Draw(context.Background(), "registration-token", 9)

	require.NoError(t, err)
	require.Equal(t, "registered", draw.Status)
	require.Equal(t, int64(31), draw.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryRegistrationDrawRejectsAfterDeadline(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	now := time.Now()
	mock.ExpectBegin()
	expectLotteryRegistrationCampaignLock(mock, "closed-registration-token", now.Add(-2*time.Hour), now.Add(-time.Minute), now.Add(time.Hour), 0, 3)
	mock.ExpectRollback()

	draw, err := repo.Draw(context.Background(), "closed-registration-token", 9)

	require.Nil(t, draw)
	require.ErrorIs(t, err, service.ErrLotteryRegistrationClosed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryRegistrationDrawEnforcesRechargeThreshold(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	now := time.Now()
	mock.ExpectBegin()
	expectLotteryRegistrationCampaignLock(mock, "threshold-registration-token", now.Add(-time.Hour), now.Add(time.Hour), now.Add(2*time.Hour), 10, 3)
	mock.ExpectQuery(`(?s)SELECT GREATEST.*u\.total_recharged.*SUM\(rc\.value\).*effective_total_recharged`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"total_recharged"}).AddRow(9.99))
	mock.ExpectRollback()

	draw, err := repo.Draw(context.Background(), "threshold-registration-token", 9)

	require.Nil(t, draw)
	require.ErrorIs(t, err, service.ErrLotteryThresholdNotMet)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryRechargeThresholdAcceptsHistoricalRechargeAtMinimum(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectBegin()
	tx, err := repo.db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	mock.ExpectQuery(`(?s)SELECT GREATEST.*u\.total_recharged.*SUM\(rc\.value\).*effective_total_recharged`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"effective_total_recharged"}).AddRow(10.0))

	err = repo.ensureRechargeThreshold(context.Background(), tx, 9, 10)

	require.NoError(t, err)
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettleRegistrationCampaignNeverExceedsWinnerLimit(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	now := time.Now()
	mock.ExpectQuery(`(?s)SELECT id FROM lottery_campaigns.*registration_lottery.*draw_at <= NOW`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT activity_type, status, draw_at, winner_limit.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"activity_type", "status", "draw_at", "winner_limit"}).
			AddRow(service.LotteryActivityTypeRegistration, "active", now.Add(-time.Minute), 2))
	mock.ExpectQuery(`(?s)SELECT id.*FROM lottery_draws.*status = 'registered'.*LIMIT \$2.*FOR UPDATE`).
		WithArgs(int64(7), 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(101)).AddRow(int64(102)))
	mock.ExpectExec(`(?s)UPDATE lottery_draws\s+SET status = 'lost'`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 5))
	for _, winnerID := range []int64{101, 102} {
		mock.ExpectExec(`(?s)UPDATE lottery_draws\s+SET status = 'won'.*WHERE id = \$1 AND campaign_id = \$2`).
			WithArgs(winnerID, int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(`(?s)UPDATE lottery_campaigns SET status = 'exhausted'`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SettleDueCampaigns(context.Background())

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryDrawWinnerCommitsRewardAtomically(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectBegin()
	expectLotteryCampaignLock(mock, "winner-token", 100)
	expectNoExistingLotteryDraw(mock, 9)
	expectFixedLotteryPrize(mock)
	mock.ExpectExec(`(?s)UPDATE lottery_prizes SET remaining_quantity = remaining_quantity - 1.*campaign_id = \$2`).
		WithArgs(int64(5), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)INSERT INTO lottery_draws.*RETURNING id, created_at`).
		WithArgs(int64(7), int64(9), "won", sqlmock.AnyArg(), 1.25).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(21), time.Now()))
	mock.ExpectQuery(`(?s)UPDATE users SET balance = balance \+ \$1.*RETURNING balance`).
		WithArgs(1.25, int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(6.25))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), 1.25, int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE lottery_campaigns SET status = 'exhausted'`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	draw, err := repo.Draw(context.Background(), "winner-token", 9)

	require.NoError(t, err)
	require.Equal(t, "won", draw.Status)
	require.Equal(t, 1.25, draw.Amount)
	require.NotNil(t, draw.NewBalance)
	require.Equal(t, 6.25, *draw.NewBalance)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryDrawZeroAmountIsLossWithoutConsumingPrize(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectBegin()
	expectLotteryCampaignLock(mock, "zero-token", 100)
	expectNoExistingLotteryDraw(mock, 9)
	mock.ExpectQuery(`(?s)SELECT id, name, min_amount, max_amount, weight.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "min_amount", "max_amount", "weight"}).
			AddRow(int64(5), "随机余额", 0, 0, int64(1)))
	mock.ExpectQuery(`(?s)INSERT INTO lottery_draws.*RETURNING id, created_at`).
		WithArgs(int64(7), int64(9), "lost", nil, float64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(22), time.Now()))
	mock.ExpectExec(`(?s)UPDATE lottery_campaigns SET status = 'exhausted'`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	draw, err := repo.Draw(context.Background(), "zero-token", 9)

	require.NoError(t, err)
	require.Equal(t, "lost", draw.Status)
	require.Zero(t, draw.Amount)
	require.Nil(t, draw.PrizeID)
	require.Nil(t, draw.NewBalance)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryDrawRollsBackWhenPrizeReservationFails(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectBegin()
	expectLotteryCampaignLock(mock, "empty-token", 100)
	expectNoExistingLotteryDraw(mock, 9)
	expectFixedLotteryPrize(mock)
	mock.ExpectExec(`(?s)UPDATE lottery_prizes SET remaining_quantity = remaining_quantity - 1.*campaign_id = \$2`).
		WithArgs(int64(5), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	draw, err := repo.Draw(context.Background(), "empty-token", 9)

	require.Nil(t, draw)
	require.ErrorIs(t, err, service.ErrLotteryUnavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryDrawRollsBackBalanceWhenAuditRecordFails(t *testing.T) {
	repo, mock := newLotteryRepositoryTest(t)
	mock.ExpectBegin()
	expectLotteryCampaignLock(mock, "audit-failure-token", 100)
	expectNoExistingLotteryDraw(mock, 9)
	expectFixedLotteryPrize(mock)
	mock.ExpectExec(`(?s)UPDATE lottery_prizes SET remaining_quantity = remaining_quantity - 1.*campaign_id = \$2`).
		WithArgs(int64(5), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)INSERT INTO lottery_draws.*RETURNING id, created_at`).
		WithArgs(int64(7), int64(9), "won", sqlmock.AnyArg(), 1.25).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(21), time.Now()))
	mock.ExpectQuery(`(?s)UPDATE users SET balance = balance \+ \$1.*RETURNING balance`).
		WithArgs(1.25, int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(6.25))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), 1.25, int64(9), sqlmock.AnyArg()).
		WillReturnError(errors.New("audit insert failed"))
	mock.ExpectRollback()

	draw, err := repo.Draw(context.Background(), "audit-failure-token", 9)

	require.Nil(t, draw)
	require.EqualError(t, err, "audit insert failed")
	require.NoError(t, mock.ExpectationsWereMet())
}
