package controller

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func GetTopUpInfo(c *gin.Context) {
	// 获取支付方式
	payMethods := operation_setting.PayMethods

	// 如果启用了 Stripe 支付，添加到支付方法列表
	if setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "" {
		// 检查是否已经包含 Stripe
		hasStripe := false
		for _, method := range payMethods {
			if method["type"] == "stripe" {
				hasStripe = true
				break
			}
		}

		if !hasStripe {
			stripeMethod := map[string]string{
				"name":      "Stripe",
				"type":      "stripe",
				"color":     "rgba(var(--semi-purple-5), 1)",
				"min_topup": strconv.Itoa(setting.StripeMinTopUp),
			}
			payMethods = append(payMethods, stripeMethod)
		}
	}

	// 如果启用了 Waffo 支付，添加到支付方法列表
	enableWaffo := setting.WaffoEnabled &&
		((!setting.WaffoSandbox &&
			setting.WaffoApiKey != "" &&
			setting.WaffoPrivateKey != "" &&
			setting.WaffoPublicCert != "") ||
			(setting.WaffoSandbox &&
				setting.WaffoSandboxApiKey != "" &&
				setting.WaffoSandboxPrivateKey != "" &&
				setting.WaffoSandboxPublicCert != ""))
	if enableWaffo {
		hasWaffo := false
		for _, method := range payMethods {
			if method["type"] == "waffo" {
				hasWaffo = true
				break
			}
		}

		if !hasWaffo {
			waffoMethod := map[string]string{
				"name":      "Waffo (Global Payment)",
				"type":      "waffo",
				"color":     "rgba(var(--semi-blue-5), 1)",
				"min_topup": strconv.Itoa(setting.WaffoMinTopUp),
			}
			payMethods = append(payMethods, waffoMethod)
		}
	}

	data := gin.H{
		"enable_online_topup": (operation_setting.OnlinePayProvider == "yipay" &&
			(operation_setting.YipayRequestURL != "" || operation_setting.PayAddress != "") &&
			operation_setting.YipayMchNo != "" &&
			operation_setting.YipayAppId != "" &&
			operation_setting.YipayAppSecret != "") ||
			(operation_setting.OnlinePayProvider != "yipay" &&
				operation_setting.PayAddress != "" &&
				operation_setting.EpayId != "" &&
				operation_setting.EpayKey != ""),
		"enable_stripe_topup": setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "",
		"enable_creem_topup":  setting.CreemApiKey != "" && setting.CreemProducts != "[]",
		"enable_waffo_topup":  enableWaffo,
		"waffo_pay_methods": func() interface{} {
			if enableWaffo {
				return setting.GetWaffoPayMethods()
			}
			return nil
		}(),
		"creem_products":      setting.CreemProducts,
		"pay_methods":         payMethods,
		"min_topup":           operation_setting.MinTopUp,
		"stripe_min_topup":    setting.StripeMinTopUp,
		"waffo_min_topup":     setting.WaffoMinTopUp,
		"amount_options":      operation_setting.GetPaymentSetting().AmountOptions,
		"discount":            operation_setting.GetPaymentSetting().AmountDiscount,
		"online_pay_provider": operation_setting.OnlinePayProvider,
	}
	common.ApiSuccess(c, data)
}

type EpayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

type AmountRequest struct {
	Amount int64 `json:"amount"`
}

// buildUserEpayNotifyURL 规范化用户充值异步回调地址，避免重复拼接 notify 路径。
func buildUserEpayNotifyURL(callbackAddress string) string {
	normalized := strings.TrimRight(strings.TrimSpace(callbackAddress), "/")
	if strings.HasSuffix(normalized, "/api/user/epay/notify") {
		return normalized
	}
	return normalized + "/api/user/epay/notify"
}

// verifyYipayNotify 验证 Yipay 回调签名。
func verifyYipayNotify(params map[string]string) bool {
	sign := strings.TrimSpace(params["sign"])
	if sign == "" || operation_setting.YipayAppSecret == "" {
		return false
	}
	expected := signYipayMD5(params, operation_setting.YipayAppSecret)
	return strings.EqualFold(sign, expected)
}

// sortedParamKeys 返回回调参数的有序键列表，便于日志排查。
func sortedParamKeys(params map[string]string) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// signYipayMD5 生成 Yipay MD5 签名（参数按 ASCII 排序）。
func signYipayMD5(params map[string]string, appSecret string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || strings.TrimSpace(v) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	parts = append(parts, "key="+appSecret)
	raw := strings.Join(parts, "&")
	sum := md5.Sum([]byte(raw))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// parseYipayUnifiedOrderData 解析 Jeepay/Yipay 统一下单响应中的 data 字段（文档多为 JSON 对象，部分网关序列化为 JSON 字符串）。
func parseYipayUnifiedOrderData(yipayResp map[string]any) (map[string]any, error) {
	raw, ok := yipayResp["data"]
	if !ok || raw == nil {
		return nil, fmt.Errorf("响应缺少 data 字段")
	}
	if m, ok := raw.(map[string]any); ok {
		return m, nil
	}
	s, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("data 字段类型不支持")
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return nil, fmt.Errorf("data 字段为空")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("data JSON 字符串解析失败: %w", err)
	}
	return m, nil
}

// yipayOrderStringField 从统一下单 data 中读取字符串字段（兼容驼峰与蛇形命名，忽略空值与占位 "<nil>"）。
func yipayOrderStringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			continue
		}
		return s
	}
	return ""
}

// looksLikeYipayPayJumpURL 判断是否为网关返回的可拉起收银台的链接（http(s)、微信/支付宝 App scheme 等）。
func looksLikeYipayPayJumpURL(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return true
	}
	if strings.HasPrefix(low, "weixin://") || strings.HasPrefix(low, "weixinpay://") {
		return true
	}
	if strings.HasPrefix(low, "alipays://") || strings.HasPrefix(low, "alipay://") {
		return true
	}
	return false
}

// yipayURLExtractMaxDepth 限制嵌套 map/JSON 解析深度，防止异常响应导致过深递归。
const yipayURLExtractMaxDepth = 8

// yipayPayLinkInTextRegexp 从文本中提取首个支付相关链接（http(s)、weixin://、alipay(s)://）。
var yipayPayLinkInTextRegexp = regexp.MustCompile(`(?:https?://|weixin://[^\s"'<>]*|weixinpay://[^\s"'<>]*|alipay?s://[^\s"'<>]+)`)

// yipayFirstPayLinkInString 返回文本中出现的第一个可识别的支付跳转链接。
func yipayFirstPayLinkInString(s string) string {
	m := yipayPayLinkInTextRegexp.FindString(s)
	return strings.TrimSpace(m)
}

// yipaySortedMapKeys 返回 map 键的有序列表，便于日志排查。
func yipaySortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// yipayCoerceValueToHTTPURL 将网关返回字段转为可拉起支付的链接（http(s)、weixin:// 等；string / JSON / 嵌套 map / 文本内嵌）。
func yipayCoerceValueToHTTPURL(v any, depth int) string {
	if v == nil || depth > yipayURLExtractMaxDepth {
		return ""
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "<nil>" {
			return ""
		}
		if looksLikeYipayPayJumpURL(s) {
			return s
		}
		st := strings.TrimSpace(s)
		if strings.HasPrefix(st, "{") || strings.HasPrefix(st, "[") {
			var raw json.RawMessage
			if err := json.Unmarshal([]byte(s), &raw); err == nil {
				var obj map[string]any
				if err := json.Unmarshal(raw, &obj); err == nil {
					if u := yipayExtractHTTPURLFromMap(obj, depth+1); u != "" {
						return u
					}
				}
			}
		}
		if u := yipayFirstPayLinkInString(s); u != "" {
			return u
		}
		return ""
	case map[string]any:
		return yipayExtractHTTPURLFromMap(t, depth+1)
	case []any:
		for _, item := range t {
			if u := yipayCoerceValueToHTTPURL(item, depth+1); u != "" {
				return u
			}
		}
		return ""
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", t))
		if looksLikeYipayPayJumpURL(s) {
			return s
		}
		return ""
	}
}

// yipayExtractHTTPURLFromMap 在 map 中按常见字段名优先查找支付链接，再浅层扫描其余字段（跳过错误信息类键）。
func yipayExtractHTTPURLFromMap(m map[string]any, depth int) string {
	if m == nil || depth > yipayURLExtractMaxDepth {
		return ""
	}
	priorityKeys := []string{
		"payUrl", "pay_url", "payJumpUrl", "pay_jump_url",
		"codeUrl", "code_url",
		"codeImgUrl", "code_img_url",
		"url", "link", "h5Url", "h5_url", "checkoutUrl", "checkout_url",
		"approvalUrl", "approval_url", "redirectUrl", "redirect_url",
		"mweb_url", "mwebUrl", "payData", "pay_data",
	}
	for _, k := range priorityKeys {
		if u := yipayCoerceValueToHTTPURL(m[k], depth+1); u != "" {
			return u
		}
	}
	skipKeys := map[string]struct{}{
		"errMsg": {}, "err_msg": {}, "errCode": {}, "err_code": {},
		"channelErrMsg": {}, "channelErrCode": {}, "errMsgFromChannel": {},
	}
	for k, v := range m {
		if _, skip := skipKeys[k]; skip {
			continue
		}
		if u := yipayCoerceValueToHTTPURL(v, depth+1); u != "" {
			return u
		}
	}
	return ""
}

// extractYipayPayURLFromOrderData 从 data 中提取支付跳转 URL（兼容对象型 payData、嵌套 JSON、非标准字段名；表单收银台单独提示）。
func extractYipayPayURLFromOrderData(dataObj map[string]any) (string, error) {
	payDataType := strings.ToLower(yipayOrderStringField(dataObj, "payDataType", "pay_data_type"))
	payDataStr := yipayOrderStringField(dataObj, "payData", "pay_data")
	payDataRaw := dataObj["payData"]
	if payDataRaw == nil {
		payDataRaw = dataObj["pay_data"]
	}

	if payDataType == "form" || strings.Contains(strings.ToLower(payDataStr), "<form") {
		for _, k := range []string{"payUrl", "pay_url", "payJumpUrl", "pay_jump_url"} {
			if u := yipayCoerceValueToHTTPURL(dataObj[k], 0); u != "" {
				return u, nil
			}
		}
		return "", fmt.Errorf("Yipay 返回表单收银台（payDataType=form），当前充值页仅支持链接跳转；请在网关配置返回 payUrl/payJumpUrl，或升级前端支持表单 HTML 收银台")
	}

	for _, key := range []string{"payUrl", "pay_url", "payJumpUrl", "pay_jump_url"} {
		if u := yipayCoerceValueToHTTPURL(dataObj[key], 0); u != "" {
			return u, nil
		}
	}
	if u := yipayCoerceValueToHTTPURL(payDataRaw, 0); u != "" {
		return u, nil
	}
	if u := yipayExtractHTTPURLFromMap(dataObj, 0); u != "" {
		return u, nil
	}

	ec := yipayOrderStringField(dataObj, "errCode", "err_code")
	em := yipayOrderStringField(dataObj, "errMsg", "err_msg")
	os := yipayOrderStringField(dataObj, "orderState", "order_state")
	pid := yipayOrderStringField(dataObj, "payOrderId", "pay_order_id")
	common.SysLog(fmt.Sprintf(
		"[Yipay] unifiedOrder data 未解析出跳转链接: keys=%v payDataType=%q payDataGoType=%T errCode=%q errMsg=%q orderState=%s payOrderId=%s",
		yipaySortedMapKeys(dataObj), payDataType, payDataRaw, ec, em, os, pid,
	))
	if ec != "" || em != "" {
		return "", fmt.Errorf("Jeepay 已创建订单 payOrderId=%s（orderState=%s），上游 errCode=%q errMsg=%q 且无跳转链接；请核对 PayPal 通道配置，或在管理后台「Yipay 渠道扩展参数」填写 {\"payDataType\":\"payUrl\"}", pid, os, ec, em)
	}
	if pid != "" {
		return "", fmt.Errorf("Jeepay 返回 payOrderId=%s、orderState=%s，但 data 中未解析出可用支付链接（微信支付多为 codeUrl/weixin://；H5 多为 mweb_url；PayPal 多为 payUrl；PP_PC 需在 channelExtra 提供 cancelUrl 等）", pid, os)
	}
	return "", fmt.Errorf("Yipay 返回支付链接为空（data 内无可用 http(s) 链接，详见服务端日志）")
}

// yipayJeepayDefaultChannelExtra 构造 yiPay/Jeepay 默认 channelExtra。
// PP_PC：UnifiedOrderRQ.buildBizRQ 将 channelExtra 解析为 PPPcOrderRQ，其中 cancelUrl 为 @NotBlank（见 yiPay PpPc.java / PPPcOrderRQ.java）。
func yipayJeepayDefaultChannelExtra(wayCode string, returnURL string) string {
	u := strings.ToUpper(strings.TrimSpace(wayCode))
	if u == "" {
		return ""
	}
	ret := strings.TrimSpace(returnURL)
	if u == "PP_PC" {
		if ret == "" {
			ret = "http://127.0.0.1/console/log"
		}
		b, err := json.Marshal(map[string]string{
			"payDataType": "payUrl",
			"cancelUrl":   ret,
		})
		if err != nil {
			return ""
		}
		return string(b)
	}
	if strings.HasSuffix(u, "_PC") {
		return `{"payDataType":"payUrl"}`
	}
	return ""
}

// mergeYipayJeepayChannelExtra 合并管理端配置的 channelExtra JSON 与 wayCode 默认值（仅补充对方 JSON 中未出现的键）。
func mergeYipayJeepayChannelExtra(userJSON string, wayCode string, returnURL string) string {
	userJSON = strings.TrimSpace(userJSON)
	def := yipayJeepayDefaultChannelExtra(wayCode, returnURL)
	if userJSON == "" {
		return def
	}
	if def == "" {
		return userJSON
	}
	var u map[string]any
	if err := json.Unmarshal([]byte(userJSON), &u); err != nil {
		return userJSON
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(def), &d); err != nil {
		return userJSON
	}
	for k, v := range d {
		if _, exists := u[k]; !exists {
			u[k] = v
		}
	}
	out, err := json.Marshal(u)
	if err != nil {
		return userJSON
	}
	return string(out)
}

// isPayPalJeepayWayCode 判断是否为 yiPay/Jeepay 的 PayPal 产品码（如 PP_PC）；PayPal REST 常不支持以 CNY 下单。
func isPayPalJeepayWayCode(wayCode string) bool {
	u := strings.ToUpper(strings.TrimSpace(wayCode))
	return strings.HasPrefix(u, "PP_")
}

// yipayJeepayAmountMinorAndCurrency 计算统一下单的 amount（最小货币单位）与币种。
// payMoney 与 getPayMoney 一致，单位为人民币元；PayPal 通道换算为 USD 美分（汇率优先 USDExchangeRate，其次 Price）。
func yipayJeepayAmountMinorAndCurrency(payMoney float64, wayCode string) (minor int64, currency string) {
	currency = "cny"
	minor = int64(math.Round(payMoney * 100))
	if minor < 1 {
		minor = 1
	}
	if !isPayPalJeepayWayCode(wayCode) {
		return minor, currency
	}
	rate := operation_setting.USDExchangeRate
	if rate <= 0 {
		rate = operation_setting.Price
	}
	if rate <= 0 {
		rate = 7.3
	}
	usd := payMoney / rate
	minor = int64(math.Round(usd * 100))
	if minor < 1 {
		minor = 1
	}
	return minor, "usd"
}

// requestYipayOrder 按 Yipay OpenAPI 创建统一下单请求。
func requestYipayOrder(req EpayRequest, id int, payMoney float64, paymentMethod string) (string, map[string]string, error) {
	requestURL := operation_setting.YipayRequestURL
	if requestURL == "" {
		requestURL = operation_setting.PayAddress
	}
	requestURL = strings.TrimRight(requestURL, "/")
	if requestURL == "" {
		return "", nil, fmt.Errorf("未配置 Yipay 请求地址")
	}
	if operation_setting.YipayMchNo == "" || operation_setting.YipayAppId == "" || operation_setting.YipayAppSecret == "" {
		return "", nil, fmt.Errorf("未配置完整的 Yipay 参数")
	}
	callBackAddress := service.GetCallbackAddress()
	returnURLString := system_setting.ServerAddress + "/console/log"
	if operation_setting.YipayReturnUrl != "" {
		returnURLString = operation_setting.YipayReturnUrl
	}
	notifyURLString := buildUserEpayNotifyURL(callBackAddress)
	if operation_setting.YipayNotifyUrl != "" {
		notifyURLString = operation_setting.YipayNotifyUrl
	}
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)
	amountMinor, orderCurrency := yipayJeepayAmountMinorAndCurrency(payMoney, paymentMethod)
	reqTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	chExtra := mergeYipayJeepayChannelExtra(strings.TrimSpace(operation_setting.YipayChannelExtra), paymentMethod, returnURLString)
	params := map[string]string{
		"mchNo":      operation_setting.YipayMchNo,
		"appId":      operation_setting.YipayAppId,
		"mchOrderNo": tradeNo,
		"wayCode":    paymentMethod,
		"amount":     strconv.FormatInt(amountMinor, 10),
		"currency":   strings.ToLower(orderCurrency),
		"clientIp":   "127.0.0.1",
		"subject":    fmt.Sprintf("TUC%d", req.Amount),
		"body":       fmt.Sprintf("TUC%d", req.Amount),
		"notifyUrl":  notifyURLString,
		"returnUrl":  returnURLString,
		"reqTime":    reqTime,
		"version":    "1.0",
		"signType":   "MD5",
	}
	if chExtra != "" {
		params["channelExtra"] = chExtra
	}
	// 调试日志：输出关键参数的可见信息，便于定位是否包含前后空格（不输出 AppSecret 明文）。
	common.SysLog(fmt.Sprintf(
		"[Yipay] unifiedOrder params: mchNo=%q(len=%d,trimChanged=%t), appId=%q(len=%d,trimChanged=%t), wayCode=%q, amountMinor=%s currency=%s payMoneyCNY=%.4f channelExtra=%q, notifyUrl=%q, returnUrl=%q, reqTime=%q",
		params["mchNo"], len(params["mchNo"]), strings.TrimSpace(params["mchNo"]) != params["mchNo"],
		params["appId"], len(params["appId"]), strings.TrimSpace(params["appId"]) != params["appId"],
		params["wayCode"], params["amount"], params["currency"], payMoney, chExtra, params["notifyUrl"], params["returnUrl"], params["reqTime"],
	))
	params["sign"] = signYipayMD5(params, operation_setting.YipayAppSecret)
	requestJSON, err := common.Marshal(params)
	if err != nil {
		return "", nil, err
	}
	unifiedOrderURL := requestURL
	if !strings.Contains(strings.ToLower(unifiedOrderURL), "/api/pay/unifiedorder") {
		unifiedOrderURL = requestURL + "/api/pay/unifiedOrder"
	}
	httpResp, err := http.Post(unifiedOrderURL, "application/json", strings.NewReader(string(requestJSON)))
	if err != nil {
		return "", nil, err
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", nil, err
	}
	var yipayResp map[string]any
	if err = common.Unmarshal(respBody, &yipayResp); err != nil {
		respPreview := string(respBody)
		if len(respPreview) > 300 {
			respPreview = respPreview[:300]
		}
		return "", nil, fmt.Errorf("Yipay 响应解析失败(status=%d,url=%s): %s", httpResp.StatusCode, unifiedOrderURL, respPreview)
	}
	code := fmt.Sprintf("%v", yipayResp["code"])
	if code != "0" && code != "200" {
		msg := fmt.Sprintf("%v", yipayResp["msg"])
		if msg == "<nil>" || msg == "" {
			if v, ok := yipayResp["message"]; ok {
				msg = fmt.Sprintf("%v", v)
			}
		}
		if msg == "<nil>" || msg == "" {
			if v, ok := yipayResp["errMsg"]; ok {
				msg = fmt.Sprintf("%v", v)
			}
		}
		if msg == "<nil>" || msg == "" {
			msg = string(respBody)
			if len(msg) > 300 {
				msg = msg[:300]
			}
		}
		return "", nil, fmt.Errorf("Yipay 下单失败(status=%d,code=%s,url=%s): %s", httpResp.StatusCode, code, unifiedOrderURL, msg)
	}
	dataObj, parseErr := parseYipayUnifiedOrderData(yipayResp)
	if parseErr != nil {
		return "", nil, fmt.Errorf("Yipay 统一下单返回解析失败: %w", parseErr)
	}
	// 少数网关/代理把 payUrl、payData 等放在响应根级而非 data 内，合并后再解析。
	for _, k := range []string{"payUrl", "pay_url", "payJumpUrl", "pay_jump_url", "payData", "pay_data", "payDataType", "pay_data_type"} {
		if v, ok := yipayResp[k]; ok && v != nil {
			if _, exists := dataObj[k]; !exists {
				dataObj[k] = v
			}
		}
	}
	payURL, extractErr := extractYipayPayURLFromOrderData(dataObj)
	if extractErr != nil {
		return "", nil, extractErr
	}
	payURL = strings.TrimSpace(payURL)
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:        id,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        "pending",
	}
	if err = topUp.Insert(); err != nil {
		return "", nil, err
	}
	return payURL, map[string]string{}, nil
}

// GetEpayClient 创建并返回在线充值客户端。
func GetEpayClient() *epay.Client {
	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		return nil
	}
	withUrl, err := epay.NewClient(&epay.Config{
		PartnerID: operation_setting.EpayId,
		Key:       operation_setting.EpayKey,
	}, operation_setting.PayAddress)
	if err != nil {
		return nil
	}
	return withUrl
}

func getPayMoney(amount int64, group string) float64 {
	dAmount := decimal.NewFromInt(amount)
	// 充值金额以“展示类型”为准：
	// - USD/CNY: 前端传 amount 为金额单位；TOKENS: 前端传 tokens，需要换成 USD 金额
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dAmount = dAmount.Div(dQuotaPerUnit)
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	dTopupGroupRatio := decimal.NewFromFloat(topupGroupRatio)
	dPrice := decimal.NewFromFloat(operation_setting.Price)
	// apply optional preset discount by the original request amount (if configured), default 1.0
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	dDiscount := decimal.NewFromFloat(discount)

	payMoney := dAmount.Mul(dPrice).Mul(dTopupGroupRatio).Mul(dDiscount)

	return payMoney.InexactFloat64()
}

func getMinTopup() int64 {
	minTopup := operation_setting.MinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dMinTopup := decimal.NewFromInt(int64(minTopup))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		minTopup = int(dMinTopup.Mul(dQuotaPerUnit).IntPart())
	}
	return int64(minTopup)
}

// RequestEpay 创建在线充值订单并拉起支付。
func RequestEpay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	paymentMethod := req.PaymentMethod
	if !operation_setting.ContainsPayMethod(paymentMethod) {
		c.JSON(200, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}
	if operation_setting.OnlinePayProvider == "yipay" {
		payURL, params, yipayErr := requestYipayOrder(req, id, payMoney, paymentMethod)
		if yipayErr != nil {
			c.JSON(200, gin.H{"message": "error", "data": yipayErr.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "success", "data": params, "url": payURL})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl, _ := url.Parse(system_setting.ServerAddress + "/console/log")
	notifyUrl, _ := url.Parse(buildUserEpayNotifyURL(callBackAddress))
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)
	client := GetEpayClient()
	if client == nil {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           paymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("TUC%d", req.Amount),
		Money:          strconv.FormatFloat(payMoney, 'f', 2, 64),
		Device:         epay.PC,
		NotifyUrl:      notifyUrl,
		ReturnUrl:      returnUrl,
	})
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:        id,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        "pending",
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": params, "url": uri})
}

// tradeNo lock
var orderLocks sync.Map
var createLock sync.Mutex

// refCountedMutex 带引用计数的互斥锁，确保最后一个使用者才从 map 中删除
type refCountedMutex struct {
	mu       sync.Mutex
	refCount int
}

// LockOrder 尝试对给定订单号加锁
func LockOrder(tradeNo string) {
	createLock.Lock()
	var rcm *refCountedMutex
	if v, ok := orderLocks.Load(tradeNo); ok {
		rcm = v.(*refCountedMutex)
	} else {
		rcm = &refCountedMutex{}
		orderLocks.Store(tradeNo, rcm)
	}
	rcm.refCount++
	createLock.Unlock()
	rcm.mu.Lock()
}

// UnlockOrder 释放给定订单号的锁
func UnlockOrder(tradeNo string) {
	v, ok := orderLocks.Load(tradeNo)
	if !ok {
		return
	}
	rcm := v.(*refCountedMutex)
	rcm.mu.Unlock()

	createLock.Lock()
	rcm.refCount--
	if rcm.refCount == 0 {
		orderLocks.Delete(tradeNo)
	}
	createLock.Unlock()
}

func EpayNotify(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == "POST" {
		// POST 请求：从 POST body 解析参数
		if err := c.Request.ParseForm(); err != nil {
			log.Println("易支付回调POST解析失败:", err)
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
		// 兼容 JSON 回调体（部分 Yipay/Jeepay 部署会使用 application/json 推送）。
		if len(params) == 0 {
			bodyBytes, readErr := io.ReadAll(c.Request.Body)
			if readErr == nil && len(bodyBytes) > 0 {
				var payload map[string]any
				if unmarshalErr := common.Unmarshal(bodyBytes, &payload); unmarshalErr == nil {
					params = lo.Reduce(lo.Keys(payload), func(r map[string]string, t string, i int) map[string]string {
						r[t] = fmt.Sprintf("%v", payload[t])
						return r
					}, map[string]string{})
				}
			}
		}
	} else {
		// GET 请求：从 URL Query 解析参数
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		log.Printf("易支付回调参数为空，method=%s, contentType=%s, remoteIP=%s, rawQuery=%s", c.Request.Method, c.ContentType(), c.ClientIP(), c.Request.URL.RawQuery)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	// 回调入站日志：用于确认回调是否到达以及携带了哪些参数。
	log.Printf("支付回调已到达，provider=%s, method=%s, contentType=%s, remoteIP=%s, keys=%v",
		operation_setting.OnlinePayProvider, c.Request.Method, c.ContentType(), c.ClientIP(), sortedParamKeys(params))

	// 先尝试按 Yipay 回调处理：使用 Yipay AppSecret 做 MD5 验签，并按 mchOrderNo/state 更新订单。
	if operation_setting.OnlinePayProvider == "yipay" {
		log.Printf("Yipay 回调关键参数：mchOrderNo=%s, state=%s, payOrderId=%s",
			params["mchOrderNo"], params["state"], params["payOrderId"])
		if !verifyYipayNotify(params) {
			log.Printf("Yipay 回调签名验证失败，mchOrderNo=%s, state=%s", params["mchOrderNo"], params["state"])
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		tradeNo := strings.TrimSpace(params["mchOrderNo"])
		state := strings.TrimSpace(params["state"])
		if tradeNo == "" {
			log.Println("Yipay 回调缺少 mchOrderNo")
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		if state != "2" {
			log.Printf("Yipay 回调非成功状态，mchOrderNo=%s, state=%s", tradeNo, state)
			_, _ = c.Writer.Write([]byte("success"))
			return
		}
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		topUp := model.GetTopUpByTradeNo(tradeNo)
		if topUp == nil {
			log.Printf("Yipay 回调未找到订单: %s", tradeNo)
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		if topUp.Status == "pending" {
			topUp.Status = "success"
			if err := topUp.Update(); err != nil {
				log.Printf("Yipay 回调更新订单失败: %v", topUp)
				_, _ = c.Writer.Write([]byte("fail"))
				return
			}
			dAmount := decimal.NewFromInt(int64(topUp.Amount))
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
			if err := model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true); err != nil {
				log.Printf("Yipay 回调更新用户失败: %v", topUp)
				_, _ = c.Writer.Write([]byte("fail"))
				return
			}
			log.Printf("Yipay 回调更新用户成功 %v", topUp)
			model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money))
			// 与易支付成功路径一致：到账后按邀请关系给邀请人分销提成
			model.ApplyAffiliateTopupReward(topUp.UserId, quotaToAdd)
		}
		_, _ = c.Writer.Write([]byte("success"))
		return
	}

	client := GetEpayClient()
	if client == nil {
		log.Println("易支付回调失败 未找到配置信息")
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
		return
	}
	verifyInfo, err := client.Verify(params)
	if err == nil && verifyInfo.VerifyStatus {
		_, err := c.Writer.Write([]byte("success"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
	} else {
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
		log.Println("易支付回调签名验证失败")
		return
	}

	if verifyInfo.TradeStatus == epay.StatusTradeSuccess {
		log.Println(verifyInfo)
		LockOrder(verifyInfo.ServiceTradeNo)
		defer UnlockOrder(verifyInfo.ServiceTradeNo)
		topUp := model.GetTopUpByTradeNo(verifyInfo.ServiceTradeNo)
		if topUp == nil {
			log.Printf("易支付回调未找到订单: %v", verifyInfo)
			return
		}
		if topUp.Status == "pending" {
			topUp.Status = "success"
			err := topUp.Update()
			if err != nil {
				log.Printf("易支付回调更新订单失败: %v", topUp)
				return
			}
			//user, _ := model.GetUserById(topUp.UserId, false)
			//user.Quota += topUp.Amount * 500000
			dAmount := decimal.NewFromInt(int64(topUp.Amount))
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
			err = model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true)
			if err != nil {
				log.Printf("易支付回调更新用户失败: %v", topUp)
				return
			}
			log.Printf("易支付回调更新用户成功 %v", topUp)
			model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money))
			// 支付回调成功且额度已入账后发放邀请人分销奖励（与 model.Recharge / Yipay 等路径一致）
			model.ApplyAffiliateTopupReward(topUp.UserId, quotaToAdd)
		}
	} else {
		log.Printf("易支付异常回调: %v", verifyInfo)
	}
}

func RequestAmount(c *gin.Context) {
	var req AmountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func GetUserTopUps(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchUserTopUps(userId, keyword, pageInfo)
	} else {
		topups, total, err = model.GetUserTopUps(userId, pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

// GetAllTopUps 管理员获取全平台充值记录
func GetAllTopUps(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchAllTopUps(keyword, pageInfo)
	} else {
		topups, total, err = model.GetAllTopUps(pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

type AdminCompleteTopupRequest struct {
	TradeNo string `json:"trade_no"`
}

// AdminCompleteTopUp 管理员补单接口
func AdminCompleteTopUp(c *gin.Context) {
	var req AdminCompleteTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	// 订单级互斥，防止并发补单
	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	// adminUsername 用于补单后写入使用日志详情，便于追溯具体操作者。
	adminUsername := c.GetString("username")
	if err := model.ManualCompleteTopUp(req.TradeNo, adminUsername); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
