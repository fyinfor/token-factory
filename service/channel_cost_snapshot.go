package service

import "github.com/QuantumNous/new-api/model"

func taskBillingContextEffectiveCostPercent(bc *model.TaskBillingContext, channelID int) float64 {
	if bc != nil {
		if bc.EffectiveCostPercent != nil {
			return *bc.EffectiveCostPercent
		}
		if bc.ChannelPriceDiscountPercent > 0 {
			return bc.ChannelPriceDiscountPercent
		}
	}
	if channelID > 0 {
		return model.ResolveChannelEffectiveCostPercent(channelID)
	}
	return 100
}
