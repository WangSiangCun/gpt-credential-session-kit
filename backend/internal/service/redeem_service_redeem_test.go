package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemRejectRepo struct {
	code      RedeemCode
	useCalled bool
	created   []RedeemCode
}

func (r *redeemRejectRepo) Create(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Create call")
}

func (r *redeemRejectRepo) CreateBatch(ctx context.Context, codes []RedeemCode) error {
	r.created = append([]RedeemCode(nil), codes...)
	return nil
}

func TestGenerateCodesAssignsOneSharedRestrictedBatch(t *testing.T) {
	redeemRepo := &redeemRejectRepo{}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	codes, err := redeemService.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count:         3,
		Type:          RedeemTypeBalance,
		Value:         10,
		BatchUserOnce: true,
	})

	require.NoError(t, err)
	require.Len(t, codes, 3)
	require.Len(t, redeemRepo.created, 3)
	require.NotNil(t, codes[0].BatchID)
	require.Len(t, *codes[0].BatchID, 32)
	for _, code := range codes {
		require.True(t, code.BatchUserOnce)
		require.Equal(t, codes[0].BatchID, code.BatchID)
	}
}

func TestGenerateCodesLeavesBatchFieldsEmptyWhenRestrictionDisabled(t *testing.T) {
	redeemRepo := &redeemRejectRepo{}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	codes, err := redeemService.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count: 2,
		Type:  RedeemTypeBalance,
		Value: 10,
	})

	require.NoError(t, err)
	require.Len(t, codes, 2)
	for _, code := range codes {
		require.False(t, code.BatchUserOnce)
		require.False(t, code.GlobalUserOnce)
		require.Nil(t, code.BatchID)
	}
}

func TestGenerateCodesMarksCodesAsGloballyLimited(t *testing.T) {
	redeemRepo := &redeemRejectRepo{}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	codes, err := redeemService.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count:          2,
		Type:           RedeemTypeBalance,
		Value:          10,
		GlobalUserOnce: true,
	})

	require.NoError(t, err)
	require.Len(t, codes, 2)
	for _, code := range codes {
		require.True(t, code.GlobalUserOnce)
		require.False(t, code.BatchUserOnce)
		require.Nil(t, code.BatchID)
	}
}

func TestGenerateCodesRejectsConflictingUserOnceRestrictions(t *testing.T) {
	redeemRepo := &redeemRejectRepo{}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	_, err := redeemService.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count:          2,
		Type:           RedeemTypeBalance,
		Value:          10,
		BatchUserOnce:  true,
		GlobalUserOnce: true,
	})

	require.Error(t, err)
	require.Equal(t, "REDEEM_USER_ONCE_CONFLICT", infraerrors.Reason(err))
	require.Empty(t, redeemRepo.created)
}

func TestGenerateCodesRejectsBatchRestrictionForInvitationCodes(t *testing.T) {
	redeemRepo := &redeemRejectRepo{}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	_, err := redeemService.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count:         2,
		Type:          RedeemTypeInvitation,
		BatchUserOnce: true,
	})

	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, "REDEEM_BATCH_USER_ONCE_UNSUPPORTED", infraerrors.Reason(err))
	require.Empty(t, redeemRepo.created)
}

func TestGenerateCodesRejectsGlobalRestrictionForInvitationCodes(t *testing.T) {
	redeemRepo := &redeemRejectRepo{}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	_, err := redeemService.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count:          2,
		Type:           RedeemTypeInvitation,
		GlobalUserOnce: true,
	})

	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Empty(t, redeemRepo.created)
}

func (r *redeemRejectRepo) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	if r.code.ID != id {
		return nil, ErrRedeemCodeNotFound
	}
	clone := r.code
	return &clone, nil
}

func (r *redeemRejectRepo) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	if r.code.Code != code {
		return nil, ErrRedeemCodeNotFound
	}
	clone := r.code
	return &clone, nil
}

func (r *redeemRejectRepo) Update(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Update call")
}

func (r *redeemRejectRepo) BatchUpdate(ctx context.Context, ids []int64, fields RedeemCodeBatchUpdateFields) (int64, error) {
	panic("unexpected BatchUpdate call")
}

func (r *redeemRejectRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (r *redeemRejectRepo) Use(ctx context.Context, id, userID int64) error {
	r.useCalled = true
	r.code.Status = StatusUsed
	r.code.UsedBy = &userID
	return nil
}

func (r *redeemRejectRepo) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *redeemRejectRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *redeemRejectRepo) ListByUser(ctx context.Context, userID int64, limit int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (r *redeemRejectRepo) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (r *redeemRejectRepo) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

func TestRedeemRejectsInvitationCodeBeforeTransaction(t *testing.T) {
	ctx := context.Background()
	redeemRepo := &redeemRejectRepo{
		code: RedeemCode{
			ID:     1,
			Code:   "INVITE-001",
			Type:   RedeemTypeInvitation,
			Status: StatusUnused,
		},
	}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	got, err := redeemService.Redeem(ctx, 2, redeemRepo.code.Code)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, "REDEEM_CODE_UNSUPPORTED_TYPE", infraerrors.Reason(err))
	require.Equal(t, "invitation codes can only be used during registration", infraerrors.Message(err))
	require.False(t, redeemRepo.useCalled)
	require.Equal(t, StatusUnused, redeemRepo.code.Status)
	require.Nil(t, redeemRepo.code.UsedBy)
}
