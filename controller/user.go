package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest 用户注册请求体：关闭短信注册时邮箱必填；开启短信注册时邮箱与手机号二选一（至少填其一）；开启邮箱验证且填写了邮箱时需验证码；开启短信且填写了手机号时需短信验证码。邮箱/手机号占用仅与未注销用户冲突。
type RegisterRequest struct {
	Username         string `json:"username" validate:"required,max=20"`
	Password         string `json:"password" validate:"required,min=8,max=20"`
	Email            string `json:"email" validate:"omitempty,email,max=50"`
	VerificationCode string `json:"verification_code"`
	AffCode          string `json:"aff_code"`
	Phone            string `json:"phone"`
	SMSCode          string `json:"sms_verification_code"`
}

func ApplyStudent(c *gin.Context) {
	id := c.GetInt("id")
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Role >= common.RoleAdminUser {
		common.ApiErrorMsg(c, "管理员账号无需申请学员身份")
		return
	}
	if user.IsStudent == 1 && user.StudentStatus == common.StudentStatusApproved {
		common.ApiErrorMsg(c, "你已经是学员")
		return
	}
	if user.StudentStatus == common.StudentStatusPending {
		common.ApiErrorMsg(c, "学员申请正在审批中")
		return
	}
	now := time.Now()
	user.IsStudent = 0
	user.StudentStatus = common.StudentStatusPending
	user.StudentApplied = &now
	user.StudentApprovedAt = nil
	user.StudentApprovedBy = 0
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "学员申请已提交，请等待管理员审批",
	})
}

func Login(c *gin.Context) {
	if !common.PasswordLoginEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserPasswordLoginDisabled)
		return
	}
	var loginRequest LoginRequest
	err := json.NewDecoder(c.Request.Body).Decode(&loginRequest)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	username := loginRequest.Username
	password := loginRequest.Password
	if username == "" || password == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	user := model.User{
		Username: username,
		Password: password,
	}
	err = user.ValidateAndFill()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// 检查是否启用2FA
	if model.IsTwoFAEnabled(user.Id) {
		// 设置pending session，等待2FA验证
		session := sessions.Default(c)
		session.Set("pending_username", user.Username)
		session.Set("pending_user_id", user.Id)
		err := session.Save()
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgUserSessionSaveFailed)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": i18n.T(c, i18n.MsgUserRequire2FA),
			"success": true,
			"data": map[string]interface{}{
				"require_2fa": true,
			},
		})
		return
	}

	setupLogin(&user, c)
}

// setup session & cookies and then return user info
func setupLogin(user *model.User, c *gin.Context) {
	session := sessions.Default(c)
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	session.Set("group", user.Group)
	err := session.Save()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserSessionSaveFailed)
		return
	}
	model.TouchUserLastLogin(user.Id)
	requireAdminInitialSetup := user.CreatedBy == common.UserCreatedByAdmin && !user.AdminInitialSetupCompleted
	adminSetupPhoneRequired := requireAdminInitialSetup && strings.TrimSpace(user.Phone) == ""
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
		"data": map[string]any{
			"id":                          user.Id,
			"username":                    user.Username,
			"display_name":                user.DisplayName,
			"role":                        user.Role,
			"status":                      user.Status,
			"group":                       user.Group,
			"is_distributor":              user.IsDistributor,
			"require_admin_initial_setup": requireAdminInitialSetup,
			"admin_setup_phone_required":  adminSetupPhoneRequired,
		},
	})
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
	})
}

// Register 处理用户名密码注册：未开启短信时邮箱必填；开启短信时邮箱与手机至少填其一；短信与邮箱验证码仅在对应字段填写时校验；邮箱与手机号是否与已占用冲突仅检查未软删用户。
func Register(c *gin.Context) {
	if !common.RegisterEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserRegisterDisabled)
		return
	}
	if !common.PasswordRegisterEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserPasswordRegisterDisabled)
		return
	}
	var req RegisterRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = common.NormalizePhone(req.Phone)
	req.SMSCode = strings.TrimSpace(req.SMSCode)
	if !common.SMSVerificationEnabled && req.Email == "" {
		common.ApiErrorI18n(c, i18n.MsgUserEmailEmpty)
		return
	}
	if err := common.Validate.Struct(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	if common.SMSVerificationEnabled {
		if req.Email == "" && req.Phone == "" {
			common.ApiErrorI18n(c, i18n.MsgUserRegisterEmailOrPhoneRequired)
			return
		}
		if req.Phone != "" {
			if !common.ValidateMainlandChinaPhone(req.Phone) {
				common.ApiError(c, fmt.Errorf("手机号格式无效，请输入 11 位中国大陆手机号"))
				return
			}
			if common.IsSMSPhoneBlacklisted(req.Phone) {
				common.ApiError(c, fmt.Errorf("该手机号已被加入短信黑名单"))
				return
			}
			if len(strings.TrimSpace(req.SMSCode)) != 6 {
				common.ApiError(c, fmt.Errorf("请输入 6 位短信验证码"))
				return
			}
			if !common.VerifyAndConsumeSMSCode(req.Phone, req.SMSCode) {
				common.ApiError(c, fmt.Errorf("短信验证码错误或已过期"))
				return
			}
			if model.IsPhoneAlreadyTaken(req.Phone) {
				common.ApiError(c, fmt.Errorf("手机号已被占用"))
				return
			}
		} else {
			req.Phone = ""
			req.SMSCode = ""
		}
	} else {
		req.Phone = ""
		req.SMSCode = ""
	}
	if common.EmailVerificationEnabled && req.Email != "" {
		if req.VerificationCode == "" {
			common.ApiErrorI18n(c, i18n.MsgUserEmailVerificationRequired)
			return
		}
		if !common.VerifyCodeWithKey(req.Email, req.VerificationCode, common.EmailVerificationPurpose) {
			common.ApiErrorI18n(c, i18n.MsgUserVerificationCodeError)
			return
		}
	}
	nameTaken, err := model.IsUsernameTakenUnscoped(req.Username)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		common.SysLog(fmt.Sprintf("IsUsernameTakenUnscoped error: %v", err))
		return
	}
	if nameTaken {
		common.ApiErrorI18n(c, i18n.MsgUserUsernameTaken)
		return
	}
	if req.Email != "" {
		emailTaken, err := model.IsEmailTakenByActiveUser(req.Email)
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			common.SysLog(fmt.Sprintf("IsEmailTakenByActiveUser error: %v", err))
			return
		}
		if emailTaken {
			common.ApiErrorI18n(c, i18n.MsgUserEmailTaken)
			return
		}
	}
	affCode := req.AffCode // this code is the inviter's code, not the user's own code
	inviterId, _ := model.GetUserIdByAffCode(affCode)
	cleanUser := model.User{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.Username,
		InviterId:   inviterId,
		Role:        common.RoleCommonUser, // 明确设置角色为普通用户
		Phone:       req.Phone,
		Email:       req.Email,
	}
	if err := cleanUser.Insert(inviterId); err != nil {
		common.ApiError(c, err)
		return
	}

	// 获取插入后的用户ID
	var insertedUser model.User
	if err := model.DB.Where("username = ?", cleanUser.Username).First(&insertedUser).Error; err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserRegisterFailed)
		return
	}
	// 生成默认令牌
	if constant.GenerateDefaultToken {
		key, err := common.GenerateKey()
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgUserDefaultTokenFailed)
			common.SysLog("failed to generate token key: " + err.Error())
			return
		}
		// 生成默认令牌
		token := model.Token{
			UserId:             insertedUser.Id, // 使用插入后的用户ID
			Name:               cleanUser.Username + "的初始令牌",
			Key:                key,
			CreatedTime:        common.GetTimestamp(),
			AccessedTime:       common.GetTimestamp(),
			ExpiredTime:        -1,     // 永不过期
			RemainQuota:        500000, // 示例额度
			UnlimitedQuota:     true,
			ModelLimitsEnabled: false,
		}
		if setting.DefaultUseAutoGroup {
			token.Group = "auto"
		}
		if err := token.Insert(); err != nil {
			common.ApiErrorI18n(c, i18n.MsgCreateDefaultTokenErr)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func GetAllUsers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	studentView := strings.TrimSpace(c.Query("student_view"))
	users, total, err := model.GetAllUsers(pageInfo, studentView)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)

	common.ApiSuccess(c, pageInfo)
	return
}

func SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	group := c.Query("group")
	studentView := strings.TrimSpace(c.Query("student_view"))
	pageInfo := common.GetPageQuery(c)
	users, total, err := model.SearchUsers(keyword, group, studentView, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	myRole := c.GetInt("role")
	if myRole <= user.Role && myRole != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionSameLevel)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
	return
}

// AdminCheckPhoneAvailable 管理端校验手机号是否未被他人占用：新建时不传 exclude_id；编辑用户时传 exclude_id 为当前用户 ID。
func AdminCheckPhoneAvailable(c *gin.Context) {
	phone := c.Query("phone")
	excludeStr := strings.TrimSpace(c.Query("exclude_id"))
	excludeID := 0
	if excludeStr != "" {
		var convErr error
		excludeID, convErr = strconv.Atoi(excludeStr)
		if convErr != nil || excludeID < 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
	}
	normalized := common.NormalizePhone(phone)
	if normalized == "" {
		common.ApiSuccess(c, gin.H{"available": true})
		return
	}
	if !common.ValidateMainlandChinaPhone(normalized) {
		common.ApiSuccess(c, gin.H{"available": true})
		return
	}
	var taken bool
	if excludeID == 0 {
		taken = model.IsPhoneAlreadyTaken(normalized)
	} else {
		taken = model.IsPhoneTakenByOtherUser(normalized, excludeID)
	}
	common.ApiSuccess(c, gin.H{"available": !taken})
}

// UserSelfCheckPhoneAvailable 当前登录用户校验欲使用的手机号是否与他人冲突（exclude 固定为本人，不可伪造）。
func UserSelfCheckPhoneAvailable(c *gin.Context) {
	id := c.GetInt("id")
	if id <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	phone := c.Query("phone")
	normalized := common.NormalizePhone(phone)
	if normalized == "" {
		common.ApiSuccess(c, gin.H{"available": true})
		return
	}
	if !common.ValidateMainlandChinaPhone(normalized) {
		common.ApiSuccess(c, gin.H{"available": true})
		return
	}
	taken := model.IsPhoneTakenByOtherUser(normalized, id)
	common.ApiSuccess(c, gin.H{"available": !taken})
}

// isPhoneUniqueConstraintError 判断数据库错误是否为手机号唯一约束冲突。
func isPhoneUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "duplicate") && !strings.Contains(msg, "unique constraint") {
		return false
	}
	return strings.Contains(msg, "phone")
}

// GenerateAccessToken godoc
// @Summary 生成当前用户 AccessToken
// @Description 生成并返回当前登录用户的 access_token，用于在 Authorization 请求头中进行接口鉴权
// @Tags 用户
// @Produce json
// @Security ApiKeyAuth
// @Security ApiUserID
// @Success 200 {object} map[string]interface{} "success + data{access_token}"
// @Router /user/token [get]
func GenerateAccessToken(c *gin.Context) {
	id := c.GetInt("id")
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// get rand int 28-32
	randI := common.GetRandomInt(4)
	key, err := common.GenerateRandomKey(29 + randI)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgGenerateFailed)
		common.SysLog("failed to generate key: " + err.Error())
		return
	}
	user.SetAccessToken(key)

	if model.DB.Where("access_token = ?", user.AccessToken).First(user).RowsAffected != 0 {
		common.ApiErrorI18n(c, i18n.MsgUuidDuplicate)
		return
	}

	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user.AccessToken,
	})
	return
}

type TransferAffQuotaRequest struct {
	Quota int `json:"quota" binding:"required"`
}

func TransferAffQuota(c *gin.Context) {
	id := c.GetInt("id")
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !model.UserIsDistributor(user) {
		common.ApiErrorMsg(c, "仅分销商可划转邀请收益")
		return
	}
	tran := TransferAffQuotaRequest{}
	if err := c.ShouldBindJSON(&tran); err != nil {
		common.ApiError(c, err)
		return
	}
	err = user.TransferAffQuotaToQuota(tran.Quota)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserTransferFailed, map[string]any{"Error": err.Error()})
		return
	}
	common.ApiSuccessI18n(c, i18n.MsgUserTransferSuccess, nil)
}

func GetAffCode(c *gin.Context) {
	id := c.GetInt("id")
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !model.UserIsDistributor(user) {
		common.ApiErrorMsg(c, "仅分销商可使用邀请链接")
		return
	}
	if user.AffCode == "" {
		user.EnsureAffCode()
		if err := user.Update(false); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user.AffCode,
	})
	return
}

func GetSelf(c *gin.Context) {
	id := c.GetInt("id")
	userRole := c.GetInt("role")
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Hide admin remarks: set to empty to trigger omitempty tag, ensuring the remark field is not included in JSON returned to regular users
	user.Remark = ""

	// 计算用户权限信息
	permissions := calculateUserPermissions(userRole)

	// 获取用户设置并提取sidebar_modules
	userSetting := user.GetSetting()

	requireAdminInitialSetup := user.CreatedBy == common.UserCreatedByAdmin && !user.AdminInitialSetupCompleted
	adminSetupPhoneRequired := requireAdminInitialSetup && strings.TrimSpace(user.Phone) == ""

	// 构建响应数据，包含用户信息和权限
	responseData := map[string]interface{}{
		"id":                          user.Id,
		"username":                    user.Username,
		"display_name":                user.DisplayName,
		"role":                        user.Role,
		"status":                      user.Status,
		"email":                       user.Email,
		"phone":                       user.Phone,
		"github_id":                   user.GitHubId,
		"discord_id":                  user.DiscordId,
		"oidc_id":                     user.OidcId,
		"wechat_id":                   user.WeChatId,
		"telegram_id":                 user.TelegramId,
		"group":                       user.Group,
		"quota":                       user.Quota,
		"used_quota":                  user.UsedQuota,
		"request_count":               user.RequestCount,
		"aff_code":                    user.AffCode,
		"aff_count":                   user.AffCount,
		"aff_quota":                   user.AffQuota,
		"aff_history_quota":           user.AffHistoryQuota,
		"distributor_commission_bps":  user.DistributorCommissionBps,
		"inviter_id":                  user.InviterId,
		"linux_do_id":                 user.LinuxDOId,
		"setting":                     user.Setting,
		"stripe_customer":             user.StripeCustomer,
		"supplier_id":                 user.SupplierID,
		"is_distributor":              user.IsDistributor,
		"is_student":                  user.IsStudent,
		"student_status":              user.StudentStatus,
		"student_applied_at":          user.StudentApplied,
		"student_approved_at":         user.StudentApprovedAt,
		"student_approved_by":         user.StudentApprovedBy,
		"sidebar_modules":             userSetting.SidebarModules, // 正确提取sidebar_modules字段
		"permissions":                 permissions,                // 新增权限字段
		"require_admin_initial_setup": requireAdminInitialSetup,
		"admin_setup_phone_required":  adminSetupPhoneRequired,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    responseData,
	})
	return
}

// AdminInitialSetupRequest 管理员代建用户首次登录补全信息（改密；未预留手机时须绑定手机号）。
type AdminInitialSetupRequest struct {
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
	Phone           string `json:"phone"`
}

// CompleteAdminInitialSetup 管理员创建的账号首次登录后提交：修改密码；若创建时未填手机号则必须绑定且不可与他人重复。
func CompleteAdminInitialSetup(c *gin.Context) {
	id := c.GetInt("id")
	var req AdminInitialSetupRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	req.ConfirmPassword = strings.TrimSpace(req.ConfirmPassword)
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 20 {
		common.ApiErrorMsg(c, "新密码长度须在 8～20 位之间")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		common.ApiErrorMsg(c, "两次输入的密码不一致")
		return
	}
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.CreatedBy != common.UserCreatedByAdmin || user.AdminInitialSetupCompleted {
		common.ApiErrorMsg(c, "当前账号无需执行此操作")
		return
	}
	var phoneNorm string
	if strings.TrimSpace(user.Phone) == "" {
		var valErr error
		phoneNorm, valErr = model.NormalizeAndValidateAdminUserPhone(req.Phone, user.Id)
		if valErr != nil {
			common.ApiError(c, valErr)
			return
		}
	}
	user.Password = req.NewPassword
	user.AdminInitialSetupCompleted = true
	if phoneNorm != "" {
		user.Phone = phoneNorm
	}
	if err := user.Update(true); err != nil {
		if isPhoneUniqueConstraintError(err) {
			common.ApiErrorMsg(c, "手机号已被占用")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccessI18n(c, i18n.MsgUpdateSuccess, nil)
}

// 计算用户权限的辅助函数
func calculateUserPermissions(userRole int) map[string]interface{} {
	permissions := map[string]interface{}{}

	// 根据用户角色计算权限
	if userRole == common.RoleRootUser {
		// 超级管理员不需要边栏设置功能
		permissions["sidebar_settings"] = false
		permissions["sidebar_modules"] = map[string]interface{}{}
	} else if userRole == common.RoleAdminUser {
		// 管理员可以设置边栏，但不包含系统设置功能
		permissions["sidebar_settings"] = true
		permissions["sidebar_modules"] = map[string]interface{}{
			"admin": map[string]interface{}{
				"setting": false, // 管理员不能访问系统设置
			},
		}
	} else {
		// 普通用户、分销商：仅个人功能，不含管理后台
		permissions["sidebar_settings"] = true
		permissions["sidebar_modules"] = map[string]interface{}{
			"admin": false,
		}
	}

	return permissions
}

// 根据用户角色生成默认的边栏配置
func generateDefaultSidebarConfig(userRole int) string {
	defaultConfig := map[string]interface{}{}

	// 聊天区域 - 所有用户都可以访问
	defaultConfig["chat"] = map[string]interface{}{
		"enabled":    true,
		"playground": true,
		"chat":       true,
	}

	// 控制台区域 - 所有用户都可以访问
	defaultConfig["console"] = map[string]interface{}{
		"enabled":    true,
		"detail":     true,
		"token":      true,
		"log":        true,
		"midjourney": true,
		"task":       true,
	}

	// 个人中心区域 - 所有用户都可以访问
	defaultConfig["personal"] = map[string]interface{}{
		"enabled":  true,
		"topup":    true,
		"personal": true,
	}

	// 管理员区域 - 根据角色决定
	if userRole == common.RoleAdminUser {
		// 管理员可以访问管理员区域，但不能访问系统设置
		defaultConfig["admin"] = map[string]interface{}{
			"enabled":    true,
			"channel":    true,
			"models":     true,
			"redemption": true,
			"user":       true,
			"setting":    false, // 管理员不能访问系统设置
		}
	} else if userRole == common.RoleRootUser {
		// 超级管理员可以访问所有功能
		defaultConfig["admin"] = map[string]interface{}{
			"enabled":    true,
			"channel":    true,
			"models":     true,
			"redemption": true,
			"user":       true,
			"setting":    true,
		}
	}
	// 普通用户不包含admin区域

	// 转换为JSON字符串
	configBytes, err := json.Marshal(defaultConfig)
	if err != nil {
		common.SysLog("生成默认边栏配置失败: " + err.Error())
		return ""
	}

	return string(configBytes)
}

func GetUserModels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		id = c.GetInt("id")
	}
	user, err := model.GetUserCache(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groups := service.GetUserUsableGroups(user.Group)
	var models []string
	for group := range groups {
		for _, g := range model.GetGroupEnabledModels(group) {
			if !common.StringsContains(models, g) {
				models = append(models, g)
			}
		}
	}
	// scene=playground 时返回结构化模型列表：
	// - 展示口径与 /pricing 完全一致：模型必须已配置定价（ratio_setting.ModelHasConfiguredPricing），
	//   且至少存在一个 (模型, 可见渠道) 在 model_test_results 中满足
	//   ManualDisplayResponseTime>0 或 (LastTestSuccess && LastResponseTime>0)；
	//   不再使用 model_test_results 全表 last_test_success=1 的模糊名字匹配（口径偏宽且与定价页不一致）。
	// - 在此基础上再叠加「该模型在用户可用分组下的 abilities 已 enabled」的用户视角过滤；
	//   也即同时通过 GetGroupEnabledModels 与 CollectPricingShowableModelNames 两层门禁。
	// - vendor: 模型类型；类型选项仅由「通过判定后的」items 中出现的 vendor_id 推导。
	// - tested_success 在返回项中恒为 true（因已按 pricing 同源条件过滤）。
	if c.Query("scene") == "playground" {
		type playgroundChannelOption struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			ChannelNo    string `json:"channel_no,omitempty"`
			RouteSlug    string `json:"route_slug,omitempty"`
			SupplierType string `json:"supplier_type,omitempty"`
		}
		type playgroundModelItem struct {
			ModelName      string                    `json:"model_name"`
			VendorID       int                       `json:"vendor_id"`
			Vendor         string                    `json:"vendor"`
			Tags           string                    `json:"tags"`
			TestedSuccess  bool                      `json:"tested_success"`
			ChannelOptions []playgroundChannelOption `json:"channel_options,omitempty"`
		}
		modelRows := make([]struct {
			ModelName string `gorm:"column:model_name"`
			VendorID  int    `gorm:"column:vendor_id"`
			Tags      string `gorm:"column:tags"`
			NameRule  int    `gorm:"column:name_rule"`
		}, 0)
		modelVendorIDByName := make(map[string]int, len(models))
		modelTagsByName := make(map[string]string, len(models))
		if len(models) > 0 {
			// 与“模型广场”一致：按模型元数据(model_meta)中的规则（精确/前缀/后缀/包含）做归属映射
			if err := model.DB.Model(&model.Model{}).
				Select("model_name", "vendor_id", "tags", "name_rule").
				Where("status = ?", 1).
				Find(&modelRows).Error; err != nil {
				common.ApiError(c, err)
				return
			}
			rulePriority := func(rule int) int {
				switch rule {
				case model.NameRuleExact:
					return 0
				case model.NameRulePrefix:
					return 1
				case model.NameRuleSuffix:
					return 2
				case model.NameRuleContains:
					return 3
				default:
					return 9
				}
			}
			matchRule := func(pattern, target string, rule int) bool {
				switch rule {
				case model.NameRuleExact:
					return target == pattern
				case model.NameRulePrefix:
					return strings.HasPrefix(target, pattern)
				case model.NameRuleSuffix:
					return strings.HasSuffix(target, pattern)
				case model.NameRuleContains:
					return strings.Contains(target, pattern)
				default:
					return false
				}
			}
			for _, targetModelName := range models {
				bestIdx := -1
				for i := range modelRows {
					row := modelRows[i]
					if !matchRule(row.ModelName, targetModelName, row.NameRule) {
						continue
					}
					if bestIdx < 0 {
						bestIdx = i
						continue
					}
					cur := modelRows[bestIdx]
					curPriority := rulePriority(cur.NameRule)
					newPriority := rulePriority(row.NameRule)
					if newPriority < curPriority {
						bestIdx = i
						continue
					}
					if newPriority == curPriority && len(row.ModelName) > len(cur.ModelName) {
						bestIdx = i
					}
				}
				if bestIdx >= 0 {
					row := modelRows[bestIdx]
					modelVendorIDByName[targetModelName] = row.VendorID
					modelTagsByName[targetModelName] = strings.TrimSpace(row.Tags)
				}
			}
		}
		// 先列出分组内每个已启用模型名 + 元数据 vendor；再用 CollectPricingShowableModelNames 与 /pricing 同口径过滤，只返回与定价页一致的可展示模型，并据此推导「模型类型」选项
		playgroundNameRows := make([]struct {
			ModelName string
			VendorID  int
		}, 0, len(models))
		for _, name := range models {
			vid := 0
			if v, ok := modelVendorIDByName[name]; ok {
				vid = v
			}
			playgroundNameRows = append(playgroundNameRows, struct {
				ModelName string
				VendorID  int
			}{ModelName: name, VendorID: vid})
		}
		// 与 /pricing 完全一致地过滤：仅保留定价页当前可展示的模型集合中的项。
		// 之前操练场是按 model_test_results 全表 last_test_success=1 的模糊名字匹配做判定，
		// 与 /pricing 的「(模型,可见渠道) 严格匹配 + testMs>0 + ManualDisplayResponseTime 兜底 + ModelHasConfiguredPricing」口径不一致，
		// 导致诸如「最近一次单测失败但运营手动覆盖了展示耗时」「定价未配但偶然有过成功单测」等场景两端展示差异。
		pricingShowable := CollectPricingShowableModelNames()
		filteredNameRows := make([]struct {
			ModelName string
			VendorID  int
		}, 0, len(playgroundNameRows))
		for i := range playgroundNameRows {
			if !pricingShowable[playgroundNameRows[i].ModelName] {
				continue
			}
			filteredNameRows = append(filteredNameRows, playgroundNameRows[i])
		}
		vendorIDSet := make(map[int]struct{})
		for i := range filteredNameRows {
			if filteredNameRows[i].VendorID > 0 {
				vendorIDSet[filteredNameRows[i].VendorID] = struct{}{}
			}
		}
		vendorIDs := make([]int, 0, len(vendorIDSet))
		for id := range vendorIDSet {
			vendorIDs = append(vendorIDs, id)
		}
		vendorNameByID := make(map[int]string)
		// 操练场按 vendor_id 筛选须与下拉的 id 一一对应；不限制 status，避免元数据有 vendor_id 但库中已禁用时名称为空导致前端「按类型无数据」
		if len(vendorIDs) > 0 {
			vendorRows := make([]struct {
				Id   int    `gorm:"column:id"`
				Name string `gorm:"column:name"`
			}, 0)
			if err := model.DB.Model(&model.Vendor{}).
				Select("id", "name").
				Where("id IN ?", vendorIDs).
				Find(&vendorRows).Error; err != nil {
				common.ApiError(c, err)
				return
			}
			for i := range vendorRows {
				vendorNameByID[vendorRows[i].Id] = vendorRows[i].Name
			}
		}
		// 按用户可用分组 + 模型统计可选渠道（channels.id），供操练场前端做模型下的渠道联动下拉。
		modelChannelIDSet := make(map[string]map[int]struct{}, len(filteredNameRows))
		for i := range filteredNameRows {
			modelName := filteredNameRows[i].ModelName
			if modelName == "" {
				continue
			}
			if _, ok := modelChannelIDSet[modelName]; !ok {
				modelChannelIDSet[modelName] = make(map[int]struct{})
			}
			for group := range groups {
				channelIDs := model.ListChannelIDsForGroupModel(group, modelName)
				for _, channelID := range channelIDs {
					ch, chErr := model.CacheGetChannel(channelID)
					if chErr != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
						continue
					}
					modelChannelIDSet[modelName][channelID] = struct{}{}
				}
			}
		}
		channelMeta := make(map[int]playgroundChannelOption)
		for _, idSet := range modelChannelIDSet {
			for channelID := range idSet {
				if _, ok := channelMeta[channelID]; ok {
					continue
				}
				ch, chErr := model.CacheGetChannel(channelID)
				if chErr != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
					continue
				}
				channelMeta[channelID] = playgroundChannelOption{
					ID:           ch.Id,
					Name:         strings.TrimSpace(ch.Name),
					ChannelNo:    strings.TrimSpace(ch.ChannelNo),
					RouteSlug:    strings.TrimSpace(ch.RouteSlug),
					SupplierType: strings.TrimSpace(ch.SupplierType),
				}
			}
		}

		// 返回项均为单测成功的模型；有元数据则带 vendor，并附可选渠道列表
		items := make([]playgroundModelItem, 0, len(filteredNameRows))
		for i := range filteredNameRows {
			modelName := filteredNameRows[i].ModelName
			vendorID := filteredNameRows[i].VendorID
			vendorName := vendorNameByID[vendorID]
			channelOptions := make([]playgroundChannelOption, 0)
			for channelID := range modelChannelIDSet[modelName] {
				meta, ok := channelMeta[channelID]
				if !ok {
					continue
				}
				channelOptions = append(channelOptions, meta)
			}
			sort.Slice(channelOptions, func(i, j int) bool {
				if channelOptions[i].Name == channelOptions[j].Name {
					return channelOptions[i].ID < channelOptions[j].ID
				}
				return strings.Compare(channelOptions[i].Name, channelOptions[j].Name) < 0
			})
			items = append(items, playgroundModelItem{
				ModelName:      modelName,
				VendorID:       vendorID,
				Vendor:         vendorName,
				Tags:           modelTagsByName[modelName],
				TestedSuccess:  true,
				ChannelOptions: channelOptions,
			})
		}

		// 与模型广场（PricingVendors 仅基于当前模型集推导供应商）一致：类型选项只含本页 items 中实际出现过的 vendor_id，不用 GetVendors 全量，避免多一个「幽灵类型」、按类型筛选与 vendor_id 对不上导致列表全空
		// playgroundVendorOption 为操练场「模型类型」下拉中的一项，与 items[].vendor_id 一一可对应
		type playgroundVendorOption struct {
			id   int
			name string
		}
		playgroundVendorOptions := make([]playgroundVendorOption, 0, len(vendorIDSet)+1)
		for id := range vendorIDSet {
			nm := strings.TrimSpace(vendorNameByID[id])
			if nm == "" {
				nm = fmt.Sprintf("未关联#%d", id)
			}
			playgroundVendorOptions = append(playgroundVendorOptions, playgroundVendorOption{
				id:   id,
				name: nm,
			})
		}
		sort.Slice(playgroundVendorOptions, func(i, j int) bool {
			if playgroundVendorOptions[i].name == playgroundVendorOptions[j].name {
				return playgroundVendorOptions[i].id < playgroundVendorOptions[j].id
			}
			return strings.Compare(playgroundVendorOptions[i].name, playgroundVendorOptions[j].name) < 0
		})
		var hasUnassignedModel bool
		for i := range filteredNameRows {
			if filteredNameRows[i].VendorID == 0 {
				hasUnassignedModel = true
				break
			}
		}
		vendorOptions := make([]map[string]interface{}, 0, len(playgroundVendorOptions)+1)
		for i := range playgroundVendorOptions {
			vendorOptions = append(vendorOptions, map[string]interface{}{
				"id":   playgroundVendorOptions[i].id,
				"name": playgroundVendorOptions[i].name,
			})
		}
		if hasUnassignedModel {
			vendorOptions = append(vendorOptions, map[string]interface{}{
				"id":   0,
				"name": "未知模型类型",
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"items":          items,
				"vendor_options": vendorOptions,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
	return
}

func UpdateUser(c *gin.Context) {
	var updatedUser model.User
	err := json.NewDecoder(c.Request.Body).Decode(&updatedUser)
	if err != nil || updatedUser.Id == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if updatedUser.Password == "" {
		updatedUser.Password = "$I_LOVE_U" // make Validator happy :)
	}
	if err := common.Validate.Struct(&updatedUser); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	originUser, err := model.GetUserById(updatedUser.Id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	myRole := c.GetInt("role")
	if myRole <= originUser.Role && myRole != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
		return
	}
	if myRole <= updatedUser.Role && myRole != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserCannotCreateHigherLevel)
		return
	}
	if updatedUser.Password == "$I_LOVE_U" {
		updatedUser.Password = "" // rollback to what it should be
	}
	updatePassword := updatedUser.Password != ""
	if err := updatedUser.Edit(updatePassword); err != nil {
		if isPhoneUniqueConstraintError(err) {
			common.ApiErrorMsg(c, "手机号已被占用")
			return
		}
		common.ApiError(c, err)
		return
	}
	if originUser.Quota != updatedUser.Quota {
		model.RecordLog(originUser.Id, model.LogTypeManage, fmt.Sprintf("管理员将用户额度从 %s修改为 %s", logger.LogQuota(originUser.Quota), logger.LogQuota(updatedUser.Quota)))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func AdminClearUserBinding(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	bindingType := strings.ToLower(strings.TrimSpace(c.Param("binding_type")))
	if bindingType == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	myRole := c.GetInt("role")
	if myRole <= user.Role && myRole != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionSameLevel)
		return
	}

	if err := user.ClearBinding(bindingType); err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordLog(user.Id, model.LogTypeManage, fmt.Sprintf("admin cleared %s binding for user %s", bindingType, user.Username))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "success",
	})
}

func UpdateSelf(c *gin.Context) {
	var requestData map[string]interface{}
	err := json.NewDecoder(c.Request.Body).Decode(&requestData)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	// 检查是否是用户设置更新请求 (sidebar_modules 或 language)
	if sidebarModules, sidebarExists := requestData["sidebar_modules"]; sidebarExists {
		userId := c.GetInt("id")
		user, err := model.GetUserById(userId, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}

		// 获取当前用户设置
		currentSetting := user.GetSetting()

		// 更新sidebar_modules字段
		if sidebarModulesStr, ok := sidebarModules.(string); ok {
			currentSetting.SidebarModules = sidebarModulesStr
		}

		// 保存更新后的设置
		user.SetSetting(currentSetting)
		if err := user.Update(false); err != nil {
			common.ApiErrorI18n(c, i18n.MsgUpdateFailed)
			return
		}

		common.ApiSuccessI18n(c, i18n.MsgUpdateSuccess, nil)
		return
	}

	// 检查是否是语言偏好更新请求
	if language, langExists := requestData["language"]; langExists {
		userId := c.GetInt("id")
		user, err := model.GetUserById(userId, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}

		// 获取当前用户设置
		currentSetting := user.GetSetting()

		// 更新language字段
		if langStr, ok := language.(string); ok {
			currentSetting.Language = langStr
		}

		// 保存更新后的设置
		user.SetSetting(currentSetting)
		if err := user.Update(false); err != nil {
			common.ApiErrorI18n(c, i18n.MsgUpdateFailed)
			return
		}

		common.ApiSuccessI18n(c, i18n.MsgUpdateSuccess, nil)
		return
	}

	// 原有的用户信息更新逻辑
	var user model.User
	requestDataBytes, err := json.Marshal(requestData)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	err = json.Unmarshal(requestDataBytes, &user)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	if user.Password == "" {
		user.Password = "$I_LOVE_U" // make Validator happy :)
	}
	if err := common.Validate.Struct(&user); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidInput)
		return
	}

	if user.Password == "$I_LOVE_U" {
		user.Password = "" // rollback to what it should be
	}

	// 必须以数据库完整行为基础再合并请求字段；仅用 JSON 解出的局部 User 会含大量零值，
	// 若直接传入 Update() 会用 Select("*") 把角色/状态/用户名等全部覆盖掉。
	userId := c.GetInt("id")
	current, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	merged := *current
	if _, ok := requestData["username"]; ok {
		if s, ok := requestData["username"].(string); ok {
			merged.Username = strings.TrimSpace(s)
		}
	}
	if _, ok := requestData["display_name"]; ok {
		if s, ok := requestData["display_name"].(string); ok {
			merged.DisplayName = strings.TrimSpace(s)
		}
	}

	updatePassword, err := checkUpdatePassword(user.OriginalPassword, user.Password, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if updatePassword {
		merged.Password = user.Password
	}
	if err := merged.Update(updatePassword); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func checkUpdatePassword(originalPassword string, newPassword string, userId int) (updatePassword bool, err error) {
	var currentUser *model.User
	currentUser, err = model.GetUserById(userId, true)
	if err != nil {
		return
	}

	// 密码不为空,需要验证原密码
	// 支持第一次账号绑定时原密码为空的情况
	if !common.ValidatePasswordAndHash(originalPassword, currentUser.Password) && currentUser.Password != "" {
		err = fmt.Errorf("原密码错误")
		return
	}
	if newPassword == "" {
		return
	}
	updatePassword = true
	return
}

func DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	originUser, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	myRole := c.GetInt("role")
	if myRole <= originUser.Role {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
		return
	}
	err = model.HardDeleteUserById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
		})
		return
	}
}

func DeleteSelf(c *gin.Context) {
	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)

	if user.Role == common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserCannotDeleteRootUser)
		return
	}

	err := model.DeleteUserById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func CreateUser(c *gin.Context) {
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)
	if err != nil || user.Username == "" || user.Password == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := common.Validate.Struct(&user); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	if user.Role == common.RoleDistributorUser {
		user.Role = common.RoleCommonUser
		user.IsDistributor = common.DistributorFlagYes
	}
	myRole := c.GetInt("role")
	if user.Role >= myRole {
		common.ApiErrorI18n(c, i18n.MsgUserCannotCreateHigherLevel)
		return
	}
	normalizedPhone, err := model.NormalizeAndValidateAdminUserPhone(user.Phone, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	normalizedEmail, err := model.NormalizeAndValidateAdminUserEmail(user.Email, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Even for admin users, we cannot fully trust them!
	cleanUser := model.User{
		Username:                   user.Username,
		Password:                   user.Password,
		DisplayName:                user.DisplayName,
		Role:                       user.Role,
		IsDistributor:              user.IsDistributor,
		CreatedBy:                  common.UserCreatedByAdmin,
		Phone:                      normalizedPhone,
		Email:                      normalizedEmail,
		Remark:                     user.Remark,
		AdminInitialSetupCompleted: false,
	}
	if err := cleanUser.Insert(0); err != nil {
		if isPhoneUniqueConstraintError(err) {
			common.ApiErrorMsg(c, "手机号已被占用")
			return
		}
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

type ManageRequest struct {
	Id          int    `json:"id"`
	Action      string `json:"action"`
	RewardQuota int    `json:"reward_quota,omitempty"`
}

// ManageUser 管理员对用户启用/禁用、删除、提升/降级身份；分销商资格使用 is_distributor 与 set_distributor / unset_distributor。
func ManageUser(c *gin.Context) {
	var req ManageRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	user := model.User{
		Id: req.Id,
	}
	// Fill attributes
	model.DB.Unscoped().Where(&user).First(&user)
	if user.Id == 0 {
		common.ApiErrorI18n(c, i18n.MsgUserNotExists)
		return
	}
	myRole := c.GetInt("role")
	if myRole <= user.Role && myRole != common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
		return
	}
	beforeAdminDemote := false
	switch req.Action {
	case "disable":
		user.Status = common.UserStatusDisabled
		if user.Role == common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserCannotDisableRootUser)
			return
		}
	case "enable":
		user.Status = common.UserStatusEnabled
	case "delete":
		if user.Role == common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserCannotDeleteRootUser)
			return
		}
		if err := user.Delete(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "promote":
		// 仅超级管理员可将普通用户（含已开通分销商）提升为管理员；开通分销请使用 set_distributor
		switch user.Role {
		case common.RoleCommonUser:
			if myRole != common.RoleRootUser {
				common.ApiErrorMsg(c, "仅超级管理员可提升为管理员；为普通用户开通分销请使用「设为分销商」")
				return
			}
			user.Role = common.RoleAdminUser
			user.IsDistributor = common.DistributorFlagNo
		case common.RoleDistributorUser:
			if myRole != common.RoleRootUser {
				common.ApiErrorI18n(c, i18n.MsgUserAdminCannotPromote)
				return
			}
			user.Role = common.RoleAdminUser
			user.IsDistributor = common.DistributorFlagNo
		case common.RoleAdminUser, common.RoleRootUser:
			common.ApiErrorI18n(c, i18n.MsgUserCannotPromoteFurther)
			return
		default:
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
	case "demote":
		if user.Role == common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserCannotDemoteRootUser)
			return
		}
		switch user.Role {
		case common.RoleAdminUser:
			if myRole != common.RoleRootUser {
				common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
				return
			}
			user.Role = common.RoleCommonUser
			user.IsDistributor = common.DistributorFlagNo
			beforeAdminDemote = true
		case common.RoleDistributorUser:
			user.Role = common.RoleCommonUser
			user.IsDistributor = common.DistributorFlagNo
		case common.RoleCommonUser:
			common.ApiErrorMsg(c, "已是普通用户；取消分销资格请使用「取消分销商」")
			return
		default:
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
	case "set_distributor":
		if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "管理员账号无需开通分销商")
			return
		}
		if model.UserIsDistributor(&user) {
			common.ApiErrorMsg(c, "该用户已是分销商")
			return
		}
		user.IsDistributor = common.DistributorFlagYes
	case "unset_distributor":
		if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "管理员账号无分销商标记")
			return
		}
		if !model.UserIsDistributor(&user) {
			common.ApiErrorMsg(c, "该用户不是分销商")
			return
		}
		user.IsDistributor = common.DistributorFlagNo
	case "approve_student":
		if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "管理员账号不支持学员审批")
			return
		}
		if user.IsStudent == 1 && user.StudentStatus == common.StudentStatusApproved {
			common.ApiErrorMsg(c, "该用户已经是学员")
			return
		}
		now := time.Now()
		user.IsStudent = 1
		user.StudentStatus = common.StudentStatusApproved
		user.StudentApprovedAt = &now
		user.StudentApprovedBy = c.GetInt("id")
		if user.StudentApplied == nil {
			user.StudentApplied = &now
		}
	case "reject_student":
		if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "管理员账号不支持学员审批")
			return
		}
		if user.StudentStatus != common.StudentStatusPending {
			common.ApiErrorMsg(c, "该用户当前不在待审批状态")
			return
		}
		user.IsStudent = 0
		user.StudentStatus = common.StudentStatusRejected
		user.StudentApprovedAt = nil
		user.StudentApprovedBy = 0
	case "unset_student":
		if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "管理员账号不支持学员身份操作")
			return
		}
		if user.IsStudent != 1 || user.StudentStatus != common.StudentStatusApproved {
			common.ApiErrorMsg(c, "该用户不是学员")
			return
		}
		user.IsStudent = 0
		user.StudentStatus = common.StudentStatusNone
	case "set_student":
		if myRole != common.RoleAdminUser && myRole != common.RoleRootUser {
			common.ApiErrorI18n(c, i18n.MsgUserNoPermissionHigherLevel)
			return
		}
		if user.Role >= common.RoleAdminUser {
			common.ApiErrorMsg(c, "管理员账号不支持学员身份操作")
			return
		}
		if user.IsStudent == 1 && user.StudentStatus == common.StudentStatusApproved {
			common.ApiErrorMsg(c, "该用户已经是学员")
			return
		}
		now := time.Now()
		user.IsStudent = 1
		user.StudentStatus = common.StudentStatusApproved
		user.StudentApprovedAt = &now
		user.StudentApprovedBy = c.GetInt("id")
		if user.StudentApplied == nil {
			user.StudentApplied = &now
		}
	default:
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Action == "approve_student" || req.Action == "set_student" {
		rewardQuota := common.StudentApprovalRewardQuota
		if req.RewardQuota > 0 {
			rewardQuota = req.RewardQuota
		}
		if rewardQuota > 0 {
			if err := model.IncreaseUserQuota(user.Id, rewardQuota, true); err != nil {
				common.ApiError(c, err)
				return
			}
			actionLabel := "管理员审批学员申请"
			if req.Action == "set_student" {
				actionLabel = "管理员指定学员身份"
			}
			model.RecordLog(user.Id, model.LogTypeManage, fmt.Sprintf("%s，赠送 %s", actionLabel, logger.LogQuota(rewardQuota)))
		}
	}
	switch req.Action {
	case "set_distributor":
		service.NotifyDistributorRoleGranted(user.Id)
	case "unset_distributor":
		service.NotifyDistributorRoleRevoked(user.Id)
	case "demote":
		if beforeAdminDemote {
			service.NotifyUserDemotedFromAdmin(user.Id)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"role":           user.Role,
			"status":         user.Status,
			"is_distributor": user.IsDistributor,
			"is_student":     user.IsStudent,
			"student_status": user.StudentStatus,
		},
	})
	return
}

type emailBindRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func EmailBind(c *gin.Context) {
	var req emailBindRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	email := req.Email
	code := req.Code
	if !common.VerifyCodeWithKey(email, code, common.EmailVerificationPurpose) {
		common.ApiErrorI18n(c, i18n.MsgUserVerificationCodeError)
		return
	}
	session := sessions.Default(c)
	id := session.Get("id")
	user := model.User{
		Id: id.(int),
	}
	err := user.FillUserById()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user.Email = email
	// no need to check if this email already taken, because we have used verification code to check it
	err = user.Update(false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

type topUpRequest struct {
	Key string `json:"key"`
}

var topUpLocks sync.Map
var topUpCreateLock sync.Mutex

type topUpTryLock struct {
	ch chan struct{}
}

func newTopUpTryLock() *topUpTryLock {
	return &topUpTryLock{ch: make(chan struct{}, 1)}
}

func (l *topUpTryLock) TryLock() bool {
	select {
	case l.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *topUpTryLock) Unlock() {
	select {
	case <-l.ch:
	default:
	}
}

func getTopUpLock(userID int) *topUpTryLock {
	if v, ok := topUpLocks.Load(userID); ok {
		return v.(*topUpTryLock)
	}
	topUpCreateLock.Lock()
	defer topUpCreateLock.Unlock()
	if v, ok := topUpLocks.Load(userID); ok {
		return v.(*topUpTryLock)
	}
	l := newTopUpTryLock()
	topUpLocks.Store(userID, l)
	return l
}

func TopUp(c *gin.Context) {
	id := c.GetInt("id")
	lock := getTopUpLock(id)
	if !lock.TryLock() {
		common.ApiErrorI18n(c, i18n.MsgUserTopUpProcessing)
		return
	}
	defer lock.Unlock()
	req := topUpRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	quota, err := model.Redeem(req.Key, id)
	if err != nil {
		if errors.Is(err, model.ErrRedeemFailed) {
			common.ApiErrorI18n(c, i18n.MsgRedeemFailed)
			return
		}
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    quota,
	})
}

type UpdateUserSettingRequest struct {
	QuotaWarningType                 string  `json:"notify_type"`
	QuotaWarningThreshold            float64 `json:"quota_warning_threshold"`
	WebhookUrl                       string  `json:"webhook_url,omitempty"`
	WebhookSecret                    string  `json:"webhook_secret,omitempty"`
	NotificationEmail                string  `json:"notification_email,omitempty"`
	BarkUrl                          string  `json:"bark_url,omitempty"`
	GotifyUrl                        string  `json:"gotify_url,omitempty"`
	GotifyToken                      string  `json:"gotify_token,omitempty"`
	GotifyPriority                   int     `json:"gotify_priority,omitempty"`
	UpstreamModelUpdateNotifyEnabled *bool   `json:"upstream_model_update_notify_enabled,omitempty"`
	AcceptUnsetModelRatioModel       bool    `json:"accept_unset_model_ratio_model"`
	RecordIpLog                      bool    `json:"record_ip_log"`
}

func UpdateUserSetting(c *gin.Context) {
	var req UpdateUserSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	// 验证预警类型
	if req.QuotaWarningType != dto.NotifyTypeEmail && req.QuotaWarningType != dto.NotifyTypeWebhook && req.QuotaWarningType != dto.NotifyTypeBark && req.QuotaWarningType != dto.NotifyTypeGotify {
		common.ApiErrorI18n(c, i18n.MsgSettingInvalidType)
		return
	}

	// 验证预警阈值
	if req.QuotaWarningThreshold <= 0 {
		common.ApiErrorI18n(c, i18n.MsgQuotaThresholdGtZero)
		return
	}

	// 如果是webhook类型,验证webhook地址
	if req.QuotaWarningType == dto.NotifyTypeWebhook {
		if req.WebhookUrl == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingWebhookEmpty)
			return
		}
		// 验证URL格式
		if _, err := url.ParseRequestURI(req.WebhookUrl); err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingWebhookInvalid)
			return
		}
	}

	// 如果是邮件类型，验证邮箱地址
	if req.QuotaWarningType == dto.NotifyTypeEmail && req.NotificationEmail != "" {
		// 验证邮箱格式
		if !strings.Contains(req.NotificationEmail, "@") {
			common.ApiErrorI18n(c, i18n.MsgSettingEmailInvalid)
			return
		}
	}

	// 如果是Bark类型，验证Bark URL
	if req.QuotaWarningType == dto.NotifyTypeBark {
		if req.BarkUrl == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingBarkUrlEmpty)
			return
		}
		// 验证URL格式
		if _, err := url.ParseRequestURI(req.BarkUrl); err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingBarkUrlInvalid)
			return
		}
		// 检查是否是HTTP或HTTPS
		if !strings.HasPrefix(req.BarkUrl, "https://") && !strings.HasPrefix(req.BarkUrl, "http://") {
			common.ApiErrorI18n(c, i18n.MsgSettingUrlMustHttp)
			return
		}
	}

	// 如果是Gotify类型，验证Gotify URL和Token
	if req.QuotaWarningType == dto.NotifyTypeGotify {
		if req.GotifyUrl == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingGotifyUrlEmpty)
			return
		}
		if req.GotifyToken == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingGotifyTokenEmpty)
			return
		}
		// 验证URL格式
		if _, err := url.ParseRequestURI(req.GotifyUrl); err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingGotifyUrlInvalid)
			return
		}
		// 检查是否是HTTP或HTTPS
		if !strings.HasPrefix(req.GotifyUrl, "https://") && !strings.HasPrefix(req.GotifyUrl, "http://") {
			common.ApiErrorI18n(c, i18n.MsgSettingUrlMustHttp)
			return
		}
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	existingSettings := user.GetSetting()
	upstreamModelUpdateNotifyEnabled := existingSettings.UpstreamModelUpdateNotifyEnabled
	if user.Role >= common.RoleAdminUser && req.UpstreamModelUpdateNotifyEnabled != nil {
		upstreamModelUpdateNotifyEnabled = *req.UpstreamModelUpdateNotifyEnabled
	}

	// 构建设置
	settings := dto.UserSetting{
		NotifyType:                       req.QuotaWarningType,
		QuotaWarningThreshold:            req.QuotaWarningThreshold,
		UpstreamModelUpdateNotifyEnabled: upstreamModelUpdateNotifyEnabled,
		AcceptUnsetRatioModel:            req.AcceptUnsetModelRatioModel,
		RecordIpLog:                      req.RecordIpLog,
	}

	// 如果是webhook类型,添加webhook相关设置
	if req.QuotaWarningType == dto.NotifyTypeWebhook {
		settings.WebhookUrl = req.WebhookUrl
		if req.WebhookSecret != "" {
			settings.WebhookSecret = req.WebhookSecret
		}
	}

	// 如果提供了通知邮箱，添加到设置中
	if req.QuotaWarningType == dto.NotifyTypeEmail && req.NotificationEmail != "" {
		settings.NotificationEmail = req.NotificationEmail
	}

	// 如果是Bark类型，添加Bark URL到设置中
	if req.QuotaWarningType == dto.NotifyTypeBark {
		settings.BarkUrl = req.BarkUrl
	}

	// 如果是Gotify类型，添加Gotify配置到设置中
	if req.QuotaWarningType == dto.NotifyTypeGotify {
		settings.GotifyUrl = req.GotifyUrl
		settings.GotifyToken = req.GotifyToken
		// Gotify优先级范围0-10，超出范围则使用默认值5
		if req.GotifyPriority < 0 || req.GotifyPriority > 10 {
			settings.GotifyPriority = 5
		} else {
			settings.GotifyPriority = req.GotifyPriority
		}
	}

	// 更新用户设置
	user.SetSetting(settings)
	if err := user.Update(false); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUpdateFailed)
		return
	}

	common.ApiSuccessI18n(c, i18n.MsgSettingSaved, nil)
}
