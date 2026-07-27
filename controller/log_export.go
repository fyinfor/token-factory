package controller

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// 对账单导出相关常量。
const (
	logExportMaxWindowSeconds        = int64(90 * 24 * 60 * 60)  // 对账单/日志导出最长 90 天
	settlementExportMaxWindowSeconds = int64(365 * 24 * 60 * 60) // 结算单导出最长 1 年
	// 复用 model.logExportCountLimit 100000
)

// 对账单 CSV 表头（与日志页面一致，最新记录在前，序号自增）。
// 最后一列展示"当前账户剩余金额"，对账用户最关心这一列。
var statementCSVHeader = []string{
	"序号", "时间", "类型", "事件", "渠道/订单号", "模型", "令牌",
	"输入 tokens", "输出 tokens", "缓存 tokens", "发生额(quota)", "发生额(等值)", "变动后余额(quota)", "剩余余额",
}

// 对账单导出文案国际化表。
// 11 种语言（与 web/src/i18n/i18n.js 的 supportedLanguages 对齐）每种提供：
//   - Header：14 列 CSV 表头
//   - Meta1/2/3：前 3 行元信息前缀
//   - Footer1/2：末尾对账两行标签
//   - LogType 6 个标签（消耗/充值/退款/管理/系统/错误）
//   - DefaultLabel："其他"兜底
//
// 浏览器未指定 lang 时回退到 zh-CN。前端调用时会传 lang=xx-XX。
type statementI18n struct {
	Header       []string
	Meta1        string // "用户: %s (id=%d)"
	Meta2        string // "账期: %s ~ %s"
	Meta3        string // "当前账户余额（Quota）: %d quota / %s"
	Footer1Label string // "总和"
	Footer1Key   string // "账户期末余额"
	Footer2Label string // "校验"
	Footer2Key   string // "对账差异(应=0)"
	LogType      map[int]string
	DefaultLabel string
}

var statementDictZHCN = statementI18n{
	Header: []string{
		"序号", "时间", "类型", "事件", "渠道/订单号", "模型", "令牌",
		"输入 tokens", "输出 tokens", "缓存 tokens", "发生额(quota)", "发生额(等值)", "变动后余额(quota)", "剩余余额",
	},
	Meta1:        "用户: %s (id=%d)",
	Meta2:        "账期: %s ~ %s",
	Meta3:        "当前账户余额（Quota）: %d quota / %s",
	Footer1Label: "总和",
	Footer1Key:   "账户期末余额",
	Footer2Label: "校验",
	Footer2Key:   "对账差异(应=0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "充值",
		model.LogTypeConsume: "消耗",
		model.LogTypeManage:  "管理",
		model.LogTypeSystem:  "系统",
		model.LogTypeRefund:  "退款",
		model.LogTypeError:   "错误",
	},
	DefaultLabel: "其他",
}

var statementDictZHTW = statementI18n{
	Header: []string{
		"序號", "時間", "類型", "事件", "渠道/訂單號", "模型", "令牌",
		"輸入 tokens", "輸出 tokens", "快取 tokens", "發生額(quota)", "發生額(等值)", "變動後餘額(quota)", "剩餘餘額",
	},
	Meta1:        "用戶: %s (id=%d)",
	Meta2:        "帳期: %s ~ %s",
	Meta3:        "目前帳戶餘額（Quota）: %d quota / %s",
	Footer1Label: "總和",
	Footer1Key:   "帳戶期末餘額",
	Footer2Label: "校驗",
	Footer2Key:   "對帳差異(應=0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "儲值",
		model.LogTypeConsume: "消耗",
		model.LogTypeManage:  "管理",
		model.LogTypeSystem:  "系統",
		model.LogTypeRefund:  "退款",
		model.LogTypeError:   "錯誤",
	},
	DefaultLabel: "其他",
}

var statementDictEN = statementI18n{
	Header: []string{
		"No.", "Time", "Type", "Event", "Channel/Order No.", "Model", "Token",
		"Input tokens", "Output tokens", "Cache tokens", "Quota (signed)", "Amount (signed)", "Running quota", "Running balance",
	},
	Meta1:        "User: %s (id=%d)",
	Meta2:        "Period: %s ~ %s",
	Meta3:        "Current balance (Quota): %d quota / %s",
	Footer1Label: "Total",
	Footer1Key:   "Period-end balance",
	Footer2Label: "Check",
	Footer2Key:   "Reconciliation delta (should be 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Topup",
		model.LogTypeConsume: "Consume",
		model.LogTypeManage:  "Manage",
		model.LogTypeSystem:  "System",
		model.LogTypeRefund:  "Refund",
		model.LogTypeError:   "Error",
	},
	DefaultLabel: "Other",
}

var statementDictFR = statementI18n{
	Header: []string{
		"N°", "Heure", "Type", "Événement", "Canal/N° commande", "Modèle", "Jeton",
		"Tokens d'entrée", "Tokens de sortie", "Tokens de cache", "Quota (signé)", "Montant (signé)", "Quota cumulé", "Solde courant",
	},
	Meta1:        "Utilisateur : %s (id=%d)",
	Meta2:        "Période : %s ~ %s",
	Meta3:        "Solde actuel (Quota) : %d quota / %s",
	Footer1Label: "Total",
	Footer1Key:   "Solde de fin de période",
	Footer2Label: "Vérification",
	Footer2Key:   "Écart de réconciliation (doit être 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Recharge",
		model.LogTypeConsume: "Consommation",
		model.LogTypeManage:  "Gestion",
		model.LogTypeSystem:  "Système",
		model.LogTypeRefund:  "Remboursement",
		model.LogTypeError:   "Erreur",
	},
	DefaultLabel: "Autre",
}

var statementDictRU = statementI18n{
	Header: []string{
		"№", "Время", "Тип", "Событие", "Канал/№ заказа", "Модель", "Токен",
		"Входные токены", "Выходные токены", "Токены кэша", "Квота (со знаком)", "Сумма (со знаком)", "Накопленная квота", "Текущий баланс",
	},
	Meta1:        "Пользователь: %s (id=%d)",
	Meta2:        "Период: %s ~ %s",
	Meta3:        "Текущий баланс (квота): %d quota / %s",
	Footer1Label: "Итого",
	Footer1Key:   "Конечный баланс периода",
	Footer2Label: "Проверка",
	Footer2Key:   "Расхождение сверки (должно быть 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Пополнение",
		model.LogTypeConsume: "Расход",
		model.LogTypeManage:  "Управление",
		model.LogTypeSystem:  "Система",
		model.LogTypeRefund:  "Возврат",
		model.LogTypeError:   "Ошибка",
	},
	DefaultLabel: "Прочее",
}

var statementDictJA = statementI18n{
	Header: []string{
		"No.", "日時", "種別", "イベント", "チャネル/注文番号", "モデル", "トークン",
		"入力 tokens", "出力 tokens", "キャッシュ tokens", "発生額(quota)", "発生額(等価)", "変動後残高(quota)", "残額",
	},
	Meta1:        "ユーザー: %s (id=%d)",
	Meta2:        "対象期間: %s ~ %s",
	Meta3:        "現在の口座残高(Quota): %d quota / %s",
	Footer1Label: "合計",
	Footer1Key:   "期末残高",
	Footer2Label: "照合",
	Footer2Key:   "差異(0になるはず)",
	LogType: map[int]string{
		model.LogTypeTopup:   "チャージ",
		model.LogTypeConsume: "消費",
		model.LogTypeManage:  "管理",
		model.LogTypeSystem:  "システム",
		model.LogTypeRefund:  "返金",
		model.LogTypeError:   "エラー",
	},
	DefaultLabel: "その他",
}

var statementDictVI = statementI18n{
	Header: []string{
		"STT", "Thời gian", "Loại", "Sự kiện", "Kênh/Mã đơn", "Mô hình", "Token",
		"Token đầu vào", "Token đầu ra", "Token bộ nhớ đệm", "Biến động (quota)", "Số tiền (tương đương)", "Số dư quota", "Số dư hiện tại",
	},
	Meta1:        "Người dùng: %s (id=%d)",
	Meta2:        "Kỳ: %s ~ %s",
	Meta3:        "Số dư hiện tại (Quota): %d quota / %s",
	Footer1Label: "Tổng",
	Footer1Key:   "Số dư cuối kỳ",
	Footer2Label: "Đối chiếu",
	Footer2Key:   "Chênh lệch (phải = 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Nạp tiền",
		model.LogTypeConsume: "Tiêu dùng",
		model.LogTypeManage:  "Quản trị",
		model.LogTypeSystem:  "Hệ thống",
		model.LogTypeRefund:  "Hoàn tiền",
		model.LogTypeError:   "Lỗi",
	},
	DefaultLabel: "Khác",
}

var statementDictID = statementI18n{
	Header: []string{
		"No.", "Waktu", "Tipe", "Peristiwa", "Saluran/No. Pesanan", "Model", "Token",
		"Token input", "Token output", "Token cache", "Kuota (bertanda)", "Jumlah (setara)", "Kuota berjalan", "Saldo berjalan",
	},
	Meta1:        "Pengguna: %s (id=%d)",
	Meta2:        "Periode: %s ~ %s",
	Meta3:        "Saldo saat ini (Kuota): %d quota / %s",
	Footer1Label: "Total",
	Footer1Key:   "Saldo akhir periode",
	Footer2Label: "Pemeriksaan",
	Footer2Key:   "Selisih rekonsiliasi (harus 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Isi ulang",
		model.LogTypeConsume: "Konsumsi",
		model.LogTypeManage:  "Kelola",
		model.LogTypeSystem:  "Sistem",
		model.LogTypeRefund:  "Pengembalian",
		model.LogTypeError:   "Kesalahan",
	},
	DefaultLabel: "Lainnya",
}

var statementDictMS = statementI18n{
	Header: []string{
		"No.", "Masa", "Jenis", "Peristiwa", "Saluran/No. Pesanan", "Model", "Token",
		"Token input", "Token output", "Token cache", "Kuota (bertanda)", "Jumlah (setara)", "Kuota berjalan", "Baki berjalan",
	},
	Meta1:        "Pengguna: %s (id=%d)",
	Meta2:        "Tempoh: %s ~ %s",
	Meta3:        "Baki semasa (Kuota): %d quota / %s",
	Footer1Label: "Jumlah",
	Footer1Key:   "Baki akhir tempoh",
	Footer2Label: "Semakan",
	Footer2Key:   "Perbezaan rekonsiliasi (perlu 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Tambah nilai",
		model.LogTypeConsume: "Penggunaan",
		model.LogTypeManage:  "Urus",
		model.LogTypeSystem:  "Sistem",
		model.LogTypeRefund:  "Pemulangan",
		model.LogTypeError:   "Ralat",
	},
	DefaultLabel: "Lain-lain",
}

var statementDictTH = statementI18n{
	Header: []string{
		"ลำดับ", "เวลา", "ประเภท", "เหตุการณ์", "ช่องทาง/เลขคำสั่งซื้อ", "โมเดล", "โทเค็น",
		"โทเค็นเข้า", "โทเค็นออก", "โทเค็นแคช", "โควต้า (เครื่องหมาย)", "จำนวนเงิน (เทียบเท่า)", "โควต้าสะสม", "ยอดคงเหลือ",
	},
	Meta1:        "ผู้ใช้: %s (id=%d)",
	Meta2:        "ช่วงเวลา: %s ~ %s",
	Meta3:        "ยอดคงเหลือปัจจุบัน (โควต้า): %d quota / %s",
	Footer1Label: "รวม",
	Footer1Key:   "ยอดปลายงวด",
	Footer2Label: "ตรวจสอบ",
	Footer2Key:   "ส่วนต่างกระทบยอด (ควรเป็น 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "เติมเงิน",
		model.LogTypeConsume: "ใช้ไป",
		model.LogTypeManage:  "จัดการ",
		model.LogTypeSystem:  "ระบบ",
		model.LogTypeRefund:  "คืนเงิน",
		model.LogTypeError:   "ข้อผิดพลาด",
	},
	DefaultLabel: "อื่นๆ",
}

var statementDictSW = statementI18n{
	Header: []string{
		"Nambari", "Muda", "Aina", "Tukio", "Njia/Nambari ya Agizo", "Mfano", "Tokeni",
		"Tokeni za kuingia", "Tokeni za kutoka", "Tokeni za hifadhi", "Kuota (ishara)", "Kiasi (sawa)", "Kuota inayoendesha", "Salio linaloendesha",
	},
	Meta1:        "Mtumiaji: %s (id=%d)",
	Meta2:        "Kipindi: %s ~ %s",
	Meta3:        "Salio la sasa (Kuota): %d quota / %s",
	Footer1Label: "Jumla",
	Footer1Key:   "Salio la mwisho wa kipindi",
	Footer2Label: "Uthibitisho",
	Footer2Key:   "Tofauti ya ulinganisho (inapaswa kuwa 0)",
	LogType: map[int]string{
		model.LogTypeTopup:   "Weka pesa",
		model.LogTypeConsume: "Tumia",
		model.LogTypeManage:  "Simamia",
		model.LogTypeSystem:  "Mfumo",
		model.LogTypeRefund:  "Rudisha",
		model.LogTypeError:   "Hitilafu",
	},
	DefaultLabel: "Nyingine",
}

// resolveStatementDict 根据 lang 查表，未命中则回退 zh-CN。
// 合法 lang 集合与 web/src/i18n/i18n.js 的 supportedLanguages 对齐。
func resolveStatementDict(lang string) statementI18n {
	var dict statementI18n
	switch lang {
	case "zh-CN":
		dict = statementDictZHCN
	case "zh-TW":
		dict = statementDictZHTW
	case "en":
		dict = statementDictEN
	case "fr":
		dict = statementDictFR
	case "ru":
		dict = statementDictRU
	case "ja":
		dict = statementDictJA
	case "vi":
		dict = statementDictVI
	case "id":
		dict = statementDictID
	case "ms":
		dict = statementDictMS
	case "th":
		dict = statementDictTH
	case "sw":
		dict = statementDictSW
	default:
		dict = statementDictZHCN
	}
	return withStatementDetailsHeader(dict, statementDetailsLabel(lang))
}

func statementDetailsLabel(lang string) string {
	if lang == "en" {
		return "Details"
	}
	return "详情"
}

func withStatementDetailsHeader(dict statementI18n, label string) statementI18n {
	if len(dict.Header) > 0 && dict.Header[len(dict.Header)-1] == label {
		return dict
	}
	header := make([]string, 0, len(dict.Header)+1)
	header = append(header, dict.Header...)
	header = append(header, label)
	dict.Header = header
	return dict
}

// 类型字段的本地化映射（中文，向后兼容保留——具体写入由 statementDict 决定）。
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

// logTypeLabelI18n 按当前语言的字典取类型标签。
func logTypeLabelI18n(t int, dict statementI18n) string {
	if dict.LogType != nil {
		if s, ok := dict.LogType[t]; ok {
			return s
		}
	}
	return dict.DefaultLabel
}

// logExportQuery 对账单/日志导出查询参数（与控制台使用日志筛选对齐）。
type logExportQuery struct {
	StartTs, EndTs                         int64
	ModelName, TokenName, Group, RequestID string
	LogTypes                               []int
	Lang                                   string
}

// adminLogExportQuery 管理员全站日志导出参数，在 logExportQuery 基础上增加 username/channel。
type adminLogExportQuery struct {
	logExportQuery
	Username string
	Channel  int
}

// 解析对账单导出参数；统一做合法性校验（不抛错时返回 0 表示不限）。
// 返回 lang 时做白名单校验，未知 lang 回退到 zh-CN。
func parseLogExportQuery(c *gin.Context) (logExportQuery, error) {
	return parseLogExportQueryWithMaxWindow(c, logExportMaxWindowSeconds, "最多 3 个月")
}

func parseLogExportQueryWithMaxWindow(c *gin.Context, maxWindow int64, rangeLimitLabel string) (logExportQuery, error) {
	var q logExportQuery
	if maxWindow <= 0 {
		maxWindow = logExportMaxWindowSeconds
	}
	if rangeLimitLabel == "" {
		rangeLimitLabel = "最多 3 个月"
	}
	if s := c.Query("start_timestamp"); s != "" {
		v, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || v < 0 {
			return q, fmt.Errorf("start_timestamp 非法")
		}
		q.StartTs = v
	}
	if s := c.Query("end_timestamp"); s != "" {
		v, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || v < 0 {
			return q, fmt.Errorf("end_timestamp 非法")
		}
		q.EndTs = v
	}
	// 默认窗口：不超过 maxWindow。
	now := common.GetTimestamp()
	if q.EndTs == 0 {
		q.EndTs = now
	}
	if q.StartTs == 0 {
		q.StartTs = q.EndTs - maxWindow
	}
	if q.EndTs < q.StartTs {
		return q, fmt.Errorf("end_timestamp 早于 start_timestamp")
	}
	if q.EndTs-q.StartTs > maxWindow {
		return q, fmt.Errorf("时间范围超出限制(%s)", rangeLimitLabel)
	}
	q.ModelName = c.Query("model_name")
	q.TokenName = c.Query("token_name")
	q.Group = c.Query("group")
	q.RequestID = c.Query("request_id")
	q.LogTypes = model.ParseLogTypesQuery(c.Query("type"))
	q.Lang = c.Query("lang")
	switch q.Lang {
	case "zh-CN", "zh-TW", "en", "fr", "ru", "ja", "vi", "id", "ms", "th", "sw":
		// ok
	case "":
		q.Lang = "zh-CN"
	default:
		q.Lang = "zh-CN"
	}
	return q, nil
}

func parseAdminLogExportQuery(c *gin.Context) (adminLogExportQuery, error) {
	base, err := parseLogExportQuery(c)
	if err != nil {
		return adminLogExportQuery{}, err
	}
	q := adminLogExportQuery{logExportQuery: base}
	q.Username = c.Query("username")
	if s := c.Query("channel"); s != "" {
		ch, perr := strconv.Atoi(s)
		if perr != nil || ch < 0 {
			return q, fmt.Errorf("channel 非法")
		}
		q.Channel = ch
	}
	return q, nil
}

func (q adminLogExportQuery) toAdminModelFilter() model.AdminLogExportFilter {
	return model.AdminLogExportFilter{
		LogExportFilter: q.toModelFilter(),
		Username:        q.Username,
		Channel:         q.Channel,
	}
}

func (q logExportQuery) toModelFilter() model.LogExportFilter {
	return model.LogExportFilter{
		FromTs:    q.StartTs,
		ToTs:      q.EndTs,
		ModelName: q.ModelName,
		TokenName: q.TokenName,
		Group:     q.Group,
		RequestID: q.RequestID,
		LogTypes:  q.LogTypes,
	}
}

func (q logExportQuery) hasRowFilterBeyondTime() bool {
	return q.ModelName != "" || q.TokenName != "" || q.Group != "" || q.RequestID != "" || len(q.LogTypes) > 0
}

// 写入一条 CSV 行。csv.Writer 已做 RFC4180 转义（含逗号、引号、换行）。
func writeStatementRow(w *csv.Writer, row []string) error {
	if err := w.Write(row); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func logExportLabel(lang, zh, en string) string {
	if lang == "en" {
		return en
	}
	return zh
}

func logExportString(other map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := other[key]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func logExportNumber(other map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		if v, ok := other[key]; ok && v != nil {
			switch x := v.(type) {
			case float64:
				return x, true
			case float32:
				return float64(x), true
			case int:
				return float64(x), true
			case int64:
				return float64(x), true
			case int32:
				return float64(x), true
			case uint:
				return float64(x), true
			case uint64:
				return float64(x), true
			case uint32:
				return float64(x), true
			case string:
				n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
				if err == nil {
					return n, true
				}
			}
		}
	}
	return 0, false
}

func formatLogDetailForExport(l *model.Log, lang string) string {
	if l == nil {
		return ""
	}
	other, err := common.StrToMap(l.Other)
	if err != nil || other == nil {
		return l.Content
	}

	lines := make([]string, 0, 8)
	addLine := func(label string, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, label+": "+value)
		}
	}
	addNumber := func(label string, keys ...string) {
		if v, ok := logExportNumber(other, keys...); ok {
			addLine(label, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", v), "0"), "."))
		}
	}

	if l.Type == model.LogTypeRefund {
		switch logExportString(other, "billing_phase") {
		case model.BillingPhaseDeltaRefund:
			lines = append(lines, logExportLabel(lang, "异步任务差额退款", "Async task delta refund"))
			if v, ok := logExportNumber(other, "pre_consumed_quota"); ok && v > 0 {
				addLine(logExportLabel(lang, "预扣费", "Pre-charged"), formatQuotaDisplay(int64(v)))
			}
			if v, ok := logExportNumber(other, "actual_quota"); ok && v > 0 {
				addLine(logExportLabel(lang, "实际扣费", "Actual charge"), formatQuotaDisplay(int64(v)))
			}
			if v, ok := logExportNumber(other, "display_quota", "balance_delta"); ok {
				addLine(logExportLabel(lang, "返还差额", "Refunded delta"), formatQuotaDisplay(int64(v)))
			}
		case model.BillingPhaseRefund:
			lines = append(lines, logExportLabel(lang, "异步任务失败退款", "Async task failure refund"))
		default:
			lines = append(lines, logExportLabel(lang, "异步任务退款", "Async task refund"))
		}
		return strings.Join(lines, "\n")
	}

	if l.Type != model.LogTypeConsume {
		return l.Content
	}

	if v, ok := other["violation_fee"].(bool); ok && v || logExportString(other, "violation_fee_code", "violation_fee_marker") != "" {
		lines = append(lines, logExportLabel(lang, "违规扣费", "Violation fee"))
		if v, ok := logExportNumber(other, "fee_quota"); ok {
			addLine(logExportLabel(lang, "扣费", "Fee"), formatQuotaDisplay(int64(v)))
		}
		if l.Content != "" {
			addLine(logExportLabel(lang, "详情", "Details"), l.Content)
		}
		return strings.Join(lines, "\n")
	}

	billingMode := logExportString(other, "billing_mode")
	modelPrice, hasModelPrice := logExportNumber(other, "model_price")
	switch {
	case billingMode == "video_per_second" && (!hasModelPrice || modelPrice == 0 || modelPrice == -1):
		lines = append(lines, logExportLabel(lang, "分辨率阶梯计费", "Resolution tier billing"))
	case billingMode == "video_token_output" && (!hasModelPrice || modelPrice == 0 || modelPrice == -1):
		lines = append(lines, logExportLabel(lang, "视频按 token 计费", "Video token billing"))
	case billingMode == "video_per_video" && (!hasModelPrice || modelPrice == 0 || modelPrice == -1):
		lines = append(lines, logExportLabel(lang, "按视频数量计费", "Per-video billing"))
	case billingMode == "image_per_image":
		lines = append(lines, logExportLabel(lang, "按图片数量计费", "Per-image billing"))
	case logExportString(other, "request_tier_pricing") == "true" || logExportString(other, "request_tier_pricing") == "1":
		lines = append(lines, logExportLabel(lang, "阶梯计费", "Tier billing"))
		if display := logExportString(other, "request_tier_display"); display != "" {
			lines = append(lines, display)
		}
	default:
		if hasModelPrice && modelPrice != 0 && modelPrice != -1 {
			lines = append(lines, logExportLabel(lang, "按次", "Per request"))
			addNumber(logExportLabel(lang, "模型价格", "Model price"), "model_price")
		} else if _, ok := logExportNumber(other, "model_ratio"); ok {
			addNumber(logExportLabel(lang, "模型", "Model"), "model_ratio")
		}
	}

	addLine(logExportLabel(lang, "分辨率", "Resolution"), logExportString(other, "video_resolution"))
	addNumber(logExportLabel(lang, "缓存", "Cache"), "cache_ratio")
	addNumber(logExportLabel(lang, "缓存创建", "Cache creation"), "cache_creation_ratio")
	addNumber(logExportLabel(lang, "5m缓存创建", "5m cache creation"), "cache_creation_ratio_5m")
	addNumber(logExportLabel(lang, "1h缓存创建", "1h cache creation"), "cache_creation_ratio_1h")
	if v, ok := other["is_system_prompt_overwritten"].(bool); ok && v {
		lines = append(lines, logExportLabel(lang, "系统提示覆盖", "System prompt override"))
	}

	if len(lines) == 0 {
		return l.Content
	}
	return strings.Join(lines, "\n")
}

const logExportXLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type usageLogWorkbook struct {
	file        *excelize.File
	stream      *excelize.StreamWriter
	headerStyle int
	metaStyle   int
	amountStyle int
	wrapStyle   int
}

func usageLogAmountNumberFormat() string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return "#,##0"
	}
	symbol := strings.ReplaceAll(operation_setting.GetCurrencySymbol(), `"`, `""`)
	literal := `"` + symbol + `"`
	return literal + "#,##0.000000;[Red]-" + literal + "#,##0.000000"
}

func newUsageLogWorkbook(sheet string, widths []float64, headerRow int) (*usageLogWorkbook, error) {
	f := excelize.NewFile()
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		_ = f.Close()
		return nil, err
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	metaStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	amountFormat := usageLogAmountNumberFormat()
	amountStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &amountFormat,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	wrapStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	stream, err := f.NewStreamWriter(sheet)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	for idx, width := range widths {
		if err := stream.SetColWidth(idx+1, idx+1, width); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	if err := stream.SetPanes(&excelize.Panes{
		Freeze:      true,
		YSplit:      headerRow,
		TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
		ActivePane:  "bottomLeft",
	}); err != nil {
		_ = f.Close()
		return nil, err
	}

	return &usageLogWorkbook{
		file:        f,
		stream:      stream,
		headerStyle: headerStyle,
		metaStyle:   metaStyle,
		amountStyle: amountStyle,
		wrapStyle:   wrapStyle,
	}, nil
}

func styledLogExportRow(values []string, styleID int) []interface{} {
	row := make([]interface{}, len(values))
	for idx, value := range values {
		row[idx] = excelize.Cell{StyleID: styleID, Value: value}
	}
	return row
}

func finishUsageLogWorkbook(workbook *usageLogWorkbook) (*excelize.File, error) {
	if err := workbook.stream.Flush(); err != nil {
		_ = workbook.file.Close()
		return nil, err
	}
	return workbook.file, nil
}

func writeUsageLogWorkbook(c *gin.Context, workbook *excelize.File, filename string) {
	defer workbook.Close()
	c.Header("Content-Type", logExportXLSXContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(200)
	if err := workbook.Write(c.Writer); err != nil {
		common.SysError("write usage log workbook: " + err.Error())
	}
}

// 以 XLSX 写出单个用户的对账单，金额列使用数值单元格并固定显示 6 位小数。
func streamUserStatementXLSX(c *gin.Context, user *model.User, query logExportQuery, filename string, dict statementI18n) {
	if user == nil {
		common.ApiError(c, fmt.Errorf("用户不存在"))
		return
	}
	logs, total, err := model.GetUserLogsForExport(user.Id, query.toModelFilter())
	if err != nil {
		// 行数超限
		common.ApiError(c, err)
		return
	}
	_ = total

	// 1) 反推"窗口期初余额"：当前余额 - 窗口内净变动。
	// 注意：净变动只取真实影响 User.Quota 的三类日志（Consume/Topup/Refund）。
	// 若存在除时间外的筛选条件，则余额列按导出子集累计，不再做全量对账。
	running := int64(0)
	if !query.hasRowFilterBeyondTime() {
		delta, derr := model.GetChargeableDeltaByUser(user.Id, query.StartTs, query.EndTs)
		if derr != nil {
			common.ApiError(c, derr)
			return
		}
		running = int64(user.Quota) - delta
	}

	workbook, err := newUsageLogWorkbook(
		"Statement",
		[]float64{8, 20, 12, 36, 20, 24, 18, 14, 14, 14, 18, 18, 20, 20, 48},
		4,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	fail := func(err error) {
		_ = workbook.file.Close()
		common.ApiError(c, err)
	}

	// 顶部元信息行（作为备注，前 3 行；便于用户理解对账口径）。
	emptyRow := make([]string, len(dict.Header))
	meta1 := fmt.Sprintf(dict.Meta1, user.Username, user.Id)
	if err := workbook.stream.SetRow("A1", styledLogExportRow(append([]string{meta1}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	periodStart := time.Unix(query.StartTs, 0).Format("2006-01-02 15:04:05")
	periodEnd := time.Unix(query.EndTs, 0).Format("2006-01-02 15:04:05")
	meta2 := fmt.Sprintf(dict.Meta2, periodStart, periodEnd)
	if err := workbook.stream.SetRow("A2", styledLogExportRow(append([]string{meta2}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	// 期末余额以"quota + 等值金额"双口径展示，避免系统内部单位与展示货币混淆。
	meta3 := fmt.Sprintf(dict.Meta3, user.Quota, formatBalanceAmount(user.Quota))
	if err := workbook.stream.SetRow("A3", styledLogExportRow(append([]string{meta3}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	if err := workbook.stream.SetRow("A4", styledLogExportRow(dict.Header, workbook.headerStyle), excelize.RowOpts{Height: 30}); err != nil {
		fail(err)
		return
	}

	// 2) 逐行输出。
	balances := make([]int64, len(logs))
	finalRunning := running
	for idx, l := range logs {
		finalRunning += model.SignedLogDelta(l.Quota, l.Type)
		balances[idx] = finalRunning
	}
	for idx := len(logs) - 1; idx >= 0; idx-- {
		l := logs[idx]
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		signed := model.SignedLogDelta(l.Quota, l.Type)

		channelOrOrder := strconv.Itoa(l.ChannelId)
		if l.Type == model.LogTypeTopup || l.Type == model.LogTypeRefund {
			// 充值/退款行用 Other 字段里可能保存的 trade_no 替代渠道号（如果有）。
			if order, ok := extractOrderNo(l.Other); ok {
				channelOrOrder = order
			}
		}

		cacheTokens := extractCacheReadTokens(l.Other)
		other, _ := common.StrToMap(l.Other)
		displaySigned := resolveUsageLogExportSignedQuota(l, other)

		row := []interface{}{
			len(logs) - idx,
			ts,
			logTypeLabelI18n(l.Type, dict),
			excelize.Cell{StyleID: workbook.wrapStyle, Value: l.Content},
			channelOrOrder,
			l.ModelName,
			l.TokenName,
			l.PromptTokens,
			l.CompletionTokens,
			cacheTokens,
			signed,
			excelize.Cell{StyleID: workbook.amountStyle, Value: usageLogExportAmountValue(displaySigned, other)},
			balances[idx],
			formatBalanceAmount(int(balances[idx])),
			excelize.Cell{StyleID: workbook.wrapStyle, Value: formatLogDetailForExport(l, query.Lang)},
		}
		cell := fmt.Sprintf("A%d", 5+len(logs)-1-idx)
		if err := workbook.stream.SetRow(cell, row); err != nil {
			fail(err)
			return
		}
	}
	running = finalRunning

	// 3) 末尾对账校验行：仅全量时间窗口导出时输出，筛选导出跳过以免误导。
	if !query.hasRowFilterBeyondTime() {
		footer1 := append([]string{dict.Footer1Label}, emptyRow[1:len(emptyRow)-2]...)
		footer1 = append(footer1, dict.Footer1Key, fmt.Sprintf("%d / %s", user.Quota, formatBalanceAmount(user.Quota)))
		footerRow := 5 + len(logs)
		if err := workbook.stream.SetRow(fmt.Sprintf("A%d", footerRow), styledLogExportRow(footer1, workbook.metaStyle)); err != nil {
			fail(err)
			return
		}
		footer2 := append([]string{dict.Footer2Label}, emptyRow[1:len(emptyRow)-2]...)
		footer2 = append(footer2, dict.Footer2Key, fmt.Sprintf("%d quota (%s)", running-int64(user.Quota), formatBalanceAmount(int(running-int64(user.Quota)))))
		if err := workbook.stream.SetRow(fmt.Sprintf("A%d", footerRow+1), styledLogExportRow(footer2, workbook.metaStyle)); err != nil {
			fail(err)
			return
		}
	}

	f, err := finishUsageLogWorkbook(workbook)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("X-Statement-Current-Balance", strconv.Itoa(user.Quota))
	c.Header("X-Statement-Window-Start", strconv.FormatInt(query.StartTs, 10))
	c.Header("X-Statement-Window-End", strconv.FormatInt(query.EndTs, 10))
	c.Header("X-Statement-Lang", dictHeaderLangTag(dict))
	writeUsageLogWorkbook(c, f, filename)
}

// dictHeaderLangTag 仅用于在 X-Statement-Lang 响应头里标注本次输出用了哪个 lang 字典（便于前端排查）。
func dictHeaderLangTag(dict statementI18n) string {
	if len(dict.Header) == 0 {
		return "zh-CN"
	}
	switch dict.Header[0] {
	case "序號":
		return "zh-TW"
	case "No.":
		return "en"
	case "N°":
		return "fr"
	case "№":
		return "ru"
	case "STT":
		return "vi"
	case "Nambari":
		return "sw"
	case "ลำดับ":
		return "th"
	default:
		return "zh-CN"
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
//     （前端 web/src/helpers/render.jsx 与所有日志列定义都在读这个键）
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

// resolveUsageLogExportQuota 与 /console/log 花费列保持一致：结算后的异步任务优先展示实际扣费。
func resolveUsageLogExportQuota(l *model.Log, other map[string]interface{}) int64 {
	if l == nil {
		return 0
	}
	if v, ok := logExportNumber(other, "video_final_quota", "actual_quota"); ok && v > 0 {
		return int64(v)
	}
	if v, ok := logExportNumber(other, "video_billed_quota"); ok && v > 0 {
		return int64(v)
	}
	return int64(l.Quota)
}

func isVideoUsageLog(other map[string]interface{}) bool {
	switch logExportString(other, "billing_mode") {
	case "video_per_second", "video_token_output", "video_per_video":
		return true
	}
	if v, ok := logExportNumber(other, "video_billed_quota"); ok && v != 0 {
		return true
	}
	return strings.Contains(logExportString(other, "request_path"), "/videos")
}

// usageLogExportAmountValue 复用 /console/log 花费列的 6 位进一法与视频额度单位换算。
func usageLogExportAmountValue(quota int64, other map[string]interface{}) float64 {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return float64(quota)
	}

	quotaPerUnit := common.QuotaPerUnit
	if isVideoUsageLog(other) {
		if v, ok := logExportNumber(other, "video_quota_per_unit"); ok && v > 0 {
			quotaPerUnit = v
		}
	}
	if quotaPerUnit <= 0 {
		quotaPerUnit = 1
	}

	rate := 1.0
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		rate = operation_setting.USDExchangeRate
	case operation_setting.QuotaDisplayTypeCustom:
		rate = operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
		if rate <= 0 {
			rate = 1
		}
	}

	raw := float64(quota) / quotaPerUnit * rate
	amount := logger.CeilToFixedDecimals(raw, 6)
	if amount == 0 && quota > 0 && raw > 0 {
		amount = 0.000001
	}
	return amount
}

// formatUsageLogExportAmount 保留给批量 ZIP 内的旧 CSV 文件使用。
func formatUsageLogExportAmount(quota int64, other map[string]interface{}) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return strconv.FormatInt(quota, 10)
	}
	amount := usageLogExportAmountValue(quota, other)
	return fmt.Sprintf("%s%.6f\t", operation_setting.GetCurrencySymbol(), amount)
}

func resolveUsageLogExportSignedQuota(l *model.Log, other map[string]interface{}) int64 {
	if l == nil {
		return 0
	}
	displayQuota := resolveUsageLogExportQuota(l, other)
	return model.SignedLogDelta(int(displayQuota), l.Type)
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
	query, err := parseLogExportQuery(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	dict := resolveStatementDict(query.Lang)
	filename := fmt.Sprintf("statement-%s-%d.xlsx", sanitizeFilename(user.Username), time.Now().Unix())
	streamUserStatementXLSX(c, user, query, filename, dict)
}

// GET /api/log/export 管理员导出全站日志（筛选口径与 /api/log/ 列表一致）
func ExportAdminLogs(c *gin.Context) {
	query, err := parseAdminLogExportQuery(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	logs, _, err := model.GetAllLogsForExport(query.toAdminModelFilter())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	dict := resolveAdminLogExportDict(query.Lang)
	filename := fmt.Sprintf("admin-logs-%d.xlsx", time.Now().Unix())
	streamAdminLogsXLSX(c, logs, query.logExportQuery, filename, dict)
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
	query, err := parseLogExportQuery(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	dict := resolveStatementDict(query.Lang)
	filename := fmt.Sprintf("statement-%s-%d-admin.xlsx", sanitizeFilename(user.Username), time.Now().Unix())
	streamUserStatementXLSX(c, user, query, filename, dict)
}

// POST /api/admin/log/export_all 管理员全平台批量对账单：返回 zip 包，
// 每个用户一个 CSV（命名 <username>-<id>.csv）。为避免内存爆炸，
// 当用户数 > 200 时返回 400，要求改用单用户导出。
func ExportAllUsersLogsAdmin(c *gin.Context) {
	if !model.IsAdmin(c.GetInt("role")) {
		c.JSON(403, gin.H{"success": false, "message": "需要管理员权限"})
		return
	}
	query, err := parseLogExportQuery(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	dict := resolveStatementDict(query.Lang)
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
		writeSingleUserCSV(fw, u, query, dict)
	}
}

// writeSingleUserCSV 保留给管理员批量 ZIP 导出，目标 io.Writer 由调用方控制。
func writeSingleUserCSV(w interface {
	Write(p []byte) (int, error)
}, user *model.User, query logExportQuery, dict statementI18n) {
	logs, _, err := model.GetUserLogsForExport(user.Id, query.toModelFilter())
	if err != nil {
		return
	}
	running := int64(0)
	if !query.hasRowFilterBeyondTime() {
		delta, _ := model.GetChargeableDeltaByUser(user.Id, query.StartTs, query.EndTs)
		running = int64(user.Quota) - delta
	}

	// BOM
	w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	emptyRow := make([]string, len(dict.Header))
	cw.Write(append([]string{fmt.Sprintf(dict.Meta1, user.Username, user.Id)}, emptyRow[1:]...))
	cw.Write(append([]string{fmt.Sprintf(dict.Meta2,
		time.Unix(query.StartTs, 0).Format("2006-01-02 15:04:05"),
		time.Unix(query.EndTs, 0).Format("2006-01-02 15:04:05"),
	)}, emptyRow[1:]...))
	cw.Write(append([]string{
		fmt.Sprintf(dict.Meta3, user.Quota, formatBalanceAmount(user.Quota)),
	}, emptyRow[1:]...))
	cw.Write(dict.Header)
	balances := make([]int64, len(logs))
	finalRunning := running
	for idx, l := range logs {
		finalRunning += model.SignedLogDelta(l.Quota, l.Type)
		balances[idx] = finalRunning
	}
	for idx := len(logs) - 1; idx >= 0; idx-- {
		l := logs[idx]
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		signed := model.SignedLogDelta(l.Quota, l.Type)
		channelOrOrder := strconv.Itoa(l.ChannelId)
		if l.Type == model.LogTypeTopup || l.Type == model.LogTypeRefund {
			if order, ok := extractOrderNo(l.Other); ok {
				channelOrOrder = order
			}
		}
		cacheTokens := extractCacheReadTokens(l.Other)
		other, _ := common.StrToMap(l.Other)
		displaySigned := resolveUsageLogExportSignedQuota(l, other)
		cw.Write([]string{
			strconv.Itoa(len(logs) - idx), ts, logTypeLabelI18n(l.Type, dict), l.Content, channelOrOrder,
			l.ModelName, l.TokenName, strconv.Itoa(l.PromptTokens), strconv.Itoa(l.CompletionTokens),
			strconv.Itoa(cacheTokens),
			strconv.FormatInt(signed, 10), formatUsageLogExportAmount(displaySigned, other), strconv.FormatInt(balances[idx], 10),
			formatBalanceAmount(int(balances[idx])), formatLogDetailForExport(l, query.Lang),
		})
	}
	running = finalRunning
	if !query.hasRowFilterBeyondTime() {
		footer1 := append([]string{dict.Footer1Label}, emptyRow[1:len(emptyRow)-2]...)
		footer1 = append(footer1, dict.Footer1Key, fmt.Sprintf("%d / %s", user.Quota, formatBalanceAmount(user.Quota)))
		cw.Write(footer1)
		footer2 := append([]string{dict.Footer2Label}, emptyRow[1:len(emptyRow)-2]...)
		footer2 = append(footer2, dict.Footer2Key, fmt.Sprintf("%d quota (%s)", running-int64(user.Quota), formatBalanceAmount(int(running-int64(user.Quota)))))
		cw.Write(footer2)
	}
	cw.Flush()
}

// supplierChannelLogExportI18n 供应商渠道日志导出表头。
type supplierChannelLogExportI18n struct {
	Header []string
	Meta1  string // "供应商渠道日志"
	Meta2  string // "账期: %s ~ %s"
}

// adminLogExportI18n 管理员全站日志导出表头。
type adminLogExportI18n struct {
	Header []string
	Meta1  string // "全站使用日志"
	Meta2  string // "账期: %s ~ %s"
}

var adminLogExportDictZHCN = adminLogExportI18n{
	Header: []string{
		"序号", "时间", "类型", "用户", "令牌", "模型", "渠道", "分组", "Request ID", "事件",
		"输入 tokens", "输出 tokens", "缓存 tokens", "消耗额度(quota)", "消耗额度(等值)",
	},
	Meta1: "全站使用日志",
	Meta2: "账期: %s ~ %s",
}

var adminLogExportDictEN = adminLogExportI18n{
	Header: []string{
		"No.", "Time", "Type", "User", "Token", "Model", "Channel", "Group", "Request ID", "Event",
		"Input tokens", "Output tokens", "Cache tokens", "Quota", "Amount",
	},
	Meta1: "Platform usage logs",
	Meta2: "Period: %s ~ %s",
}

func resolveAdminLogExportDict(lang string) adminLogExportI18n {
	var dict adminLogExportI18n
	if lang == "en" {
		dict = adminLogExportDictEN
	} else {
		dict = adminLogExportDictZHCN
	}
	if len(dict.Header) == 0 || dict.Header[len(dict.Header)-1] != statementDetailsLabel(lang) {
		dict.Header = append(append([]string{}, dict.Header...), statementDetailsLabel(lang))
	}
	return dict
}

// streamAdminLogsXLSX 以日志页面顺序写出管理员全站日志。
func streamAdminLogsXLSX(c *gin.Context, logs []*model.Log, query logExportQuery, filename string, dict adminLogExportI18n) {
	workbook, err := newUsageLogWorkbook(
		"Usage Logs",
		[]float64{8, 20, 12, 18, 18, 24, 18, 14, 38, 36, 14, 14, 14, 18, 18, 48},
		3,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	fail := func(err error) {
		_ = workbook.file.Close()
		common.ApiError(c, err)
	}

	emptyRow := make([]string, len(dict.Header))
	if err := workbook.stream.SetRow("A1", styledLogExportRow(append([]string{dict.Meta1}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	periodStart := time.Unix(query.StartTs, 0).Format("2006-01-02 15:04:05")
	periodEnd := time.Unix(query.EndTs, 0).Format("2006-01-02 15:04:05")
	meta2 := fmt.Sprintf(dict.Meta2, periodStart, periodEnd)
	if err := workbook.stream.SetRow("A2", styledLogExportRow(append([]string{meta2}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	if err := workbook.stream.SetRow("A3", styledLogExportRow(dict.Header, workbook.headerStyle), excelize.RowOpts{Height: 30}); err != nil {
		fail(err)
		return
	}

	statementDict := resolveStatementDict(query.Lang)
	for idx, l := range logs {
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		channelDisplay := l.ChannelDisplay
		if channelDisplay == "" {
			channelDisplay = strconv.Itoa(l.ChannelId)
		}
		cacheTokens := extractCacheReadTokens(l.Other)
		other, _ := common.StrToMap(l.Other)
		quota := resolveUsageLogExportQuota(l, other)
		row := []interface{}{
			idx + 1,
			ts,
			logTypeLabelI18n(l.Type, statementDict),
			l.Username,
			l.TokenName,
			l.ModelName,
			channelDisplay,
			l.Group,
			l.RequestId,
			excelize.Cell{StyleID: workbook.wrapStyle, Value: l.Content},
			l.PromptTokens,
			l.CompletionTokens,
			cacheTokens,
			quota,
			excelize.Cell{StyleID: workbook.amountStyle, Value: usageLogExportAmountValue(quota, other)},
			excelize.Cell{StyleID: workbook.wrapStyle, Value: formatLogDetailForExport(l, query.Lang)},
		}
		if err := workbook.stream.SetRow(fmt.Sprintf("A%d", idx+4), row); err != nil {
			fail(err)
			return
		}
	}

	f, err := finishUsageLogWorkbook(workbook)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("X-Statement-Window-Start", strconv.FormatInt(query.StartTs, 10))
	c.Header("X-Statement-Window-End", strconv.FormatInt(query.EndTs, 10))
	writeUsageLogWorkbook(c, f, filename)
}

var supplierChannelLogExportDictZHCN = supplierChannelLogExportI18n{
	Header: []string{
		"序号", "时间", "类型", "用户", "令牌", "模型", "渠道", "分组", "Request ID",
		"输入 tokens", "输出 tokens", "缓存 tokens", "消耗额度(quota)", "消耗额度(等值)",
	},
	Meta1: "供应商渠道日志",
	Meta2: "账期: %s ~ %s",
}

var supplierChannelLogExportDictEN = supplierChannelLogExportI18n{
	Header: []string{
		"No.", "Time", "Type", "User", "Token", "Model", "Channel", "Group", "Request ID",
		"Input tokens", "Output tokens", "Cache tokens", "Quota", "Amount",
	},
	Meta1: "Supplier channel logs",
	Meta2: "Period: %s ~ %s",
}

func resolveSupplierChannelLogExportDict(lang string) supplierChannelLogExportI18n {
	if lang == "en" {
		return supplierChannelLogExportDictEN
	}
	return supplierChannelLogExportDictZHCN
}

// streamSupplierChannelLogsXLSX 以日志页面顺序写出供应商渠道日志。
func streamSupplierChannelLogsXLSX(c *gin.Context, logs []*model.Log, query logExportQuery, filename string, dict supplierChannelLogExportI18n) {
	workbook, err := newUsageLogWorkbook(
		"Usage Logs",
		[]float64{8, 20, 12, 18, 18, 24, 18, 14, 38, 14, 14, 14, 18, 18},
		3,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	fail := func(err error) {
		_ = workbook.file.Close()
		common.ApiError(c, err)
	}

	emptyRow := make([]string, len(dict.Header))
	if err := workbook.stream.SetRow("A1", styledLogExportRow(append([]string{dict.Meta1}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	periodStart := time.Unix(query.StartTs, 0).Format("2006-01-02 15:04:05")
	periodEnd := time.Unix(query.EndTs, 0).Format("2006-01-02 15:04:05")
	meta2 := fmt.Sprintf(dict.Meta2, periodStart, periodEnd)
	if err := workbook.stream.SetRow("A2", styledLogExportRow(append([]string{meta2}, emptyRow[1:]...), workbook.metaStyle)); err != nil {
		fail(err)
		return
	}
	if err := workbook.stream.SetRow("A3", styledLogExportRow(dict.Header, workbook.headerStyle), excelize.RowOpts{Height: 30}); err != nil {
		fail(err)
		return
	}

	statementDict := resolveStatementDict(query.Lang)
	for idx, l := range logs {
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		channelDisplay := l.ChannelDisplay
		if channelDisplay == "" {
			channelDisplay = strconv.Itoa(l.ChannelId)
		}
		cacheTokens := extractCacheReadTokens(l.Other)
		other, _ := common.StrToMap(l.Other)
		quota := resolveUsageLogExportQuota(l, other)
		row := []interface{}{
			idx + 1,
			ts,
			logTypeLabelI18n(l.Type, statementDict),
			l.Username,
			l.TokenName,
			l.ModelName,
			channelDisplay,
			l.Group,
			l.RequestId,
			l.PromptTokens,
			l.CompletionTokens,
			cacheTokens,
			quota,
			excelize.Cell{StyleID: workbook.amountStyle, Value: usageLogExportAmountValue(quota, other)},
		}
		if err := workbook.stream.SetRow(fmt.Sprintf("A%d", idx+4), row); err != nil {
			fail(err)
			return
		}
	}

	f, err := finishUsageLogWorkbook(workbook)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("X-Statement-Window-Start", strconv.FormatInt(query.StartTs, 10))
	c.Header("X-Statement-Window-End", strconv.FormatInt(query.EndTs, 10))
	writeUsageLogWorkbook(c, f, filename)
}

// supplierDashboardExportI18n 供应商看板使用详情账单导出文案。
type supplierDashboardExportI18n struct {
	Meta1          string   // "供应商使用详情账单"
	Meta2Supplier  string   // "供应商: %s"
	Meta3Period    string   // "账期: %s ~ %s"
	SummarySection string   // "【模型汇总】"
	SummaryHeader  []string // 模型汇总表头
	SummaryTotal   string   // "合计"
	DetailSection  string   // "【使用明细】"
	DetailHeader   []string // 明细表头（与渠道日志导出一致）
}

var supplierDashboardExportDictZHCN = supplierDashboardExportI18n{
	Meta1:          "供应商使用详情账单",
	Meta2Supplier:  "供应商: %s",
	Meta3Period:    "账期: %s ~ %s",
	SummarySection: "【模型汇总】",
	SummaryHeader: []string{
		"模型", "请求次数", "输入 tokens", "输出 tokens", "总 tokens", "消耗额度(quota)", "消耗额度(等值)",
	},
	SummaryTotal:  "合计",
	DetailSection: "【使用明细】",
	DetailHeader: []string{
		"序号", "时间", "类型", "用户", "令牌", "模型", "渠道", "分组", "Request ID",
		"输入 tokens", "输出 tokens", "缓存 tokens", "消耗额度(quota)", "消耗额度(等值)",
		"成本折扣", "经营成本", "加价折扣", "销售折扣",
		"官方总价", "成本价", "经营单价", "销售单价",
	},
}

var supplierDashboardExportDictEN = supplierDashboardExportI18n{
	Meta1:          "Supplier usage detail statement",
	Meta2Supplier:  "Supplier: %s",
	Meta3Period:    "Period: %s ~ %s",
	SummarySection: "[Model summary]",
	SummaryHeader: []string{
		"Model", "Requests", "Input tokens", "Output tokens", "Total tokens", "Quota", "Amount",
	},
	SummaryTotal:  "Total",
	DetailSection: "[Usage details]",
	DetailHeader: []string{
		"No.", "Time", "Type", "User", "Token", "Model", "Channel", "Group", "Request ID",
		"Input tokens", "Output tokens", "Cache tokens", "Quota", "Amount",
		"Cost discount", "Operating cost", "Markup discount", "Sales discount",
		"Official total", "Cost price", "Operating price", "Sales price",
	},
}

func resolveSupplierDashboardExportDict(lang string) supplierDashboardExportI18n {
	if lang == "en" {
		return supplierDashboardExportDictEN
	}
	return supplierDashboardExportDictZHCN
}

// streamSupplierDashboardUsageCSV 流式写出供应商看板使用详情账单（模型汇总 + 逐条明细）。
func streamSupplierDashboardUsageCSV(
	c *gin.Context,
	modelRows []model.SupplierUsageByModelDetail,
	logs []*model.Log,
	query supplierDashboardExportQuery,
	filename string,
	dict supplierDashboardExportI18n,
) {
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

	summaryColCount := len(dict.SummaryHeader)
	detailColCount := len(dict.DetailHeader)
	if detailColCount > summaryColCount {
		summaryColCount = detailColCount
	}
	emptyRow := make([]string, summaryColCount)

	if err := writeStatementRow(w, append([]string{dict.Meta1}, emptyRow[1:]...)); err != nil {
		common.SysError("write supplier dashboard export meta1: " + err.Error())
		return
	}
	if query.SupplierName != "" {
		meta2 := fmt.Sprintf(dict.Meta2Supplier, query.SupplierName)
		if err := writeStatementRow(w, append([]string{meta2}, emptyRow[1:]...)); err != nil {
			common.SysError("write supplier dashboard export meta2: " + err.Error())
			return
		}
	}
	metaPeriod := fmt.Sprintf(dict.Meta3Period, periodStart, periodEnd)
	if err := writeStatementRow(w, append([]string{metaPeriod}, emptyRow[1:]...)); err != nil {
		common.SysError("write supplier dashboard export period: " + err.Error())
		return
	}
	if err := writeStatementRow(w, emptyRow); err != nil {
		common.SysError("write supplier dashboard export blank: " + err.Error())
		return
	}

	if err := writeStatementRow(w, append([]string{dict.SummarySection}, emptyRow[1:]...)); err != nil {
		common.SysError("write supplier dashboard export summary title: " + err.Error())
		return
	}
	if err := writeStatementRow(w, padRowToWidth(dict.SummaryHeader, summaryColCount)); err != nil {
		common.SysError("write supplier dashboard export summary header: " + err.Error())
		return
	}

	totalRequests := 0
	totalPrompt := 0
	totalCompletion := 0
	totalTokens := 0
	totalQuota := int64(0)
	for _, row := range modelRows {
		totalRequests += row.Count
		totalPrompt += row.PromptTokens
		totalCompletion += row.CompletionTokens
		totalTokens += row.TokenUsed
		totalQuota += int64(row.Quota)
		summaryRow := []string{
			row.ModelName,
			strconv.Itoa(row.Count),
			strconv.Itoa(row.PromptTokens),
			strconv.Itoa(row.CompletionTokens),
			strconv.Itoa(row.TokenUsed),
			strconv.FormatInt(int64(row.Quota), 10),
			formatQuotaDisplay(int64(row.Quota)),
		}
		if err := writeStatementRow(w, padRowToWidth(summaryRow, summaryColCount)); err != nil {
			common.SysError("write supplier dashboard export summary row: " + err.Error())
			return
		}
	}
	totalRow := []string{
		dict.SummaryTotal,
		strconv.Itoa(totalRequests),
		strconv.Itoa(totalPrompt),
		strconv.Itoa(totalCompletion),
		strconv.Itoa(totalTokens),
		strconv.FormatInt(totalQuota, 10),
		formatQuotaDisplay(totalQuota),
	}
	if err := writeStatementRow(w, padRowToWidth(totalRow, summaryColCount)); err != nil {
		common.SysError("write supplier dashboard export summary total: " + err.Error())
		return
	}

	if err := writeStatementRow(w, emptyRow); err != nil {
		common.SysError("write supplier dashboard export blank2: " + err.Error())
		return
	}
	if err := writeStatementRow(w, append([]string{dict.DetailSection}, emptyRow[1:]...)); err != nil {
		common.SysError("write supplier dashboard export detail title: " + err.Error())
		return
	}
	if err := writeStatementRow(w, padRowToWidth(appendSettlementPriceCurrencyToHeaders(dict.DetailHeader), summaryColCount)); err != nil {
		common.SysError("write supplier dashboard export detail header: " + err.Error())
		return
	}

	statementDict := resolveStatementDict(query.Lang)
	for idx, l := range logs {
		ts := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05")
		channelDisplay := l.ChannelDisplay
		if channelDisplay == "" {
			channelDisplay = strconv.Itoa(l.ChannelId)
		}
		cacheTokens := extractCacheReadTokens(l.Other)
		quota := int64(l.Quota)
		detailRow := []string{
			strconv.Itoa(idx + 1),
			ts,
			logTypeLabelI18n(l.Type, statementDict),
			l.Username,
			l.TokenName,
			l.ModelName,
			channelDisplay,
			l.Group,
			l.RequestId,
			strconv.Itoa(l.PromptTokens),
			strconv.Itoa(l.CompletionTokens),
			strconv.Itoa(cacheTokens),
			strconv.FormatInt(quota, 10),
			formatQuotaDisplay(quota),
		}
		detailRow = append(detailRow, buildSettlementDetailSuffix(l)...)
		if err := writeStatementRow(w, padRowToWidth(detailRow, summaryColCount)); err != nil {
			common.SysError("write supplier dashboard export detail row: " + err.Error())
			return
		}
	}
}

// padRowToWidth 将 CSV 行补齐到指定列宽，便于分段表头列数不一致时对齐。
func padRowToWidth(row []string, width int) []string {
	if len(row) >= width {
		return row
	}
	out := make([]string, width)
	copy(out, row)
	return out
}

func appendSettlementPriceCurrencyToHeaders(headers []string) []string {
	if len(headers) < 4 {
		return headers
	}
	currency := model.SettlementCurrencyLabel()
	out := make([]string, len(headers))
	copy(out, headers)
	priceStart := len(headers) - 4
	for i := priceStart; i < len(headers); i++ {
		out[i] = fmt.Sprintf("%s(%s)", headers[i], currency)
	}
	return out
}

func buildSettlementDetailSuffix(l *model.Log) []string {
	if l == nil {
		return make([]string, 8)
	}
	otherMap, _ := common.StrToMap(l.Other)
	cacheTokens := extractCacheReadTokens(l.Other)
	bd := model.ComputeSettlementPriceBreakdown(l.PromptTokens, l.CompletionTokens, cacheTokens, l.Quota, otherMap)
	d := bd.Discounts
	return []string{
		model.FormatSettlementPercent(d.PriceDiscountPercent),
		model.FormatSettlementPercent(d.OperatingCostPercent),
		model.FormatSettlementPercent(d.MarkupDiscountPercent),
		model.FormatSettlementPercent(d.SalesDiscountPercent),
		model.FormatSettlementMoney(bd.OfficialTotal),
		model.FormatSettlementMoney(bd.CostPrice),
		model.FormatSettlementMoney(bd.OperatingPrice),
		model.FormatSettlementMoney(bd.SalesPrice),
	}
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
