package controller

import (
	"fmt"
	"net/url"
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

// 允许上传的视频扩展名。
var allowedMaterialVideoExt = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".webm": true,
	".mkv":  true,
	".avi":  true,
	".m4v":  true,
}

// detectMaterialAssetType 根据文件扩展名推断素材类型（仅图片/视频）。
// 返回值：AssetType 枚举值（Image/Video）与是否为受支持类型。
func detectMaterialAssetType(ext string) (string, bool) {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if allowedMaterialImageExt[ext] {
		return service.MaterialAssetTypeImage, true
	}
	if allowedMaterialVideoExt[ext] {
		return service.MaterialAssetTypeVideo, true
	}
	return "", false
}

// materialAssetNeedsUpstreamRefresh 判断本地记录是否仍需向上游同步 URL/状态。
func materialAssetNeedsUpstreamRefresh(asset *model.MaterialAsset) bool {
	if asset == nil {
		return false
	}
	if asset.Status == service.MaterialStatusPending {
		return true
	}
	return service.IsLocalMaterialUploadURL(asset.URL)
}

// refreshMaterialAssetFromUpstream 用 GetAsset 结果刷新本地素材，返回是否有变更。
func refreshMaterialAssetFromUpstream(asset *model.MaterialAsset, info *service.MaterialAssetResult) bool {
	if asset == nil || info == nil {
		return false
	}
	oldURL := asset.URL
	newStatus := service.NormalizeMaterialStatus(info.Status)
	newURL := strings.TrimSpace(info.URL)
	newAssetType := strings.TrimSpace(info.AssetType)

	changed := (newStatus != "" && newStatus != asset.Status) ||
		(newURL != "" && newURL != asset.URL) ||
		(newAssetType != "" && newAssetType != asset.AssetType)
	if !changed {
		return false
	}

	_ = model.UpdateMaterialAssetInfo(asset.Id, newStatus, newURL, newAssetType)
	if newURL != "" && newURL != oldURL {
		_ = service.CleanupLocalUploadByURL(oldURL)
		asset.URL = newURL
	}
	if newStatus != "" {
		asset.Status = newStatus
	}
	if newAssetType != "" {
		asset.AssetType = newAssetType
	}
	return true
}

// finalizeMaterialUpload 上传后置逻辑：上游 CreateAsset 成功后，
// 必须同步等待 GetAsset 拉取完整素材信息，再持久化关键字段（URL/AssetType/Status/GroupId），
// 最后清理本地临时文件。
//   - fallbackGroupId/fallbackType/fallbackURL：接口缺失字段时的回退值。
//   - tempLocalURL：本地上传产生的临时公网 URL（在线链接上传时传空，无需清理）。
func finalizeMaterialUpload(userId int, fallbackGroupId, assetId, name, fallbackType, fallbackURL, tempLocalURL string) (*model.MaterialAsset, error) {
	// 上传后置逻辑（硬性要求）：轮询 GetAsset 直至拿到上游永久 URL 或超时。
	info, err := service.MaterialPollAsset(assetId, tempLocalURL)
	if err != nil {
		// GetAsset 失败属异常：尝试回收上游素材，避免产生孤儿资产。
		_, _ = service.MaterialDeleteAsset(assetId)
		return nil, fmt.Errorf("拉取素材信息失败: %w", err)
	}

	// 数据存储规则：持久化 Result 下的关键字段，缺失时回退本地推断值。
	assetType := fallbackType
	if t := strings.TrimSpace(info.AssetType); t != "" {
		assetType = t
	}
	// 仅允许图片/视频素材，拦截音频等其他类型（拦截时回收上游素材）。
	if !service.IsValidMaterialAssetType(assetType) {
		_, _ = service.MaterialDeleteAsset(assetId)
		return nil, fmt.Errorf("仅支持图片或视频素材，当前素材类型: %s", assetType)
	}

	// URL：素材永久访问地址，优先使用接口返回值。
	permanentURL := fallbackURL
	if u := strings.TrimSpace(info.URL); u != "" {
		permanentURL = u
	}
	// Status：素材可用性状态，缺失时按处理中处理。
	status := service.MaterialStatusPending
	if s := service.NormalizeMaterialStatus(info.Status); s != "" {
		status = s
	}
	groupId := fallbackGroupId
	if g := strings.TrimSpace(info.GroupId); g != "" {
		groupId = g
	}

	asset := &model.MaterialAsset{
		UserId:    userId,
		GroupId:   groupId,
		AssetId:   assetId,
		Name:      name,
		AssetType: assetType,
		URL:       permanentURL,
		Status:    status,
	}
	if err := model.CreateMaterialAsset(asset); err != nil {
		return nil, err
	}

	// 清理临时文件：已拿到上游永久 URL 且与本地临时 URL 不一致时，丢弃本地临时上传文件。
	// 若上游仍在处理（未返回永久 URL），暂不清理以保证图片可正常预览，待列表轮询拿到永久 URL 后再清理。
	if tempLocalURL != "" && permanentURL != tempLocalURL {
		_ = service.CleanupLocalUploadByURL(tempLocalURL)
	}

	return asset, nil
}

// materialAssetResponse 素材列表/上传返回结构（对外统一使用 asset_id，不暴露数据库真实主键）。
type materialAssetResponse struct {
	AssetId   string `json:"asset_id"`
	AssetURI  string `json:"asset_uri"` // asset://asset-xxxx，用于复制替换图片资源地址
	GroupId   string `json:"group_id"`  // 上游分组 ID
	Name      string `json:"name"`
	AssetType string `json:"asset_type"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

func toMaterialAssetResponse(a *model.MaterialAsset) materialAssetResponse {
	return materialAssetResponse{
		AssetId:   a.AssetId,
		AssetURI:  "asset://" + a.AssetId,
		GroupId:   a.GroupId,
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

	items := make([]materialAssetResponse, 0, len(assets))
	for _, a := range assets {
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

	// 扩展名校验（上传格式拦截）：仅支持图片/视频，并据此标记 AssetType。
	ext := strings.ToLower(filepath.Ext(file.Filename))
	assetType, ok := detectMaterialAssetType(ext)
	if !ok {
		common.ApiErrorMsg(c, "仅支持上传图片或视频格式（图片：jpg/jpeg/png/webp/gif/bmp；视频：mp4/mov/webm/mkv/avi/m4v）")
		return
	}

	// 大小校验：单个文件 < 配置上限（默认 10MB）。
	maxSizeMB := operation_setting.GetSeedanceSetting().MaxImageSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if file.Size > int64(maxSizeMB)*1024*1024 {
		common.ApiErrorMsg(c, "文件超过大小限制（最大 "+strconv.Itoa(maxSizeMB)+"MB）")
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

	// 调用素材库上传素材（按扩展名标记 AssetType=Image/Video）。
	assetName := strings.TrimSpace(file.Filename)
	if assetName == "" {
		assetName = "portrait"
	}
	assetId, err := service.MaterialCreateAsset(group.GroupId, publicURL, assetName, assetType)
	if err != nil {
		common.ApiErrorMsg(c, "素材上传失败: "+err.Error())
		return
	}

	// 上传后置逻辑：等待 GetAsset 拉取完整信息 -> 落库 -> 清理本地临时文件。
	asset, err := finalizeMaterialUpload(userId, group.GroupId, assetId, assetName, assetType, publicURL, publicURL)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}

// UploadMaterialByURL 在线资源链接上传素材：校验链接与类型 -> 自动建组 -> 调用素材库上传 -> GetAsset 落库。
// 与本地上传的区别：素材直接来源于远端 URL，不产生本地临时文件，无需清理。
func UploadMaterialByURL(c *gin.Context) {
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

	// 请求体：在线链接、可选名称、可选素材类型、合规协议同意标记。
	var req struct {
		URL       string `json:"url"`
		Name      string `json:"name"`
		AssetType string `json:"asset_type"`
		Agreed    bool   `json:"agreed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数解析失败")
		return
	}

	// 协议同意校验（后端兜底）。
	if !req.Agreed {
		common.ApiErrorMsg(c, "请先阅读并勾选同意虚拟人像合规协议")
		return
	}

	// 链接合法性校验：必须为 http(s) 绝对地址。
	resourceURL := strings.TrimSpace(req.URL)
	parsed, perr := url.ParseRequestURI(resourceURL)
	if perr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		common.ApiErrorMsg(c, "请输入合法的在线资源链接（http/https）")
		return
	}

	// 素材类型判定：优先按链接扩展名识别，其次采用前端传入的合法类型，默认 Image。
	assetType := ""
	if t, ok := detectMaterialAssetType(filepath.Ext(parsed.Path)); ok {
		assetType = t
	} else if service.IsValidMaterialAssetType(strings.TrimSpace(req.AssetType)) {
		assetType = strings.TrimSpace(req.AssetType)
	} else {
		assetType = service.MaterialAssetTypeImage
	}

	// 自动建组。
	group, err := ensureMaterialGroup(userId)
	if err != nil {
		common.ApiErrorMsg(c, "创建素材分组失败: "+err.Error())
		return
	}

	// 素材名称：优先用户填写，其次取链接文件名，最后兜底。
	assetName := strings.TrimSpace(req.Name)
	if assetName == "" {
		assetName = strings.TrimSpace(filepath.Base(parsed.Path))
	}
	if assetName == "" || assetName == "." || assetName == "/" {
		assetName = "portrait"
	}

	// 调用素材库上传素材。
	assetId, err := service.MaterialCreateAsset(group.GroupId, resourceURL, assetName, assetType)
	if err != nil {
		common.ApiErrorMsg(c, "素材上传失败: "+err.Error())
		return
	}

	// 上传后置逻辑：等待 GetAsset 拉取完整信息 -> 落库（在线链接无本地临时文件，传空）。
	asset, err := finalizeMaterialUpload(userId, group.GroupId, assetId, assetName, assetType, resourceURL, "")
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}

// DeleteMaterial 删除素材：校验归属 -> 调用上游 DeleteAsset（传入素材唯一 Id）-> 删除本地记录与临时文件。
// 返回 Result.Id：本次删除成功的资产 ID。
func DeleteMaterial(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}

	id, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "素材 ID 无效")
		return
	}

	// 校验素材归属当前用户。
	asset, err := model.GetMaterialAssetByIdAndUser(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if asset == nil {
		common.ApiErrorMsg(c, "素材不存在或无权操作")
		return
	}

	// 调用上游 DeleteAsset 删除资产（传入素材唯一 Id = 上游 asset_id）。
	if operation_setting.IsSeedanceReady() && strings.TrimSpace(asset.AssetId) != "" {
		if _, e := service.MaterialDeleteAsset(asset.AssetId); e != nil {
			common.ApiErrorMsg(c, "素材删除失败: "+e.Error())
			return
		}
	}

	// 删除本地记录。
	if err := model.DeleteMaterialAsset(asset.Id); err != nil {
		common.ApiError(c, err)
		return
	}

	// 清理本地临时文件（仅本地存储模式生效，best-effort）。
	_ = service.CleanupLocalUploadByURL(asset.URL)

	common.ApiSuccess(c, gin.H{"id": asset.AssetId})
}

// UploadPersonalMaterial 个人素材上传：基于 API 令牌（sk-xxx）鉴权，
// 自动识别当前令牌归属用户，仅允许操作该用户的个人素材。
// 复用现有素材上传的文件校验、存储、建组、上游 CreateAsset + GetAsset 落库逻辑。
func UploadPersonalMaterial(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}

	// Token 鉴权中间件已写入当前用户 ID，直接读取即可隔离个人数据。
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

	// 协议同意校验（后端兜底）。
	if strings.ToLower(strings.TrimSpace(c.PostForm("agreed"))) != "true" {
		common.ApiErrorMsg(c, "请先阅读并勾选同意虚拟人像合规协议")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请选择文件字段 file")
		return
	}

	// 扩展名校验：仅支持图片 / 视频，并据此标记 AssetType。
	ext := strings.ToLower(filepath.Ext(file.Filename))
	assetType, ok := detectMaterialAssetType(ext)
	if !ok {
		common.ApiErrorMsg(c, "仅支持上传图片或视频格式（图片：jpg/jpeg/png/webp/gif/bmp；视频：mp4/mov/webm/mkv/avi/m4v）")
		return
	}

	// 大小校验：单个文件 < 配置上限（默认 10MB）。
	maxSizeMB := operation_setting.GetSeedanceSetting().MaxImageSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if file.Size > int64(maxSizeMB)*1024*1024 {
		common.ApiErrorMsg(c, "文件超过大小限制（最大 "+strconv.Itoa(maxSizeMB)+"MB）")
		return
	}

	// 复用通用上传逻辑生成公网 URL（本地存储或 OSS）。
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

	// 复用自动建组逻辑：首次上传时自动创建用户专属素材分组。
	group, err := ensureMaterialGroup(userId)
	if err != nil {
		common.ApiErrorMsg(c, "创建素材分组失败: "+err.Error())
		return
	}

	// 复用素材库上传素材逻辑。
	assetName := strings.TrimSpace(file.Filename)
	if assetName == "" {
		assetName = "portrait"
	}
	assetId, err := service.MaterialCreateAsset(group.GroupId, publicURL, assetName, assetType)
	if err != nil {
		common.ApiErrorMsg(c, "素材上传失败: "+err.Error())
		return
	}

	// 复用上传后置逻辑：等待 GetAsset 拉取完整信息 -> 落库 -> 清理本地临时文件。
	asset, err := finalizeMaterialUpload(userId, group.GroupId, assetId, assetName, assetType, publicURL, publicURL)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}

// UploadPersonalMaterialByURL 个人素材在线链接上传：基于 API 令牌鉴权，复用现有 URL 上传核心逻辑。
// 远端资源直接由素材库服务拉取，不产生本地临时文件。
func UploadPersonalMaterialByURL(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}

	// Token 鉴权中间件已写入当前用户 ID。
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

	// 请求体：在线链接、可选名称、可选素材类型、合规协议同意标记。
	var req struct {
		URL       string `json:"url"`
		Name      string `json:"name"`
		AssetType string `json:"asset_type"`
		Agreed    bool   `json:"agreed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数解析失败")
		return
	}

	// 协议同意校验（后端兜底）。
	if !req.Agreed {
		common.ApiErrorMsg(c, "请先阅读并勾选同意虚拟人像合规协议")
		return
	}

	// 链接合法性校验：必须为 http(s) 绝对地址。
	resourceURL := strings.TrimSpace(req.URL)
	parsed, perr := url.ParseRequestURI(resourceURL)
	if perr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		common.ApiErrorMsg(c, "请输入合法的在线资源链接（http/https）")
		return
	}

	// 素材类型判定：优先按链接扩展名识别，其次采用前端传入的合法类型，默认 Image。
	assetType := ""
	if t, ok := detectMaterialAssetType(filepath.Ext(parsed.Path)); ok {
		assetType = t
	} else if service.IsValidMaterialAssetType(strings.TrimSpace(req.AssetType)) {
		assetType = strings.TrimSpace(req.AssetType)
	} else {
		assetType = service.MaterialAssetTypeImage
	}

	// 复用自动建组逻辑。
	group, err := ensureMaterialGroup(userId)
	if err != nil {
		common.ApiErrorMsg(c, "创建素材分组失败: "+err.Error())
		return
	}

	// 素材名称：优先用户填写，其次取链接文件名，最后兜底。
	assetName := strings.TrimSpace(req.Name)
	if assetName == "" {
		assetName = strings.TrimSpace(filepath.Base(parsed.Path))
	}
	if assetName == "" || assetName == "." || assetName == "/" {
		assetName = "portrait"
	}

	// 复用素材库上传素材逻辑。
	assetId, err := service.MaterialCreateAsset(group.GroupId, resourceURL, assetName, assetType)
	if err != nil {
		common.ApiErrorMsg(c, "素材上传失败: "+err.Error())
		return
	}

	// 复用上传后置逻辑：等待 GetAsset 拉取完整信息 -> 落库（在线链接无本地临时文件，传空）。
	asset, err := finalizeMaterialUpload(userId, group.GroupId, assetId, assetName, assetType, resourceURL, "")
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}

// ListPersonalMaterialAssets 个人素材列表查询：基于 API 令牌鉴权，复用现有列表查询核心逻辑。
// 仅返回当前令牌归属用户的素材分页数据，自动隔离其他用户数据。
func ListPersonalMaterialAssets(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}

	pageInfo := common.GetPageQuery(c)

	// 复用自动建组逻辑以获取 group_id（首次调用会自动创建分组）。
	group, err := ensureMaterialGroup(userId)
	if err != nil {
		common.ApiErrorMsg(c, "创建素材分组失败: "+err.Error())
		return
	}
	groupId := ""
	if group != nil {
		groupId = group.GroupId
	}

	// 复用模型层分页查询。
	assets, total, err := model.ListMaterialAssets(userId, groupId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]materialAssetResponse, 0, len(assets))
	for _, a := range assets {
		items = append(items, toMaterialAssetResponse(a))
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// DeletePersonalMaterial 个人素材删除：基于 API 令牌鉴权，仅删除当前用户归属的素材。
// 对外统一使用 asset_id 作为标识，不暴露数据库真实主键。
// 复用现有删除逻辑：上游 DeleteAsset -> 本地记录删除 -> 清理本地临时文件。
func DeletePersonalMaterial(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}

	assetId := strings.TrimSpace(c.Param("asset_id"))
	if assetId == "" {
		common.ApiErrorMsg(c, "素材 ID 无效")
		return
	}

	// 校验素材归属当前令牌用户，防止越权操作其他用户素材。
	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if asset == nil {
		common.ApiErrorMsg(c, "素材不存在或无权操作")
		return
	}

	// 复用上游素材删除逻辑。
	if operation_setting.IsSeedanceReady() && strings.TrimSpace(asset.AssetId) != "" {
		if _, e := service.MaterialDeleteAsset(asset.AssetId); e != nil {
			common.ApiErrorMsg(c, "素材删除失败: "+e.Error())
			return
		}
	}

	// 复用本地记录删除逻辑（内部仍使用真实主键，对外不暴露）。
	if err := model.DeleteMaterialAsset(asset.Id); err != nil {
		common.ApiError(c, err)
		return
	}

	// 复用本地临时文件清理逻辑（best-effort）。
	_ = service.CleanupLocalUploadByURL(asset.URL)

	common.ApiSuccess(c, gin.H{"asset_id": asset.AssetId})
}

// GetPersonalMaterial 个人素材详情查询：基于 API 令牌鉴权，仅查询当前用户归属素材。
// 对外统一使用 asset_id 作为标识，不暴露数据库真实主键。
// 若素材仍待同步或仍为本地临时 URL，执行一次 best-effort 上游刷新。
func GetPersonalMaterial(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}

	assetId := strings.TrimSpace(c.Param("asset_id"))
	if assetId == "" {
		common.ApiErrorMsg(c, "素材 ID 无效")
		return
	}

	// 校验素材归属当前令牌用户，隔离其他用户数据。
	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if asset == nil {
		common.ApiErrorMsg(c, "素材不存在或无权操作")
		return
	}

	// 复用列表页的状态刷新逻辑：Pending 或本地 URL 时尝试向上游同步一次。
	if materialAssetNeedsUpstreamRefresh(asset) && operation_setting.IsSeedanceReady() {
		if info, e := service.MaterialGetAsset(asset.AssetId); e == nil {
			refreshMaterialAssetFromUpstream(asset, info)
		}
	}

	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}
