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
		apiRouter.GET("/changelog", controller.ListPublicChangelogs)
		apiRouter.GET("/compute-page/status", controller.GetComputePageStatus)
		apiRouter.GET("/compute-page/content", controller.GetComputePageContent)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/api/vendors", controller.GetVendors)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.POST("/real-name/start", middleware.CriticalRateLimit(), controller.StartPublicRealNameVerification)
		apiRouter.GET("/real-name/status", controller.GetPublicRealNameVerificationStatus)
		apiRouter.GET("/about", controller.GetAbout)
		apiRouter.GET("/api/pricing", controller.GetPricing)
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
		apiRouter.POST("/price_sync", middleware.CriticalRateLimit(), controller.PriceSync)

		// Seedance callback_url 探测：创建会话后把返回的 callback_url 填入透传请求，
		// 上游状态变化会 POST 到 Receive；再用 GET 同路径检查是否成功收到回调。
		apiRouter.POST("/debug/seedance/callback", middleware.CriticalRateLimit(), controller.CreateSeedanceCallbackProbe)
		apiRouter.POST("/debug/seedance/callback/:token", middleware.CriticalRateLimit(), controller.ReceiveSeedanceCallbackProbe)
		apiRouter.GET("/debug/seedance/callback/:token", middleware.CriticalRateLimit(), controller.InspectSeedanceCallbackProbe)
		apiRouter.DELETE("/debug/seedance/callback/:token", middleware.CriticalRateLimit(), controller.DeleteSeedanceCallbackProbe)

		// 终端用户素材库 Action 网关：POST /api/material?Action=xxx（Token 鉴权）。
		// 与 Web 控制台 REST 路由（UserAuth）及上游 tokenspace 内部调用（seedance_material）三层隔离。
		apiRouter.POST("/material", middleware.TokenAuth(), middleware.CriticalRateLimit(), controller.HandleMaterialAction)

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
			distributorRoute.GET("/identity_application", controller.GetMyDistributorIdentityApplication)
			distributorRoute.POST("/identity_application", controller.PostDistributorIdentityApplication)
			distributorRoute.GET("/center", controller.GetDistributorCenterInfo)
			distributorRoute.GET("/analytics", controller.GetDistributorAnalytics)
			distributorRoute.GET("/bindable_user", controller.GetDistributorBindableUser)
			distributorRoute.POST("/bind_requests", controller.PostDistributorBindRequest)
			distributorRoute.POST("/bind_requests/:id/accept", controller.AcceptDistributorBindRequest)
			distributorRoute.POST("/bind_requests/:id/reject", controller.RejectDistributorBindRequest)
			distributorRoute.GET("/invitee/:invitee_id/commissions", controller.GetDistributorInviteeCommissionLogs)
			distributorRoute.GET("/invitee/:invitee_id/profit-shares", controller.GetDistributorInviteeProfitShareLogs)
			distributorRoute.GET("/invitee/:invitee_id/topups", controller.GetDistributorInviteeTopUps)
			distributorRoute.GET("/invitee-model-discounts", controller.GetInviteeModelDiscounts)
			distributorRoute.GET("/invitee-model-discounts/export", controller.ExportInviteeModelDiscounts)
			distributorRoute.PUT("/invitee-model-discounts", controller.PutInviteeModelDiscounts)
			distributorRoute.POST("/invitee-model-discounts/batch", controller.PostBatchInviteeModelDiscounts)
			distributorRoute.GET("/model-discount-template", controller.GetDistributorModelDiscountTemplate)
			distributorRoute.GET("/model-discount-template/export", controller.ExportDistributorModelDiscountTemplate)
			distributorRoute.PUT("/model-discount-template", controller.PutDistributorModelDiscountTemplate)
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
			distributorAdminRoute.GET("/identity_applications", controller.ListDistributorIdentityApplicationsAdmin)
			distributorAdminRoute.GET("/identity_applications/:id", controller.GetDistributorIdentityApplicationAdmin)
			distributorAdminRoute.POST("/identity_applications/:id/approve", controller.ApproveDistributorIdentityApplicationAdmin)
			distributorAdminRoute.POST("/identity_applications/:id/reject", controller.RejectDistributorIdentityApplicationAdmin)
			distributorAdminRoute.GET("/distributors", controller.ListDistributorsAdmin)
			distributorAdminRoute.PUT("/distributors/commission", controller.PutDistributorsCommissionAdmin)
			distributorAdminRoute.GET("/distributors/:id/application", controller.GetDistributorApplicationByUserAdmin)
			distributorAdminRoute.PUT("/distributors/:id/application", controller.PutDistributorApplicationByUserAdmin)
			distributorAdminRoute.PUT("/distributors/:id/commission", controller.PutDistributorCommissionAdmin)
			distributorAdminRoute.GET("/distributors/:id/invitees", controller.GetDistributorInviteesAdmin)
			distributorAdminRoute.GET("/distributors/:id/invitee-unbind-logs", controller.GetDistributorInviteeUnbindLogsAdmin)
			distributorAdminRoute.GET("/distributors/:id/invitees/:invitee_id/profit-shares", controller.GetDistributorInviteeProfitSharesAdmin)
			distributorAdminRoute.POST("/distributors/:id/invitees/:invitee_id/unbind", controller.PostDistributorInviteeUnbindAdmin)
			distributorAdminRoute.GET("/invitee-model-discounts", controller.GetInviteeModelDiscountsAdmin)
			distributorAdminRoute.GET("/invitee-model-discounts/export", controller.ExportInviteeModelDiscountsAdmin)
			distributorAdminRoute.PUT("/invitee-model-discounts", controller.PutInviteeModelDiscountsAdmin)
			distributorAdminRoute.GET("/model-discount-template/export", controller.ExportDistributorModelDiscountTemplateAdmin)
			distributorAdminRoute.POST("/distributors/:id/settle", controller.PostDistributorSettleAdmin)
			distributorAdminRoute.GET("/withdrawals", controller.ListDistributorWithdrawalsAdmin)
			distributorAdminRoute.POST("/withdrawals/:id/approve", controller.ApproveDistributorWithdrawalAdmin)
			distributorAdminRoute.POST("/withdrawals/:id/reject", controller.RejectDistributorWithdrawalAdmin)
			distributorAdminRoute.GET("/analytics", controller.GetDistributorAdminAnalytics)
			distributorAdminRoute.GET("/file/download", controller.DownloadDistributorAdminFile)
		}

		apiRouter.POST("/stripe/webhook", controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", controller.WaffoWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		// 通用文件上传与超级管理员文件管理。
		apiRouter.POST("/oss/upload", middleware.UserAuth(), middleware.UploadRateLimit(), controller.OssUpload)
		apiRouter.GET("/oss/files", middleware.RootAuth(), controller.ListUploadFiles)
		apiRouter.POST("/oss/files/sync", middleware.RootAuth(), controller.SyncUploadFiles)
		apiRouter.POST("/oss/files/batch-delete", middleware.RootAuth(), controller.BatchDeleteUploadFiles)
		apiRouter.PUT("/oss/files/:id/expiration", middleware.RootAuth(), controller.UpdateUploadFileExpiration)
		apiRouter.DELETE("/oss/files/:id", middleware.RootAuth(), controller.DeleteUploadFile)

		apiRouter.GET("/playground/media-options", middleware.CORS(), middleware.TokenAuth(), controller.GetPlaygroundMediaOptions)

		playgroundRoute := apiRouter.Group("/playground")
		playgroundRoute.Use(middleware.UserAuth(), middleware.Distribute())
		{
			playgroundRoute.POST("/chat/completions", controller.Playground)
			playgroundRoute.POST("/images/generations", controller.PlaygroundImage)
			playgroundRoute.POST("/images/edits", controller.PlaygroundImageEdits)
			playgroundRoute.GET("/images/generations/:task_id", controller.PlaygroundImageFetch)
			playgroundRoute.POST("/videos", controller.PlaygroundVideo)
			playgroundRoute.GET("/videos/:task_id", controller.PlaygroundVideoFetch)
		}

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), controller.Verify2FALogin)
			userRoute.POST("/login/sms", middleware.CriticalRateLimit(), controller.SMSLogin)
			userRoute.POST("/login/sms/send", middleware.CriticalRateLimit(), controller.SendSMSLoginVerification)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.POST("/ubcoin/notify", controller.UcoinNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self/phone_available", controller.UserSelfCheckPhoneAvailable)
				selfRoute.GET("/self/sms_bind_verification", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendSMSBindVerification)
				selfRoute.POST("/self/phone/bind", middleware.CriticalRateLimit(), controller.PhoneBind)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.POST("/self/real-name-verification", middleware.CriticalRateLimit(), controller.CreateRealNameVerification)
				selfRoute.GET("/self/real-name-verification", controller.GetRealNameVerificationStatus)
				selfRoute.POST("/student/apply", controller.ApplyStudent)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.GET("/playground/image-pricing-tiers", controller.GetPlaygroundImagePricingTiers)
				selfRoute.GET("/playground/video-pricing-tiers", controller.GetPlaygroundVideoPricingTiers)
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
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.RequestWaffoPay)
				selfRoute.POST("/ubcoin/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.RequestUcoinPay)
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
				selfRoute.GET("/supplier-dashboard/model-users", controller.GetSupplierDashboardModelUserUsage)
				selfRoute.GET("/supplier-dashboard/export", controller.ExportSupplierDashboardUsage)
				selfRoute.GET("/supplier-channel-logs", controller.GetSupplierChannelLogs)
				selfRoute.GET("/supplier-channel-logs/stat", controller.GetSupplierChannelLogsStat)
				selfRoute.GET("/supplier-channel-logs/export", middleware.SearchRateLimit(), controller.ExportSupplierChannelLogs)
				selfRoute.GET("/supplier/pricing/global", controller.GetSupplierGlobalPricing)
				selfRoute.PUT("/supplier/pricing/global", controller.PutSupplierGlobalPricing)
				selfRoute.GET("/supplier/pricing/channel/:channel_id", controller.GetSupplierChannelPricing)
				selfRoute.PUT("/supplier/pricing/channel/:channel_id", controller.PutSupplierChannelPricing)
				selfRoute.GET("/messages/self", controller.ListMyMessages)
				selfRoute.POST("/messages/:id/read", controller.MarkMyMessageRead)
				selfRoute.POST("/messages/read_all", controller.MarkAllMyMessagesRead)
				selfRoute.GET("/messages/unread_count", controller.GetMyUnreadMessageCount)

				selfRoute.GET("/invoice/profile", controller.GetInvoiceProfile)
				selfRoute.PUT("/invoice/profile", controller.PutInvoiceProfile)
				selfRoute.GET("/invoice/eligible-orders", controller.GetInvoiceEligibleOrders)
				selfRoute.GET("/invoice/balance-summary", controller.GetInvoiceBalanceSummarySelf)
				selfRoute.POST("/invoice/request", controller.PostInvoiceRequest)
				selfRoute.GET("/invoice/requests", controller.GetInvoiceRequestsSelf)
				selfRoute.GET("/invoice/requests/:id", controller.GetInvoiceRequestDetailSelf)
				selfRoute.POST("/invoice/requests/:id/cancel", controller.CancelInvoiceRequestSelf)

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

				// 用户智能路由策略（本地 route_* 表）
				selfRoute.GET("/route-policy", controller.UserGetRoutePolicy)
				selfRoute.PUT("/route-policy/mode", controller.UserUpdateRouteMode)
				selfRoute.POST("/route-policy/weights", controller.UserUpsertRouteWeight)
				selfRoute.DELETE("/route-policy/weights/:id", controller.UserDeleteRouteWeight)
				selfRoute.POST("/route-policy/overrides", controller.UserUpsertRouteOverride)
				selfRoute.DELETE("/route-policy/overrides/:id", controller.UserDeleteRouteOverride)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.GET("/", controller.GetAllUsers)
				adminRoute.GET("/tags", controller.GetUserTags)
				adminRoute.GET("/topup", controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
				adminRoute.GET("/invoice/admin/requests", controller.ListInvoiceRequestsAdmin)
				adminRoute.GET("/invoice/admin/requests/:id", controller.GetInvoiceRequestDetailAdmin)
				adminRoute.POST("/invoice/admin/requests/:id/process", controller.MarkInvoiceRequestProcessingAdmin)
				adminRoute.POST("/invoice/admin/requests/:id/issue", controller.IssueInvoiceRequestAdmin)
				adminRoute.POST("/invoice/admin/upload", controller.UploadInvoiceFileAdmin)
				adminRoute.POST("/invoice/admin/requests/:id/reject", controller.RejectInvoiceRequestAdmin)
				adminRoute.POST("/invoice/admin/backfill-attribution", controller.AdminBackfillInvoiceAttribution)
				adminRoute.POST("/invoice/admin/grant-gift", controller.AdminGrantGiftQuota)
				adminRoute.POST("/invoice/admin/corporate-topup", controller.AdminCorporateTopUp)
				adminRoute.GET("/search", controller.SearchUsers)
				adminRoute.GET("/import/template", controller.DownloadUserImportTemplate)
				adminRoute.POST("/import", controller.ImportUsers)
				adminRoute.GET("/import/failures/:id", controller.DownloadUserImportFailures)
				adminRoute.GET("/supplier/application", controller.AdminListSupplierApplications)
				adminRoute.PUT("/supplier/application/:id", controller.AdminUpdateSupplierApplication)
				adminRoute.POST("/supplier/application/activate", controller.ActivateSupplierApplication)
				adminRoute.GET("/supplier/list", controller.AdminListSuppliers)
				adminRoute.GET("/supplier/:id/revenue-push/config", controller.GetSupplierRevenuePushConfig)
				adminRoute.PUT("/supplier/:id/revenue-push/config", controller.PutSupplierRevenuePushConfig)
				adminRoute.POST("/supplier/:id/revenue-push/test", controller.TestSupplierRevenuePush)
				adminRoute.POST("/supplier/:id/revenue-push/run", controller.RunSupplierRevenuePush)
				adminRoute.POST("/supplier/:id/revenue-push/manual", controller.ManualSupplierRevenuePush)
				adminRoute.GET("/supplier/:id/revenue-push/deliveries", controller.ListSupplierRevenuePushDeliveries)
				adminRoute.GET("/supplier/:id/revenue-push/deliveries/:delivery_id/attempts", controller.ListSupplierRevenuePushAttempts)
				adminRoute.POST("/supplier/:id/revenue-push/deliveries/:delivery_id/resolve", controller.ResolveSupplierRevenuePushDelivery)
				adminRoute.GET("/supplier/:id", controller.AdminGetSupplierDetail)
				adminRoute.POST("/supplier/application/:id/review", controller.AdminReviewSupplierApplication)
				adminRoute.POST("/messages/publish", controller.AdminPublishUserMessage)
				adminRoute.GET("/:id/oauth/bindings", controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", controller.AdminClearUserBinding)
				adminRoute.GET("/check_phone", controller.AdminCheckPhoneAvailable)
				adminRoute.GET("/:id", controller.GetUser)
				adminRoute.GET("/:id/log/export", controller.ExportUserLogsAdmin)
				adminRoute.POST("/log/export_all", controller.ExportAllUsersLogsAdmin)
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
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), middleware.RequireRealNameVerificationForTopUp(), controller.SubscriptionRequestCreemPay)
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
		changelogAdminRoute := apiRouter.Group("/changelog/admin")
		changelogAdminRoute.Use(middleware.AdminAuth())
		{
			changelogAdminRoute.GET("/", controller.AdminListChangelogs)
			changelogAdminRoute.POST("/", controller.AdminCreateChangelog)
			changelogAdminRoute.PUT("/:id", controller.AdminUpdateChangelog)
			changelogAdminRoute.DELETE("/:id", controller.AdminDeleteChangelog)
		}

		computePageAdminRoute := apiRouter.Group("/compute-page/admin")
		computePageAdminRoute.Use(middleware.RootAuth())
		{
			computePageAdminRoute.GET("/", controller.AdminGetComputePageConfig)
			computePageAdminRoute.PUT("/enabled", controller.AdminUpdateComputePageEnabled)
			computePageAdminRoute.PUT("/javascript", controller.AdminUpdateComputePageJavaScript)
			computePageAdminRoute.PUT("/popups", controller.AdminUpdateComputePagePopups)
			computePageAdminRoute.POST("/content", controller.AdminUploadComputePageHTML)
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
		apiRouter.GET("/perf_metrics", controller.GetPerfMetrics)
		apiRouter.GET("/perf_metrics/summary", controller.GetPerfMetricsSummary)
		apiRouter.GET("/rankings", controller.GetRankings)
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
		// 价格导出/导入（仅管理员）
		priceRoute := apiRouter.Group("/admin/price")
		priceRoute.Use(middleware.AdminAuth())
		{
			priceRoute.GET("/export", controller.ExportPrices)
			priceRoute.POST("/import", controller.ImportPrices)
		}

		// 用户指定价管理（仅管理员）：按 用户×模型 覆盖三折扣并约束智能路由价格上限
		userModelPricingRoute := apiRouter.Group("/user_model_pricing")
		userModelPricingRoute.Use(middleware.AdminAuth())
		{
			userModelPricingRoute.GET("/", controller.ListUserModelPricing)
			userModelPricingRoute.GET("/users", controller.ListUserModelPricingUsers)
			userModelPricingRoute.GET("/preview", controller.PreviewUserModelPricing)
			userModelPricingRoute.GET("/import_preview", controller.PreviewImportUserModelPricing)
			userModelPricingRoute.GET("/export", controller.ExportUserModelPricing)
			userModelPricingRoute.POST("/", controller.UpsertUserModelPricing)
			userModelPricingRoute.POST("/import", controller.ImportUserModelPricing)
			userModelPricingRoute.POST("/convert_channel_list", controller.ConvertUserModelPricingToChannelList)
			userModelPricingRoute.DELETE("/by_user/:user_id", controller.DeleteUserModelPricingByUser)
			userModelPricingRoute.DELETE("/:id", controller.DeleteUserModelPricing)
		}

		tfOpenSyncRoute := apiRouter.Group("/tf_open_sync")
		{
			// 子站 TokenFactoryOpen 拉全站渠道（脱敏+定价）；鉴权见 controller.authorizeTFOpenSyncExport
			tfOpenSyncRoute.GET("/channels", middleware.CriticalRateLimit(), controller.TFOpenSyncExportChannels)
			tfOpenSyncRoute.POST("/channel_test", middleware.CriticalRateLimit(), controller.TFOpenSyncChannelTest)
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
			// 渠道导出/导入（仅管理员）
			channelRoute.POST("/export", middleware.AdminAuth(), controller.ExportChannels)
			channelRoute.POST("/import", middleware.AdminAuth(), controller.ImportChannels)
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
			// 渠道-模型热力配置
			channelRoute.GET("/heats", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetChannelModelHeats)
			channelRoute.GET("/:id/heats", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetChannelModelHeatsByChannel)
			channelRoute.PUT("/heat", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.SaveChannelModelHeat)
			channelRoute.PUT("/heats/batch", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.BatchSaveChannelModelHeats)
			channelRoute.DELETE("/:id/heats/:model_name", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.DeleteChannelModelHeat)
			channelRoute.GET("/heat/period", middleware.AdminAuth(), controller.GetHeatStatPeriod)
			channelRoute.PUT("/heat/period", middleware.AdminAuth(), controller.SetHeatStatPeriod)
			channelRoute.GET("/hot-overrides", middleware.AdminAuth(), controller.GetChannelModelHotOverrides)
			channelRoute.PUT("/hot-override", middleware.AdminAuth(), controller.SaveChannelModelHotOverride)
			channelRoute.PUT("/hot-overrides/batch", middleware.AdminAuth(), controller.BatchSaveChannelModelHotOverrides)
			channelRoute.GET("/hot-settings", middleware.AdminAuth(), controller.GetHomeHotSettings)
			channelRoute.PUT("/hot-settings", middleware.AdminAuth(), controller.SetHomeHotSettings)
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

		materialRoute := apiRouter.Group("/material")
		materialRoute.Use(middleware.UserAuth())
		{
			materialRoute.GET("/config", controller.GetMaterialConfig)
			materialRoute.GET("/group", controller.GetMaterialGroup)
			materialRoute.GET("/assets", controller.ListMaterialAssets)
			materialRoute.POST("/upload", middleware.UploadRateLimit(), controller.UploadMaterial)
			materialRoute.POST("/upload-url", middleware.UploadRateLimit(), controller.UploadMaterialByURL)
			materialRoute.GET("/asset/:asset_id", controller.GetMaterial)
			materialRoute.PUT("/asset/:asset_id", controller.UpdateMaterial)
			materialRoute.DELETE("/asset/:asset_id", controller.DeleteMaterial)
			// 真人认证会话（Web 控制台，BytedToken 仅后端存储）。
			materialRoute.POST("/visual/session", controller.CreateVisualSession)
			materialRoute.GET("/visual/result", controller.PollVisualResult)
			// 真人分组与素材管理（Web 控制台）。
			materialRoute.GET("/real/groups", controller.ListRealGroups)
			materialRoute.PUT("/real/groups/:group_id", controller.UpdateRealGroup)
			materialRoute.DELETE("/real/groups/:group_id", controller.DeleteRealGroup)
			materialRoute.GET("/real/assets", controller.ListRealAssets)
			materialRoute.POST("/real/upload", middleware.UploadRateLimit(), controller.UploadRealMaterial)
			materialRoute.POST("/real/upload-url", middleware.UploadRateLimit(), controller.UploadRealMaterialByURL)
			materialRoute.GET("/real/asset/:asset_id", controller.GetRealMaterial)
			materialRoute.PUT("/real/asset/:asset_id", controller.UpdateRealMaterial)
			materialRoute.DELETE("/real/asset/:asset_id", controller.DeleteRealMaterial)
		}

		// 个人素材接口：基于用户 API 令牌（sk-xxx）鉴权，自动识别归属用户，
		// 仅允许操作当前令牌所属用户的个人素材，与 /api/material/* 互不干扰。
		personalMaterialRoute := apiRouter.Group("/material/personal")
		personalMaterialRoute.Use(middleware.TokenAuth())
		{
			personalMaterialRoute.POST("/upload", middleware.UploadRateLimit(), controller.UploadPersonalMaterial)
			personalMaterialRoute.POST("/upload-url", middleware.UploadRateLimit(), controller.UploadPersonalMaterialByURL)
			personalMaterialRoute.GET("/assets", controller.ListPersonalMaterialAssets)
			personalMaterialRoute.DELETE("/asset/:asset_id", controller.DeletePersonalMaterial)
			personalMaterialRoute.GET("/asset/:asset_id", controller.GetPersonalMaterial)
			// 真人认证会话（个人 API 令牌，直接返回 BytedToken 供程序化客户端轮询）。
			personalMaterialRoute.POST("/visual/session", controller.CreatePersonalVisualSession)
			personalMaterialRoute.POST("/visual/result", controller.GetPersonalVisualResult)
			// 真人分组与素材管理（个人 API 令牌）。
			personalMaterialRoute.GET("/real/groups", controller.ListPersonalRealGroups)
			personalMaterialRoute.PUT("/real/groups/:group_id", controller.UpdatePersonalRealGroup)
			personalMaterialRoute.DELETE("/real/groups/:group_id", controller.DeletePersonalRealGroup)
			personalMaterialRoute.GET("/real/assets", controller.ListPersonalRealAssets)
			personalMaterialRoute.POST("/real/upload", middleware.UploadRateLimit(), controller.UploadPersonalRealMaterial)
			personalMaterialRoute.POST("/real/upload-url", middleware.UploadRateLimit(), controller.UploadPersonalRealMaterialByURL)
			personalMaterialRoute.GET("/real/asset/:asset_id", controller.GetPersonalRealMaterial)
			personalMaterialRoute.DELETE("/real/asset/:asset_id", controller.DeletePersonalRealMaterial)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
			userUsageRoute := usageRoute.Group("/user")
			userUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				userUsageRoute.GET("/", controller.GetUserQuotaByToken)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.UserAuth(), middleware.AdminOrDistributorAuth())
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
		logRoute.GET("/export", middleware.AdminAuth(), middleware.SearchRateLimit(), controller.ExportAdminLogs)
		logRoute.GET("/settlement/export", middleware.AdminAuth(), middleware.SearchRateLimit(), controller.ExportSettlementLogs)
		logRoute.GET("/settlement/summary", middleware.AdminAuth(), middleware.SearchRateLimit(), controller.GetSettlementSummary)
		logRoute.GET("/self/export", middleware.UserAuth(), middleware.SearchRateLimit(), controller.ExportUserLogsSelf)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)
		logRoute.GET("/aliyun-guardrail", middleware.AdminAuth(), controller.GetAliyunGuardrailLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/user", middleware.TokenAuthReadOnly(), controller.GetUserLogsByToken)
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
			modelsRoute.GET("/document_ai/prompts", middleware.AdminAuth(), controller.GetDocumentAIPrompts)
			modelsRoute.PUT("/document_ai/prompts", middleware.AdminAuth(), controller.UpdateDocumentAIPrompts)
			modelsRoute.DELETE("/document_ai/prompts", middleware.AdminAuth(), controller.ResetDocumentAIPrompts)
			modelsRoute.POST("/document_ai/generate", middleware.AdminAuth(), controller.PrepareDocumentAIRequest, middleware.Distribute(), controller.Playground)
			modelsRoute.GET("/sync_upstream/preview", middleware.AdminAuth(), controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", middleware.AdminAuth(), controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", middleware.AdminAuth(), controller.GetMissingModels)
			modelsRoute.GET("/tags", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetModelTags)
			modelsRoute.GET("/export", middleware.AdminAuth(), controller.ExportModelsMeta)
			modelsRoute.GET("/channel_doc_templates", middleware.AdminAuth(), controller.GetChannelModelDocTemplates)
			modelsRoute.POST("/import", middleware.AdminAuth(), controller.ImportModelsMeta)
			modelsRoute.GET("/", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.GetAllModelsMeta)
			modelsRoute.GET("/search", middleware.UserAuth(), middleware.AdminOrApprovedSupplierAuth(), controller.SearchModelsMeta)
			modelsRoute.GET("/:id", middleware.AdminAuth(), controller.GetModelMeta)
			modelsRoute.GET("/:id/channel_docs", middleware.AdminAuth(), controller.GetChannelModelDocs)
			modelsRoute.PUT("/:id/introduction", middleware.AdminAuth(), controller.PutModelIntroduction)
			modelsRoute.POST("/", middleware.AdminAuth(), controller.CreateModelMeta)
			modelsRoute.POST("/batch_tags", middleware.AdminAuth(), controller.BatchSetModelTags)
			modelsRoute.PUT("/", middleware.AdminAuth(), controller.UpdateModelMeta)
			modelsRoute.PUT("/:id/channel_docs", middleware.AdminAuth(), controller.PutChannelModelDoc)
			modelsRoute.POST("/:id/channel_docs/sync", middleware.AdminAuth(), controller.SyncChannelModelDocs)
			modelsRoute.DELETE("/:id/channel_docs", middleware.AdminAuth(), controller.DeleteChannelModelDoc)
			modelsRoute.DELETE("/:id", middleware.AdminAuth(), controller.DeleteModelMeta)
			modelsRoute.POST("/batch_weight", middleware.AdminAuth(), controller.BatchUpdateModelWeight)
		}

		modelVisibilityRoute := apiRouter.Group("/model_visibility")
		modelVisibilityRoute.Use(middleware.AdminAuth())
		{
			modelVisibilityRoute.GET("/sets", controller.GetModelVisibilitySets)
			modelVisibilityRoute.GET("/sets/:id", controller.GetModelVisibilitySet)
			modelVisibilityRoute.POST("/sets", controller.CreateModelVisibilitySet)
			modelVisibilityRoute.PUT("/sets", controller.UpdateModelVisibilitySet)
			modelVisibilityRoute.DELETE("/sets/:id", controller.DeleteModelVisibilitySet)
			modelVisibilityRoute.PUT("/models/:id", controller.UpdateModelVisibilityBindings)
			modelVisibilityRoute.GET("/users", controller.SearchModelVisibilityUsers)
			modelVisibilityRoute.POST("/users/preview", controller.PreviewModelVisibilityUsers)
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
