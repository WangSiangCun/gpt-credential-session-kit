package securityaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

const cyberPolicyScannerBackend = "upstream_cyber_policy"

// CyberPolicyMetadata is the small set of upstream fields needed to correlate
// a rejection with the selected account. Raw upstream responses stay in the
// existing cyber-policy log.
type CyberPolicyMetadata struct {
	AccountID       int64
	AccountName     string
	UpstreamBaseURL string
	UpstreamStatus  int
	UpstreamMessage string
	ResponseID      string
	SequenceNumber  int64
}

// ExtractCyberPolicySnapshot keeps all text with the latest user turn first.
// Shared protocol extractors exclude binary media and non-text tool payloads.
func ExtractCyberPolicySnapshot(req Request) (PromptSnapshot, error) {
	var document any
	if err := json.Unmarshal(req.Body, &document); err != nil {
		return PromptSnapshot{}, errors.New("prompt audit request JSON is invalid")
	}
	segments := normalizeSegmentsLatestUserFirst(extractProtocolSegments(req.Protocol, document))
	if len(segments) == 0 {
		return PromptSnapshot{}, ErrNoPromptText
	}
	rawText := strings.ReplaceAll(strings.Join(segments, "\n\n"), "\x00", "")
	promptDigest := sha256.Sum256([]byte(rawText))
	stage := strings.TrimSpace(req.Stage)
	if stage == "" {
		stage = "http"
	}
	snapshot := PromptSnapshot{
		RequestID: req.RequestID, UserID: req.UserID, UsernameSnapshot: req.Username,
		UserEmailSnapshot: req.UserEmail, APIKeyID: req.APIKeyID, APIKeyNameSnapshot: req.APIKeyName,
		GroupID: cloneInt64Ptr(req.GroupID), GroupName: req.GroupName, Provider: req.Provider,
		Endpoint: req.Endpoint, Protocol: req.Protocol, Model: req.Model,
		PromptHash: hex.EncodeToString(promptDigest[:]), RedactedPreview: BuildPromptPreview(rawText, DefaultPromptPreviewMaxRunes),
		FullPrompt: strings.TrimSpace(rawText), PromptLength: len([]rune(rawText)), MessageCount: len(segments), Stage: stage,
	}
	return snapshot, nil
}

func (s *PromptService) RecordCyberPolicy(ctx context.Context, req Request, metadata CyberPolicyMetadata) error {
	if s == nil || s.repo == nil {
		return errors.New("prompt audit database unavailable")
	}
	snapshot, err := ExtractCyberPolicySnapshot(req)
	if err != nil {
		return err
	}
	evidence := map[string]string{}
	evidence["account_id"] = strconv.FormatInt(metadata.AccountID, 10)
	evidence["account_name"] = metadata.AccountName
	evidence["upstream_base_url"] = metadata.UpstreamBaseURL
	evidence["upstream_status"] = strconv.Itoa(metadata.UpstreamStatus)
	evidence["upstream_message"] = metadata.UpstreamMessage
	evidence["response_id"] = metadata.ResponseID
	evidence["sequence_number"] = strconv.FormatInt(metadata.SequenceNumber, 10)
	result := &NormalizedResult{
		Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock,
		Categories: []string{"cyber_policy"}, MatchedScanners: []string{"cyber_policy"},
		ScannerScores: map[string]float64{"cyber_policy": 1}, ScannerEvidence: evidence,
		ScannerBackend: cyberPolicyScannerBackend, ScannerVersion: "1", PolicyID: "upstream", PolicyVersion: 1,
		ChunkTotal: snapshot.MessageCount,
	}
	// Cyber evidence is a blocking audit event, so use the minimum valid
	// configuration version required by prompt_audit_jobs.
	if _, err := s.repo.RecordBlocking(ctx, snapshot, 1, result, false); err != nil {
		return err
	}
	return nil
}
