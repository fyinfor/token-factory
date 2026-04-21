package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type affiliateTrackRequest struct {
	Event string `json:"event"`
	Aff   string `json:"aff"`
}

// PostAffiliateTrack 公开埋点：短链点击、带 aff 的注册页浏览（不校验登录）。
func PostAffiliateTrack(c *gin.Context) {
	var req affiliateTrackRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	ev := strings.TrimSpace(strings.ToLower(req.Event))
	aff := strings.TrimSpace(req.Aff)
	if len(aff) > 32 {
		aff = aff[:32]
	}
	if aff == "" || (ev != "short_link_click" && ev != "register_page_view") {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	inviterId, err := model.GetUserIdByAffCode(aff)
	if err != nil || inviterId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	day := time.Now().UTC().Format("2006-01-02")
	var incErr error
	if ev == "short_link_click" {
		incErr = model.UpsertAffFunnelIncrShortLink(inviterId, day)
	} else {
		incErr = model.UpsertAffFunnelIncrRegisterPageView(inviterId, day)
	}
	if incErr != nil {
		common.SysError("affiliate track: " + incErr.Error())
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
