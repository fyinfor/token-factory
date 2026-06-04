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

// 阿里云短信国内/国际端点。
// 国内：dysmsapi.aliyuncs.com  (Action=SendSms)
// 国际：dysmsapi.ap-southeast-1.aliyuncs.com  (Action=SendMessageToGlobe)
const (
	aliyunSMSAPIDomesticEndpoint      = "https://dysmsapi.aliyuncs.com/"
	aliyunSMSAPIInternationalEndpoint = "https://dysmsapi.ap-southeast-1.aliyuncs.com/"
	aliyunSMSAPIDomesticVersion       = "2017-05-25"
	aliyunSMSAPIInternationalVersion  = "2018-05-01"
	aliyunSMSAPIDomesticRegion        = "cn-hangzhou"
	aliyunSMSAPIInternationalRegion   = "ap-southeast-1"
	aliyunSMSAPIDomesticAction        = "SendSms"
	aliyunSMSAPIInternationalAction   = "SendMessageToGlobe"
)

// AliyunSMSConfig 阿里云短信发送配置。
type AliyunSMSConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string
	Version         string
	RegionID        string
	Action          string
	International   bool
}

// LoadAliyunSMSConfig 读取阿里云短信配置（region = domestic / international；优先系统设置，环境变量兜底）。
func LoadAliyunSMSConfig(region common.SMSRegion) (*AliyunSMSConfig, error) {
	var (
		accessKeyID     string
		accessKeySecret string
		signName        string
		templateCode    string
		endpoint        string
		version         string
		regionID        string
		action          string
		international   bool
	)

	if region == common.SMSRegionInternational {
		international = true
		accessKeyID = strings.TrimSpace(common.SMSAccessKeyIDIntl)
		if accessKeyID == "" {
			accessKeyID = strings.TrimSpace(os.Getenv("ALIYUN_SMS_INTL_ACCESS_KEY_ID"))
		}
		if accessKeyID == "" {
			accessKeyID = strings.TrimSpace(common.SMSAccessKeyID)
		}
		accessKeySecret = strings.TrimSpace(common.SMSAccessKeySecretIntl)
		if accessKeySecret == "" {
			accessKeySecret = strings.TrimSpace(os.Getenv("ALIYUN_SMS_INTL_ACCESS_KEY_SECRET"))
		}
		if accessKeySecret == "" {
			accessKeySecret = strings.TrimSpace(common.SMSAccessKeySecret)
		}
		// 国际包：使用 SendMessageToGlobe，不依赖 SignName / TemplateCode（直接传 Message）。
		signName = strings.TrimSpace(common.SMSCodeSignNameIntl)
		if signName == "" {
			signName = strings.TrimSpace(os.Getenv("ALIYUN_SMS_INTL_SIGN_NAME"))
		}
		templateCode = strings.TrimSpace(common.SMSCodeTemplateCodeIntl)
		if templateCode == "" {
			templateCode = strings.TrimSpace(os.Getenv("ALIYUN_SMS_INTL_TEMPLATE_CODE"))
		}
		endpoint = aliyunSMSAPIInternationalEndpoint
		version = aliyunSMSAPIInternationalVersion
		regionID = aliyunSMSAPIInternationalRegion
		action = aliyunSMSAPIInternationalAction
	} else {
		international = false
		accessKeyID = strings.TrimSpace(common.SMSAccessKeyID)
		if accessKeyID == "" {
			accessKeyID = strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_ID"))
		}
		accessKeySecret = strings.TrimSpace(common.SMSAccessKeySecret)
		if accessKeySecret == "" {
			accessKeySecret = strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_SECRET"))
		}
		signName = strings.TrimSpace(common.SMSCodeSignName)
		if signName == "" {
			signName = strings.TrimSpace(os.Getenv("ALIYUN_SMS_SIGN_NAME"))
		}
		templateCode = strings.TrimSpace(common.SMSCodeTemplateCode)
		if templateCode == "" {
			templateCode = strings.TrimSpace(os.Getenv("ALIYUN_SMS_TEMPLATE_CODE"))
		}
		endpoint = aliyunSMSAPIDomesticEndpoint
		version = aliyunSMSAPIDomesticVersion
		regionID = aliyunSMSAPIDomesticRegion
		action = aliyunSMSAPIDomesticAction
	}

	cfg := &AliyunSMSConfig{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		SignName:        signName,
		TemplateCode:    templateCode,
		Endpoint:        endpoint,
		Version:         version,
		RegionID:        regionID,
		Action:          action,
		International:   international,
	}
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("短信服务未配置 AccessKey，请在系统设置填写“短信API账号/短信API密钥”或设置 ALIYUN_SMS_ACCESS_KEY_ID / ALIYUN_SMS_ACCESS_KEY_SECRET")
	}
	if !cfg.International {
		if cfg.SignName == "" {
			return nil, fmt.Errorf("短信服务未配置签名，请在系统设置填写“短信签名”或设置 ALIYUN_SMS_SIGN_NAME")
		}
		if cfg.TemplateCode == "" {
			return nil, fmt.Errorf("短信服务未配置模板，请在系统设置填写“短信模板Code”（SMSCodeTemplateCode）或设置 ALIYUN_SMS_TEMPLATE_CODE")
		}
	}
	return cfg, nil
}

// SendAliyunSMSCode 旧接口（保留）：按手机号自动判定国内/国际发送。
func SendAliyunSMSCode(phone, code string) error {
	region := common.ClassifyPhoneRegion(phone)
	switch region {
	case common.SMSRegionDomestic:
		return SendAliyunSMSCodeWithRegion(phone, code, common.SMSRegionDomestic)
	case common.SMSRegionInternational:
		return SendAliyunSMSCodeWithRegion(phone, code, common.SMSRegionInternational)
	default:
		return fmt.Errorf("不支持的手机号格式")
	}
}

// SendAliyunSMSCodeWithRegion 显式指定地区发送。
func SendAliyunSMSCodeWithRegion(phone, code string, region common.SMSRegion) error {
	cfg, err := LoadAliyunSMSConfig(region)
	if err != nil {
		return err
	}
	return sendAliyunSMS(cfg, phone, code, pickInternationalMessage(phone, code))
}

func sendAliyunSMS(cfg *AliyunSMSConfig, phone, code, intlMessage string) error {
	if cfg.International {
		return sendAliyunSMSInternational(cfg, phone, intlMessage)
	}
	return sendAliyunSMSDomestic(cfg, phone, code)
}

// pickInternationalMessage 根据手机号所属国家/地区返回对应的国际短信文案。
// 美国 / 加拿大 用英文强提示；
// 台湾 用中文提示；
// 其它国际号码 用通用英文。
// 文案中可使用 %s 占位符，由本函数填充验证码。
func pickInternationalMessage(phone, code string) string {
	dial := extractDialCode(phone)
	switch dial {
	case "1": // 北美：US / CA / 加勒比 等都以 +1 开头，统一走英文
		return fmt.Sprintf("Your verification code is: %s. Please do not disclose it to others!", code)
	case "886": // 台湾
		return fmt.Sprintf("您的验证码: %s，您正在进行身份验证，5分钟内有效，请忽泄露于他人!", code)
	default:
		return fmt.Sprintf("Your verification code is %s. Valid for 5 minutes.", code)
	}
}

// extractDialCode 从 + 国码+号码 中抽出国码（如 +85295931349 → "852"，+13800000000 → "1"）。
// 美国 / 加拿大 都是 +1，无法仅靠国码区分，统一走同一英文模板即可。
func extractDialCode(phone string) string {
	p := strings.TrimPrefix(phone, "+")
	// 按长度优先匹配：1-3 位
	for _, l := range []int{3, 2, 1} {
		if len(p) > l && p[0] != '0' {
			return p[:l]
		}
	}
	return p
}

// sendAliyunSMSDomestic 国内：Action=SendSms，模板 + 模板变量。
func sendAliyunSMSDomestic(cfg *AliyunSMSConfig, phone, code string) error {
	templateParamBytes, err := common.Marshal(map[string]string{
		"code": code,
	})
	if err != nil {
		return fmt.Errorf("构造短信模板参数失败: %w", err)
	}
	params := map[string]string{
		"Action":           cfg.Action,
		"Format":           "JSON",
		"Version":          cfg.Version,
		"AccessKeyId":      cfg.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.NewString(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"RegionId":         cfg.RegionID,
		"PhoneNumbers":     phone,
		"SignName":         cfg.SignName,
		"TemplateCode":     cfg.TemplateCode,
		"TemplateParam":    string(templateParamBytes),
	}
	body, err := callAliyunRPC(cfg.Endpoint, cfg.AccessKeySecret, params)
	if err != nil {
		return err
	}
	return parseAliyunResponse(body)
}

// sendAliyunSMSInternational 国际：Action=SendMessageToGlobe，直接传 Message 完整文本。
// 接收方号码格式：CountryCode+Number（不带 +）。如香港 852 + 95931349 → "85295931349"。
func sendAliyunSMSInternational(cfg *AliyunSMSConfig, phone, message string) error {
	// 标准化：去掉 + 号，保留国码 + 本地号
	to := strings.TrimPrefix(phone, "+")
	params := map[string]string{
		"Action":           cfg.Action,
		"Format":           "JSON",
		"Version":          cfg.Version,
		"AccessKeyId":      cfg.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.NewString(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"RegionId":         cfg.RegionID,
		"To":               to,
		"Message":          message,
	}
	body, err := callAliyunRPC(cfg.Endpoint, cfg.AccessKeySecret, params)
	if err != nil {
		return err
	}
	return parseAliyunResponse(body)
}

// callAliyunRPC 公共：签名 + GET 请求阿里云 OpenAPI，返回响应 body。
func callAliyunRPC(endpoint, accessKeySecret string, params map[string]string) ([]byte, error) {
	signature, err := aliyunSignRPCRequest(params, accessKeySecret)
	if err != nil {
		return nil, err
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
	reqURL := endpoint + "?" + values.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建短信请求失败: %w", err)
	}
	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("短信请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("短信发送失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// parseAliyunResponse 解析阿里云响应：SendSms 用 Code/Message；SendMessageToGlobe 用 ResponseCode/ResponseDescription。
func parseAliyunResponse(body []byte) error {
	// 尝试 SendSms 响应格式
	var smsResp struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := common.Unmarshal(body, &smsResp); err == nil && smsResp.Code != "" {
		if strings.EqualFold(smsResp.Code, "OK") {
			return nil
		}
		return fmt.Errorf("短信发送失败: %s", strings.TrimSpace(smsResp.Message))
	}
	// 尝试 SendMessageToGlobe 响应格式
	var intlResp struct {
		ResponseCode        string `json:"ResponseCode"`
		ResponseDescription string `json:"ResponseDescription"`
		Message             string `json:"Message"`
	}
	if err := common.Unmarshal(body, &intlResp); err == nil {
		// 顶层错误信息（如 SignatureDoesNotMatch 等）放在 Message
		if intlResp.Message != "" {
			return fmt.Errorf("短信发送失败: %s", strings.TrimSpace(intlResp.Message))
		}
		if strings.EqualFold(intlResp.ResponseCode, "OK") {
			return nil
		}
		if intlResp.ResponseCode != "" {
			return fmt.Errorf("短信发送失败: %s", strings.TrimSpace(intlResp.ResponseDescription))
		}
	}
	return fmt.Errorf("短信发送失败，无法解析响应: %s", strings.TrimSpace(string(body)))
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
