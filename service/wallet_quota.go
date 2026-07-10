package service

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/model"
)

func walletGiftOffset(relayInfo *relaycommon.RelayInfo) int {
	return WalletGiftOffset(relayInfo)
}

func WalletGiftOffset(relayInfo *relaycommon.RelayInfo) int {
	if relayInfo == nil {
		return 0
	}
	return relayInfo.WalletGiftConsumed
}

func recordWalletUsedQuota(relayInfo *relaycommon.RelayInfo, userID, quota int) {
	if userID <= 0 || quota <= 0 {
		return
	}
	model.UpdateUserUsedQuotaAndRequestCountWithGiftOffset(userID, quota, walletGiftOffset(relayInfo))
}
