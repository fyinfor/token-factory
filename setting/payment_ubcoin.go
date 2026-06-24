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
	}
	if err := common.UnmarshalJsonStr(jsonStr, &raw); err != nil {
		return []UcoinCoinPair{}
	}
	pairs := make([]UcoinCoinPair, 0, len(raw))
	for _, item := range raw {
		pairs = append(pairs, UcoinCoinPair{
			Name:         item.Name,
			MainCoinType: item.MainCoinType,
			CoinType:     normalizeUcoinCoinType(item.CoinType),
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

// UcoinCoinPairs2JsonString 返回默认币种对 JSON（供 InitOptionMap 使用）。
// 注意：InitOptionMap 已持有 OptionMap 写锁，此处不能再 RLock OptionMap，否则会死锁导致启动卡住。
func UcoinCoinPairs2JsonString() string {
	return "[]"
}
