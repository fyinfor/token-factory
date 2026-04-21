package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetDistributorAnalytics 分销商专属数据序列 + 被邀请人 TOP。
func GetDistributorAnalytics(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看"})
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	series, err := model.BuildDistributorSelfAnalytics(userId, days)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	topInvitees, err := model.ListInviteeTopForDistributorAnalytics(userId, 10)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	effectiveBps := u.DistributorCommissionBps
	if effectiveBps <= 0 {
		effectiveBps = common.AffiliateDefaultCommissionBps
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"days":                     days,
			"series":                   series,
			"top_invitees":             topInvitees,
			"effective_commission_bps": effectiveBps,
		},
	})
}

// GetDistributorAdminAnalytics 管理端：全平台漏斗/收益序列 + 分销商 TOP 榜。
func GetDistributorAdminAnalytics(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	series, topTotal, topPeriod, topInvite, err := model.BuildPlatformAffiliateAnalytics(days)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	for i := range topPeriod {
		topPeriod[i].PeriodRewardQuota = topPeriod[i].TotalRewardQuota
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"days":                 days,
			"series":               series,
			"top_total_reward":     topTotal,
			"top_period_reward":    topPeriod,
			"top_invitee_count":    topInvite,
		},
	})
}
