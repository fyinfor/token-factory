package service

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/google/uuid"
)

const aliyunSMSAPIEndpoint = "https://dysmsapi.aliyuncs.com/"

// AliyunSMSConfig 阿里云短信发送配置。
type AliyunSMSConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
}

// LoadAliyunSMSConfig 读取阿里云短信配置（优先系统设置，环境变量兜底）。
func LoadAliyunSMSConfig() (*AliyunSMSConfig, error) {
	accessKeyID := strings.TrimSpace(common.SMSAccessKeyID)
	if accessKeyID == "" {
		accessKeyID = strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_ID"))
	}
	accessKeySecret := strings.TrimSpace(common.SMSAccessKeySecret)
	if accessKeySecret == "" {
		accessKeySecret = strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_SECRET"))
	}
	signName := strings.TrimSpace(common.SMSCodeSignName)
	if signName == "" {
		signName = strings.TrimSpace(os.Getenv("ALIYUN_SMS_SIGN_NAME"))
	}
	templateCode := strings.TrimSpace(common.SMSCodeTemplateCode)
	if templateCode == "" {
		templateCode = strings.TrimSpace(os.Getenv("ALIYUN_SMS_TEMPLATE_CODE"))
	}
	cfg := &AliyunSMSConfig{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		SignName:        signName,
		TemplateCode:    templateCode,
	}
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("短信服务未配置 AccessKey，请在系统设置填写“短信API账号/短信API密钥”或设置 ALIYUN_SMS_ACCESS_KEY_ID / ALIYUN_SMS_ACCESS_KEY_SECRET")
	}
	if cfg.SignName == "" {
		return nil, fmt.Errorf("短信服务未配置签名，请在系统设置填写“短信签名”或设置 ALIYUN_SMS_SIGN_NAME")
	}
	if cfg.TemplateCode == "" {
		return nil, fmt.Errorf("短信服务未配置模板，请在系统设置填写“短信模板Code”（SMSCodeTemplateCode）或设置 ALIYUN_SMS_TEMPLATE_CODE")
	}
	return cfg, nil
}

// SendAliyunSMSCode 通过阿里云短信服务发送验证码短信。
func SendAliyunSMSCode(phone, code string) error {
	cfg, err := LoadAliyunSMSConfig()
	if err != nil {
		return err
	}
	templateParamBytes, err := common.Marshal(map[string]string{
		"code": code,
	})
	if err != nil {
		return fmt.Errorf("构造短信模板参数失败: %w", err)
	}
	params := map[string]string{
		"Action":           "SendSms",
		"Format":           "JSON",
		"Version":          "2017-05-25",
		"AccessKeyId":      cfg.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.NewString(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"RegionId":         "cn-hangzhou",
		"PhoneNumbers":     phone,
		"SignName":         cfg.SignName,
		"TemplateCode":     cfg.TemplateCode,
		"TemplateParam":    string(templateParamBytes),
	}

	signature, err := aliyunSignRPCRequest(params, cfg.AccessKeySecret)
	if err != nil {
		return err
	}

	values := url.Values{}
	values.Set("Signature", signature)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		values.Set(k, params[k])
	}

	reqURL := aliyunSMSAPIEndpoint + "?" + values.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("构建短信请求失败: %w", err)
	}
	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return fmt.Errorf("短信请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("短信发送失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析短信服务响应失败: %w", err)
	}
	if strings.ToUpper(result.Code) != "OK" {
		return fmt.Errorf("短信发送失败: %s", strings.TrimSpace(result.Message))
	}
	return nil
}

// aliyunSignRPCRequest 按阿里云 RPC 协议计算 Signature。
func aliyunSignRPCRequest(params map[string]string, accessKeySecret string) (string, error) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, aliyunPercentEncode(k)+"="+aliyunPercentEncode(params[k]))
	}
	canonicalizedQuery := strings.Join(pairs, "&")
	stringToSign := "GET&%2F&" + aliyunPercentEncode(canonicalizedQuery)
	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// aliyunPercentEncode 采用阿里云要求的 RFC3986 百分号编码规则。
func aliyunPercentEncode(s string) string {
	escaped := url.QueryEscape(s)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}
