package controller

import (
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func distributorDownloadFileName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil {
		base := path.Base(u.Path)
		if base != "." && base != "/" && strings.TrimSpace(base) != "" {
			return strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(base)
		}
	}
	return "image"
}

func normalizeDistributorLocalStorageAccessPrefix(raw string) string {
	prefix := strings.TrimSpace(raw)
	if prefix == "" {
		return "/api"
	}
	if u, err := url.Parse(prefix); err == nil && u.Scheme != "" && u.Path != "" {
		prefix = u.Path
	}
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return "/" + prefix
}

func distributorLocalUploadFileParam(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	cfg := operation_setting.GetOssSetting()
	objPrefix, err := service.NormalizeLocalUploadPrefix(cfg.LocalObjectKeyPrefix)
	if err != nil {
		return "", false
	}
	routes := []struct {
		urlPrefix    string
		objectPrefix string
	}{
		{
			urlPrefix:    "/" + strings.Trim(path.Join(strings.Trim(normalizeDistributorLocalStorageAccessPrefix(cfg.LocalURLPrefix), "/"), service.LocalUploadFolder, objPrefix), "/") + "/",
			objectPrefix: objPrefix,
		},
		{
			urlPrefix:    "/" + strings.Trim(path.Join("api", service.LocalUploadFolder), "/") + "/",
			objectPrefix: "",
		},
	}
	for _, route := range routes {
		if route.urlPrefix == "" || !strings.HasPrefix(u.Path, route.urlPrefix) {
			continue
		}
		fileParam := path.Join(route.objectPrefix, strings.TrimPrefix(u.Path, route.urlPrefix))
		if fileParam == "." || strings.TrimSpace(fileParam) == "" {
			return "", false
		}
		return fileParam, true
	}
	return "", false
}

func tryServeDistributorLocalUploadAttachment(c *gin.Context, rawURL string) bool {
	fileParam, ok := distributorLocalUploadFileParam(rawURL)
	if !ok {
		return false
	}
	cfg := operation_setting.GetOssSetting()
	storeDir := service.LocalUploadBaseDir(cfg.LocalStoragePath)
	fileParam = filepath.Clean("/" + fileParam)
	fileParam = strings.TrimPrefix(fileParam, "/")
	fullPath := filepath.Join(storeDir, filepath.FromSlash(fileParam))
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "file not found"})
		return true
	}
	absDir, err := filepath.Abs(storeDir)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "file not found"})
		return true
	}
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) && absPath != absDir {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "forbidden"})
		return true
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "file not found"})
		return true
	}
	contentType := mime.TypeByExtension(filepath.Ext(absPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(filepath.Base(absPath))
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, url.PathEscape(filename)))
	c.File(absPath)
	return true
}

// DownloadDistributorAdminFile proxies uploaded distributor images as an attachment.
func DownloadDistributorAdminFile(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "url is required"})
		return
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid url"})
		return
	}
	if tryServeDistributorLocalUploadAttachment(c, rawURL) {
		return
	}
	resp, err := service.DoDownloadRequest(rawURL, "distributor_admin_file_download")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("download failed: %d", resp.StatusCode)})
		return
	}
	filename := distributorDownloadFileName(rawURL)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	escapedName := url.PathEscape(filename)
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, escapedName))
	if resp.ContentLength > 0 {
		c.Header("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	}
	_, _ = io.Copy(c.Writer, resp.Body)
}

type createDistributorWithdrawalRequest struct {
	AccountType   int    `json:"account_type"`
	RealName      string `json:"real_name"`
	BankName      string `json:"bank_name"`
	BankAccount   string `json:"bank_account"`
	WithdrawMonth string `json:"withdraw_month"`
	// 使用 float64 兼容前端 JSON 中的小数（如 InputNumber），再取整
	QuotaAmount float64 `json:"quota_amount"`
	// 个人扩展
	IdCardNo          string `json:"id_card_no"`
	IdCardExpiry      string `json:"id_card_expiry"`
	Mobile            string `json:"mobile"`
	BankReservedPhone string `json:"bank_reserved_phone"`
	IdCardFrontUrl    string `json:"id_card_front_url"`
	IdCardBackUrl     string `json:"id_card_back_url"`
	BankCardPhotoUrl  string `json:"bank_card_photo_url"`
	// 企业扩展
	CreditCode               string `json:"credit_code"`
	LegalPersonName          string `json:"legal_person_name"`
	LegalPersonPhone         string `json:"legal_person_phone"`
	BankBranchCode           string `json:"bank_branch_code"`
	ContactPerson            string `json:"contact_person"`
	BusinessLicenseUrl       string `json:"business_license_url"`
	LegalPersonIdCardUrl     string `json:"legal_person_id_card_url"`
	CorporateAccountProofUrl string `json:"corporate_account_proof_url"`
	InvoiceUrl               string `json:"invoice_url"`
}

func distributorWithdrawalToJSON(w model.DistributorWithdrawal, username string) gin.H {
	profile := model.ParseDistributorWithdrawalProfile(w.ProfileData)
	return gin.H{
		"id":                          w.Id,
		"user_id":                     w.UserId,
		"username":                    username,
		"account_type":                w.AccountType,
		"real_name":                   w.RealName,
		"bank_name":                   w.BankName,
		"bank_account":                w.BankAccount,
		"profile_data":                profile,
		"voucher_urls":                w.VoucherUrls,
		"withdraw_month":              w.WithdrawMonth,
		"quota_amount":                w.QuotaAmount,
		"status":                      w.Status,
		"reject_reason":               w.RejectReason,
		"reviewer_id":                 w.ReviewerId,
		"reviewed_at":                 w.ReviewedAt,
		"cancelled_at":                w.CancelledAt,
		"created_at":                  w.CreatedAt,
		"updated_at":                  w.UpdatedAt,
		"id_card_no":                  profile.IdCardNo,
		"id_card_expiry":              profile.IdCardExpiry,
		"mobile":                      profile.Mobile,
		"bank_reserved_phone":         profile.BankReservedPhone,
		"id_card_front_url":           profile.IdCardFrontUrl,
		"id_card_back_url":            profile.IdCardBackUrl,
		"bank_card_photo_url":         profile.BankCardPhotoUrl,
		"credit_code":                 profile.CreditCode,
		"legal_person_name":           profile.LegalPersonName,
		"legal_person_phone":          profile.LegalPersonPhone,
		"bank_branch_code":            profile.BankBranchCode,
		"contact_person":              profile.ContactPerson,
		"business_license_url":        profile.BusinessLicenseUrl,
		"legal_person_id_card_url":    profile.LegalPersonIdCardUrl,
		"corporate_account_proof_url": profile.CorporateAccountProofUrl,
		"invoice_url":                 profile.InvoiceUrl,
	}
}

func buildWithdrawalProfileJSON(req createDistributorWithdrawalRequest) (string, error) {
	accountType := req.AccountType
	if accountType == 0 {
		accountType = model.DistributorApplyTypePersonal
	}
	p := model.DistributorWithdrawalProfile{
		IdCardNo:                 strings.TrimSpace(req.IdCardNo),
		IdCardExpiry:             strings.TrimSpace(req.IdCardExpiry),
		Mobile:                   strings.TrimSpace(req.Mobile),
		BankReservedPhone:        strings.TrimSpace(req.BankReservedPhone),
		IdCardFrontUrl:           strings.TrimSpace(req.IdCardFrontUrl),
		IdCardBackUrl:            strings.TrimSpace(req.IdCardBackUrl),
		BankCardPhotoUrl:         strings.TrimSpace(req.BankCardPhotoUrl),
		CreditCode:               strings.TrimSpace(req.CreditCode),
		LegalPersonName:          strings.TrimSpace(req.LegalPersonName),
		LegalPersonPhone:         strings.TrimSpace(req.LegalPersonPhone),
		BankBranchCode:           strings.TrimSpace(req.BankBranchCode),
		ContactPerson:            strings.TrimSpace(req.ContactPerson),
		BusinessLicenseUrl:       strings.TrimSpace(req.BusinessLicenseUrl),
		LegalPersonIdCardUrl:     strings.TrimSpace(req.LegalPersonIdCardUrl),
		CorporateAccountProofUrl: strings.TrimSpace(req.CorporateAccountProofUrl),
		InvoiceUrl:               strings.TrimSpace(req.InvoiceUrl),
	}
	b, err := common.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type submitDistributorApplicationRequest struct {
	ApplyType         int      `json:"apply_type"`
	RealName          string   `json:"real_name"`
	IdCardNo          string   `json:"id_card_no"`
	QualificationUrls []string `json:"qualification_urls"`
	Contact           string   `json:"contact"`
}

type createDistributorBindRequestBody struct {
	UserID int `json:"user_id"`
}

func GetDistributorBindableUser(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅代理用户可发起绑定"})
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 120 {
		keyword = keyword[:120]
	}
	item, err := model.SearchDistributorBindableUser(userId, keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": item})
}

func PostDistributorBindRequest(c *gin.Context) {
	userId := c.GetInt("id")
	var req createDistributorBindRequestBody
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	bindReq, err := model.CreateDistributorBindRequest(userId, req.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "绑定请求已发送，请等待对方确认", "data": gin.H{"request_id": bindReq.ID}})
}

func AcceptDistributorBindRequest(c *gin.Context) {
	requestId, err := strconv.Atoi(c.Param("id"))
	if err != nil || requestId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求ID"})
		return
	}
	req, err := model.RespondDistributorBindRequest(requestId, c.GetInt("id"), true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已接受绑定请求", "data": gin.H{"status": req.Status}})
}

func RejectDistributorBindRequest(c *gin.Context) {
	requestId, err := strconv.Atoi(c.Param("id"))
	if err != nil || requestId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求ID"})
		return
	}
	req, err := model.RespondDistributorBindRequest(requestId, c.GetInt("id"), false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已拒绝绑定请求", "data": gin.H{"status": req.Status}})
}

// PostDistributorApplication 提交/重新提交分销商申请
func PostDistributorApplication(c *gin.Context) {
	userId := c.GetInt("id")
	var req submitDistributorApplicationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if req.ApplyType == 0 {
		req.ApplyType = model.DistributorApplyTypePersonal
	}
	urlsJSON, err := common.Marshal(req.QualificationUrls)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "资料序列化失败"})
		return
	}
	err = model.UpsertDistributorApplication(userId, req.ApplyType, req.RealName, req.IdCardNo, string(urlsJSON), req.Contact)
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

func GetMyDistributorIdentityApplication(c *gin.Context) {
	userId := c.GetInt("id")
	app, err := model.GetLatestDistributorIdentityApplicationByUserId(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": app})
}

func PostDistributorIdentityApplication(c *gin.Context) {
	userId := c.GetInt("id")
	var req submitDistributorApplicationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if req.ApplyType == 0 {
		req.ApplyType = model.DistributorApplyTypePersonal
	}
	urlsJSON, err := common.Marshal(req.QualificationUrls)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "资料序列化失败"})
		return
	}
	if err := model.SubmitDistributorIdentityApplication(userId, req.ApplyType, req.RealName, req.IdCardNo, string(urlsJSON), req.Contact); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
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
	applyType, applicationRealName, err := model.GetDistributorWithdrawAccountType(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	app, err := model.GetDistributorApplicationByUserId(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	identityNeedsSupplement := !model.IsDistributorApplicationProfileComplete(app)
	applicationIdCardNo := ""
	applicationContact := ""
	applicationQualificationUrls := "[]"
	if app != nil {
		applicationIdCardNo = app.IdCardNo
		applicationContact = app.Contact
		applicationQualificationUrls = app.QualificationUrls
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"aff_code":                       user.AffCode,
			"aff_quota":                      user.AffQuota,
			"aff_history_quota":              user.AffHistoryQuota,
			"aff_count":                      user.AffCount,
			"distributor_commission_bps":     user.DistributorCommissionBps,
			"effective_commission_bps":       bps,
			"default_commission_bps":         common.AffiliateDefaultCommissionBps,
			"apply_type":                     applyType,
			"application_real_name":          applicationRealName,
			"application_id_card_no":         applicationIdCardNo,
			"application_contact":            applicationContact,
			"application_qualification_urls": applicationQualificationUrls,
			"identity_needs_supplement":      identityNeedsSupplement,
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

// GetDistributorInviteeProfitShareLogs 分销商查看某一被邀请用户的利润分成明细（按次结算）。
func GetDistributorInviteeProfitShareLogs(c *gin.Context) {
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
	billingMode := strings.TrimSpace(c.Query("billing_mode"))
	hideZeroReward, _ := strconv.ParseBool(c.Query("hide_zero_reward"))
	items, total, err := model.ListAffInviteProfitShareLogs(userId, inviteeId, billingMode, hideZeroReward, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// GetDistributorInviteeTopUps 分销商查看某个被邀请用户的充值记录。
func GetDistributorInviteeTopUps(c *gin.Context) {
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
	if err != nil || invitee == nil || invitee.InviterId != userId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权查看或用户不存在"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	tradeNoKeyword := parseTopUpTradeNoKeyword(c)
	statusFilter := parseTopUpListStatusFilter(c)
	items, total, err := model.GetUserTopUps(inviteeId, pageInfo, statusFilter, tradeNoKeyword)
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
	applyType, _ := strconv.Atoi(c.Query("apply_type"))
	q := model.DistributorApplicationListQuery{
		Keyword:   c.Query("keyword"),
		Status:    status,
		ApplyType: applyType,
		DateFrom:  parseInt64Query(c.Query("date_from")),
		DateTo:    parseInt64Query(c.Query("date_to")),
		PageInfo:  pageInfo,
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
			"apply_type":         rows[i].ApplyType,
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

func distributorIdentityApplicationListItemToJSON(item model.DistributorIdentityApplicationListItem, includeFullIdCard bool) gin.H {
	app := item.Application
	data := gin.H{
		"id":                 app.Id,
		"user_id":            app.UserId,
		"username":           item.Username,
		"source_apply_type":  app.SourceApplyType,
		"source_real_name":   app.SourceRealName,
		"is_supplement":      app.SourceApplyType == app.TargetApplyType,
		"current_apply_type": item.CurrentApplyType,
		"current_real_name":  item.CurrentRealName,
		"target_apply_type":  app.TargetApplyType,
		"real_name":          app.RealName,
		"contact":            app.Contact,
		"status":             app.Status,
		"reject_reason":      app.RejectReason,
		"reviewer_id":        app.ReviewerId,
		"reviewed_at":        app.ReviewedAt,
		"created_at":         app.CreatedAt,
		"updated_at":         app.UpdatedAt,
		"id_card_no_mask":    maskIdCard(app.IdCardNo),
		"qualification_urls": app.QualificationUrls,
	}
	if includeFullIdCard {
		data["id_card_no"] = app.IdCardNo
	}
	return data
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
			"apply_type":         app.ApplyType,
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
			"ordinary_invite_preview": func() interface{} {
				preview, previewErr := model.GetOrdinaryInviteConversionPreview(app.UserId)
				if previewErr != nil {
					return nil
				}
				return preview
			}(),
		},
	})
}

type approveDistributorApplicationRequest struct {
	DistributorCommissionBps *int `json:"distributor_commission_bps"`
	ConvertOrdinaryInvites   bool `json:"convert_ordinary_invites"`
}

// ApproveDistributorApplicationAdmin 通过申请（可选 body：distributor_commission_bps 万分之一，0=跟随系统）
func ApproveDistributorApplicationAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req approveDistributorApplicationRequest
	body, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "读取请求失败"})
		return
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := common.Unmarshal(body, &req); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
			return
		}
	}
	reviewerId := c.GetInt("id")
	conversionResult, err := model.ApproveDistributorApplication(id, reviewerId, req.DistributorCommissionBps, req.ConvertOrdinaryInvites)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if app, _, err := model.GetDistributorApplicationByIdAdmin(id); err == nil && app != nil {
		service.NotifyDistributorApplicationApproved(app.UserId)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"ordinary_invite_conversion": conversionResult}})
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

func ListDistributorIdentityApplicationsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status, _ := strconv.Atoi(c.Query("status"))
	targetApplyType, _ := strconv.Atoi(c.Query("target_apply_type"))
	rows, total, err := model.ListDistributorIdentityApplicationsAdmin(model.DistributorIdentityApplicationListQuery{
		Keyword:         c.Query("keyword"),
		Status:          status,
		TargetApplyType: targetApplyType,
		PageInfo:        pageInfo,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, distributorIdentityApplicationListItemToJSON(row, false))
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

func GetDistributorIdentityApplicationAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	item, err := model.GetDistributorIdentityApplicationByIdAdmin(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    distributorIdentityApplicationListItemToJSON(*item, true),
	})
}

func ApproveDistributorIdentityApplicationAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	if err := model.ApproveDistributorIdentityApplication(id, c.GetInt("id")); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func RejectDistributorIdentityApplicationAdmin(c *gin.Context) {
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
	if err := model.RejectDistributorIdentityApplication(id, c.GetInt("id"), req.Reason); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// ListDistributorsAdmin 分销商人员列表
func ListDistributorsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	applyType, _ := strconv.Atoi(c.Query("apply_type"))
	rows, total, err := model.ListDistributorsAdmin(model.DistributorListAdminQuery{
		Keyword:   keyword,
		ApplyType: applyType,
		PageInfo:  pageInfo,
	})
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
			"application_apply_type":     it.ApplicationApplyType,
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

type putDistributorsCommissionRequest struct {
	DistributorCommissionBps *int   `json:"distributor_commission_bps"`
	Scope                    string `json:"scope"`
	Keyword                  string `json:"keyword"`
	ApplyType                *int   `json:"apply_type"`
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

// PutDistributorsCommissionAdmin 批量设置全部或当前筛选结果中的分销商默认分成比例。
func PutDistributorsCommissionAdmin(c *gin.Context) {
	var req putDistributorsCommissionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if req.DistributorCommissionBps == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请填写分销比例"})
		return
	}
	applyType := 0
	if req.ApplyType != nil {
		applyType = *req.ApplyType
	}
	keywordRunes := []rune(strings.TrimSpace(req.Keyword))
	if len(keywordRunes) > 120 {
		keywordRunes = keywordRunes[:120]
	}
	keyword := string(keywordRunes)
	switch strings.TrimSpace(req.Scope) {
	case "all":
		keyword = ""
		applyType = 0
	case "filtered":
		if keyword == "" && applyType == 0 {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "筛选条件不能为空"})
			return
		}
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的设置范围"})
		return
	}
	updated, err := model.SetDistributorsCommissionBps(
		*req.DistributorCommissionBps,
		keyword,
		applyType,
	)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"updated_count": updated},
	})
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
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 120 {
		keyword = keyword[:120]
	}
	items, total, err := model.ListAffInvitees(id, keyword, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// GetDistributorBindableUserAdmin 管理端查找可直接绑定到指定代理名下的普通用户。
func GetDistributorBindableUserAdmin(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("id"))
	if err != nil || distributorId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid distributor id"})
		return
	}
	dist, err := model.GetUserById(distributorId, false)
	if err != nil || dist == nil || !model.UserIsDistributor(dist) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不是代理"})
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 120 {
		keyword = keyword[:120]
	}
	item, err := model.SearchDistributorBindableUser(distributorId, keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": item})
}

type distributorInviteeBindAdminRequest struct {
	UserID int `json:"user_id"`
}

// PostDistributorInviteeBindAdmin 管理端直接建立代理与普通用户的邀请关系，无需目标用户确认。
func PostDistributorInviteeBindAdmin(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("id"))
	if err != nil || distributorId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid distributor id"})
		return
	}
	var req distributorInviteeBindAdminRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || req.UserID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if err := model.AdminBindDistributorInvitee(distributorId, req.UserID); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.RecordLog(req.UserID, model.LogTypeManage, fmt.Sprintf("管理员将用户绑定到代理ID %d 名下", distributorId))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "绑定成功"})
}

// GetDistributorInviteeProfitSharesAdmin 管理端查看某分销商下某一被邀请用户的利润分成消费流水（分页），包含已解绑用户的历史流水。
func GetDistributorInviteeProfitSharesAdmin(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("id"))
	if err != nil || distributorId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid distributor id"})
		return
	}
	inviteeId, err := strconv.Atoi(c.Param("invitee_id"))
	if err != nil || inviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid invitee id"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}
	dist, err := model.GetUserById(distributorId, false)
	if err != nil || !model.UserIsDistributor(dist) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不是分销商"})
		return
	}
	invitee, err := model.GetUserById(inviteeId, false)
	if err != nil || invitee == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	if invitee.InviterId != distributorId {
		hasProfitHistory, err := model.HasAffInviteProfitShareHistory(distributorId, inviteeId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		hasUnbindLog, err := model.HasDistributorInviteeUnbindLog(distributorId, inviteeId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		if !hasProfitHistory && !hasUnbindLog {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户不是此分销商邀请的下级"})
			return
		}
	}
	pageInfo := common.GetPageQuery(c)
	billingMode := strings.TrimSpace(c.Query("billing_mode"))
	items, total, err := model.ListAffInviteProfitShareLogs(distributorId, inviteeId, billingMode, false, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// PostDistributorInviteeUnbindAdmin 管理端解除某分销商与下级用户的绑定。
type distributorInviteeUnbindAdminRequest struct {
	Reason string `json:"reason"`
}

func PostDistributorInviteeUnbindAdmin(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("id"))
	if err != nil || distributorId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid distributor id"})
		return
	}
	inviteeId, err := strconv.Atoi(c.Param("invitee_id"))
	if err != nil || inviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid invitee id"})
		return
	}
	var req distributorInviteeUnbindAdminRequest
	if c.Request.Body != nil {
		body, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
			return
		}
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := common.Unmarshal(body, &req); err != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
				return
			}
		}
	}
	if err := model.UnbindDistributorInvitee(distributorId, inviteeId, c.GetInt("id"), req.Reason); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetDistributorInviteeUnbindLogsAdmin 管理端查看某分销商的下级解绑记录。
func GetDistributorInviteeUnbindLogsAdmin(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("id"))
	if err != nil || distributorId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid distributor id"})
		return
	}
	dist, err := model.GetUserById(distributorId, false)
	if err != nil || dist == nil || !model.UserIsDistributor(dist) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不是分销商"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListDistributorInviteeUnbindLogs(distributorId, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// PostDistributorWithdrawal 提交线下提现申请（暂扣 aff_quota）
func PostDistributorWithdrawal(c *gin.Context) {
	userId := c.GetInt("id")
	var req createDistributorWithdrawalRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	accountType, _, err := model.GetDistributorWithdrawAccountType(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	profileJSON, err := buildWithdrawalProfileJSON(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	quotaAmt := int(math.Round(req.QuotaAmount))
	if err := model.CreateDistributorWithdrawal(
		userId,
		accountType,
		strings.TrimSpace(req.RealName),
		strings.TrimSpace(req.BankName),
		strings.TrimSpace(req.BankAccount),
		profileJSON,
		"[]",
		strings.TrimSpace(req.WithdrawMonth),
		quotaAmt,
	); err != nil {
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
	rows, total, err := model.ListDistributorWithdrawals(userId, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, distributorWithdrawalToJSON(rows[i], ""))
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
	accountType, _ := strconv.Atoi(c.Query("account_type"))
	keyword := c.Query("keyword")
	rows, total, err := model.ListDistributorWithdrawalsAdmin(status, accountType, keyword, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, distributorWithdrawalToJSON(rows[i].DistributorWithdrawal, rows[i].Username))
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
	ApplyType         int      `json:"apply_type"`
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
	urlsJSON, err := common.Marshal(urls)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "资料序列化失败"})
		return
	}
	applyType := req.ApplyType
	if applyType == 0 {
		applyType = model.DistributorApplyTypePersonal
	}
	reviewerId := c.GetInt("id")
	if err := model.AdminUpsertDistributorApplicationByUser(userId, reviewerId, applyType, req.RealName, req.IdCardNo, string(urlsJSON), req.Contact); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
