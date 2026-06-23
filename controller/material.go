package controller

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// 允许上传的图片扩展名。
var allowedMaterialImageExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
	".bmp":  true,
}

// materialAssetResponse 素材列表/上传返回结构。
type materialAssetResponse struct {
	Id        int    `json:"id"`
	AssetId   string `json:"asset_id"`
	AssetURI  string `json:"asset_uri"` // asset://asset-xxxx，用于复制替换图片资源地址
	Name      string `json:"name"`
	AssetType string `json:"asset_type"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

func toMaterialAssetResponse(a *model.MaterialAsset) materialAssetResponse {
	return materialAssetResponse{
		Id:        a.Id,
		AssetId:   a.AssetId,
		AssetURI:  "asset://" + a.AssetId,
		Name:      a.Name,
		AssetType: a.AssetType,
		URL:       a.URL,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
	}
}

// GetMaterialConfig 返回素材库前端所需配置（启用状态、上传大小上限、合规协议中英文文案及详情）。
func GetMaterialConfig(c *gin.Context) {
	setting := operation_setting.GetSeedanceSetting()
	maxSize := setting.MaxImageSizeMB
	if maxSize <= 0 {
		maxSize = 10
	}
	common.ApiSuccess(c, gin.H{
		"enabled":             setting.Enabled,
		"ready":               operation_setting.IsSeedanceReady(),
		"max_image_size_mb":   maxSize,
		"agreement_zh":        operation_setting.GetPortraitAgreement("zh"),
		"agreement_en":        operation_setting.GetPortraitAgreement("en"),
		"agreement_detail_zh": setting.AgreementDetailZh,
		"agreement_detail_en": setting.AgreementDetailEn,
	})
}

// GetMaterialGroup 查询当前用户的素材分组，不存在时返回 null。
func GetMaterialGroup(c *gin.Context) {
	userId := c.GetInt("id")
	group, err := model.GetMaterialGroupByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if group == nil {
		common.ApiSuccess(c, nil)
		return
	}
	common.ApiSuccess(c, gin.H{
		"id":         group.Id,
		"group_name": group.GroupName,
		"group_id":   group.GroupId,
	})
}

// ListMaterialAssets 分页查询当前用户的素材列表。
func ListMaterialAssets(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)

	group, err := model.GetMaterialGroupByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groupId := ""
	if group != nil {
		groupId = group.GroupId
	}

	assets, total, err := model.ListMaterialAssets(userId, groupId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 对处于 Pending 状态的素材做一次best-effort状态刷新（最多刷新 10 条，避免阻塞）。
	refreshed := 0
	items := make([]materialAssetResponse, 0, len(assets))
	for _, a := range assets {
		if a.Status == "Pending" && refreshed < 10 && operation_setting.IsSeedanceReady() {
			if info, e := service.MaterialGetAsset(a.AssetId); e == nil && info != nil && info.Status != "" && info.Status != a.Status {
				_ = model.UpdateMaterialAssetStatus(a.Id, info.Status)
				a.Status = info.Status
			}
			refreshed++
		}
		items = append(items, toMaterialAssetResponse(a))
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// ensureMaterialGroup 获取或创建当前用户的素材分组（首次上传时自动建组）。
func ensureMaterialGroup(userId int) (*model.MaterialGroup, error) {
	group, err := model.GetMaterialGroupByUserId(userId)
	if err != nil {
		return nil, err
	}
	if group != nil {
		return group, nil
	}
	groupName := model.BuildMaterialGroupName(userId)
	upstreamGroupId, err := service.MaterialCreateAssetGroup(groupName, "虚拟人像素材组")
	if err != nil {
		return nil, err
	}
	group = &model.MaterialGroup{
		UserId:    userId,
		GroupName: groupName,
		GroupId:   upstreamGroupId,
	}
	if err := model.CreateMaterialGroup(group); err != nil {
		return nil, err
	}
	return group, nil
}

// UploadMaterial 本地图片上传素材：校验 -> 生成公网URL -> 自动建组 -> 调用素材库上传 -> 落库。
func UploadMaterial(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}

	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		common.ApiErrorMsg(c, "用户无效")
		return
	}
	if user.Role < common.FileUploadPermission {
		common.ApiErrorMsg(c, "无上传权限")
		return
	}

	// 协议同意校验（前端拦截 + 后端兜底）。
	if strings.ToLower(strings.TrimSpace(c.PostForm("agreed"))) != "true" {
		common.ApiErrorMsg(c, "请先阅读并勾选同意虚拟人像合规协议")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请选择文件字段 file")
		return
	}

	// 扩展名校验：仅支持图片。
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedMaterialImageExt[ext] {
		common.ApiErrorMsg(c, "仅支持上传图片格式（jpg/jpeg/png/webp/gif/bmp）")
		return
	}

	// 大小校验：单图 < 配置上限（默认 10MB）。
	maxSizeMB := operation_setting.GetSeedanceSetting().MaxImageSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if file.Size > int64(maxSizeMB)*1024*1024 {
		common.ApiErrorMsg(c, "图片超过大小限制（最大 "+strconv.Itoa(maxSizeMB)+"MB）")
		return
	}

	// 上传文件获取公网 URL（复用通用上传：本地存储或 OSS）。
	ossCfg := operation_setting.GetOssSetting()
	if !ossCfg.Enabled {
		common.ApiErrorMsg(c, "文件上传未启用，请先在运营设置中启用文件上传")
		return
	}
	var publicURL string
	if ossCfg.StorageType == operation_setting.StorageTypeLocal {
		publicURL, err = service.LocalUploadMultipartFile(file, userId)
	} else {
		if !operation_setting.IsOssUploadReady() {
			common.ApiErrorMsg(c, service.ErrOssNotConfigured.Error())
			return
		}
		publicURL, err = service.OssUploadMultipartFile(file, userId)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 自动建组。
	group, err := ensureMaterialGroup(userId)
	if err != nil {
		common.ApiErrorMsg(c, "创建素材分组失败: "+err.Error())
		return
	}

	// 调用素材库上传素材。
	assetName := strings.TrimSpace(file.Filename)
	if assetName == "" {
		assetName = "portrait"
	}
	assetId, err := service.MaterialCreateAsset(group.GroupId, publicURL, assetName, service.MaterialAssetTypeImage)
	if err != nil {
		common.ApiErrorMsg(c, "素材上传失败: "+err.Error())
		return
	}

	// 查询素材状态（best-effort）。
	status := "Pending"
	if info, e := service.MaterialGetAsset(assetId); e == nil && info != nil && info.Status != "" {
		status = info.Status
	}

	asset := &model.MaterialAsset{
		UserId:    userId,
		GroupId:   group.GroupId,
		AssetId:   assetId,
		Name:      assetName,
		AssetType: service.MaterialAssetTypeImage,
		URL:       publicURL,
		Status:    status,
	}
	if err := model.CreateMaterialAsset(asset); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}
