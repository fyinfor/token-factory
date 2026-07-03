package setting

const DefaultAntomGatewayURL = "https://open-sea-global.alipay.com"

var (
	AntomEnabled             bool
	AntomSandbox             bool
	AntomClientId            string
	AntomPublicKey           string
	AntomPrivateKey          string
	AntomSandboxClientId     string
	AntomSandboxPublicKey    string
	AntomSandboxPrivateKey   string
	AntomGatewayUrl          string = DefaultAntomGatewayURL
	AntomNotifyUrl           string
	AntomReturnUrl           string
	AntomCurrency            string  = "USD"
	AntomSettlementCurrency  string  = "USD"
	AntomUnitPrice           float64 = 1.0
	AntomMinTopUp            int     = 1
)

// IsAntomConfigured 判断当前环境（生产/沙盒）下 Antom 必要凭证是否已配置完整。
func IsAntomConfigured() bool {
	if !AntomEnabled {
		return false
	}
	if AntomSandbox {
		return AntomSandboxClientId != "" &&
			AntomSandboxPublicKey != "" &&
			AntomSandboxPrivateKey != ""
	}
	return AntomClientId != "" &&
		AntomPublicKey != "" &&
		AntomPrivateKey != ""
}

// GetAntomClientId 返回当前环境对应的 Client ID。
func GetAntomClientId() string {
	if AntomSandbox {
		return AntomSandboxClientId
	}
	return AntomClientId
}

// GetAntomPublicKey 返回当前环境对应的 Antom 公钥。
func GetAntomPublicKey() string {
	if AntomSandbox {
		return AntomSandboxPublicKey
	}
	return AntomPublicKey
}

// GetAntomPrivateKey 返回当前环境对应的商户私钥。
func GetAntomPrivateKey() string {
	if AntomSandbox {
		return AntomSandboxPrivateKey
	}
	return AntomPrivateKey
}

// GetAntomGatewayUrl 返回 Antom 网关地址，未配置时使用亚洲区默认域名。
func GetAntomGatewayUrl() string {
	if AntomGatewayUrl == "" {
		return DefaultAntomGatewayURL
	}
	return AntomGatewayUrl
}
