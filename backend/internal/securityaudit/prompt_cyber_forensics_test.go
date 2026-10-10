package securityaudit

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractCyberPolicySnapshotPrioritizesLatestUserAndRedactsPreview(t *testing.T) {
	body := []byte(`{
		"instructions":"system Bearer secret-token-value",
		"input":[
			{"role":"user","content":[{"type":"input_text","text":"older user"}]},
			{"role":"assistant","content":[{"type":"output_text","text":"older answer"}]},
			{"role":"user","content":[{"type":"input_text","text":"latest user api_key=abcdefghijk user@example.com"}]}
		]
	}`)
	snapshot, err := ExtractCyberPolicySnapshot(Request{
		RequestID: "req-cyber", Protocol: "openai_responses", Body: body,
	})
	require.NoError(t, err)
	require.Equal(t, 4, snapshot.MessageCount)
	require.Contains(t, snapshot.FullPrompt, "latest user")
	require.Less(t, strings.Index(snapshot.FullPrompt, "latest user"), strings.Index(snapshot.FullPrompt, "older user"))
	require.Contains(t, snapshot.FullPrompt, "secret-token-value")
	require.Contains(t, snapshot.FullPrompt, "abcdefghijk")
	require.Contains(t, snapshot.FullPrompt, "user@example.com")
	require.NotContains(t, snapshot.RedactedPreview, "secret-token-value")
	require.NotContains(t, snapshot.RedactedPreview, "user@example.com")
	require.Len(t, snapshot.PromptHash, 64)
}

func TestExtractCyberPolicySnapshotDoesNotUseNormal64KPromptCap(t *testing.T) {
	text := strings.Repeat("长上下文", 30000)
	body := []byte(`{"input":` + strconv.Quote(text) + `}`)
	snapshot, err := ExtractCyberPolicySnapshot(Request{Protocol: "openai_responses", Body: body})
	require.NoError(t, err)
	require.Contains(t, snapshot.FullPrompt, text)
	require.NotContains(t, snapshot.FullPrompt, "…")
	require.Greater(t, len([]rune(snapshot.FullPrompt)), DefaultFullPromptMaxRunes)
}

type recordingPromptEngine struct {
	request  Request
	metadata CyberPolicyMetadata
	calls    int
}

func (*recordingPromptEngine) EffectiveMode() Mode                    { return ModeOff }
func (*recordingPromptEngine) Enqueue(context.Context, Request) error { return nil }
func (*recordingPromptEngine) Evaluate(context.Context, Request) (*PromptDecision, error) {
	return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true}, nil
}
func (e *recordingPromptEngine) RecordCyberPolicy(_ context.Context, req Request, metadata CyberPolicyMetadata) error {
	e.calls++
	e.request = req
	e.metadata = metadata
	return nil
}

func TestCoordinatorRecordsCyberPolicyWhenPromptAuditModeIsOff(t *testing.T) {
	engine := &recordingPromptEngine{}
	coordinator := NewCoordinator(nil, engine)
	body := []byte(`{"input":"hello"}`)
	err := coordinator.RecordCyberPolicy(context.Background(), Request{RequestID: "req-1", Body: body}, CyberPolicyMetadata{AccountID: 12})
	require.NoError(t, err)
	require.Equal(t, 1, engine.calls)
	require.Equal(t, "req-1", engine.request.RequestID)
	require.Equal(t, int64(12), engine.metadata.AccountID)
	body[0] = 'x'
	require.Equal(t, byte('{'), engine.request.Body[0], "the async forensic request must own its body copy")
}
