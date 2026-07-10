package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id     int `json:"id"`
	UserId int `json:"user_id" gorm:"index"`
	// Username 列表接口填充，关联 users.username，仅 JSON 输出，不参与持久化（不使用 omitempty，便于前端始终拿到字段）
	Username       string  `json:"username" gorm:"-"`
	Amount         float64 `json:"amount" gorm:"type:decimal(20,6);default:0"`
	Money          float64 `json:"money"`
	InputAmount    float64 `json:"input_amount" gorm:"default:0"`
	InputCurrency  string  `json:"input_currency" gorm:"type:varchar(16);default:''"`
	PayCurrency    string  `json:"pay_currency" gorm:"type:varchar(16);default:''"`
	QuotaToAdd     int     `json:"quota_to_add" gorm:"default:0"`
	TradeNo        string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	DepositAddress string  `json:"deposit_address" gorm:"type:varchar(255);index"`
	PaymentMethod  string  `json:"payment_method" gorm:"type:varchar(50)"`
	CreateTime     int64   `json:"create_time"`
	CompleteTime   int64   `json:"complete_time"`
	Status         string  `json:"status"`
	InvoicedAmount float64 `json:"invoiced_amount" gorm:"type:decimal(20,6);default:0"`
	InvoiceEligible bool   `json:"invoice_eligible" gorm:"not null;default:true"`
}

func (topUp *TopUp) ResolveQuotaToAdd() int {
	if topUp == nil {
		return 0
	}
	if topUp.QuotaToAdd > 0 {
		return topUp.QuotaToAdd
	}
	switch strings.ToLower(strings.TrimSpace(topUp.PaymentMethod)) {
	case "stripe":
		return common.QuotaFromUSD(topUp.Money)
	case "creem":
		return int(topUpAmountDecimal(topUp.Amount).IntPart())
	default:
		dAmount := topUpAmountDecimal(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		return int(dAmount.Mul(dQuotaPerUnit).IntPart())
	}
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func topUpAmountDecimal(v float64) decimal.Decimal {
	return decimal.NewFromFloat(v)
}

func formatTopUpAmount(v float64) string {
	return topUpAmountDecimal(v).String()
}

func formatTopUpMoneyLogAmount(money float64, currency string) string {
	d := topUpAmountDecimal(money)
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "CNY":
		return fmt.Sprintf("¥%s", d.StringFixed(2))
	case "USD":
		return fmt.Sprintf("＄%s", d.StringFixed(2))
	default:
		return d.StringFixed(2)
	}
}

// formatTopUpCreditedDisplay 按订单实付/输入金额生成到账展示（与支付金额口径对齐，不再从 quota 反算）。
func formatTopUpCreditedDisplay(topUp *TopUp) string {
	if topUp == nil {
		return ""
	}
	inputCurrency := strings.ToUpper(strings.TrimSpace(topUp.InputCurrency))
	payCurrency := strings.ToUpper(strings.TrimSpace(topUp.PayCurrency))
	displayType := operation_setting.GetQuotaDisplayType()

	switch displayType {
	case operation_setting.QuotaDisplayTypeCNY:
		amount := topUp.Money
		if inputCurrency == "CNY" && topUp.InputAmount > 0 {
			amount = topUp.InputAmount
		} else if payCurrency == "USD" {
			amount = topUp.Money * operation_setting.USDExchangeRate
		}
		return fmt.Sprintf("¥%s 额度", topUpAmountDecimal(amount).StringFixed(2))
	case operation_setting.QuotaDisplayTypeCustom:
		rate := operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
		symbol := operation_setting.GetGeneralSetting().CustomCurrencySymbol
		if symbol == "" {
			symbol = "¤"
		}
		if rate <= 0 {
			rate = 1
		}
		amount := topUp.Money
		if topUp.InputAmount > 0 && inputCurrency != "" {
			amount = topUp.InputAmount
		} else if payCurrency == "USD" {
			amount = topUp.Money * rate
		} else if payCurrency == "CNY" {
			usdRate := operation_setting.USDExchangeRate
			if usdRate <= 0 {
				usdRate = operation_setting.Price
			}
			if usdRate > 0 {
				amount = topUp.Money / usdRate * rate
			}
		}
		return fmt.Sprintf("%s%s 额度", symbol, topUpAmountDecimal(amount).StringFixed(2))
	case operation_setting.QuotaDisplayTypeTokens:
		if topUp.QuotaToAdd > 0 {
			return fmt.Sprintf("%d 点额度", topUp.QuotaToAdd)
		}
		return logger.LogQuota(topUp.ResolveQuotaToAdd())
	default: // USD
		amount := topUp.Money
		if inputCurrency == "USD" && topUp.InputAmount > 0 {
			amount = topUp.InputAmount
		} else if payCurrency == "CNY" {
			usdRate := operation_setting.USDExchangeRate
			if usdRate <= 0 {
				usdRate = operation_setting.Price
			}
			if usdRate > 0 {
				amount = topUp.Money / usdRate
			}
		}
		return fmt.Sprintf("＄%s 额度", topUpAmountDecimal(amount).StringFixed(2))
	}
}

// FormatTopUpSuccessUserLog 充值成功写入用户日志（到账额度与支付金额均按订单实付展示，数值对齐）。
func FormatTopUpSuccessUserLog(prefix string, topUp *TopUp) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "使用在线充值成功"
	}
	if topUp == nil {
		return prefix
	}
	return fmt.Sprintf("%s，到账额度: %s，支付金额: %s",
		prefix,
		formatTopUpCreditedDisplay(topUp),
		formatTopUpMoneyLogAmount(topUp.Money, topUp.PayCurrency),
	)
}

// fillTopUpUsernamesWithDB 为充值记录批量填充关联用户名（管理员全平台列表与当前用户本人充值列表均使用）。
// 使用独立 Session 避免与同一事务上先查 top_ups 再查 users 时 GORM 语句状态串扰；Unscoped 以包含已软删除用户，保证历史订单仍能显示用户名。
func fillTopUpUsernamesWithDB(db *gorm.DB, topups []*TopUp) error {
	if len(topups) == 0 {
		return nil
	}
	idSet := make(map[int]struct{})
	for _, t := range topups {
		if t != nil && t.UserId > 0 {
			idSet[t.UserId] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return nil
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	type idName struct {
		Id       int    `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}
	var rows []idName
	// NewDB: 与 TopUp 查询复用同一 *gorm.DB 时，避免 Statement 残留导致用户表查询异常
	q := db.Session(&gorm.Session{NewDB: true}).Unscoped().Model(&User{}).Select("id", "username").Where("id IN ?", ids)
	if err := q.Scan(&rows).Error; err != nil {
		return err
	}
	nameByID := make(map[int]string, len(rows))
	for i := range rows {
		nameByID[rows[i].Id] = rows[i].Username
	}
	for _, t := range topups {
		if t != nil {
			t.Username = nameByID[t.UserId]
		}
	}
	return nil
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

// GetPendingUcoinTopUpByDepositAddress 按收款地址查找待支付的 U币订单（地址大小写不敏感）。
func GetPendingUcoinTopUpByDepositAddress(address string) *TopUp {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil
	}
	var topUp TopUp
	err := DB.Where(
		"payment_method = ? AND status = ? AND LOWER(deposit_address) = LOWER(?)",
		"ubcoin",
		common.TopUpStatusPending,
		address,
	).Order("id desc").First(&topUp).Error
	if err != nil {
		return nil
	}
	return &topUp
}

// GetPendingUcoinTopUpByUserId 查找用户最近一笔待支付的 U币订单。
func GetPendingUcoinTopUpByUserId(userId int) *TopUp {
	if userId <= 0 {
		return nil
	}
	var topUp TopUp
	err := DB.Where(
		"user_id = ? AND payment_method = ? AND status = ?",
		userId,
		"ubcoin",
		common.TopUpStatusPending,
	).Order("id desc").First(&topUp).Error
	if err != nil {
		return nil
	}
	return &topUp
}

func Recharge(referenceId string, customerId string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = topUp.ResolveQuotaToAdd()
		if quota <= 0 {
			return errors.New("无效的充值额度")
		}
		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(map[string]interface{}{"stripe_customer": customerId, "quota": gorm.Expr("quota + ?", quota)}).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordLog(topUp.UserId, LogTypeTopup, FormatTopUpSuccessUserLog("", topUp))

	ApplyAffiliateTopupReward(topUp.UserId, quota)
	return nil
}

// RechargeStripe 按 Stripe 回调完成充值，并在事务中校验渠道、金额和币种。
// 仅当订单为待支付且支付渠道为 stripe、回调金额与订单金额匹配、币种为 USD 时才会入账。
func RechargeStripe(referenceId string, customerId string, paidMoney float64, currency string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}
	if paidMoney <= 0 {
		return errors.New("无效的支付金额")
	}
	if strings.ToUpper(strings.TrimSpace(currency)) != "USD" {
		return errors.New("不支持的支付币种")
	}

	var quota int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}
		if topUp.PaymentMethod != "stripe" {
			return fmt.Errorf("支付渠道不匹配: expect stripe, got %s", topUp.PaymentMethod)
		}

		expectedMoney := decimal.NewFromFloat(topUp.Money)
		actualMoney := decimal.NewFromFloat(paidMoney)
		diff := expectedMoney.Sub(actualMoney).Abs()
		// 允许 1 美分误差，兼容支付平台与本地浮点换算的微小偏差
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			return fmt.Errorf("支付金额不匹配: expect %s, got %s", expectedMoney.StringFixed(2), actualMoney.StringFixed(2))
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = topUp.ResolveQuotaToAdd()
		if quota <= 0 {
			return errors.New("无效的充值额度")
		}
		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(map[string]interface{}{"stripe_customer": customerId, "quota": gorm.Expr("quota + ?", quota)}).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError(fmt.Sprintf("stripe topup verify failed, trade_no=%s, currency=%s, paid=%.2f, err=%s", referenceId, strings.ToUpper(strings.TrimSpace(currency)), paidMoney, err.Error()))
		return errors.New("充值失败，请稍后重试")
	}

	RecordLog(topUp.UserId, LogTypeTopup, FormatTopUpSuccessUserLog("", topUp))
	ApplyAffiliateTopupReward(topUp.UserId, int(quota))
	return nil
}

// topUpStatusFilterQuery 为充值列表查询追加 status 条件；空字符串或非法值不追加。
func topUpStatusFilterQuery(db *gorm.DB, status string) *gorm.DB {
	s := strings.TrimSpace(strings.ToLower(status))
	if s == "" {
		return db
	}
	switch s {
	case common.TopUpStatusPending, common.TopUpStatusSuccess, common.TopUpStatusFailed, common.TopUpStatusExpired:
		return db.Where("status = ?", s)
	default:
		return db
	}
}

// applyTopUpTradeNoLike 按订单号关键字模糊筛选 trade_no；keyword 为空时不追加条件。
func applyTopUpTradeNoLike(db *gorm.DB, keyword string) *gorm.DB {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return db
	}
	like := "%%" + kw + "%%"
	return db.Where("trade_no LIKE ?", like)
}

// applyTopUpUsernameJoin 管理员全平台列表按用户名模糊筛选：INNER JOIN users（避免 IN 子查询在部分驱动下的兼容问题）。
func applyTopUpUsernameJoin(db *gorm.DB, usernameKeyword string) *gorm.DB {
	u := strings.TrimSpace(usernameKeyword)
	if u == "" {
		return db
	}
	like := "%%" + u + "%%"
	return db.Joins("INNER JOIN users ON users.id = top_ups.user_id AND users.username LIKE ?", like)
}

// GetUserTopUps 分页返回指定用户的充值订单；statusFilter、tradeNoKeyword 为空时不对对应维度筛选。
func GetUserTopUps(userId int, pageInfo *common.PageInfo, statusFilter, tradeNoKeyword string) (topups []*TopUp, total int64, err error) {
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

	userScope := func(db *gorm.DB) *gorm.DB {
		q := db.Where("user_id = ?", userId)
		q = topUpStatusFilterQuery(q, statusFilter)
		q = applyTopUpTradeNoLike(q, tradeNoKeyword)
		return q
	}

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Scopes(userScope).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Model(&TopUp{}).Scopes(userScope).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = fillTopUpUsernamesWithDB(tx, topups); err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用）；各筛选为空时不追加对应条件。
func GetAllTopUps(pageInfo *common.PageInfo, statusFilter, tradeNoKeyword, usernameKeyword string) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	allScope := func(db *gorm.DB) *gorm.DB {
		q := topUpStatusFilterQuery(db, statusFilter)
		q = applyTopUpTradeNoLike(q, tradeNoKeyword)
		q = applyTopUpUsernameJoin(q, usernameKeyword)
		return q
	}

	if err = tx.Model(&TopUp{}).Scopes(allScope).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Model(&TopUp{}).Scopes(allScope).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = fillTopUpUsernamesWithDB(tx, topups); err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值。
// adminUsername 会写入使用日志详情，便于审计是谁执行了补单。
func ManualCompleteTopUp(tradeNo string, adminUsername string) error {
	if tradeNo == "" {
		return errors.New("未提供订单号")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var userId int
	var quotaToAdd int
	var logTopUp TopUp

	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		// 行级锁，避免并发补单
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("订单状态不是待支付，无法补单")
		}

		quotaToAdd = topUp.ResolveQuotaToAdd()
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		// 标记完成
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		// 增加用户额度（立即写库，保持一致性）
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		userId = topUp.UserId
		logTopUp = *topUp
		return nil
	})

	if err != nil {
		return err
	}

	// 事务外记录日志，避免阻塞
	if userId > 0 && quotaToAdd > 0 {
		operatorPrefix := "管理员"
		if adminUsername != "" {
			operatorPrefix = adminUsername
		}
		RecordLog(userId, LogTypeTopup, FormatTopUpSuccessUserLog(operatorPrefix+"管理员补单成功", &logTopUp))
		ApplyAffiliateTopupReward(userId, quotaToAdd)
	}
	return nil
}

// CreateCorporateTopUp 管理员对公入账：创建可开票充值订单并增加用户充值余额（非赠送）。
func CreateCorporateTopUp(userID int, money float64, quota int, reference, remark string, adminID int) (*TopUp, error) {
	if userID <= 0 || money <= 0 || quota <= 0 {
		return nil, errors.New("invalid corporate topup")
	}
	reference = strings.TrimSpace(reference)
	tradeNo := fmt.Sprintf("CORP-%d-%d-%s", userID, common.GetTimestamp(), strings.ToUpper(common.GetRandomString(6)))
	topUp := &TopUp{
		UserId:          userID,
		Money:           money,
		Amount:          money,
		QuotaToAdd:      quota,
		TradeNo:         tradeNo,
		PaymentMethod:   TopUpPaymentMethodCorporate,
		CreateTime:      common.GetTimestamp(),
		CompleteTime:    common.GetTimestamp(),
		Status:          common.TopUpStatusSuccess,
		InvoiceEligible: true,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(topUp).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", userID).Update("quota", gorm.Expr("quota + ?", quota)).Error
	})
	if err != nil {
		return nil, err
	}
	adminNote := strings.TrimSpace(remark)
	logMsg := fmt.Sprintf("对公入账成功，金额: %.2f，到账额度: %s", money, logger.FormatQuota(quota))
	if reference != "" {
		logMsg = fmt.Sprintf("对公入账成功，凭证号: %s，金额: %.2f，到账额度: %s", reference, money, logger.FormatQuota(quota))
	}
	if adminNote != "" {
		logMsg += "，备注: " + adminNote
	}
	if adminID > 0 {
		logMsg += fmt.Sprintf("（操作人ID: %d）", adminID)
	}
	RecordLog(userID, LogTypeTopup, logMsg)
	ApplyAffiliateTopupReward(userID, quota)
	return topUp, nil
}

func RechargeCreem(referenceId string, customerEmail string, customerName string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int64
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = int64(topUp.ResolveQuotaToAdd())
		if quota <= 0 {
			return errors.New("无效的充值额度")
		}

		// 构建更新字段，优先使用邮箱，如果邮箱为空则使用用户名
		updateFields := map[string]interface{}{
			"quota": gorm.Expr("quota + ?", quota),
		}

		// 如果有客户邮箱，尝试更新用户邮箱（仅当用户邮箱为空时）
		if customerEmail != "" {
			// 先检查用户当前邮箱是否为空
			var user User
			err = tx.Where("id = ?", topUp.UserId).First(&user).Error
			if err != nil {
				return err
			}

			// 如果用户邮箱为空，则更新为支付时使用的邮箱
			if user.Email == "" {
				updateFields["email"] = customerEmail
			}
		}

		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(updateFields).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordLog(topUp.UserId, LogTypeTopup, FormatTopUpSuccessUserLog("使用Creem充值成功", topUp))

	ApplyAffiliateTopupReward(topUp.UserId, int(quota))
	return nil
}

// RechargeUcoin 按 U币（虚拟币）回调完成充值，幂等处理。
// U币与美元固定 1:1：回调 amount 即为到账美元额度（如 0.3 U => $0.3）。
func RechargeUcoin(tradeNo string, actualAmount string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}
	actualAmount = strings.TrimSpace(actualAmount)
	if actualAmount == "" {
		return errors.New("未提供实际充值金额")
	}
	dActualAmount, err := decimal.NewFromString(actualAmount)
	if err != nil {
		return errors.New("实际充值金额格式错误")
	}
	if !dActualAmount.IsPositive() {
		return errors.New("实际充值金额必须大于 0")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil // 幂等：已成功直接返回
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		if topUp.PaymentMethod != "ubcoin" {
			return fmt.Errorf("支付渠道不匹配: expect ubcoin, got %s", topUp.PaymentMethod)
		}

		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd = int(dActualAmount.Mul(dQuotaPerUnit).Round(0).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		actualAmountFloat, _ := dActualAmount.Float64()
		topUp.Amount = actualAmountFloat
		topUp.Money = actualAmountFloat
		topUp.InputAmount = actualAmountFloat
		topUp.InputCurrency = "USD"
		topUp.PayCurrency = "USD"
		topUp.QuotaToAdd = quotaToAdd
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("ubcoin topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordLog(topUp.UserId, LogTypeTopup, FormatTopUpSuccessUserLog("U币充值成功", topUp))
		ApplyAffiliateTopupReward(topUp.UserId, quotaToAdd)
	}

	return nil
}

func RechargeWaffo(tradeNo string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil // 幂等：已成功直接返回
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		quotaToAdd = topUp.ResolveQuotaToAdd()
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordLog(topUp.UserId, LogTypeTopup, FormatTopUpSuccessUserLog("Waffo充值成功", topUp))
		ApplyAffiliateTopupReward(topUp.UserId, quotaToAdd)
	}

	return nil
}
