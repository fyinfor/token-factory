package model

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/performance_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	var err error
	err = DB.Find(&options).Error
	return options, err
}

func InitOptionMap() {
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)

	// 添加原有的系统配置
	common.OptionMap["FileUploadPermission"] = strconv.Itoa(common.FileUploadPermission)
	common.OptionMap["FileDownloadPermission"] = strconv.Itoa(common.FileDownloadPermission)
	common.OptionMap["ImageUploadPermission"] = strconv.Itoa(common.ImageUploadPermission)
	common.OptionMap["ImageDownloadPermission"] = strconv.Itoa(common.ImageDownloadPermission)
	common.OptionMap["PasswordLoginEnabled"] = strconv.FormatBool(common.PasswordLoginEnabled)
	common.OptionMap["PasswordRegisterEnabled"] = strconv.FormatBool(common.PasswordRegisterEnabled)
	common.OptionMap["EmailVerificationEnabled"] = strconv.FormatBool(common.EmailVerificationEnabled)
	common.OptionMap["GitHubOAuthEnabled"] = strconv.FormatBool(common.GitHubOAuthEnabled)
	common.OptionMap["LinuxDOOAuthEnabled"] = strconv.FormatBool(common.LinuxDOOAuthEnabled)
	common.OptionMap["TelegramOAuthEnabled"] = strconv.FormatBool(common.TelegramOAuthEnabled)
	common.OptionMap["WeChatAuthEnabled"] = strconv.FormatBool(common.WeChatAuthEnabled)
	common.OptionMap["TurnstileCheckEnabled"] = strconv.FormatBool(common.TurnstileCheckEnabled)
	common.OptionMap["RegisterEnabled"] = strconv.FormatBool(common.RegisterEnabled)
	common.OptionMap["AutomaticDisableChannelEnabled"] = strconv.FormatBool(common.AutomaticDisableChannelEnabled)
	common.OptionMap["AutomaticEnableChannelEnabled"] = strconv.FormatBool(common.AutomaticEnableChannelEnabled)
	common.OptionMap["LogConsumeEnabled"] = strconv.FormatBool(common.LogConsumeEnabled)
	common.OptionMap["DisplayInCurrencyEnabled"] = strconv.FormatBool(common.DisplayInCurrencyEnabled)
	common.OptionMap["DisplayTokenStatEnabled"] = strconv.FormatBool(common.DisplayTokenStatEnabled)
	common.OptionMap["DrawingEnabled"] = strconv.FormatBool(common.DrawingEnabled)
	common.OptionMap["TaskEnabled"] = strconv.FormatBool(common.TaskEnabled)
	common.OptionMap["DataExportEnabled"] = strconv.FormatBool(common.DataExportEnabled)
	common.OptionMap["ChannelDisableThreshold"] = strconv.FormatFloat(common.ChannelDisableThreshold, 'f', -1, 64)
	common.OptionMap["EmailDomainRestrictionEnabled"] = strconv.FormatBool(common.EmailDomainRestrictionEnabled)
	common.OptionMap["EmailAliasRestrictionEnabled"] = strconv.FormatBool(common.EmailAliasRestrictionEnabled)
	common.OptionMap["EmailDomainWhitelist"] = strings.Join(common.EmailDomainWhitelist, ",")
	common.OptionMap["SMTPServer"] = ""
	common.OptionMap["SMTPFrom"] = ""
	common.OptionMap["SMTPPort"] = strconv.Itoa(common.SMTPPort)
	common.OptionMap["SMTPAccount"] = ""
	common.OptionMap["SMTPToken"] = ""
	common.OptionMap["SMTPSSLEnabled"] = strconv.FormatBool(common.SMTPSSLEnabled)
	common.OptionMap["Notice"] = ""
	common.OptionMap["About"] = ""
	common.OptionMap["HomePageContent"] = ""
	// 首页轮播广告 JSON 数组，见 web SettingsHomeBanner / HomeBannerCarousel
	common.OptionMap["HomeBannerSlides"] = "[]"
	common.OptionMap["Footer"] = common.Footer
	common.OptionMap["SystemName"] = common.SystemName
	common.OptionMap["Logo"] = common.Logo
	common.OptionMap["ServerAddress"] = ""
	common.OptionMap["WorkerUrl"] = system_setting.WorkerUrl
	common.OptionMap["WorkerValidKey"] = system_setting.WorkerValidKey
	common.OptionMap["WorkerAllowHttpImageRequestEnabled"] = strconv.FormatBool(system_setting.WorkerAllowHttpImageRequestEnabled)
	common.OptionMap["PayAddress"] = ""
	common.OptionMap["CustomCallbackAddress"] = ""
	common.OptionMap["EpayId"] = ""
	common.OptionMap["EpayKey"] = ""
	common.OptionMap["YipayAppSecret"] = operation_setting.YipayAppSecret
	common.OptionMap["OnlinePayProvider"] = operation_setting.OnlinePayProvider
	common.OptionMap["YipayMchNo"] = operation_setting.YipayMchNo
	common.OptionMap["YipayAppId"] = operation_setting.YipayAppId
	common.OptionMap["YipayWayCode"] = operation_setting.YipayWayCode
	common.OptionMap["YipayNotifyUrl"] = operation_setting.YipayNotifyUrl
	common.OptionMap["YipayReturnUrl"] = operation_setting.YipayReturnUrl
	common.OptionMap["YipayRequestURL"] = operation_setting.YipayRequestURL
	common.OptionMap["YipayChannelExtra"] = operation_setting.YipayChannelExtra
	common.OptionMap["Price"] = strconv.FormatFloat(operation_setting.Price, 'f', -1, 64)
	common.OptionMap["USDExchangeRate"] = strconv.FormatFloat(operation_setting.USDExchangeRate, 'f', -1, 64)
	common.OptionMap["MinTopUp"] = strconv.Itoa(operation_setting.MinTopUp)
	common.OptionMap["StripeMinTopUp"] = strconv.Itoa(setting.StripeMinTopUp)
	common.OptionMap["StripeApiSecret"] = setting.StripeApiSecret
	common.OptionMap["StripeWebhookSecret"] = setting.StripeWebhookSecret
	common.OptionMap["StripePriceId"] = setting.StripePriceId
	common.OptionMap["StripeUnitPrice"] = strconv.FormatFloat(setting.StripeUnitPrice, 'f', -1, 64)
	common.OptionMap["StripePromotionCodesEnabled"] = strconv.FormatBool(setting.StripePromotionCodesEnabled)
	common.OptionMap["CreemApiKey"] = setting.CreemApiKey
	common.OptionMap["CreemProducts"] = setting.CreemProducts
	common.OptionMap["CreemTestMode"] = strconv.FormatBool(setting.CreemTestMode)
	common.OptionMap["CreemWebhookSecret"] = setting.CreemWebhookSecret
	common.OptionMap["WaffoEnabled"] = strconv.FormatBool(setting.WaffoEnabled)
	common.OptionMap["WaffoApiKey"] = setting.WaffoApiKey
	common.OptionMap["WaffoPrivateKey"] = setting.WaffoPrivateKey
	common.OptionMap["WaffoPublicCert"] = setting.WaffoPublicCert
	common.OptionMap["WaffoSandboxPublicCert"] = setting.WaffoSandboxPublicCert
	common.OptionMap["WaffoSandboxApiKey"] = setting.WaffoSandboxApiKey
	common.OptionMap["WaffoSandboxPrivateKey"] = setting.WaffoSandboxPrivateKey
	common.OptionMap["WaffoSandbox"] = strconv.FormatBool(setting.WaffoSandbox)
	common.OptionMap["WaffoMerchantId"] = setting.WaffoMerchantId
	common.OptionMap["WaffoNotifyUrl"] = setting.WaffoNotifyUrl
	common.OptionMap["WaffoReturnUrl"] = setting.WaffoReturnUrl
	common.OptionMap["WaffoSubscriptionReturnUrl"] = setting.WaffoSubscriptionReturnUrl
	common.OptionMap["WaffoCurrency"] = setting.WaffoCurrency
	common.OptionMap["WaffoUnitPrice"] = strconv.FormatFloat(setting.WaffoUnitPrice, 'f', -1, 64)
	common.OptionMap["WaffoMinTopUp"] = strconv.Itoa(setting.WaffoMinTopUp)
	common.OptionMap["WaffoPayMethods"] = setting.WaffoPayMethods2JsonString()
	common.OptionMap["TopupGroupRatio"] = common.TopupGroupRatio2JSONString()
	common.OptionMap["Chats"] = setting.Chats2JsonString()
	common.OptionMap["AutoGroups"] = setting.AutoGroups2JsonString()
	common.OptionMap["DefaultUseAutoGroup"] = strconv.FormatBool(setting.DefaultUseAutoGroup)
	common.OptionMap["PayMethods"] = operation_setting.PayMethods2JsonString()
	common.OptionMap["GitHubClientId"] = ""
	common.OptionMap["GitHubClientSecret"] = ""
	common.OptionMap["TelegramBotToken"] = ""
	common.OptionMap["TelegramBotName"] = ""
	common.OptionMap["WeChatServerAddress"] = ""
	common.OptionMap["WeChatServerToken"] = ""
	common.OptionMap["WeChatAccountQRCodeImageURL"] = ""
	common.OptionMap["TurnstileSiteKey"] = ""
	common.OptionMap["TurnstileSecretKey"] = ""
	common.OptionMap["SMSVerificationEnabled"] = strconv.FormatBool(common.SMSVerificationEnabled)
	common.OptionMap["SMSAccessKeyID"] = common.SMSAccessKeyID
	common.OptionMap["SMSAccessKeySecret"] = common.SMSAccessKeySecret
	common.OptionMap["SMSCodeSignName"] = common.SMSCodeSignName
	common.OptionMap["SMSCodeTemplateCode"] = common.SMSCodeTemplateCode
	common.OptionMap["SMSCodeValidMinutes"] = strconv.Itoa(common.SMSCodeValidMinutes)
	common.OptionMap["SMSCodeCooldownMinutes"] = strconv.Itoa(common.SMSCodeCooldownMinutes)
	common.OptionMap["SMSCodeDailyLimit"] = strconv.Itoa(common.SMSCodeDailyLimit)
	common.OptionMap["SMSPhoneBlacklist"] = strings.Join(common.SMSPhoneBlacklist, ",")
	common.OptionMap["QuotaForNewUser"] = strconv.Itoa(common.QuotaForNewUser)
	common.OptionMap["QuotaForInviter"] = strconv.Itoa(common.QuotaForInviter)
	common.OptionMap["QuotaForInvitee"] = strconv.Itoa(common.QuotaForInvitee)
	common.OptionMap["StudentApprovalRewardQuota"] = strconv.Itoa(common.StudentApprovalRewardQuota)
	common.OptionMap["AffiliateDefaultCommissionBps"] = strconv.Itoa(common.AffiliateDefaultCommissionBps)
	common.OptionMap["DistributorApplyCsImageUrl"] = ""
	common.OptionMap["DistributorWithdrawCsImageUrl"] = ""
	common.OptionMap["DistributorWithdrawNotice"] = ""
	// 分销商申请页标题下方展示的富文本（HTML，内容由运营设置编辑）
	common.OptionMap["DistributorApplyIntroHtml"] = ""
	common.OptionMap["DistributorMinWithdrawQuota"] = ""
	common.OptionMap["QuotaRemindThreshold"] = strconv.Itoa(common.QuotaRemindThreshold)
	common.OptionMap["PreConsumedQuota"] = strconv.Itoa(common.PreConsumedQuota)
	common.OptionMap["ModelRequestRateLimitCount"] = strconv.Itoa(setting.ModelRequestRateLimitCount)
	common.OptionMap["ModelRequestRateLimitDurationMinutes"] = strconv.Itoa(setting.ModelRequestRateLimitDurationMinutes)
	common.OptionMap["ModelRequestRateLimitSuccessCount"] = strconv.Itoa(setting.ModelRequestRateLimitSuccessCount)
	common.OptionMap["ModelRequestRateLimitGroup"] = setting.ModelRequestRateLimitGroup2JSONString()
	common.OptionMap["ModelRatio"] = ratio_setting.ModelRatio2JSONString()
	common.OptionMap["ModelPrice"] = ratio_setting.ModelPrice2JSONString()
	common.OptionMap["CacheRatio"] = ratio_setting.CacheRatio2JSONString()
	common.OptionMap["CreateCacheRatio"] = ratio_setting.CreateCacheRatio2JSONString()
	common.OptionMap["GroupRatio"] = ratio_setting.GroupRatio2JSONString()
	common.OptionMap["GroupGroupRatio"] = ratio_setting.GroupGroupRatio2JSONString()
	common.OptionMap["GroupModelPrice"] = ratio_setting.GroupModelPrice2JSONString()
	common.OptionMap["GroupModelRatio"] = ratio_setting.GroupModelRatio2JSONString()
	common.OptionMap["ChannelModelPrice"] = ratio_setting.ChannelModelPrice2JSONString()
	common.OptionMap["ChannelModelRatio"] = ratio_setting.ChannelModelRatio2JSONString()
	common.OptionMap["ChannelCompletionRatio"] = ratio_setting.ChannelCompletionRatio2JSONString()
	common.OptionMap["ChannelCacheRatio"] = ratio_setting.ChannelCacheRatio2JSONString()
	common.OptionMap["ChannelCreateCacheRatio"] = ratio_setting.ChannelCreateCacheRatio2JSONString()
	common.OptionMap["ChannelImageRatio"] = ratio_setting.ChannelImageRatio2JSONString()
	common.OptionMap["ChannelAudioRatio"] = ratio_setting.ChannelAudioRatio2JSONString()
	common.OptionMap["ChannelAudioCompletionRatio"] = ratio_setting.ChannelAudioCompletionRatio2JSONString()
	common.OptionMap["ChannelVideoRatio"] = ratio_setting.ChannelVideoRatio2JSONString()
	common.OptionMap["ChannelVideoCompletionRatio"] = ratio_setting.ChannelVideoCompletionRatio2JSONString()
	common.OptionMap["ChannelVideoPrice"] = ratio_setting.ChannelVideoPrice2JSONString()
	common.OptionMap["ChannelVideoPricingRules"] = ratio_setting.ChannelVideoPricingRules2JSONString()
	common.OptionMap["RequestTierPricing"] = ratio_setting.RequestTierPricing2JSONString()
	common.OptionMap["ChannelRequestTierPricing"] = ratio_setting.ChannelRequestTierPricing2JSONString()
	common.OptionMap["RequestTierPricingTemplates"] = ratio_setting.RequestTierPricingTemplates2JSONString()
	common.OptionMap["SupplierModelPrice"] = ratio_setting.SupplierModelPrice2JSONString()
	common.OptionMap["SupplierModelRatio"] = ratio_setting.SupplierModelRatio2JSONString()
	common.OptionMap["UserUsableGroups"] = setting.UserUsableGroups2JSONString()
	common.OptionMap["CompletionRatio"] = ratio_setting.CompletionRatio2JSONString()
	common.OptionMap["ImageRatio"] = ratio_setting.ImageRatio2JSONString()
	common.OptionMap["AudioRatio"] = ratio_setting.AudioRatio2JSONString()
	common.OptionMap["AudioCompletionRatio"] = ratio_setting.AudioCompletionRatio2JSONString()
	common.OptionMap["VideoRatio"] = ratio_setting.VideoRatio2JSONString()
	common.OptionMap["VideoCompletionRatio"] = ratio_setting.VideoCompletionRatio2JSONString()
	common.OptionMap["VideoPrice"] = ratio_setting.VideoPrice2JSONString()
	common.OptionMap["VideoPricingRules"] = ratio_setting.VideoPricingRules2JSONString()
	common.OptionMap["TopUpLink"] = common.TopUpLink
	//common.OptionMap["ChatLink"] = common.ChatLink
	//common.OptionMap["ChatLink2"] = common.ChatLink2
	common.OptionMap["QuotaPerUnit"] = strconv.FormatFloat(common.QuotaPerUnit, 'f', -1, 64)
	common.OptionMap["RetryTimes"] = strconv.Itoa(common.RetryTimes)
	common.OptionMap["DataExportInterval"] = strconv.Itoa(common.DataExportInterval)
	common.OptionMap["DataExportDefaultTime"] = common.DataExportDefaultTime
	common.OptionMap["DefaultCollapseSidebar"] = strconv.FormatBool(common.DefaultCollapseSidebar)
	common.OptionMap["MjNotifyEnabled"] = strconv.FormatBool(setting.MjNotifyEnabled)
	common.OptionMap["MjAccountFilterEnabled"] = strconv.FormatBool(setting.MjAccountFilterEnabled)
	common.OptionMap["MjModeClearEnabled"] = strconv.FormatBool(setting.MjModeClearEnabled)
	common.OptionMap["MjForwardUrlEnabled"] = strconv.FormatBool(setting.MjForwardUrlEnabled)
	common.OptionMap["MjActionCheckSuccessEnabled"] = strconv.FormatBool(setting.MjActionCheckSuccessEnabled)
	common.OptionMap["CheckSensitiveEnabled"] = strconv.FormatBool(setting.CheckSensitiveEnabled)
	common.OptionMap["DemoSiteEnabled"] = strconv.FormatBool(operation_setting.DemoSiteEnabled)
	common.OptionMap["SelfUseModeEnabled"] = strconv.FormatBool(operation_setting.SelfUseModeEnabled)
	common.OptionMap["ChannelBalanceAlertEnabled"] = strconv.FormatBool(false)
	common.OptionMap["ChannelBalanceSoftAlertThreshold"] = strconv.FormatFloat(50, 'f', -1, 64)
	common.OptionMap["ChannelBalanceRiskAlertThreshold"] = strconv.FormatFloat(20, 'f', -1, 64)
	common.OptionMap["ModelRequestRateLimitEnabled"] = strconv.FormatBool(setting.ModelRequestRateLimitEnabled)
	common.OptionMap["GlobalApiRateLimitEnable"] = strconv.FormatBool(common.GlobalApiRateLimitEnable)
	common.OptionMap["GlobalApiRateLimitNum"] = strconv.Itoa(common.GlobalApiRateLimitNum)
	common.OptionMap["GlobalApiRateLimitDuration"] = strconv.FormatInt(common.GlobalApiRateLimitDuration, 10)
	common.OptionMap["GlobalWebRateLimitEnable"] = strconv.FormatBool(common.GlobalWebRateLimitEnable)
	common.OptionMap["GlobalWebRateLimitNum"] = strconv.Itoa(common.GlobalWebRateLimitNum)
	common.OptionMap["GlobalWebRateLimitDuration"] = strconv.FormatInt(common.GlobalWebRateLimitDuration, 10)
	common.OptionMap["CriticalRateLimitEnable"] = strconv.FormatBool(common.CriticalRateLimitEnable)
	common.OptionMap["CriticalRateLimitNum"] = strconv.Itoa(common.CriticalRateLimitNum)
	common.OptionMap["CriticalRateLimitDuration"] = strconv.FormatInt(common.CriticalRateLimitDuration, 10)
	common.OptionMap["RateLimitUserWhitelist"] = setting.RateLimitUserWhitelist2JSONString()
	common.OptionMap["CheckSensitiveOnPromptEnabled"] = strconv.FormatBool(setting.CheckSensitiveOnPromptEnabled)
	common.OptionMap["StopOnSensitiveEnabled"] = strconv.FormatBool(setting.StopOnSensitiveEnabled)
	common.OptionMap["SensitiveWords"] = setting.SensitiveWordsToString()
	common.OptionMap["StreamCacheQueueLength"] = strconv.Itoa(setting.StreamCacheQueueLength)
	common.OptionMap["AutomaticDisableKeywords"] = operation_setting.AutomaticDisableKeywordsToString()
	common.OptionMap["AutomaticDisableStatusCodes"] = operation_setting.AutomaticDisableStatusCodesToString()
	common.OptionMap["AutomaticRetryStatusCodes"] = operation_setting.AutomaticRetryStatusCodesToString()
	common.OptionMap["ExposeRatioEnabled"] = strconv.FormatBool(ratio_setting.IsExposeRatioEnabled())

	// 自动添加所有注册的模型配置
	modelConfigs := config.GlobalConfig.ExportAllConfigs()
	for k, v := range modelConfigs {
		common.OptionMap[k] = v
	}

	common.OptionMapRWMutex.Unlock()
	loadOptionsFromDatabase()
}

func loadOptionsFromDatabase() {
	options, _ := AllOption()
	for _, option := range options {
		err := updateOptionMap(option.Key, option.Value)
		if err != nil {
			common.SysLog("failed to update option map [" + option.Key + "]: " + err.Error())
		}
	}
}

func SyncOptions(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing options from database")
		loadOptionsFromDatabase()
	}
}

func UpdateOption(key string, value string) error {
	// Save to database first
	option := Option{
		Key: key,
	}
	// https://gorm.io/docs/update.html#Save-All-Fields
	DB.FirstOrCreate(&option, Option{Key: key})
	option.Value = value
	// Save is a combination function.
	// If save value does not contain primary key, it will execute Create,
	// otherwise it will execute Update (with all fields).
	DB.Save(&option)
	// Update OptionMap
	return updateOptionMap(key, value)
}

func updateOptionMap(key string, value string) (err error) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	common.OptionMap[key] = value

	// 检查是否是模型配置 - 使用更规范的方式处理
	if handleConfigUpdate(key, value) {
		return nil // 已由配置系统处理
	}

	// 处理传统配置项...
	if strings.HasSuffix(key, "Permission") {
		intValue, _ := strconv.Atoi(value)
		switch key {
		case "FileUploadPermission":
			common.FileUploadPermission = intValue
		case "FileDownloadPermission":
			common.FileDownloadPermission = intValue
		case "ImageUploadPermission":
			common.ImageUploadPermission = intValue
		case "ImageDownloadPermission":
			common.ImageDownloadPermission = intValue
		}
	}
	if strings.HasSuffix(key, "Enabled") || key == "DefaultCollapseSidebar" || key == "DefaultUseAutoGroup" {
		boolValue := value == "true"
		switch key {
		case "PasswordRegisterEnabled":
			common.PasswordRegisterEnabled = boolValue
		case "PasswordLoginEnabled":
			common.PasswordLoginEnabled = boolValue
		case "EmailVerificationEnabled":
			common.EmailVerificationEnabled = boolValue
		case "GitHubOAuthEnabled":
			common.GitHubOAuthEnabled = boolValue
		case "LinuxDOOAuthEnabled":
			common.LinuxDOOAuthEnabled = boolValue
		case "WeChatAuthEnabled":
			common.WeChatAuthEnabled = boolValue
		case "TelegramOAuthEnabled":
			common.TelegramOAuthEnabled = boolValue
		case "TurnstileCheckEnabled":
			common.TurnstileCheckEnabled = boolValue
		case "SMSVerificationEnabled":
			common.SMSVerificationEnabled = boolValue
		case "RegisterEnabled":
			common.RegisterEnabled = boolValue
		case "EmailDomainRestrictionEnabled":
			common.EmailDomainRestrictionEnabled = boolValue
		case "EmailAliasRestrictionEnabled":
			common.EmailAliasRestrictionEnabled = boolValue
		case "AutomaticDisableChannelEnabled":
			common.AutomaticDisableChannelEnabled = boolValue
		case "AutomaticEnableChannelEnabled":
			common.AutomaticEnableChannelEnabled = boolValue
		case "LogConsumeEnabled":
			common.LogConsumeEnabled = boolValue
		case "DisplayInCurrencyEnabled":
			// 兼容旧字段：同步到新配置 general_setting.quota_display_type（运行时生效）
			// true -> USD, false -> TOKENS
			newVal := "USD"
			if !boolValue {
				newVal = "TOKENS"
			}
			if cfg := config.GlobalConfig.Get("general_setting"); cfg != nil {
				_ = config.UpdateConfigFromMap(cfg, map[string]string{"quota_display_type": newVal})
			}
		case "DisplayTokenStatEnabled":
			common.DisplayTokenStatEnabled = boolValue
		case "DrawingEnabled":
			common.DrawingEnabled = boolValue
		case "TaskEnabled":
			common.TaskEnabled = boolValue
		case "DataExportEnabled":
			common.DataExportEnabled = boolValue
		case "DefaultCollapseSidebar":
			common.DefaultCollapseSidebar = boolValue
		case "MjNotifyEnabled":
			setting.MjNotifyEnabled = boolValue
		case "MjAccountFilterEnabled":
			setting.MjAccountFilterEnabled = boolValue
		case "MjModeClearEnabled":
			setting.MjModeClearEnabled = boolValue
		case "MjForwardUrlEnabled":
			setting.MjForwardUrlEnabled = boolValue
		case "MjActionCheckSuccessEnabled":
			setting.MjActionCheckSuccessEnabled = boolValue
		case "CheckSensitiveEnabled":
			setting.CheckSensitiveEnabled = boolValue
		case "DemoSiteEnabled":
			operation_setting.DemoSiteEnabled = boolValue
		case "SelfUseModeEnabled":
			operation_setting.SelfUseModeEnabled = boolValue
		case "CheckSensitiveOnPromptEnabled":
			setting.CheckSensitiveOnPromptEnabled = boolValue
		case "ModelRequestRateLimitEnabled":
			setting.ModelRequestRateLimitEnabled = boolValue
		case "StopOnSensitiveEnabled":
			setting.StopOnSensitiveEnabled = boolValue
		case "SMTPSSLEnabled":
			common.SMTPSSLEnabled = boolValue
		case "WorkerAllowHttpImageRequestEnabled":
			system_setting.WorkerAllowHttpImageRequestEnabled = boolValue
		case "DefaultUseAutoGroup":
			setting.DefaultUseAutoGroup = boolValue
		case "ExposeRatioEnabled":
			ratio_setting.SetExposeRatioEnabled(boolValue)
		}
	}
	switch key {
	case "EmailDomainWhitelist":
		common.EmailDomainWhitelist = strings.Split(value, ",")
	case "SMTPServer":
		common.SMTPServer = value
	case "SMTPPort":
		intValue, _ := strconv.Atoi(value)
		common.SMTPPort = intValue
	case "SMTPAccount":
		common.SMTPAccount = value
	case "SMTPFrom":
		common.SMTPFrom = value
	case "SMTPToken":
		common.SMTPToken = value
	case "ServerAddress":
		system_setting.ServerAddress = value
	case "WorkerUrl":
		system_setting.WorkerUrl = value
	case "WorkerValidKey":
		system_setting.WorkerValidKey = value
	case "PayAddress":
		operation_setting.PayAddress = value
	case "Chats":
		err = setting.UpdateChatsByJsonString(value)
	case "AutoGroups":
		err = setting.UpdateAutoGroupsByJsonString(value)
	case "CustomCallbackAddress":
		operation_setting.CustomCallbackAddress = value
	case "EpayId":
		operation_setting.EpayId = value
	case "EpayKey":
		operation_setting.EpayKey = value
	case "YipayAppSecret":
		operation_setting.YipayAppSecret = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayAppSecret
	case "OnlinePayProvider":
		operation_setting.OnlinePayProvider = value
	case "YipayMchNo":
		operation_setting.YipayMchNo = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayMchNo
	case "YipayAppId":
		operation_setting.YipayAppId = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayAppId
	case "YipayWayCode":
		operation_setting.YipayWayCode = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayWayCode
	case "YipayNotifyUrl":
		operation_setting.YipayNotifyUrl = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayNotifyUrl
	case "YipayReturnUrl":
		operation_setting.YipayReturnUrl = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayReturnUrl
	case "YipayRequestURL":
		operation_setting.YipayRequestURL = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayRequestURL
	case "YipayChannelExtra":
		operation_setting.YipayChannelExtra = strings.TrimSpace(value)
		common.OptionMap[key] = operation_setting.YipayChannelExtra
	case "Price":
		operation_setting.Price, _ = strconv.ParseFloat(value, 64)
	case "USDExchangeRate":
		operation_setting.USDExchangeRate, _ = strconv.ParseFloat(value, 64)
	case "MinTopUp":
		operation_setting.MinTopUp, _ = strconv.Atoi(value)
	case "StripeApiSecret":
		setting.StripeApiSecret = value
	case "StripeWebhookSecret":
		setting.StripeWebhookSecret = value
	case "StripePriceId":
		setting.StripePriceId = value
	case "StripeUnitPrice":
		setting.StripeUnitPrice, _ = strconv.ParseFloat(value, 64)
	case "StripeMinTopUp":
		setting.StripeMinTopUp, _ = strconv.Atoi(value)
	case "StripePromotionCodesEnabled":
		setting.StripePromotionCodesEnabled = value == "true"
	case "CreemApiKey":
		setting.CreemApiKey = value
	case "CreemProducts":
		setting.CreemProducts = value
	case "CreemTestMode":
		setting.CreemTestMode = value == "true"
	case "CreemWebhookSecret":
		setting.CreemWebhookSecret = value
	case "WaffoEnabled":
		setting.WaffoEnabled = value == "true"
	case "WaffoApiKey":
		setting.WaffoApiKey = value
	case "WaffoPrivateKey":
		setting.WaffoPrivateKey = value
	case "WaffoPublicCert":
		setting.WaffoPublicCert = value
	case "WaffoSandboxPublicCert":
		setting.WaffoSandboxPublicCert = value
	case "WaffoSandboxApiKey":
		setting.WaffoSandboxApiKey = value
	case "WaffoSandboxPrivateKey":
		setting.WaffoSandboxPrivateKey = value
	case "WaffoSandbox":
		setting.WaffoSandbox = value == "true"
	case "WaffoMerchantId":
		setting.WaffoMerchantId = value
	case "WaffoNotifyUrl":
		setting.WaffoNotifyUrl = value
	case "WaffoReturnUrl":
		setting.WaffoReturnUrl = value
	case "WaffoSubscriptionReturnUrl":
		setting.WaffoSubscriptionReturnUrl = value
	case "WaffoCurrency":
		setting.WaffoCurrency = value
	case "WaffoUnitPrice":
		setting.WaffoUnitPrice, _ = strconv.ParseFloat(value, 64)
	case "WaffoMinTopUp":
		setting.WaffoMinTopUp, _ = strconv.Atoi(value)
	case "TopupGroupRatio":
		err = common.UpdateTopupGroupRatioByJSONString(value)
	case "GitHubClientId":
		common.GitHubClientId = value
	case "GitHubClientSecret":
		common.GitHubClientSecret = value
	case "LinuxDOClientId":
		common.LinuxDOClientId = value
	case "LinuxDOClientSecret":
		common.LinuxDOClientSecret = value
	case "LinuxDOMinimumTrustLevel":
		common.LinuxDOMinimumTrustLevel, _ = strconv.Atoi(value)
	case "Footer":
		common.Footer = value
	case "SystemName":
		common.SystemName = value
	case "Logo":
		common.Logo = value
	case "WeChatServerAddress":
		common.WeChatServerAddress = value
	case "WeChatServerToken":
		common.WeChatServerToken = value
	case "WeChatAccountQRCodeImageURL":
		common.WeChatAccountQRCodeImageURL = value
	case "TelegramBotToken":
		common.TelegramBotToken = value
	case "TelegramBotName":
		common.TelegramBotName = value
	case "TurnstileSiteKey":
		common.TurnstileSiteKey = value
	case "TurnstileSecretKey":
		common.TurnstileSecretKey = value
	case "SMSAccessKeyID":
		common.SMSAccessKeyID = strings.TrimSpace(value)
	case "SMSAccessKeySecret":
		common.SMSAccessKeySecret = strings.TrimSpace(value)
	case "SMSCodeSignName":
		common.SMSCodeSignName = strings.TrimSpace(value)
	case "SMSCodeTemplateCode":
		common.SMSCodeTemplateCode = strings.TrimSpace(value)
	case "SMSCodeValidMinutes":
		if n, parseErr := strconv.Atoi(value); parseErr == nil && n > 0 {
			common.SMSCodeValidMinutes = n
		}
	case "SMSCodeCooldownMinutes":
		if n, parseErr := strconv.Atoi(value); parseErr == nil && n > 0 {
			common.SMSCodeCooldownMinutes = n
		}
	case "SMSCodeDailyLimit":
		if n, parseErr := strconv.Atoi(value); parseErr == nil && n > 0 {
			common.SMSCodeDailyLimit = n
		}
	case "SMSPhoneBlacklist":
		if strings.TrimSpace(value) == "" {
			common.SMSPhoneBlacklist = []string{}
		} else {
			parts := strings.Split(value, ",")
			list := make([]string, 0, len(parts))
			for _, p := range parts {
				phone := strings.TrimSpace(p)
				if phone != "" {
					list = append(list, phone)
				}
			}
			common.SMSPhoneBlacklist = list
		}
	case "QuotaForNewUser":
		// 站内额度整数；管理端以美元填写，提交前已按 QuotaPerUnit 换算（与 common.QuotaFromUSD 一致）
		common.QuotaForNewUser, _ = strconv.Atoi(value)
	case "QuotaForInviter":
		// 站内额度整数；管理端以美元填写，提交前已按 QuotaPerUnit 换算（与 common.QuotaFromUSD 一致）
		common.QuotaForInviter, _ = strconv.Atoi(value)
	case "QuotaForInvitee":
		// 同上：被邀请人注册奖励写入 quota，存库为换算后的额度整数
		common.QuotaForInvitee, _ = strconv.Atoi(value)
	case "StudentApprovalRewardQuota":
		common.StudentApprovalRewardQuota, _ = strconv.Atoi(value)
	case "AffiliateDefaultCommissionBps":
		if n, err := strconv.Atoi(value); err == nil && n >= 0 && n <= 10000 {
			if n == 0 {
				// 历史或未配置为 0 时按系统默认 10% 计，避免分销奖励恒为 0
				common.AffiliateDefaultCommissionBps = 1000
			} else {
				common.AffiliateDefaultCommissionBps = n
			}
		}
	case "QuotaRemindThreshold":
		common.QuotaRemindThreshold, _ = strconv.Atoi(value)
	case "PreConsumedQuota":
		common.PreConsumedQuota, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitCount":
		setting.ModelRequestRateLimitCount, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitDurationMinutes":
		setting.ModelRequestRateLimitDurationMinutes, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitSuccessCount":
		setting.ModelRequestRateLimitSuccessCount, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitGroup":
		err = setting.UpdateModelRequestRateLimitGroupByJSONString(value)
	case "RetryTimes":
		common.RetryTimes, _ = strconv.Atoi(value)
	case "DataExportInterval":
		common.DataExportInterval, _ = strconv.Atoi(value)
	case "DataExportDefaultTime":
		common.DataExportDefaultTime = value
	case "ModelRatio":
		err = ratio_setting.UpdateModelRatioByJSONString(value)
	case "GroupRatio":
		err = ratio_setting.UpdateGroupRatioByJSONString(value)
	case "GroupGroupRatio":
		err = ratio_setting.UpdateGroupGroupRatioByJSONString(value)
	case "GroupModelPrice":
		err = ratio_setting.UpdateGroupModelPriceByJSONString(value)
	case "GroupModelRatio":
		err = ratio_setting.UpdateGroupModelRatioByJSONString(value)
	case "ChannelModelPrice":
		err = ratio_setting.UpdateChannelModelPriceByJSONString(value)
	case "ChannelModelRatio":
		err = ratio_setting.UpdateChannelModelRatioByJSONString(value)
	case "ChannelCompletionRatio":
		err = ratio_setting.UpdateChannelCompletionRatioByJSONString(value)
	case "ChannelCacheRatio":
		err = ratio_setting.UpdateChannelCacheRatioByJSONString(value)
	case "ChannelCreateCacheRatio":
		err = ratio_setting.UpdateChannelCreateCacheRatioByJSONString(value)
	case "ChannelImageRatio":
		err = ratio_setting.UpdateChannelImageRatioByJSONString(value)
	case "ChannelAudioRatio":
		err = ratio_setting.UpdateChannelAudioRatioByJSONString(value)
	case "ChannelAudioCompletionRatio":
		err = ratio_setting.UpdateChannelAudioCompletionRatioByJSONString(value)
	case "ChannelVideoRatio":
		err = ratio_setting.UpdateChannelVideoRatioByJSONString(value)
	case "ChannelVideoCompletionRatio":
		err = ratio_setting.UpdateChannelVideoCompletionRatioByJSONString(value)
	case "ChannelVideoPrice":
		err = ratio_setting.UpdateChannelVideoPriceByJSONString(value)
	case "ChannelVideoPricingRules":
		err = ratio_setting.UpdateChannelVideoPricingRulesByJSONString(value)
	case "RequestTierPricing":
		err = ratio_setting.UpdateRequestTierPricingByJSONString(value)
	case "ChannelRequestTierPricing":
		err = ratio_setting.UpdateChannelRequestTierPricingByJSONString(value)
	case "RequestTierPricingTemplates":
		err = ratio_setting.UpdateRequestTierPricingTemplatesByJSONString(value)
	case "SupplierModelPrice":
		err = ratio_setting.UpdateSupplierModelPriceByJSONString(value)
	case "SupplierModelRatio":
		err = ratio_setting.UpdateSupplierModelRatioByJSONString(value)
	case "UserUsableGroups":
		err = setting.UpdateUserUsableGroupsByJSONString(value)
	case "CompletionRatio":
		err = ratio_setting.UpdateCompletionRatioByJSONString(value)
	case "ModelPrice":
		err = ratio_setting.UpdateModelPriceByJSONString(value)
	case "CacheRatio":
		err = ratio_setting.UpdateCacheRatioByJSONString(value)
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(value)
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(value)
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(value)
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(value)
	case "VideoRatio":
		err = ratio_setting.UpdateVideoRatioByJSONString(value)
	case "VideoCompletionRatio":
		err = ratio_setting.UpdateVideoCompletionRatioByJSONString(value)
	case "VideoPrice":
		err = ratio_setting.UpdateVideoPriceByJSONString(value)
	case "VideoPricingRules":
		err = ratio_setting.UpdateVideoPricingRulesByJSONString(value)
	case "TopUpLink":
		common.TopUpLink = value
	//case "ChatLink":
	//	common.ChatLink = value
	//case "ChatLink2":
	//	common.ChatLink2 = value
	case "ChannelDisableThreshold":
		common.ChannelDisableThreshold, _ = strconv.ParseFloat(value, 64)
	case "QuotaPerUnit":
		common.QuotaPerUnit, _ = strconv.ParseFloat(value, 64)
	case "SensitiveWords":
		setting.SensitiveWordsFromString(value)
	case "AutomaticDisableKeywords":
		operation_setting.AutomaticDisableKeywordsFromString(value)
	case "AutomaticDisableStatusCodes":
		err = operation_setting.AutomaticDisableStatusCodesFromString(value)
	case "AutomaticRetryStatusCodes":
		err = operation_setting.AutomaticRetryStatusCodesFromString(value)
	case "StreamCacheQueueLength":
		setting.StreamCacheQueueLength, _ = strconv.Atoi(value)
	case "GlobalApiRateLimitNum":
		common.GlobalApiRateLimitNum, _ = strconv.Atoi(value)
	case "GlobalApiRateLimitDuration":
		common.GlobalApiRateLimitDuration, _ = strconv.ParseInt(value, 10, 64)
	case "GlobalApiRateLimitEnable":
		common.GlobalApiRateLimitEnable = value == "true"
	case "GlobalWebRateLimitNum":
		common.GlobalWebRateLimitNum, _ = strconv.Atoi(value)
	case "GlobalWebRateLimitDuration":
		common.GlobalWebRateLimitDuration, _ = strconv.ParseInt(value, 10, 64)
	case "GlobalWebRateLimitEnable":
		common.GlobalWebRateLimitEnable = value == "true"
	case "CriticalRateLimitNum":
		common.CriticalRateLimitNum, _ = strconv.Atoi(value)
	case "CriticalRateLimitDuration":
		common.CriticalRateLimitDuration, _ = strconv.ParseInt(value, 10, 64)
	case "CriticalRateLimitEnable":
		common.CriticalRateLimitEnable = value == "true"
	case "RateLimitUserWhitelist":
		err = setting.UpdateRateLimitUserWhitelistByJSONString(value)
	case "PayMethods":
		err = operation_setting.UpdatePayMethodsByJsonString(value)
	case "WaffoPayMethods":
		// WaffoPayMethods is read directly from OptionMap via setting.GetWaffoPayMethods().
		// The value is already stored in OptionMap at the top of this function (line: common.OptionMap[key] = value).
		// No additional in-memory variable to update.
	}
	return err
}

// handleConfigUpdate 处理分层配置更新，返回是否已处理
func handleConfigUpdate(key, value string) bool {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false // 不是分层配置
	}

	configName := parts[0]
	configKey := parts[1]

	// 获取配置对象
	cfg := config.GlobalConfig.Get(configName)
	if cfg == nil {
		return false // 未注册的配置
	}

	// 更新配置
	configMap := map[string]string{
		configKey: value,
	}
	config.UpdateConfigFromMap(cfg, configMap)

	// 特定配置的后处理
	if configName == "performance_setting" {
		// 同步磁盘缓存配置到 common 包
		performance_setting.UpdateAndSync()
	}

	return true // 已处理
}
