package system_setting

import (
	_ "embed"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	LegalContentFormatAuto     = "auto"
	LegalContentFormatMarkdown = "markdown"
	LegalContentFormatHTML     = "html"
	LegalContentFormatRichText = "richtext"
)

type LegalSettings struct {
	UserAgreement       string `json:"user_agreement"`
	UserAgreementFormat string `json:"user_agreement_format"`
	PrivacyPolicy       string `json:"privacy_policy"`
	PrivacyPolicyFormat string `json:"privacy_policy_format"`
}

//go:embed legal_content/user-agreement.html
var defaultUserAgreement string

//go:embed legal_content/privacy-policy.html
var defaultPrivacyPolicy string

var defaultLegalSettings = LegalSettings{
	UserAgreement:       strings.TrimSpace(defaultUserAgreement),
	UserAgreementFormat: LegalContentFormatHTML,
	PrivacyPolicy:       strings.TrimSpace(defaultPrivacyPolicy),
	PrivacyPolicyFormat: LegalContentFormatHTML,
}

func init() {
	config.GlobalConfig.Register("legal", &defaultLegalSettings)
}

func GetLegalSettings() *LegalSettings {
	return &defaultLegalSettings
}

func NormalizeLegalContentFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case LegalContentFormatMarkdown:
		return LegalContentFormatMarkdown
	case LegalContentFormatHTML:
		return LegalContentFormatHTML
	case LegalContentFormatRichText:
		return LegalContentFormatRichText
	default:
		return LegalContentFormatAuto
	}
}
