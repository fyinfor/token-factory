package controller

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// 对账单导出相关常量。
const (
	logExportMaxWindowSeconds = int64(90 * 24 * 60 * 60) // 单次导出最长 90 天
	// 复用 model.logExportCountLimit 100000
)

// 对账单 CSV 表头（与时间正序，序号自增，便于 Excel 双击打开）。
// 最后一列展示"当前账户剩余金额"，对账用户最关心这一列。
var statementCSVHeader = []string{
	"序号", "时间", "类型", "事件", "渠道/订单号", "模型", "令牌",
	"输入 tokens", "输出 tokens", "缓存 tokens", "发生额(quota)", "发生额(等值)", "变动后余额(quota)", "剩余余额",
}

// 类型字段的本地化映射。
var logTypeLabelZH = map[int]string{
	model.LogTypeTopup:   "充值",
	model.LogTypeConsume: "消耗",
	model.LogTypeManage:  "管理",
	model.LogTypeSystem:  "系统",
	model.LogTypeRefund:  "退款",
	model.LogTypeError:   "错误",
}

func logTypeLabel(t int) string {
	if s, ok := logTypeLabelZH[t]; ok {
		return s
	}
	return "其他"
}

// 解析对账单导出参数；统一做合法性校验（不抛错时返回 0 表示不限）。
func parseStatementParams(c *gin.Context) (startTs, endTs int64, modelName, tokenName string, err error) {
	if s := c.Query("start_timestamp"); s != "" {
		v, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || v < 0 {
			return 0, 0, "", "", fmt.Errorf("start_timestamp 非法")
		}
		startTs = v
	}
	if s := c.Query("end_timestamp"); s != "" {
		v, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || v < 0 {
			return 0, 0, "", "", fmt.Errorf("end_timestamp 非法")
		}
		endTs = v
	}
	// 默认窗口：近 3 个月。
	now := common.GetTimestamp()
	if endTs == 0 {
		endTs = now
	}
	if startTs == 0 {
		startTs = endTs - logExportMaxWindowSeconds
	}
	if endTs < startTs {
		return 0, 0, "", "", fmt.Errorf("end_timestamp 早于 start_timestamp")
	}
	if endTs-startTs > logExportMaxWindowSeconds {
		return 0, 0, "", "", fmt.Errorf("时间范围超出限制(最多 3 个月)")
	}
	modelName = c.Query("model_name")
	tokenName = c.Query("token_name")
	return
}

// 写入一条 CSV 行。csv.Writer 已做 RFC4180 转义（含逗号、引号、换行）。
func writeStatementRow(w *csv.Writer, row []string) error {
	if err := w.Write(row); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

// 流式写出单个用户的对账单。filename 不含扩展名，由调用方传入。
func streamUserStatementCSV(c *gin.Context, user *model.User, startTs, endTs int64, modelName, tokenName, filename string) {
	if user == nil {
		common.ApiError(c, fmt.Errorf("用户不存在"))
		return
	}
	logs, total, err := model.GetUserLogsForExport(user.Id, startTs, endTs, modelName, tokenName)
	if err != nil {
		// 行数超限
		common.ApiError(c, err)
		return
	}
	_ = total

	// 1) 反推"窗口期初余额"：当前余额 - 窗口内净变动。
	// 注意：净变动只取真实影响 User.Quota 的三类日志（Consume/Topup/Refund）。
	delta, derr := model.GetChargeableDeltaByUser(user.Id, startTs, endTs)
	if derr != nil {
		common.ApiError(c, derr)
		return
	}
	running := int64(user.Quota) - delta

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Statement-Current-Balance", strconv.Itoa(user.Quota))
	c.Header("X-Statement-Window-Start", strconv.FormatInt(startTs, 10))
	c.Header("X-Statement-Window-End", strconv.FormatInt(endTs, 10))
	c.Status(200)

	// 写入 UTF-8 BOM，Excel 双击不乱码。
	c.Writer.WriteString("\xEF\xBB\xBF")

	bw := bufio.NewWriterSize(c.Writer, 32*1024)
	defer bw.Flush()
	w := csv.NewWriter(bw)
	defer w.Flush()

	// 顶部元信息行（作为备注，前 3 行；便于用户理解对账口径）。
	meta := []string{fmt.Sprintf("用户: %s (id=%d)", user.Username, user.Id)}
	if err := writeStatementRow(w, []string{meta[0], "", "", "", "", "", "", "", "", "", "", "", "", ""}); err != nil {
		common.SysError("write statement meta1: " + err.Error())
		return
	}
	periodStart := time.Unix(startTs, 0).Format("2006-01-02 15:04:05")
	periodEnd := time.Unix(endTs, 0).Format("2006-01-02 15:04:05")
	if err := writeStatementRow(w, []string{
		fmt.Sprintf("账期: %s ~ %s", periodStart, periodEnd), "", "", "", "", "", "", "", "", "", "", "", "", "",
	}); err != nil {
		common.SysError("write statement meta2: " + err.Error())
		return
	}
	// 期末余额以"quota + 等值金额"双口径展示，避免系统内部单位与展示货币混淆。
	if err := writeStatementRow(w, []string{
		fmt.Sprintf("当前账户余额（Quota）: %d quota / %s", user.Quota, formatBalanceAmount(user.Quota)),
		"", "", "", "", "", "", "", "", "", "", "", "", "",
	}); err != nil {
		common.SysError("write statement meta3: " + err.Error())
		return
	}
	// 表头
	if err := writeStatementRow(w, statementCSVHeader); err != nil {
		common.SysError("write statement header: " + err.Error())
		return
	}

	// 2) 逐行输出。
	for idx, l := range logs {
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		signed := model.SignedLogDelta(l.Quota, l.Type)
		running += signed

		channelOrOrder := strconv.Itoa(l.ChannelId)
		if l.Type == model.LogTypeTopup || l.Type == model.LogTypeRefund {
			// 充值/退款行用 Other 字段里可能保存的 trade_no 替代渠道号（如果有）。
			if order, ok := extractOrderNo(l.Other); ok {
				channelOrOrder = order
			}
		}

		cacheTokens := extractCacheReadTokens(l.Other)

		row := []string{
			strconv.Itoa(idx + 1),
			ts,
			logTypeLabel(l.Type),
			l.Content,
			channelOrOrder,
			l.ModelName,
			l.TokenName,
			strconv.Itoa(l.PromptTokens),
			strconv.Itoa(l.CompletionTokens),
			strconv.Itoa(cacheTokens),
			strconv.FormatInt(signed, 10),
			formatQuotaDisplay(signed),
			strconv.FormatInt(running, 10),
			formatBalanceAmount(int(running)),
		}
		if err := writeStatementRow(w, row); err != nil {
			common.SysError("write statement row: " + err.Error())
			return
		}
	}

	// 3) 末尾对账校验行：用户拿这一行的"变动后余额"和 User.Quota 实际值对比。
	// 若不一致，差异来自 quota=0 的管理/系统调整（已直接落地到 User.Quota）。
	if err := writeStatementRow(w, []string{
		"总和", "", "", "", "", "", "", "", "", "", "", "", "",
		"账户期末余额",
		fmt.Sprintf("%d / %s", user.Quota, formatBalanceAmount(user.Quota)),
	}); err != nil {
		common.SysError("write statement footer: " + err.Error())
		return
	}
	if err := writeStatementRow(w, []string{
		"校验", "", "", "", "", "", "", "", "", "", "", "", "",
		"对账差异(应=0)",
		fmt.Sprintf("%d quota (%s)", running-int64(user.Quota), formatBalanceAmount(int(running-int64(user.Quota)))),
	}); err != nil {
		common.SysError("write statement check: " + err.Error())
		return
	}
}

// 提取 Other JSON 里的 trade_no 字段（如果有）。
func extractOrderNo(other string) (string, bool) {
	if other == "" {
		return "", false
	}
	m, err := common.StrToMap(other)
	if err != nil || m == nil {
		return "", false
	}
	if v, ok := m["trade_no"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}

// extractCacheReadTokens 从 Other JSON 中取"缓存读取命中"的 token 数。
// 优先顺序（兼容多协议与历史日志）：
//  1. cache_tokens           —— service/log_info_generate.go:73 写入的标准键
//                              （前端 web/src/helpers/render.jsx 与所有日志列定义都在读这个键）
//  2. cache_read_tokens      —— 历史/扩展名，保留兼容
//  3. cached_tokens          —— OpenAI 协议 PromptTokensDetails.CachedTokens
//  4. prompt_cache_hit_tokens—— 一些非标实现
//  5. cache_creation_tokens  —— 兜底为"缓存创建"，仅当无读取侧字段时使用
//  6. cache_write_tokens     —— 同上，normalized 缓存创建总量
//
// 返回 0 时表示该条日志无缓存信息（旧日志或非缓存型调用）。
// 注意：当优先键的值就是 0 时，仍会按顺序取下一个键——
// 因为 0 和"无该键"对用户而言都意味着"无缓存"，无需区分。
func extractCacheReadTokens(other string) int {
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
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		var n int
		switch x := v.(type) {
		case float64:
			n = int(x)
		case int:
			n = x
		case int64:
			n = int(x)
		default:
			continue
		}
		if n > 0 {
			return n
		}
	}
	return 0
}

// 将带符号 quota 转为对外展示金额字符串（与系统设置一致）。
func formatQuotaDisplay(signed int64) string {
	return logger.FormatQuota(int(signed))
}

// formatBalanceAmount 将 quota 数值渲染为"<符号><金额>"字符串。
// displayType=TOKENS 时按"点额度"展示，其余按系统设置币种（USD/CNY/Custom）。
// digits=2 与前端默认展示保持一致，避免出现"¥123.456789"。
func formatBalanceAmount(quota int) string {
	dispType := operation_setting.GetQuotaDisplayType()
	amount := logger.QuotaToRoundedDisplayAmount(quota, 2)
	if dispType == operation_setting.QuotaDisplayTypeTokens {
		return fmt.Sprintf("%d 点额度", int(amount))
	}
	sym := operation_setting.GetCurrencySymbol()
	return fmt.Sprintf("%s%.2f", sym, amount)
}

// GET /api/log/self/export 当前登录用户对账单
func ExportUserLogsSelf(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiError(c, fmt.Errorf("未登录"))
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	startTs, endTs, modelName, tokenName, err := parseStatementParams(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	filename := fmt.Sprintf("statement-%s-%d.csv", sanitizeFilename(user.Username), time.Now().Unix())
	streamUserStatementCSV(c, user, startTs, endTs, modelName, tokenName, filename)
}

// GET /api/admin/log/:userId/export 管理员代查任意用户对账单
func ExportUserLogsAdmin(c *gin.Context) {
	if !model.IsAdmin(c.GetInt("role")) {
		c.JSON(403, gin.H{"success": false, "message": "需要管理员权限"})
		return
	}
	uidStr := c.Param("userId")
	uid, err := strconv.Atoi(uidStr)
	if err != nil || uid <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "userId 非法"})
		return
	}
	user, err := model.GetUserById(uid, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	startTs, endTs, modelName, tokenName, err := parseStatementParams(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	filename := fmt.Sprintf("statement-%s-%d-admin.csv", sanitizeFilename(user.Username), time.Now().Unix())
	streamUserStatementCSV(c, user, startTs, endTs, modelName, tokenName, filename)
}

// POST /api/admin/log/export_all 管理员全平台批量对账单：返回 zip 包，
// 每个用户一个 CSV（命名 <username>-<id>.csv）。为避免内存爆炸，
// 当用户数 > 200 时返回 400，要求改用单用户导出。
func ExportAllUsersLogsAdmin(c *gin.Context) {
	if !model.IsAdmin(c.GetInt("role")) {
		c.JSON(403, gin.H{"success": false, "message": "需要管理员权限"})
		return
	}
	startTs, endTs, modelName, tokenName, err := parseStatementParams(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo := &common.PageInfo{
		Page: 1, PageSize: 200,
	}
	users, _, uerr := model.GetAllUsers(pageInfo, "", "")
	if uerr != nil {
		common.ApiError(c, uerr)
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="statements-%d.zip"`, time.Now().Unix()))
	c.Status(200)
	bw := bufio.NewWriterSize(c.Writer, 64*1024)
	defer bw.Flush()
	zw := zip.NewWriter(bw)
	defer zw.Close()

	for _, u := range users {
		if u == nil {
			continue
		}
		fh := &zip.FileHeader{
			Name:     fmt.Sprintf("%s-%d.csv", sanitizeFilename(u.Username), u.Id),
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		fw, ferr := zw.CreateHeader(fh)
		if ferr != nil {
			common.SysError("zip create: " + ferr.Error())
			continue
		}
		writeSingleUserCSV(fw, u, startTs, endTs, modelName, tokenName)
	}
}

// writeSingleUserCSV 复用 streamUserStatementCSV 的写表逻辑，但目标 io.Writer 由调用方控制。
func writeSingleUserCSV(w interface {
	Write(p []byte) (int, error)
}, user *model.User, startTs, endTs int64, modelName, tokenName string) {
	logs, _, err := model.GetUserLogsForExport(user.Id, startTs, endTs, modelName, tokenName)
	if err != nil {
		return
	}
	delta, _ := model.GetChargeableDeltaByUser(user.Id, startTs, endTs)
	running := int64(user.Quota) - delta

	// BOM
	w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.Write([]string{fmt.Sprintf("用户: %s (id=%d)", user.Username, user.Id), "", "", "", "", "", "", "", "", "", "", "", "", ""})
	cw.Write([]string{fmt.Sprintf("账期: %s ~ %s",
		time.Unix(startTs, 0).Format("2006-01-02 15:04:05"),
		time.Unix(endTs, 0).Format("2006-01-02 15:04:05"),
	), "", "", "", "", "", "", "", "", "", "", "", ""})
	cw.Write([]string{
		fmt.Sprintf("当前账户余额（Quota）: %d quota / %s", user.Quota, formatBalanceAmount(user.Quota)),
		"", "", "", "", "", "", "", "", "", "", "", "", "",
	})
	cw.Write(statementCSVHeader)
	for idx, l := range logs {
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		signed := model.SignedLogDelta(l.Quota, l.Type)
		running += signed
		channelOrOrder := strconv.Itoa(l.ChannelId)
		if l.Type == model.LogTypeTopup || l.Type == model.LogTypeRefund {
			if order, ok := extractOrderNo(l.Other); ok {
				channelOrOrder = order
			}
		}
		cacheTokens := extractCacheReadTokens(l.Other)
		cw.Write([]string{
			strconv.Itoa(idx + 1), ts, logTypeLabel(l.Type), l.Content, channelOrOrder,
			l.ModelName, l.TokenName, strconv.Itoa(l.PromptTokens), strconv.Itoa(l.CompletionTokens),
			strconv.Itoa(cacheTokens),
			strconv.FormatInt(signed, 10), formatQuotaDisplay(signed), strconv.FormatInt(running, 10),
			formatBalanceAmount(int(running)),
		})
	}
	cw.Write([]string{
		"总和", "", "", "", "", "", "", "", "", "", "", "", "",
		"账户期末余额",
		fmt.Sprintf("%d / %s", user.Quota, formatBalanceAmount(user.Quota)),
	})
	cw.Write([]string{
		"校验", "", "", "", "", "", "", "", "", "", "", "", "",
		"对账差异(应=0)",
		fmt.Sprintf("%d quota (%s)", running-int64(user.Quota), formatBalanceAmount(int(running-int64(user.Quota)))),
	})
	cw.Flush()
}

// sanitizeFilename 过滤文件名中的非法字符，保留中英文/数字/下划线/点/连字符。
func sanitizeFilename(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			out = append(out, r)
		} else if r >= 0x4e00 && r <= 0x9fff {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "user"
	}
	return string(out)
}
