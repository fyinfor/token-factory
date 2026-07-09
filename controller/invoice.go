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

func ListInvoiceRequestsAdmin(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := c.Query("status")
	rows, total, err := model.ListInvoiceRequestsAdminEnriched(status, pageInfo)
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
	req, err := model.GetInvoiceRequestByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.IssueInvoiceRequest(id, body.InvoiceCode, body.InvoiceUrl, body.AdminNote); err != nil {
		common.ApiError(c, err)
		return
	}
	user, uerr := model.GetUserById(req.UserId, false)
	if uerr == nil && user != nil && strings.TrimSpace(user.Email) != "" {
		subject := "您的电子发票已开具"
		content := fmt.Sprintf("<p>您好，</p><p>发票申请单号：%s</p><p>开票金额：%.2f</p>", req.RequestNo, req.TotalAmount)
		if body.InvoiceCode != "" {
			content += fmt.Sprintf("<p>发票号码：%s</p>", body.InvoiceCode)
		}
		if body.InvoiceUrl != "" {
			content += fmt.Sprintf("<p><a href=\"%s\">点击下载电子发票</a></p>", body.InvoiceUrl)
		}
		_ = common.SendEmail(subject, user.Email, content)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
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
