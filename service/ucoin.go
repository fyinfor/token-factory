package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/thanhpk/randstr"
)

var ucoinHTTPClient = &http.Client{Timeout: 30 * time.Second}

type ucoinEnvelope struct {
	Body      string `json:"body"`
	Nonce     string `json:"nonce"`
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
}

type ucoinGenerateAddressBody struct {
	MerchantId   string `json:"merchantId"`
	MainCoinType int    `json:"mainCoinType"`
	CallUrl      string `json:"callUrl"`
}

type ucoinResponse struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
}

// UcoinConfigured U币支付是否已完整配置。
func UcoinConfigured() bool {
	return setting.UcoinEnabled &&
		setting.UcoinBaseUrl != "" &&
		setting.UcoinMerchantId != "" &&
		setting.UcoinApiKey != "" &&
		len(setting.GetUcoinCoinPairs()) > 0
}

// UcoinCallbackURL 返回 U币充币回调地址。
func UcoinCallbackURL() string {
	if strings.TrimSpace(setting.UcoinNotifyUrl) != "" {
		return strings.TrimSpace(setting.UcoinNotifyUrl)
	}
	return strings.TrimRight(GetCallbackAddress(), "/") + "/api/user/ubcoin/notify"
}

// UcoinDefaultMainCoinType 返回默认主币种编号（取配置的第一项）。
func UcoinDefaultMainCoinType() (int, error) {
	pairs := setting.GetUcoinCoinPairs()
	if len(pairs) == 0 {
		return 0, fmt.Errorf("未配置 U币币种")
	}
	return pairs[0].MainCoinType, nil
}

// UcoinGenerateAddress 调用上游 /api/generateAddress 生成收款地址。
func UcoinGenerateAddress(mainCoinType int) (string, error) {
	if !UcoinConfigured() {
		return "", fmt.Errorf("U币支付未配置")
	}
	addrResp, err := ucoinPost("/api/generateAddress", ucoinGenerateAddressBody{
		MerchantId:   setting.UcoinMerchantId,
		MainCoinType: mainCoinType,
		CallUrl:      UcoinCallbackURL(),
	})
	if err != nil {
		return "", err
	}
	if addrResp.Code != 200 {
		msg := ucoinResponseMessage(addrResp)
		if msg != "" {
			return "", fmt.Errorf("%s (code=%d)", msg, addrResp.Code)
		}
		return "", fmt.Errorf("生成地址失败 (code=%d)", addrResp.Code)
	}
	return ucoinParseAddress(addrResp.Data)
}

// EnsureUserUcoinAddress 确保用户已有 U地址；若无则向 U币申请并写入用户表。
func EnsureUserUcoinAddress(userId int) (string, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		return "", fmt.Errorf("用户不存在")
	}
	if addr := strings.TrimSpace(user.UcoinAddress); addr != "" {
		return addr, nil
	}
	mainCoinType, err := UcoinDefaultMainCoinType()
	if err != nil {
		return "", err
	}
	address, err := UcoinGenerateAddress(mainCoinType)
	if err != nil {
		return "", err
	}
	if err := model.UpdateUserUcoinAddress(userId, address); err != nil {
		return "", err
	}
	log.Printf("U币已为用户生成 U地址: userId=%d address=%s", userId, address)
	return address, nil
}

// TryProvisionUcoinAddressAsync 用户注册后异步申请 U地址（失败仅记日志，不阻塞注册）。
func TryProvisionUcoinAddressAsync(userId int) {
	if !UcoinConfigured() || userId <= 0 {
		return
	}
	go func() {
		if _, err := EnsureUserUcoinAddress(userId); err != nil {
			log.Printf("U币注册后生成 U地址失败: userId=%d err=%v", userId, err)
		}
	}()
}

func ucoinResponseMessage(resp *ucoinResponse) string {
	if resp == nil {
		return ""
	}
	if msg := strings.TrimSpace(resp.Msg); msg != "" {
		return msg
	}
	return strings.TrimSpace(resp.Message)
}

func ucoinSign(bodyBytes []byte, nonce, timestamp string) string {
	raw := string(bodyBytes) + setting.UcoinApiKey + nonce + timestamp
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func ucoinMarshalBodyString(bodyObj interface{}) (string, error) {
	bodyBytes, err := common.Marshal(bodyObj)
	if err != nil {
		return "", fmt.Errorf("请求体序列化失败: %w", err)
	}
	return string(bodyBytes), nil
}

func ucoinPost(path string, bodyObj interface{}) (*ucoinResponse, error) {
	bodyStr, err := ucoinMarshalBodyString(bodyObj)
	if err != nil {
		return nil, err
	}
	nonce := randstr.String(16)
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sign := ucoinSign([]byte(bodyStr), nonce, timestamp)

	envBytes, err := common.Marshal(ucoinEnvelope{
		Body:      bodyStr,
		Nonce:     nonce,
		Timestamp: timestamp,
		Sign:      sign,
	})
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %w", err)
	}

	reqURL := strings.TrimRight(strings.TrimSpace(setting.UcoinBaseUrl), "/") + path
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(envBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if common.DebugEnabled {
		log.Printf("U币请求 %s envelope=%s", path, string(envBytes))
	}

	resp, err := ucoinHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed ucoinResponse
	if err := common.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("响应解析失败: %s", string(respBytes))
	}
	return &parsed, nil
}

func ucoinParseAddress(data json.RawMessage) (string, error) {
	if len(data) == 0 || string(data) == "null" {
		return "", fmt.Errorf("empty address data")
	}
	var address string
	if err := common.Unmarshal(data, &address); err == nil && address != "" {
		return address, nil
	}
	var payload struct {
		Address string `json:"address"`
	}
	if err := common.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	if payload.Address == "" {
		return "", fmt.Errorf("address missing in data")
	}
	return payload.Address, nil
}
