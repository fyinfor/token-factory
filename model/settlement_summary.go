package model

import (
	"sort"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const settlementSummaryMaxGroups = 100

// SettlementSummaryAmounts 结算汇总金额与用量。
type SettlementSummaryAmounts struct {
	RecordCount      int64  `json:"record_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	CacheTokens      int64  `json:"cache_tokens"`
	OfficialTotal    string `json:"official_total"`
	CostPrice        string `json:"cost_price"`
	OperatingPrice   string `json:"operating_price"`
	SalesPrice       string `json:"sales_price"`
	UserPaid         string `json:"user_paid"`
}

// SettlementSummaryGroup 按视角分组的汇总行。
type SettlementSummaryGroup struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	SettlementSummaryAmounts
}

// SettlementSummaryResult 结算汇总响应。
type SettlementSummaryResult struct {
	Scope           string                   `json:"scope"`
	Currency        string                   `json:"currency"`
	GroupsTruncated bool                     `json:"groups_truncated"`
	Totals          SettlementSummaryAmounts `json:"totals"`
	Groups          []SettlementSummaryGroup `json:"groups"`
}

type settlementSummaryAccumulator struct {
	RecordCount       int64
	PromptTokens      int64
	CompletionTokens  int64
	CacheTokens       int64
	OfficialTotalUSD  float64
	CostPriceUSD      float64
	OperatingPriceUSD float64
	SalesPriceUSD     float64
	UserPaidUSD       float64
}

func (a *settlementSummaryAccumulator) addLog(l *Log, breakdown SettlementPriceBreakdown, cacheTokens int) {
	if l == nil {
		return
	}
	a.RecordCount++
	a.PromptTokens += int64(l.PromptTokens)
	a.CompletionTokens += int64(l.CompletionTokens)
	a.CacheTokens += int64(cacheTokens)
	a.OfficialTotalUSD += breakdown.OfficialTotal
	a.CostPriceUSD += breakdown.CostPrice
	a.OperatingPriceUSD += breakdown.OperatingPrice
	a.SalesPriceUSD += breakdown.SalesPrice
	a.UserPaidUSD += QuotaToMoneyAmount(l.Quota)
}

func (a settlementSummaryAccumulator) toAmounts() SettlementSummaryAmounts {
	return SettlementSummaryAmounts{
		RecordCount:      a.RecordCount,
		PromptTokens:     a.PromptTokens,
		CompletionTokens: a.CompletionTokens,
		CacheTokens:      a.CacheTokens,
		OfficialTotal:    FormatSettlementMoney(a.OfficialTotalUSD),
		CostPrice:        FormatSettlementMoney(a.CostPriceUSD),
		OperatingPrice:   FormatSettlementMoney(a.OperatingPriceUSD),
		SalesPrice:       FormatSettlementMoney(a.SalesPriceUSD),
		UserPaid:         FormatSettlementMoney(a.UserPaidUSD),
	}
}

// ExtractCacheReadTokensFromOther 从日志 other 字段解析缓存读取 tokens。
func ExtractCacheReadTokensFromOther(other string) int {
	if other == "" {
		return 0
	}
	m, err := common.StrToMap(other)
	if err != nil || m == nil {
		return 0
	}
	keys := []string{
		"cache_tokens",
		"cache_read_tokens",
		"cached_tokens",
		"prompt_cache_hit_tokens",
		"cache_creation_tokens",
		"cache_write_tokens",
	}
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch x := v.(type) {
			case float64:
				if x > 0 {
					return int(x)
				}
			case int:
				if x > 0 {
					return x
				}
			case int64:
				if x > 0 {
					return int(x)
				}
			}
		}
	}
	return 0
}

func settlementGroupKeyLabel(scope string, l *Log, agentMap map[int]string) (string, string) {
	switch scope {
	case "user":
		key := strconv.Itoa(l.UserId)
		label := l.Username
		if label == "" {
			label = key
		}
		return key, label
	case "agent":
		agent := agentMap[l.UserId]
		if agent == "" {
			agent = "(无代理)"
		}
		return agent, agent
	default:
		key := strconv.Itoa(l.ChannelId)
		label := l.ChannelDisplay
		if label == "" {
			label = "#" + key
		}
		return key, label
	}
}

// BuildSettlementSummaryFromLogs 基于已加载日志构建结算汇总（导出与 API 共用）。
func BuildSettlementSummaryFromLogs(logs []*Log, scope string, agentMap map[int]string) *SettlementSummaryResult {
	totalsAcc := &settlementSummaryAccumulator{}
	groupAcc := make(map[string]*settlementSummaryAccumulator)
	groupLabel := make(map[string]string)

	for _, l := range logs {
		if l == nil {
			continue
		}
		otherMap, _ := common.StrToMap(l.Other)
		cacheTokens := ExtractCacheReadTokensFromOther(l.Other)
		breakdown := ComputeSettlementPriceBreakdown(l.PromptTokens, l.CompletionTokens, cacheTokens, l.Quota, otherMap)
		totalsAcc.addLog(l, breakdown, cacheTokens)

		key, label := settlementGroupKeyLabel(scope, l, agentMap)
		acc, ok := groupAcc[key]
		if !ok {
			acc = &settlementSummaryAccumulator{}
			groupAcc[key] = acc
			groupLabel[key] = label
		}
		acc.addLog(l, breakdown, cacheTokens)
	}

	groups := make([]SettlementSummaryGroup, 0, len(groupAcc))
	for key, acc := range groupAcc {
		groups = append(groups, SettlementSummaryGroup{
			Key:                      key,
			Label:                    groupLabel[key],
			SettlementSummaryAmounts: acc.toAmounts(),
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].RecordCount > groups[j].RecordCount
	})
	groupsTruncated := len(groupAcc) > settlementSummaryMaxGroups
	if groupsTruncated {
		groups = groups[:settlementSummaryMaxGroups]
	}

	if scope == "" {
		scope = "platform"
	}
	return &SettlementSummaryResult{
		Scope:           scope,
		Currency:        SettlementCurrencyLabel(),
		GroupsTruncated: groupsTruncated,
		Totals:          totalsAcc.toAmounts(),
		Groups:          groups,
	}
}

// GetSettlementSummary 按筛选条件汇总结算数据（与导出共用筛选，上限同导出）。
func GetSettlementSummary(filter SettlementExportFilter, scope string) (*SettlementSummaryResult, error) {
	logs, _, err := GetSettlementLogsForExport(filter)
	if err != nil {
		return nil, err
	}
	userIDs := make([]int, 0, len(logs))
	for _, l := range logs {
		if l != nil && l.UserId > 0 {
			userIDs = append(userIDs, l.UserId)
		}
	}
	agentMap := LoadInviterUsernameByUserIDs(userIDs)
	return BuildSettlementSummaryFromLogs(logs, scope, agentMap), nil
}

// applySettlementExportFilter 结算导出/汇总共用筛选。
func applySettlementExportFilter(tx *gorm.DB, filter SettlementExportFilter) *gorm.DB {
	tx = applyLogTypesFilter(tx, []int{LogTypeConsume})
	tx = applyBillingLogVisibility(tx, false)

	if filter.ModelName != "" {
		tx = tx.Where("logs.model_name LIKE ?", filter.ModelName)
	}
	if filter.Username != "" {
		tx = tx.Where("logs.username = ?", filter.Username)
	}
	if filter.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", filter.TokenName)
	}
	if filter.RequestID != "" {
		tx = tx.Where("logs.request_id = ?", filter.RequestID)
	}
	if filter.FromTs > 0 {
		tx = tx.Where("logs.created_at >= ?", filter.FromTs)
	}
	if filter.ToTs > 0 {
		tx = tx.Where("logs.created_at <= ?", filter.ToTs)
	}
	if len(filter.ChannelIDs) > 0 {
		tx = tx.Where("logs.channel_id IN ?", filter.ChannelIDs)
	} else if filter.Channel > 0 {
		tx = tx.Where("logs.channel_id = ?", filter.Channel)
	}
	if filter.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", filter.Group)
	}
	if len(filter.UserIDs) > 0 {
		tx = tx.Where("logs.user_id IN ?", filter.UserIDs)
	}
	if len(filter.InviterIDs) > 0 {
		tx = tx.Where("logs.user_id IN (?)",
			DB.Model(&User{}).Select("id").Where("inviter_id IN ?", filter.InviterIDs))
	}
	return tx
}
