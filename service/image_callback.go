package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const imageCallbackTimeout = 30 * time.Second

// ValidateImageCallbackURL 校验图片异步回调地址（含 SSRF 防护）。
func ValidateImageCallbackURL(callbackURL string) error {
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return fmt.Errorf("callback_url is empty")
	}
	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(callbackURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		return fmt.Errorf("invalid callback_url: %w", err)
	}
	return nil
}

// BuildImageSuccessCallbackPayload 将上游图片响应合并进回调负载，确保 edits/generations 的图片结果都能回传。
func BuildImageSuccessCallbackPayload(requestID string, created int64, responseBody []byte) *dto.ImageCallbackPayload {
	payload := &dto.ImageCallbackPayload{
		ID:      requestID,
		Created: created,
		Status:  dto.ImageCallbackStatusSuccess,
	}
	if len(responseBody) == 0 {
		return payload
	}

	var imageResp dto.ImageResponse
	if err := common.Unmarshal(responseBody, &imageResp); err == nil {
		payload.Data = imageResp.Data
		payload.Metadata = imageResp.Metadata
		if imageResp.Created > 0 {
			payload.Created = imageResp.Created
		}
	}

	// 即使标准 ImageResponse 解析不到 data，也尽量从原始 JSON 提取图片字段，
	// 并把上游其余字段（如 usage）平铺进回调，兼容 edits 等非完全标准回包。
	var rawMap map[string]json.RawMessage
	if err := common.Unmarshal(responseBody, &rawMap); err != nil {
		return payload
	}
	if createdRaw, ok := rawMap["created"]; ok {
		var createdVal int64
		if common.Unmarshal(createdRaw, &createdVal) == nil && createdVal > 0 {
			payload.Created = createdVal
		}
	}
	if !imageDataHasContent(payload.Data) {
		if extracted := extractImageDataFromRawMap(rawMap); len(extracted) > 0 {
			payload.Data = extracted
		}
	}
	if len(payload.Metadata) == 0 {
		if metaRaw, ok := rawMap["metadata"]; ok {
			payload.Metadata = metaRaw
		}
	}

	reserved := map[string]struct{}{
		"id": {}, "created": {}, "status": {}, "data": {}, "metadata": {}, "error": {},
	}
	extra := make(map[string]json.RawMessage)
	for k, v := range rawMap {
		if _, skip := reserved[k]; skip {
			continue
		}
		extra[k] = v
	}
	if len(extra) > 0 {
		payload.Extra = extra
	}
	return payload
}

func extractImageDataFromRawMap(rawMap map[string]json.RawMessage) []dto.ImageData {
	candidates := []string{"data", "images", "output", "result", "results"}
	for _, key := range candidates {
		raw, ok := rawMap[key]
		if !ok || len(raw) == 0 {
			continue
		}
		if items := parseImageDataList(raw); len(items) > 0 {
			return items
		}
	}
	return nil
}

func parseImageDataList(raw json.RawMessage) []dto.ImageData {
	var standard []dto.ImageData
	if err := common.Unmarshal(raw, &standard); err == nil && imageDataHasContent(standard) {
		return standard
	}

	var asStrings []string
	if err := common.Unmarshal(raw, &asStrings); err == nil {
		out := make([]dto.ImageData, 0, len(asStrings))
		for _, item := range asStrings {
			if d := imageDataFromString(item); d != nil {
				out = append(out, *d)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	var asMaps []map[string]any
	if err := common.Unmarshal(raw, &asMaps); err == nil {
		out := make([]dto.ImageData, 0, len(asMaps))
		for _, item := range asMaps {
			if d := imageDataFromMap(item); d != nil {
				out = append(out, *d)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	var singleMap map[string]any
	if err := common.Unmarshal(raw, &singleMap); err == nil {
		if d := imageDataFromMap(singleMap); d != nil {
			return []dto.ImageData{*d}
		}
	}
	return nil
}

func imageDataHasContent(items []dto.ImageData) bool {
	for _, item := range items {
		if strings.TrimSpace(item.Url) != "" || strings.TrimSpace(item.B64Json) != "" {
			return true
		}
	}
	return false
}

func imageDataFromString(v string) *dto.ImageData {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return &dto.ImageData{Url: v}
	}
	return &dto.ImageData{B64Json: v}
}

func imageDataFromMap(item map[string]any) *dto.ImageData {
	if item == nil {
		return nil
	}
	data := &dto.ImageData{}
	if v, ok := item["url"].(string); ok {
		data.Url = strings.TrimSpace(v)
	}
	if v, ok := item["b64_json"].(string); ok {
		data.B64Json = strings.TrimSpace(v)
	}
	if data.B64Json == "" {
		if v, ok := item["b64Json"].(string); ok {
			data.B64Json = strings.TrimSpace(v)
		}
	}
	if data.Url == "" {
		if v, ok := item["image_url"].(string); ok {
			data.Url = strings.TrimSpace(v)
		}
	}
	if data.Url == "" {
		if v, ok := item["image"].(string); ok {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				data.Url = strings.TrimSpace(v)
			} else if data.B64Json == "" {
				data.B64Json = strings.TrimSpace(v)
			}
		}
	}
	if v, ok := item["revised_prompt"].(string); ok {
		data.RevisedPrompt = v
	}
	if data.Url == "" && data.B64Json == "" {
		return nil
	}
	return data
}

// PostImageCallback 向用户 callback_url 推送图片异步结果。
func PostImageCallback(callbackURL string, payload *dto.ImageCallbackPayload) error {
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return fmt.Errorf("callback_url is empty")
	}
	if payload == nil {
		return fmt.Errorf("callback payload is nil")
	}
	if err := ValidateImageCallbackURL(callbackURL); err != nil {
		return err
	}

	body, err := common.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal image callback payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create image callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := GetHttpClient()
	if client == nil {
		client = &http.Client{Timeout: imageCallbackTimeout}
	} else {
		cloned := *client
		if cloned.Timeout == 0 || cloned.Timeout > imageCallbackTimeout {
			cloned.Timeout = imageCallbackTimeout
		}
		client = &cloned
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post image callback: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("image callback returned status %d", resp.StatusCode)
	}
	return nil
}
