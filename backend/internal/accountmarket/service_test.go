package accountmarket

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type stringArrayArgument string

func (want stringArrayArgument) Match(value driver.Value) bool {
	encoded, ok := value.(string)
	return ok && encoded == string(want)
}

func TestParseUploadAcceptsSupportedShapes(t *testing.T) {
	tests := []struct {
		name string
		json string
		want int
	}{
		{"single", `{"name":"one","platform":"openai","type":"oauth","credentials":{"email":"one@example.com"}}`, 1},
		{"array", `[{"name":"one"},{"name":"two"}]`, 2},
		{"wrapper", `{"accounts":[{"name":"one"},{"name":"two"}]}`, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := ParseUpload([]byte(tt.json))
			require.NoError(t, err)
			require.Len(t, items, tt.want)
		})
	}
}

func TestParseUploadRejectsInvalidAndOversizedPayloads(t *testing.T) {
	_, err := ParseUpload([]byte(`not-json`))
	require.Error(t, err)

	_, err = ParseUpload([]byte(`[]`))
	require.Error(t, err)

	_, err = ParseUpload([]byte(strings.Repeat("x", MaxUploadBytes+1)))
	require.Error(t, err)
}

func TestExtractAccountEmailUsesAccessTokenBeforeFields(t *testing.T) {
	claims, err := json.Marshal(map[string]any{"email": "jwt@example.com"})
	require.NoError(t, err)
	token := "header." + base64.RawURLEncoding.EncodeToString(claims) + ".signature"

	require.Equal(t, "jwt@example.com", extractAccountEmail(
		map[string]any{"email": "field@example.com", "access_token": token}, nil,
	))
	require.Equal(t, "jwt@example.com", extractAccountEmail(
		map[string]any{"access_token": token}, nil,
	))
}

func TestCredentialFingerprintIsStableAcrossMapOrder(t *testing.T) {
	first := credentialFingerprint("openai", "oauth", map[string]any{"b": 2, "a": 1})
	second := credentialFingerprint("openai", "oauth", map[string]any{"a": 1, "b": 2})
	require.Equal(t, first, second)
	require.Len(t, first, 64)
}

func TestCredentialFingerprintUsesStableJWTIdentity(t *testing.T) {
	claims, err := json.Marshal(map[string]any{"sub": "user-123", "email": "owner@example.com"})
	require.NoError(t, err)
	firstToken := "header." + base64.RawURLEncoding.EncodeToString(claims) + ".signature-one"
	secondToken := "header." + base64.RawURLEncoding.EncodeToString(claims) + ".signature-two"

	first := credentialFingerprint("openai", "oauth", map[string]any{"access_token": firstToken})
	second := credentialFingerprint("openai", "oauth", map[string]any{"access_token": secondToken})
	require.Equal(t, first, second)
}

func TestContributionForbiddenKeyRejectsOperatorRoutingFields(t *testing.T) {
	require.Equal(t, "base_url", contributionForbiddenKey(map[string]any{"base_url": "https://example.com"}))
	require.Equal(t, "headers", contributionForbiddenKey(map[string]any{
		"nested": map[string]any{"headers": map[string]any{"x-test": "value"}},
	}))
	require.Empty(t, contributionForbiddenKey(map[string]any{
		"access_token": "token", "account_id": "account-1", "email": "owner@example.com",
	}))
}

func TestMaskEmail(t *testing.T) {
	require.Equal(t, "a***e@example.com", maskEmail("alice@example.com"))
	require.Equal(t, "***@example.com", maskEmail("ab@example.com"))
	require.Empty(t, maskEmail(""))
}

func TestAccountMarketQuotaUsesExistingProviderSnapshots(t *testing.T) {
	quota, reset := accountMarketQuota(&service.Account{
		Platform: service.PlatformOpenAI,
		Extra: map[string]any{
			"codex_5h_used_percent": 25.0,
			"codex_5h_reset_at":     "2026-09-19T12:00:00Z",
			"codex_7d_used_percent": 60.0,
		},
	})
	require.Equal(t, "5h 75% · 7d 40%", quota)
	require.Equal(t, "2026-09-19T12:00:00Z", reset)

	windowEnd := time.Date(2026, 9, 19, 13, 0, 0, 0, time.UTC)
	quota, reset = accountMarketQuota(&service.Account{
		Platform:         service.PlatformAnthropic,
		SessionWindowEnd: &windowEnd,
		Extra: map[string]any{
			"session_window_utilization":   0.1,
			"passive_usage_7d_utilization": 0.75,
		},
	})
	require.Equal(t, "5h 90% · 7d 25%", quota)
	require.Equal(t, &windowEnd, reset)
}

func TestIsDuplicateDetectsJWTEmailInExistingAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	claims, err := json.Marshal(map[string]any{"email": "owner@example.com"})
	require.NoError(t, err)
	token := "header." + base64.RawURLEncoding.EncodeToString(claims) + ".signature"
	credentials, err := json.Marshal(map[string]any{"access_token": token})
	require.NoError(t, err)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(digest("owner@example.com"), sqlmock.AnyArg(), "openai", "oauth").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT credentials, extra FROM accounts").
		WithArgs("openai", "oauth").
		WillReturnRows(sqlmock.NewRows([]string{"credentials", "extra"}).AddRow(credentials, []byte(`{}`)))

	svc := &Service{db: db}
	duplicate, err := svc.isDuplicate(
		context.Background(), "openai", "oauth", "owner@example.com",
		digest("owner@example.com"), credentialFingerprint("openai", "oauth", map[string]any{"other": true}),
	)
	require.NoError(t, err)
	require.True(t, duplicate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestWithdrawalLocksOnlySummedEarnings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("INSERT INTO account_market_earnings").
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT enabled, target_group_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"enabled", "target_group_id", "commission_percent", "minimum_withdrawal", "withdrawal_review_required",
		}).AddRow(true, nil, 10, 1, true))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, gross_amount FROM account_market_earnings").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "gross_amount"}).AddRow(11, 4).AddRow(12, 6))
	mock.ExpectQuery("INSERT INTO account_market_withdrawals").
		WithArgs(int64(7), float64(10), float64(10), float64(1), float64(9), "alipay", "qr-code", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(99))
	mock.ExpectExec("(?s)UPDATE account_market_earnings.*id=ANY\\(\\$3\\)").
		WithArgs(int64(99), int64(7), stringArrayArgument("{11,12}")).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	now := time.Now()
	mock.ExpectQuery("SELECT w.id, w.user_id").
		WithArgs(int64(99), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "email", "gross_amount", "commission_percent", "commission_amount", "net_amount",
			"payout_method", "payout_account", "status", "review_note", "reviewed_at", "paid_at", "created_at",
		}).AddRow(99, 7, "user@example.com", 10, 10, 1, 9, "alipay", "qr-code", "pending", "", nil, nil, now))

	withdrawal, err := (&Service{db: db}).RequestWithdrawal(context.Background(), 7, WithdrawalRequest{
		PayoutMethod: "alipay", PayoutAccount: "qr-code",
	})
	require.NoError(t, err)
	require.Equal(t, int64(99), withdrawal.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
