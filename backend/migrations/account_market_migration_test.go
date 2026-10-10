package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountMarketMigrationIsIsolatedAndIdempotent(t *testing.T) {
	content, err := FS.ReadFile("239_account_market.sql")
	require.NoError(t, err)

	sql := strings.ToLower(strings.Join(strings.Fields(string(content)), " "))
	for _, table := range []string{
		"account_market_config",
		"account_market_accounts",
		"account_market_earnings",
		"account_market_withdrawals",
	} {
		require.Contains(t, sql, "create table if not exists "+table)
	}
	require.Contains(t, sql, "on conflict (id) do nothing")
	require.Contains(t, sql, "usage_log_id bigint not null unique")
	require.Contains(t, sql, "account_market_earnings_legacy_pre239")
	require.Contains(t, sql, "account_market_withdrawals_legacy_pre239")
	require.NotContains(t, sql, "drop table")
	require.NotContains(t, sql, "truncate")
}

func TestLegacySequencePreparationPrecedesPublishedMigration(t *testing.T) {
	const name = "239_account_000_legacy_sequences.sql"
	require.Less(t, name, "239_account_market.sql")
	content, err := FS.ReadFile(name)
	require.NoError(t, err)
	require.Contains(t, string(content), "ALTER SEQUENCE IF EXISTS account_market_earnings_id_seq")
	require.Contains(t, string(content), "ALTER SEQUENCE IF EXISTS account_market_withdrawals_id_seq")
}

func TestAccountMarketIdentityScopeUpgradeMigration(t *testing.T) {
	content, err := FS.ReadFile("240_account_market_identity_scope.sql")
	require.NoError(t, err)

	sql := strings.ToLower(strings.Join(strings.Fields(string(content)), " "))
	require.Contains(t, sql, "add column if not exists platform")
	require.Contains(t, sql, "add column if not exists account_type")
	require.Contains(t, sql, "from accounts a")
	require.Contains(t, sql, "on account_market_accounts (platform, account_type, account_email_hash)")
}
