package setting

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// U币支付（虚拟币充值）相关配置。
// 充值流程：调用 BaseUrl + /api/generateAddress 生成收款地址，用户转账后由回调入账。
var (
	UcoinEnabled    bool
	UcoinBaseUrl    string
	UcoinMerchantId string
	UcoinApiKey     string // 签名密钥（写入式敏感字段，不回显）
	UcoinMinTopUp   int    = 1
	UcoinNotifyUrl  string // 回调地址，留空则使用 服务器地址 + /api/user/ubcoin/notify
)

// UcoinCoinPair 表示一组可选的主币种/子币种编号。
// 管理端可配置多组，供用户在充值时选择。
type UcoinCoinPair struct {
	Name         string `json:"name"`         // 展示名称，如 USDT-TRC20
	MainCoinType int    `json:"mainCoinType"` // 主币种编号
	CoinType     string `json:"coinType"`     // 子币种编号/地址（字符串，可为链上地址或编号）
	Network      string `json:"network"`      // 网络展示，如 BNB Chain、Ethereum
	Currency     string `json:"currency"`     // 币种展示，如 USDT
}

// GetUcoinCoinPairs 从 OptionMap 读取币种对配置。
func GetUcoinCoinPairs() []UcoinCoinPair {
	common.OptionMapRWMutex.RLock()
	jsonStr := common.OptionMap["UcoinCoinPairs"]
	common.OptionMapRWMutex.RUnlock()

	if jsonStr == "" {
		return []UcoinCoinPair{}
	}
	return parseUcoinCoinPairs(jsonStr)
}

func parseUcoinCoinPairs(jsonStr string) []UcoinCoinPair {
	var raw []struct {
		Name         string      `json:"name"`
		MainCoinType int         `json:"mainCoinType"`
		CoinType     interface{} `json:"coinType"`
		Network      string      `json:"network"`
		Currency     string      `json:"currency"`
	}
	if err := common.UnmarshalJsonStr(jsonStr, &raw); err != nil {
		return []UcoinCoinPair{}
	}
	pairs := make([]UcoinCoinPair, 0, len(raw))
	for _, item := range raw {
		name := strings.TrimSpace(item.Name)
		network := strings.TrimSpace(item.Network)
		currency := strings.TrimSpace(item.Currency)
		if network == "" {
			network = ucoinInferNetwork(name)
		}
		if currency == "" {
			currency = ucoinInferCurrency(name)
		}
		pairs = append(pairs, UcoinCoinPair{
			Name:         name,
			MainCoinType: item.MainCoinType,
			CoinType:     normalizeUcoinCoinType(item.CoinType),
			Network:      network,
			Currency:     currency,
		})
	}
	return pairs
}

func normalizeUcoinCoinType(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func ucoinInferNetwork(name string) string {
	upper := strings.ToUpper(name)
	switch {
	case strings.Contains(upper, "BNB"), strings.Contains(upper, "BSC"):
		return "BNB Chain"
	case strings.Contains(upper, "ETH"), strings.Contains(upper, "ERC"):
		return "Ethereum"
	case strings.Contains(upper, "ERP"), strings.Contains(upper, "EORAPTOR"):
		return "Eoraptor"
	case strings.Contains(upper, "TRC"), strings.Contains(upper, "TRON"):
		return "TRON"
	default:
		if name != "" {
			return name
		}
		return "—"
	}
}

func ucoinInferCurrency(name string) string {
	upper := strings.ToUpper(name)
	if strings.Contains(upper, "USDT") {
		return "USDT"
	}
	if strings.Contains(upper, "USDC") {
		return "USDC"
	}
	return "USDT"
}

// UcoinCoinPairs2JsonString 返回默认币种对 JSON（供 InitOptionMap 使用）。
// 注意：InitOptionMap 已持有 OptionMap 写锁，此处不能再 RLock OptionMap，否则会死锁导致启动卡住。
func UcoinCoinPairs2JsonString() string {
	return "[]"
}
