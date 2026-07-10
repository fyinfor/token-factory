package controller

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// 真人认证会话有效期（H5 链接 5 分钟），与前端轮询上限对齐。
const visualSessionTTL = 5 * time.Minute

// realGroupResponse 真人分组列表项返回结构。
type realGroupResponse struct {
	Id          int    `json:"id"`
	GroupName   string `json:"group_name"`
	Description string `json:"description"`
	GroupId     string `json:"group_id"`
	GroupType   string `json:"group_type"`
	CreatedAt   int64  `json:"created_at"`
}

func toRealGroupResponse(g *model.MaterialGroup) realGroupResponse {
	return realGroupResponse{
		Id:          g.Id,
		GroupName:   g.GroupName,
		Description: g.Description,
		GroupId:     g.GroupId,
		GroupType:   g.GroupType,
		CreatedAt:   g.CreatedAt,
	}
}

// visualSessionResponse Web 控制台认证会话返回结构（不含 BytedToken）。
type visualSessionResponse struct {
	SessionId int    `json:"session_id"`
	H5Link    string `json:"h5_link"`
	QrCode    string `json:"qr_code"`
	ExpiresAt int64  `json:"expires_at"`
	Status    string `json:"status"`
}

// ensureRealGroup 校验指定上游分组 ID 属于当前用户且为真人认证分组，返回该分组。
func ensureRealGroup(c *gin.Context, groupId string) (*model.MaterialGroup, bool) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return nil, false
	}
	groupId = strings.TrimSpace(groupId)
	if groupId == "" {
		common.ApiErrorMsg(c, "分组 ID 无效")
		return nil, false
	}
	group, err := model.GetMaterialGroupByGroupIdAndUser(groupId, userId)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if group == nil {
		common.ApiErrorMsg(c, "分组不存在或无权操作")
		return nil, false
	}
	if group.GroupType != model.MaterialGroupTypeReal {
		common.ApiErrorMsg(c, "该分组不是真人认证分组")
		return nil, false
	}
	return group, true
}

// ===========================================================================
// Web 控制台（UserAuth）路由处理器
// ===========================================================================

// CreateVisualSession 创建真人认证会话（CreateVisualValidateSession）。
// 空请求体调用上游获取 BytedToken/H5Link/QrCode，后端存储 BytedToken，仅返回 session_id 给前端。
func CreateVisualSession(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}

	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}

	result, err := service.MaterialCreateVisualValidateSession()
	if err != nil {
		common.ApiErrorMsg(c, "创建真人认证会话失败: "+err.Error())
		return
	}

	now := time.Now().Unix()
	session := &model.MaterialVisualSession{
		UserId:     userId,
		BytedToken: result.BytedToken,
		H5Link:     result.H5Link,
		QrCode:     result.QrCode,
		Status:     model.VisualSessionStatusPending,
		ExpiresAt:  now + int64(visualSessionTTL.Seconds()),
	}
	if err := model.CreateVisualSession(session); err != nil {
		common.ApiErrorMsg(c, "保存认证会话失败: "+err.Error())
		return
	}

	common.ApiSuccess(c, visualSessionResponse{
		SessionId: session.Id,
		H5Link:    session.H5Link,
		QrCode:    session.QrCode,
		ExpiresAt: session.ExpiresAt,
		Status:    session.Status,
	})
}

// PollVisualResult 轮询真人认证结果（GetVisualValidateResult）。
// 前端每 3s 调用一次，后端用 BytedToken 查询上游，成功时创建/去重真人分组。
func PollVisualResult(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}

	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}

	sessionIdStr := strings.TrimSpace(c.Query("session_id"))
	if sessionIdStr == "" {
		common.ApiErrorMsg(c, "会话 ID 无效")
		return
	}
	sessionId, err := strconv.Atoi(sessionIdStr)
	if err != nil || sessionId <= 0 {
		common.ApiErrorMsg(c, "会话 ID 无效")
		return
	}

	session, err := model.GetVisualSessionByIdAndUser(sessionId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if session == nil {
		common.ApiErrorMsg(c, "认证会话不存在或无权操作")
		return
	}

	// 会话已终态（成功/失败）：直接返回缓存结果，不再轮询上游。
	if session.Status == model.VisualSessionStatusSuccess {
		common.ApiSuccess(c, gin.H{
			"status":   session.Status,
			"group_id": session.GroupId,
		})
		return
	}
	if session.Status == model.VisualSessionStatusFailed {
		common.ApiSuccess(c, gin.H{
			"status":  session.Status,
			"message": session.ErrorMessage,
		})
		return
	}

	// 会话过期检查。
	if time.Now().Unix() >= session.ExpiresAt {
		_ = model.UpdateVisualSessionStatus(session.Id, model.VisualSessionStatusExpired, "", "")
		common.ApiSuccess(c, gin.H{
			"status":  model.VisualSessionStatusExpired,
			"message": "认证会话已过期，请重新发起认证",
		})
		return
	}

	result, err := service.MaterialGetVisualValidateResult(session.BytedToken)
	if err != nil {
		// 网络异常视为认证中，继续轮询（不透传原始错误信息）。
		common.ApiSuccess(c, gin.H{
			"status": model.VisualSessionStatusPending,
		})
		return
	}

	switch result.Status {
	case "success":
		// 认证成功：去重创建真人分组，更新会话状态。
		groupId := strings.TrimSpace(result.GroupId)
		if groupId == "" {
			common.ApiErrorMsg(c, "认证成功但未返回有效分组 ID")
			return
		}
		// 去重：同 GroupId 已存在则复用。
		existing, gErr := model.GetMaterialGroupByGroupId(groupId)
		if gErr != nil {
			common.ApiError(c, gErr)
			return
		}
		if existing == nil {
			groupName := "未命名"
			newGroup := &model.MaterialGroup{
				UserId:      userId,
				GroupName:   groupName,
				GroupId:     groupId,
				GroupType:   model.MaterialGroupTypeReal,
			}
			if cErr := model.CreateMaterialGroup(newGroup); cErr != nil {
				common.ApiErrorMsg(c, "创建真人分组失败: "+cErr.Error())
				return
			}
		}
		_ = model.UpdateVisualSessionStatus(session.Id, model.VisualSessionStatusSuccess, groupId, "")
		common.ApiSuccess(c, gin.H{
			"status":   model.VisualSessionStatusSuccess,
			"group_id": groupId,
		})
	case "failed":
		_ = model.UpdateVisualSessionStatus(session.Id, model.VisualSessionStatusFailed, "", result.Message)
		common.ApiSuccess(c, gin.H{
			"status":  model.VisualSessionStatusFailed,
			"message": result.Message,
		})
	default: // pending
		common.ApiSuccess(c, gin.H{
			"status": model.VisualSessionStatusPending,
		})
	}
}

// ListRealGroups 查询当前用户的所有真人认证分组。
func ListRealGroups(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}
	groups, err := model.GetRealMaterialGroupsByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]realGroupResponse, 0, len(groups))
	for _, g := range groups {
		items = append(items, toRealGroupResponse(g))
	}
	common.ApiSuccess(c, gin.H{"items": items})
}

// DeleteRealGroup 删除真人认证分组：校验归属+类型 -> 调用上游 DeleteAssetGroup -> 删除本地记录。
func DeleteRealGroup(c *gin.Context) {
	groupId := strings.TrimSpace(c.Param("group_id"))
	group, ok := ensureRealGroup(c, groupId)
	if !ok {
		return
	}
	// 调用上游删除分组（best-effort，上游失败仍删除本地记录）。
	if operation_setting.IsSeedanceReady() && strings.TrimSpace(group.GroupId) != "" {
		_, _ = service.MaterialDeleteAssetGroup(group.GroupId)
	}
	if err := model.DeleteMaterialGroup(group.Id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"group_id": group.GroupId})
}

// UpdateRealGroup 更新真人认证分组的名称和描述。
func UpdateRealGroup(c *gin.Context) {
	groupId := strings.TrimSpace(c.Param("group_id"))
	group, ok := ensureRealGroup(c, groupId)
	if !ok {
		return
	}
	var req struct {
		GroupName   string `json:"group_name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数解析失败")
		return
	}
	groupName := strings.TrimSpace(req.GroupName)
	if groupName == "" {
		common.ApiErrorMsg(c, "分组名称不能为空")
		return
	}
	if len([]rune(groupName)) > 64 {
		common.ApiErrorMsg(c, "分组名称不能超过 64 个字符")
		return
	}
	if len([]rune(req.Description)) > 256 {
		common.ApiErrorMsg(c, "分组描述不能超过 256 个字符")
		return
	}
	if err := model.UpdateMaterialGroup(group.Id, c.GetInt("id"), groupName, req.Description); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"id":          group.Id,
		"group_name":  groupName,
		"description": strings.TrimSpace(req.Description),
		"group_id":    group.GroupId,
		"group_type":  group.GroupType,
		"created_at":  group.CreatedAt,
	})
}

// ListRealAssets 分页查询真人素材列表。
// group_id 非空时按指定真人分组查询（校验归属）；为空时跨组列出该用户全部真人素材。
func ListRealAssets(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}
	pageInfo := common.GetPageQuery(c)

	groupId := strings.TrimSpace(c.Query("group_id"))
	if groupId != "" {
		// 校验分组归属与类型。
		group, err := model.GetMaterialGroupByGroupIdAndUser(groupId, userId)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if group == nil || group.GroupType != model.MaterialGroupTypeReal {
			common.ApiErrorMsg(c, "分组不存在或不是真人认证分组")
			return
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
		return
	}

	// 跨组列出全部真人素材。
	assets, total, err := model.ListMaterialAssetsByGroupType(userId, model.MaterialGroupTypeReal, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
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

// uploadRealMaterialCore 真人素材上传核心逻辑（本地文件与在线链接共用）。
// group 必须为已校验的真人认证分组。
func uploadRealMaterialCore(c *gin.Context, group *model.MaterialGroup, publicURL, assetName, assetType, tempLocalURL string) {
	assetId, err := service.MaterialCreateAsset(group.GroupId, publicURL, assetName, assetType)
	if err != nil {
		common.ApiErrorMsg(c, "素材上传失败: "+err.Error())
		return
	}
	asset, err := finalizeMaterialUpload(c.GetInt("id"), group.GroupId, assetId, assetName, assetType, publicURL, tempLocalURL)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	// 标记为真人素材。
	asset.GroupType = model.MaterialGroupTypeReal
	_ = model.UpdateMaterialAssetGroupType(asset.Id, model.MaterialGroupTypeReal)
	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}

// UploadRealMaterial 真人素材本地上传：校验真人分组 -> 文件校验 -> OSS 上传 -> 上游 CreateAsset -> 落库。
func UploadRealMaterial(c *gin.Context) {
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
	groupId := strings.TrimSpace(c.PostForm("group_id"))
	group, ok := ensureRealGroup(c, groupId)
	if !ok {
		return
	}
	if strings.ToLower(strings.TrimSpace(c.PostForm("agreed"))) != "true" {
		common.ApiErrorMsg(c, "请先阅读并勾选同意真人素材合规协议")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请选择文件字段 file")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	assetType, typeOk := detectMaterialAssetType(ext)
	if !typeOk {
		common.ApiErrorMsg(c, "仅支持上传图片或视频格式（图片：jpg/jpeg/png/webp/gif/bmp；视频：mp4/mov/webm/mkv/avi/m4v）")
		return
	}
	maxSizeMB := operation_setting.GetSeedanceSetting().MaxImageSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if file.Size > int64(maxSizeMB)*1024*1024 {
		common.ApiErrorMsg(c, "文件超过大小限制（最大 "+strconv.Itoa(maxSizeMB)+"MB）")
		return
	}
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
	assetName := strings.TrimSpace(file.Filename)
	if assetName == "" {
		assetName = "real-portrait"
	}
	uploadRealMaterialCore(c, group, publicURL, assetName, assetType, publicURL)
}

// UploadRealMaterialByURL 真人素材在线链接上传。
func UploadRealMaterialByURL(c *gin.Context) {
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
	var req struct {
		GroupId   string `json:"group_id"`
		URL       string `json:"url"`
		Name      string `json:"name"`
		AssetType string `json:"asset_type"`
		Agreed    bool   `json:"agreed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数解析失败")
		return
	}
	group, ok := ensureRealGroup(c, req.GroupId)
	if !ok {
		return
	}
	if !req.Agreed {
		common.ApiErrorMsg(c, "请先阅读并勾选同意真人素材合规协议")
		return
	}
	resourceURL := strings.TrimSpace(req.URL)
	parsed, perr := url.ParseRequestURI(resourceURL)
	if perr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		common.ApiErrorMsg(c, "请输入合法的在线资源链接（http/https）")
		return
	}
	assetType := ""
	if t, ok := detectMaterialAssetType(filepath.Ext(parsed.Path)); ok {
		assetType = t
	} else if service.IsValidMaterialAssetType(strings.TrimSpace(req.AssetType)) {
		assetType = strings.TrimSpace(req.AssetType)
	} else {
		assetType = service.MaterialAssetTypeImage
	}
	assetName := strings.TrimSpace(req.Name)
	if assetName == "" {
		assetName = strings.TrimSpace(filepath.Base(parsed.Path))
	}
	if assetName == "" || assetName == "." || assetName == "/" {
		assetName = "real-portrait"
	}
	uploadRealMaterialCore(c, group, resourceURL, assetName, assetType, "")
}

// GetRealMaterial 真人素材详情查询：校验归属+真人组 -> best-effort 上游刷新。
func GetRealMaterial(c *gin.Context) {
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
	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if asset == nil {
		common.ApiErrorMsg(c, "素材不存在或无权操作")
		return
	}
	// 校验素材属于真人分组。
	if asset.GroupType != model.MaterialGroupTypeReal {
		common.ApiErrorMsg(c, "该素材不是真人素材")
		return
	}
	if materialAssetNeedsUpstreamRefresh(asset) && operation_setting.IsSeedanceReady() {
		if info, e := service.MaterialGetAsset(asset.AssetId); e == nil {
			refreshMaterialAssetFromUpstream(asset, info)
		}
	}
	common.ApiSuccess(c, toMaterialAssetResponse(asset))
}

// DeleteRealMaterial 真人素材删除：校验归属+真人组 -> 上游 DeleteAsset -> 删除本地记录。
func DeleteRealMaterial(c *gin.Context) {
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
	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if asset == nil {
		common.ApiErrorMsg(c, "素材不存在或无权操作")
		return
	}
	if asset.GroupType != model.MaterialGroupTypeReal {
		common.ApiErrorMsg(c, "该素材不是真人素材")
		return
	}
	if operation_setting.IsSeedanceReady() && strings.TrimSpace(asset.AssetId) != "" {
		if _, e := service.MaterialDeleteAsset(asset.AssetId); e != nil {
			common.ApiErrorMsg(c, "素材删除失败: "+e.Error())
			return
		}
	}
	if err := model.DeleteMaterialAsset(asset.Id); err != nil {
		common.ApiError(c, err)
		return
	}
	_ = service.CleanupLocalUploadByURL(asset.URL)
	common.ApiSuccess(c, gin.H{"asset_id": asset.AssetId})
}

// ===========================================================================
// 个人 API 令牌（TokenAuth）路由处理器
// ===========================================================================

// CreatePersonalVisualSession 个人 API 令牌创建真人认证会话。
// 与 Web 控制台不同：直接返回 BytedToken，供程序化客户端自行轮询 GetVisualValidateResult。
func CreatePersonalVisualSession(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}
	result, err := service.MaterialCreateVisualValidateSession()
	if err != nil {
		common.ApiErrorMsg(c, "创建真人认证会话失败: "+err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{
		"byted_token": result.BytedToken,
		"h5_link":     result.H5Link,
		"qr_code":     result.QrCode,
		"expires_in":  int64(visualSessionTTL.Seconds()),
	})
}

// GetPersonalVisualResult 个人 API 令牌查询真人认证结果。
// 入参 BytedToken（请求体），返回结构化状态。成功时不自动建组（由客户端决定）。
func GetPersonalVisualResult(c *gin.Context) {
	if !operation_setting.IsSeedanceReady() {
		common.ApiErrorMsg(c, "素材库功能未启用或基础地址未配置，请联系管理员")
		return
	}
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未授权")
		return
	}
	var req struct {
		BytedToken string `json:"byted_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数解析失败")
		return
	}
	bytedToken := strings.TrimSpace(req.BytedToken)
	if bytedToken == "" {
		common.ApiErrorMsg(c, "byted_token 不能为空")
		return
	}
	result, err := service.MaterialGetVisualValidateResult(bytedToken)
	if err != nil {
		common.ApiSuccess(c, gin.H{
			"status": model.VisualSessionStatusPending,
		})
		return
	}
	switch result.Status {
	case "success":
		common.ApiSuccess(c, gin.H{
			"status":   model.VisualSessionStatusSuccess,
			"group_id": result.GroupId,
		})
	case "failed":
		common.ApiSuccess(c, gin.H{
			"status":  model.VisualSessionStatusFailed,
			"message": result.Message,
		})
	default:
		common.ApiSuccess(c, gin.H{
			"status": model.VisualSessionStatusPending,
		})
	}
}

// ListPersonalRealGroups 个人 API 令牌查询真人分组列表。
func ListPersonalRealGroups(c *gin.Context) {
	ListRealGroups(c)
}

// ListPersonalRealAssets 个人 API 令牌查询真人素材列表。
func ListPersonalRealAssets(c *gin.Context) {
	ListRealAssets(c)
}

// DeletePersonalRealGroup 个人 API 令牌删除真人分组。
func DeletePersonalRealGroup(c *gin.Context) {
	DeleteRealGroup(c)
}

// UpdatePersonalRealGroup 个人 API 令牌更新真人分组名称和描述。
func UpdatePersonalRealGroup(c *gin.Context) {
	UpdateRealGroup(c)
}

// UploadPersonalRealMaterial 个人 API 令牌真人素材本地上传。
func UploadPersonalRealMaterial(c *gin.Context) {
	UploadRealMaterial(c)
}

// UploadPersonalRealMaterialByURL 个人 API 令牌真人素材在线链接上传。
func UploadPersonalRealMaterialByURL(c *gin.Context) {
	UploadRealMaterialByURL(c)
}

// GetPersonalRealMaterial 个人 API 令牌真人素材详情查询。
func GetPersonalRealMaterial(c *gin.Context) {
	GetRealMaterial(c)
}

// DeletePersonalRealMaterial 个人 API 令牌真人素材删除。
func DeletePersonalRealMaterial(c *gin.Context) {
	DeleteRealMaterial(c)
}
