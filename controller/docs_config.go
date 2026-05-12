package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

type docsConfigResponse struct {
	BrandName    string                     `json:"brandName"`
	SiteName     map[string]string          `json:"siteName"`
	LogoUrl      string                     `json:"logoUrl"`
	HomeUrl      string                     `json:"homeUrl"`
	GithubUrl    string                     `json:"githubUrl"`
	MetaKeywords []string                   `json:"metaKeywords"`
	Business     docsBusinessConfigResponse `json:"business"`
	Raw          map[string]string          `json:"raw"`
}

type docsBusinessConfigResponse struct {
	Phone       string            `json:"phone"`
	PhoneHref   string            `json:"phoneHref"`
	WorkTime    map[string]string `json:"workTime"`
	WechatQrUrl string            `json:"wechatQrUrl"`
}

func docsOptionValue(key string) string {
	return strings.TrimSpace(common.OptionMap[key])
}

func splitDocsKeywords(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		keyword := strings.TrimSpace(part)
		if keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	return keywords
}

func GetDocsConfig(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	raw := map[string]string{
		"DocsBrandName":           docsOptionValue("DocsBrandName"),
		"DocsSiteNameEn":          docsOptionValue("DocsSiteNameEn"),
		"DocsSiteNameZh":          docsOptionValue("DocsSiteNameZh"),
		"DocsSiteNameJa":          docsOptionValue("DocsSiteNameJa"),
		"DocsLogoUrl":             docsOptionValue("DocsLogoUrl"),
		"DocsHomeUrl":             docsOptionValue("DocsHomeUrl"),
		"DocsGithubUrl":           docsOptionValue("DocsGithubUrl"),
		"DocsMetaKeywords":        docsOptionValue("DocsMetaKeywords"),
		"DocsBusinessPhone":       docsOptionValue("DocsBusinessPhone"),
		"DocsBusinessPhoneHref":   docsOptionValue("DocsBusinessPhoneHref"),
		"DocsBusinessWorkTimeZh":  docsOptionValue("DocsBusinessWorkTimeZh"),
		"DocsBusinessWorkTimeEn":  docsOptionValue("DocsBusinessWorkTimeEn"),
		"DocsBusinessWorkTimeJa":  docsOptionValue("DocsBusinessWorkTimeJa"),
		"DocsBusinessWechatQrUrl": docsOptionValue("DocsBusinessWechatQrUrl"),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": docsConfigResponse{
			BrandName: docsOptionValue("DocsBrandName"),
			SiteName: map[string]string{
				"en": docsOptionValue("DocsSiteNameEn"),
				"zh": docsOptionValue("DocsSiteNameZh"),
				"ja": docsOptionValue("DocsSiteNameJa"),
			},
			LogoUrl:      docsOptionValue("DocsLogoUrl"),
			HomeUrl:      docsOptionValue("DocsHomeUrl"),
			GithubUrl:    docsOptionValue("DocsGithubUrl"),
			MetaKeywords: splitDocsKeywords(docsOptionValue("DocsMetaKeywords")),
			Business: docsBusinessConfigResponse{
				Phone:     docsOptionValue("DocsBusinessPhone"),
				PhoneHref: docsOptionValue("DocsBusinessPhoneHref"),
				WorkTime: map[string]string{
					"en": docsOptionValue("DocsBusinessWorkTimeEn"),
					"zh": docsOptionValue("DocsBusinessWorkTimeZh"),
					"ja": docsOptionValue("DocsBusinessWorkTimeJa"),
				},
				WechatQrUrl: docsOptionValue("DocsBusinessWechatQrUrl"),
			},
			Raw: raw,
		},
	})
}
