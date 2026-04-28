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

	// ContextKeyResolvedRoutingPolicy 缓存 distributor 解析出的 user 路由策略（*service.ResolvedRoutingPolicy）。
	// 后续在同一请求里被多次消费（distributor 主路径、retry 路径、observability）时避免重复查 DB。
	// 值类型 any 是为了避免 constant 包反向依赖 service 包（循环依赖）。
	ContextKeyResolvedRoutingPolicy ContextKey = "resolved_routing_policy"
	// ContextKeyRoutingPolicySource 仅记录 ResolvedRoutingPolicy.Source（便于日志/监控直接拉取，不必类型断言）。
	ContextKeyRoutingPolicySource ContextKey = "routing_policy_source"
	// ContextKeyRoutingPolicyFallbackUsed 标志主候选池已耗尽且本次请求走过 fallback 兜底（true）。
	// 为 false 表示要么没有候选池约束、要么主候选池就够用。供日志统计兜底命中率使用。
	ContextKeyRoutingPolicyFallbackUsed ContextKey = "routing_policy_fallback_used"

	// ContextKeyForcedChannelID 当用户通过 {alias}/{model}/{channel_no} 形式指定具体渠道调用时，
	// 由分发中间件解析后写入该上下文键；存在该键时跳过 SmartRouter 等自动路由逻辑。
	ContextKeyForcedChannelID       ContextKey = "forced_channel_id"
	ContextKeyForcedChannelModelKey ContextKey = "forced_channel_model_key"

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
