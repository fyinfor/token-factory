package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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

type ucoinPayRequest struct {
	Amount        int64 `json:"amount"`
	CoinPairIndex int   `json:"coin_pair_index"`
}

// ucoinDepositCallback 充币回调 body 内层字段（优盾/U币常见格式）。
type ucoinDepositCallback struct {
	Address      string      `json:"address"`
	Amount       string      `json:"amount"`
	Status       interface{} `json:"status"`
	TradeType    interface{} `json:"tradeType"`
	BusinessId   string      `json:"businessId"`
	TxId         string      `json:"txId"`
	MainCoinType string      `json:"mainCoinType"`
}

// RequestUcoinPay 发起 U币充值：使用用户 U地址创建本地订单并展示收款信息。
func RequestUcoinPay(c *gin.Context) {
	if !service.UcoinConfigured() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "U币支付未启用或未配置完整"})
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

	address, err := service.EnsureUserUcoinAddress(id)
	if err != nil {
		log.Printf("U币获取用户 U地址失败: userId=%d err=%v", id, err)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取收款地址失败"})
		return
	}

	tradeNo := fmt.Sprintf("UCOIN-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))
	topUp := &model.TopUp{
		UserId:         id,
		Amount:         float64(req.Amount),
		Money:          float64(req.Amount),
		TradeNo:        tradeNo,
		DepositAddress: address,
		PaymentMethod:  "ubcoin",
		CreateTime:     time.Now().Unix(),
		Status:         common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		log.Printf("U币创建本地订单失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	log.Printf("U币充值订单已创建 - 用户: %d, 订单: %s, U地址: %s, 数量: %d, 币种: %s",
		id, tradeNo, address, req.Amount, pair.Name)
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"address":   address,
			"amount":    req.Amount,
			"order_id":  tradeNo,
			"coin":      pair.Name,
			"network":   pair.Network,
			"currency":  pair.Currency,
			"min_topup": setting.UcoinMinTopUp,
		},
	})
}

// UcoinNotify 处理 U币充币回调通知（application/x-www-form-urlencoded + body JSON 字符串）。
func UcoinNotify(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 40125, "msg": "read body failed"})
		return
	}
	log.Printf("U币回调原始报文: %s", string(bodyBytes))

	deposit, err := ucoinParseDepositNotify(bodyBytes)
	if err != nil {
		log.Printf("U币回调解析失败: %v, raw=%s", err, string(bodyBytes))
		c.JSON(http.StatusOK, gin.H{"code": 40125, "msg": "invalid payload"})
		return
	}
	log.Printf("U币回调解析成功: address=%s amount=%s status=%v tradeType=%v businessId=%s txId=%s",
		deposit.Address, deposit.Amount, deposit.Status, deposit.TradeType, deposit.BusinessId, deposit.TxId)

	if !ucoinDepositNotifySuccess(deposit) {
		log.Printf("U币回调非成功状态, businessId=%s address=%s status=%v tradeType=%v",
			deposit.BusinessId, deposit.Address, deposit.Status, deposit.TradeType)
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "received"})
		return
	}

	tradeNo := strings.TrimSpace(deposit.BusinessId)
	if tradeNo == "" {
		tradeNo = ucoinResolveTradeNoByAddress(deposit.Address)
		if tradeNo == "" {
			log.Printf("U币回调无法匹配订单, address=%s", deposit.Address)
			c.JSON(http.StatusOK, gin.H{"code": 40101, "msg": "missing businessId"})
			return
		}
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	if err := model.RechargeUcoin(tradeNo, deposit.Amount); err != nil {
		log.Printf("U币充值处理失败: %v, 订单: %s", err, tradeNo)
		c.JSON(http.StatusOK, gin.H{"code": 10124, "msg": err.Error()})
		return
	}

	log.Printf("U币充值成功 - 订单: %s, txId: %s, address: %s", tradeNo, deposit.TxId, deposit.Address)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

func ucoinResolveTradeNoByAddress(address string) string {
	user := model.GetUserByUcoinAddress(address)
	if user != nil {
		if topUp := model.GetPendingUcoinTopUpByUserId(user.Id); topUp != nil {
			log.Printf("U币回调按用户 U地址匹配订单: userId=%d address=%s tradeNo=%s",
				user.Id, address, topUp.TradeNo)
			return topUp.TradeNo
		}
	}
	if topUp := model.GetPendingUcoinTopUpByDepositAddress(address); topUp != nil {
		log.Printf("U币回调按订单收款地址匹配: address=%s tradeNo=%s", address, topUp.TradeNo)
		return topUp.TradeNo
	}
	return ""
}

func ucoinParseDepositNotify(bodyBytes []byte) (*ucoinDepositCallback, error) {
	raw := strings.TrimSpace(string(bodyBytes))
	if raw == "" {
		return nil, fmt.Errorf("empty body")
	}

	if strings.Contains(raw, "body=") && !strings.HasPrefix(raw, "{") {
		values, err := url.ParseQuery(raw)
		if err != nil {
			return nil, fmt.Errorf("parse form: %w", err)
		}
		bodyStr := values.Get("body")
		if bodyStr == "" {
			return nil, fmt.Errorf("missing body field in form")
		}
		var deposit ucoinDepositCallback
		if err := common.Unmarshal([]byte(bodyStr), &deposit); err != nil {
			return nil, fmt.Errorf("parse body json: %w", err)
		}
		return &deposit, nil
	}

	var payload struct {
		BusinessId string          `json:"businessId"`
		Body       json.RawMessage `json:"body"`
		ucoinDepositCallback
	}
	if err := common.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, err
	}
	if len(payload.Body) > 0 && string(payload.Body) != "null" {
		var deposit ucoinDepositCallback
		if err := common.Unmarshal(payload.Body, &deposit); err != nil {
			return nil, fmt.Errorf("parse nested body: %w", err)
		}
		if deposit.BusinessId == "" {
			deposit.BusinessId = payload.BusinessId
		}
		return &deposit, nil
	}
	deposit := payload.ucoinDepositCallback
	if deposit.BusinessId == "" {
		deposit.BusinessId = payload.BusinessId
	}
	if deposit.Address == "" && deposit.Status == nil {
		return nil, fmt.Errorf("unrecognized json callback")
	}
	return &deposit, nil
}

func ucoinDepositNotifySuccess(deposit *ucoinDepositCallback) bool {
	if deposit == nil {
		return false
	}
	tradeTypeOK := ucoinNotifyStatusInt(deposit.TradeType) == 1
	if deposit.TradeType != nil && !tradeTypeOK {
		return false
	}
	return ucoinNotifySuccess(0, deposit.Status)
}

func ucoinNotifyStatusInt(v interface{}) int {
	switch t := v.(type) {
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
}

func ucoinNotifySuccess(code int, statuses ...interface{}) bool {
	if code == 200 {
		return true
	}
	for _, s := range statuses {
		switch v := s.(type) {
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "2", "3", "success", "succeed", "ok", "1", "completed", "complete":
				return true
			}
		case float64:
			if v == 2 || v == 3 || v == 1 || v == 200 {
				return true
			}
		}
	}
	return false
}
