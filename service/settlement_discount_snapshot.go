package service

import (
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

// appendSettlementDiscountSnapshots 写入结算对账所需的折扣快照（成本折扣、经营成本、经营折扣、加价折扣、销售折扣）。
func appendSettlementDiscountSnapshots(relayInfo *relaycommon.RelayInfo, other map[string]interface{}) {
	if other == nil {
		return
	}
	rawDisc, operatingCost, operatingDiscount, markupDisc := resolveSettlementDiscountPercents(relayInfo)
	other["price_discount_percent"] = rawDisc
	other["operating_cost_percent"] = operatingCost
	other["channel_price_discount_percent"] = operatingDiscount
	other["markup_discount_rate"] = markupDisc
	other["sales_discount_percent"] = model.SalesDiscountPercent(rawDisc, operatingCost, markupDisc)
}

func appendSettlementDiscountSnapshotsFromPriceData(channelID int, priceData types.PriceData, other map[string]interface{}) {
	if other == nil {
		return
	}
	rawDisc := priceData.RawPriceDiscountPercent
	operatingCost := priceData.OperatingCostPercent
	operatingDiscount := priceData.CostDiscountPercent
	if priceData.ChannelPriceDiscount != nil {
		operatingDiscount = *priceData.ChannelPriceDiscount
	}
	if rawDisc == 0 && operatingCost == 0 && operatingDiscount == 0 {
		rawDisc = model.ResolveChannelPriceDiscountPercent(channelID)
		operatingCost = model.ResolveChannelOperatingCostPercent(channelID)
		operatingDiscount = model.EffectiveCostPercent(rawDisc, operatingCost)
	} else if operatingDiscount == 0 {
		operatingDiscount = model.EffectiveCostPercent(rawDisc, operatingCost)
	}
	markupDisc := priceData.MarkupDiscountPercent
	other["price_discount_percent"] = rawDisc
	other["operating_cost_percent"] = operatingCost
	other["channel_price_discount_percent"] = operatingDiscount
	other["markup_discount_rate"] = markupDisc
	other["sales_discount_percent"] = model.SalesDiscountPercent(rawDisc, operatingCost, markupDisc)
}

func resolveSettlementDiscountPercents(relayInfo *relaycommon.RelayInfo) (rawDisc, operatingCost, operatingDiscount, markupDisc float64) {
	if relayInfo != nil {
		rawDisc = relayInfo.PriceData.RawPriceDiscountPercent
		operatingCost = relayInfo.PriceData.OperatingCostPercent
		operatingDiscount = relayInfo.PriceData.CostDiscountPercent
		if relayInfo.PriceData.ChannelPriceDiscount != nil {
			operatingDiscount = *relayInfo.PriceData.ChannelPriceDiscount
		}
		markupDisc = relayInfo.PriceData.MarkupDiscountPercent
	}
	chID := 0
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		chID = relayInfo.ChannelId
	}
	if rawDisc == 0 && operatingCost == 0 && operatingDiscount == 0 {
		rawDisc = model.ResolveChannelPriceDiscountPercent(chID)
		operatingCost = model.ResolveChannelOperatingCostPercent(chID)
		operatingDiscount = model.EffectiveCostPercent(rawDisc, operatingCost)
	} else if operatingDiscount == 0 {
		operatingDiscount = model.EffectiveCostPercent(rawDisc, operatingCost)
	}
	return rawDisc, operatingCost, operatingDiscount, markupDisc
}
