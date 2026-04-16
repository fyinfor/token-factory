package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetAffInvitees 分页返回当前登录用户邀请注册的用户列表及各自分销比例（万分比）。
func GetAffInvitees(c *gin.Context) {
	inviterId := c.GetInt("id")
	u, err := model.GetUserById(inviterId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看邀请列表"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListAffInvitees(inviterId, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":                        items,
			"total":                        total,
			"default_commission_ratio_bps": common.AffiliateDefaultCommissionBps,
		},
	})
}

type updateAffInviteeCommissionRequest struct {
	InviterId          int `json:"inviter_id"`
	InviteeId          int `json:"invitee_id"`
	CommissionRatioBps int `json:"commission_ratio_bps"`
}

// PutAffInviteeCommission 管理员修改指定邀请人与其被邀请人之间的分销比例（0–10000 万分比）。
// 路由挂载在 AdminAuth 下，仅管理员/超级管理员可调用；需显式传 inviter_id，防止冒充邀请人越权改比例。
func PutAffInviteeCommission(c *gin.Context) {
	myRole := c.GetInt("role")
	if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "permission denied"})
		return
	}
	var req updateAffInviteeCommissionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request body"})
		return
	}
	if req.InviterId <= 0 || req.InviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid inviter_id or invitee_id"})
		return
	}
	if err := model.UpdateAffInviteeCommission(req.InviterId, req.InviteeId, req.CommissionRatioBps); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
