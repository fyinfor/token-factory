package common

func GetTrustQuota() int {
	return int(10 * QuotaPerUnit)
}

// QuotaFromUSD 将美元金额换算为站内额度整数（与充值入账、TopUp.Money * QuotaPerUnit 的策略一致：向零截断）。
// 用于运营后台「注册类邀请奖励」等以美元配置、以 quota 存储的场景。
func QuotaFromUSD(usd float64) int {
	if usd <= 0 || QuotaPerUnit <= 0 {
		return 0
	}
	return int(usd * QuotaPerUnit)
}
