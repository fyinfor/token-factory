package helper

import (
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

// effectiveMarkupDiscountPercent 返回 relay 计费使用的加价折扣率（百分数）：利润分成链下优先被邀请人覆盖，否则为渠道默认。
func effectiveMarkupDiscountPercent(c *gin.Context, info *relaycommon.RelayInfo, channelID int, originModel string) float64 {
	return effectiveMarkupDiscountPercentWithBase(
		c,
		info,
		channelID,
		originModel,
		model.ResolveChannelMarkupDiscountRate(channelID),
	)
}

func effectiveMarkupDiscountPercentWithBase(c *gin.Context, info *relaycommon.RelayInfo, channelID int, originModel string, base float64) float64 {
	if info == nil || info.UserId <= 0 {
		return base
	}
	return model.ResolveEffectiveMarkupDiscountPercentForInviteeBillingWithBase(info.UserId, channelID, originModel, base)
}
