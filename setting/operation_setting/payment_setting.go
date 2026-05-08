package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// Yipay/Epay 通用支付配置（兼容旧字段）。
var PayAddress = ""
var CustomCallbackAddress = ""
var EpayId = ""
var EpayKey = ""
var YipayAppSecret = ""
var OnlinePayProvider = "yipay"
var Price = 7.3
var MinTopUp = 1
var USDExchangeRate = 7.3

// Yipay 扩展配置。
var YipayMchNo = ""
var YipayAppId = ""
var YipayWayCode = ""
var YipayNotifyUrl = ""
var YipayReturnUrl = ""
var YipayRequestURL = ""
// YipayChannelExtra 为 Jeepay 统一下单的 channelExtra（JSON 字符串）；可与服务端按 wayCode 自动默认值合并。
var YipayChannelExtra = ""

// PayMethods 为在线充值方式配置。
var PayMethods = []map[string]string{
	{
		"name":  "支付宝",
		"color": "rgba(var(--semi-blue-5), 1)",
		"type":  "alipay",
	},
	{
		"name":  "微信",
		"color": "rgba(var(--semi-green-5), 1)",
		"type":  "wxpay",
	},
	{
		"name":      "自定义1",
		"color":     "black",
		"type":      "custom1",
		"min_topup": "50",
	},
}

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	AmountDiscount map[int]float64 `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠
}

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:  []int{10, 20, 50, 100, 200, 500},
	AmountDiscount: map[int]float64{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

// GetPaymentSetting 返回支付配置对象。
func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}
