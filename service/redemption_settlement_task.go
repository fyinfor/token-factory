package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const redemptionSettlementTickInterval = 5 * time.Minute

var (
	redemptionSettlementOnce    sync.Once
	redemptionSettlementRunning atomic.Bool
)

func StartRedemptionSettlementTask() {
	redemptionSettlementOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("redemption settlement task started: tick=%s", redemptionSettlementTickInterval))

			ticker := time.NewTicker(redemptionSettlementTickInterval)
			defer ticker.Stop()

			runRedemptionSettlementOnce()
			for range ticker.C {
				runRedemptionSettlementOnce()
			}
		})
	})
}

func runRedemptionSettlementOnce() {
	if !redemptionSettlementRunning.CompareAndSwap(false, true) {
		return
	}
	defer redemptionSettlementRunning.Store(false)

	if err := model.SettleExpiredRedemptions(0, common.RoleRootUser); err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("redemption settlement failed: %v", err))
	}
}
