package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// 素材库接口 Action 常量。
const (
	materialActionCreateAssetGroup = "CreateAssetGroup"
	materialActionGetAssetGroup    = "GetAssetGroup"
	materialActionCreateAsset      = "CreateAsset"
	materialActionGetAsset         = "GetAsset"

	// MaterialAssetTypeImage 图片素材类型。
	MaterialAssetTypeImage = "Image"
)

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

// MaterialGetAsset 查询单个素材信息（用于状态刷新）。
func MaterialGetAsset(assetId string) (*MaterialAssetResult, error) {
	payload := map[string]string{"Id": assetId}
	var result MaterialAssetResult
	if err := doMaterialRequest(materialActionGetAsset, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
