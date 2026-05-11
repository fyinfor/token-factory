package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/QuantumNous/new-api/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.RouteTag("api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		docsRoute := apiRouter.Group("/docs")
		docsRoute.Use(middleware.CORS())
		{
			docsRoute.GET("/config", controller.GetDocsConfig)
			docsRoute.OPTIONS("/config", func(c *gin.Context) {
				c.Status(204)
			})
		}
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.TryUserAuth(), controller.GetPricing)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/sms_verification", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendSMSVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.GET("/reset_password_email_code", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmailCode)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), controller.ResetPassword)
		apiRouter.POST("/user/reset_by_email_code", middleware.CriticalRateLimit(), controller.ResetPasswordByEmailCode)
		apiRouter.GET("/reset_password_sms", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetSMS)
		apiRouter.POST("/user/reset_by_phone", middleware.CriticalRateLimit(), controller.ResetPasswordByPhone)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind", middleware.CriticalRateLimit(), controller.EmailBind)
		// Non-standard OAuth (WeChat, Telegram) - keep original routes
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), controller.WeChatAuth)
		apiRouter.POST("/oauth/wechat/bind", middleware.CriticalRateLimit(), controller.WeChatBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), controller.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.CriticalRateLimit(), controller.TelegramBind)
		// Standard OAuth providers (GitHub, Discord, OIDC, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), controller.GetRatioConfig)
		apiRouter.POST("/aff/track", middleware.CriticalRateLimit(), controller.PostAffiliateTrack)

		// 分销商：申请、中心（需登录）
		distributorRoute := apiRouter.Group("/distributor")
		distributorRoute.Use(middleware.UserAuth())
		{
			distributorRoute.GET("/my_application", controller.GetMyDistributorApplication)
			distributorRoute.POST("/application", controller.PostDistributorApplication)
			distributorRoute.GET("/center", controller.GetDistributorCenterInfo)
			distributorRoute.GET("/analytics", controller.GetDistributorAnalytics)
			distributorRoute.GET("/invitee/:invitee_id/commissions", controller.GetDistributorInviteeCommissionLogs)
			distributorRoute.POST("/withdrawal", controller.PostDistributorWithdrawal)
			distributorRoute.GET("/withdrawals", controller.GetDistributorWithdrawals)
			distributorRoute.POST("/withdrawals/:id/cancel", controller.PostDistributorWithdrawalCancel)
		}
		distributorAdminRoute := apiRouter.Group("/distributor/admin")
		distributorAdminRoute.Use(middleware.AdminAuth())
		{
			distributorAdminRoute.GET("/applications", controller.ListDistributorApplicationsAdmin)
			distributorAdminRoute.GET("/applications/:id", controller.GetDistributorApplicationAdmin)
			distributorAdminRoute.POST("/applications/:id/approve", controller.ApproveDistributorApplicationAdmin)
			distributorAdminRoute.POST("/applications/:id/reject", controller.RejectDistributorApplicationAdmin)
			distributorAdminRoute.GET("/distributors", controller.ListDistributorsAdmin)
			distributorAdminRoute.GET("/distributors/:id/application", controller.GetDistributorApplicationByUserAdmin)
			distributorAdminRoute.PUT("/distributors/:id/application", controller.PutDistributorApplicationByUserAdmin)
			distributorAdminRoute.PUT("/distributors/:id/commission", controller.PutDistributorCommissionAdmin)
			distributorAdminRoute.GET("/distributors/:id/invitees", controller.GetDistributorInviteesAdmin)
			distributorAdminRoute.POST("/distributors/:id/settle", controller.PostDistributorSettleAdmin)
			distributorAdminRoute.GET("/withdrawals", controller.ListDistributorWithdrawalsAdmin)
			distributorAdminRoute.POST("/withdrawals/:id/approve", controller.ApproveDistributorWithdrawalAdmin)
			distributorAdminRoute.POST("/withdrawals/:id/reject", controller.RejectDistributorWithdrawalAdmin)
			distributorAdminRoute.GET("/analytics", controller.GetDistributorAdminAnalytics)
		}

		apiRouter.POST("/stripe/webhook", controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", controller.WaffoWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		// 阿里云 OSS 通用上传（需在运营设置中启用 OSS）
		apiRouter.POST("/oss/upload", middleware.UserAuth(), middleware.UploadRateLimit(), controller.OssUpload)

		playgroundRoute := apiRouter.Group("/playground")
		playgroundRoute.Use(middleware.UserAuth(), middleware.Distribute())
		{
			playgroundRoute.POST("/chat/completions", controller.Playground)
			playgroundRoute.POST("/images/generations", controller.PlaygroundImage)
			playgroundRoute.GET("/images/generations/:task_id", controller.PlaygroundImageFetch)
			playgroundRoute.POST("/videos", controller.PlaygroundVideo)
			playgroundRoute.GET("/videos/:task_id", controller.PlaygroundVideoFetch)
		}

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self/phone_available", controller.UserSelfCheckPhoneAvailable)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.POST("/student/apply", controller.ApplyStudent)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.POST("/self/admin_initial_setup", controller.CompleteAdminInitialSetup)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPay)
				selfRoute.POST("/aff_transfer", controller.TransferAffQuota)
				selfRoute.GET("/aff_invitees", controller.GetAffInvitees)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)
				selfRoute.POST("/supplier/application", controller.SubmitSupplierApplication)
				selfRoute.GET("/supplier/application/self", controller.GetMySupplierApplication)
				selfRoute.PUT("/supplier/application/self", controller.UpdateMySupplierApplication)
				selfRoute.GET("/supplier/application/:id/capability", controller.GetSupplierCapability)
				selfRoute.PUT("/supplier/application/:id/capability", controller.UpsertSupplierCapability)
				selfRoute.POST("/supplier/application/deactivate", controller.DeactivateMySupplierApplication)
				selfRoute.POST("/supplier/channels", controller.CreateMySupplierChannel)
				selfRoute.GET("/supplier/channels", controller.ListMySupplierChannels)
				selfRoute.POST("/supplier/models", controller.CreateMySupplierModel)
				selfRoute.GET("/supplier/models", controller.ListMySupplierModels)
				selfRoute.GET("/supplier-dashboard", controller.GetSupplierDashboardData)
				selfRoute.GET("/supplier/pricing/global", controller.GetSupplierGlobalPricing)
				selfRoute.PUT("/supplier/pricing/global", controller.PutSupplierGlobalPricing)
				selfRoute.GET("/supplier/pricing/channel/:channel_id", controller.GetSupplierChannelPricing)
				selfRoute.PUT("/supplier/pricing/channel/:channel_id", controller.PutSupplierChannelPricing)
				selfRoute.GET("/messages/self", controller.ListMyMessages)
				selfRoute.POST("/messages/:id/read", controller.MarkMyMessageRead)
				selfRoute.POST("/messages/read_all", controller.MarkAllMyMessagesRead)
				selfRoute.GET("/messages/unread_count", controller.GetMyUnreadMessageCount)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", controller.Setup2FA)
				selfRoute.POST("/2fa/enable", controller.Enable2FA)
				selfRoute.POST("/2fa/disable", controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

				// Custom OAuth bindings
				selfRoute.GET("/oauth/bindings", controller.GetUserOAuthBindings)
				selfRoute.DELETE("/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.GET("/", controller.GetAllUsers)
				adminRoute.GET("/topup", controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
				adminRoute.GET("/search", controller.SearchUsers)
				adminRoute.GET("/supplier/application", controller.AdminListSupplierApplications)
				adminRoute.PUT("/supplier/application/:id", controller.AdminUpdateSupplierApplication)
				adminRoute.POST("/supplier/application/activate", controller.ActivateSupplierApplication)
				adminRoute.GET("/supplier/list", controller.AdminListSuppliers)
				adminRoute.GET("/supplier/:id", controller.AdminGetSupplierDetail)
				adminRoute.POST("/supplier/application/:id/review", controller.AdminReviewSupplierApplication)
				adminRoute.POST("/messages/publish", controller.AdminPublishUserMessage)
				adminRoute.GET("/:id/oauth/bindings", controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", controller.AdminClearUserBinding)
				adminRoute.GET("/check_phone", controller.AdminCheckPhoneAvailable)
				adminRoute.GET("/:id", controller.GetUser)
				adminRoute.POST("/", controller.CreateUser)
				adminRoute.POST("/manage", controller.ManageUser)
				adminRoute.PUT("/aff_invitees/commission", controller.PutAffInviteeCommission)
				adminRoute.PUT("/", controller.UpdateUser)
				adminRoute.DELETE("/:id", controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", controller.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", controller.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", controller.AdminDisable2FA)
			}
		}

		// Subscription billing (plans, purchase, admin management)
		subscriptionRoute := apiRouter.Group("/subscription")
		subscriptionRoute.Use(middleware.UserAuth())
		{
			subscriptionRoute.GET("/plans", controller.GetSubscriptionPlans)
			subscriptionRoute.GET("/self", controller.GetSubscriptionSelf)
			subscriptionRoute.PUT("/self/preference", controller.UpdateSubscriptionPreference)
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestCreemPay)
		}
		subscriptionAdminRoute := apiRouter.Group("/subscription/admin")
		subscriptionAdminRoute.Use(middleware.AdminAuth())
		{
			subscriptionAdminRoute.GET("/plans", controller.AdminListSubscriptionPlans)
			subscriptionAdminRoute.POST("/plans", controller.AdminCreateSubscriptionPlan)
			subscriptionAdminRoute.PUT("/plans/:id", controller.AdminUpdateSubscriptionPlan)
			subscriptionAdminRoute.PATCH("/plans/:id", controller.AdminUpdateSubscriptionPlanStatus)
			subscriptionAdminRoute.POST("/bind", controller.AdminBindSubscription)

			// User subscription management (admin)
			subscriptionAdminRoute.GET("/users/:id/subscriptions", controller.AdminListUserSubscriptions)
			subscriptionAdminRoute.POST("/users/:id/subscriptions", controller.AdminCreateUserSubscription)
			subscriptionAdminRoute.POST("/user_subscriptions/:id/invalidate", controller.AdminInvalidateUserSubscription)
			subscriptionAdminRoute.DELETE("/user_subscriptions/:id", controller.AdminDeleteUserSubscription)
		}

		// Subscription payment callbacks (no auth)
		apiRouter.POST("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", controller.SubscriptionEpayReturn)
		optionRoute := apiRouter.Group("/option")
		{
			optionRoute.GET("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetOptions)
			optionRoute.PUT("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.UpdateOption)
			optionRoute.GET("/channel_affinity_cache", middleware.RootAuth(), controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", middleware.RootAuth(), controller.ClearChannelAffinityCache)
			optionRoute.GET("/rate_limit_blacklist_users", middleware.RootAuth(), controller.GetRateLimitBlacklistUsers)
			optionRoute.DELETE("/rate_limit_blacklist_users", middleware.RootAuth(), controller.DeleteRateLimitBlacklistUser)
			optionRoute.POST("/rest_model_ratio", middleware.RootAuth(), controller.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", middleware.RootAuth(), controller.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
		}

		// Custom OAuth provider management (root only)
		customOAuthRoute := apiRouter.Group("/custom-oauth-provider")
		customOAuthRoute.Use(middleware.RootAuth())
		{
			customOAuthRoute.POST("/discovery", controller.FetchCustomOAuthDiscovery)
			customOAuthRoute.GET("/", controller.GetCustomOAuthProviders)
			customOAuthRoute.GET("/:id", controller.GetCustomOAuthProvider)
			customOAuthRoute.POST("/", controller.CreateCustomOAuthProvider)
			customOAuthRoute.PUT("/:id", controller.UpdateCustomOAuthProvider)
			customOAuthRoute.DELETE("/:id", controller.DeleteCustomOAuthProvider)
		}
		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.RootAuth())
		{
			performanceRoute.GET("/stats", controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", controller.ForceGC)
			performanceRoute.GET("/logs", controller.GetLogFiles)
			performanceRoute.DELETE("/logs", controller.CleanupLogFiles)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		{
			ratioSyncRoute.GET("/channels", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetSyncableChannels)
			// 管理员或已审核供应商可拉取上游差异；供应商侧仅自有模型参与对比（见 controller.FetchUpstreamRatios）
			ratioSyncRoute.POST("/fetch", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.FetchUpstreamRatios)
		}
		tfOpenSyncRoute := apiRouter.Group("/tf_open_sync")
		{
			// 子站 TokenFactoryOpen 拉全站渠道（脱敏+定价）；鉴权见 controller.authorizeTFOpenSyncExport
			tfOpenSyncRoute.GET("/channels", middleware.CriticalRateLimit(), controller.TFOpenSyncExportChannels)
		}
		channelRoute := apiRouter.Group("/channel")
		{
			channelRoute.GET("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetAllChannels)
			channelRoute.GET("/search", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.SearchChannels)
			channelRoute.GET("/models", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.ChannelListModels)
			channelRoute.GET("/models_enabled", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.EnabledListModels)
			// 须注册在 /:id 之前，否则会被当成 id
			channelRoute.GET("/model-test-results", middleware.TryUserAuth(), controller.GetModelTestResultsForChannels)
			channelRoute.PUT("/model-test-result-display", middleware.UserAuth(), middleware.AdminAuth(), controller.PutModelTestResultDisplay)
			channelRoute.GET("/:id", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), controller.GetChannelKey)
			channelRoute.GET("/test", middleware.AdminAuth(), controller.TestAllChannels)
			channelRoute.GET("/test/:id", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.TestChannel)
			channelRoute.GET("/update_balance", middleware.AdminAuth(), controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", middleware.AdminAuth(), controller.UpdateChannelBalance)
			channelRoute.POST("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.AddChannel)
			channelRoute.PUT("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.UpdateChannel)
			channelRoute.DELETE("/disabled", middleware.AdminAuth(), controller.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", middleware.AdminAuth(), controller.DisableTagChannels)
			channelRoute.POST("/tag/enabled", middleware.AdminAuth(), controller.EnableTagChannels)
			channelRoute.PUT("/tag", middleware.AdminAuth(), controller.EditTagChannels)
			channelRoute.DELETE("/:id", middleware.AdminAuth(), controller.DeleteChannel)
			channelRoute.POST("/batch", middleware.AdminAuth(), controller.DeleteChannelBatch)
			channelRoute.POST("/fix", middleware.AdminAuth(), controller.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.FetchModels)
			channelRoute.POST("/codex/oauth/start", middleware.AdminAuth(), controller.StartCodexOAuth)
			channelRoute.POST("/codex/oauth/complete", middleware.AdminAuth(), controller.CompleteCodexOAuth)
			channelRoute.POST("/:id/codex/oauth/start", middleware.AdminAuth(), controller.StartCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/oauth/complete", middleware.AdminAuth(), controller.CompleteCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/refresh", middleware.AdminAuth(), controller.RefreshCodexChannelCredential)
			channelRoute.GET("/:id/codex/usage", middleware.AdminAuth(), controller.GetCodexChannelUsage)
			channelRoute.POST("/ollama/pull", middleware.AdminAuth(), controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", middleware.AdminAuth(), controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", middleware.AdminAuth(), controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", middleware.AdminAuth(), controller.OllamaVersion)
			channelRoute.POST("/batch/tag", middleware.AdminAuth(), controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", middleware.AdminAuth(), controller.GetTagModels)
			channelRoute.POST("/copy/:id", middleware.AdminAuth(), controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", middleware.AdminAuth(), controller.ManageMultiKeys)
			channelRoute.POST("/upstream_updates/apply", middleware.AdminAuth(), controller.ApplyChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/apply_all", middleware.AdminAuth(), controller.ApplyAllChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect", middleware.AdminAuth(), controller.DetectChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect_all", middleware.AdminAuth(), controller.DetectAllChannelUpstreamModelUpdates)
			// 上架向导：诊断 + 局部模型更新 + 元数据自动推断
			channelRoute.GET("/:id/onboard", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.OnboardChannel)
			channelRoute.PATCH("/:id/models", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.UpdateChannelModels)
			channelRoute.POST("/:id/onboard/auto_meta", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.AutoMetaChannelModels)
			channelRoute.POST("/:id/onboard/test", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.BulkTestChannelModels)
			channelRoute.GET("/:id/test_results", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetChannelTestResults)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKey)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AdminAuth())
		{
			redemptionRoute.GET("/", controller.GetAllRedemptions)
			redemptionRoute.GET("/search", controller.SearchRedemptions)
			redemptionRoute.GET("/:id", controller.GetRedemption)
			redemptionRoute.POST("/", controller.AddRedemption)
			redemptionRoute.PUT("/", controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", controller.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AdminAuth(), controller.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminAuth(), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.AdminAuth(), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.AdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		{
			groupRoute.GET("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		{
			prefillGroupRoute.GET("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetPrefillGroups)
			prefillGroupRoute.Use(middleware.AdminAuth())
			prefillGroupRoute.POST("/", controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", middleware.AdminAuth(), controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("/", middleware.AdminAuth(), controller.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		{
			vendorRoute.GET("/", middleware.AdminAuth(), controller.GetAllVendors)
			vendorRoute.GET("/search", middleware.AdminAuth(), controller.SearchVendors)
			vendorRoute.GET("/:id", middleware.AdminAuth(), controller.GetVendorMeta)
			vendorRoute.POST("/", middleware.AdminAuth(), controller.CreateVendorMeta)
			vendorRoute.PUT("/", middleware.AdminAuth(), controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", middleware.AdminAuth(), controller.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		{
			modelsRoute.GET("/sync_upstream/preview", middleware.AdminAuth(), controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", middleware.AdminAuth(), controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", middleware.AdminAuth(), controller.GetMissingModels)
			modelsRoute.GET("/tags", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetModelTags)
			modelsRoute.GET("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetAllModelsMeta)
			modelsRoute.GET("/search", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.SearchModelsMeta)
			modelsRoute.GET("/:id", middleware.AdminAuth(), controller.GetModelMeta)
			modelsRoute.POST("/", middleware.AdminAuth(), controller.CreateModelMeta)
			modelsRoute.POST("/batch_tags", middleware.AdminAuth(), controller.BatchSetModelTags)
			modelsRoute.PUT("/", middleware.AdminAuth(), controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", middleware.AdminAuth(), controller.DeleteModelMeta)
		}

		// Deployments (model deployment management)
		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.AdminAuth())
		{
			deploymentsRoute.GET("/settings", controller.GetModelDeploymentSettings)
			deploymentsRoute.POST("/settings/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/", controller.GetAllDeployments)
			deploymentsRoute.GET("/search", controller.SearchDeployments)
			deploymentsRoute.POST("/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/hardware-types", controller.GetHardwareTypes)
			deploymentsRoute.GET("/locations", controller.GetLocations)
			deploymentsRoute.GET("/available-replicas", controller.GetAvailableReplicas)
			deploymentsRoute.POST("/price-estimation", controller.GetPriceEstimation)
			deploymentsRoute.GET("/check-name", controller.CheckClusterNameAvailability)
			deploymentsRoute.POST("/", controller.CreateDeployment)

			deploymentsRoute.GET("/:id", controller.GetDeployment)
			deploymentsRoute.GET("/:id/logs", controller.GetDeploymentLogs)
			deploymentsRoute.GET("/:id/containers", controller.ListDeploymentContainers)
			deploymentsRoute.GET("/:id/containers/:container_id", controller.GetContainerDetails)
			deploymentsRoute.PUT("/:id", controller.UpdateDeployment)
			deploymentsRoute.PUT("/:id/name", controller.UpdateDeploymentName)
			deploymentsRoute.POST("/:id/extend", controller.ExtendDeployment)
			deploymentsRoute.DELETE("/:id", controller.DeleteDeployment)
		}
	}
}
