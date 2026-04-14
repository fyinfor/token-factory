package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// SupplierApplicationSubmitRequest 供应商提交申请请求体。
type SupplierApplicationSubmitRequest struct {
	CompanyName        string `json:"company_name"`
	CreditCode         string `json:"credit_code"`
	BusinessLicenseURL string `json:"business_license_url"`
	LegalRepresentative string `json:"legal_representative"`
	CompanySize        string `json:"company_size"`
	ContactName        string `json:"contact_name"`
	ContactMobile      string `json:"contact_mobile"`
	ContactWechat      string `json:"contact_wechat"`
}

// SupplierApplicationReviewRequest 管理员审核请求体。
type SupplierApplicationReviewRequest struct {
	Status int    `json:"status"`
	Reason string `json:"reason"`
}

// readOptionalStatusQuery 读取可选 status 查询参数。
func readOptionalStatusQuery(c *gin.Context) (*int, error) {
	statusRaw := strings.TrimSpace(c.Query("status"))
	if statusRaw == "" {
		return nil, nil
	}
	status, err := strconv.Atoi(statusRaw)
	if err != nil {
		return nil, err
	}
	if status < model.SupplierApplicationStatusPending || status > model.SupplierApplicationStatusRejected {
		return nil, errors.New("invalid status")
	}
	return &status, nil
}

// SubmitSupplierApplication godoc
// @Summary 提交供应商入驻申请
// @Description 普通用户提交供应商申请，提交后生成管理员待审核站内消息
// @Tags Supplier
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body SupplierApplicationSubmitRequest true "申请信息"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application [post]
func SubmitSupplierApplication(c *gin.Context) {
	var req SupplierApplicationSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.CreditCode = strings.TrimSpace(req.CreditCode)
	req.BusinessLicenseURL = strings.TrimSpace(req.BusinessLicenseURL)
	req.LegalRepresentative = strings.TrimSpace(req.LegalRepresentative)
	req.CompanySize = strings.TrimSpace(req.CompanySize)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.ContactMobile = strings.TrimSpace(req.ContactMobile)
	req.ContactWechat = strings.TrimSpace(req.ContactWechat)
	if req.CompanyName == "" || req.CreditCode == "" || req.BusinessLicenseURL == "" ||
		req.LegalRepresentative == "" || req.ContactName == "" || req.ContactMobile == "" || req.ContactWechat == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请填写完整的必填字段"})
		return
	}
	if len(req.CreditCode) != 18 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "统一社会信用代码需为18位"})
		return
	}

	app := &model.SupplierApplication{
		ApplicantUserID:    c.GetInt("id"),
		CompanyName:        req.CompanyName,
		CreditCode:         req.CreditCode,
		BusinessLicenseURL: req.BusinessLicenseURL,
		LegalRepresentative: req.LegalRepresentative,
		CompanySize:        req.CompanySize,
		ContactName:        req.ContactName,
		ContactMobile:      req.ContactMobile,
		ContactWechat:      req.ContactWechat,
		Status:             model.SupplierApplicationStatusPending,
	}
	if err := model.CreateSupplierApplication(app); err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.CreateSupplierApplicationAudit(&model.SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: app.ApplicantUserID,
		Action:         model.SupplierApplicationAuditActionSubmit,
		FromStatus:     model.SupplierApplicationStatusPending,
		ToStatus:       model.SupplierApplicationStatusPending,
		Reason:         "",
	})
	_ = model.CreateUserMessage(&model.UserMessage{
		ReceiverUserID:  0,
		ReceiverMinRole: common.RoleAdminUser,
		Type:            model.UserMessageTypeSupplierSubmitted,
		Title:           "供应商入驻待审核",
		Content:         fmt.Sprintf("收到新的供应商申请：%s（统一社会信用代码：%s）", app.CompanyName, app.CreditCode),
		BizType:         model.UserMessageBizTypeSupplierApplication,
		BizID:           app.ID,
	})
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// ListMySupplierApplications godoc
// @Summary 查询当前用户提交的供应商申请
// @Tags Supplier
// @Produce json
// @Security ApiKeyAuth
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query int false "状态：0待审核 1审核通过 2审核驳回"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/supplier/application/self [get]
func ListMySupplierApplications(c *gin.Context) {
	status, err := readOptionalStatusQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的status参数"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListSupplierApplicationsByApplicant(c.GetInt("id"), status, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// AdminListSupplierApplications godoc
// @Summary 管理员分页查询供应商申请
// @Tags SupplierAdmin
// @Produce json
// @Security ApiKeyAuth
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query int false "状态：0待审核 1审核通过 2审核驳回"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/supplier/application [get]
func AdminListSupplierApplications(c *gin.Context) {
	status, err := readOptionalStatusQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的status参数"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListSupplierApplications(status, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// AdminReviewSupplierApplication godoc
// @Summary 管理员审核供应商申请
// @Description 任一管理员可审核一次，仅待审核状态允许处理
// @Tags SupplierAdmin
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "申请ID"
// @Param request body SupplierApplicationReviewRequest true "审核信息"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application/{id}/review [post]
func AdminReviewSupplierApplication(c *gin.Context) {
	applicationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的申请ID"})
		return
	}
	var req SupplierApplicationReviewRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if req.Status != model.SupplierApplicationStatusApproved && req.Status != model.SupplierApplicationStatusRejected {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "审核状态仅支持通过或驳回"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Status == model.SupplierApplicationStatusRejected && req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "驳回时请填写原因"})
		return
	}

	app, err := model.ReviewSupplierApplication(applicationID, c.GetInt("id"), req.Status, req.Reason)
	if err != nil {
		if errors.Is(err, model.ErrSupplierApplicationAlreadyReviewed) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该申请已被其他管理员处理"})
			return
		}
		common.ApiError(c, err)
		return
	}

	msgType := model.UserMessageTypeSupplierApproved
	msgTitle := "供应商入驻审核通过"
	msgContent := fmt.Sprintf("你的供应商申请“%s”已审核通过。", app.CompanyName)
	if app.Status == model.SupplierApplicationStatusRejected {
		msgType = model.UserMessageTypeSupplierRejected
		msgTitle = "供应商入驻审核驳回"
		msgContent = fmt.Sprintf("你的供应商申请“%s”已驳回，原因：%s", app.CompanyName, req.Reason)
	}
	_ = model.CreateUserMessage(&model.UserMessage{
		ReceiverUserID:  app.ApplicantUserID,
		ReceiverMinRole: 0,
		Type:            msgType,
		Title:           msgTitle,
		Content:         msgContent,
		BizType:         model.UserMessageBizTypeSupplierApplication,
		BizID:           app.ID,
	})
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// ListMyMessages godoc
// @Summary 查询当前用户站内消息
// @Tags Message
// @Produce json
// @Security ApiKeyAuth
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/messages/self [get]
func ListMyMessages(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userID := c.GetInt("id")
	role := c.GetInt("role")
	items, total, err := model.ListUserMessagesForUser(userID, role, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// MarkMyMessageRead godoc
// @Summary 标记当前用户消息为已读
// @Tags Message
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "消息ID"
// @Success 200 {object} map[string]interface{} "success + data{updated}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/messages/{id}/read [post]
func MarkMyMessageRead(c *gin.Context) {
	messageID, err := strconv.Atoi(c.Param("id"))
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的消息ID"})
		return
	}
	ok, err := model.MarkUserMessageAsRead(messageID, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"updated": ok})
}

// GetMyUnreadMessageCount godoc
// @Summary 获取当前用户未读站内消息数量
// @Tags Message
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "success + data{unread_count}"
// @Router /user/messages/unread_count [get]
func GetMyUnreadMessageCount(c *gin.Context) {
	total, err := model.CountUnreadUserMessages(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"unread_count": total})
}
