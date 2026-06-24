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
	materialActionCreateAssetGroup = "CreateAssetGroup"
	materialActionGetAssetGroup    = "GetAssetGroup"
	materialActionCreateAsset      = "CreateAsset"
	materialActionGetAsset         = "GetAsset"
	materialActionDeleteAsset      = "DeleteAsset"

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

// doMaterialRequest 向素材库 API 发送 POST JSON 请求并解析通用响应。
func doMaterialRequest(action string, payload any, result any) error {
	base := operation_setting.GetMaterialAPIBaseURL()
	if base == "" {
		return errors.New("素材库 API 基础地址未配置")
	}
	reqURL := fmt.Sprintf("%s/api/material?Action=%s", base, action)

	body, err := common.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化素材库请求失败: %w", err)
	}

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
		return fmt.Errorf("素材库接口错误: %s", wrapper.ResponseError.Message)
	}
	if result != nil && len(wrapper.Result) > 0 {
		if err := common.Unmarshal(wrapper.Result, result); err != nil {
			return fmt.Errorf("解析素材库返回结果失败: %w", err)
		}
	}
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
