package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// modelTestResultItemDTO 返回给前端的单条 (channel, model) 单测/运营展示信息。
type modelTestResultItemDTO struct {
	ChannelId                 int    `json:"channel_id"`
	ModelName                 string `json:"model_name"`
	LastTestSuccess           bool   `json:"last_test_success"`
	LastResponseTime          int    `json:"last_response_time"`
	ManualDisplayResponseTime int    `json:"manual_display_response_time"`
	ManualStabilityGrade      int    `json:"manual_stability_grade"`
	// DisplayResponseTimeMs 用于颜色/耗时展示：手动耗时优先，否则取最近一次成功时的 last_response_time
	DisplayResponseTimeMs int    `json:"display_response_time_ms"`
	DisplayStabilityGrade int    `json:"display_stability_grade"`
	DisplaySource         string `json:"display_source"` // manual_time | manual_grade | auto | none
}

// buildModelTestResultDTOs 从库行补全 DTO 展示字段：展示耗时时长以手动毫秒优先，否则为最近一次成功时的实测毫秒。
func buildModelTestResultDTOs(rows []model.ModelTestResult) []modelTestResultItemDTO {
	out := make([]modelTestResultItemDTO, 0, len(rows))
	for i := range rows {
		r := rows[i]
		dto := modelTestResultItemDTO{
			ChannelId:                 r.ChannelId,
			ModelName:                 r.ModelName,
			LastTestSuccess:           r.LastTestSuccess,
			LastResponseTime:          r.LastResponseTime,
			ManualDisplayResponseTime: r.ManualDisplayResponseTime,
			ManualStabilityGrade:      r.ManualStabilityGrade,
			DisplayStabilityGrade:     r.ManualStabilityGrade,
		}
		if r.ManualDisplayResponseTime > 0 {
			dto.DisplayResponseTimeMs = r.ManualDisplayResponseTime
			dto.DisplaySource = "manual_time"
		} else if r.LastTestSuccess && r.LastResponseTime > 0 {
			dto.DisplayResponseTimeMs = r.LastResponseTime
			dto.DisplaySource = "auto"
		} else {
			dto.DisplayResponseTimeMs = 0
			if r.ManualStabilityGrade > 0 {
				dto.DisplaySource = "manual_grade"
			} else {
				dto.DisplaySource = "none"
			}
		}
		out = append(out, dto)
	}
	return out
}

// GetModelTestResultsForChannels 查询 model_test_results，支持两种维度：
// 1) model_name= & channel_ids=1,2,3 — 模型广场某模型在多个渠道上的结果；
// 2) channel_id= & model_names=a,b,c — 渠道测试弹窗中某渠道在多个模型上的结果。
// TryUserAuth：与 /api/pricing 一致，未登录也可拉取展示用数据（不含敏感信息）。
func GetModelTestResultsForChannels(c *gin.Context) {
	modelNameSingle := strings.TrimSpace(c.Query("model_name"))
	channelIDsStr := strings.TrimSpace(c.Query("channel_ids"))
	channelIDStr := strings.TrimSpace(c.Query("channel_id"))
	modelNamesStr := strings.TrimSpace(c.Query("model_names"))

	var rows []model.ModelTestResult
	var err error

	if modelNameSingle != "" && channelIDsStr != "" {
		parts := strings.Split(channelIDsStr, ",")
		ids := make([]int, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, e := strconv.Atoi(p)
			if e != nil {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			rows = nil
		} else {
			rows, err = model.GetModelTestResultsByModelNameAndChannelIDs(modelNameSingle, ids)
		}
	} else if channelIDStr != "" {
		cid, e := strconv.Atoi(channelIDStr)
		if e != nil || cid <= 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
		if modelNamesStr != "" {
			names := make([]string, 0)
			for _, n := range strings.Split(modelNamesStr, ",") {
				n = strings.TrimSpace(n)
				if n != "" {
					names = append(names, n)
				}
			}
			rows, err = model.GetModelTestResultsByChannelIDAndModelNames(cid, names)
		} else {
			rows, err = model.GetAllModelTestResultsByChannelID(cid)
		}
	} else {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    buildModelTestResultDTOs(rows),
	})
}

// putModelTestResultDisplayRequest 管理端设置运营展示覆盖。
type putModelTestResultDisplayRequest struct {
	ChannelId                 int    `json:"channel_id"`
	ModelName                 string `json:"model_name"`
	ManualDisplayResponseTime int    `json:"manual_display_response_time"`
	ManualStabilityGrade      int    `json:"manual_stability_grade"`
}

// PutModelTestResultDisplay 管理员/运营更新某 (channel, model) 的展示用响应时间与等级；均为 0 表示取消覆盖。
func PutModelTestResultDisplay(c *gin.Context) {
	var req putModelTestResultDisplayRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.ModelName = strings.TrimSpace(req.ModelName)
	if req.ChannelId <= 0 || req.ModelName == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.SetModelTestResultManualDisplay(req.ChannelId, req.ModelName, req.ManualDisplayResponseTime, req.ManualStabilityGrade); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
