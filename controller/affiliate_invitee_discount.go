/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetInviteeModelDiscounts 获取被邀请用户的模型折扣列表
// GET /api/distributor/invitee-model-discounts?invitee_id=xxx
func GetInviteeModelDiscounts(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}

	inviteeId, err := strconv.Atoi(c.Query("invitee_id"))
	if err != nil || inviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	items, _, err := model.GetInviteeModelDiscounts(userId, inviteeId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": items,
			"total": len(items),
		},
	})
}

// PutInviteeModelDiscounts 更新被邀请用户的模型折扣配置
// PUT /api/distributor/invitee-model-discounts
type putInviteeModelDiscountsRequest struct {
	InviteeId int                                          `json:"invitee_id"`
	Discounts []model.ModelMarkupDiscountRateUpdateRequest `json:"discounts"`
}

func PutInviteeModelDiscounts(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可操作"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}

	var req putInviteeModelDiscountsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}

	if req.InviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	if err := model.UpdateInviteeModelDiscounts(userId, req.InviteeId, req.Discounts); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
