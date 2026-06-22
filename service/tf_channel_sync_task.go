package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
)

var (
	tfSiteSyncOnce    sync.Once
	tfSiteSyncRunning atomic.Bool
)

// StartTokenFactoryChannelSyncTask 启动定时站点数据同步（渠道 + 用户，同频率）。
// 仅 master 节点、且智能路由已启用时运行；按 TOKENFACTORY_SITE_KEY 隔离推送。
func StartTokenFactoryChannelSyncTask() {
	tfSiteSyncOnce.Do(func() {
		if !common.TokenFactoryChannelSyncEnabled() {
			return
		}
		if !common.IsMasterNode {
			return
		}

		interval := common.TokenFactoryChannelSyncInterval()
		gopool.Go(func() {
			siteKey := common.TokenFactorySiteKey()
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"TokenFactory site sync task started: site_key=%s interval=%s endpoint=%s (channels+users)",
				siteKey, interval, common.TokenFactoryGRPCEndpoint(),
			))

			// 启动后延迟 30s 再首次同步，避免与进程初始化抢资源。
			time.Sleep(30 * time.Second)
			runTokenFactorySiteSyncOnce()

			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				runTokenFactorySiteSyncOnce()
			}
		})
	})
}

func runTokenFactorySiteSyncOnce() {
	if !tfSiteSyncRunning.CompareAndSwap(false, true) {
		return
	}
	defer tfSiteSyncRunning.Store(false)

	ctx := context.Background()
	siteKey := common.TokenFactorySiteKey()

	runTokenFactoryChannelSyncOnce(ctx, siteKey)
	runTokenFactoryUserSyncOnce(ctx, siteKey)
}

func runTokenFactoryChannelSyncOnce(ctx context.Context, siteKey string) {
	pushed, total, err := SyncChannelsToTokenFactory()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf(
			"TokenFactory channel sync failed (site_key=%s): %v (pushed=%d total=%d)",
			siteKey, err, pushed, total,
		))
		return
	}
	if total == 0 {
		logger.LogInfo(ctx, fmt.Sprintf(
			"TokenFactory channel sync skipped: no channels (site_key=%s)",
			siteKey,
		))
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf(
		"TokenFactory channel sync ok: site_key=%s pushed=%d total=%d",
		siteKey, pushed, total,
	))
}

func runTokenFactoryUserSyncOnce(ctx context.Context, siteKey string) {
	pushed, total, err := SyncUsersToTokenFactory()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf(
			"TokenFactory user sync failed (site_key=%s): %v (pushed=%d total=%d)",
			siteKey, err, pushed, total,
		))
		return
	}
	if total == 0 {
		logger.LogInfo(ctx, fmt.Sprintf(
			"TokenFactory user sync skipped: no users (site_key=%s)",
			siteKey,
		))
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf(
		"TokenFactory user sync ok: site_key=%s pushed=%d total=%d",
		siteKey, pushed, total,
	))
}
