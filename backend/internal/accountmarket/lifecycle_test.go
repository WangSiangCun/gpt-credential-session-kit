package accountmarket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type lifecycleAdmin struct {
	service.AdminService
	account   service.Account
	proxy     *service.Proxy
	proxies   []service.Proxy
	setCalls  []bool
	setErr    error
	deleteErr error
	deleted   bool
}

func (a *lifecycleAdmin) GetGroup(context.Context, int64) (*service.Group, error) {
	return &service.Group{ID: 3, Platform: "openai", Status: "active"}, nil
}

func (a *lifecycleAdmin) CreateAccount(_ context.Context, in *service.CreateAccountInput) (*service.Account, error) {
	if in.Schedulable == nil || *in.Schedulable {
		return nil, errors.New("untested account was enabled")
	}
	a.account = service.Account{ID: 42, Schedulable: false, Credentials: in.Credentials, ProxyID: in.ProxyID}
	return &a.account, nil
}

func (a *lifecycleAdmin) GetProxy(_ context.Context, id int64) (*service.Proxy, error) {
	if a.proxy != nil {
		return a.proxy, nil
	}
	return &service.Proxy{ID: id, Name: "US pool", Protocol: "socks5", Status: service.StatusActive}, nil
}

func (a *lifecycleAdmin) GetAllProxies(context.Context) ([]service.Proxy, error) {
	return a.proxies, nil
}

func (a *lifecycleAdmin) GetAccount(context.Context, int64) (*service.Account, error) {
	copy := a.account
	return &copy, nil
}

func (a *lifecycleAdmin) SetAccountSchedulable(_ context.Context, _ int64, enabled bool) (*service.Account, error) {
	a.setCalls = append(a.setCalls, enabled)
	a.account.Schedulable = enabled
	// Simulate a successful write followed by a failed readback.
	err := a.setErr
	a.setErr = nil
	return &a.account, err
}

func (a *lifecycleAdmin) DeleteAccount(context.Context, int64) error {
	if a.deleteErr != nil {
		return a.deleteErr
	}
	a.deleted = true
	return nil
}

type lifecycleTester struct {
	status string
	model  *string
}

func (s lifecycleTester) RunTestBackground(_ context.Context, _ int64, model string) (*service.ScheduledTestResult, error) {
	if s.model != nil {
		*s.model = model
	}
	return &service.ScheduledTestResult{Status: s.status, ErrorMessage: "401"}, nil
}

func expectLifecycleLock(mock sqlmock.Sqlmock, locked bool) {
	rows := sqlmock.NewRows([]string{"account_id"})
	if locked {
		rows.AddRow(42)
	}
	mock.ExpectQuery("UPDATE account_market_accounts").WithArgs(int64(42), int64(7), sqlmock.AnyArg()).WillReturnRows(rows)
}

func expectLifecycleUnlock(mock sqlmock.Sqlmock) {
	mock.ExpectExec("UPDATE account_market_accounts SET operation_token=NULL").WithArgs(int64(42), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectOwnership(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT EXISTS").WithArgs(int64(7), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
}

func TestSchedulingRestoresAfterWriteReadbackFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	admin := &lifecycleAdmin{account: service.Account{ID: 42}, setErr: errors.New("readback failed")}
	s := &Service{db: db, admin: admin}
	expectLifecycleLock(mock, true)
	expectOwnership(mock)
	expectLifecycleUnlock(mock)
	err = s.SetSchedulable(context.Background(), 7, 42, true)
	require.ErrorContains(t, err, "readback failed")
	require.Equal(t, []bool{true, false}, admin.setCalls)
	require.False(t, admin.account.Schedulable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConcurrentLifecycleOperationReturnsBusy(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	expectLifecycleLock(mock, false)
	err = (&Service{db: db}).Remove(context.Background(), 7, 42)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRemovalRestoresOwnershipBeforeEnabling(t *testing.T) {
	for _, restoreFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "restored", true: "restore_failed_stays_paused"}[restoreFails], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			admin := &lifecycleAdmin{account: service.Account{ID: 42, Schedulable: true}, deleteErr: errors.New("delete failed")}
			expectLifecycleLock(mock, true)
			expectOwnership(mock)
			mock.ExpectExec("UPDATE account_market_accounts").WithArgs(int64(42), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
			restore := mock.ExpectExec("UPDATE account_market_accounts").WithArgs(int64(42), int64(7), true)
			if restoreFails {
				restore.WillReturnError(errors.New("restore failed"))
			} else {
				restore.WillReturnResult(sqlmock.NewResult(0, 1))
			}
			expectLifecycleUnlock(mock)
			err = (&Service{db: db, admin: admin}).Remove(context.Background(), 7, 42)
			require.ErrorContains(t, err, "delete failed")
			require.Equal(t, !restoreFails, admin.account.Schedulable)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUploadLifecycle(t *testing.T) {
	for _, scenario := range []string{"success", "selected_proxy", "401", "duplicate_race", "activation_readback_failure", "cleanup_failure"} {
		t.Run(scenario, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			admin := &lifecycleAdmin{}
			var testedModel string
			tester := lifecycleTester{status: "success", model: &testedModel}
			if scenario == "401" || scenario == "cleanup_failure" {
				tester.status = "failed"
			}
			if scenario == "cleanup_failure" {
				admin.deleteErr = errors.New("cleanup failed")
			}
			mock.ExpectQuery("SELECT enabled, target_group_id").WillReturnRows(sqlmock.NewRows([]string{"enabled", "group", "commission", "minimum", "review"}).AddRow(true, 3, 10, 1, true))
			mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			mock.ExpectQuery("SELECT credentials, extra FROM accounts").WillReturnRows(sqlmock.NewRows([]string{"credentials", "extra"}))
			insert := mock.ExpectExec("INSERT INTO account_market_accounts")
			if scenario == "duplicate_race" {
				insert.WillReturnError(errors.New("duplicate key"))
			} else {
				insert.WillReturnResult(sqlmock.NewResult(1, 1))
				if tester.status == "success" {
					mock.ExpectExec("UPDATE account_market_accounts SET status='paused'").WillReturnResult(sqlmock.NewResult(0, 1))
					expectLifecycleLock(mock, true)
					expectOwnership(mock)
					if scenario == "activation_readback_failure" {
						admin.setErr = errors.New("readback failed")
					} else {
						mock.ExpectExec("UPDATE account_market_accounts").WillReturnResult(sqlmock.NewResult(0, 1))
					}
					expectLifecycleUnlock(mock)
				}
			}
			if scenario != "success" && scenario != "selected_proxy" && scenario != "cleanup_failure" {
				mock.ExpectExec("DELETE FROM account_market_accounts").WithArgs(int64(42)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			s := &Service{db: db, admin: admin, tester: tester}
			var proxyID *int64
			if scenario == "selected_proxy" {
				id := int64(9)
				proxyID = &id
			}
			result, err := s.Upload(context.Background(), 7, UploadInput{Platform: "openai", Type: "oauth", Credentials: map[string]any{"access_token": "token"}}, proxyID)
			if scenario == "success" || scenario == "selected_proxy" {
				require.NoError(t, err)
				require.True(t, result.Success)
				require.Equal(t, accountMarketOpenAIOAuthTestModel, testedModel)
				require.True(t, admin.account.Schedulable)
				if scenario == "selected_proxy" {
					require.Equal(t, int64(9), *admin.account.ProxyID)
				}
			} else {
				require.Error(t, err)
				require.Nil(t, result)
				require.False(t, admin.account.Schedulable)
				require.Equal(t, scenario != "cleanup_failure", admin.deleted)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSelectedProxyMustBeUsable(t *testing.T) {
	id := int64(9)
	for _, proxy := range []*service.Proxy{
		{ID: id, Status: service.StatusDisabled},
		{ID: id, Status: service.StatusActive, ExpiresAt: timePointer(time.Now().Add(-time.Minute))},
	} {
		s := &Service{admin: &lifecycleAdmin{proxy: proxy}}
		err := s.validateSelectedProxy(context.Background(), &id)
		require.Error(t, err)
	}
}

func TestSelectableProxiesExcludeExpiredAndSensitiveFields(t *testing.T) {
	admin := &lifecycleAdmin{proxies: []service.Proxy{
		{ID: 2, Name: "B", Protocol: "socks5", Host: "secret.example", Port: 1080, Username: "user", Password: "pass", Status: service.StatusActive},
		{ID: 1, Name: "expired", Protocol: "http", Status: service.StatusActive, ExpiresAt: timePointer(time.Now().Add(-time.Minute))},
	}}
	items, err := (&Service{admin: admin}).ListSelectableProxies(context.Background())
	require.NoError(t, err)
	require.Equal(t, []ProxyOption{{ID: 2, Name: "B", Protocol: "socks5"}}, items)
}

func timePointer(value time.Time) *time.Time { return &value }

func TestIdentityIgnoresUnrelatedIDTokenAndMetadata(t *testing.T) {
	first := map[string]any{"access_token": "actual-token", "email": "first@example.com", "id_token": "forged-one"}
	second := map[string]any{"access_token": "actual-token", "email": "second@example.com", "id_token": "forged-two"}
	require.Equal(t, credentialFingerprint("openai", "oauth", first), credentialFingerprint("openai", "oauth", second))
	second["access_token"] = "another-token"
	require.NotEqual(t, credentialFingerprint("openai", "oauth", first), credentialFingerprint("openai", "oauth", second))
}

func TestAPIKeyIdentityIgnoresOAuthFields(t *testing.T) {
	first := map[string]any{"api_key": "same-key", "access_token": "fake-one"}
	second := map[string]any{"api_key": "same-key", "access_token": "fake-two"}
	require.Equal(t, credentialFingerprint("openai", "apikey", first), credentialFingerprint("openai", "apikey", second))
}

func TestPendingAccountCannotBeEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	expectLifecycleLock(mock, true)
	mock.ExpectQuery("SELECT EXISTS.*status IN").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	expectLifecycleUnlock(mock)
	err = (&Service{db: db}).SetSchedulable(context.Background(), 7, 42, true)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithdrawalRowFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectExec("INSERT INTO account_market_earnings").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT enabled, target_group_id").WillReturnRows(sqlmock.NewRows([]string{"enabled", "group", "commission", "minimum", "review"}).AddRow(true, nil, 10, 1, true))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, gross_amount").WillReturnRows(sqlmock.NewRows([]string{"id", "gross"}).AddRow(1, 10).AddRow(2, 20).RowError(1, errors.New("stream failed")))
	mock.ExpectRollback()
	_, err = (&Service{db: db}).RequestWithdrawal(context.Background(), 7, WithdrawalRequest{PayoutMethod: "alipay", PayoutAccount: "qr"})
	require.ErrorContains(t, err, "stream failed")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithdrawalTerminalTransitions(t *testing.T) {
	for _, status := range []string{"rejected", "paid", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT status FROM account_market_withdrawals").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))
			if status == "paid" {
				mock.ExpectExec("UPDATE account_market_earnings SET status='withdrawn'").WithArgs(int64(9)).WillReturnResult(sqlmock.NewResult(0, 2))
			} else {
				mock.ExpectExec("(?s)UPDATE account_market_earnings.*status='available', withdrawal_id=NULL").WithArgs(int64(9)).WillReturnResult(sqlmock.NewResult(0, 2))
			}
			mock.ExpectExec("UPDATE account_market_withdrawals").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			s := &Service{db: db}
			if status == "cancelled" {
				err = s.CancelWithdrawal(context.Background(), 7, 9)
			} else {
				err = s.ReviewWithdrawal(context.Background(), 1, 9, ReviewRequest{Status: status})
			}
			require.NoError(t, err)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT status FROM account_market_withdrawals").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(status))
			mock.ExpectRollback()
			require.Error(t, s.ReviewWithdrawal(context.Background(), 1, 9, ReviewRequest{Status: "paid"}))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
