package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

var completionRatioMetaOptionKeys = []string{
	"ModelPrice",
	"ModelRatio",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

func collectModelNamesFromOptionValue(raw string, modelNames map[string]struct{}) {
	if strings.TrimSpace(raw) == "" {
		return
	}

	var parsed map[string]any
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return
	}

	for modelName := range parsed {
		modelNames[modelName] = struct{}{}
	}
}

func buildCompletionRatioMetaValue(optionValues map[string]string) string {
	modelNames := make(map[string]struct{})
	for _, key := range completionRatioMetaOptionKeys {
		collectModelNamesFromOptionValue(optionValues[key], modelNames)
	}

	meta := make(map[string]ratio_setting.CompletionRatioInfo, len(modelNames))
	for modelName := range modelNames {
		meta[modelName] = ratio_setting.GetCompletionRatioInfo(modelName)
	}

	jsonBytes, err := common.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func GetOptions(c *gin.Context) {
	// 已审核供应商仅返回其自有模型相关配置项，避免读取全局敏感配置。
	if c.GetInt("role") < common.RoleAdminUser {
		ownedModels, err := collectSupplierOwnedModelNames(c.GetInt("id"))
		if err != nil {
			common.ApiError(c, err)
			return
		}
		options := make([]*model.Option, 0, len(supplierEditableModelOptionKeys))
		common.OptionMapRWMutex.Lock()
		for key := range supplierEditableModelOptionKeys {
			value := strings.TrimSpace(common.Interface2String(common.OptionMap[key]))
			filteredValue, filterErr := filterModelJSONByOwnedModels(value, ownedModels)
			if filterErr != nil {
				continue
			}
			options = append(options, &model.Option{
				Key:   key,
				Value: filteredValue,
			})
		}
		common.OptionMapRWMutex.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data":    options,
		})
		return
	}

	var options []*model.Option
	optionValues := make(map[string]string)
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		// YipayAppSecret 在循环结束后单独追加，以 operation_setting 为准并避免与 OptionMap 不同步
		if k == "YipayAppSecret" {
			continue
		}
		// OSS AccessKey 与 Secret 在循环结束后单独追加（脱敏）
		if k == "oss_setting.access_key_id" || k == "oss_setting.access_key_secret" {
			continue
		}
		value := common.Interface2String(v)
		if strings.HasSuffix(k, "Token") ||
			strings.HasSuffix(k, "Secret") ||
			strings.HasSuffix(k, "Key") ||
			strings.HasSuffix(k, "secret") ||
			strings.HasSuffix(k, "api_key") {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: value,
		})
		for _, optionKey := range completionRatioMetaOptionKeys {
			if optionKey == k {
				optionValues[k] = value
				break
			}
		}
	}
	rawYipay := strings.TrimSpace(operation_setting.YipayAppSecret)
	if rawYipay == "" {
		if v, ok := common.OptionMap["YipayAppSecret"]; ok {
			rawYipay = strings.TrimSpace(common.Interface2String(v))
		}
	}
	yipayDisp := ""
	if rawYipay != "" {
		yipayDisp = common.MaskCredentialForAdminDisplay(rawYipay)
	}
	options = append(options, &model.Option{
		Key:   "YipayAppSecret",
		Value: yipayDisp,
	})
	rawOssID := strings.TrimSpace(operation_setting.GetOssSetting().AccessKeyID)
	if rawOssID == "" {
		if v, ok := common.OptionMap["oss_setting.access_key_id"]; ok {
			rawOssID = strings.TrimSpace(common.Interface2String(v))
		}
	}
	ossIDDisp := ""
	if rawOssID != "" {
		ossIDDisp = common.MaskCredentialForAdminDisplay(rawOssID)
	}
	options = append(options, &model.Option{
		Key:   "oss_setting.access_key_id",
		Value: ossIDDisp,
	})
	rawOssSecret := strings.TrimSpace(operation_setting.GetOssSetting().AccessKeySecret)
	if rawOssSecret == "" {
		if v, ok := common.OptionMap["oss_setting.access_key_secret"]; ok {
			rawOssSecret = strings.TrimSpace(common.Interface2String(v))
		}
	}
	ossSecretDisp := ""
	if rawOssSecret != "" {
		ossSecretDisp = common.MaskCredentialForAdminDisplay(rawOssSecret)
	}
	options = append(options, &model.Option{
		Key:   "oss_setting.access_key_secret",
		Value: ossSecretDisp,
	})
	common.OptionMapRWMutex.Unlock()
	options = append(options, &model.Option{
		Key:   "CompletionRatioMeta",
		Value: buildCompletionRatioMetaValue(optionValues),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	switch option.Value.(type) {
	case bool:
		option.Value = common.Interface2String(option.Value.(bool))
	case float64:
		option.Value = common.Interface2String(option.Value.(float64))
	case int:
		option.Value = common.Interface2String(option.Value.(int))
	default:
		option.Value = fmt.Sprintf("%v", option.Value)
	}
	valStr := strings.TrimSpace(option.Value.(string))
	// 已审核供应商仅可更新自己模型范围内的倍率相关配置，不可修改其他全局设置。
	if c.GetInt("role") < common.RoleAdminUser {
		if _, ok := supplierEditableModelOptionKeys[option.Key]; !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "供应商仅可修改模型倍率相关配置",
			})
			return
		}
		ownedModels, err := collectSupplierOwnedModelNames(c.GetInt("id"))
		if err != nil {
			common.ApiError(c, err)
			return
		}
		common.OptionMapRWMutex.Lock()
		currentValue := strings.TrimSpace(common.Interface2String(common.OptionMap[option.Key]))
		common.OptionMapRWMutex.Unlock()
		mergedValue, err := mergeModelJSONByOwnedModels(currentValue, valStr, ownedModels)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "配置格式错误，仅支持 JSON 对象",
			})
			return
		}
		if err := model.UpdateOption(option.Key, mergedValue); err != nil {
			common.ApiError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
		})
		return
	}

	if option.Key == "YipayAppSecret" && strings.TrimSpace(operation_setting.YipayAppSecret) != "" {
		if valStr == common.MaskCredentialForAdminDisplay(operation_setting.YipayAppSecret) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "",
			})
			return
		}
	}
	if option.Key == "oss_setting.access_key_id" && strings.TrimSpace(operation_setting.GetOssSetting().AccessKeyID) != "" {
		if valStr == common.MaskCredentialForAdminDisplay(operation_setting.GetOssSetting().AccessKeyID) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "",
			})
			return
		}
	}
	if option.Key == "oss_setting.access_key_secret" && strings.TrimSpace(operation_setting.GetOssSetting().AccessKeySecret) != "" {
		if valStr == common.MaskCredentialForAdminDisplay(operation_setting.GetOssSetting().AccessKeySecret) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "",
			})
			return
		}
	}
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 GitHub OAuth，请先填入 GitHub Client Id 以及 GitHub Client Secret！",
			})
			return
		}
	case "discord.enabled":
		if option.Value == "true" && system_setting.GetDiscordSettings().ClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Discord OAuth，请先填入 Discord Client Id 以及 Discord Client Secret！",
			})
			return
		}
	case "oidc.enabled":
		if option.Value == "true" && system_setting.GetOIDCSettings().ClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 OIDC 登录，请先填入 OIDC Client Id 以及 OIDC Client Secret！",
			})
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 LinuxDO OAuth，请先填入 LinuxDO Client Id 以及 LinuxDO Client Secret！",
			})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用邮箱域名限制，请先填入限制的邮箱域名！",
			})
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.WeChatServerAddress == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用微信登录，请先填入微信登录相关配置信息！",
			})
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.TurnstileSiteKey == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！",
			})

			return
		}
	case "TelegramOAuthEnabled":
		if option.Value == "true" && common.TelegramBotToken == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Telegram OAuth，请先填入 Telegram Bot Token！",
			})
			return
		}
	case "GroupRatio":
		err = ratio_setting.CheckGroupRatio(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "图片倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频补全倍率设置失败: " + err.Error(),
			})
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "缓存创建倍率设置失败: " + err.Error(),
			})
			return
		}
	case "ModelRequestRateLimitGroup":
		err = setting.CheckModelRequestRateLimitGroup(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticDisableStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticRetryStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.api_info":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "ApiInfo")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	err = model.UpdateOption(option.Key, option.Value.(string))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
