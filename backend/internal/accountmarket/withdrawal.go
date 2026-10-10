package accountmarket

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
)

func (s *Service) RequestWithdrawal(ctx context.Context, userID int64, req WithdrawalRequest) (*Withdrawal, error) {
	if err := s.SyncEarnings(ctx, userID); err != nil {
		return nil, err
	}
	req.PayoutMethod = strings.TrimSpace(req.PayoutMethod)
	req.PayoutAccount = strings.TrimSpace(req.PayoutAccount)
	if req.PayoutMethod == "" || req.PayoutAccount == "" {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_PAYOUT_REQUIRED", "请填写收款方式和收款信息")
	}
	if len(req.PayoutMethod) > 32 || len(req.PayoutAccount) > 2000 {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_PAYOUT_TOO_LONG", "收款信息长度超出限制")
	}
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, gross_amount FROM account_market_earnings
		WHERE uploader_user_id=$1 AND status='available'
		ORDER BY id FOR UPDATE
	`, userID)
	if err != nil {
		return nil, err
	}
	var gross float64
	var earningIDs []int64
	for rows.Next() {
		var id int64
		var amount float64
		if scanErr := rows.Scan(&id, &amount); scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		earningIDs = append(earningIDs, id)
		gross += amount
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	gross = roundMoney(gross)
	if gross <= 0 || gross < cfg.MinimumWithdrawal {
		return nil, appError(http.StatusBadRequest, "ACCOUNT_MARKET_WITHDRAWAL_TOO_SMALL", "可提现收益未达到最低提现金额")
	}
	commission := roundMoney(gross * cfg.CommissionPercent / 100)
	netAmount := roundMoney(gross - commission)
	status := "pending"
	if !cfg.WithdrawalReviewRequired {
		status = "approved"
	}

	var withdrawalID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO account_market_withdrawals
		(user_id, gross_amount, commission_percent, commission_amount, net_amount,
		 payout_method, payout_account, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, userID, gross, cfg.CommissionPercent, commission, netAmount,
		req.PayoutMethod, req.PayoutAccount, status).Scan(&withdrawalID)
	if err != nil {
		return nil, err
	}
	if len(earningIDs) == 0 {
		return nil, appError(http.StatusConflict, "ACCOUNT_MARKET_NO_AVAILABLE_EARNINGS", "暂无可提现收益")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE account_market_earnings
		SET status='locked', withdrawal_id=$1, updated_at=NOW()
		WHERE uploader_user_id=$2 AND status='available' AND id=ANY($3)
	`, withdrawalID, userID, pq.Array(earningIDs)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.getWithdrawal(ctx, withdrawalID, userID)
}

func (s *Service) CancelWithdrawal(ctx context.Context, userID, withdrawalID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM account_market_withdrawals
		WHERE id=$1 AND user_id=$2 FOR UPDATE
	`, withdrawalID, userID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return appError(http.StatusNotFound, "ACCOUNT_MARKET_WITHDRAWAL_NOT_FOUND", "提现记录不存在")
	}
	if err != nil {
		return err
	}
	if status != "pending" {
		return appError(http.StatusConflict, "ACCOUNT_MARKET_WITHDRAWAL_NOT_PENDING", "只有待审核的提现可以取消")
	}
	if err := releaseWithdrawalEarnings(ctx, tx, withdrawalID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE account_market_withdrawals SET status='cancelled', updated_at=NOW()
		WHERE id=$1
	`, withdrawalID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ReviewWithdrawal(ctx context.Context, adminID, withdrawalID int64, req ReviewRequest) error {
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status != "approved" && req.Status != "paid" && req.Status != "rejected" {
		return appError(http.StatusBadRequest, "ACCOUNT_MARKET_BAD_WITHDRAWAL_STATUS", "状态必须是 approved、paid 或 rejected")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var current string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM account_market_withdrawals WHERE id=$1 FOR UPDATE
	`, withdrawalID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return appError(http.StatusNotFound, "ACCOUNT_MARKET_WITHDRAWAL_NOT_FOUND", "提现记录不存在")
	}
	if err != nil {
		return err
	}
	if current == "paid" || current == "rejected" || current == "cancelled" {
		return appError(http.StatusConflict, "ACCOUNT_MARKET_WITHDRAWAL_FINAL", "该提现记录已经结束")
	}
	if req.Status == "rejected" {
		if err := releaseWithdrawalEarnings(ctx, tx, withdrawalID); err != nil {
			return err
		}
	}
	if req.Status == "paid" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE account_market_earnings SET status='withdrawn', updated_at=NOW()
			WHERE withdrawal_id=$1 AND status='locked'
		`, withdrawalID); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		UPDATE account_market_withdrawals
		SET status=$1, review_note=$2, reviewed_by=$3, reviewed_at=$4,
		    paid_at=CASE WHEN $1='paid' THEN $4 ELSE paid_at END,
		    updated_at=NOW()
		WHERE id=$5
	`, req.Status, strings.TrimSpace(req.Note), adminID, now, withdrawalID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func releaseWithdrawalEarnings(ctx context.Context, tx *sql.Tx, withdrawalID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE account_market_earnings
		SET status='available', withdrawal_id=NULL, updated_at=NOW()
		WHERE withdrawal_id=$1 AND status='locked'
	`, withdrawalID)
	return err
}

func (s *Service) ListWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error) {
	return s.listWithdrawals(ctx, `WHERE w.user_id=$1`, userID)
}

func (s *Service) ListAdminWithdrawals(ctx context.Context) ([]Withdrawal, error) {
	return s.listWithdrawals(ctx, "")
}

func (s *Service) getWithdrawal(ctx context.Context, withdrawalID, userID int64) (*Withdrawal, error) {
	items, err := s.listWithdrawals(ctx, `WHERE w.id=$1 AND w.user_id=$2`, withdrawalID, userID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, appError(http.StatusNotFound, "ACCOUNT_MARKET_WITHDRAWAL_NOT_FOUND", "提现记录不存在")
	}
	return &items[0], nil
}

func (s *Service) listWithdrawals(ctx context.Context, where string, args ...any) ([]Withdrawal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.user_id, COALESCE(u.email,''), w.gross_amount,
		       w.commission_percent, w.commission_amount, w.net_amount,
		       w.payout_method, w.payout_account, w.status,
		       COALESCE(w.review_note,''), w.reviewed_at, w.paid_at, w.created_at
		FROM account_market_withdrawals w
		LEFT JOIN users u ON u.id=w.user_id
		`+where+`
		ORDER BY w.created_at DESC LIMIT 500
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Withdrawal, 0)
	for rows.Next() {
		var item Withdrawal
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.UserEmail, &item.GrossAmount,
			&item.CommissionPercent, &item.CommissionAmount, &item.NetAmount,
			&item.PayoutMethod, &item.PayoutAccount, &item.Status,
			&item.ReviewNote, &item.ReviewedAt, &item.PaidAt, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
