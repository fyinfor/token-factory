package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// ResolveEffectiveMarkupDiscountPercentForInviteeBilling 返回本次计费应使用的加价折扣率（百分数）。
// 非利润分成模式、或用户非分销商邀请链下被邀请人时，回退为渠道默认加价折扣率。
func ResolveEffectiveMarkupDiscountPercentForInviteeBilling(inviteeUserId, channelId int, originModelName string) float64 {
	base := ResolveChannelMarkupDiscountRate(channelId)
	if !common.IsDistributorProfitShareMode() || inviteeUserId <= 0 || channelId <= 0 {
		return base
	}
	modelName := strings.TrimSpace(originModelName)
	if modelName == "" {
		return base
	}
	u, err := GetUserById(inviteeUserId, false)
	if err != nil || u == nil || u.InviterId <= 0 {
		return base
	}
	inv, err := GetUserById(u.InviterId, false)
	if err != nil || inv == nil || !UserIsDistributor(inv) {
		return base
	}
	var rel AffInviteRelation
	err = DB.Where("inviter_id = ? AND invitee_user_id = ?", u.InviterId, inviteeUserId).First(&rel).Error
	if err != nil {
		return base
	}
	list, perr := parseInviteeModelMarkupDiscountRates(rel.ModelMarkupDiscountRate)
	if perr != nil || len(list) == 0 {
		return base
	}
	m := inviteeModelMarkupDiscountRateMap(list)
	if v, ok := m[inviteeModelMarkupKey(channelId, modelName)]; ok {
		return clampChannelMarkupDiscountRate(v)
	}
	return base
}

// ApplyInviteeMarkupToPricingAPIForUser 在利润分成模式下，将登录被邀请用户的模型×渠道加价覆盖到定价接口展示数据，
// 并按被邀请人加价折扣率重算视频/图片分档展示价（video_flat_clip_hint / image_per_image_hint）。
func ApplyInviteeMarkupToPricingAPIForUser(inviteeUserId int, pricingData []PricingAPIItem) {
	if inviteeUserId <= 0 || !common.IsDistributorProfitShareMode() || len(pricingData) == 0 {
		return
	}
	for i := range pricingData {
		modelName := strings.TrimSpace(pricingData[i].ModelName)
		if modelName == "" {
			continue
		}
		for j := range pricingData[i].ChannelList {
			ch := &pricingData[i].ChannelList[j]
			if ch.ChannelID <= 0 {
				continue
			}
			ch.MarkupDiscountRate = ResolveEffectiveMarkupDiscountPercentForInviteeBilling(inviteeUserId, ch.ChannelID, modelName)
		}
		// 定价 data 按「模型×单渠道」展开，重算分档 hint 使首页卡片/侧栏与实扣一致。
		if len(pricingData[i].ChannelList) == 1 {
			ch := pricingData[i].ChannelList[0]
			pricingData[i].VideoFlatClipHint = BuildVideoFlatClipHint(ch.ChannelID, modelName, ch.PriceDiscountPercent, ch.MarkupDiscountRate)
			pricingData[i].ImagePerImageHint = BuildImagePerImageHint(ch.ChannelID, modelName, ch.PriceDiscountPercent, ch.MarkupDiscountRate)
		}
	}
}
