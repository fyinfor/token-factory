package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// OnboardResult 渠道上架诊断结果，前端根据此结构引导用户完成各上架步骤。
type OnboardResult struct {
	// 上游可拉取的模型列表（拉取失败时为空列表）
	ModelsAvailable []string `json:"models_available"`
	// 当前渠道已启用的模型列表
	ModelsImported []string `json:"models_imported"`
	// 已有 model_meta 记录的模型（类型/描述已配置）
	MetaLinked []string `json:"meta_linked"`
	// 缺少 model_meta 记录的模型（需去 /console/models 配置）
	MetaMissing []string `json:"meta_missing"`
	// 已有定价配置的模型
	RatioConfigured []string `json:"ratio_configured"`
	// 缺少定价配置的模型（需同步或手动配置）
	RatioMissing []string `json:"ratio_missing"`
	// 该渠道是否支持上游倍率同步（有 http base_url）
	CanSyncRatio bool `json:"can_sync_ratio"`
	// 满足测试条件：已导入模型 + 所有模型均有定价
	ReadyToTest bool `json:"ready_to_test"`
	// 非阻断性警告信息
	Warnings []string `json:"warnings,omitempty"`
}

// OnboardChannel 渠道上架状态诊断（只读）。
// 拉取上游模型列表、检查 model_meta 配置状态、检查定价配置状态，
// 返回 OnboardResult 供前端引导用户完成各步骤。
func OnboardChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	channel, err := model.GetChannelById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if c.GetInt("role") < common.RoleAdminUser && channel.OwnerUserID != c.GetInt("id") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无权访问其他供应商渠道",
		})
		return
	}

	result := OnboardResult{
		ModelsAvailable: []string{},
		ModelsImported:  []string{},
		MetaLinked:      []string{},
		MetaMissing:     []string{},
		RatioConfigured: []string{},
		RatioMissing:    []string{},
		Warnings:        []string{},
	}

	// 1. 拉取上游模型列表
	upstreamModelIDs, fetchErr := fetchChannelUpstreamModelIDs(channel)
	if fetchErr != nil {
		result.Warnings = append(result.Warnings, "拉取上游模型列表失败: "+fetchErr.Error())
	} else {
		result.ModelsAvailable = upstreamModelIDs
	}

	// 2. 当前渠道已启用的模型
	channelModels := channel.GetModels()
	if channelModels != nil {
		result.ModelsImported = channelModels
	}

	// 3. 诊断目标：优先用已导入的模型，否则用上游可用模型
	diagModels := result.ModelsImported
	if len(diagModels) == 0 {
		diagModels = result.ModelsAvailable
	}

	// 4. 检查 model_meta 记录
	if len(diagModels) > 0 {
		existingNames, _ := model.GetExistingModelNames(diagModels)
		existingSet := make(map[string]bool, len(existingNames))
		for _, name := range existingNames {
			existingSet[name] = true
		}
		for _, m := range diagModels {
			if existingSet[m] {
				result.MetaLinked = append(result.MetaLinked, m)
			} else {
				result.MetaMissing = append(result.MetaMissing, m)
			}
		}
	}

	// 5. 检查定价配置（渠道级优先，再查全局）
	for _, m := range diagModels {
		// 渠道级 price 优先（通过 ratio_sync 配置的渠道专属定价）
		if _, ok := ratio_setting.GetChannelModelPrice(channel.Id, m); ok {
			result.RatioConfigured = append(result.RatioConfigured, m)
			continue
		}
		// 渠道级 ratio
		if _, ok := ratio_setting.GetChannelModelRatio(channel.Id, m); ok {
			result.RatioConfigured = append(result.RatioConfigured, m)
			continue
		}
		// 全局 model_price / model_ratio 兜底
		if _, _, exist := ratio_setting.GetModelRatioOrPrice(m); exist {
			result.RatioConfigured = append(result.RatioConfigured, m)
			continue
		}
		result.RatioMissing = append(result.RatioMissing, m)
	}

	// 6. 是否支持上游 ratio_sync
	if base := channel.GetBaseURL(); strings.HasPrefix(base, "http") {
		result.CanSyncRatio = true
	}

	// 7. 就绪状态
	result.ReadyToTest = len(result.ModelsImported) > 0 && len(result.RatioMissing) == 0

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// UpdateChannelModelsRequest 更新渠道模型列表请求。
type UpdateChannelModelsRequest struct {
	Models []string `json:"models" binding:"required"`
}

// UpdateChannelModels 仅更新渠道的模型列表，同步更新 abilities 表。
// 不需要传输完整渠道信息（包括密钥），适合用于上架向导模型导入场景。
func UpdateChannelModels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req UpdateChannelModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数格式错误: " + err.Error(),
		})
		return
	}

	channel, err := model.GetChannelById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if c.GetInt("role") < common.RoleAdminUser && channel.OwnerUserID != c.GetInt("id") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无权修改其他供应商渠道",
		})
		return
	}

	// 去重并过滤空值
	seen := make(map[string]bool)
	clean := make([]string, 0, len(req.Models))
	for _, m := range req.Models {
		m = strings.TrimSpace(m)
		if m != "" && !seen[m] {
			seen[m] = true
			clean = append(clean, m)
		}
	}
	channel.Models = strings.Join(clean, ",")

	if err := model.DB.Model(channel).Update("models", channel.Models).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := channel.UpdateAbilities(nil); err != nil {
		common.SysError("onboard: failed to update abilities after model patch: " + err.Error())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// AutoMetaRequest 自动推断元数据请求：对指定模型名列表执行自动创建。
// 若 Models 为空，则使用当前渠道的已导入模型列表。
type AutoMetaRequest struct {
	Models []string `json:"models"`
}

// AutoMetaChannelModels 为渠道中缺少 model_meta 的模型自动推断并创建元数据。
// 推断优先级：① 官方预设精确匹配 → ② 模型名称规则推断。
// 已有记录的模型直接跳过，幂等安全。
func AutoMetaChannelModels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	channel, err := model.GetChannelById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if c.GetInt("role") < common.RoleAdminUser && channel.OwnerUserID != c.GetInt("id") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无权访问其他供应商渠道",
		})
		return
	}

	var req AutoMetaRequest
	_ = c.ShouldBindJSON(&req) // 允许空体

	// 目标模型列表：优先用请求体，否则取渠道已导入模型
	targets := req.Models
	if len(targets) == 0 {
		targets = channel.GetModels()
	}
	if len(targets) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "渠道尚未导入任何模型，请先导入模型后再执行自动推断",
		})
		return
	}

	results := service.AutoCreateMissingModelMeta(c.Request.Context(), targets)

	// 统计摘要
	var created, skipped, failed int
	for _, r := range results {
		switch r.Source {
		case "exists":
			skipped++
		default:
			if r.Err != "" {
				failed++
			} else {
				created++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"created": created,
			"skipped": skipped,
			"failed":  failed,
			"items":   results,
		},
	})
}

// BulkTestModelItem 批量测试单个模型的结果。
type BulkTestModelItem struct {
	ModelName string  `json:"model_name"`
	Success   bool    `json:"success"`
	Time      float64 `json:"time"`    // 秒
	Message   string  `json:"message"`
}

// BulkTestChannelModels 批量测试渠道的指定模型列表，每个模型串行执行，
// 避免前端发出大量并发请求触发全局限流。
// POST /api/channel/:id/onboard/test
func BulkTestChannelModels(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 解析请求体
	var req struct {
		Models []string `json:"models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	targets := req.Models
	if len(targets) == 0 {
		targets = channel.GetModels()
	}
	if len(targets) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "no models to test"})
		return
	}

	results := make([]BulkTestModelItem, 0, len(targets))
	for _, modelName := range targets {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		tik := time.Now()
		res := testChannel(channel, modelName, "", false)
		elapsed := float64(time.Since(tik).Milliseconds()) / 1000.0

		// 判断成功与否
		success := res.localErr == nil && res.tokenFactoryError == nil
		msg := ""
		if res.localErr != nil {
			msg = res.localErr.Error()
		} else if res.tokenFactoryError != nil {
			msg = res.tokenFactoryError.Error()
		}

		// 持久化（与单测保持一致）
		ms := int64(elapsed * 1000)
		go func(ch *model.Channel, mn string, ok bool, ms int64, m string) {
			ch.UpdateTestResult(ok, ms, m, mn)
			_ = model.UpsertModelTestResult(ch.Id, mn, ok, ms, m)
		}(channel, modelName, success, ms, msg)

		results = append(results, BulkTestModelItem{
			ModelName: modelName,
			Success:   success,
			Time:      elapsed,
			Message:   msg,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    results,
	})
}

// GetChannelTestResults 返回某渠道在 model_test_results 表中的全部历史测试记录。
// GET /api/channel/:id/test_results
func GetChannelTestResults(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	rows, err := model.GetAllModelTestResultsByChannelID(channelId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
}
