package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetInvoiceProfile(c *gin.Context) {
	userID := c.GetInt("id")
	profile, err := model.GetInvoiceProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiSuccess(c, nil)
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profile)
}

func PutInvoiceProfile(c *gin.Context) {
	userID := c.GetInt("id")
	var req model.InvoiceProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	req.UserId = userID
	if req.Email == "" {
		user, err := model.GetUserById(userID, false)
		if err == nil && user != nil {
			req.Email = strings.TrimSpace(user.Email)
		}
	}
	if err := model.UpsertInvoiceProfile(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, req)
}

func GetInvoiceEligibleOrders(c *gin.Context) {
	userID := c.GetInt("id")
	keyword := c.Query("keyword")
	orders, err := model.ListInvoiceEligibleOrders(userID, keyword)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, orders)
}

func PostInvoiceRequest(c *gin.Context) {
	userID := c.GetInt("id")
	var body struct {
		Items  []model.InvoiceRequestItemInput `json:"items"`
		Remark string                          `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	profile, err := model.GetInvoiceProfileByUserID(userID)
	if err != nil {
		common.ApiError(c, fmt.Errorf("请先完善开票信息"))
		return
	}
	req, err := model.CreateInvoiceRequest(userID, body.Items, body.Remark, profile)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, req)
}

func GetInvoiceRequestsSelf(c *gin.Context) {
	userID := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	rows, total, err := model.ListInvoiceRequestsByUser(userID, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": rows,
		"total": total,
		"page":  pageInfo.GetPage(),
	})
}

func GetInvoiceRequestDetailSelf(c *gin.Context) {
	userID := c.GetInt("id")
	id, _ := strconv.Atoi(c.Param("id"))
	req, err := model.GetInvoiceRequestByID(id)
	if err != nil || req.UserId != userID {
		common.ApiError(c, fmt.Errorf("invoice request not found"))
		return
	}
	items, err := model.GetInvoiceRequestItems(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"request": req,
		"items":   items,
	})
}

func CancelInvoiceRequestSelf(c *gin.Context) {
	userID := c.GetInt("id")
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.CancelInvoiceRequest(userID, id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ListInvoiceRequestsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := c.Query("status")
	keyword := c.Query("keyword")
	rows, total, err := model.ListInvoiceRequestsAdminEnriched(status, keyword, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": rows,
		"total": total,
		"page":  pageInfo.GetPage(),
	})
}

func IssueInvoiceRequestAdmin(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		InvoiceCode string `json:"invoice_code"`
		InvoiceUrl  string `json:"invoice_url"`
		AdminNote   string `json:"admin_note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	body.InvoiceUrl = strings.TrimSpace(body.InvoiceUrl)
	if body.InvoiceUrl == "" {
		common.ApiError(c, fmt.Errorf("请先上传电子发票文件"))
		return
	}
	req, err := model.GetInvoiceRequestByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.IssueInvoiceRequest(id, body.InvoiceCode, body.InvoiceUrl, body.AdminNote); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := sendInvoiceIssuedNotification(req, body.InvoiceCode, body.InvoiceUrl); err != nil {
		common.SysLog("invoice issued email failed: " + err.Error())
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func MarkInvoiceRequestProcessingAdmin(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.MarkInvoiceRequestProcessing(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func UploadInvoiceFileAdmin(c *gin.Context) {
	adminID := c.GetInt("id")
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiError(c, fmt.Errorf("请选择文件字段 file"))
		return
	}
	publicURL, err := service.UploadInvoiceFile(file, adminID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"url": publicURL})
}

func sendInvoiceIssuedNotification(req *model.InvoiceRequest, invoiceCode, invoiceURL string) error {
	if req == nil {
		return fmt.Errorf("invoice request is nil")
	}
	receiver := invoiceReceiverEmail(req)
	if receiver == "" {
		return fmt.Errorf("收票邮箱未配置")
	}
	subject := "您的电子发票已开具"
	content := fmt.Sprintf("<p>您好，</p><p>发票申请单号：%s</p><p>开票金额：%.2f</p>", req.RequestNo, req.TotalAmount)
	if invoiceCode != "" {
		content += fmt.Sprintf("<p>发票号码：%s</p>", invoiceCode)
	}
	if invoiceURL != "" {
		content += fmt.Sprintf("<p><a href=\"%s\">点击下载电子发票</a></p>", invoiceURL)
	}

	var attachments []common.EmailAttachment
	if localPath, ok := service.ResolveLocalUploadFilePath(invoiceURL); ok {
		data, err := os.ReadFile(localPath)
		if err != nil {
			return err
		}
		attachments = append(attachments, common.EmailAttachment{
			Filename:    fmt.Sprintf("invoice-%s.pdf", req.RequestNo),
			ContentType: "application/pdf",
			Data:        data,
		})
		content += "<p>电子发票亦随邮件附件发送，请注意查收。</p>"
	} else if invoiceURL != "" {
		content += "<p>请通过上方链接下载电子发票。</p>"
	}
	return common.SendEmailWithAttachments(subject, receiver, content, attachments)
}

func invoiceReceiverEmail(req *model.InvoiceRequest) string {
	if req == nil {
		return ""
	}
	if profile := parseInvoiceProfileSnapshot(req.ProfileSnapshot); profile != nil {
		if email := strings.TrimSpace(profile.Email); email != "" {
			return email
		}
	}
	user, err := model.GetUserById(req.UserId, false)
	if err != nil || user == nil {
		return ""
	}
	return strings.TrimSpace(user.Email)
}

func parseInvoiceProfileSnapshot(raw string) *model.InvoiceProfile {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var profile model.InvoiceProfile
	if err := common.UnmarshalJsonStr(raw, &profile); err != nil {
		return nil
	}
	return &profile
}

func GetInvoiceBalanceSummarySelf(c *gin.Context) {
	userID := c.GetInt("id")
	summary, err := model.GetInvoiceBalanceSummary(userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func AdminGrantGiftQuota(c *gin.Context) {
	var body struct {
		UserID int    `json:"user_id"`
		Quota  int    `json:"quota"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if body.UserID <= 0 || body.Quota <= 0 {
		common.ApiError(c, fmt.Errorf("用户与赠送额度必填"))
		return
	}
	if err := model.GrantUserGiftQuota(body.UserID, body.Quota); err != nil {
		common.ApiError(c, err)
		return
	}
	remark := strings.TrimSpace(body.Remark)
	logMsg := fmt.Sprintf("管理员赠送额度 %s", logger.LogQuota(body.Quota))
	if remark != "" {
		logMsg += "，备注: " + remark
	}
	model.RecordLog(body.UserID, model.LogTypeManage, logMsg)
	common.ApiSuccess(c, gin.H{"user_id": body.UserID, "quota": body.Quota})
}

func AdminCorporateTopUp(c *gin.Context) {
	adminID := c.GetInt("id")
	var body struct {
		UserID    int     `json:"user_id"`
		Money     float64 `json:"money"`
		Quota     int     `json:"quota"`
		Reference string  `json:"reference"`
		Remark    string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if body.UserID <= 0 || body.Money <= 0 || body.Quota <= 0 {
		common.ApiError(c, fmt.Errorf("用户、金额与额度必填"))
		return
	}
	topUp, err := model.CreateCorporateTopUp(body.UserID, body.Money, body.Quota, body.Reference, body.Remark, adminID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, topUp)
}

func RejectInvoiceRequestAdmin(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		AdminNote string `json:"admin_note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RejectInvoiceRequest(id, body.AdminNote); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
