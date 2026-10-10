package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type intelligenceSettingsStub struct {
	service.SettingRepository
	value  string
	writes int
}

func (s *intelligenceSettingsStub) GetValue(context.Context, string) (string, error) {
	return s.value, nil
}
func (s *intelligenceSettingsStub) Set(_ context.Context, _ string, value string) error {
	s.value = value
	s.writes++
	return nil
}

type intelligenceGroupsStub struct{ service.GroupRepository }

func (*intelligenceGroupsStub) ListActive(context.Context) ([]service.Group, error) {
	return []service.Group{{ID: 47, Name: "Primary", Platform: service.PlatformOpenAI}, {ID: 48, Name: "Offline", Platform: service.PlatformOpenAI}}, nil
}

func (*intelligenceGroupsStub) GetByID(_ context.Context, id int64) (*service.Group, error) {
	return &service.Group{ID: id, Platform: service.PlatformOpenAI}, nil
}

type intelligenceCoolingAccountsStub struct{ service.AccountRepository }

func (*intelligenceCoolingAccountsStub) ListSchedulableByGroupID(context.Context, int64) ([]service.Account, error) {
	return []service.Account{{ID: 7, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}}, nil
}

func TestIntelligenceCooldownPreservesCauseAndRetryTime(t *testing.T) {
	repo := &intelligenceSettingsStub{value: `{"group_configs":{"47":{"model":"gpt-6-astra"}}}`}
	h := &IntelligenceTestHandler{settingService: service.NewSettingService(repo, nil), groupRepo: &intelligenceGroupsStub{}, accountRepo: &intelligenceCoolingAccountsStub{}}
	h.markAccountCooldown(7, 47, "gpt-6-astra", "upstream rejected 403", time.Minute)
	before, _ := h.accountCooldown(7, 47, "gpt-6-astra")
	result, err := h.testGroup(context.Background(), 47)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Error, "upstream rejected 403") || !strings.Contains(result.Error, before.until.Format("15:04:05")) {
		t.Fatalf("cooldown lost diagnostic details: %s", result.Error)
	}
	after, _ := h.accountCooldown(7, 47, "gpt-6-astra")
	if !before.until.Equal(after.until) {
		t.Fatal("skipping extended cooldown")
	}
}

func TestIntelligenceConcurrentRunDoesNotQueue(t *testing.T) {
	h := &IntelligenceTestHandler{}
	h.runMu.Lock()
	defer h.runMu.Unlock()
	if _, err := h.testGroup(context.Background(), 47); err == nil || !strings.Contains(err.Error(), "正在运行") {
		t.Fatalf("expected immediate busy error, got %v", err)
	}
}

func TestIntelligenceCancellationNeverMarksAccount(t *testing.T) {
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, errors.New("Stream read error: context canceled"), errors.New("upstream: context deadline exceeded")} {
		for _, proxy := range []bool{false, true} {
			_, duration, marked := classifyIntelligenceTestError(err, proxy)
			if duration != 0 || marked {
				t.Fatalf("cancellation was classified as account failure: %v", err)
			}
		}
	}
}

type intelligenceEmptyAccountsStub struct{ service.AccountRepository }

func (*intelligenceEmptyAccountsStub) ListSchedulableByGroupID(context.Context, int64) ([]service.Account, error) {
	return nil, nil
}

func TestIntelligenceUpdateValidationAndPreservation(t *testing.T) {
	const original = `{"prompt":"old","group_configs":{"47":{"model":"gpt-6-astra","reasoning_effort":"medium"},"48":{"model":"gpt-5.5"}},"results":{"47":{"group_id":47,"svg":"<svg/>","status":"success"}}}`
	tests := []struct {
		name    string
		prompt  string
		configs map[string]intelligenceGroupConfig
		status  int
	}{
		{name: "Chinese at character limit", prompt: strings.Repeat("中", 4000), status: 200},
		{name: "Unicode outside BMP at limit", prompt: strings.Repeat("\U0001f680", 4000), status: 200},
		{name: "Chinese over limit", prompt: strings.Repeat("中", 4001), status: 400},
		{name: "ASCII over limit", prompt: strings.Repeat("a", 4001), status: 400},
		{name: "unchanged offline group", prompt: "SVG", configs: map[string]intelligenceGroupConfig{"48": {Model: "gpt-5.5"}}, status: 200},
		{name: "clear one group preserves others", prompt: "SVG", configs: map[string]intelligenceGroupConfig{"47": {}}, status: 200},
		{name: "new unsupported model still rejected", prompt: "SVG", configs: map[string]intelligenceGroupConfig{"47": {Model: "unsupported"}}, status: 400},
		{name: "invalid reasoning still rejected", prompt: "SVG", configs: map[string]intelligenceGroupConfig{"47": {ReasoningEffort: "invalid"}}, status: 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &intelligenceSettingsStub{value: original}
			h := &IntelligenceTestHandler{settingService: service.NewSettingService(repo, nil), groupRepo: &intelligenceGroupsStub{}, accountRepo: &intelligenceEmptyAccountsStub{}}
			payload := map[string]any{"prompt": tt.prompt, "frequency_mins": 60, "timeout_seconds": 1800, "public_group_filter_enabled": true, "public_group_ids": []int64{}}
			if tt.configs != nil {
				payload["group_configs"] = tt.configs
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(string(body)))
			c.Request.Header.Set("Content-Type", "application/json")
			h.Update(c)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.status, w.Body.String())
			}
			if tt.status != 200 {
				if repo.writes != 0 || repo.value != original {
					t.Fatal("rejected config changed stored settings")
				}
				if strings.Contains(tt.name, "over limit") && !strings.Contains(w.Body.String(), "4000") {
					t.Fatal("missing useful validation error")
				}
				return
			}
			var stored intelligenceTestConfig
			if err := json.Unmarshal([]byte(repo.value), &stored); err != nil {
				t.Fatal(err)
			}
			if stored.Prompt != tt.prompt || stored.TimeoutSeconds != 1800 || !stored.PublicGroupFilterEnabled || len(stored.PublicGroupIDs) != 0 {
				t.Fatal("global settings did not round-trip")
			}
			if stored.GroupConfigs["48"].Model != "gpt-5.5" || stored.Results["47"].SVG != "<svg/>" {
				t.Fatal("unrelated config or results were lost")
			}
			want := intelligenceGroupConfig{Model: "gpt-6-astra", ReasoningEffort: "medium"}
			if selected, ok := tt.configs["47"]; ok {
				want = selected
			}
			if stored.GroupConfigs["47"] != want {
				t.Fatal("target group was not preserved/updated correctly")
			}
		})
	}
}

func TestFilterPublicIntelligenceGroups(t *testing.T) {
	groups := []service.Group{{ID: 1}, {ID: 2}, {ID: 3}}
	if got := filterPublicIntelligenceGroups(groups, &intelligenceTestConfig{}); len(got) != 3 {
		t.Fatalf("disabled filter should keep all groups, got %d", len(got))
	}
	got := filterPublicIntelligenceGroups(groups, &intelligenceTestConfig{PublicGroupFilterEnabled: true, PublicGroupIDs: []int64{3, 1}})
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("filtered groups = %#v, want ids 1 and 3", got)
	}
	if got := filterPublicIntelligenceGroups(groups, &intelligenceTestConfig{PublicGroupFilterEnabled: true}); len(got) != 0 {
		t.Fatalf("enabled empty filter should hide all groups, got %d", len(got))
	}
}

func TestIntelligenceEmptyReasoningOptionsSerializeAsArray(t *testing.T) {
	for _, models := range [][]string{nil, {}, {"gpt-image-2"}} {
		options := intelligenceReasoningEffortsForModels(service.PlatformOpenAI, models)
		data, err := json.Marshal(options)
		if err != nil || string(data) != "[]" {
			t.Fatalf("empty options must serialize as [], got %s (%v)", data, err)
		}
	}
}

func TestClassifyIntelligenceTestError(t *testing.T) {
	tests := []struct {
		name       string
		err        string
		hasProxy   bool
		wantText   string
		wantMarked bool
	}{
		{name: "proxy eof is not account failure", err: "socks connect tcp 204.1.94.159:443 -> newapi.host:443: EOF", hasProxy: true, wantText: "测试代理连接失败", wantMarked: false},
		{name: "unauthorized quarantines account", err: "upstream returned 401 Unauthorized", wantText: "账号鉴权失败", wantMarked: true},
		{name: "model unavailable quarantines account", err: "no available channel for model gpt-6-astra", wantText: "当前账号或上游没有可用的测试模型", wantMarked: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, cooldown, marked := classifyIntelligenceTestError(errors.New(tt.err), tt.hasProxy)
			if !strings.Contains(got, tt.wantText) || marked != tt.wantMarked {
				t.Fatalf("classifyIntelligenceTestError() = (%q, %v, %v), want text %q marked %v", got, cooldown, marked, tt.wantText, tt.wantMarked)
			}
			if marked && cooldown <= 0 {
				t.Fatal("marked account must have a positive cooldown")
			}
		})
	}
}

func TestIntelligenceTestAccountCooldownIsScoped(t *testing.T) {
	h := &IntelligenceTestHandler{}
	h.markAccountCooldown(7, 49, "gpt-5.6-terra", "bad credentials", time.Minute)
	if _, ok := h.accountCooldown(7, 49, "gpt-5.6-terra"); !ok {
		t.Fatal("expected matching account cooldown")
	}
	if _, ok := h.accountCooldown(7, 49, "gpt-5.6-sol"); ok {
		t.Fatal("model change should bypass the old cooldown")
	}
	if _, ok := h.accountCooldown(7, 53, "gpt-5.6-terra"); ok {
		t.Fatal("group change should bypass the old cooldown")
	}
}

func TestIntelligenceTestModelIDUsesGroupAndAccountConfiguration(t *testing.T) {
	group := &service.Group{DefaultMappedModel: "claude-sonnet-4", ModelAllowlist: service.GroupModelAllowlist{Models: []string{"claude-sonnet-4-6", "claude-3-5-haiku-20241022"}}}
	account := &service.Account{Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Credentials: map[string]any{"model_mapping": map[string]any{"claude-sonnet-4-6": "claude-sonnet-4-6", "claude-3-5-haiku-20241022": "claude-3-5-haiku-20241022"}}}
	if got := intelligenceTestModelIDForGroup(group, account); got != "claude-sonnet-4-6" {
		t.Fatalf("intelligenceTestModelIDForGroup() = %q, want group-supported model", got)
	}

	group = &service.Group{ModelAllowlist: service.GroupModelAllowlist{Models: []string{"gpt-5.6-sol"}}}
	account = &service.Account{Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	if got := intelligenceTestModelIDForGroup(group, account); got != "gpt-5.6-sol" {
		t.Fatalf("intelligenceTestModelIDForGroup() = %q, want group model for unmapped API key", got)
	}

	group = &service.Group{DefaultMappedModel: "claude-sonnet-4"}
	account = &service.Account{}
	if got := intelligenceTestModelID(group, account); got != "" {
		t.Fatalf("intelligenceTestModelID() = %q, want empty fallback for an unmapped account", got)
	}
}

func TestIntelligenceTestPromptRequiresRawSVG(t *testing.T) {
	got := intelligenceTestPrompt("draw a bicycle")
	if got == "draw a bicycle" || !strings.Contains(got, "<svg>") {
		t.Fatalf("intelligenceTestPrompt() did not add raw SVG instruction: %q", got)
	}
}

func TestNormalizeIntelligenceTestTimeout(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "missing uses default", in: 0, want: defaultIntelligenceTestTimeoutSeconds},
		{name: "too short uses minimum", in: 1, want: minIntelligenceTestTimeoutSeconds},
		{name: "too long uses maximum", in: 9999, want: maxIntelligenceTestTimeoutSeconds},
		{name: "valid value is preserved", in: 180, want: 180},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeIntelligenceTestTimeout(tt.in); got != tt.want {
				t.Fatalf("normalizeIntelligenceTestTimeout(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestIntelligenceReasoningEffortsMatchUpstreamModel(t *testing.T) {
	got := intelligenceReasoningEffortsForModel(service.PlatformOpenAI, "gpt-6-astra")
	if containsString(got, "minimal") {
		t.Fatalf("gpt-6-astra must not expose minimal: %v", got)
	}
	if !containsString(got, "medium") {
		t.Fatalf("gpt-6-astra should expose medium: %v", got)
	}
	got = intelligenceReasoningEffortsForModels(service.PlatformOpenAI, []string{"gpt-6-astra", "gpt-5.6-terra"})
	if containsString(got, "minimal") {
		t.Fatalf("shared options must exclude values rejected by one model: %v", got)
	}
}

func TestExtractIntelligenceSVGFromSSE(t *testing.T) {
	body := "data: {\"type\":\"content\",\"text\":\"\\u003csvg viewBox=\\\"0 0 10 10\\\"\\u003e\\u003ccircle cx=\\\"5\\\" /\\u003e\\u003c/svg\\u003e\"}\n" +
		"data: {\"type\":\"test_complete\",\"success\":true}\n"
	got := extractIntelligenceSVG(body)
	want := `<svg viewBox="0 0 10 10"><circle cx="5" /></svg>`
	if got != want {
		t.Fatalf("extractIntelligenceSVG() = %q, want %q", got, want)
	}
}

func TestExtractIntelligenceSVGIgnoresNonContentEvents(t *testing.T) {
	body := "data: {\"type\":\"status\",\"text\":\"<svg>bad</svg>\"}\n" +
		"data: {\"type\":\"content\",\"text\":\"prefix <svg><rect /></svg> suffix\"}\n"
	got := extractIntelligenceSVG(body)
	if got != "<svg><rect /></svg>" {
		t.Fatalf("extractIntelligenceSVG() = %q", got)
	}
}
