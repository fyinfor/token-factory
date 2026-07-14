package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
)

const (
	tencentTMSEndpoint   = "https://tms.tencentcloudapi.com/"
	tencentTMSHost       = "tms.tencentcloudapi.com"
	tencentTMSService    = "tms"
	tencentTMSAction     = "TextModeration"
	tencentTMSAPIVersion = "2020-12-29"
	tencentTMSMaxRunes   = 10000
	tencentTMSTimeout    = 15 * time.Second
	tencentIMSEndpoint   = "https://ims.tencentcloudapi.com/"
	tencentIMSHost       = "ims.tencentcloudapi.com"
	tencentIMSService    = "ims"
	tencentIMSAction     = "ImageModeration"
	tencentIMSAPIVersion = "2020-12-29"
	tencentIMSMaxBase64  = 10 * 1024 * 1024
)

type TencentTMSResult struct {
	Blocked    bool
	Suggestion string
	Label      string
	SubLabel   string
	Score      int
}

type tencentTMSRequest struct {
	Content string `json:"Content"`
	BizType string `json:"BizType,omitempty"`
	Type    string `json:"Type"`
}

type tencentTMSResponse struct {
	Response struct {
		Suggestion string `json:"Suggestion"`
		Label      string `json:"Label"`
		SubLabel   string `json:"SubLabel"`
		Score      int    `json:"Score"`
		Error      *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"Response"`
}

type tencentIMSRequest struct {
	FileContent string `json:"FileContent,omitempty"`
	FileURL     string `json:"FileUrl,omitempty"`
	BizType     string `json:"BizType,omitempty"`
	Type        string `json:"Type"`
}

type TencentIMSResult struct {
	Blocked    bool
	Suggestion string
	Label      string
	SubLabel   string
	Score      int
}

func CheckTextWithTencentTMS(ctx context.Context, text string) (TencentTMSResult, error) {
	if strings.TrimSpace(text) == "" {
		return TencentTMSResult{}, nil
	}
	if setting.TencentTMSSecretID == "" || setting.TencentTMSSecretKey == "" || setting.TencentTMSRegion == "" {
		return TencentTMSResult{}, errors.New("tencent tms credentials or region are not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, tencentTMSTimeout)
	defer cancel()

	for _, chunk := range splitTextByRunes(text, tencentTMSMaxRunes) {
		result, err := moderateTencentTMSChunk(ctx, chunk)
		if err != nil {
			return TencentTMSResult{}, err
		}
		if result.Blocked {
			return result, nil
		}
	}
	return TencentTMSResult{}, nil
}

func moderateTencentTMSChunk(ctx context.Context, text string) (TencentTMSResult, error) {
	payload, err := common.Marshal(tencentTMSRequest{
		Content: base64.StdEncoding.EncodeToString([]byte(text)),
		BizType: setting.TencentTMSBizType,
		Type:    "TEXT",
	})
	if err != nil {
		return TencentTMSResult{}, fmt.Errorf("marshal tencent tms request: %w", err)
	}

	timestamp := time.Now().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tencentTMSEndpoint, bytes.NewReader(payload))
	if err != nil {
		return TencentTMSResult{}, fmt.Errorf("create tencent tms request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", tencentCloudAuthorization(setting.TencentTMSSecretID, setting.TencentTMSSecretKey, tencentTMSHost, tencentTMSService, tencentTMSAction, timestamp, payload))
	req.Header.Set("X-TC-Action", tencentTMSAction)
	req.Header.Set("X-TC-Version", tencentTMSAPIVersion)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Region", setting.TencentTMSRegion)

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return TencentTMSResult{}, fmt.Errorf("request tencent tms: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TencentTMSResult{}, fmt.Errorf("read tencent tms response: %w", err)
	}

	var response tencentTMSResponse
	if err = common.Unmarshal(body, &response); err != nil {
		return TencentTMSResult{}, fmt.Errorf("decode tencent tms response: %w", err)
	}
	if response.Response.Error != nil {
		return TencentTMSResult{}, fmt.Errorf("tencent tms error %s: %s", response.Response.Error.Code, response.Response.Error.Message)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return TencentTMSResult{}, fmt.Errorf("tencent tms returned status %d", resp.StatusCode)
	}

	suggestion := strings.TrimSpace(response.Response.Suggestion)
	return TencentTMSResult{
		Blocked:    strings.EqualFold(suggestion, "Block") || strings.EqualFold(suggestion, "Review"),
		Suggestion: suggestion,
		Label:      response.Response.Label,
		SubLabel:   response.Response.SubLabel,
		Score:      response.Response.Score,
	}, nil
}

func CheckImagesWithTencentIMS(ctx context.Context, files []*types.FileMeta) (TencentIMSResult, error) {
	if setting.TencentTMSSecretID == "" || setting.TencentTMSSecretKey == "" || setting.TencentTMSRegion == "" {
		return TencentIMSResult{}, errors.New("腾讯云 IMS 凭证或地域未配置 / Tencent Cloud IMS credentials or region are not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, tencentTMSTimeout)
	defer cancel()
	for _, file := range files {
		if file == nil || file.FileType != types.FileTypeImage || file.Source == nil {
			continue
		}
		result, err := moderateTencentIMSImage(ctx, file.Source)
		if err != nil {
			return TencentIMSResult{}, err
		}
		if result.Blocked {
			return result, nil
		}
	}
	return TencentIMSResult{}, nil
}

func moderateTencentIMSImage(ctx context.Context, source *types.FileSource) (TencentIMSResult, error) {
	requestBody := tencentIMSRequest{BizType: setting.TencentIMSBizType, Type: "IMAGE"}
	if source.IsURL() {
		requestBody.FileURL = strings.TrimSpace(source.URL)
	} else {
		requestBody.FileContent = normalizeImageBase64(source.Base64Data)
		if len(requestBody.FileContent) > tencentIMSMaxBase64 {
			return TencentIMSResult{}, errors.New("图片 Base64 超过腾讯云 IMS 10MB 限制 / image Base64 exceeds the Tencent Cloud IMS 10MB limit")
		}
	}
	if requestBody.FileURL == "" && requestBody.FileContent == "" {
		return TencentIMSResult{}, nil
	}
	payload, err := common.Marshal(requestBody)
	if err != nil {
		return TencentIMSResult{}, fmt.Errorf("序列化腾讯云 IMS 请求失败 / failed to marshal Tencent Cloud IMS request: %w", err)
	}
	timestamp := time.Now().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tencentIMSEndpoint, bytes.NewReader(payload))
	if err != nil {
		return TencentIMSResult{}, fmt.Errorf("创建腾讯云 IMS 请求失败 / failed to create Tencent Cloud IMS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", tencentCloudAuthorization(setting.TencentTMSSecretID, setting.TencentTMSSecretKey, tencentIMSHost, tencentIMSService, tencentIMSAction, timestamp, payload))
	req.Header.Set("X-TC-Action", tencentIMSAction)
	req.Header.Set("X-TC-Version", tencentIMSAPIVersion)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Region", setting.TencentTMSRegion)
	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return TencentIMSResult{}, fmt.Errorf("请求腾讯云 IMS 失败 / failed to request Tencent Cloud IMS: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TencentIMSResult{}, fmt.Errorf("读取腾讯云 IMS 响应失败 / failed to read Tencent Cloud IMS response: %w", err)
	}
	var response tencentTMSResponse
	if err = common.Unmarshal(body, &response); err != nil {
		return TencentIMSResult{}, fmt.Errorf("解析腾讯云 IMS 响应失败 / failed to decode Tencent Cloud IMS response: %w", err)
	}
	if response.Response.Error != nil {
		return TencentIMSResult{}, fmt.Errorf("腾讯云 IMS 错误 / Tencent Cloud IMS error %s: %s", response.Response.Error.Code, response.Response.Error.Message)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return TencentIMSResult{}, fmt.Errorf("腾讯云 IMS 返回状态码 %d / Tencent Cloud IMS returned status %d", resp.StatusCode, resp.StatusCode)
	}
	suggestion := strings.TrimSpace(response.Response.Suggestion)
	return TencentIMSResult{
		Blocked:    strings.EqualFold(suggestion, "Block") || strings.EqualFold(suggestion, "Review"),
		Suggestion: suggestion,
		Label:      response.Response.Label,
		SubLabel:   response.Response.SubLabel,
		Score:      response.Response.Score,
	}, nil
}

func normalizeImageBase64(data string) string {
	value := strings.TrimSpace(data)
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		return value[comma+1:]
	}
	return value
}

func splitTextByRunes(text string, maxRunes int) []string {
	if maxRunes <= 0 || utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}
	runes := []rune(text)
	chunks := make([]string, 0, (len(runes)+maxRunes-1)/maxRunes)
	for start := 0; start < len(runes); start += maxRunes {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func tencentCloudAuthorization(secretID, secretKey, host, service, action string, timestamp int64, payload []byte) string {
	canonicalHeaders := "content-type:application/json\nhost:" + host + "\nx-tc-action:" + strings.ToLower(action) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := "POST\n/\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + sha256HexBytes(payload)

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := "TC3-HMAC-SHA256\n" + strconv.FormatInt(timestamp, 10) + "\n" + credentialScope + "\n" + sha256HexBytes([]byte(canonicalRequest))

	secretDate := hmacSHA256Bytes([]byte("TC3"+secretKey), []byte(date))
	secretService := hmacSHA256Bytes(secretDate, []byte(service))
	secretSigning := hmacSHA256Bytes(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSHA256Bytes(secretSigning, []byte(stringToSign)))

	return fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", secretID, credentialScope, signedHeaders, signature)
}

func sha256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256Bytes(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}
