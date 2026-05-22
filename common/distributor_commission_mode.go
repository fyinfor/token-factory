package common

import "strings"

// 代理与邀请人分成模式（运营「代理设置」中配置，存 options.DistributorCommissionMode）。
const (
	DistributorCommissionModeTopup       = "topup"
	DistributorCommissionModeProfitShare = "profit_share"
)

// DistributorCommissionMode 当前模式：topup=充值分成；profit_share=利润分成（用量加价部分入账 aff_quota）。
var DistributorCommissionMode = DistributorCommissionModeTopup

func IsDistributorProfitShareMode() bool {
	return strings.EqualFold(strings.TrimSpace(DistributorCommissionMode), DistributorCommissionModeProfitShare)
}
