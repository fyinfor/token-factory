/**
此文件为旧版支付设置文件，如需增加新的参数、变量等，请在 payment_setting.go 中添加
This file is the old version of the payment settings file. If you need to add new parameters, variables, etc., please add them in payment_setting.go
*/

package operation_setting

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// UpdatePayMethodsByJsonString 兼容旧版调用，更新支付方式配置。
func UpdatePayMethodsByJsonString(jsonString string) error {
	PayMethods = make([]map[string]string, 0)
	return common.Unmarshal([]byte(jsonString), &PayMethods)
}

// PayMethods2JsonString 兼容旧版调用，输出支付方式 JSON。
func PayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(PayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// ContainsPayMethod 检查支付方式是否存在。
func ContainsPayMethod(method string) bool {
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method {
			return true
		}
	}
	return false
}

// IsLimitedEpayPayMethod 判断是否为需应用单笔充值上限的在线支付方式（支付宝/微信/PayPal）。
func IsLimitedEpayPayMethod(method string) bool {
	m := strings.TrimSpace(method)
	lower := strings.ToLower(m)
	return lower == "alipay" || lower == "wxpay" || lower == "paypal" || strings.HasPrefix(strings.ToUpper(m), "PP_")
}

// GetPayMethodMaxTopup 返回指定支付方式的最大充值数量；0 表示不限制。
func GetPayMethodMaxTopup(method string) int64 {
	if !IsLimitedEpayPayMethod(method) {
		return 0
	}
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method {
			if maxStr, ok := payMethod["max_topup"]; ok && strings.TrimSpace(maxStr) != "" {
				if maxVal, err := strconv.ParseInt(strings.TrimSpace(maxStr), 10, 64); err == nil && maxVal > 0 {
					return maxVal
				}
			}
			break
		}
	}
	return DefaultEpayMaxTopUp
}
