package handler

import (
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildHvoyPricingModelsUsesPoolAISettlementAndGroupMultiplier(t *testing.T) {
	input := 1e-6
	output := 2e-6
	cacheRead := 0.25e-6
	cacheWrite := 1.25e-6

	channels := []service.AvailableChannel{
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{{Name: "codex", Platform: "openai", RateMultiplier: 1.5}},
			SupportedModels: []service.SupportedModel{{
				Name:     "gpt-test",
				Platform: "openai",
				Pricing: &service.ChannelModelPricing{
					InputPrice:      &input,
					OutputPrice:     &output,
					CacheReadPrice:  &cacheRead,
					CacheWritePrice: &cacheWrite,
				},
			}},
		},
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{{Name: "codex", Platform: "openai", RateMultiplier: 1.5}},
			SupportedModels: []service.SupportedModel{{
				Name:     "gpt-test",
				Platform: "openai",
				Pricing:  &service.ChannelModelPricing{InputPrice: &input},
			}},
		},
		{
			Status: service.StatusDisabled,
			Groups: []service.AvailableGroupRef{{Name: "disabled", Platform: "openai"}},
			SupportedModels: []service.SupportedModel{{
				Name:     "disabled-model",
				Platform: "openai",
				Pricing:  &service.ChannelModelPricing{InputPrice: &input},
			}},
		},
	}

	models := buildHvoyPricingModels(channels, poolAISettlementCNYPerUSD)
	if len(models) != 1 {
		t.Fatalf("expected one deduplicated active model, got %d", len(models))
	}
	model := models[0]
	if model.PriceUnit != "per_1m_tokens" || model.GroupName != "codex" || !model.Enabled {
		t.Fatalf("unexpected model metadata: %+v", model)
	}
	assertHvoyPrice(t, model.InputPrice, 1.5)
	assertHvoyPrice(t, model.OutputPrice, 3.0)
	assertHvoyPrice(t, model.CacheInputPrice, 0.375)
	assertHvoyPrice(t, model.CacheCreatePrice, 1.875)
}

func TestHvoyModelUsesPerCallForNonTokenPricing(t *testing.T) {
	perRequest := 0.01
	models := buildHvoyPricingModels([]service.AvailableChannel{{
		Status: service.StatusActive,
		SupportedModels: []service.SupportedModel{{
			Name:     "image-model",
			Platform: "openai",
			Pricing: &service.ChannelModelPricing{
				BillingMode:     service.BillingModePerRequest,
				PerRequestPrice: &perRequest,
			},
		}},
	}}, poolAISettlementCNYPerUSD)

	if len(models) != 1 || models[0].PriceUnit != "per_call" {
		t.Fatalf("expected per_call model, got %+v", models)
	}
	assertHvoyPrice(t, models[0].UnitPrice, 0.01)
}

func TestBuildHvoyPricingModelsAppliesCurrentPeakMultiplier(t *testing.T) {
	input := 1e-6
	models := buildHvoyPricingModelsAt([]service.AvailableChannel{{
		Status: service.StatusActive,
		Groups: []service.AvailableGroupRef{{
			Name: "subscription", Platform: "openai", RateMultiplier: 2,
			SubscriptionType: "subscription", PeakRateEnabled: true,
			PeakStart: "09:00", PeakEnd: "18:00", PeakRateMultiplier: 1.25,
		}},
		SupportedModels: []service.SupportedModel{{
			Name: "gpt-test", Platform: "openai",
			Pricing: &service.ChannelModelPricing{InputPrice: &input},
		}},
	}}, poolAISettlementCNYPerUSD, atProviderPrice(12, 0))

	if len(models) != 1 {
		t.Fatalf("expected one model, got %d", len(models))
	}
	assertHvoyPrice(t, models[0].InputPrice, 2.5)
}

func atProviderPrice(hour, minute int) time.Time {
	return time.Date(2026, time.July, 26, hour, minute, 0, 0, time.Local)
}

func assertHvoyPrice(t *testing.T, value *float64, expected float64) {
	t.Helper()
	if value == nil || math.Abs(*value-expected) > 1e-9 {
		t.Fatalf("price = %v, want %v", value, expected)
	}
}
