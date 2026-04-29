package controller

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// tfOpenSyncExportRow 仅用于跨站同步导出，不包含渠道密钥。
type tfOpenSyncExportRow struct {
	ID                    int                `json:"id"`
	Name                  string             `json:"name"`
	Models                string             `json:"models"`
	Group                 string             `json:"group"`
	Status                int                `json:"status"`
	Type                  int                `json:"type"`
	ChannelNo             string             `json:"channel_no"`
	SupplierApplicationID int                `json:"supplier_application_id"`
	SupplierAlias         string             `json:"supplier_alias,omitempty"`
	ModelMapping          string             `json:"model_mapping,omitempty"`
	ModelPrice            map[string]float64 `json:"model_price,omitempty"`
	ModelRatio            map[string]float64 `json:"model_ratio,omitempty"`
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
	if len(appIDs) > 0 {
		type appRow struct {
			ID    int    `gorm:"column:id"`
			Alias string `gorm:"column:supplier_alias"`
		}
		var apps []appRow
		if err := model.DB.Table("supplier_applications").
			Select("id, supplier_alias").
			Where("id IN ?", appIDs).
			Scan(&apps).Error; err == nil {
			for _, a := range apps {
				aliasByAppID[a.ID] = strings.TrimSpace(a.Alias)
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
			SupplierApplicationID: ch.SupplierApplicationID,
			SupplierAlias:         aliasByAppID[ch.SupplierApplicationID],
			ModelMapping:          strings.TrimSpace(ch.GetModelMapping()),
			ModelPrice:            mp,
			ModelRatio:            mr,
		})
	}

	common.ApiSuccess(c, gin.H{"channels": out})
}
