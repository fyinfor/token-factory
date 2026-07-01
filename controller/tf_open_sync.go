package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// coalesceStr 返回第一个非空字符串，若均为空则返回空串。
func coalesceStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// tfOpenSyncExportRow 仅用于跨站同步导出，不包含渠道密钥。
type tfOpenSyncExportRow struct {
	ID                    int                `json:"id"`
	Name                  string             `json:"name"`
	Models                string             `json:"models"`
	Group                 string             `json:"group"`
	Status                int                `json:"status"`
	Type                  int                `json:"type"`
	ChannelNo             string             `json:"channel_no"`
	RouteSlug             string             `json:"route_slug,omitempty"`
	SupplierApplicationID int                `json:"supplier_application_id"`
	SupplierAlias         string             `json:"supplier_alias,omitempty"`
	SupplierType          string             `json:"supplier_type,omitempty"`
	CompanyLogoURL        string             `json:"company_logo_url,omitempty"`
	PriceDiscountPercent  *float64           `json:"price_discount_percent,omitempty"`
	OperatingCostPercent  *float64           `json:"operating_cost_percent,omitempty"`
	MarkupDiscountRate    *float64           `json:"markup_discount_rate,omitempty"`
	ModelMapping          string             `json:"model_mapping,omitempty"`
	ModelPrice            map[string]float64 `json:"model_price,omitempty"`
	ModelRatio            map[string]float64 `json:"model_ratio,omitempty"`
}

type tfOpenSyncChannelTestRequest struct {
	Model                 string `json:"model"`
	EndpointType          string `json:"endpoint_type,omitempty"`
	Stream                bool   `json:"stream,omitempty"`
	UpstreamChannelID     int    `json:"upstream_channel_id,omitempty"`
	UpstreamRouteSlug     string `json:"upstream_route_slug,omitempty"`
	UpstreamSupplierAlias string `json:"upstream_supplier_alias,omitempty"`
	UpstreamChannelNo     string `json:"upstream_channel_no,omitempty"`
}

type tfOpenSyncChannelTestResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Time    float64 `json:"time"`
	Model   string  `json:"model,omitempty"`
}

func authorizeTFOpenSyncExport(c *gin.Context) bool {
	secretEnv := strings.TrimSpace(os.Getenv("TOKENFACTORY_OPEN_SYNC_SECRET"))
	hdr := strings.TrimSpace(c.GetHeader("X-TokenFactory-Open-Sync-Secret"))
	if secretEnv != "" && hdr != "" && hdr == secretEnv {
		return true
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = strings.TrimSpace(auth[7:])
	}
	// 优先支持普通 API 令牌（sk- 前缀），方便上游发放非管理员同步 key。
	tokenKey := strings.TrimPrefix(auth, "sk-")
	if tokenKey != "" {
		if _, err := model.ValidateUserToken(tokenKey); err == nil {
			return true
		}
	}
	// 兼容 access token（不再强制管理员角色）。
	return model.ValidateAccessToken(auth) != nil
}

// TFOpenSyncExportChannels 供子站 TokenFactoryOpen 同步：返回全站渠道（脱敏）及渠道级定价/倍率。
// 鉴权：环境变量 TOKENFACTORY_OPEN_SYNC_SECRET + 请求头；或 Bearer 携带可用普通 API 令牌（sk-）；或有效 access token。
func TFOpenSyncExportChannels(c *gin.Context) {
	if !authorizeTFOpenSyncExport(c) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权导出：请使用同步密钥（X-TokenFactory-Open-Sync-Secret）或 Bearer 携带可用令牌（sk- 或 access token）",
		})
		return
	}

	var channels []*model.Channel
	q := model.DB.Model(&model.Channel{}).
		Omit("key").
		Where("type <> ?", constant.ChannelTypeTokenFactoryOpen).
		Order("supplier_application_id asc, channel_no asc, id asc")
	if err := q.Find(&channels).Error; err != nil {
		common.SysError("tf_open_sync export: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "查询渠道失败"})
		return
	}

	appIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, ch := range channels {
		if ch != nil && ch.SupplierApplicationID > 0 {
			if _, ok := seen[ch.SupplierApplicationID]; !ok {
				seen[ch.SupplierApplicationID] = struct{}{}
				appIDs = append(appIDs, ch.SupplierApplicationID)
			}
		}
	}
	aliasByAppID := make(map[int]string, len(appIDs))
	logoByAppID := make(map[int]string, len(appIDs))
	supplierTypeByAppID := make(map[int]string, len(appIDs))
	if len(appIDs) > 0 {
		type appRow struct {
			ID           int    `gorm:"column:id"`
			Alias        string `gorm:"column:supplier_alias"`
			LogoURL      string `gorm:"column:company_logo_url"`
			SupplierType string `gorm:"column:supplier_type"`
		}
		var apps []appRow
		if err := model.DB.Table("supplier_applications").
			Select("id, supplier_alias, company_logo_url, supplier_type").
			Where("id IN ?", appIDs).
			Scan(&apps).Error; err == nil {
			for _, a := range apps {
				aliasByAppID[a.ID] = strings.TrimSpace(a.Alias)
				logoByAppID[a.ID] = strings.TrimSpace(a.LogoURL)
				supplierTypeByAppID[a.ID] = strings.TrimSpace(a.SupplierType)
			}
		}
	}

	priceAll := ratio_setting.GetChannelModelPriceCopy()
	ratioAll := ratio_setting.GetChannelModelRatioCopy()

	out := make([]tfOpenSyncExportRow, 0, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		idStr := strconv.Itoa(ch.Id)
		mp := priceAll[idStr]
		mr := ratioAll[idStr]
		if len(mp) == 0 {
			mp = nil
		}
		if len(mr) == 0 {
			mr = nil
		}
		// 仅导出该渠道 models 列表中出现的模型，控制体积
		modelSet := make(map[string]struct{})
		for _, m := range ch.GetModels() {
			mk := ratio_setting.FormatMatchingModelName(m)
			if mk != "" {
				modelSet[mk] = struct{}{}
			}
		}
		if len(modelSet) > 0 {
			filteredP := make(map[string]float64)
			filteredR := make(map[string]float64)
			for mk := range modelSet {
				if mp != nil {
					if v, ok := mp[mk]; ok {
						filteredP[mk] = v
					}
				}
				if mr != nil {
					if v, ok := mr[mk]; ok {
						filteredR[mk] = v
					}
				}
			}
			if len(filteredP) == 0 {
				filteredP = nil
			}
			if len(filteredR) == 0 {
				filteredR = nil
			}
			mp, mr = filteredP, filteredR
		}

		out = append(out, tfOpenSyncExportRow{
			ID:                    ch.Id,
			Name:                  ch.Name,
			Models:                ch.Models,
			Group:                 ch.Group,
			Status:                ch.Status,
			Type:                  ch.Type,
			ChannelNo:             strings.TrimSpace(ch.ChannelNo),
			RouteSlug:             strings.TrimSpace(ch.RouteSlug),
			SupplierApplicationID: ch.SupplierApplicationID,
			SupplierAlias:         aliasByAppID[ch.SupplierApplicationID],
			SupplierType:          coalesceStr(supplierTypeByAppID[ch.SupplierApplicationID], strings.TrimSpace(ch.SupplierType)),
			CompanyLogoURL:        coalesceStr(logoByAppID[ch.SupplierApplicationID], strings.TrimSpace(ch.CompanyLogoURL)),
			PriceDiscountPercent:  ch.PriceDiscountPercent,
			OperatingCostPercent:  ch.OperatingCostPercent,
			MarkupDiscountRate:    ch.MarkupDiscountRate,
			ModelMapping:          strings.TrimSpace(ch.GetModelMapping()),
			ModelPrice:            mp,
			ModelRatio:            mr,
		})
	}

	common.ApiSuccess(c, gin.H{"channels": out})
}

func resolveTFOpenSyncChannelTestChannel(req tfOpenSyncChannelTestRequest) (*model.Channel, error) {
	modelName := strings.TrimSpace(req.Model)
	modelNameForLookup := modelName
	if strings.HasSuffix(modelNameForLookup, ratio_setting.CompactModelSuffix) {
		modelNameForLookup = strings.TrimSuffix(modelNameForLookup, ratio_setting.CompactModelSuffix)
	}
	if slug := strings.TrimSpace(req.UpstreamRouteSlug); slug != "" {
		var candidates []model.Channel
		if err := model.DB.Select("id", "models").
			Where("route_slug = ?", slug).
			Find(&candidates).Error; err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("未找到上游 route_slug: %s", slug)
		}
		for i := range candidates {
			if modelNameForLookup == "" || model.ChannelModelsRawContains(candidates[i].Models, modelNameForLookup) {
				return model.GetChannelById(candidates[i].Id, true)
			}
		}
		return nil, fmt.Errorf("route_slug %s 未绑定模型 %s", slug, modelName)
	}

	if req.UpstreamChannelID > 0 {
		ch, err := model.GetChannelById(req.UpstreamChannelID, true)
		if err != nil {
			return nil, fmt.Errorf("未找到上游渠道 ID %d: %w", req.UpstreamChannelID, err)
		}
		if modelNameForLookup != "" && !model.ChannelModelsRawContains(ch.Models, modelNameForLookup) {
			return nil, fmt.Errorf("上游渠道 ID %d 未绑定模型 %s", req.UpstreamChannelID, modelName)
		}
		return ch, nil
	}

	alias := strings.TrimSpace(req.UpstreamSupplierAlias)
	channelNo := strings.TrimSpace(req.UpstreamChannelNo)
	if alias != "" && channelNo != "" {
		channelID, err := model.FindChannelIDBySupplierAliasAndNo(alias, channelNo)
		if err != nil {
			return nil, err
		}
		return model.GetChannelById(channelID, true)
	}
	return nil, fmt.Errorf("缺少上游渠道身份")
}

func TFOpenSyncChannelTest(c *gin.Context) {
	if !authorizeTFOpenSyncExport(c) {
		c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
			Success: false,
			Message: "无权测试：请使用同步密钥（X-TokenFactory-Open-Sync-Secret）或 Bearer 携带可用令牌（sk- 或 access token）",
		})
		return
	}

	var req tfOpenSyncChannelTestRequest
	if err := common.DecodeJson(io.LimitReader(c.Request.Body, 1024*1024), &req); err != nil {
		c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
			Success: false,
			Message: "解析测试请求失败: " + err.Error(),
		})
		return
	}

	channel, err := resolveTFOpenSyncChannelTestChannel(req)
	if err != nil {
		c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
			Success: false,
			Message: "定位上游渠道失败: " + err.Error(),
		})
		return
	}
	if channel.Type == constant.ChannelTypeTokenFactoryOpen {
		c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
			Success: false,
			Message: "上游测试委托不能继续委托 TokenFactoryOpen 渠道",
		})
		return
	}

	tik := time.Now()
	result := testChannel(channel, strings.TrimSpace(req.Model), strings.TrimSpace(req.EndpointType), req.Stream)
	milliseconds := time.Since(tik).Milliseconds()
	consumedTime := float64(milliseconds) / 1000.0
	modelForRecord := modelNameForChannelTestRecord(channel, req.Model, result)
	if result.localErr != nil {
		persistChannelTestResult(channel, modelForRecord, false, milliseconds, result.localErr.Error())
		c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
			Success: false,
			Message: result.localErr.Error(),
			Time:    consumedTime,
			Model:   modelForRecord,
		})
		return
	}
	if result.tokenFactoryError != nil {
		persistChannelTestResult(channel, modelForRecord, false, milliseconds, result.tokenFactoryError.Error())
		c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
			Success: false,
			Message: result.tokenFactoryError.Error(),
			Time:    consumedTime,
			Model:   modelForRecord,
		})
		return
	}
	persistChannelTestResult(channel, modelForRecord, true, milliseconds, "")
	c.JSON(http.StatusOK, tfOpenSyncChannelTestResponse{
		Success: true,
		Message: "",
		Time:    consumedTime,
		Model:   modelForRecord,
	})
}
