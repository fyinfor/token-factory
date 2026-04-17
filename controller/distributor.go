package controller

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type createDistributorWithdrawalRequest struct {
	RealName      string   `json:"real_name"`
	BankName      string   `json:"bank_name"`
	BankAccount   string   `json:"bank_account"`
	VoucherUrls   []string `json:"voucher_urls"`
	WithdrawMonth string   `json:"withdraw_month"`
	// 使用 float64 兼容前端 JSON 中的小数（如 InputNumber），再取整
	QuotaAmount float64 `json:"quota_amount"`
}

type submitDistributorApplicationRequest struct {
	RealName          string   `json:"real_name"`
	IdCardNo          string   `json:"id_card_no"`
	QualificationUrls []string `json:"qualification_urls"`
	Contact           string   `json:"contact"`
}

// PostDistributorApplication 提交/重新提交分销商申请
func PostDistributorApplication(c *gin.Context) {
	userId := c.GetInt("id")
	var req submitDistributorApplicationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if len(req.QualificationUrls) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请上传资格证书"})
		return
	}
	urlsJSON, err := common.Marshal(req.QualificationUrls)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "资料序列化失败"})
		return
	}
	err = model.UpsertDistributorApplication(userId, req.RealName, req.IdCardNo, string(urlsJSON), req.Contact)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetMyDistributorApplication 当前用户的申请状态
func GetMyDistributorApplication(c *gin.Context) {
	userId := c.GetInt("id")
	app, err := model.GetDistributorApplicationByUserId(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": app})
}

// GetDistributorCenterInfo 分销商中心汇总（邀请短链、默认比例等）
func GetDistributorCenterInfo(c *gin.Context) {
	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if !model.UserIsDistributor(user) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "您不是分销商"})
		return
	}
	if user.AffCode == "" {
		user.AffCode = common.GetRandomString(4)
		if err := user.Update(false); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	bps := user.DistributorCommissionBps
	if bps <= 0 {
		bps = common.AffiliateDefaultCommissionBps
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"aff_code":                   user.AffCode,
			"aff_quota":                  user.AffQuota,
			"aff_history_quota":          user.AffHistoryQuota,
			"aff_count":                  user.AffCount,
			"distributor_commission_bps": user.DistributorCommissionBps,
			"effective_commission_bps":   bps,
			"default_commission_bps":     common.AffiliateDefaultCommissionBps,
		},
	})
}

// GetDistributorInviteeCommissionLogs 分销商查看某一被邀请用户的充值分成明细（按笔：入账额度、当时比例、收益额度）。
func GetDistributorInviteeCommissionLogs(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看"})
		return
	}
	inviteeId, err := strconv.Atoi(c.Param("invitee_id"))
	if err != nil || inviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	invitee, err := model.GetUserById(inviteeId, false)
	if err != nil || invitee.InviterId != userId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权查看或用户不存在"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListAffInviteCommissionLogs(userId, inviteeId, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

type rejectApplicationRequest struct {
	Reason string `json:"reason"`
}

// ListDistributorApplicationsAdmin 管理端：申请列表
func ListDistributorApplicationsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status, _ := strconv.Atoi(c.Query("status"))
	q := model.DistributorApplicationListQuery{
		Keyword:  c.Query("keyword"),
		Status:   status,
		DateFrom: parseInt64Query(c.Query("date_from")),
		DateTo:   parseInt64Query(c.Query("date_to")),
		PageInfo: pageInfo,
	}
	rows, usernames, total, err := model.ListDistributorApplicationsAdmin(q)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, gin.H{
			"id":                 rows[i].Id,
			"user_id":            rows[i].UserId,
			"username":           usernames[i],
			"real_name":          rows[i].RealName,
			"contact":            rows[i].Contact,
			"status":             rows[i].Status,
			"reject_reason":      rows[i].RejectReason,
			"created_at":         rows[i].CreatedAt,
			"id_card_no_mask":    maskIdCard(rows[i].IdCardNo),
			"qualification_urls": rows[i].QualificationUrls,
		})
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

func parseInt64Query(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func maskIdCard(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return "****"
	}
	return id[:4] + strings.Repeat("*", len(id)-8) + id[len(id)-4:]
}

// GetDistributorApplicationAdmin 申请详情（管理员）
func GetDistributorApplicationAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	app, username, err := model.GetDistributorApplicationByIdAdmin(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":                 app.Id,
			"user_id":            app.UserId,
			"username":           username,
			"real_name":          app.RealName,
			"id_card_no":         app.IdCardNo,
			"qualification_urls": app.QualificationUrls,
			"contact":            app.Contact,
			"status":             app.Status,
			"reject_reason":      app.RejectReason,
			"reviewer_id":        app.ReviewerId,
			"reviewed_at":        app.ReviewedAt,
			"created_at":         app.CreatedAt,
			"updated_at":         app.UpdatedAt,
		},
	})
}

// ApproveDistributorApplicationAdmin 通过申请
func ApproveDistributorApplicationAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	reviewerId := c.GetInt("id")
	if err := model.ApproveDistributorApplication(id, reviewerId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if app, _, err := model.GetDistributorApplicationByIdAdmin(id); err == nil && app != nil {
		service.NotifyDistributorApplicationApproved(app.UserId)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// RejectDistributorApplicationAdmin 驳回
func RejectDistributorApplicationAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req rejectApplicationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	reviewerId := c.GetInt("id")
	app, _, errApp := model.GetDistributorApplicationByIdAdmin(id)
	if err := model.RejectDistributorApplication(id, reviewerId, req.Reason); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if errApp == nil && app != nil {
		service.NotifyDistributorApplicationRejected(app.UserId, req.Reason)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// ListDistributorsAdmin 分销商人员列表
func ListDistributorsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	rows, total, err := model.ListDistributorsAdmin(keyword, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, it := range rows {
		u := it.User
		bps := u.DistributorCommissionBps
		if bps <= 0 {
			bps = common.AffiliateDefaultCommissionBps
		}
		items = append(items, gin.H{
			"id":                         u.Id,
			"username":                   u.Username,
			"display_name":               u.DisplayName,
			"application_real_name":      it.ApplicationRealName,
			"needs_supplement":           it.NeedsSupplement,
			"aff_code":                   u.AffCode,
			"aff_count":                  u.AffCount,
			"aff_quota":                  u.AffQuota,
			"aff_history_quota":          u.AffHistoryQuota,
			"distributor_commission_bps": u.DistributorCommissionBps,
			"effective_commission_bps":   bps,
		})
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

type putDistributorCommissionRequest struct {
	DistributorCommissionBps int `json:"distributor_commission_bps"`
}

// PutDistributorCommissionAdmin 设置单个分销商默认分成比例
func PutDistributorCommissionAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req putDistributorCommissionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if err := model.SetUserDistributorCommissionBps(id, req.DistributorCommissionBps); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetDistributorInviteesAdmin 某分销商名下邀请用户明细
func GetDistributorInviteesAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	u, err := model.GetUserById(id, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不是分销商"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListAffInvitees(id, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    pageInfo,
	})
}

// PostDistributorWithdrawal 提交线下提现申请（暂扣 aff_quota）
func PostDistributorWithdrawal(c *gin.Context) {
	userId := c.GetInt("id")
	var req createDistributorWithdrawalRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	urls := make([]string, 0, len(req.VoucherUrls))
	for _, u := range req.VoucherUrls {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请上传票据"})
		return
	}
	urlsJSON, err := common.Marshal(urls)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "票据序列化失败"})
		return
	}
	quotaAmt := int(math.Round(req.QuotaAmount))
	if err := model.CreateDistributorWithdrawal(userId, strings.TrimSpace(req.RealName), strings.TrimSpace(req.BankName), strings.TrimSpace(req.BankAccount), string(urlsJSON), strings.TrimSpace(req.WithdrawMonth), quotaAmt); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	service.NotifyDistributorWithdrawalSubmitted(userId, quotaAmt)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetDistributorWithdrawals 当前用户提现记录
func GetDistributorWithdrawals(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListDistributorWithdrawals(userId, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// PostDistributorWithdrawalCancel 取消待审核提现，退回 aff_quota
func PostDistributorWithdrawalCancel(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	if err := model.CancelDistributorWithdrawal(userId, id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// ListDistributorWithdrawalsAdmin 管理端提现审核列表
func ListDistributorWithdrawalsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status, _ := strconv.Atoi(c.Query("status"))
	keyword := c.Query("keyword")
	rows, total, err := model.ListDistributorWithdrawalsAdmin(status, keyword, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, gin.H{
			"id":              rows[i].Id,
			"user_id":         rows[i].UserId,
			"username":        rows[i].Username,
			"real_name":       rows[i].RealName,
			"bank_name":       rows[i].BankName,
			"bank_account":    rows[i].BankAccount,
			"voucher_urls":    rows[i].VoucherUrls,
			"withdraw_month":  rows[i].WithdrawMonth,
			"quota_amount":    rows[i].QuotaAmount,
			"status":          rows[i].Status,
			"reject_reason":   rows[i].RejectReason,
			"reviewer_id":     rows[i].ReviewerId,
			"reviewed_at":     rows[i].ReviewedAt,
			"cancelled_at":    rows[i].CancelledAt,
			"created_at":      rows[i].CreatedAt,
			"updated_at":      rows[i].UpdatedAt,
		})
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// ApproveDistributorWithdrawalAdmin 审核通过
func ApproveDistributorWithdrawalAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	reviewerId := c.GetInt("id")
	var wUserId int
	if w, err := model.GetDistributorWithdrawalByID(id); err == nil && w != nil {
		wUserId = w.UserId
	}
	if err := model.ApproveDistributorWithdrawalAdmin(id, reviewerId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if wUserId > 0 {
		service.NotifyDistributorWithdrawalApproved(wUserId)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

type rejectWithdrawalRequest struct {
	Reason string `json:"reason"`
}

// RejectDistributorWithdrawalAdmin 驳回并退回 aff_quota
func RejectDistributorWithdrawalAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req rejectWithdrawalRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	reviewerId := c.GetInt("id")
	var wUserId int
	if w, err := model.GetDistributorWithdrawalByID(id); err == nil && w != nil {
		wUserId = w.UserId
	}
	if err := model.RejectDistributorWithdrawalAdmin(id, reviewerId, req.Reason); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if wUserId > 0 {
		service.NotifyDistributorWithdrawalRejected(wUserId, req.Reason)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// PostDistributorSettleAdmin 结账：清空该分销商待结算 aff_quota
func PostDistributorSettleAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	if err := model.AdminSettleDistributorAffQuota(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

type adminUpsertDistributorApplicationRequest struct {
	RealName          string   `json:"real_name"`
	IdCardNo          string   `json:"id_card_no"`
	QualificationUrls []string `json:"qualification_urls"`
	Contact           string   `json:"contact"`
}

// GetDistributorApplicationByUserAdmin 管理端：查看某分销商的申请/认证资料（手工开通可能无记录）
func GetDistributorApplicationByUserAdmin(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	username, app, needsManualEntry, err := model.GetDistributorApplicationProfileByUserIdAdmin(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	data := gin.H{
		"user_id":            userId,
		"username":           username,
		"needs_manual_entry": needsManualEntry,
	}
	if app != nil {
		data["application"] = app
	} else {
		data["application"] = nil
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

// PutDistributorApplicationByUserAdmin 管理端：补录或修改分销商申请资料
func PutDistributorApplicationByUserAdmin(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req adminUpsertDistributorApplicationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	urls := make([]string, 0, len(req.QualificationUrls))
	for _, u := range req.QualificationUrls {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请填写资格证书链接"})
		return
	}
	urlsJSON, err := common.Marshal(urls)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "资料序列化失败"})
		return
	}
	reviewerId := c.GetInt("id")
	if err := model.AdminUpsertDistributorApplicationByUser(userId, reviewerId, req.RealName, req.IdCardNo, string(urlsJSON), req.Contact); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
