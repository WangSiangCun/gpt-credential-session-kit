//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type runtimeBlockRecorder struct {
	accounts   []*Account
	until      []time.Time
	reasons    []string
	clearedIDs []int64
}

func (r *runtimeBlockRecorder) BlockAccountScheduling(account *Account, until time.Time, reason string) {
	r.accounts = append(r.accounts, account)
	r.until = append(r.until, until)
	r.reasons = append(r.reasons, reason)
}

func (r *runtimeBlockRecorder) ClearAccountSchedulingBlock(accountID int64) {
	r.clearedIDs = append(r.clearedIDs, accountID)
}

func TestRateLimitService_HandleUpstreamError_OpenAI403BelowThresholdOnlyFailsOverRequest(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1}}
	blocker := &runtimeBlockRecorder{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	service.SetAccountRuntimeBlocker(blocker)
	account := &Account{
		ID:       301,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"temporary edge rejection"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 0, repo.tempCalls)
	require.Empty(t, blocker.accounts)
}

func TestRateLimitService_HandleUpstreamError_OpenAI403ThresholdTemporarilyPauses(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{3}}
	blocker := &runtimeBlockRecorder{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	service.SetAccountRuntimeBlocker(blocker)
	account := &Account{
		ID:       302,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"workspace forbidden by policy"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.tempCalls)
	require.Contains(t, repo.lastTempReason, "workspace forbidden by policy")
	require.Contains(t, repo.lastTempReason, "(3/3)")
	require.Len(t, blocker.accounts, 1)
	require.True(t, blocker.until[0].After(time.Now()))
	require.Equal(t, []int64{account.ID}, counter.resetCalls)
}

func TestRateLimitService_HandleUpstreamError_OpenAI403UsesAdminSettings(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{2}}
	blocker := &runtimeBlockRecorder{}
	settingRepo := newMockSettingRepo()
	data, _ := json.Marshal(OpenAI403ProtectionSettings{
		Enabled: true, ThresholdCount: 2, ThresholdWindowMinutes: 15, CooldownMinutes: 7,
	})
	settingRepo.data[SettingKeyOpenAI403ProtectionSettings] = string(data)
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	service.SetSettingService(NewSettingService(settingRepo, &config.Config{}))
	service.SetAccountRuntimeBlocker(blocker)
	account := &Account{ID: 303, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	before := time.Now()
	service.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(`forbidden`))

	require.Equal(t, 1, repo.tempCalls)
	require.Len(t, blocker.until, 1)
	require.True(t, !blocker.until[0].Before(before.Add(7*time.Minute)))
}

func TestRateLimitService_HandleUpstreamError_OpenAI403ProtectionDisabled(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{20}}
	settingRepo := newMockSettingRepo()
	data, _ := json.Marshal(OpenAI403ProtectionSettings{
		Enabled: false, ThresholdCount: 3, ThresholdWindowMinutes: 180, CooldownMinutes: 10,
	})
	settingRepo.data[SettingKeyOpenAI403ProtectionSettings] = string(data)
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	service.SetSettingService(NewSettingService(settingRepo, &config.Config{}))
	account := &Account{ID: 304, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	shouldDisable := service.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(`forbidden`))

	require.True(t, shouldDisable)
	require.Zero(t, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
	require.Len(t, counter.counts, 1)
}
