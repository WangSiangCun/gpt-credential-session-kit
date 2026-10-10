//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenAI403ProtectionSettingsDefaults(t *testing.T) {
	svc := NewSettingService(newMockSettingRepo(), &config.Config{})

	settings, err := svc.GetOpenAI403ProtectionSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, DefaultOpenAI403ProtectionSettings(), settings)
}

func TestOpenAI403ProtectionSettingsRoundTrip(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})
	want := &OpenAI403ProtectionSettings{
		Enabled: false, ThresholdCount: 5, ThresholdWindowMinutes: 30, CooldownMinutes: 8,
	}

	require.NoError(t, svc.SetOpenAI403ProtectionSettings(context.Background(), want))
	got, err := svc.GetOpenAI403ProtectionSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, want, got)
	var stored OpenAI403ProtectionSettings
	require.NoError(t, json.Unmarshal([]byte(repo.data[SettingKeyOpenAI403ProtectionSettings]), &stored))
	require.Equal(t, *want, stored)
}

func TestOpenAI403ProtectionSettingsRejectInvalidRanges(t *testing.T) {
	svc := NewSettingService(newMockSettingRepo(), &config.Config{})
	tests := []OpenAI403ProtectionSettings{
		{Enabled: true, ThresholdCount: 0, ThresholdWindowMinutes: 30, CooldownMinutes: 10},
		{Enabled: true, ThresholdCount: 3, ThresholdWindowMinutes: 0, CooldownMinutes: 10},
		{Enabled: true, ThresholdCount: 3, ThresholdWindowMinutes: 30, CooldownMinutes: 0},
	}
	for _, settings := range tests {
		require.Error(t, svc.SetOpenAI403ProtectionSettings(context.Background(), &settings))
	}
}
