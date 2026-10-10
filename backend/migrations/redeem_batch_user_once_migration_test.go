package migrations

import (
	"strings"
	"testing"
)

func TestRedeemBatchUserOnceMigrationHasConcurrencyGuard(t *testing.T) {
	content, err := FS.ReadFile("187_redeem_code_batch_user_once.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, required := range []string{
		"add column if not exists batch_id",
		"add column if not exists batch_user_once",
		"create table if not exists redeem_batch_user_claims",
		"primary key (batch_id, user_id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
