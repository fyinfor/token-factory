package controller

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

const PaymentMethodAntom = "antom"

var antomAdaptor = &AntomAdaptor{}

type AntomPayRequest struct {
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	SuccessURL    string  `json:"success_url,omitempty"`
}

type AntomAdaptor struct{}

func antomPayCurrencyForQuote() string {
	cur := setting.GetAntomPayCurrency()
	if cur == "USD" {
		return topupCurrencyUSD
	}
	return topupCurrencyCNY
}

func (*AntomAdaptor) RequestAmount(c *gin.Context, req *AntomPayRequest) {
	if req.Amount < float64(getAntomMinTopup()) {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getAntomMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	quote, err := buildTopupQuote(req.Amount, group, antomPayCurrencyForQuote())
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": err.Error()})
		return
	}
	if quote.PayAmount <= 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": quote})
}

func (*AntomAdaptor) RequestPay(c *gin.Context, req *AntomPayRequest) {
	if req.PaymentMethod != PaymentMethodAntom {
		c.JSON(200, gin.H{"message": "error", "data": "不支持的支付渠道"})
		return
	}
	if !setting.AntomReady() {
		c.JSON(200, gin.H{"message": "error", "data": "管理员未开启 Antom 充值"})
		return
	}
	if req.Amount < float64(getAntomMinTopup()) {
		c.JSON(200, gin.H{"message": fmt.Sprintf("充值数量不能小于 %d", getAntomMinTopup()), "data": 10})
		return
	}
	if req.Amount > 10000 {
		c.JSON(200, gin.H{"message": "充值数量不能大于 10000", "data": 10})
		return
	}
	if req.SuccessURL != "" && common.ValidateRedirectURL(req.SuccessURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "支付成功重定向URL不在可信任域名列表中", "data": ""})
		return
	}

	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)
	quote, err := buildTopupQuote(req.Amount, user.Group, antomPayCurrencyForQuote())
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": err.Error()})
		return
	}

	payCurrency := setting.GetAntomPayCurrency()
	payAmount := quote.PayAmount

	reference := fmt.Sprintf("new-api-antom-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "ref_" + common.Sha1([]byte(reference))

	minor, err := formatAntomMinorAmount(payAmount)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": err.Error()})
		return
	}

	platformBase := platformPublicBaseURL(c)
	redirectURL := req.SuccessURL
	if redirectURL == "" {
		redirectURL = platformBase + "/console/log"
	}
	notifyURL := platformBase + "/api/antom/notify"
	terminalType := "WEB"
	osType := ""
	ua := strings.ToLower(c.GetHeader("User-Agent"))
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		terminalType = "WAP"
		if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
			osType = "IOS"
		} else {
			osType = "ANDROID"
		}
	}

	payLink, err := service.CreateAntomCheckoutSession(
		referenceId,
		minor,
		payCurrency,
		notifyURL,
		redirectURL,
		strconv.Itoa(user.Id),
		terminalType,
		osType,
	)
	if err != nil {
		log.Println("获取Antom Checkout支付链接失败", err)
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	topUp := &model.TopUp{
		UserId:        id,
		Amount:        req.Amount,
		Money:         payAmount,
		InputAmount:   quote.InputAmount,
		InputCurrency: quote.InputCurrency,
		PayCurrency:   payCurrency,
		QuotaToAdd:    quote.QuotaToAdd,
		TradeNo:       referenceId,
		PaymentMethod: PaymentMethodAntom,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
		},
	})
}

func RequestAntomAmount(c *gin.Context) {
	var req AntomPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	antomAdaptor.RequestAmount(c, &req)
}

func RequestAntomPay(c *gin.Context) {
	var req AntomPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	antomAdaptor.RequestPay(c, &req)
}

func AntomNotify(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, service.AntomFailAck("read body failed"))
		return
	}
	if !setting.AntomConfigured() {
		log.Printf("[SECURITY][Antom] 密钥未配置，拒绝处理 notify")
		c.JSON(http.StatusOK, service.AntomFailAck("not configured"))
		return
	}
	uri := c.Request.URL.Path
	if uri == "" {
		uri = "/api/antom/notify"
	}
	err = service.VerifyAntomNotification(
		uri,
		c.GetHeader("Client-Id"),
		c.GetHeader("Request-Time"),
		c.GetHeader("Signature"),
		string(payload),
	)
	if err != nil {
		log.Printf("[SECURITY][Antom] 验签失败: %v", err)
		c.JSON(http.StatusOK, service.AntomFailAck("invalid signature"))
		return
	}

	notify, err := service.ParseAntomNotify(payload)
	if err != nil {
		c.JSON(http.StatusOK, service.AntomFailAck("invalid body"))
		return
	}

	if notify.NotifyType == "PAYMENT_PENDING" {
		c.JSON(http.StatusOK, service.AntomSuccessAck())
		return
	}

	referenceId := notify.PaymentRequestId
	if !strings.HasPrefix(referenceId, "ref_") {
		log.Printf("[SECURITY][Antom] 拒绝可疑回调，订单号前缀非法: %s", referenceId)
		c.JSON(http.StatusOK, service.AntomFailAck("invalid paymentRequestId"))
		return
	}

	LockOrder(referenceId)
	defer UnlockOrder(referenceId)

	if notify.NotifyType == "PAYMENT_RESULT" && notify.Result.ResultStatus == "F" {
		expireAntomTopUp(referenceId)
		c.JSON(http.StatusOK, service.AntomSuccessAck())
		return
	}

	if notify.NotifyType != "PAYMENT_RESULT" || notify.Result.ResultStatus != "S" {
		c.JSON(http.StatusOK, service.AntomSuccessAck())
		return
	}

	paidMoney, err := parseAntomMajorAmount(notify.PaymentAmount.Value)
	if err != nil {
		log.Printf("[SECURITY][Antom] 金额解析失败 trade_no=%s value=%s err=%v", referenceId, notify.PaymentAmount.Value, err)
		c.JSON(http.StatusOK, service.AntomFailAck("invalid amount"))
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(notify.PaymentAmount.Currency))
	if err := model.RechargeAntom(referenceId, paidMoney, currency); err != nil {
		existing := model.GetTopUpByTradeNo(referenceId)
		if existing != nil && existing.Status == common.TopUpStatusSuccess {
			c.JSON(http.StatusOK, service.AntomSuccessAck())
			return
		}
		log.Printf("[SECURITY][Antom] 回调校验未通过 trade_no=%s paid=%.2f currency=%s err=%s", referenceId, paidMoney, currency, err.Error())
		c.JSON(http.StatusOK, service.AntomFailAck("recharge failed"))
		return
	}
	log.Printf("收到 Antom 款项：%s, %.2f(%s)", referenceId, paidMoney, currency)
	c.JSON(http.StatusOK, service.AntomSuccessAck())
}

func expireAntomTopUp(referenceId string) {
	topUp := model.GetTopUpByTradeNo(referenceId)
	if topUp == nil || topUp.Status != common.TopUpStatusPending {
		return
	}
	topUp.Status = common.TopUpStatusExpired
	if err := topUp.Update(); err != nil {
		log.Println("过期 Antom 充值订单失败", referenceId, err)
	}
}

func formatAntomMinorAmount(amount float64) (string, error) {
	d := decimal.NewFromFloat(amount).Mul(decimal.NewFromInt(100)).Round(0)
	if d.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("无效的支付金额")
	}
	return d.StringFixed(0), nil
}

func parseAntomMajorAmount(minor string) (float64, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(minor))
	if err != nil {
		return 0, err
	}
	f, _ := d.Div(decimal.NewFromInt(100)).Float64()
	return f, nil
}

func getAntomMinTopup() int64 {
	return int64(setting.AntomMinTopUp)
}

// platformPublicBaseURL 使用本平台「服务器地址」；未配置或仍是默认 open 站点时，改用当前请求的平台域名。
func platformPublicBaseURL(c *gin.Context) string {
	base := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if base != "" && !strings.EqualFold(base, "https://tokenfactoryopen.com") && !strings.EqualFold(base, "http://tokenfactoryopen.com") {
		return base
	}
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	host = strings.TrimSpace(strings.Split(host, ",")[0])
	if host == "" {
		return base
	}
	return scheme + "://" + host
}
