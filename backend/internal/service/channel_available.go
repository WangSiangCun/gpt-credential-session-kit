package service

import (
	"context"
	"fmt"
	"maps"
	"math"
	"sort"
	"strings"
)

// AvailableGroupRef 渠道视图中关联分组的简要信息。
//
// 用户侧「可用渠道」页面据此展示：专属分组 vs 公开分组（IsExclusive）、
// 订阅 vs 标准（SubscriptionType）、默认倍率（RateMultiplier）与高峰倍率规则。
// 用户专属倍率不在这里暴露，前端自己通过 /groups/rates 拉取，和 API 密钥页面保持一致。
type AvailableGroupRef struct {
	ID                 int64
	Name               string
	I18n               GroupI18n
	Platform           string
	SubscriptionType   string
	RateMultiplier     float64
	PeakRateEnabled    bool
	PeakStart          string
	PeakEnd            string
	PeakRateMultiplier float64
	IsExclusive        bool
}

// AvailableChannel 可用渠道视图：用于「可用渠道」页面展示渠道基础信息 +
// 关联的分组 + 推导出的支持模型列表（无通配符）。
type AvailableChannel struct {
	ID                 int64
	Name               string
	Description        string
	Status             string
	BillingModelSource string
	RestrictModels     bool
	Groups             []AvailableGroupRef
	SupportedModels    []SupportedModel
}

// ChannelQuotaSummary is the aggregate remaining quota for accounts attached
// to the groups visible to the current user. Account details never leave the
// service boundary.
type ChannelQuotaSummary struct {
	RemainingPercent float64
	AccountCount     int
}

// PublicChannelAccount is the strict, read-only account projection allowed to
// leave the service boundary. It intentionally contains only the compact
// account summary needed by the user-facing channel view.
type PublicChannelAccount struct {
	Name                  string
	Email                 string
	PlanType              string
	OfficialDirect        bool
	Status                string
	Available             bool
	QuotaRemainingPercent *float64
}

// GetPublicAccounts returns sanitized summaries for unique accounts attached
// to the supplied groups. The caller is responsible for enforcing the runtime
// visibility switch.
func (s *ChannelService) GetPublicAccounts(ctx context.Context, groupIDs []int64) ([]PublicChannelAccount, error) {
	if len(groupIDs) == 0 || s.groupRepo == nil || s.quotaAccountRepo == nil {
		return []PublicChannelAccount{}, nil
	}

	accountIDs, err := s.groupRepo.GetAccountIDsByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("list public channel account ids: %w", err)
	}
	accountIDs = uniquePositiveInt64s(accountIDs)
	if len(accountIDs) == 0 {
		return []PublicChannelAccount{}, nil
	}

	accounts, err := s.quotaAccountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("load public channel accounts: %w", err)
	}

	out := make([]PublicChannelAccount, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || strings.TrimSpace(account.Name) == "" {
			continue
		}
		out = append(out, buildPublicChannelAccount(account))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func buildPublicChannelAccount(account *Account) PublicChannelAccount {
	out := PublicChannelAccount{
		Name:           strings.TrimSpace(account.Name),
		Email:          firstAccountString(account.Extra, account.Credentials, "email_address", "email"),
		PlanType:       firstNonEmpty(mapString(account.Credentials, "plan_type"), mapString(account.Extra, "subscription_tier")),
		OfficialDirect: account.IsPublicOfficialDirect(),
		Status:         account.Status,
		Available:      account.IsSchedulable(),
	}
	if remaining, ok := accountRemainingQuotaPercent(account); ok {
		out.QuotaRemainingPercent = &remaining
	}
	return out
}

func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstAccountString(primary, secondary map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := mapString(primary, key); value != "" {
			return value
		}
		if value := mapString(secondary, key); value != "" {
			return value
		}
	}
	return ""
}

// GetQuotaSummary averages the remaining percentage of unique, enabled
// accounts attached to the supplied groups. Accounts without a trustworthy
// quota snapshot are excluded instead of being treated as fully available.
func (s *ChannelService) GetQuotaSummary(ctx context.Context, groupIDs []int64) (*ChannelQuotaSummary, error) {
	if len(groupIDs) == 0 || s.groupRepo == nil || s.quotaAccountRepo == nil {
		return nil, nil
	}

	accountIDs, err := s.groupRepo.GetAccountIDsByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("list channel quota account ids: %w", err)
	}
	accountIDs = uniquePositiveInt64s(accountIDs)
	if len(accountIDs) == 0 {
		return nil, nil
	}

	accounts, err := s.quotaAccountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("load channel quota accounts: %w", err)
	}

	var total float64
	count := 0
	for _, account := range accounts {
		if account == nil || !account.IsActive() || !account.Schedulable {
			continue
		}
		remaining, ok := accountRemainingQuotaPercent(account)
		if !ok {
			continue
		}
		total += remaining
		count++
	}
	if count == 0 {
		return nil, nil
	}

	return &ChannelQuotaSummary{
		RemainingPercent: math.Round(clampPercent(total/float64(count))*10) / 10,
		AccountCount:     count,
	}, nil
}

func uniquePositiveInt64s(values []int64) []int64 {
	unique := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func accountRemainingQuotaPercent(account *Account) (float64, bool) {
	if account == nil {
		return 0, false
	}

	remaining := make([]float64, 0, 6)
	appendLimitRemaining := func(limit, used float64, expired bool) {
		if limit <= 0 {
			return
		}
		if expired {
			used = 0
		}
		remaining = append(remaining, 100-(used/limit)*100)
	}

	appendLimitRemaining(account.GetQuotaLimit(), account.GetQuotaUsed(), false)
	appendLimitRemaining(account.GetQuotaDailyLimit(), account.GetQuotaDailyUsed(), account.IsDailyQuotaPeriodExpired())
	appendLimitRemaining(account.GetQuotaWeeklyLimit(), account.GetQuotaWeeklyUsed(), account.IsWeeklyQuotaPeriodExpired())

	if used, ok := maxAccountExtraNumber(account.Extra,
		"codex_primary_used_percent",
		"codex_secondary_used_percent",
		"codex_5h_used_percent",
		"codex_7d_used_percent",
	); ok {
		remaining = append(remaining, 100-used)
	}

	if utilization, ok := maxAccountExtraNumber(account.Extra,
		"session_window_utilization",
		"passive_usage_7d_utilization",
		"passive_usage_7d_oi_utilization",
	); ok {
		// Anthropic utilization snapshots are persisted as fractions.
		if utilization <= 1 {
			utilization *= 100
		}
		remaining = append(remaining, 100-utilization)
	}

	if billing, err := grokBillingSnapshotFromExtra(account.Extra); err == nil && billing != nil {
		usedValues := make([]float64, 0, 2)
		if billing.UsagePercent != nil {
			usedValues = append(usedValues, *billing.UsagePercent)
		}
		if billing.UsedPercent != nil {
			usedValues = append(usedValues, *billing.UsedPercent)
		}
		if len(usedValues) > 0 {
			remaining = append(remaining, 100-maxFloat64(usedValues))
		}
	}

	if len(remaining) == 0 {
		return 0, false
	}
	return clampPercent(minFloat64(remaining)), true
}

func maxAccountExtraNumber(extra map[string]any, keys ...string) (float64, bool) {
	values := make([]float64, 0, len(keys))
	for _, key := range keys {
		if value, ok := resolveAccountExtraNumber(extra, key); ok && !math.IsNaN(value) && !math.IsInf(value, 0) {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return 0, false
	}
	return maxFloat64(values), true
}

func minFloat64(values []float64) float64 {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func maxFloat64(values []float64) float64 {
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func clampPercent(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

// ListAvailable 返回所有渠道的可用视图：每个渠道附带关联分组信息与支持模型列表。
//
// 支持模型通过 (*Channel).SupportedModels() 计算（mapping ∪ pricing 并联）。
// 对于渠道未配置定价的模型，进一步用 PricingService 的全局 LiteLLM 数据合成
// 一份展示用定价，让用户看到默认价格而非"未配置"。
//
// 关联分组信息通过 groupRepo.ListActive 查询后按 ID 映射；渠道 GroupIDs 中未在活跃列表中
// 的分组（已停用或删除）会被忽略。
//
// 前置条件：s.groupRepo 必须非 nil（由 wire DI 保证）。直接 nil-deref 用于 fail-fast，
// 避免静默掩盖注入缺失。
func (s *ChannelService) ListAvailable(ctx context.Context) ([]AvailableChannel, error) {
	channels, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}
	groupByID := make(map[int64]AvailableGroupRef, len(groups))
	for i := range groups {
		g := groups[i]
		groupByID[g.ID] = AvailableGroupRef{
			ID:                 g.ID,
			Name:               g.Name,
			I18n:               g.I18n,
			Platform:           g.Platform,
			SubscriptionType:   g.SubscriptionType,
			RateMultiplier:     g.RateMultiplier,
			PeakRateEnabled:    g.PeakRateEnabled,
			PeakStart:          g.PeakStart,
			PeakEnd:            g.PeakEnd,
			PeakRateMultiplier: g.PeakRateMultiplier,
			IsExclusive:        g.IsExclusive,
		}
	}

	out := make([]AvailableChannel, 0, len(channels))
	for i := range channels {
		ch := &channels[i]
		groups := make([]AvailableGroupRef, 0, len(ch.GroupIDs))
		for _, gid := range ch.GroupIDs {
			if ref, ok := groupByID[gid]; ok {
				groups = append(groups, ref)
			}
		}
		sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })

		ch.normalizeBillingModelSource()

		supported := ch.SupportedModels()
		fillGlobalPricingFallback(s.pricingService, supported)

		out = append(out, AvailableChannel{
			ID:                 ch.ID,
			Name:               ch.Name,
			Description:        ch.Description,
			Status:             ch.Status,
			BillingModelSource: ch.BillingModelSource,
			RestrictModels:     ch.RestrictModels,
			Groups:             groups,
			SupportedModels:    supported,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// fillGlobalPricingFallback 对未命中渠道定价的支持模型，从全局 LiteLLM 数据合成一份
// 展示用定价。仅用于「可用渠道」展示，不影响真实计费链路。
//
// 触发条件：
//  1. Pricing == nil（渠道完全没声明该模型的定价条目）
//  2. Pricing 非 nil 但所有价格字段为空（admin UI 建了条目但没填价格）
//
// 当 pricingService 为 nil（测试场景），跳过价格回落。
// 可用渠道与模型广场共用。
func fillGlobalPricingFallback(pricingService *PricingService, models []SupportedModel) {
	for i := range models {
		if pricingService != nil && pricingNeedsFallback(models[i].Pricing) {
			if lp := pricingService.GetModelPricing(models[i].Name); lp != nil {
				models[i].Pricing = synthesizePricingFromLiteLLM(lp, models[i].Pricing)
			}
		}
	}
}

// pricingNeedsFallback 判定一个 ChannelModelPricing 是否需要走全局回落。
// 价格全部缺失（无 flat 字段且无任何带价 interval）即视为未配置。
func pricingNeedsFallback(p *ChannelModelPricing) bool {
	if p == nil {
		return true
	}
	if p.InputPrice != nil || p.OutputPrice != nil ||
		p.CacheWritePrice != nil || p.CacheWrite1hPrice != nil || p.CacheReadPrice != nil ||
		p.ImageOutputPrice != nil || p.PerRequestPrice != nil {
		return false
	}
	for _, iv := range p.Intervals {
		if iv.InputPrice != nil || iv.OutputPrice != nil ||
			iv.CacheWritePrice != nil || iv.CacheWrite1hPrice != nil || iv.CacheReadPrice != nil ||
			iv.PerRequestPrice != nil {
			return false
		}
	}
	return true
}

// synthesizePricingFromLiteLLM 把 LiteLLM 的定价数据转成 ChannelModelPricing 形态，
// 仅用于展示。
//
// 计费模式优先级：
//  1. 渠道已选 BillingMode（admin 在 UI 里选了 image / per_request 但没填价的场景，
//     按选定模式合成对应字段）
//  2. LiteLLM mode="image_generation" → image
//  3. 默认 token
//
// LiteLLM 中字段 0 视为未配置，不带入展示。
func synthesizePricingFromLiteLLM(lp *LiteLLMModelPricing, existing *ChannelModelPricing) *ChannelModelPricing {
	if lp == nil {
		return existing
	}

	mode := BillingModeToken
	switch {
	case existing != nil && existing.BillingMode != "":
		mode = existing.BillingMode
	case lp.Mode == "image_generation":
		mode = BillingModeImage
	}

	if mode == BillingModeImage || mode == BillingModePerRequest {
		return &ChannelModelPricing{
			BillingMode:                mode,
			PerRequestPrice:            nonZeroPtr(lp.OutputCostPerImage),
			ImageOutputPrice:           nonZeroPtr(lp.OutputCostPerImageToken),
			InputPrice:                 nonZeroPtr(lp.InputCostPerToken),
			OutputPrice:                nonZeroPtr(lp.OutputCostPerToken),
			ReasoningEffortMultipliers: reasoningEffortMultipliersFromPricing(existing),
		}
	}
	return &ChannelModelPricing{
		BillingMode:                mode,
		InputPrice:                 nonZeroPtr(lp.InputCostPerToken),
		OutputPrice:                nonZeroPtr(lp.OutputCostPerToken),
		CacheWritePrice:            nonZeroPtr(lp.CacheCreationInputTokenCost),
		CacheWrite1hPrice:          nonZeroPtr(lp.CacheCreationInputTokenCostAbove1hr),
		CacheReadPrice:             nonZeroPtr(lp.CacheReadInputTokenCost),
		ImageOutputPrice:           nonZeroPtr(lp.OutputCostPerImageToken),
		ReasoningEffortMultipliers: reasoningEffortMultipliersFromPricing(existing),
	}
}

func reasoningEffortMultipliersFromPricing(pricing *ChannelModelPricing) map[string]float64 {
	if pricing == nil {
		return nil
	}
	return maps.Clone(pricing.ReasoningEffortMultipliers)
}

func nonZeroPtr(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}
