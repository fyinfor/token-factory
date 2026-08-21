package constant

type ContextKey string

const (
	ContextKeyTokenCountMeta  ContextKey = "token_count_meta"
	ContextKeyPromptTokens    ContextKey = "prompt_tokens"
	ContextKeyEstimatedTokens ContextKey = "estimated_tokens"

	ContextKeyOriginalModel    ContextKey = "original_model"
	ContextKeyRequestStartTime ContextKey = "request_start_time"

	/* token related keys */
	ContextKeyTokenUnlimited         ContextKey = "token_unlimited_quota"
	ContextKeyTokenKey               ContextKey = "token_key"
	ContextKeyTokenId                ContextKey = "token_id"
	ContextKeyTokenGroup             ContextKey = "token_group"
	ContextKeyTokenSpecificChannelId ContextKey = "specific_channel_id"
	ContextKeyTokenModelLimitEnabled ContextKey = "token_model_limit_enabled"
	ContextKeyTokenModelLimit        ContextKey = "token_model_limit"
	ContextKeyTokenCrossGroupRetry   ContextKey = "token_cross_group_retry"

	/* channel related keys */
	ContextKeyChannelId                ContextKey = "channel_id"
	ContextKeyChannelName              ContextKey = "channel_name"
	ContextKeyChannelCreateTime        ContextKey = "channel_create_time"
	ContextKeyChannelBaseUrl           ContextKey = "base_url"
	ContextKeyChannelType              ContextKey = "channel_type"
	ContextKeyChannelSetting           ContextKey = "channel_setting"
	ContextKeyChannelOtherSetting      ContextKey = "channel_other_setting"
	ContextKeyChannelParamOverride     ContextKey = "param_override"
	ContextKeyChannelHeaderOverride    ContextKey = "header_override"
	ContextKeyChannelOrganization      ContextKey = "channel_organization"
	ContextKeyChannelAutoBan           ContextKey = "auto_ban"
	ContextKeyChannelModelMapping      ContextKey = "model_mapping"
	ContextKeyChannelStatusCodeMapping ContextKey = "status_code_mapping"
	ContextKeyChannelIsMultiKey        ContextKey = "channel_is_multi_key"
	ContextKeyChannelMultiKeyIndex     ContextKey = "channel_multi_key_index"
	ContextKeyChannelKey               ContextKey = "channel_key"

	ContextKeyAutoGroup           ContextKey = "auto_group"
	ContextKeyAutoGroupIndex      ContextKey = "auto_group_index"
	ContextKeyAutoGroupRetryIndex ContextKey = "auto_group_retry_index"

	// OpenRouter-style provider routing (parsed from chat completion body).
	ContextKeyOpenRouterProviderJSON ContextKey = "openrouter_provider_json"
	ContextKeyRequestModelsList      ContextKey = "request_models_list"
	ContextKeyRequestHasTools        ContextKey = "request_has_tools"
	ContextKeySmartRouteChannelOrder ContextKey = "smart_route_channel_order"
	ContextKeySmartRouteSelectGroup  ContextKey = "smart_route_select_group"

	// ContextKeyForcedChannelID 当用户通过 {alias}/{model}/{channel_no} 形式指定具体渠道调用时，
	// 由分发中间件解析后写入该上下文键；存在该键时跳过 SmartRouter 等自动路由逻辑，且失败不切换渠道。
	ContextKeyForcedChannelID       ContextKey = "forced_channel_id"
	ContextKeyForcedChannelModelKey ContextKey = "forced_channel_model_key"

	// ContextKeyPreferredChannelID 当用户通过 {model}/{route_slug} 指定偏好渠道时写入。
	// 智能路由开启：优先该渠道，失败可按同模型有序候选切换；路由关闭/操练场硬指定时配合 TokenSpecificChannelId 禁止切换。
	ContextKeyPreferredChannelID ContextKey = "preferred_channel_id"

	// ContextKeyNoFailover 当请求头 X-TF-No-Failover 为真时写入。
	// 禁止渠道级重试/切换，用于压测归因等场景。
	ContextKeyNoFailover ContextKey = "no_failover"

	// ContextKeyTFOpenUpstreamChannelRoute 仅当本地渠道类型为 TokenFactoryOpen(60)，且来自上游同步并
	// 记录了有效 route_slug（或旧版 alias|channel_no）时，由 SetupContextForSelectedChannel 写入。
	// relay 层读取后将发往上游 TF 的模型名改写为 "{model}/{route_slug}" 或
	// "{alias}/{model}/{channel_no}"。非 60 类型渠道（即使 source=tokenfactory_open）不得写入/拼接。
	ContextKeyTFOpenUpstreamChannelRoute ContextKey = "tf_open_upstream_channel_route"
	// ContextKeyTFOpenUpstreamChannelNoOverride 允许 playground 在已指定本地渠道时，
	// 通过模型名后缀 "{model}/{n}" 显式覆盖上游 channel_no（写入为 "c<n>"）。
	// 仅对 type=TokenFactoryOpen 且 source=tokenfactory_open 的渠道生效。
	ContextKeyTFOpenUpstreamChannelNoOverride ContextKey = "tf_open_upstream_channel_no_override"

	// ContextKeyForcedSupplierApplicationID 当用户通过 {alias}/{model} 形式指定「某供应商下任意渠道」时，
	// 由分发中间件解析后写入该上下文键（值为 supplier_applications.id，P0 时为 0），
	// 用于将 SmartRouter / 随机回退的候选渠道限制在该供应商内。
	ContextKeyForcedSupplierApplicationID ContextKey = "forced_supplier_application_id"
	// ContextKeyForcedSupplierApplicationIDSet 标志上述键已被有效设置（包括 P0 / 0），
	// 用于区分 "未设置" 与 "设置为 0" 两种语义。
	ContextKeyForcedSupplierApplicationIDSet ContextKey = "forced_supplier_application_id_set"

	/* user related keys */
	ContextKeyUserId      ContextKey = "id"
	ContextKeyUserSetting ContextKey = "user_setting"
	ContextKeyUserQuota   ContextKey = "user_quota"
	ContextKeyUserStatus  ContextKey = "user_status"
	ContextKeyUserEmail   ContextKey = "user_email"
	ContextKeyUserGroup   ContextKey = "user_group"
	ContextKeyUsingGroup  ContextKey = "group"
	ContextKeyUserName    ContextKey = "username"

	ContextKeyLocalCountTokens ContextKey = "local_count_tokens"

	ContextKeySystemPromptOverride ContextKey = "system_prompt_override"

	// ContextKeyFileSourcesToCleanup stores file sources that need cleanup when request ends
	ContextKeyFileSourcesToCleanup ContextKey = "file_sources_to_cleanup"

	// ContextKeyAdminRejectReason stores an admin-only reject/block reason extracted from upstream responses.
	// It is not returned to end users, but can be persisted into consume/error logs for debugging.
	ContextKeyAdminRejectReason ContextKey = "admin_reject_reason"

	// ContextKeyLanguage stores the user's language preference for i18n
	ContextKeyLanguage ContextKey = "language"
)
