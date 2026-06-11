package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	pb "github.com/QuantumNous/new-api/proto/route"
)

// AdminTokenFactorySyncChannels 手动触发一次渠道快照同步到 TokenFactory。
// 管理员在「渠道变更后」或「首次配置后」可调用此端点，把当前 token-factory 渠道数据
// 通过 gRPC 推送到 TokenFactory 的 ChannelSnapshot 表。
// 鉴权：AdminAuth；JWT 用 ROOT_USER 兜底（uid=1, role=100）以满足 TokenFactory 端校验。
func AdminTokenFactorySyncChannels(c *gin.Context) {
	// 1) 读取全部可用渠道（脱敏：忽略 key，type != tokenfactory_open）
	channels, err := tfSyncExportChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "拉取渠道失败: " + err.Error(),
		})
		return
	}

	// 2) 转换为 ChannelSnapshot
	snapshots := make([]*pb.ChannelSnapshot, 0, len(channels))
	for _, ch := range channels {
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
		priceDiscount := float64(100)
		if ch.PriceDiscountPercent != nil {
			priceDiscount = *ch.PriceDiscountPercent
		}
		markupRate := float64(0)
		if ch.MarkupDiscountRate != nil {
			markupRate = *ch.MarkupDiscountRate
		}
		// 已关闭渠道仍同步元数据，但不推送模型列表与单价（不参与归类 / 路由）。
		modelsCSV := ch.Models
		var modelPrices map[string]float64
		if ch.Status == common.ChannelStatusEnabled {
			modelPrices = make(map[string]float64)
			for _, m := range strings.Split(ch.Models, ",") {
				m = strings.TrimSpace(m)
				if m == "" {
					continue
				}
				modelPrices[m] = service.ResolveChannelModelUnitPrice(ch, m)
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

	// 3) 用 ROOT_USER (uid=1) 签发 JWT
	jwt, err := common.IssueTokenFactoryJWT(1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "签发 JWT 失败: " + err.Error(),
		})
		return
	}

	// 4) 调用 gRPC 推送到 TokenFactory
	count, err := service.SyncChannelsToTF(jwt, snapshots)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "gRPC 同步失败: " + err.Error(),
			"pushed":  count,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "同步成功",
		"pushed":  count,
		"total":   len(snapshots),
	})
}

// tfSyncExportChannels 内部使用的渠道拉取函数（脱敏 + 排除 tokenfactory_open 类型）。
// 独立成函数而不是复用 TFOpenSyncExportChannels 是因为后者是 HTTP controller，
// 会做权限校验、写入日志等副作用，不适合在服务间同步场景下调用。
func tfSyncExportChannels() ([]*model.Channel, error) {
	var channels []*model.Channel
	q := model.DB.Model(&model.Channel{}).
		Omit("key").
		Where("type <> ?", constant.ChannelTypeTokenFactoryOpen).
		Order("id asc")
	if err := q.Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}
