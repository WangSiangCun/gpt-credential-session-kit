package migrations

import (
	"strings"
	"testing"
)

func TestRedeemGlobalUserOnceMigrationHasConcurrencyGuard(t *testing.T) {
	content, err := FS.ReadFile("191_redeem_code_global_user_once.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, required := range []string{
		"add column if not exists global_user_once",
		"create table if not exists redeem_global_user_claims",
		"user_id bigint primary key",
		"on redeem_codes (global_user_once)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
