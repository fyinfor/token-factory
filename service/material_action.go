package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// MaterialAction 终端用户 Action 网关支持的业务标识（与上游严格对齐）。
const (
	MaterialActionCreateAssetGroup            = "CreateAssetGroup"
	MaterialActionGetAssetGroup               = "GetAssetGroup"
	MaterialActionListAssetGroups             = "ListAssetGroups"
	MaterialActionCreateAsset                 = "CreateAsset"
	MaterialActionGetAsset                    = "GetAsset"
	MaterialActionListAssets                  = "ListAssets"
	MaterialActionDeleteAsset                 = "DeleteAsset"
	MaterialActionDeleteAssetGroup            = "DeleteAssetGroup"
	MaterialActionCreateVisualValidateSession = "CreateVisualValidateSession"
	MaterialActionGetVisualValidateResult     = "GetVisualValidateResult"
	MaterialActionUpdateAssetGroup            = "UpdateAssetGroup"
	MaterialActionUpdateAsset                 = "UpdateAsset"
)

const (
	materialNameMaxLen        = 128
	materialDescriptionMaxLen = 512
	materialFilterNameMaxLen  = 64
	materialListPageSizeMax   = 100
	materialListPageSizeDefault = 10
	visualSessionTTLSeconds   = 300

	// 上游素材组类型枚举（与 ListAssetGroups / ListAssets 对齐）。
	MaterialUpstreamGroupTypeAIGC         = "AIGC"
	MaterialUpstreamGroupTypeLivenessFace = "LivenessFace"
)

// MaterialActionError 业务层错误，携带标准 code 供 Controller 映射。
type MaterialActionError struct {
	Code    int
	Message string
}

func (e *MaterialActionError) Error() string {
	if e == nil {
		return "业务错误"
	}
	return e.Message
}

func materialActionErr(code int, msg string) *MaterialActionError {
	return &MaterialActionError{Code: code, Message: msg}
}

func materialActionNotReady() *MaterialActionError {
	return materialActionErr(common.MaterialCodeServiceUnavailable, "素材库功能未启用或基础地址未配置，请联系管理员")
}

func materialActionLog(userId int, action string, resourceId string, detail string) {
	common.SysLog(fmt.Sprintf("[material-action] user_id=%d action=%s resource_id=%s %s", userId, action, resourceId, detail))
}

// ---------------------------------------------------------------------------
// 入参 DTO（与上游 JSON 字段名严格对齐）
// ---------------------------------------------------------------------------

type CreateAssetGroupInput struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

type AssetGroupIdInput struct {
	Id string `json:"Id"`
}

type UpdateAssetGroupInput struct {
	Id          string `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

type UpdateAssetInput struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

type CreateAssetInput struct {
	GroupId   string `json:"GroupId"`
	URL       string `json:"URL"`
	Name      string `json:"Name"`
	AssetType string `json:"AssetType"`
}

type VisualValidateResultInput struct {
	BytedToken string `json:"BytedToken"`
}

type ListAssetGroupsFilter struct {
	GroupType string   `json:"GroupType"`
	GroupIds  []string `json:"GroupIds"`
	Name      string   `json:"Name"`
}

type ListAssetGroupsInput struct {
	Filter      *ListAssetGroupsFilter `json:"Filter"`
	PageNumber  int                    `json:"PageNumber"`
	PageSize    int                    `json:"PageSize"`
	SortBy      string                 `json:"SortBy"`
	SortOrder   string                 `json:"SortOrder"`
	ProjectName string                 `json:"ProjectName"`
}

type ListAssetsFilter struct {
	GroupType string   `json:"GroupType"`
	GroupIds  []string `json:"GroupIds"`
	Statuses  []string `json:"Statuses"`
	Name      string   `json:"Name"`
}

type ListAssetsInput struct {
	Filter      *ListAssetsFilter `json:"Filter"`
	PageNumber  int               `json:"PageNumber"`
	PageSize    int               `json:"PageSize"`
	SortBy      string            `json:"SortBy"`
	SortOrder   string            `json:"SortOrder"`
	ProjectName string            `json:"ProjectName"`
}

// ListAssetGroupsResult 与上游 ListAssetGroups Result 对齐。
type ListAssetGroupsResult struct {
	Items      []MaterialGroupResult `json:"Items"`
	TotalCount int64                 `json:"TotalCount"`
	PageNumber int                   `json:"PageNumber"`
	PageSize   int                   `json:"PageSize"`
}

// ListAssetsResult 与上游 ListAssets Result 对齐。
type ListAssetsResult struct {
	Items      []MaterialAssetResult `json:"Items"`
	TotalCount int64                 `json:"TotalCount"`
	PageNumber int                   `json:"PageNumber"`
	PageSize   int                   `json:"PageSize"`
}

// ---------------------------------------------------------------------------
// 8 个 Action 业务实现
// ---------------------------------------------------------------------------

// ActionCreateAssetGroup 创建素材组并绑定操作用户。
func ActionCreateAssetGroup(userId int, input CreateAssetGroupInput) (map[string]string, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Name 不能为空")
	}
	if len([]rune(name)) > materialNameMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Name 长度不能超过 %d 个字符", materialNameMaxLen))
	}
	description := strings.TrimSpace(input.Description)
	if len([]rune(description)) > materialDescriptionMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Description 长度不能超过 %d 个字符", materialDescriptionMaxLen))
	}

	materialActionLog(userId, MaterialActionCreateAssetGroup, "", "start")
	groupId, err := MaterialCreateAssetGroup(name, description)
	if err != nil {
		materialActionLog(userId, MaterialActionCreateAssetGroup, "", "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "创建素材组失败: "+err.Error())
	}

	group := &model.MaterialGroup{
		UserId:      userId,
		GroupName:   name,
		Description: description,
		GroupId:     groupId,
		GroupType:   model.MaterialGroupTypeVirtual,
	}
	if err := model.CreateMaterialGroup(group); err != nil {
		materialActionLog(userId, MaterialActionCreateAssetGroup, groupId, "db_failed: "+err.Error())
		_, _ = MaterialDeleteAssetGroup(groupId)
		return nil, materialActionErr(common.MaterialCodeInternalError, "保存素材组记录失败")
	}

	materialActionLog(userId, MaterialActionCreateAssetGroup, groupId, "success")
	return map[string]string{"Id": groupId}, nil
}

// ActionGetAssetGroup 查询素材组详情（校验用户归属后调用上游）。
func ActionGetAssetGroup(userId int, input AssetGroupIdInput) (*MaterialGroupResult, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	groupId := strings.TrimSpace(input.Id)
	if groupId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Id 不能为空")
	}

	local, err := model.GetMaterialGroupByGroupIdAndUser(groupId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材组失败")
	}
	if local == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "分组不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionGetAssetGroup, groupId, "start")
	result, err := MaterialGetAssetGroup(groupId)
	if err != nil {
		materialActionLog(userId, MaterialActionGetAssetGroup, groupId, "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材组失败: "+err.Error())
	}
	if result != nil && strings.TrimSpace(result.Name) == "" {
		result.Name = local.GroupName
	}
	if result != nil && strings.TrimSpace(result.Description) == "" {
		result.Description = local.Description
	}
	materialActionLog(userId, MaterialActionGetAssetGroup, groupId, "success")
	return result, nil
}

// ActionCreateAsset 上传素材（校验分组归属 → 上游 CreateAsset → GetAsset 轮询 → 本地落库）。
func ActionCreateAsset(userId int, input CreateAssetInput) (map[string]string, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	groupId := strings.TrimSpace(input.GroupId)
	resourceURL := strings.TrimSpace(input.URL)
	name := strings.TrimSpace(input.Name)
	assetType := strings.TrimSpace(input.AssetType)

	if groupId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "GroupId 不能为空")
	}
	if resourceURL == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "URL 不能为空")
	}
	if name == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Name 不能为空")
	}
	if len([]rune(name)) > materialNameMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Name 长度不能超过 %d 个字符", materialNameMaxLen))
	}
	if assetType == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "AssetType 不能为空")
	}
	if assetType != MaterialAssetTypeImage && assetType != MaterialAssetTypeVideo && assetType != MaterialAssetTypeAudio {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "AssetType 必须为 Image、Video 或 Audio")
	}

	group, err := model.GetMaterialGroupByGroupIdAndUser(groupId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材组失败")
	}
	if group == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "分组不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionCreateAsset, groupId, "start")
	assetId, err := MaterialCreateAsset(groupId, resourceURL, name, assetType)
	if err != nil {
		materialActionLog(userId, MaterialActionCreateAsset, groupId, "upstream_create_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "上传素材失败: "+err.Error())
	}

	info, err := MaterialPollAsset(assetId, resourceURL)
	if err != nil {
		materialActionLog(userId, MaterialActionCreateAsset, assetId, "poll_failed: "+err.Error())
		_, _ = MaterialDeleteAsset(assetId)
		return nil, materialActionErr(common.MaterialCodeInternalError, "拉取素材信息失败: "+err.Error())
	}

	permanentURL := resourceURL
	if u := strings.TrimSpace(info.URL); u != "" {
		permanentURL = u
	}
	status := MaterialStatusPending
	if s := NormalizeMaterialStatus(info.Status); s != "" {
		status = s
	}
	resolvedType := assetType
	if t := strings.TrimSpace(info.AssetType); t != "" {
		resolvedType = t
	}
	resolvedGroupId := groupId
	if g := strings.TrimSpace(info.GroupId); g != "" {
		resolvedGroupId = g
	}

	asset := &model.MaterialAsset{
		UserId:    userId,
		GroupId:   resolvedGroupId,
		GroupType: group.GroupType,
		AssetId:   assetId,
		Name:      name,
		AssetType: resolvedType,
		URL:       permanentURL,
		Status:    status,
	}
	if err := model.CreateMaterialAsset(asset); err != nil {
		_, _ = MaterialDeleteAsset(assetId)
		return nil, materialActionErr(common.MaterialCodeInternalError, "保存素材记录失败")
	}
	materialActionLog(userId, MaterialActionCreateAsset, assetId, "success")
	return map[string]string{"Id": assetId}, nil
}

// ActionGetAsset 查询素材详情（校验归属 → 上游 GetAsset → 同步本地）。
func ActionGetAsset(userId int, input AssetGroupIdInput) (*MaterialAssetResult, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	assetId := strings.TrimSpace(input.Id)
	if assetId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Id 不能为空")
	}

	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材失败")
	}
	if asset == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "素材不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionGetAsset, assetId, "start")
	result, err := MaterialGetAsset(assetId)
	if err != nil {
		materialActionLog(userId, MaterialActionGetAsset, assetId, "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材失败: "+err.Error())
	}

	newStatus := NormalizeMaterialStatus(result.Status)
	newURL := strings.TrimSpace(result.URL)
	newAssetType := strings.TrimSpace(result.AssetType)
	if newStatus != "" || newURL != "" || newAssetType != "" {
		_ = model.UpdateMaterialAssetInfo(asset.Id, newStatus, newURL, newAssetType)
	}
	materialActionLog(userId, MaterialActionGetAsset, assetId, "success")
	return result, nil
}

// ActionDeleteAsset 删除素材（校验归属 → 上游删除 → 本地删除）。
func ActionDeleteAsset(userId int, input AssetGroupIdInput) (map[string]string, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	assetId := strings.TrimSpace(input.Id)
	if assetId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Id 不能为空")
	}

	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材失败")
	}
	if asset == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "素材不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionDeleteAsset, assetId, "start")
	if _, err := MaterialDeleteAsset(assetId); err != nil {
		materialActionLog(userId, MaterialActionDeleteAsset, assetId, "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "删除素材失败: "+err.Error())
	}
	if err := model.DeleteMaterialAsset(asset.Id); err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "删除本地素材记录失败")
	}
	_ = CleanupLocalUploadByURL(asset.URL)
	materialActionLog(userId, MaterialActionDeleteAsset, assetId, "success")
	return map[string]string{"Id": assetId}, nil
}

// ActionDeleteAssetGroup 删除素材组（校验归属 → 上游删除 → 本地删除）。
func ActionDeleteAssetGroup(userId int, input AssetGroupIdInput) (map[string]string, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	groupId := strings.TrimSpace(input.Id)
	if groupId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Id 不能为空")
	}

	group, err := model.GetMaterialGroupByGroupIdAndUser(groupId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材组失败")
	}
	if group == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "分组不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionDeleteAssetGroup, groupId, "start")
	if _, err := MaterialDeleteAssetGroup(groupId); err != nil {
		materialActionLog(userId, MaterialActionDeleteAssetGroup, groupId, "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "删除素材组失败: "+err.Error())
	}
	if err := model.DeleteMaterialGroup(group.Id); err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "删除本地分组记录失败")
	}
	materialActionLog(userId, MaterialActionDeleteAssetGroup, groupId, "success")
	return map[string]string{"Id": groupId}, nil
}

// ActionCreateVisualValidateSession 创建真人认证 H5 会话并绑定用户。
func ActionCreateVisualValidateSession(userId int) (*VisualValidateSessionResult, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}

	materialActionLog(userId, MaterialActionCreateVisualValidateSession, "", "start")
	result, err := MaterialCreateVisualValidateSession()
	if err != nil {
		materialActionLog(userId, MaterialActionCreateVisualValidateSession, "", "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "创建真人认证会话失败: "+err.Error())
	}

	now := time.Now().Unix()
	session := &model.MaterialVisualSession{
		UserId:     userId,
		BytedToken: result.BytedToken,
		H5Link:     result.H5Link,
		QrCode:     result.QrCode,
		Status:     model.VisualSessionStatusPending,
		ExpiresAt:  now + visualSessionTTLSeconds,
	}
	if err := model.CreateVisualSession(session); err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "保存认证会话失败")
	}
	materialActionLog(userId, MaterialActionCreateVisualValidateSession, result.BytedToken, "success")
	return result, nil
}

// ActionGetVisualValidateResult 查询真人认证结果（校验 BytedToken 归属）。
func ActionGetVisualValidateResult(userId int, input VisualValidateResultInput) (map[string]any, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	bytedToken := strings.TrimSpace(input.BytedToken)
	if bytedToken == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "BytedToken 不能为空")
	}

	session, err := model.GetVisualSessionByBytedTokenAndUser(bytedToken, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询认证会话失败")
	}
	if session == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "认证会话不存在或无权操作")
	}

	if session.Status == model.VisualSessionStatusSuccess && strings.TrimSpace(session.GroupId) != "" {
		return map[string]any{"GroupId": session.GroupId, "Status": model.VisualSessionStatusSuccess}, nil
	}
	if session.Status == model.VisualSessionStatusFailed {
		return map[string]any{"Status": model.VisualSessionStatusFailed, "Message": session.ErrorMessage}, nil
	}
	if session.Status == model.VisualSessionStatusExpired || time.Now().Unix() >= session.ExpiresAt {
		_ = model.UpdateVisualSessionStatus(session.Id, model.VisualSessionStatusExpired, "", "")
		return map[string]any{"Status": model.VisualSessionStatusExpired, "Message": "认证会话已过期，请重新发起认证"}, nil
	}

	materialActionLog(userId, MaterialActionGetVisualValidateResult, bytedToken, "poll")
	result, err := MaterialGetVisualValidateResult(bytedToken)
	if err != nil {
		return map[string]any{"Status": model.VisualSessionStatusPending}, nil
	}

	switch result.Status {
	case visualValidateSuccess:
		groupId := strings.TrimSpace(result.GroupId)
		if groupId == "" {
			return map[string]any{"Status": model.VisualSessionStatusPending}, nil
		}
		if err := ensureRealGroupForUser(userId, groupId); err != nil {
			return nil, materialActionErr(common.MaterialCodeInternalError, "保存真人分组失败")
		}
		_ = model.UpdateVisualSessionStatus(session.Id, model.VisualSessionStatusSuccess, groupId, "")
		materialActionLog(userId, MaterialActionGetVisualValidateResult, groupId, "success")
		return map[string]any{"GroupId": groupId, "Status": model.VisualSessionStatusSuccess}, nil
	case visualValidateFailed:
		_ = model.UpdateVisualSessionStatus(session.Id, model.VisualSessionStatusFailed, "", result.Message)
		return map[string]any{"Status": model.VisualSessionStatusFailed, "Message": result.Message}, nil
	default:
		return map[string]any{"Status": model.VisualSessionStatusPending}, nil
	}
}

// ActionUpdateAssetGroup 更新素材组（校验归属 → 上游更新 → 本地同步）。
func ActionUpdateAssetGroup(userId int, input UpdateAssetGroupInput) (map[string]string, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	groupId := strings.TrimSpace(input.Id)
	if groupId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Id 不能为空")
	}
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if name == "" && description == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Name 与 Description 至少填写一项")
	}
	if name != "" && len([]rune(name)) > materialNameMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Name 长度不能超过 %d 个字符", materialNameMaxLen))
	}
	if len([]rune(description)) > materialDescriptionMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Description 长度不能超过 %d 个字符", materialDescriptionMaxLen))
	}

	group, err := model.GetMaterialGroupByGroupIdAndUser(groupId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材组失败")
	}
	if group == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "分组不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionUpdateAssetGroup, groupId, "start")
	if _, err := MaterialUpdateAssetGroup(groupId, name, description); err != nil {
		materialActionLog(userId, MaterialActionUpdateAssetGroup, groupId, "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "更新素材组失败: "+err.Error())
	}

	updateName := group.GroupName
	if name != "" {
		updateName = name
	}
	if err := model.UpdateMaterialGroup(group.Id, userId, updateName, description); err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "更新本地分组记录失败")
	}
	materialActionLog(userId, MaterialActionUpdateAssetGroup, groupId, "success")
	return map[string]string{"Id": groupId}, nil
}

// ActionUpdateAsset 更新素材（校验归属 → 上游更新 → 本地同步）。
func ActionUpdateAsset(userId int, input UpdateAssetInput) (map[string]string, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	assetId := strings.TrimSpace(input.Id)
	if assetId == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Id 不能为空")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Name 不能为空")
	}
	if len([]rune(name)) > materialNameMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Name 长度不能超过 %d 个字符", materialNameMaxLen))
	}

	asset, err := model.GetMaterialAssetByAssetIdAndUser(assetId, userId)
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材失败")
	}
	if asset == nil {
		return nil, materialActionErr(common.MaterialCodeForbidden, "素材不存在或无权操作")
	}

	materialActionLog(userId, MaterialActionUpdateAsset, assetId, "start")
	if _, err := MaterialUpdateAsset(assetId, name); err != nil {
		materialActionLog(userId, MaterialActionUpdateAsset, assetId, "upstream_failed: "+err.Error())
		return nil, materialActionErr(common.MaterialCodeInternalError, "更新素材失败: "+err.Error())
	}
	if err := model.UpdateMaterialAssetName(asset.Id, name); err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "更新本地素材记录失败")
	}
	materialActionLog(userId, MaterialActionUpdateAsset, assetId, "success")
	return map[string]string{"Id": assetId}, nil
}

// ActionListAssetGroups 分页查询素材组列表（普通用户仅本人；管理员可查全部）。
func ActionListAssetGroups(userId int, input ListAssetGroupsInput) (*ListAssetGroupsResult, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	if input.Filter == nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Filter 不能为空")
	}
	localGroupType, err := mapUpstreamGroupTypeToLocal(input.Filter.GroupType)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Filter.Name)
	if len([]rune(name)) > materialFilterNameMaxLen {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Name 长度不能超过 %d 个字符", materialFilterNameMaxLen))
	}
	pageNumber, pageSize, err := normalizeListPagination(input.PageNumber, input.PageSize)
	if err != nil {
		return nil, err
	}
	sortBy, sortOrder, err := normalizeListSort(input.SortBy, input.SortOrder, false)
	if err != nil {
		return nil, err
	}

	filterUserId := userId
	if model.IsAdmin(userId) {
		filterUserId = 0
	}
	groups, total, err := model.ListMaterialGroupsFiltered(model.MaterialGroupListFilter{
		UserId:    filterUserId,
		GroupType: localGroupType,
		GroupIds:  trimStringSlice(input.Filter.GroupIds),
		Name:      name,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Offset:    (pageNumber - 1) * pageSize,
		Limit:     pageSize,
	})
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材组列表失败")
	}

	items := make([]MaterialGroupResult, 0, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		items = append(items, MaterialGroupResult{
			Id:          g.GroupId,
			Name:        g.GroupName,
			Description: g.Description,
			GroupType:   mapLocalGroupTypeToUpstream(g.GroupType),
			ProjectName: normalizeProjectName(input.ProjectName),
			CreateTime:  formatMaterialUnixTime(g.CreatedAt),
			UpdateTime:  formatMaterialUnixTime(g.UpdatedAt),
		})
	}
	materialActionLog(userId, MaterialActionListAssetGroups, "", fmt.Sprintf("success total=%d page=%d", total, pageNumber))
	return &ListAssetGroupsResult{
		Items:      items,
		TotalCount: total,
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}, nil
}

// ActionListAssets 分页查询素材列表（普通用户仅本人；管理员可查全部）。
func ActionListAssets(userId int, input ListAssetsInput) (*ListAssetsResult, error) {
	if !operation_setting.IsSeedanceReady() {
		return nil, materialActionNotReady()
	}
	localGroupType := ""
	var groupIds []string
	var statuses []string
	name := ""
	var err error
	if input.Filter != nil {
		// 文档要求：Filter 存在时 GroupType 必填。
		localGroupType, err = mapUpstreamGroupTypeToLocal(input.Filter.GroupType)
		if err != nil {
			return nil, err
		}
		groupIds = trimStringSlice(input.Filter.GroupIds)
		statuses, err = normalizeListStatuses(input.Filter.Statuses)
		if err != nil {
			return nil, err
		}
		name = strings.TrimSpace(input.Filter.Name)
		if len([]rune(name)) > materialFilterNameMaxLen {
			return nil, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("Name 长度不能超过 %d 个字符", materialFilterNameMaxLen))
		}
	}
	pageNumber, pageSize, err := normalizeListPagination(input.PageNumber, input.PageSize)
	if err != nil {
		return nil, err
	}
	sortBy, sortOrder, err := normalizeListSort(input.SortBy, input.SortOrder, true)
	if err != nil {
		return nil, err
	}

	filterUserId := userId
	if model.IsAdmin(userId) {
		filterUserId = 0
	}
	assets, total, err := model.ListMaterialAssetsFiltered(model.MaterialAssetListFilter{
		UserId:    filterUserId,
		GroupType: localGroupType,
		GroupIds:  groupIds,
		Statuses:  statuses,
		Name:      name,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Offset:    (pageNumber - 1) * pageSize,
		Limit:     pageSize,
	})
	if err != nil {
		return nil, materialActionErr(common.MaterialCodeInternalError, "查询素材列表失败")
	}

	items := make([]MaterialAssetResult, 0, len(assets))
	for _, a := range assets {
		if a == nil {
			continue
		}
		item := MaterialAssetResult{
			Id:          a.AssetId,
			Name:        a.Name,
			URL:         a.URL,
			AssetType:   a.AssetType,
			GroupId:     a.GroupId,
			Status:      NormalizeMaterialStatus(a.Status),
			ProjectName: normalizeProjectName(input.ProjectName),
			CreateTime:  formatMaterialUnixTime(a.CreatedAt),
			UpdateTime:  formatMaterialUnixTime(a.UpdatedAt),
		}
		item.Moderation.Strategy = "Default"
		items = append(items, item)
	}
	materialActionLog(userId, MaterialActionListAssets, "", fmt.Sprintf("success total=%d page=%d", total, pageNumber))
	return &ListAssetsResult{
		Items:      items,
		TotalCount: total,
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}, nil
}

func mapUpstreamGroupTypeToLocal(groupType string) (string, error) {
	switch strings.TrimSpace(groupType) {
	case MaterialUpstreamGroupTypeAIGC:
		return model.MaterialGroupTypeVirtual, nil
	case MaterialUpstreamGroupTypeLivenessFace:
		return model.MaterialGroupTypeReal, nil
	case "":
		return "", materialActionErr(common.MaterialCodeInvalidParameter, "Filter.GroupType 不能为空")
	default:
		return "", materialActionErr(common.MaterialCodeInvalidParameter, "Filter.GroupType 必须为 AIGC 或 LivenessFace")
	}
}

func mapLocalGroupTypeToUpstream(groupType string) string {
	switch strings.TrimSpace(groupType) {
	case model.MaterialGroupTypeReal:
		return MaterialUpstreamGroupTypeLivenessFace
	default:
		return MaterialUpstreamGroupTypeAIGC
	}
}

func normalizeProjectName(projectName string) string {
	if strings.TrimSpace(projectName) == "" {
		return "default"
	}
	return strings.TrimSpace(projectName)
}

func normalizeListPagination(pageNumber int, pageSize int) (int, int, error) {
	if pageNumber < 0 {
		return 0, 0, materialActionErr(common.MaterialCodeInvalidParameter, "PageNumber 必须大于等于 1")
	}
	if pageNumber == 0 {
		pageNumber = 1
	}
	if pageSize < 0 {
		return 0, 0, materialActionErr(common.MaterialCodeInvalidParameter, "PageSize 必须大于等于 1")
	}
	if pageSize == 0 {
		pageSize = materialListPageSizeDefault
	}
	if pageSize > materialListPageSizeMax {
		return 0, 0, materialActionErr(common.MaterialCodeInvalidParameter, fmt.Sprintf("PageSize 不能超过 %d", materialListPageSizeMax))
	}
	return pageNumber, pageSize, nil
}

func normalizeListSort(sortBy string, sortOrder string, allowGroupId bool) (string, string, error) {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		sortBy = "CreateTime"
	}
	var col string
	switch sortBy {
	case "CreateTime":
		col = "created_at"
	case "UpdateTime":
		col = "updated_at"
	case "GroupId":
		if !allowGroupId {
			return "", "", materialActionErr(common.MaterialCodeInvalidParameter, "SortBy 必须为 CreateTime 或 UpdateTime")
		}
		col = "group_id"
	default:
		if allowGroupId {
			return "", "", materialActionErr(common.MaterialCodeInvalidParameter, "SortBy 必须为 CreateTime、UpdateTime 或 GroupId")
		}
		return "", "", materialActionErr(common.MaterialCodeInvalidParameter, "SortBy 必须为 CreateTime 或 UpdateTime")
	}

	sortOrder = strings.TrimSpace(sortOrder)
	if sortOrder == "" {
		sortOrder = "Desc"
	}
	switch sortOrder {
	case "Desc", "desc":
		return col, "desc", nil
	case "Asc", "asc":
		return col, "asc", nil
	default:
		return "", "", materialActionErr(common.MaterialCodeInvalidParameter, "SortOrder 必须为 Desc 或 Asc")
	}
}

func normalizeListStatuses(statuses []string) ([]string, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(statuses))
	seen := make(map[string]struct{}, len(statuses))
	for _, s := range statuses {
		normalized := NormalizeMaterialStatus(s)
		switch normalized {
		case MaterialStatusActive, MaterialStatusPending, MaterialStatusFailed:
		default:
			return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Filter.Statuses 仅支持 Active、Processing、Failed")
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func trimStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func formatMaterialUnixTime(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

// ensureRealGroupForUser 认证成功后去重创建真人分组本地记录。
func ensureRealGroupForUser(userId int, groupId string) error {
	existing, err := model.GetMaterialGroupByGroupId(groupId)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.UserId != userId {
			return errors.New("该真人分组已被其他用户绑定")
		}
		return nil
	}
	group := &model.MaterialGroup{
		UserId:      userId,
		GroupName:   model.BuildMaterialGroupName(userId) + "_real_" + groupId,
		Description: "真人认证分组",
		GroupId:     groupId,
		GroupType:   model.MaterialGroupTypeReal,
	}
	return model.CreateMaterialGroup(group)
}
