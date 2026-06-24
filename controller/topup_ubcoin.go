package controller

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
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

// ucoinHTTPClient U币接口请求专用 HTTP 客户端。
var ucoinHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ucoinEnvelope 为 U币接口统一请求外层结构。
// 网关侧 RequestParam.body 类型为 string：body 字段传业务参数的 JSON 字符串（对象文本，非数组）。
type ucoinEnvelope struct {
	Body      string `json:"body"`
	Nonce     string `json:"nonce"`
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
}

// ucoinGenerateAddressBody /api/generateAddress 请求体。
type ucoinGenerateAddressBody struct {
	MerchantId   string `json:"merchantId"`
	MainCoinType int    `json:"mainCoinType"`
	CallUrl      string `json:"callUrl"`
}

// ucoinResponse 兼容两种返回：生成地址返回 data，提币返回 message。
type ucoinResponse struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
}

// ucoinResponseMessage 提取 U币接口错误/成功说明（兼容 msg / message 字段）。
func ucoinResponseMessage(resp *ucoinResponse) string {
	if resp == nil {
		return ""
	}
	if msg := strings.TrimSpace(resp.Msg); msg != "" {
		return msg
	}
	return strings.TrimSpace(resp.Message)
}

// ucoinSign 计算签名：md5(body + Apikey + nonce + timestamp)，32 位小写。
func ucoinSign(bodyBytes []byte, nonce, timestamp string) string {
	raw := string(bodyBytes) + setting.UcoinApiKey + nonce + timestamp
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ucoinMarshalBodyString 将业务参数序列化为 body 字符串（单个 JSON 对象文本）。
func ucoinMarshalBodyString(bodyObj interface{}) (string, error) {
	bodyBytes, err := common.Marshal(bodyObj)
	if err != nil {
		return "", fmt.Errorf("请求体序列化失败: %w", err)
	}
	return string(bodyBytes), nil
}

// ucoinPost 向 U币接口发送签名请求，返回解析后的响应。
// 签名规则：md5(bodyJSON + Apikey + nonce + timestamp)，bodyJSON 与 envelope.body 字符串一致。
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

// ucoinParseAddress 解析生成地址接口返回的 data（兼容字符串或对象）。
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

// ucoinCallbackURL 返回回调地址，优先使用配置项，否则使用 服务器地址 + 默认路径。
func ucoinCallbackURL() string {
	if strings.TrimSpace(setting.UcoinNotifyUrl) != "" {
		return strings.TrimSpace(setting.UcoinNotifyUrl)
	}
	return strings.TrimRight(service.GetCallbackAddress(), "/") + "/api/user/ubcoin/notify"
}

type ucoinPayRequest struct {
	Amount        int64 `json:"amount"`
	CoinPairIndex int   `json:"coin_pair_index"`
}

// RequestUcoinPay 发起 U币充值：生成收款地址并创建本地订单。
func RequestUcoinPay(c *gin.Context) {
	if !setting.UcoinEnabled {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "U币支付未启用"})
		return
	}
	if setting.UcoinBaseUrl == "" || setting.UcoinMerchantId == "" || setting.UcoinApiKey == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "U币支付配置不完整"})
		return
	}

	var req ucoinPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	minTopUp := int64(setting.UcoinMinTopUp)
	if req.Amount < minTopUp {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", minTopUp)})
		return
	}

	pairs := setting.GetUcoinCoinPairs()
	if len(pairs) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "管理员未配置可用币种"})
		return
	}
	if req.CoinPairIndex < 0 || req.CoinPairIndex >= len(pairs) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "无效的币种选择"})
		return
	}
	pair := pairs[req.CoinPairIndex]

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}

	tradeNo := fmt.Sprintf("UCOIN-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))
	topUp := &model.TopUp{
		UserId:        id,
		Amount:        req.Amount,
		Money:         float64(req.Amount),
		TradeNo:       tradeNo,
		PaymentMethod: "ubcoin",
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		log.Printf("U币创建本地订单失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	callbackURL := ucoinCallbackURL()

	// 第一步：生成收款地址
	addrReq := ucoinGenerateAddressBody{
		MerchantId:   setting.UcoinMerchantId,
		MainCoinType: pair.MainCoinType,
		CallUrl:      callbackURL,
	}
	log.Printf("U币生成地址请求: 订单=%s mainCoinType=%d callUrl=%s coin=%s",
		tradeNo, pair.MainCoinType, callbackURL, pair.Name)
	addrResp, err := ucoinPost("/api/generateAddress", addrReq)
	if err != nil {
		log.Printf("U币生成地址请求失败: %v, 订单: %s", err, tradeNo)
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "生成地址失败"})
		return
	}
	if addrResp.Code != 200 {
		addrMsg := ucoinResponseMessage(addrResp)
		log.Printf("U币生成地址业务失败: code=%d msg=%s data=%s, 订单: %s",
			addrResp.Code, addrMsg, string(addrResp.Data), tradeNo)
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		userMsg := "生成地址失败"
		if addrMsg != "" {
			userMsg = fmt.Sprintf("生成地址失败: %s (code=%d)", addrMsg, addrResp.Code)
		} else {
			userMsg = fmt.Sprintf("生成地址失败 (code=%d)", addrResp.Code)
		}
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": userMsg})
		return
	}
	var address string
	if address, err = ucoinParseAddress(addrResp.Data); err != nil {
		log.Printf("U币生成地址返回解析失败: %s, 订单: %s", string(addrResp.Data), tradeNo)
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "生成地址失败"})
		return
	}

	log.Printf("U币生成地址成功 - 用户: %d, 订单: %s, 地址: %s, 数量: %d", id, tradeNo, address, req.Amount)
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"address":  address,
			"amount":   req.Amount,
			"order_id": tradeNo,
			"coin":     pair.Name,
		},
	})
}

// UcoinNotify 处理 U币回调通知。
// 注意：附件接口文档未给出回调报文结构，这里做兼容解析；
// 实际对接前请与服务商确认回调字段与验签规则。
func UcoinNotify(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 40125, "msg": "read body failed"})
		return
	}
	log.Printf("U币回调原始报文: %s", string(bodyBytes))

	// 兼容多种字段命名解析回调内容。
	var payload struct {
		BusinessId string      `json:"businessId"`
		Status     interface{} `json:"status"`
		State      interface{} `json:"state"`
		Code       int         `json:"code"`
		Body       struct {
			BusinessId string      `json:"businessId"`
			Status     interface{} `json:"status"`
			State      interface{} `json:"state"`
		} `json:"body"`
	}
	if err := common.Unmarshal(bodyBytes, &payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 40125, "msg": "invalid payload"})
		return
	}

	businessId := payload.BusinessId
	if businessId == "" {
		businessId = payload.Body.BusinessId
	}
	if businessId == "" {
		c.JSON(http.StatusOK, gin.H{"code": 40101, "msg": "missing businessId"})
		return
	}

	if !ucoinNotifySuccess(payload.Code, payload.Status, payload.State, payload.Body.Status, payload.Body.State) {
		log.Printf("U币回调非成功状态, 订单: %s", businessId)
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "received"})
		return
	}

	LockOrder(businessId)
	defer UnlockOrder(businessId)

	if err := model.RechargeUcoin(businessId); err != nil {
		log.Printf("U币充值处理失败: %v, 订单: %s", err, businessId)
		c.JSON(http.StatusOK, gin.H{"code": 10124, "msg": err.Error()})
		return
	}

	log.Printf("U币充值成功 - 订单: %s", businessId)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// ucoinNotifySuccess 根据回调中的状态字段判断是否为成功状态。
func ucoinNotifySuccess(code int, statuses ...interface{}) bool {
	if code == 200 {
		return true
	}
	for _, s := range statuses {
		switch v := s.(type) {
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "2", "success", "succeed", "ok", "1", "completed", "complete":
				return true
			}
		case float64:
			if v == 2 || v == 1 || v == 200 {
				return true
			}
		}
	}
	return false
}
