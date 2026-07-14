package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// 素材库接口 Action 常量（与上游素材库 API 严格对齐，大小写不可改）。
const (
	materialActionCreateAssetGroup            = "CreateAssetGroup"
	materialActionGetAssetGroup               = "GetAssetGroup"
	materialActionCreateAsset                 = "CreateAsset"
	materialActionGetAsset                    = "GetAsset"
	materialActionDeleteAsset                 = "DeleteAsset"
	materialActionDeleteAssetGroup            = "DeleteAssetGroup"
	materialActionCreateVisualValidateSession = "CreateVisualValidateSession"
	materialActionGetVisualValidateResult     = "GetVisualValidateResult"
	materialActionUpdateAssetGroup            = "UpdateAssetGroup"
	materialActionUpdateAsset                 = "UpdateAsset"

	materialRequestMaxRetries = 3
	materialRequestRetryDelay = 500 * time.Millisecond

	// 素材资产类型枚举（AssetType，与上游接口严格对齐）：图片 / 视频 / 音频。
	MaterialAssetTypeImage = "Image"
	MaterialAssetTypeVideo = "Video"
	MaterialAssetTypeAudio = "Audio"

	// 素材状态枚举（Status，与上游接口严格对齐）：可用 / 失败 / 处理中。
	MaterialStatusActive  = "Active"
	MaterialStatusFailed  = "Failed"
	MaterialStatusPending = "Pending"

	materialGetAssetMaxAttempts  = 20
	materialGetAssetPollInterval = time.Second
)

// IsValidMaterialAssetType 判断素材类型是否为业务允许上传的图片/视频（音频等其他类型一律拦截）。
func IsValidMaterialAssetType(assetType string) bool {
	return assetType == MaterialAssetTypeImage || assetType == MaterialAssetTypeVideo
}

// materialResponse 素材库接口通用响应包装。
type materialResponse struct {
	ResponseMetadata struct {
		RequestId string `json:"RequestId"`
		Action    string `json:"Action"`
	} `json:"ResponseMetadata"`
	Result json.RawMessage `json:"Result"`
	// 错误信息（部分实现以此结构返回错误）。
	ResponseError *struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"Error,omitempty"`
}

// MaterialGroupResult GetAssetGroup 返回的分组信息。
type MaterialGroupResult struct {
	Id          string `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	GroupType   string `json:"GroupType"`
	ProjectName string `json:"ProjectName"`
	CreateTime  string `json:"CreateTime"`
	UpdateTime  string `json:"UpdateTime"`
}

// MaterialAssetResult GetAsset 返回的素材信息。
type MaterialAssetResult struct {
	Id         string `json:"Id"`
	Name       string `json:"Name"`
	URL        string `json:"URL"`
	AssetType  string `json:"AssetType"`
	GroupId    string `json:"GroupId"`
	Status     string `json:"Status"`
	Moderation struct {
		Strategy string `json:"Strategy"`
	} `json:"Moderation"`
	Error struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"Error"`
	ProjectName string `json:"ProjectName"`
	CreateTime  string `json:"CreateTime"`
	UpdateTime  string `json:"UpdateTime"`
}

// materialResultErrorBody Result 内嵌业务错误结构（HTTP 200 时仍可能携带 Error）。
type materialResultErrorBody struct {
	Error *struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"Error"`
}

// MaterialUpstreamError 上游素材库业务错误，便于上层区分可重试与不可重试场景。
type MaterialUpstreamError struct {
	Code    string
	Message string
}

func (e *MaterialUpstreamError) Error() string {
	if e == nil {
		return "素材库接口错误"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Code) != "" {
		return e.Code
	}
	return "素材库接口错误"
}

func isRetryableMaterialRequestErr(err error) bool {
	if err == nil {
		return false
	}
	var upErr *MaterialUpstreamError
	if errors.As(err, &upErr) {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, kw := range []string{"timeout", "connection reset", "connection refused", "eof", "temporary", "503", "502", "504"} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

func parseMaterialResultError(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var body materialResultErrorBody
	if err := common.Unmarshal(raw, &body); err != nil {
		return nil
	}
	if body.Error != nil && strings.TrimSpace(body.Error.Message) != "" {
		return &MaterialUpstreamError{Code: body.Error.Code, Message: body.Error.Message}
	}
	return nil
}

// doMaterialRequest 向素材库 API 发送 POST JSON 请求并解析通用响应（含 transient 错误重试）。
func doMaterialRequest(action string, payload any, result any) error {
	var lastErr error
	for attempt := 0; attempt < materialRequestMaxRetries; attempt++ {
		lastErr = doMaterialRequestOnce(action, payload, result)
		if lastErr == nil {
			return nil
		}
		if !isRetryableMaterialRequestErr(lastErr) || attempt >= materialRequestMaxRetries-1 {
			return lastErr
		}
		common.SysLog(fmt.Sprintf("[material-upstream] action=%s retry=%d err=%v", action, attempt+1, lastErr))
		time.Sleep(materialRequestRetryDelay)
	}
	return lastErr
}

// doMaterialRequestOnce 单次向上游素材库发起 POST JSON 请求。
func doMaterialRequestOnce(action string, payload any, result any) error {
	base := operation_setting.GetMaterialAPIBaseURL()
	if base == "" {
		return errors.New("素材库 API 基础地址未配置")
	}
	reqURL := fmt.Sprintf("%s/api/material?Action=%s", strings.TrimRight(base, "/"), action)

	body, err := common.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化素材库请求失败: %w", err)
	}

	common.SysLog(fmt.Sprintf("[material-upstream] action=%s request=%s", action, truncateMaterialBody(body)))

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造素材库请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if apiKey := strings.TrimSpace(operation_setting.GetSeedanceSetting().APIKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return fmt.Errorf("请求素材库接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取素材库响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("素材库接口返回异常状态码 %d: %s", resp.StatusCode, truncateMaterialBody(respBytes))
	}

	var wrapper materialResponse
	if err := common.Unmarshal(respBytes, &wrapper); err != nil {
		return fmt.Errorf("解析素材库响应失败: %w", err)
	}
	if wrapper.ResponseError != nil && strings.TrimSpace(wrapper.ResponseError.Message) != "" {
		return &MaterialUpstreamError{Code: wrapper.ResponseError.Code, Message: wrapper.ResponseError.Message}
	}
	if err := parseMaterialResultError(wrapper.Result); err != nil {
		return err
	}
	if result != nil && len(wrapper.Result) > 0 {
		if err := common.Unmarshal(wrapper.Result, result); err != nil {
			return fmt.Errorf("解析素材库返回结果失败: %w", err)
		}
	}
	common.SysLog(fmt.Sprintf("[material-upstream] action=%s success", action))
	return nil
}

func truncateMaterialBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

// MaterialCreateAssetGroup 创建素材组，返回上游分组 ID。
func MaterialCreateAssetGroup(name string, description string) (string, error) {
	payload := map[string]string{
		"Name":        name,
		"Description": description,
	}
	var result struct {
		Id string `json:"Id"`
	}
	if err := doMaterialRequest(materialActionCreateAssetGroup, payload, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Id) == "" {
		return "", errors.New("素材库未返回有效分组 ID")
	}
	return result.Id, nil
}

// MaterialGetAssetGroup 查询素材组信息。
func MaterialGetAssetGroup(groupId string) (*MaterialGroupResult, error) {
	payload := map[string]string{"Id": groupId}
	var result MaterialGroupResult
	if err := doMaterialRequest(materialActionGetAssetGroup, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// MaterialCreateAsset 上传素材，返回上游素材 ID（asset-xxxx）。
func MaterialCreateAsset(groupId string, url string, name string, assetType string) (string, error) {
	if strings.TrimSpace(assetType) == "" {
		assetType = MaterialAssetTypeImage
	}
	payload := map[string]string{
		"GroupId":   groupId,
		"URL":       url,
		"Name":      name,
		"AssetType": assetType,
	}
	var result struct {
		Id string `json:"Id"`
	}
	if err := doMaterialRequest(materialActionCreateAsset, payload, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Id) == "" {
		return "", errors.New("素材库未返回有效素材 ID")
	}
	return result.Id, nil
}

// NormalizeMaterialStatus 归一化上游 Status 枚举（兼容大小写差异）。
func NormalizeMaterialStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return MaterialStatusActive
	case "failed":
		return MaterialStatusFailed
	case "pending":
		return MaterialStatusPending
	default:
		return strings.TrimSpace(status)
	}
}

func materialURLsEquivalent(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// shouldContinueMaterialPoll 判断是否应继续轮询 GetAsset。
// sourceURL 为 CreateAsset 时传入的原始地址；本地上传场景下需等待上游返回不同的永久 URL。
func shouldContinueMaterialPoll(info *MaterialAssetResult, sourceURL string) bool {
	if info == nil {
		return true
	}
	status := NormalizeMaterialStatus(info.Status)
	url := strings.TrimSpace(info.URL)
	sourceURL = strings.TrimSpace(sourceURL)

	if status == MaterialStatusFailed {
		return false
	}

	if IsLocalMaterialUploadURL(sourceURL) {
		if url != "" && !materialURLsEquivalent(url, sourceURL) {
			return false
		}
		return status == MaterialStatusPending || url == "" || materialURLsEquivalent(url, sourceURL)
	}

	if status == MaterialStatusActive && url != "" {
		return false
	}
	return status == MaterialStatusPending || url == ""
}

// MaterialGetAsset 查询单个素材信息（GetAsset）。
// 入参：素材资产 ID 字符串（必填）。返回 Result 完整结构（含 URL/AssetType/GroupId/Status/CreateTime）。
func MaterialGetAsset(assetId string) (*MaterialAssetResult, error) {
	// 请求 Body：{"Id": "素材资产ID字符串，必填"}
	payload := map[string]string{"Id": assetId}
	var result MaterialAssetResult
	if err := doMaterialRequest(materialActionGetAsset, payload, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.Id) == "" {
		return nil, errors.New("素材库未返回有效素材信息")
	}
	result.Status = NormalizeMaterialStatus(result.Status)
	return &result, nil
}

// MaterialPollAsset 轮询 GetAsset，直到上游素材处理完成或达到最大次数。
// sourceURL 为 CreateAsset 传入的原始 URL，用于判断本地上传是否已拿到永久地址。
func MaterialPollAsset(assetId, sourceURL string) (*MaterialAssetResult, error) {
	var last *MaterialAssetResult
	var lastErr error
	for attempt := 0; attempt < materialGetAssetMaxAttempts; attempt++ {
		info, err := MaterialGetAsset(assetId)
		if err != nil {
			lastErr = err
			if attempt < materialGetAssetMaxAttempts-1 {
				time.Sleep(materialGetAssetPollInterval)
				continue
			}
			return nil, err
		}
		last = info
		if !shouldContinueMaterialPoll(info, sourceURL) {
			return info, nil
		}
		if attempt < materialGetAssetMaxAttempts-1 {
			time.Sleep(materialGetAssetPollInterval)
		}
	}
	if last != nil {
		return last, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("查询素材信息超时")
}

// MaterialDeleteAsset 删除素材（DeleteAsset）。
// 入参：待删除资产 ID（必填）。返回 Result.Id（本次删除成功的资产 ID）。
func MaterialDeleteAsset(assetId string) (string, error) {
	// 请求 Body：{"Id": "待删除资产ID，必填"}
	payload := map[string]string{"Id": assetId}
	var result struct {
		Id string `json:"Id"`
	}
	if err := doMaterialRequest(materialActionDeleteAsset, payload, &result); err != nil {
		return "", err
	}
	return result.Id, nil
}

// MaterialDeleteAssetGroup 删除素材分组（DeleteAssetGroup）。
// 入参：待删除分组 ID（必填）。返回 Result.Id（本次删除成功的分组 ID）。
func MaterialDeleteAssetGroup(groupId string) (string, error) {
	payload := map[string]string{"Id": groupId}
	var result struct {
		Id string `json:"Id"`
	}
	if err := doMaterialRequest(materialActionDeleteAssetGroup, payload, &result); err != nil {
		return "", err
	}
	return result.Id, nil
}

// ---------------------------------------------------------------------------
// 真人认证会话（CreateVisualValidateSession / GetVisualValidateResult）
// ---------------------------------------------------------------------------

// VisualValidateSessionResult CreateVisualValidateSession 返回的认证会话信息。
// BytedToken 仅用于后端轮询，禁止返回给前端（Web 控制台路由）；
// 个人 API 令牌路由可直接返回供程序化客户端轮询。
type VisualValidateSessionResult struct {
	BytedToken string `json:"BytedToken"` // 轮询令牌（后端存储，前端轮询使用 session_id）
	H5Link     string `json:"H5Link"`     // H5 认证页面链接（5 分钟有效）
	QrCode     string `json:"QrCode"`     // H5 链接的二维码（base64 PNG）
}

// visualValidateResultRaw GetVisualValidateResult 原始返回结构。
// 认证成功时 Result.GroupId 非空；未完成时上游返回 Error。
type visualValidateResultRaw struct {
	GroupId string `json:"GroupId"`
	Error   *struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"Error"`
}

// VisualValidateResult GetVisualValidateResult 的结构化轮询结果。
// Status 取值：success（GroupId 非空）/ pending（认证中）/ failed（人脸核验失败）。
type VisualValidateResult struct {
	Status  string // VisualSessionStatusSuccess / VisualSessionStatusPending / VisualSessionStatusFailed
	GroupId string
	Message string
}

// 视觉认证会话状态枚举（与 model.VisualSessionStatus 对齐，service 层独立定义避免循环依赖）。
const (
	visualValidateSuccess = "success"
	visualValidatePending = "pending"
	visualValidateFailed  = "failed"
)

// isFaceVerificationFailure 判断上游错误信息是否为人脸核验失败（启发式匹配关键词）。
// 官方文档未提供离散状态码，认证未完成时返回 Error.Message，此处按关键词区分"失败"与"认证中"。
func isFaceVerificationFailure(msg string) bool {
	lower := strings.ToLower(msg)
	for _, kw := range []string{"人脸", "活体", "liveness", "face", "核验失败", "认证失败", "verification failed", "verification_fail"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// MaterialCreateVisualValidateSession 创建真人认证会话（CreateVisualValidateSession）。
// 空请求体，返回 BytedToken、H5Link、QrCode。
func MaterialCreateVisualValidateSession() (*VisualValidateSessionResult, error) {
	var result VisualValidateSessionResult
	// 空请求体：官方文档要求 POST 无需参数。
	if err := doMaterialRequest(materialActionCreateVisualValidateSession, map[string]any{}, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.BytedToken) == "" {
		return nil, errors.New("素材库未返回有效的认证令牌")
	}
	return &result, nil
}

// MaterialGetVisualValidateResult 查询真人认证结果（GetVisualValidateResult）。
// 入参：CreateVisualValidateSession 返回的 BytedToken。
// 返回结构化状态：success（GroupId 非空）/ failed（人脸核验失败）/ pending（认证中，含网络/上游错误）。
func MaterialGetVisualValidateResult(bytedToken string) (*VisualValidateResult, error) {
	payload := map[string]string{"BytedToken": bytedToken}
	var result visualValidateResultRaw
	if err := doMaterialRequest(materialActionGetVisualValidateResult, payload, &result); err != nil {
		// 轮询期间网络/上游错误均视为"认证中"，交由调用方按 3s 间隔重试。
		// 不携带原始错误信息：上游在认证未完成时会返回 C500999 等内部错误，
		// 这些错误对终端用户无意义且会引起误解（如"获取素材组ID失败，请重新认证"）。
		return &VisualValidateResult{Status: visualValidatePending}, nil
	}
	// 认证成功：GroupId 非空。
	if groupId := strings.TrimSpace(result.GroupId); groupId != "" {
		return &VisualValidateResult{Status: visualValidateSuccess, GroupId: groupId}, nil
	}
	// 认证未完成：上游返回 Error.Message。
	if result.Error != nil && strings.TrimSpace(result.Error.Message) != "" {
		if isFaceVerificationFailure(result.Error.Message) {
			return &VisualValidateResult{Status: visualValidateFailed, Message: result.Error.Message}, nil
		}
		// 非人脸核验失败的错误（如 C500999 "获取素材组ID失败"）属于认证中状态，
		// 不向上层透传原始错误信息，避免在 UI 展示内部错误码。
		return &VisualValidateResult{Status: visualValidatePending}, nil
	}
	// 无 GroupId 也无 Error：视为认证中。
	return &VisualValidateResult{Status: visualValidatePending}, nil
}

// MaterialUpdateAssetGroup 更新素材组基础信息（UpdateAssetGroup）。
// 入参 Id 必填；Name/Description 选填，至少传一项。
func MaterialUpdateAssetGroup(groupId string, name string, description string) (string, error) {
	payload := map[string]string{"Id": groupId}
	if name != "" {
		payload["Name"] = name
	}
	if description != "" {
		payload["Description"] = description
	}
	var result struct {
		Id string `json:"Id"`
	}
	if err := doMaterialRequest(materialActionUpdateAssetGroup, payload, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Id) == "" {
		return groupId, nil
	}
	return result.Id, nil
}

// MaterialUpdateAsset 更新素材信息（UpdateAsset）。
// 入参 Id 必填；Name 选填。
func MaterialUpdateAsset(assetId string, name string) (string, error) {
	payload := map[string]string{"Id": assetId}
	if name != "" {
		payload["Name"] = name
	}
	var result struct {
		Id string `json:"Id"`
	}
	if err := doMaterialRequest(materialActionUpdateAsset, payload, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Id) == "" {
		return assetId, nil
	}
	return result.Id, nil
}
