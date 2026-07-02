package helper

import "github.com/QuantumNous/new-api/model"

func resolveChannelCostPercents(channelID int) (rawPriceDiscount, operatingCost, effectiveCost float64) {
	rawPriceDiscount = model.ResolveChannelPriceDiscountPercent(channelID)
	operatingCost = model.ResolveChannelOperatingCostPercent(channelID)
	effectiveCost = model.EffectiveCostPercent(rawPriceDiscount, operatingCost)
	return rawPriceDiscount, operatingCost, effectiveCost
}
