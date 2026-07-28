package service

import (
	"bytes"
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
