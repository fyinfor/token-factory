package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const UserNameMaxLength = 20

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id                       int        `json:"id"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
	LastLoginAt              *time.Time `json:"last_login_at,omitempty" gorm:"column:last_login_at"`
	CreatedBy                string     `json:"created_by,omitempty" gorm:"column:created_by;type:varchar(32)"`
	Username                 string     `json:"username" gorm:"unique;index" validate:"max=20"`
	Password                 string     `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	OriginalPassword         string     `json:"original_password" gorm:"-:all"` // this field is only for Password change verification, don't save it to database!
	DisplayName              string     `json:"display_name" gorm:"index" validate:"max=20"`
	Role                     int        `json:"role" gorm:"type:int;default:1"`   // admin, common
	Status                   int        `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email                    string     `json:"email" gorm:"index" validate:"max=50"`
	Phone                    string     `json:"phone" gorm:"column:phone;type:varchar(20);index"`
	GitHubId                 string     `json:"github_id" gorm:"column:github_id;index"`
	DiscordId                string     `json:"discord_id" gorm:"column:discord_id;index"`
	OidcId                   string     `json:"oidc_id" gorm:"column:oidc_id;index"`
	WeChatId                 string     `json:"wechat_id" gorm:"column:wechat_id;index"`
	TelegramId               string     `json:"telegram_id" gorm:"column:telegram_id;index"`
	VerificationCode         string     `json:"verification_code" gorm:"-:all"`                                    // this field is only for Email verification, don't save it to database!
	AccessToken              *string    `json:"access_token" gorm:"type:char(32);column:access_token;uniqueIndex"` // this token is for system management
	Quota                    int        `json:"quota" gorm:"type:int;default:0"`
	UsedQuota                int        `json:"used_quota" gorm:"type:int;default:0;column:used_quota"` // used quota
	RequestCount             int        `json:"request_count" gorm:"type:int;default:0;"`               // request number
	Group                    string     `json:"group" gorm:"type:varchar(64);default:'default'"`
	AffCode                  string     `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	AffCount                 int        `json:"aff_count" gorm:"type:int;default:0;column:aff_count"`
	AffQuota                 int        `json:"aff_quota" gorm:"type:int;default:0;column:aff_quota"`           // 邀请剩余额度
	AffHistoryQuota          int        `json:"aff_history_quota" gorm:"type:int;default:0;column:aff_history"` // 邀请历史额度
	InviterId                int        `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	DistributorCommissionBps int        `json:"distributor_commission_bps" gorm:"type:int;default:0;column:distributor_commission_bps"` // 分销商名下新邀请关系的默认分成（万分之一），0 表示跟随系统 AffiliateDefaultCommissionBps
	// IsDistributor 分销商资格 0/1（与 role 解耦）；普通用户 role=1 时可同时为分销商。旧版 role=5 已迁移为 role=1 + is_distributor=1。
	IsDistributor     int            `json:"is_distributor" gorm:"column:is_distributor;type:integer;default:0;index"`
	IsStudent         int            `json:"is_student" gorm:"column:is_student;type:integer;default:0;index"`
	StudentStatus     int            `json:"student_status" gorm:"column:student_status;type:integer;default:0;index"`
	StudentApplied    *time.Time     `json:"student_applied_at,omitempty" gorm:"column:student_applied_at"`
	StudentApprovedAt *time.Time     `json:"student_approved_at,omitempty" gorm:"column:student_approved_at"`
	StudentApprovedBy int            `json:"student_approved_by" gorm:"column:student_approved_by;type:int;default:0;index"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
	LinuxDOId         string         `json:"linux_do_id" gorm:"column:linux_do_id;index"`
	Setting           string         `json:"setting" gorm:"type:text;column:setting"`
	Remark            string         `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	StripeCustomer    string         `json:"stripe_customer" gorm:"type:varchar(64);column:stripe_customer;index"`
	SupplierID        int            `json:"supplier_id" gorm:"type:int;column:supplier_id;index;default:0;comment:供应商申请ID 0表示非供应商"`
	// AdminInitialSetupCompleted 管理员代建账号首次登录前须为 false；自助注册等为 true。注意：GORM Create 会省略 bool 的 false，代建分支须在 Insert 内显式 UPDATE 落库为 0。
	AdminInitialSetupCompleted bool `json:"admin_initial_setup_completed" gorm:"column:admin_initial_setup_completed;type:boolean;not null;default:true"`
}

func (user *User) ToBaseUser() *UserBase {
	cache := &UserBase{
		Id:       user.Id,
		Group:    user.Group,
		Quota:    user.Quota,
		Status:   user.Status,
		Username: user.Username,
		Setting:  user.Setting,
		Email:    user.Email,
	}
	return cache
}

func (user *User) GetAccessToken() string {
	if user.AccessToken == nil {
		return ""
	}
	return *user.AccessToken
}

func (user *User) SetAccessToken(token string) {
	user.AccessToken = &token
}

func (user *User) GetSetting() dto.UserSetting {
	setting := dto.UserSetting{}
	if user.Setting != "" {
		err := json.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog("failed to unmarshal setting: " + err.Error())
		}
	}
	return setting
}

func (user *User) SetSetting(setting dto.UserSetting) {
	settingBytes, err := json.Marshal(setting)
	if err != nil {
		common.SysLog("failed to marshal setting: " + err.Error())
		return
	}
	user.Setting = string(settingBytes)
}

// 根据用户角色生成默认的边栏配置
func generateDefaultSidebarConfigForRole(userRole int) string {
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

// CheckUserExistOrDeleted 判断是否已有用户使用相同用户名，或与传入的非空邮箱冲突（含软删除）。
// 注册接口已改用 IsUsernameTakenUnscoped / IsEmailTakenUnscoped 分别提示用户名与邮箱冲突。
func CheckUserExistOrDeleted(username string, email string) (bool, error) {
	var user User

	// err := DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	// check email if empty
	var err error
	if email == "" {
		err = DB.Unscoped().First(&user, "username = ?", username).Error
	} else {
		err = DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// not exist, return false, nil
			return false, nil
		}
		// other error, return false, err
		return false, err
	}
	// exist, return true, nil
	return true, nil
}

// IsUsernameTakenUnscoped 判断用户名是否已被占用（含软删除），用于注册等场景的精确提示。
func IsUsernameTakenUnscoped(username string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil
	}
	var user User
	err := DB.Unscoped().Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsEmailTakenUnscoped 判断邮箱是否已被占用（含软删除）；邮箱为空时不视为占用。
func IsEmailTakenUnscoped(email string) (bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return false, nil
	}
	var user User
	err := DB.Unscoped().Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsEmailTakenByOtherUser 判断邮箱是否已被除 excludeUserId 以外的用户占用（包含软删除记录）。
func IsEmailTakenByOtherUser(email string, excludeUserId int) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	return DB.Unscoped().Where("email = ? AND id <> ?", email, excludeUserId).Find(&User{}).RowsAffected > 0
}

// NormalizeAndValidateAdminUserEmail 管理员创建/编辑用户时的邮箱：去首尾空格；空表示不绑定；非空则校验格式、长度与占用（excludeUserId=0 表示新建）。
func NormalizeAndValidateAdminUserEmail(email string, excludeUserId int) (string, error) {
	n := strings.TrimSpace(email)
	if n == "" {
		return "", nil
	}
	if err := common.Validate.Var(n, "email,max=50"); err != nil {
		return "", fmt.Errorf("邮箱格式无效")
	}
	if excludeUserId == 0 {
		taken, err := IsEmailTakenUnscoped(n)
		if err != nil {
			return "", err
		}
		if taken {
			return "", fmt.Errorf("邮箱已被占用")
		}
	} else {
		if IsEmailTakenByOtherUser(n, excludeUserId) {
			return "", fmt.Errorf("邮箱已被占用")
		}
	}
	return n, nil
}

func GetMaxUserId() int {
	var user User
	DB.Unscoped().Last(&user)
	return user.Id
}

// TouchUserLastLogin 在用户成功建立会话（登录）后更新上次登录时间。
func TouchUserLastLogin(userId int) {
	if userId <= 0 {
		return
	}
	now := time.Now()
	if err := DB.Model(&User{}).Where("id = ?", userId).Update("last_login_at", now).Error; err != nil {
		common.SysLog("TouchUserLastLogin: " + err.Error())
	}
}

func applyStudentViewFilter(query *gorm.DB, studentView string) *gorm.DB {
	switch strings.TrimSpace(studentView) {
	case "pending":
		return query.Where("student_status = ?", common.StudentStatusPending)
	case "students":
		return query.Where("is_student = ? AND student_status = ?", 1, common.StudentStatusApproved)
	default:
		return query
	}
}

func GetAllUsers(pageInfo *common.PageInfo, studentView string) (users []*User, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get total count within transaction
	baseQuery := applyStudentViewFilter(tx.Unscoped().Model(&User{}), studentView)
	err = baseQuery.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated users within same transaction
	err = baseQuery.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Omit("password").Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func SearchUsers(keyword string, group string, studentView string, startIdx int, num int) ([]*User, int64, error) {
	var users []*User
	var total int64
	var err error

	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 构建基础查询
	query := tx.Unscoped().Model(&User{})

	query = applyStudentViewFilter(query, studentView)

	// 构建搜索条件
	likeCondition := "username LIKE ? OR email LIKE ? OR display_name LIKE ? OR phone LIKE ?"

	// 尝试将关键字转换为整数ID
	keywordInt, err := strconv.Atoi(keyword)
	if err == nil {
		// 如果是数字，同时搜索ID和其他字段
		likeCondition = "id = ? OR " + likeCondition
		if group != "" {
			query = query.Where("("+likeCondition+") AND "+commonGroupCol+" = ?",
				keywordInt, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", group)
		} else {
			query = query.Where(likeCondition,
				keywordInt, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
	} else {
		// 非数字关键字，只搜索字符串字段
		if group != "" {
			query = query.Where("("+likeCondition+") AND "+commonGroupCol+" = ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", group)
		} else {
			query = query.Where(likeCondition,
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
	}

	// 获取总数
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = query.Omit("password").Order("id desc").Limit(num).Offset(startIdx).Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	user := User{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&user, "id = ?", id).Error
	} else {
		err = DB.Omit("password").First(&user, "id = ?", id).Error
	}
	return &user, err
}

func GetUserIdByAffCode(affCode string) (int, error) {
	if affCode == "" {
		return 0, errors.New("affCode 为空！")
	}
	var user User
	err := DB.Select("id").First(&user, "aff_code = ?", affCode).Error
	return user.Id, err
}

// EnsureAffCode generates a unique aff_code for the user if it is empty,
// retrying on rare collisions. This prevents duplicate-key errors on
// the idx_users_aff_code unique index when multiple users have aff_code = ”.
func (user *User) EnsureAffCode() {
	if user.AffCode != "" {
		return
	}
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		code := common.GetRandomString(6) // 6 chars ≈ 2.2B combos (alphanumeric), negligible collision
		var count int64
		DB.Model(&User{}).Where("aff_code = ? AND id != ?", code, user.Id).Count(&count)
		if count == 0 {
			user.AffCode = code
			return
		}
	}
	// Fallback: append user id to guarantee uniqueness
	user.AffCode = common.GetRandomString(4) + fmt.Sprintf("%d", user.Id)
}

// BackfillEmptyAffCodes finds all users whose aff_code is empty and assigns
// each a unique aff_code. This is needed because aff_code has a uniqueIndex,
// and multiple rows with aff_code = ” violate that constraint on update.
func BackfillEmptyAffCodes() error {
	var users []User
	if err := DB.Unscoped().Select("id").Where("aff_code = ''").Find(&users).Error; err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	common.SysLog(fmt.Sprintf("backfill empty aff_code: %d user(s) need assignment", len(users)))
	for i := range users {
		users[i].EnsureAffCode()
		if users[i].AffCode == "" {
			common.SysError(fmt.Sprintf("backfill empty aff_code: failed to generate code for user %d", users[i].Id))
			continue
		}
		if err := DB.Model(&User{}).Where("id = ?", users[i].Id).UpdateColumn("aff_code", users[i].AffCode).Error; err != nil {
			common.SysError(fmt.Sprintf("backfill empty aff_code: user %d: %s", users[i].Id, err.Error()))
		}
	}
	return nil
}

func DeleteUserById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	user := User{Id: id}
	return user.Delete()
}

func HardDeleteUserById(id int) error {
	if id == 0 {
		return errors.New("id 为空！")
	}
	err := DB.Unscoped().Delete(&User{}, "id = ?", id).Error
	return err
}

// inviteUser 在新用户通过邀请注册成功后调用：邀请人数（aff_count）+1；
// 若运营配置了邀请人注册奖励（QuotaForInviter），则直接增加邀请人可用额度（quota）。
//
// 说明：注册类邀请奖励与「分销充值提成」分流——后者仍通过 IncreaseUserAffCommissionQuota
// 写入 aff_quota / aff_history；本函数不再触碰 aff_quota、aff_history，避免与分销待结算/历史统计混淆。
// 历史已写入 aff_* 的数据不做迁移，仅新产生的注册奖励走 quota。
func inviteUser(inviterId int) (err error) {
	if inviterId <= 0 {
		return nil
	}
	if _, err = GetUserById(inviterId, true); err != nil {
		return err
	}

	reward := common.QuotaForInviter
	// 与 IncreaseUserQuota 一致：Batch 模式下额度写入走批处理队列，不能在事务内直接改 quota。
	useBatchQuota := reward > 0 && common.BatchUpdateEnabled

	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	if err = tx.Model(&User{}).Where("id = ?", inviterId).UpdateColumn("aff_count", gorm.Expr("aff_count + ?", 1)).Error; err != nil {
		return err
	}
	if reward > 0 && !useBatchQuota {
		if err = tx.Model(&User{}).Where("id = ?", inviterId).UpdateColumn("quota", gorm.Expr("quota + ?", reward)).Error; err != nil {
			return err
		}
	}
	if err = tx.Commit().Error; err != nil {
		return err
	}

	if useBatchQuota {
		if err = IncreaseUserQuota(inviterId, reward, true); err != nil {
			return err
		}
	} else if reward > 0 {
		gopool.Go(func() {
			if err := cacheIncrUserQuota(inviterId, int64(reward)); err != nil {
				common.SysLog("inviteUser cacheIncrUserQuota: " + err.Error())
			}
		})
	}

	inviter, err := GetUserById(inviterId, true)
	if err != nil {
		return err
	}
	return updateUserCache(*inviter)
}

func (user *User) TransferAffQuotaToQuota(quota int) error {
	// 检查quota是否小于最小额度
	if float64(quota) < common.QuotaPerUnit {
		return fmt.Errorf("转移额度最小为%s！", logger.LogQuota(int(common.QuotaPerUnit)))
	}

	// 开始数据库事务
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // 确保在函数退出时事务能回滚

	// 加锁查询用户以确保数据一致性
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, user.Id).Error
	if err != nil {
		return err
	}

	// 再次检查用户的AffQuota是否足够
	if user.AffQuota < quota {
		return errors.New("邀请额度不足！")
	}

	// 更新用户额度
	user.AffQuota -= quota
	user.Quota += quota

	// 保存用户状态
	if err := tx.Save(user).Error; err != nil {
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

func (user *User) Insert(inviterId int) error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = common.QuotaForNewUser
	//user.SetAccessToken(common.GetUUID())
	user.EnsureAffCode()

	// 初始化用户设置，包括默认的边栏配置
	if user.Setting == "" {
		defaultSetting := dto.UserSetting{}
		// 这里暂时不设置SidebarModules，因为需要在用户创建后根据角色设置
		user.SetSetting(defaultSetting)
	}
	if user.CreatedBy == "" {
		user.CreatedBy = common.UserCreatedByRegistration
	}
	// 非管理员代建账号默认可正常使用；管理员代建由 controller 显式置为 false
	if user.CreatedBy != common.UserCreatedByAdmin {
		user.AdminInitialSetupCompleted = true
	}

	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}
	// 管理员代建：Create 不会写入 false，MySQL 会落在列默认值 1；必须显式更新为 0，首次登录才会要求改密/补手机。
	if user.CreatedBy == common.UserCreatedByAdmin && user.Id > 0 {
		if err := DB.Model(&User{}).Where("id = ?", user.Id).UpdateColumn("admin_initial_setup_completed", false).Error; err != nil {
			return err
		}
		user.AdminInitialSetupCompleted = false
	}

	// 用户创建成功后，根据角色初始化边栏配置
	// 需要重新获取用户以确保有正确的ID和Role
	var createdUser User
	if err := DB.Where("username = ?", user.Username).First(&createdUser).Error; err == nil {
		// 生成基于角色的默认边栏配置
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(fmt.Sprintf("为新用户 %s (角色: %d) 初始化边栏配置", createdUser.Username, createdUser.Role))
		}
	}

	if common.QuotaForNewUser > 0 {
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("新用户注册赠送 %s", logger.LogQuota(common.QuotaForNewUser)))
	}
	if inviterId != 0 {
		_ = EnsureAffInviteRelation(inviterId, user.Id)
		if common.QuotaForInvitee > 0 {
			_ = IncreaseUserQuota(user.Id, common.QuotaForInvitee, true)
			RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		if common.QuotaForInviter > 0 {
			RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(common.QuotaForInviter)))
		}
		_ = inviteUser(inviterId)
	}
	return nil
}

// InsertWithTx inserts a new user within an existing transaction.
// This is used for OAuth registration where user creation and binding need to be atomic.
// Post-creation tasks (sidebar config, logs, inviter rewards) are handled after the transaction commits.
func (user *User) InsertWithTx(tx *gorm.DB, inviterId int) error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = common.QuotaForNewUser
	user.EnsureAffCode()

	// 初始化用户设置
	if user.Setting == "" {
		defaultSetting := dto.UserSetting{}
		user.SetSetting(defaultSetting)
	}
	if user.CreatedBy == "" {
		user.CreatedBy = common.UserCreatedByRegistration
	}
	if user.CreatedBy != common.UserCreatedByAdmin {
		user.AdminInitialSetupCompleted = true
	}

	result := tx.Create(user)
	if result.Error != nil {
		return result.Error
	}
	if user.CreatedBy == common.UserCreatedByAdmin && user.Id > 0 {
		if err := tx.Model(&User{}).Where("id = ?", user.Id).UpdateColumn("admin_initial_setup_completed", false).Error; err != nil {
			return err
		}
		user.AdminInitialSetupCompleted = false
	}

	return nil
}

// FinalizeOAuthUserCreation performs post-transaction tasks for OAuth user creation.
// This should be called after the transaction commits successfully.
func (user *User) FinalizeOAuthUserCreation(inviterId int) {
	// 用户创建成功后，根据角色初始化边栏配置
	var createdUser User
	if err := DB.Where("id = ?", user.Id).First(&createdUser).Error; err == nil {
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(fmt.Sprintf("为新用户 %s (角色: %d) 初始化边栏配置", createdUser.Username, createdUser.Role))
		}
	}

	if common.QuotaForNewUser > 0 {
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("新用户注册赠送 %s", logger.LogQuota(common.QuotaForNewUser)))
	}
	if inviterId != 0 {
		_ = EnsureAffInviteRelation(inviterId, user.Id)
		if common.QuotaForInvitee > 0 {
			_ = IncreaseUserQuota(user.Id, common.QuotaForInvitee, true)
			RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		if common.QuotaForInviter > 0 {
			RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(common.QuotaForInviter)))
		}
		_ = inviteUser(inviterId)
	}
}

func (user *User) Update(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	newUser := *user
	DB.First(&user, user.Id)

	// 避免请求体/部分结构体中的零值覆盖注册时间、上次登录；并刷新修改时间
	newUser.CreatedAt = user.CreatedAt
	newUser.LastLoginAt = user.LastLoginAt
	newUser.CreatedBy = user.CreatedBy
	newUser.UpdatedAt = time.Now()
	// 确保 aff_code 不为空，避免唯一索引冲突（多个空字符串行无法共存）
	newUser.EnsureAffCode()
	// Select("*") 否则 Updates(struct) 会忽略零值字段（如 is_distributor=0、quota=0），导致无法取消分销商等操作失效
	if err = DB.Model(user).Select("*").Updates(newUser).Error; err != nil {
		return err
	}
	// 缓存必须与落库的 newUser 一致；*user 仅为 First 后的快照，密码等可能已过时。
	return updateUserCache(newUser)
}

func (user *User) Edit(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}

	newUser := *user
	normalizedPhone, err := NormalizeAndValidateAdminUserPhone(newUser.Phone, newUser.Id)
	if err != nil {
		return err
	}
	normalizedEmail, err := NormalizeAndValidateAdminUserEmail(newUser.Email, newUser.Id)
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"username":     newUser.Username,
		"display_name": newUser.DisplayName,
		"group":        newUser.Group,
		"quota":        newUser.Quota,
		"remark":       newUser.Remark,
		"phone":        normalizedPhone,
		"email":        normalizedEmail,
		"updated_at":   time.Now(),
	}
	if updatePassword {
		updates["password"] = newUser.Password
	}

	DB.First(&user, user.Id)
	if err = DB.Model(user).Updates(updates).Error; err != nil {
		return err
	}

	// Update cache
	return updateUserCache(*user)
}

func (user *User) ClearBinding(bindingType string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}

	bindingColumnMap := map[string]string{
		"email":    "email",
		"github":   "github_id",
		"discord":  "discord_id",
		"oidc":     "oidc_id",
		"wechat":   "wechat_id",
		"telegram": "telegram_id",
		"linuxdo":  "linux_do_id",
	}

	column, ok := bindingColumnMap[bindingType]
	if !ok {
		return errors.New("invalid binding type")
	}

	if err := DB.Model(&User{}).Where("id = ?", user.Id).Update(column, "").Error; err != nil {
		return err
	}

	if err := DB.Where("id = ?", user.Id).First(user).Error; err != nil {
		return err
	}

	return updateUserCache(*user)
}

func (user *User) Delete() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	if err := DB.Delete(user).Error; err != nil {
		return err
	}

	// 清除缓存
	return invalidateUserCache(user.Id)
}

func (user *User) HardDelete() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	err := DB.Unscoped().Delete(user).Error
	return err
}

// ValidateAndFill check password & user status
func (user *User) ValidateAndFill() (err error) {
	// When querying with struct, GORM will only query with non-zero fields,
	// that means if your field's value is 0, '', false or other zero values,
	// it won't be used to build query conditions
	password := user.Password
	username := strings.TrimSpace(user.Username)
	if username == "" || password == "" {
		return errors.New("用户名或密码为空")
	}
	// find buy username or email
	DB.Where("username = ? OR email = ?", username, username).First(user)
	okay := common.ValidatePasswordAndHash(password, user.Password)
	if !okay || user.Status != common.UserStatusEnabled {
		return errors.New("用户名或密码错误，或用户已被封禁")
	}
	return nil
}

func (user *User) FillUserById() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	DB.Where(User{Id: user.Id}).First(user)
	return nil
}

func (user *User) FillUserByEmail() error {
	if user.Email == "" {
		return errors.New("email 为空！")
	}
	DB.Where(User{Email: user.Email}).First(user)
	return nil
}

func (user *User) FillUserByGitHubId() error {
	if user.GitHubId == "" {
		return errors.New("GitHub id 为空！")
	}
	DB.Where(User{GitHubId: user.GitHubId}).First(user)
	return nil
}

// UpdateGitHubId updates the user's GitHub ID (used for migration from login to numeric ID)
func (user *User) UpdateGitHubId(newGitHubId string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}
	return DB.Model(user).Update("github_id", newGitHubId).Error
}

func (user *User) FillUserByDiscordId() error {
	if user.DiscordId == "" {
		return errors.New("discord id 为空！")
	}
	DB.Where(User{DiscordId: user.DiscordId}).First(user)
	return nil
}

func (user *User) FillUserByOidcId() error {
	if user.OidcId == "" {
		return errors.New("oidc id 为空！")
	}
	DB.Where(User{OidcId: user.OidcId}).First(user)
	return nil
}

func (user *User) FillUserByWeChatId() error {
	if user.WeChatId == "" {
		return errors.New("WeChat id 为空！")
	}
	DB.Where(User{WeChatId: user.WeChatId}).First(user)
	return nil
}

func (user *User) FillUserByTelegramId() error {
	if user.TelegramId == "" {
		return errors.New("Telegram id 为空！")
	}
	err := DB.Where(User{TelegramId: user.TelegramId}).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("该 Telegram 账户未绑定")
	}
	return nil
}

func IsEmailAlreadyTaken(email string) bool {
	return DB.Unscoped().Where("email = ?", email).Find(&User{}).RowsAffected == 1
}

// IsPhoneAlreadyTaken 判断手机号是否已被占用（包含软删除记录）。
func IsPhoneAlreadyTaken(phone string) bool {
	phone = common.NormalizePhone(phone)
	if phone == "" {
		return false
	}
	return DB.Unscoped().Where("phone = ?", phone).Find(&User{}).RowsAffected > 0
}

// IsPhoneTakenByOtherUser 判断手机号是否已被除 excludeUserId 以外的用户占用（包含软删除记录）。
func IsPhoneTakenByOtherUser(phone string, excludeUserId int) bool {
	phone = common.NormalizePhone(phone)
	if phone == "" {
		return false
	}
	return DB.Unscoped().Where("phone = ? AND id <> ?", phone, excludeUserId).Find(&User{}).RowsAffected > 0
}

// NormalizeAndValidateAdminUserPhone 管理员创建/编辑用户时的手机号：规范化；空字符串表示不绑定；非空则校验格式、黑名单与占用（excludeUserId=0 表示新建用户）。
func NormalizeAndValidateAdminUserPhone(phone string, excludeUserId int) (string, error) {
	n := common.NormalizePhone(phone)
	if n == "" {
		return "", nil
	}
	if !common.ValidateMainlandChinaPhone(n) {
		return "", fmt.Errorf("手机号格式无效，请输入 11 位中国大陆手机号")
	}
	if common.IsSMSPhoneBlacklisted(n) {
		return "", fmt.Errorf("该手机号已被加入短信黑名单")
	}
	if excludeUserId == 0 {
		if IsPhoneAlreadyTaken(n) {
			return "", fmt.Errorf("手机号已被占用")
		}
	} else {
		if IsPhoneTakenByOtherUser(n, excludeUserId) {
			return "", fmt.Errorf("手机号已被占用")
		}
	}
	return n, nil
}

func IsWeChatIdAlreadyTaken(wechatId string) bool {
	return DB.Unscoped().Where("wechat_id = ?", wechatId).Find(&User{}).RowsAffected == 1
}

func IsGitHubIdAlreadyTaken(githubId string) bool {
	return DB.Unscoped().Where("github_id = ?", githubId).Find(&User{}).RowsAffected == 1
}

func IsDiscordIdAlreadyTaken(discordId string) bool {
	return DB.Unscoped().Where("discord_id = ?", discordId).Find(&User{}).RowsAffected == 1
}

func IsOidcIdAlreadyTaken(oidcId string) bool {
	return DB.Where("oidc_id = ?", oidcId).Find(&User{}).RowsAffected == 1
}

func IsTelegramIdAlreadyTaken(telegramId string) bool {
	return DB.Unscoped().Where("telegram_id = ?", telegramId).Find(&User{}).RowsAffected == 1
}

func ResetUserPasswordByEmail(email string, password string) error {
	if email == "" || password == "" {
		return errors.New("邮箱地址或密码为空！")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	err = DB.Model(&User{}).Where("email = ?", email).Update("password", hashedPassword).Error
	return err
}

// ResetUserPasswordByPhone 按手机号重置用户密码。
func ResetUserPasswordByPhone(phone string, password string) error {
	phone = common.NormalizePhone(phone)
	if phone == "" || password == "" {
		return errors.New("手机号或密码为空！")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	err = DB.Model(&User{}).Where("phone = ?", phone).Update("password", hashedPassword).Error
	return err
}

func IsAdmin(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role").Find(&user).Error
	if err != nil {
		common.SysLog("no such user " + err.Error())
		return false
	}
	return user.Role >= common.RoleAdminUser
}

//// IsUserEnabled checks user status from Redis first, falls back to DB if needed
//func IsUserEnabled(id int, fromDB bool) (status bool, err error) {
//	defer func() {
//		// Update Redis cache asynchronously on successful DB read
//		if shouldUpdateRedis(fromDB, err) {
//			gopool.Go(func() {
//				if err := updateUserStatusCache(id, status); err != nil {
//					common.SysError("failed to update user status cache: " + err.Error())
//				}
//			})
//		}
//	}()
//	if !fromDB && common.RedisEnabled {
//		// Try Redis first
//		status, err := getUserStatusCache(id)
//		if err == nil {
//			return status == common.UserStatusEnabled, nil
//		}
//		// Don't return error - fall through to DB
//	}
//	fromDB = true
//	var user User
//	err = DB.Where("id = ?", id).Select("status").Find(&user).Error
//	if err != nil {
//		return false, err
//	}
//
//	return user.Status == common.UserStatusEnabled, nil
//}

func ValidateAccessToken(token string) (user *User) {
	if token == "" {
		return nil
	}
	token = strings.Replace(token, "Bearer ", "", 1)
	user = &User{}
	if DB.Where("access_token = ?", token).First(user).RowsAffected == 1 {
		return user
	}
	return nil
}

// GetUserQuota gets quota from Redis first, falls back to DB if needed
func GetUserQuota(id int, fromDB bool) (quota int, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserQuotaCache(id, quota); err != nil {
					common.SysLog("failed to update user quota cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		quota, err := getUserQuotaCache(id)
		if err == nil {
			return quota, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("quota").Find(&quota).Error
	if err != nil {
		return 0, err
	}

	return quota, nil
}

func GetUserUsedQuota(id int) (quota int, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("used_quota").Find(&quota).Error
	return quota, err
}

func GetUserEmail(id int) (email string, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("email").Find(&email).Error
	return email, err
}

// GetUserGroup gets group from Redis first, falls back to DB if needed
func GetUserGroup(id int, fromDB bool) (group string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserGroupCache(id, group); err != nil {
					common.SysLog("failed to update user group cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		group, err := getUserGroupCache(id)
		if err == nil {
			return group, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select(commonGroupCol).Find(&group).Error
	if err != nil {
		return "", err
	}

	return group, nil
}

// GetUserSetting gets setting from Redis first, falls back to DB if needed
func GetUserSetting(id int, fromDB bool) (settingMap dto.UserSetting, err error) {
	var setting string
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserSettingCache(id, setting); err != nil {
					common.SysLog("failed to update user setting cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		setting, err := getUserSettingCache(id)
		if err == nil {
			return setting, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	// can be nil setting
	var safeSetting sql.NullString
	err = DB.Model(&User{}).Where("id = ?", id).Select("setting").Find(&safeSetting).Error
	if err != nil {
		return settingMap, err
	}
	if safeSetting.Valid {
		setting = safeSetting.String
	} else {
		setting = ""
	}
	userBase := &UserBase{
		Setting: setting,
	}
	return userBase.GetSetting(), nil
}

func IncreaseUserQuota(id int, quota int, db bool) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	gopool.Go(func() {
		err := cacheIncrUserQuota(id, int64(quota))
		if err != nil {
			common.SysLog("failed to increase user quota: " + err.Error())
		}
	})
	if !db && common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, quota)
		return nil
	}
	return increaseUserQuota(id, quota)
}

func increaseUserQuota(id int, quota int) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota + ?", quota)).Error
	if err != nil {
		return err
	}
	return err
}

// IncreaseUserAffCommissionQuota 分销提成计入邀请人待使用收益(aff_quota)与累计总收益(aff_history)，不增加 quota。
func IncreaseUserAffCommissionQuota(inviterId int, delta int) error {
	if inviterId <= 0 || delta <= 0 {
		return nil
	}
	tx := DB.Model(&User{}).Where("id = ?", inviterId).Updates(map[string]interface{}{
		"aff_quota":   gorm.Expr("aff_quota + ?", delta),
		"aff_history": gorm.Expr("aff_history + ?", delta),
	})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("inviter user not found: %d", inviterId)
	}
	return nil
}

func DecreaseUserQuota(id int, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	gopool.Go(func() {
		err := cacheDecrUserQuota(id, int64(quota))
		if err != nil {
			common.SysLog("failed to decrease user quota: " + err.Error())
		}
	})
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, -quota)
		return nil
	}
	return decreaseUserQuota(id, quota)
}

func decreaseUserQuota(id int, quota int) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota - ?", quota)).Error
	if err != nil {
		return err
	}
	return err
}

func DeltaUpdateUserQuota(id int, delta int) (err error) {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return IncreaseUserQuota(id, delta, false)
	} else {
		return DecreaseUserQuota(id, -delta)
	}
}

//func GetRootUserEmail() (email string) {
//	DB.Model(&User{}).Where("role = ?", common.RoleRootUser).Select("email").Find(&email)
//	return email
//}

func GetRootUser() (user *User) {
	DB.Where("role = ?", common.RoleRootUser).First(&user)
	return user
}

func UpdateUserUsedQuotaAndRequestCount(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUsedQuota, id, quota)
		addNewRecord(BatchUpdateTypeRequestCount, id, 1)
		return
	}
	updateUserUsedQuotaAndRequestCount(id, quota, 1)
}

func updateUserUsedQuotaAndRequestCount(id int, quota int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"request_count": gorm.Expr("request_count + ?", count),
		},
	).Error
	if err != nil {
		common.SysLog("failed to update user used quota and request count: " + err.Error())
		return
	}

	//// 更新缓存
	//if err := invalidateUserCache(id); err != nil {
	//	common.SysError("failed to invalidate user cache: " + err.Error())
	//}
}

func updateUserUsedQuota(id int, quota int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota": gorm.Expr("used_quota + ?", quota),
		},
	).Error
	if err != nil {
		common.SysLog("failed to update user used quota: " + err.Error())
	}
}

func updateUserRequestCount(id int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Update("request_count", gorm.Expr("request_count + ?", count)).Error
	if err != nil {
		common.SysLog("failed to update user request count: " + err.Error())
	}
}

// GetUsernameById gets username from Redis first, falls back to DB if needed
func GetUsernameById(id int, fromDB bool) (username string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserNameCache(id, username); err != nil {
					common.SysLog("failed to update user name cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		username, err := getUserNameCache(id)
		if err == nil {
			return username, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("username").Find(&username).Error
	if err != nil {
		return "", err
	}

	return username, nil
}

func IsLinuxDOIdAlreadyTaken(linuxDOId string) bool {
	var user User
	err := DB.Unscoped().Where("linux_do_id = ?", linuxDOId).First(&user).Error
	return !errors.Is(err, gorm.ErrRecordNotFound)
}

func (user *User) FillUserByLinuxDOId() error {
	if user.LinuxDOId == "" {
		return errors.New("linux do id is empty")
	}
	err := DB.Where("linux_do_id = ?", user.LinuxDOId).First(user).Error
	return err
}

func RootUserExists() bool {
	var user User
	err := DB.Where("role = ?", common.RoleRootUser).First(&user).Error
	if err != nil {
		return false
	}
	return true
}

// UserIsDistributor 是否具备分销商能力：is_distributor=1 且非管理员/超级管理员。
// 兼容尚未迁移的 role=5（启动迁移后会转为 role=1 + is_distributor=1）。
func UserIsDistributor(u *User) bool {
	if u == nil {
		return false
	}
	if u.Role >= common.RoleAdminUser {
		return false
	}
	if u.Role == common.RoleDistributorUser {
		return true
	}
	return u.IsDistributor == common.DistributorFlagYes
}
