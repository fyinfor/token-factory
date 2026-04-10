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
	InviteeId          int `json:"invitee_id"`
	CommissionRatioBps int `json:"commission_ratio_bps"`
}

// PutAffInviteeCommission 邀请人修改某一被邀请用户的分销比例（0–10000 万分比）。
func PutAffInviteeCommission(c *gin.Context) {
	inviterId := c.GetInt("id")
	var req updateAffInviteeCommissionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request body"})
		return
	}
	if req.InviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid invitee_id"})
		return
	}
	if err := model.UpdateAffInviteeCommission(inviterId, req.InviteeId, req.CommissionRatioBps); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
