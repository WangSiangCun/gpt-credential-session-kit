package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// PoolAI settles balance as 1 CNY = 1 USD. Do not use a market FX rate here:
// the public price feed must match the amount charged by the billing service.
const poolAISettlementCNYPerUSD = 1.0

// ProviderPricingHandler exposes the public pricing document consumed by
// Hvoy's provider price directory. It reads the same active channel pricing
// used by the user-facing available-channels view.
type ProviderPricingHandler struct {
	channelService *service.ChannelService
	usdToCNY       float64
}

// NewProviderPricingHandler creates the public Hvoy pricing endpoint. Prices
// in channel configuration are USD per token; PoolAI settles 1 USD as 1 CNY.
func NewProviderPricingHandler(channelService *service.ChannelService) *ProviderPricingHandler {
	return &ProviderPricingHandler{channelService: channelService, usdToCNY: poolAISettlementCNYPerUSD}
}

type hvoyProviderPricingModel struct {
	ModelName          string   `json:"model_name"`
	GroupName          string   `json:"group_name"`
	PriceUnit          string   `json:"price_unit,omitempty"`
	UnitPrice          *float64 `json:"unit_price,omitempty"`
	InputPrice         *float64 `json:"input_price,omitempty"`
	OutputPrice        *float64 `json:"output_price,omitempty"`
	CacheInputPrice    *float64 `json:"cache_input_price,omitempty"`
	CacheCreatePrice   *float64 `json:"cache_create_price,omitempty"`
	CacheCreatePrice1H *float64 `json:"cache_create_price_1h,omitempty"`
	Enabled            bool     `json:"enabled"`
	Note               string   `json:"note,omitempty"`
}

func (h *ProviderPricingHandler) Get(c *gin.Context) {
	if !h.validSignature(c) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"schema_version": "1.1",
			"success":        false,
			"message":        "invalid or expired Hvoy signature",
		})
		return
	}
	if h.channelService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"schema_version": "1.1",
			"success":        false,
			"message":        "pricing service unavailable",
		})
		return
	}

	channels, err := h.channelService.ListAvailable(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"schema_version": "1.1",
			"success":        false,
			"message":        "failed to load prices",
		})
		return
	}

	models := buildHvoyPricingModelsAt(channels, h.usdToCNY, time.Now())
	c.JSON(http.StatusOK, gin.H{
		"schema_version": "1.1",
		"success":        true,
		"message":        "",
		"data": gin.H{
			"currency":    "CNY",
			"price_unit":  "per_1m_tokens",
			"site_name":   providerSiteName(),
			"site_domain": providerSiteDomain(),
			"updated_at":  time.Now().UTC().Format(time.RFC3339),
			"models":      models,
		},
	})
}

func buildHvoyPricingModels(channels []service.AvailableChannel, usdToCNY float64) []hvoyProviderPricingModel {
	return buildHvoyPricingModelsAt(channels, usdToCNY, time.Now())
}

func buildHvoyPricingModelsAt(channels []service.AvailableChannel, usdToCNY float64, now time.Time) []hvoyProviderPricingModel {
	if usdToCNY <= 0 {
		usdToCNY = poolAISettlementCNYPerUSD
	}

	// A model may be present in more than one active channel. Publish one
	// stable row per group/model pair, matching the first channel in the
	// deterministic ListAvailable order.
	seen := make(map[string]struct{})
	result := make([]hvoyProviderPricingModel, 0)
	for _, channel := range channels {
		if channel.Status != service.StatusActive {
			continue
		}
		for _, model := range channel.SupportedModels {
			groups := matchingProviderGroups(channel.Groups, model.Platform)
			for _, group := range groups {
				key := strings.ToLower(strings.TrimSpace(group.Name)) + "\x00" + strings.ToLower(strings.TrimSpace(model.Name))
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				result = append(result, hvoyModelFromSupportedModel(model, group, usdToCNY, now))
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].GroupName != result[j].GroupName {
			return result[i].GroupName < result[j].GroupName
		}
		return result[i].ModelName < result[j].ModelName
	})
	return result
}

func matchingProviderGroups(groups []service.AvailableGroupRef, platform string) []service.AvailableGroupRef {
	matched := make([]service.AvailableGroupRef, 0, len(groups))
	for _, group := range groups {
		if providerPlatformsMatch(group.Platform, platform) {
			matched = append(matched, group)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	name := strings.TrimSpace(platform)
	if name == "" {
		name = "default"
	}
	return []service.AvailableGroupRef{{Name: name, Platform: platform}}
}

func providerPlatformsMatch(left, right string) bool {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	if left == right {
		return true
	}
	aliases := map[string]string{
		"anthropic": "claude",
		"cc":        "claude",
		"openai":    "codex",
	}
	return aliases[left] != "" && aliases[left] == aliases[right]
}

func hvoyModelFromSupportedModel(model service.SupportedModel, group service.AvailableGroupRef, usdToCNY float64, now time.Time) hvoyProviderPricingModel {
	out := hvoyProviderPricingModel{
		ModelName: model.Name,
		GroupName: group.Name,
		Enabled:   true,
	}
	rateMultiplier := effectiveProviderGroupRate(group, now)
	if model.Pricing == nil {
		out.PriceUnit = "per_1m_tokens"
		return out
	}
	if model.Pricing.BillingMode == service.BillingModePerRequest || model.Pricing.BillingMode == service.BillingModeImage || model.Pricing.BillingMode == service.BillingModeVideo {
		out.PriceUnit = "per_call"
		out.UnitPrice = hvoyCNYPrice(model.Pricing.PerRequestPrice, usdToCNY, rateMultiplier)
		return out
	}
	out.PriceUnit = "per_1m_tokens"
	out.InputPrice = hvoyCNYPerMillion(model.Pricing.InputPrice, usdToCNY, rateMultiplier)
	out.OutputPrice = hvoyCNYPerMillion(model.Pricing.OutputPrice, usdToCNY, rateMultiplier)
	out.CacheInputPrice = hvoyCNYPerMillion(model.Pricing.CacheReadPrice, usdToCNY, rateMultiplier)
	out.CacheCreatePrice = hvoyCNYPerMillion(model.Pricing.CacheWritePrice, usdToCNY, rateMultiplier)
	return out
}

func effectiveProviderGroupRate(group service.AvailableGroupRef, now time.Time) float64 {
	base := group.RateMultiplier
	if base <= 0 || math.IsNaN(base) || math.IsInf(base, 0) {
		base = 1
	}
	peak := (&service.Group{
		SubscriptionType:   group.SubscriptionType,
		PeakRateEnabled:    group.PeakRateEnabled,
		PeakStart:          group.PeakStart,
		PeakEnd:            group.PeakEnd,
		PeakRateMultiplier: group.PeakRateMultiplier,
	}).PeakMultiplierAt(now)
	return base * peak
}

func hvoyCNYPerMillion(value *float64, usdToCNY, rateMultiplier float64) *float64 {
	if value == nil {
		return nil
	}
	converted := *value * usdToCNY * rateMultiplier * 1_000_000
	return roundHvoyPrice(converted)
}

func hvoyCNYPrice(value *float64, usdToCNY, rateMultiplier float64) *float64 {
	if value == nil {
		return nil
	}
	return roundHvoyPrice(*value * usdToCNY * rateMultiplier)
}

func roundHvoyPrice(value float64) *float64 {
	rounded := float64(int64(value*1_000_000+0.5)) / 1_000_000
	return &rounded
}

func (h *ProviderPricingHandler) validSignature(c *gin.Context) bool {
	secret := strings.TrimSpace(os.Getenv("HVOY_PRICING_AUTH_SECRET"))
	if secret == "" {
		return true
	}
	timestamp := strings.TrimSpace(c.GetHeader("X-Hvoy-Ts"))
	signature := strings.TrimSpace(c.GetHeader("X-Hvoy-Sign"))
	unixTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || signature == "" || absHvoyInt64(time.Now().Unix()-unixTimestamp) > 60 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected))
}

func absHvoyInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func providerSiteDomain() string {
	if domain := strings.TrimSpace(os.Getenv("SITE_DOMAIN")); domain != "" {
		return domain
	}
	if frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL")); frontendURL != "" {
		if parsed, err := url.Parse(frontendURL); err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}
	return "poolai.online"
}

func providerSiteName() string {
	if name := strings.TrimSpace(os.Getenv("SITE_NAME")); name != "" {
		return name
	}
	return "PoolAI"
}
