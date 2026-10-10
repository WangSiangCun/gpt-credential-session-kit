package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type lotteryRepository struct {
	db *sql.DB
}

func NewLotteryRepository(db *sql.DB) service.LotteryRepository {
	return &lotteryRepository{db: db}
}

func (r *lotteryRepository) CreateCampaign(ctx context.Context, adminID int64, token string, input service.CreateLotteryCampaignInput) (*service.LotteryCampaign, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var campaign service.LotteryCampaign
	err = tx.QueryRowContext(ctx, `
		INSERT INTO lottery_campaigns
			(token, name, description, activity_type, win_rate, min_total_recharged,
			 starts_at, expires_at, registration_deadline, draw_at, winner_limit, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, token, name, description, activity_type, status, win_rate,
			min_total_recharged, starts_at, expires_at, created_by, created_at, updated_at`,
		token, input.Name, input.Description, input.ActivityType, input.WinRate, input.MinTotalRecharged,
		*input.StartsAt, input.ExpiresAt, input.RegistrationDeadline, input.DrawAt, input.WinnerLimit, adminID,
	).Scan(&campaign.ID, &campaign.Token, &campaign.Name, &campaign.Description, &campaign.ActivityType,
		&campaign.Status, &campaign.WinRate, &campaign.MinTotalRecharged, &campaign.StartsAt,
		&campaign.ExpiresAt, &campaign.CreatedBy, &campaign.CreatedAt, &campaign.UpdatedAt)
	if err != nil {
		return nil, err
	}
	campaign.RegistrationDeadline = input.RegistrationDeadline
	campaign.DrawAt = input.DrawAt
	campaign.WinnerLimit = input.WinnerLimit
	campaign.Prizes = make([]service.LotteryPrize, 0, len(input.Prizes))
	for i, p := range input.Prizes {
		var prize service.LotteryPrize
		err = tx.QueryRowContext(ctx, `
			INSERT INTO lottery_prizes
				(campaign_id, name, min_amount, max_amount, total_quantity, remaining_quantity, weight, sort_order)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $7)
			RETURNING id, name, min_amount, max_amount, total_quantity, remaining_quantity, weight, sort_order, created_at`,
			campaign.ID, p.Name, p.MinAmount, p.MaxAmount, p.Quantity, p.Weight, i,
		).Scan(&prize.ID, &prize.Name, &prize.MinAmount, &prize.MaxAmount, &prize.TotalQuantity,
			&prize.RemainingQuantity, &prize.Weight, &prize.SortOrder, &prize.CreatedAt)
		if err != nil {
			return nil, err
		}
		campaign.Prizes = append(campaign.Prizes, prize)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &campaign, nil
}

func (r *lotteryRepository) ListCampaigns(ctx context.Context, page, pageSize int) ([]service.LotteryCampaign, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lottery_campaigns WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.token, c.name, c.description,
			CASE
				WHEN c.activity_type = 'registration_lottery' AND c.status = 'active' AND c.draw_at <= NOW() THEN 'exhausted'
				WHEN c.status = 'active' AND c.expires_at <= NOW() THEN 'expired'
				WHEN c.status = 'active' AND c.starts_at > NOW() THEN 'scheduled'
				ELSE c.status
			END,
			c.activity_type, c.win_rate, c.min_total_recharged, c.starts_at, c.expires_at,
			c.registration_deadline, c.draw_at, c.winner_limit,
			c.created_by, c.created_at, c.updated_at,
			COUNT(d.id) FILTER (WHERE d.status = 'won'), COUNT(d.id)
		FROM lottery_campaigns c
		LEFT JOIN lottery_draws d ON d.campaign_id = c.id
		WHERE c.deleted_at IS NULL
		GROUP BY c.id ORDER BY c.created_at DESC LIMIT $1 OFFSET $2`,
		pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.LotteryCampaign, 0)
	for rows.Next() {
		var c service.LotteryCampaign
		var registrationDeadline, drawAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.Token, &c.Name, &c.Description, &c.Status, &c.ActivityType, &c.WinRate,
			&c.MinTotalRecharged, &c.StartsAt, &c.ExpiresAt, &registrationDeadline, &drawAt, &c.WinnerLimit,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.WinnerCount, &c.DrawCount); err != nil {
			return nil, 0, err
		}
		c.RegistrationDeadline = lotteryNullableTime(registrationDeadline)
		c.DrawAt = lotteryNullableTime(drawAt)
		items = append(items, c)
	}
	return items, total, rows.Err()
}

func (r *lotteryRepository) GetCampaignByID(ctx context.Context, id int64) (*service.LotteryCampaign, error) {
	return r.getCampaign(ctx, `c.id = $1`, id)
}

func (r *lotteryRepository) GetCampaignByToken(ctx context.Context, token string, userID int64) (*service.LotteryCampaign, *service.LotteryDraw, error) {
	campaign, err := r.getCampaign(ctx, `c.token = $1`, token)
	if err != nil {
		return nil, nil, err
	}
	var draw service.LotteryDraw
	var prizeID sql.NullInt64
	var prizeName sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT d.id, d.campaign_id, d.user_id, d.status, d.prize_id, COALESCE(p.name, ''), d.amount, d.created_at
		FROM lottery_draws d LEFT JOIN lottery_prizes p ON p.id = d.prize_id
		WHERE d.campaign_id = $1 AND d.user_id = $2`, campaign.ID, userID).
		Scan(&draw.ID, &draw.CampaignID, &draw.UserID, &draw.Status, &prizeID, &prizeName, &draw.Amount, &draw.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return campaign, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if prizeID.Valid {
		draw.PrizeID = &prizeID.Int64
	}
	draw.PrizeName = prizeName.String
	return campaign, &draw, nil
}

func (r *lotteryRepository) getCampaign(ctx context.Context, where string, arg any) (*service.LotteryCampaign, error) {
	var c service.LotteryCampaign
	var registrationDeadline, drawAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.token, c.name, c.description,
			CASE
				WHEN c.activity_type = 'registration_lottery' AND c.status = 'active' AND c.draw_at <= NOW() THEN 'exhausted'
				WHEN c.status = 'active' AND c.expires_at <= NOW() THEN 'expired'
				WHEN c.status = 'active' AND c.starts_at > NOW() THEN 'scheduled'
				ELSE c.status
			END,
			c.activity_type, c.win_rate, c.min_total_recharged, c.starts_at, c.expires_at,
			c.registration_deadline, c.draw_at, c.winner_limit,
			c.created_by, c.created_at, c.updated_at,
			COUNT(d.id) FILTER (WHERE d.status = 'won'), COUNT(d.id)
		FROM lottery_campaigns c LEFT JOIN lottery_draws d ON d.campaign_id = c.id
		WHERE c.deleted_at IS NULL AND `+where+` GROUP BY c.id`,
		arg).Scan(&c.ID, &c.Token, &c.Name, &c.Description, &c.Status, &c.ActivityType, &c.WinRate,
		&c.MinTotalRecharged, &c.StartsAt, &c.ExpiresAt, &registrationDeadline, &drawAt, &c.WinnerLimit,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.WinnerCount, &c.DrawCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrLotteryNotFound
	}
	if err != nil {
		return nil, err
	}
	c.RegistrationDeadline = lotteryNullableTime(registrationDeadline)
	c.DrawAt = lotteryNullableTime(drawAt)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, min_amount, max_amount, total_quantity, remaining_quantity, weight, sort_order, created_at
		FROM lottery_prizes WHERE campaign_id = $1 ORDER BY sort_order, id`, c.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	c.Prizes = []service.LotteryPrize{}
	for rows.Next() {
		var p service.LotteryPrize
		if err := rows.Scan(&p.ID, &p.Name, &p.MinAmount, &p.MaxAmount, &p.TotalQuantity,
			&p.RemainingQuantity, &p.Weight, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		c.Prizes = append(c.Prizes, p)
	}
	return &c, rows.Err()
}

func (r *lotteryRepository) StopCampaign(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE lottery_campaigns SET status = 'stopped', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrLotteryNotFound
	}
	return nil
}

func (r *lotteryRepository) DeleteCampaign(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE lottery_campaigns
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND (status IN ('stopped', 'exhausted') OR expires_at <= NOW())`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n > 0 {
		return nil
	}

	var exists bool
	if err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM lottery_campaigns WHERE id = $1 AND deleted_at IS NULL)`, id,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return service.ErrLotteryNotFound
	}
	return service.ErrLotteryDeleteDenied
}

func (r *lotteryRepository) ListWinners(ctx context.Context, campaignID int64, page, pageSize int) ([]service.LotteryDraw, int64, error) {
	return r.listWinners(ctx, campaignID, page, pageSize, false)
}

func (r *lotteryRepository) ListPublicWinners(ctx context.Context, campaignID int64, page, pageSize int) ([]service.LotteryDraw, int64, error) {
	return r.listWinners(ctx, campaignID, page, pageSize, true)
}

// SettleDueCampaigns finalizes registration campaigns once their draw time has
// passed. The campaign row is locked before selecting entrants, so concurrent
// requests cannot select more than the configured winner count.
func (r *lotteryRepository) SettleDueCampaigns(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM lottery_campaigns
		WHERE activity_type = 'registration_lottery'
		  AND status = 'active'
		  AND draw_at IS NOT NULL
		  AND draw_at <= NOW()
		  AND deleted_at IS NULL
		ORDER BY id`)
	if err != nil {
		return err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, id := range ids {
		if err := r.settleRegistrationCampaign(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *lotteryRepository) settleRegistrationCampaign(ctx context.Context, campaignID int64) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var activityType, status string
	var drawAt sql.NullTime
	var winnerLimit int
	err = tx.QueryRowContext(ctx, `
		SELECT activity_type, status, draw_at, winner_limit
		FROM lottery_campaigns
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, campaignID).Scan(&activityType, &status, &drawAt, &winnerLimit)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if activityType != service.LotteryActivityTypeRegistration || status != "active" ||
		!drawAt.Valid || time.Now().Before(drawAt.Time) {
		return nil
	}
	if winnerLimit <= 0 {
		return service.ErrLotteryInvalidConfig
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM lottery_draws
		WHERE campaign_id = $1 AND status = 'registered'
		ORDER BY random()
		LIMIT $2
		FOR UPDATE`, campaignID, winnerLimit)
	if err != nil {
		return err
	}
	winnerIDs := make([]int64, 0, winnerLimit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		winnerIDs = append(winnerIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if _, err := tx.ExecContext(ctx, `
		UPDATE lottery_draws
		SET status = 'lost'
		WHERE campaign_id = $1 AND status = 'registered'`, campaignID); err != nil {
		return err
	}
	for _, winnerID := range winnerIDs {
		if _, err := tx.ExecContext(ctx, `
			UPDATE lottery_draws
			SET status = 'won', amount = 0, prize_id = NULL
			WHERE id = $1 AND campaign_id = $2`, winnerID, campaignID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE lottery_campaigns SET status = 'exhausted', updated_at = NOW()
		WHERE id = $1`, campaignID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *lotteryRepository) listWinners(ctx context.Context, campaignID int64, page, pageSize int, publicOnly bool) ([]service.LotteryDraw, int64, error) {
	from := `
		FROM lottery_draws d
		LEFT JOIN lottery_prizes p ON p.id = d.prize_id
		JOIN users u ON u.id = d.user_id`
	where := `WHERE d.campaign_id = $1 AND d.status = 'won'`
	if publicOnly {
		from = `
		FROM lottery_draws d
		JOIN lottery_campaigns c ON c.id = d.campaign_id
		LEFT JOIN lottery_prizes p ON p.id = d.prize_id
		JOIN users u ON u.id = d.user_id`
		where += ` AND c.deleted_at IS NULL`
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) `+from+` `+where, campaignID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.campaign_id, d.user_id, d.status, d.prize_id, p.name, d.amount, d.created_at,
			u.email, COALESCE(u.username, ''), u.role, u.status, u.balance, u.created_at
		`+from+` `+where+`
		ORDER BY d.created_at DESC LIMIT $2 OFFSET $3`, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.LotteryDraw, 0)
	for rows.Next() {
		var d service.LotteryDraw
		var prizeID sql.NullInt64
		var prizeName sql.NullString
		if err := rows.Scan(&d.ID, &d.CampaignID, &d.UserID, &d.Status, &prizeID, &prizeName, &d.Amount,
			&d.CreatedAt, &d.UserEmail, &d.Username, &d.UserRole, &d.UserStatus, &d.UserBalance, &d.UserCreatedAt); err != nil {
			return nil, 0, err
		}
		if prizeID.Valid {
			d.PrizeID = &prizeID.Int64
		}
		d.PrizeName = prizeName.String
		items = append(items, d)
	}
	return items, total, rows.Err()
}

func (r *lotteryRepository) Draw(ctx context.Context, token string, userID int64) (*service.LotteryDraw, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var campaignID int64
	var activityType, status string
	var winRate float64
	var startsAt, expiresAt time.Time
	var minTotalRecharged float64
	var registrationDeadline, drawAt sql.NullTime
	var winnerLimit int
	err = tx.QueryRowContext(ctx, `
		SELECT id, status, activity_type, win_rate, starts_at, expires_at,
			min_total_recharged, registration_deadline, draw_at, winner_limit
		FROM lottery_campaigns WHERE token = $1 AND deleted_at IS NULL FOR UPDATE`, token).
		Scan(&campaignID, &status, &activityType, &winRate, &startsAt, &expiresAt,
			&minTotalRecharged, &registrationDeadline, &drawAt, &winnerLimit)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrLotteryNotFound
	}
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if activityType == service.LotteryActivityTypeRegistration {
		if status != "active" || now.Before(startsAt) || !drawAt.Valid || !now.Before(drawAt.Time) {
			return nil, service.ErrLotteryUnavailable
		}
		if !registrationDeadline.Valid || !now.Before(registrationDeadline.Time) {
			return nil, service.ErrLotteryRegistrationClosed
		}
		if err := r.ensureRechargeThreshold(ctx, tx, userID, minTotalRecharged); err != nil {
			return nil, err
		}
		if winnerLimit <= 0 {
			return nil, service.ErrLotteryInvalidConfig
		}
		var existingID int64
		err = tx.QueryRowContext(ctx, `SELECT id FROM lottery_draws WHERE campaign_id = $1 AND user_id = $2`,
			campaignID, userID).Scan(&existingID)
		if err == nil {
			return nil, service.ErrLotteryAlreadyDrawn
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		draw := &service.LotteryDraw{CampaignID: campaignID, UserID: userID, Status: "registered"}
		err = tx.QueryRowContext(ctx, `
			INSERT INTO lottery_draws (campaign_id, user_id, status, prize_id, amount)
			VALUES ($1, $2, 'registered', NULL, 0)
			RETURNING id, created_at`, campaignID, userID).Scan(&draw.ID, &draw.CreatedAt)
		if err != nil {
			if isLotteryUniqueViolation(err) {
				return nil, service.ErrLotteryAlreadyDrawn
			}
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return draw, nil
	}
	if status != "active" || now.Before(startsAt) || !now.Before(expiresAt) {
		return nil, service.ErrLotteryUnavailable
	}
	if err := r.ensureRechargeThreshold(ctx, tx, userID, minTotalRecharged); err != nil {
		return nil, err
	}

	var existingID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM lottery_draws WHERE campaign_id = $1 AND user_id = $2`,
		campaignID, userID).Scan(&existingID)
	if err == nil {
		return nil, service.ErrLotteryAlreadyDrawn
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	type prizeRow struct {
		id, weight int64
		name       string
		min, max   float64
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, name, min_amount, max_amount, weight
		FROM lottery_prizes WHERE campaign_id = $1 AND remaining_quantity > 0
		ORDER BY sort_order, id FOR UPDATE`, campaignID)
	if err != nil {
		return nil, err
	}
	var prizes []prizeRow
	var totalWeight int64
	for rows.Next() {
		var p prizeRow
		if err := rows.Scan(&p.id, &p.name, &p.min, &p.max, &p.weight); err != nil {
			rows.Close()
			return nil, err
		}
		if p.weight <= 0 || totalWeight > math.MaxInt64-p.weight {
			rows.Close()
			return nil, service.ErrLotteryInvalidConfig
		}
		prizes = append(prizes, p)
		totalWeight += p.weight
	}
	rows.Close()
	if len(prizes) == 0 {
		_, _ = tx.ExecContext(ctx, `UPDATE lottery_campaigns SET status = 'exhausted', updated_at = NOW() WHERE id = $1`, campaignID)
		return nil, service.ErrLotteryUnavailable
	}

	draw := &service.LotteryDraw{CampaignID: campaignID, UserID: userID, Status: "lost"}
	winRoll, err := secureRandomInt(1_000_000)
	if err != nil {
		return nil, err
	}
	if float64(winRoll) < winRate*10_000 {
		choice, err := secureRandomInt(totalWeight)
		if err != nil {
			return nil, err
		}
		var selected prizeRow
		for _, p := range prizes {
			if choice < p.weight {
				selected = p
				break
			}
			choice -= p.weight
		}
		minCents, err := lotteryAmountToCents(selected.min)
		if err != nil {
			return nil, err
		}
		maxCents, err := lotteryAmountToCents(selected.max)
		if err != nil {
			return nil, err
		}
		amountCents := minCents
		if maxCents > minCents {
			offset, err := secureRandomInt(maxCents - minCents + 1)
			if err != nil {
				return nil, err
			}
			amountCents += offset
		}
		if amountCents > 0 {
			draw.Status = "won"
			draw.PrizeID = &selected.id
			draw.PrizeName = selected.name
			draw.Amount = float64(amountCents) / 100
			updateResult, err := tx.ExecContext(ctx, `
				UPDATE lottery_prizes SET remaining_quantity = remaining_quantity - 1
				WHERE id = $1 AND campaign_id = $2 AND remaining_quantity > 0`, selected.id, campaignID)
			if err != nil {
				return nil, err
			}
			affected, err := updateResult.RowsAffected()
			if err != nil {
				return nil, err
			}
			if affected != 1 {
				return nil, service.ErrLotteryUnavailable
			}
		}
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO lottery_draws (campaign_id, user_id, status, prize_id, amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`, campaignID, userID, draw.Status, draw.PrizeID, draw.Amount).
		Scan(&draw.ID, &draw.CreatedAt)
	if err != nil {
		if isLotteryUniqueViolation(err) {
			return nil, service.ErrLotteryAlreadyDrawn
		}
		return nil, err
	}

	if draw.Status == "won" {
		var newBalance float64
		if err := tx.QueryRowContext(ctx, `
			UPDATE users SET balance = balance + $1, updated_at = NOW()
			WHERE id = $2 AND deleted_at IS NULL RETURNING balance`, draw.Amount, userID).Scan(&newBalance); err != nil {
			return nil, err
		}
		draw.NewBalance = &newBalance
		codeBytes := make([]byte, 12)
		if _, err := rand.Read(codeBytes); err != nil {
			return nil, err
		}
		code := "LOT-" + hex.EncodeToString(codeBytes)
		notes := fmt.Sprintf("抽奖活动 #%d：%s（随机余额 %.2f 元）", campaignID, draw.PrizeName, draw.Amount)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes)
			VALUES ($1, 'lottery_balance', $2, 'used', $3, NOW(), $4)`,
			code, draw.Amount, userID, notes); err != nil {
			return nil, err
		}
	}

	_, _ = tx.ExecContext(ctx, `
		UPDATE lottery_campaigns SET status = 'exhausted', updated_at = NOW()
		WHERE id = $1 AND NOT EXISTS (
			SELECT 1 FROM lottery_prizes WHERE campaign_id = $1 AND remaining_quantity > 0
		)`, campaignID)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return draw, nil
}

func secureRandomInt(max int64) (int64, error) {
	if max <= 0 {
		return 0, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func lotteryAmountToCents(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > service.MaxLotteryPayoutAmount {
		return 0, service.ErrLotteryInvalidConfig
	}
	return int64(math.Round(value * 100)), nil
}

func (r *lotteryRepository) ensureRechargeThreshold(ctx context.Context, tx *sql.Tx, userID int64, minimum float64) error {
	if minimum <= 0 {
		return nil
	}
	var totalRecharged float64
	if err := tx.QueryRowContext(ctx, `
		SELECT GREATEST(
			u.total_recharged,
			COALESCE((
				SELECT SUM(rc.value)
				FROM redeem_codes rc
				WHERE rc.used_by = u.id
				  AND rc.value > 0
				  AND rc.type IN ('balance', 'admin_balance')
			), 0)
		) AS effective_total_recharged
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL`, userID).Scan(&totalRecharged); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrLotteryUnavailable
		}
		return err
	}
	if totalRecharged+0.00000001 < minimum {
		return service.ErrLotteryThresholdNotMet
	}
	return nil
}

func lotteryNullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func isLotteryUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
