package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	pb "github.com/QuantumNous/new-api/proto/route"
)

// ExportChannelsForTFSync 拉取本站可同步渠道（脱敏，排除 tokenfactory_open 类型）。
func ExportChannelsForTFSync() ([]*model.Channel, error) {
	var channels []*model.Channel
	var totalChannels int64
	model.DB.Model(&model.Channel{}).Count(&totalChannels)

	q := model.DB.Model(&model.Channel{}).
		Omit("key").
		Where("type <> ?", constant.ChannelTypeTokenFactoryOpen).
		Order("id asc")
	if err := q.Find(&channels).Error; err != nil {
		log.Printf("[SYS] ExportChannelsForTFSync: query failed, total=%d, err=%v", totalChannels, err)
		return nil, err
	}
	log.Printf("[SYS] ExportChannelsForTFSync: total_channels=%d, exported=%d (excluded type=%d)",
		totalChannels, len(channels), constant.ChannelTypeTokenFactoryOpen)
	return channels, nil
}

// BuildChannelSnapshotsForTF 将本地渠道转为 TokenFactory gRPC ChannelSnapshot。
// 仅包含模型/定价等路由元数据，不含渠道密钥。
func BuildChannelSnapshotsForTF(channels []*model.Channel) []*pb.ChannelSnapshot {
	snapshots := make([]*pb.ChannelSnapshot, 0, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		prio := int32(0)
		if ch.Priority != nil {
			prio = int32(*ch.Priority)
		}
		wt := int32(0)
		if ch.Weight != nil {
			wt = int32(*ch.Weight)
		}
		baseURL := ""
		if ch.BaseURL != nil {
			baseURL = *ch.BaseURL
		}
		priceDiscount := ch.ResolvedEffectiveCostPercent()
		markupRate := float64(0)
		if ch.MarkupDiscountRate != nil {
			markupRate = *ch.MarkupDiscountRate
		}
		modelsCSV := ch.Models
		var modelPrices map[string]float64
		if ch.Status == common.ChannelStatusEnabled {
			modelPrices = make(map[string]float64)
			for _, m := range strings.Split(ch.Models, ",") {
				m = strings.TrimSpace(m)
				if m == "" {
					continue
				}
				// 仅推送「已配置定价」的有效单价（与 /api/pricing 口径一致）；
				// 未配置定价的模型不写入，前端价格优模式展示为「—」并排在最后，
				// 不再像旧逻辑兜底为倍率 1（≈$2/1M）造成虚假定价。
				if price, ok := ResolveChannelModelConfiguredUnitPrice(ch, m); ok {
					modelPrices[m] = price
				}
			}
		} else {
			modelsCSV = ""
			modelPrices = nil
		}
		routeSlug := strings.TrimSpace(ch.RouteSlug)
		if routeSlug == "" {
			routeSlug = model.DefaultRouteSlugFromChannelID(int64(ch.Id))
		}
		snapshots = append(snapshots, &pb.ChannelSnapshot{
			Id:                   int32(ch.Id),
			Name:                 ch.Name,
			Type:                 int32(ch.Type),
			Models:               modelsCSV,
			Group:                ch.Group,
			Status:               int32(ch.Status),
			Priority:             prio,
			Weight:               wt,
			BaseUrl:              baseURL,
			Balance:              0,
			ChannelNo:            ch.ChannelNo,
			RouteSlug:            routeSlug,
			SupplierAlias:        ch.SupplierType,
			ProviderSlug:         strings.ToLower(strings.TrimSpace(ch.SupplierType)),
			PriceDiscountPercent: priceDiscount,
			MarkupDiscountRate:   markupRate,
			ModelPrices:          modelPrices,
		})
	}
	return snapshots
}

// SyncChannelsToTokenFactory 将本站渠道快照推送到 TokenFactory（按 TOKENFACTORY_SITE_KEY 隔离）。
// TokenFactory 端仅更新 channel_snapshots 表，不会修改归类权重等路由配置。
func SyncChannelsToTokenFactory() (pushed int, total int, err error) {
	if !common.TokenFactoryRouteEnabled() {
		return 0, 0, fmt.Errorf("TokenFactory route not enabled")
	}

	channels, err := ExportChannelsForTFSync()
	if err != nil {
		return 0, 0, err
	}
	snapshots := BuildChannelSnapshotsForTF(channels)
	if len(snapshots) == 0 {
		return 0, 0, nil
	}

	jwt, err := common.IssueTokenFactoryJWT(1, 100)
	if err != nil {
		return 0, len(snapshots), fmt.Errorf("issue JWT: %w", err)
	}

	count, err := SyncChannelsToTF(jwt, snapshots)
	if err != nil {
		return count, len(snapshots), err
	}
	return count, len(snapshots), nil
}
