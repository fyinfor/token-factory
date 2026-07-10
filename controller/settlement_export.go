package controller

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type settlementColumn struct {
	Key    string
	Header func(lang string) string
	Value  func(ctx settlementRowContext) string
}

type settlementRowContext struct {
	Index         int
	Log           *model.Log
	AgentUsername string
	Breakdown     model.SettlementPriceBreakdown
	CacheTokens   int
	Lang          string
}

var settlementColumns = []settlementColumn{
	{Key: "seq", Header: settlementHeader("序号", "No."), Value: func(ctx settlementRowContext) string {
		return strconv.Itoa(ctx.Index)
	}},
	{Key: "request_id", Header: settlementHeader("订单ID", "Request ID"), Value: func(ctx settlementRowContext) string {
		return ctx.Log.RequestId
	}},
	{Key: "time", Header: settlementHeader("时间", "Time"), Value: func(ctx settlementRowContext) string {
		return time.Unix(ctx.Log.CreatedAt, 0).Format("2006-01-02 15:04:05")
	}},
	{Key: "channel", Header: settlementHeader("渠道", "Channel"), Value: func(ctx settlementRowContext) string {
		if ctx.Log.ChannelDisplay != "" {
			return ctx.Log.ChannelDisplay
		}
		return strconv.Itoa(ctx.Log.ChannelId)
	}},
	{Key: "model", Header: settlementHeader("模型", "Model"), Value: func(ctx settlementRowContext) string {
		return ctx.Log.ModelName
	}},
	{Key: "agent", Header: settlementHeader("代理商", "Agent"), Value: func(ctx settlementRowContext) string {
		return ctx.AgentUsername
	}},
	{Key: "user", Header: settlementHeader("用户", "User"), Value: func(ctx settlementRowContext) string {
		return ctx.Log.Username
	}},
	{Key: "prompt_tokens", Header: settlementHeader("输入 tokens", "Input tokens"), Value: func(ctx settlementRowContext) string {
		return strconv.Itoa(ctx.Log.PromptTokens)
	}},
	{Key: "completion_tokens", Header: settlementHeader("输出 tokens", "Output tokens"), Value: func(ctx settlementRowContext) string {
		return strconv.Itoa(ctx.Log.CompletionTokens)
	}},
	{Key: "cache_tokens", Header: settlementHeader("缓存 tokens", "Cache tokens"), Value: func(ctx settlementRowContext) string {
		return strconv.Itoa(ctx.CacheTokens)
	}},
	{Key: "official_input_price", Header: settlementMoneyHeader("官方输入价格", "Official input price"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.OfficialInputPrice)
	}},
	{Key: "official_output_price", Header: settlementMoneyHeader("官方输出价格", "Official output price"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.OfficialOutputPrice)
	}},
	{Key: "official_cache_price", Header: settlementMoneyHeader("官方缓存价格", "Official cache price"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.OfficialCachePrice)
	}},
	{Key: "official_total", Header: settlementMoneyHeader("官方总价", "Official total"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.OfficialTotal)
	}},
	{Key: "cost_discount", Header: settlementHeader("成本折扣", "Cost discount"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementPercent(ctx.Breakdown.Discounts.PriceDiscountPercent)
	}},
	{Key: "operating_cost", Header: settlementHeader("经营成本", "Operating cost"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementPercent(ctx.Breakdown.Discounts.OperatingCostPercent)
	}},
	{Key: "markup_discount", Header: settlementHeader("加价折扣", "Markup discount"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementPercent(ctx.Breakdown.Discounts.MarkupDiscountPercent)
	}},
	{Key: "sales_discount", Header: settlementHeader("销售折扣", "Sales discount"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementPercent(ctx.Breakdown.Discounts.SalesDiscountPercent)
	}},
	{Key: "cost_price", Header: settlementMoneyHeader("成本价", "Cost price"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.CostPrice)
	}},
	{Key: "operating_price", Header: settlementMoneyHeader("经营单价", "Operating price"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.OperatingPrice)
	}},
	{Key: "sales_price", Header: settlementMoneyHeader("销售单价", "Sales price"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(ctx.Breakdown.SalesPrice)
	}},
	{Key: "quota", Header: settlementMoneyHeader("用户实付", "User paid"), Value: func(ctx settlementRowContext) string {
		return model.FormatSettlementMoney(model.QuotaToMoneyAmount(ctx.Log.Quota))
	}},
}

func settlementHeader(zh, en string) func(string) string {
	return func(lang string) string {
		if lang == "en" {
			return en
		}
		return zh
	}
}

func settlementMoneyHeader(zh, en string) func(string) string {
	return func(lang string) string {
		currency := model.SettlementCurrencyLabel()
		if lang == "en" {
			return fmt.Sprintf("%s (%s)", en, currency)
		}
		return fmt.Sprintf("%s(%s)", zh, currency)
	}
}

func defaultSettlementColumnKeys() []string {
	keys := make([]string, 0, len(settlementColumns))
	for _, col := range settlementColumns {
		keys = append(keys, col.Key)
	}
	return keys
}

func parseSettlementColumnKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return defaultSettlementColumnKeys()
	}
	requested := strings.Split(raw, ",")
	allowed := make(map[string]struct{}, len(settlementColumns))
	for _, col := range settlementColumns {
		allowed[col.Key] = struct{}{}
	}
	out := make([]string, 0, len(requested))
	for _, key := range requested {
		key = strings.TrimSpace(key)
		if _, ok := allowed[key]; ok {
			out = append(out, key)
		}
	}
	if len(out) == 0 {
		return defaultSettlementColumnKeys()
	}
	return out
}

func resolveSettlementColumns(keys []string) []settlementColumn {
	allowed := make(map[string]settlementColumn, len(settlementColumns))
	for _, col := range settlementColumns {
		allowed[col.Key] = col
	}
	out := make([]settlementColumn, 0, len(keys))
	for _, key := range keys {
		if col, ok := allowed[key]; ok {
			out = append(out, col)
		}
	}
	return out
}

type settlementExportQuery struct {
	StartTs     int64
	EndTs       int64
	Lang        string
	Scope       string
	ChannelIDs  []int
	UserIDs     []int
	InviterIDs  []int
	ColumnKeys  []string
}

func parseIntListQuery(c *gin.Context, keys ...string) []int {
	seen := make(map[int]struct{})
	out := make([]int, 0)
	for _, key := range keys {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" {
			continue
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil || id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func parseSettlementExportQuery(c *gin.Context) (settlementExportQuery, error) {
	query, err := parseLogExportQueryWithMaxWindow(c, settlementExportMaxWindowSeconds, "最多 1 年")
	if err != nil {
		return settlementExportQuery{}, err
	}
	out := settlementExportQuery{
		StartTs:    query.StartTs,
		EndTs:      query.EndTs,
		Lang:       query.Lang,
		Scope:      strings.TrimSpace(c.Query("scope")),
		ColumnKeys: parseSettlementColumnKeys(c.Query("columns")),
	}
	out.ChannelIDs = parseIntListQuery(c, "channel_ids", "channel_id", "channel")
	out.UserIDs = parseIntListQuery(c, "user_ids", "user_id")
	out.InviterIDs = parseIntListQuery(c, "inviter_ids", "inviter_id")
	switch out.Scope {
	case "channel":
		if len(out.ChannelIDs) == 0 {
			return out, fmt.Errorf("按渠道结算需至少选择一个渠道")
		}
	case "user":
		if len(out.UserIDs) == 0 {
			return out, fmt.Errorf("按用户结算需至少选择一个用户")
		}
	case "agent":
		if len(out.InviterIDs) == 0 {
			return out, fmt.Errorf("按代理结算需至少选择一个代理商")
		}
	}
	return out, nil
}

// GetSettlementSummary 管理员结算汇总（与导出共用筛选条件）。
func GetSettlementSummary(c *gin.Context) {
	query, err := parseSettlementExportQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filter := model.SettlementExportFilter{
		AdminLogExportFilter: model.AdminLogExportFilter{
			LogExportFilter: model.LogExportFilter{
				FromTs: query.StartTs,
				ToTs:   query.EndTs,
			},
		},
		ChannelIDs: query.ChannelIDs,
		UserIDs:    query.UserIDs,
		InviterIDs: query.InviterIDs,
	}
	result, err := model.GetSettlementSummary(filter, query.Scope)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

// ExportSettlementLogs 管理员结算单导出（可勾选字段、多视角筛选）。
func ExportSettlementLogs(c *gin.Context) {
	query, err := parseSettlementExportQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filter := model.SettlementExportFilter{
		AdminLogExportFilter: model.AdminLogExportFilter{
			LogExportFilter: model.LogExportFilter{
				FromTs: query.StartTs,
				ToTs:   query.EndTs,
			},
		},
		ChannelIDs: query.ChannelIDs,
		UserIDs:    query.UserIDs,
		InviterIDs: query.InviterIDs,
	}
	logs, _, err := model.GetSettlementLogsForExport(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userIDs := make([]int, 0, len(logs))
	for _, l := range logs {
		if l != nil && l.UserId > 0 {
			userIDs = append(userIDs, l.UserId)
		}
	}
	agentMap := model.LoadInviterUsernameByUserIDs(userIDs)
	cols := resolveSettlementColumns(query.ColumnKeys)
	summary := model.BuildSettlementSummaryFromLogs(logs, query.Scope, agentMap)
	streamSettlementCSV(c, logs, query, cols, agentMap, summary)
}

type settlementSummaryExportI18n struct {
	DetailSection      string
	TotalSection       string
	GroupSectionUser   string
	GroupSectionAgent  string
	GroupSectionChannel string
	TotalLabel         string
	GroupLabelCol      string
	GroupsTruncatedFmt string
	Headers            []string
}

var settlementSummaryExportDictZHCN = settlementSummaryExportI18n{
	DetailSection:       "【明细】",
	TotalSection:        "【汇总合计】",
	GroupSectionChannel: "【渠道明细】",
	GroupSectionUser:    "【用户明细】",
	GroupSectionAgent:   "【代理明细】",
	TotalLabel:          "合计",
	GroupLabelCol:       "分组",
	GroupsTruncatedFmt:  "分组较多，仅展示前 %d 项",
	Headers: []string{
		"请求笔数", "输入 tokens", "输出 tokens", "缓存 tokens",
		"官方总价", "成本价", "经营单价", "销售单价", "用户实付",
	},
}

var settlementSummaryExportDictEN = settlementSummaryExportI18n{
	DetailSection:       "[Details]",
	TotalSection:        "[Summary total]",
	GroupSectionChannel: "[Channel breakdown]",
	GroupSectionUser:    "[User breakdown]",
	GroupSectionAgent:   "[Agent breakdown]",
	TotalLabel:          "Total",
	GroupLabelCol:       "Group",
	GroupsTruncatedFmt:  "Showing first %d groups only",
	Headers: []string{
		"Requests", "Input tokens", "Output tokens", "Cache tokens",
		"Official total", "Cost price", "Operating price", "Sales price", "User paid",
	},
}

func resolveSettlementSummaryExportDict(lang string) settlementSummaryExportI18n {
	if lang == "en" {
		return settlementSummaryExportDictEN
	}
	return settlementSummaryExportDictZHCN
}

func settlementSummaryGroupSectionTitle(scope, lang string) string {
	dict := resolveSettlementSummaryExportDict(lang)
	switch scope {
	case "user":
		return dict.GroupSectionUser
	case "agent":
		return dict.GroupSectionAgent
	default:
		return dict.GroupSectionChannel
	}
}

func settlementSummaryMoneyHeaders(headers []string, lang string) []string {
	currency := model.SettlementCurrencyLabel()
	out := make([]string, len(headers))
	for i, h := range headers {
		if i < 4 {
			out[i] = h
			continue
		}
		if lang == "en" {
			out[i] = fmt.Sprintf("%s (%s)", h, currency)
		} else {
			out[i] = fmt.Sprintf("%s(%s)", h, currency)
		}
	}
	return out
}

func settlementSummaryAmountsToRow(label string, amounts model.SettlementSummaryAmounts) []string {
	return []string{
		label,
		strconv.FormatInt(amounts.RecordCount, 10),
		strconv.FormatInt(amounts.PromptTokens, 10),
		strconv.FormatInt(amounts.CompletionTokens, 10),
		strconv.FormatInt(amounts.CacheTokens, 10),
		amounts.OfficialTotal,
		amounts.CostPrice,
		amounts.OperatingPrice,
		amounts.SalesPrice,
		amounts.UserPaid,
	}
}

func writeSettlementSummarySection(w *csv.Writer, summary *model.SettlementSummaryResult, lang string) error {
	if summary == nil {
		return nil
	}
	dict := resolveSettlementSummaryExportDict(lang)
	colCount := len(dict.Headers) + 1
	emptyRow := make([]string, colCount)
	moneyHeaders := settlementSummaryMoneyHeaders(dict.Headers, lang)
	summaryHeader := append([]string{dict.GroupLabelCol}, moneyHeaders...)

	if err := writeStatementRow(w, emptyRow); err != nil {
		return err
	}
	if err := writeStatementRow(w, append([]string{dict.TotalSection}, emptyRow[1:]...)); err != nil {
		return err
	}
	if err := writeStatementRow(w, summaryHeader); err != nil {
		return err
	}
	totalRow := settlementSummaryAmountsToRow(dict.TotalLabel, summary.Totals)
	if err := writeStatementRow(w, totalRow); err != nil {
		return err
	}

	if len(summary.Groups) == 0 {
		return nil
	}
	if err := writeStatementRow(w, emptyRow); err != nil {
		return err
	}
	groupTitle := settlementSummaryGroupSectionTitle(summary.Scope, lang)
	if err := writeStatementRow(w, append([]string{groupTitle}, emptyRow[1:]...)); err != nil {
		return err
	}
	if summary.GroupsTruncated {
		note := fmt.Sprintf(dict.GroupsTruncatedFmt, len(summary.Groups))
		if err := writeStatementRow(w, append([]string{note}, emptyRow[1:]...)); err != nil {
			return err
		}
	}
	if err := writeStatementRow(w, summaryHeader); err != nil {
		return err
	}
	for _, group := range summary.Groups {
		row := settlementSummaryAmountsToRow(group.Label, group.SettlementSummaryAmounts)
		if err := writeStatementRow(w, row); err != nil {
			return err
		}
	}
	return nil
}

func streamSettlementCSV(c *gin.Context, logs []*model.Log, query settlementExportQuery, cols []settlementColumn, agentMap map[int]string, summary *model.SettlementSummaryResult) {
	filename := fmt.Sprintf("settlement-%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Statement-Window-Start", strconv.FormatInt(query.StartTs, 10))
	c.Header("X-Statement-Window-End", strconv.FormatInt(query.EndTs, 10))
	c.Status(200)

	c.Writer.WriteString("\xEF\xBB\xBF")
	bw := bufio.NewWriterSize(c.Writer, 32*1024)
	defer bw.Flush()
	w := csv.NewWriter(bw)
	defer w.Flush()

	periodStart := time.Unix(query.StartTs, 0).Format("2006-01-02 15:04:05")
	periodEnd := time.Unix(query.EndTs, 0).Format("2006-01-02 15:04:05")
	metaTitle := "结算对账单"
	if query.Lang == "en" {
		metaTitle = "Settlement statement"
	}
	_ = writeStatementRow(w, []string{metaTitle})
	_ = writeStatementRow(w, []string{fmt.Sprintf("%s ~ %s", periodStart, periodEnd)})

	summaryDict := resolveSettlementSummaryExportDict(query.Lang)
	detailSection := summaryDict.DetailSection
	_ = writeStatementRow(w, []string{detailSection})

	header := make([]string, len(cols))
	for i, col := range cols {
		header[i] = col.Header(query.Lang)
	}
	_ = writeStatementRow(w, header)

	for idx, l := range logs {
		if l == nil {
			continue
		}
		otherMap, _ := common.StrToMap(l.Other)
		cacheTokens := extractCacheReadTokens(l.Other)
		breakdown := model.ComputeSettlementPriceBreakdown(l.PromptTokens, l.CompletionTokens, cacheTokens, l.Quota, otherMap)
		ctx := settlementRowContext{
			Index:         idx + 1,
			Log:           l,
			AgentUsername: agentMap[l.UserId],
			Breakdown:     breakdown,
			CacheTokens:   cacheTokens,
			Lang:          query.Lang,
		}
		row := make([]string, len(cols))
		for i, col := range cols {
			row[i] = col.Value(ctx)
		}
		_ = writeStatementRow(w, row)
	}

	if err := writeSettlementSummarySection(w, summary, query.Lang); err != nil {
		common.SysError("write settlement export summary: " + err.Error())
	}
}

// AdminBackfillInvoiceAttribution 管理员按用户回填充值消耗归因。
func AdminBackfillInvoiceAttribution(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Query("user_id"))
	if userID <= 0 {
		common.ApiError(c, fmt.Errorf("user_id required"))
		return
	}
	if err := model.BackfillTopUpConsumeAttribution(userID); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"user_id": userID})
}

// GetInvoiceRequestDetailAdmin 运营端查看发票申请详情。
func GetInvoiceRequestDetailAdmin(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	req, err := model.GetInvoiceRequestByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items, err := model.GetInvoiceRequestItems(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user, _ := model.GetUserById(req.UserId, false)
	username, email := "", ""
	if user != nil {
		username = user.Username
		email = user.Email
	}
	common.ApiSuccess(c, gin.H{
		"request":  req,
		"items":    items,
		"username": username,
		"email":    email,
	})
}
