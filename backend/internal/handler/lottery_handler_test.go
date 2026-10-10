package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestMaskLotteryAccount(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		want     string
	}{
		{name: "username", username: "测试用户", email: "user@example.com", want: "测***户"},
		{name: "short username", username: "阿明", want: "阿*"},
		{name: "email", email: "person@example.com", want: "p***@example.com"},
		{name: "fallback", want: "***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskLotteryAccount(tt.username, tt.email); got != tt.want {
				t.Fatalf("maskLotteryAccount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublicLotteryCampaignOmitsAdminFields(t *testing.T) {
	payload, err := json.Marshal(toPublicLotteryCampaign(service.LotteryCampaign{
		ID:          12,
		Token:       "public-token",
		Name:        "活动",
		CreatedBy:   88,
		DrawCount:   99,
		WinnerCount: 3,
		StartsAt:    time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, forbidden := range []string{"token", "created_by", "draw_count", "weight", "user_id", "user_email"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("public payload leaked %q: %s", forbidden, text)
		}
	}
}
