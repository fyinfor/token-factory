package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"gorm.io/gorm"
)

// AffInviteRelation 邀请人与被邀请人关系表：为每个被邀请人单独配置充值分销比例。
// CommissionRatioBps 存储单位为万分之一（相对于「百分比」）：1 表示 0.01%，100 表示 1%，10000 表示 100%。
type AffInviteRelation struct {
	Id                    int `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterId             int `json:"inviter_id" gorm:"not null;uniqueIndex:idx_aff_inv_pair"`
	InviteeUserId         int `json:"invitee_user_id" gorm:"not null;uniqueIndex:idx_aff_inv_pair;column:invitee_user_id"`
	CommissionRatioBps    int `json:"commission_ratio_bps" gorm:"not null;default:0;column:commission_ratio_bps"`
	CommissionEarnedQuota int `json:"commission_earned_quota" gorm:"not null;default:0;column:commission_earned_quota"` // 该被邀请人为邀请人累计贡献的分销额度（与 aff_quota 增量一致）
	// ProfitShareEarnedQuota 利润分成模式下，该被邀请人用量加价切片累计为邀请人贡献的收益（与 aff_quota 中对应增量一致；与 commission_earned_quota 分列统计）。
	ProfitShareEarnedQuota int `json:"profit_share_earned_quota" gorm:"not null;default:0;column:profit_share_earned_quota"`
	// 被邀请用户模型加价折扣率：JSON 数组 [{model_name, channel_id, markup_discount_rate}]，仅存与渠道默认不同的项。
	ModelMarkupDiscountRate string `json:"model_markup_discount_rate" gorm:"type:text;column:model_markup_discount_rate;comment:被邀请用户模型加价折扣率(JSON数组)"`
	// 自动时间戳：创建/更新时 GORM 自动赋值
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime;bigint;comment:创建时间"`
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime;bigint;comment:更新时间"`
}

func (AffInviteRelation) TableName() string {
	return "aff_invite_relations"
}

const maxAffiliateCommissionBps = 10000
const maxInviteeSaleRatePercent = 200.0

// AffInviteeListItem 邀请人视角下的被邀请人列表项
type AffInviteeListItem struct {
	InviteeId              int    `json:"invitee_id"`
	Username               string `json:"username"`
	DisplayName            string `json:"display_name"`
	CommissionRatioBps     int    `json:"commission_ratio_bps"` // 万分之一单位（1=0.01%），前端展示为百分比
	CommissionEarnedQuota  int    `json:"commission_earned_quota"`
	ProfitShareEarnedQuota int    `json:"profit_share_earned_quota"`
	CreatedAt              int64  `json:"created_at"` // 邀请关系建立时间（aff_invite_relations.created_at）
}

func defaultCommissionBpsForNewInviteRelation(inviterId int) int {
	var inviter User
	err := DB.Select("id", "role", "distributor_commission_bps", "is_distributor").Where("id = ?", inviterId).First(&inviter).Error
	if err != nil {
		return common.AffiliateDefaultCommissionBps
	}
	if UserIsDistributor(&inviter) && inviter.DistributorCommissionBps > 0 {
		return inviter.DistributorCommissionBps
	}
	return common.AffiliateDefaultCommissionBps
}

func defaultModelMarkupDiscountRateForNewInviteRelation(db *gorm.DB, inviterId int) string {
	if db == nil || inviterId <= 0 {
		return "[]"
	}
	var inviter User
	err := db.Select("id", "role", "is_distributor", "distributor_model_markup_discount_template", "distributor_model_discount_auto_apply").
		Where("id = ?", inviterId).First(&inviter).Error
	if err != nil || !UserIsDistributor(&inviter) || inviter.DistributorModelDiscountAutoApply != common.DistributorFlagYes {
		return "[]"
	}
	raw := strings.TrimSpace(inviter.DistributorModelMarkupDiscountTemplate)
	if raw == "" {
		return "[]"
	}
	return raw
}

// EnsureAffInviteRelation 注册成功后建立关系行，比例初始为系统默认或分销商单独默认。
func EnsureAffInviteRelation(inviterId, inviteeUserId int) error {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return nil
	}
	var cnt int64
	err := DB.Model(&AffInviteRelation{}).Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).Count(&cnt).Error
	if err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	ts := common.GetTimestamp()
	bps := defaultCommissionBpsForNewInviteRelation(inviterId)
	modelMarkupDiscountRate := defaultModelMarkupDiscountRateForNewInviteRelation(DB, inviterId)
	rel := AffInviteRelation{
		InviterId:               inviterId,
		InviteeUserId:           inviteeUserId,
		CommissionRatioBps:      bps,
		CommissionEarnedQuota:   0,
		ProfitShareEarnedQuota:  0,
		ModelMarkupDiscountRate: modelMarkupDiscountRate,
		CreatedAt:               ts,
		UpdatedAt:               ts,
	}
	return DB.Create(&rel).Error
}

// BackfillAffInviteRelationsIfNeeded 表为空时执行一次历史数据补全，避免每次启动全表扫描。
func BackfillAffInviteRelationsIfNeeded() error {
	var cnt int64
	if err := DB.Model(&AffInviteRelation{}).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	return BackfillAffInviteRelationsFromUsers()
}

// BackfillAffInviteRelationsFromUsers 为历史数据补全关系行。
func BackfillAffInviteRelationsFromUsers() error {
	var users []User
	err := DB.Unscoped().Model(&User{}).Select("id", "inviter_id").Where("inviter_id > ?", 0).Find(&users).Error
	if err != nil {
		return err
	}
	for i := range users {
		if err := EnsureAffInviteRelation(users[i].InviterId, users[i].Id); err != nil {
			common.SysError("backfill aff_invite_relations: " + err.Error())
		}
	}
	return nil
}

// EffectiveAffiliateCommissionBps 计算邀请人对某一被邀请人生效的分销比例（万分之一）。
// 与充值分成、利润分成（加价切片分润）共用同一套优先级：分销商账号 distributor_commission_bps > 0 优先，
// 否则 aff_invite_relations.commission_ratio_bps（>0），否则系统 AffiliateDefaultCommissionBps。
func EffectiveAffiliateCommissionBps(inviter *User, inviteeUserId int) int {
	return effectiveAffiliateCommissionBps(inviter, inviteeUserId)
}

// effectiveAffiliateCommissionBps（内部）：充值与利润分成逻辑一致。
func effectiveAffiliateCommissionBps(inviter *User, inviteeUserId int) int {
	if inviter == nil || inviter.Id <= 0 {
		return common.AffiliateDefaultCommissionBps
	}
	if UserIsDistributor(inviter) && inviter.DistributorCommissionBps > 0 {
		bps := inviter.DistributorCommissionBps
		if bps > maxAffiliateCommissionBps {
			bps = maxAffiliateCommissionBps
		}
		return bps
	}
	var rel AffInviteRelation
	err := DB.Where("inviter_id = ? AND invitee_user_id = ?", inviter.Id, inviteeUserId).First(&rel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.AffiliateDefaultCommissionBps
		}
		common.SysError("effectiveAffiliateCommissionBps: " + err.Error())
		return common.AffiliateDefaultCommissionBps
	}
	if rel.CommissionRatioBps <= 0 {
		return common.AffiliateDefaultCommissionBps
	}
	return rel.CommissionRatioBps
}

// ApplyAffiliateTopupReward 被邀请用户获得充值额度 quotaAdded 后，按 effectiveAffiliateCommissionBps 将提成记入邀请人 aff_quota / aff_history（不增加 quota）。
// 须在支付回调完成入账后调用，与订单事务解耦。
func ApplyAffiliateTopupReward(inviteeUserId int, quotaAdded int) {
	if common.IsDistributorProfitShareMode() {
		return
	}
	if inviteeUserId <= 0 || quotaAdded <= 0 {
		return
	}
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil {
		return
	}
	inviterId := invitee.InviterId
	if inviterId <= 0 {
		return
	}
	inviterUser, errInv := GetUserById(inviterId, false)
	if errInv != nil || !UserIsDistributor(inviterUser) {
		return
	}
	bps := effectiveAffiliateCommissionBps(inviterUser, inviteeUserId)
	if bps <= 0 {
		return
	}
	if bps > maxAffiliateCommissionBps {
		bps = maxAffiliateCommissionBps
	}
	reward := int(int64(quotaAdded) * int64(bps) / int64(maxAffiliateCommissionBps))
	if reward <= 0 {
		return
	}
	if err := IncreaseUserAffCommissionQuota(inviterId, reward); err != nil {
		common.SysError(fmt.Sprintf("ApplyAffiliateTopupReward: inviter=%d invitee=%d reward=%d err=%v", inviterId, inviteeUserId, reward, err))
		return
	}
	if err := InsertAffInviteCommissionLog(inviterId, inviteeUserId, quotaAdded, bps, reward); err != nil {
		common.SysError(fmt.Sprintf("ApplyAffiliateTopupReward commission log: inviter=%d invitee=%d err=%v", inviterId, inviteeUserId, err))
	}
	if err := DB.Model(&AffInviteRelation{}).
		Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).
		UpdateColumn("commission_earned_quota", gorm.Expr("commission_earned_quota + ?", reward)).Error; err != nil {
		common.SysError(fmt.Sprintf("ApplyAffiliateTopupReward update earned: inviter=%d invitee=%d err=%v", inviterId, inviteeUserId, err))
	}
	inviteeLabel := strings.TrimSpace(invitee.Username)
	if inviteeLabel == "" {
		inviteeLabel = strings.TrimSpace(invitee.DisplayName)
	}
	if inviteeLabel == "" {
		inviteeLabel = fmt.Sprintf("ID:%d", invitee.Id)
	}
	pct := logger.FormatCommissionRatioAsPercent(bps)
	amt := logger.LogQuotaConcise(reward)
	RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请分销奖励（被邀请用户 %s 充值）%s，分成比例 %s", inviteeLabel, amt, pct))
}

// ListAffInvitees 分页返回当前用户邀请注册的用户（含关系表累计分成等；单笔明细见 aff_invite_commission_logs）。
// keyword 非空时按用户名、显示名模糊匹配；若 keyword 为十进制正整数则同时匹配被邀请用户 id。
func ListAffInvitees(inviterId int, keyword string, pageInfo *common.PageInfo) ([]AffInviteeListItem, int64, error) {
	if inviterId <= 0 {
		return nil, 0, errors.New("invalid inviter")
	}
	kw := strings.TrimSpace(keyword)
	inviteesScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("inviter_id = ?", inviterId)
		if kw != "" {
			pattern := "%" + kw + "%"
			if uid, err := strconv.Atoi(kw); err == nil && uid > 0 {
				db = db.Where("(id = ? OR username LIKE ? OR display_name LIKE ?)", uid, pattern, pattern)
			} else {
				db = db.Where("(username LIKE ? OR display_name LIKE ?)", pattern, pattern)
			}
		}
		return db
	}

	var total int64
	if err := DB.Model(&User{}).Scopes(inviteesScope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []User
	err := DB.Model(&User{}).Scopes(inviteesScope).
		Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	if len(users) == 0 {
		return []AffInviteeListItem{}, total, nil
	}
	ids := make([]int, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.Id)
	}
	var rels []AffInviteRelation
	_ = DB.Where("inviter_id = ? AND invitee_user_id IN ?", inviterId, ids).Find(&rels).Error
	bpsMap := make(map[int]int, len(rels))
	earnedMap := make(map[int]int, len(rels))
	profitEarnedMap := make(map[int]int, len(rels))
	relCreatedMap := make(map[int]int64, len(rels))
	for _, r := range rels {
		bpsMap[r.InviteeUserId] = r.CommissionRatioBps
		earnedMap[r.InviteeUserId] = r.CommissionEarnedQuota
		profitEarnedMap[r.InviteeUserId] = r.ProfitShareEarnedQuota
		relCreatedMap[r.InviteeUserId] = r.CreatedAt
	}
	defaultBps := common.AffiliateDefaultCommissionBps
	items := make([]AffInviteeListItem, 0, len(users))
	for _, u := range users {
		bps, ok := bpsMap[u.Id]
		if !ok {
			bps = defaultBps
		} else if bps <= 0 {
			bps = defaultBps
		}
		earned := earnedMap[u.Id]
		profitEarned := profitEarnedMap[u.Id]
		relAt := relCreatedMap[u.Id]
		items = append(items, AffInviteeListItem{
			InviteeId:              u.Id,
			Username:               u.Username,
			DisplayName:            u.DisplayName,
			CommissionRatioBps:     bps,
			CommissionEarnedQuota:  earned,
			ProfitShareEarnedQuota: profitEarned,
			CreatedAt:              relAt,
		})
	}
	return items, total, nil
}

func CountAffInvitees(inviterId int) (int64, error) {
	if inviterId <= 0 {
		return 0, errors.New("invalid inviter")
	}
	var total int64
	err := DB.Model(&User{}).Where("inviter_id = ?", inviterId).Count(&total).Error
	return total, err
}

// UpdateAffInviteeCommission 邀请人修改某一被邀请人的分销比例（验证被邀请人确实属于当前邀请人）。
func UpdateAffInviteeCommission(inviterId, inviteeUserId, commissionBps int) error {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return errors.New("invalid id")
	}
	if commissionBps < 0 || commissionBps > maxAffiliateCommissionBps {
		return fmt.Errorf("commission_ratio_bps must be 0..%d (万分之一单位，1=0.01%%)", maxAffiliateCommissionBps)
	}
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil {
		return errors.New("user not found")
	}
	if invitee.InviterId != inviterId {
		return errors.New("not your invitee")
	}
	ts := common.GetTimestamp()
	var rel AffInviteRelation
	err = DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).First(&rel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rel = AffInviteRelation{
			InviterId:              inviterId,
			InviteeUserId:          inviteeUserId,
			CommissionRatioBps:     commissionBps,
			ProfitShareEarnedQuota: 0,
			CreatedAt:              ts,
			UpdatedAt:              ts,
		}
		return DB.Create(&rel).Error
	}
	if err != nil {
		return err
	}
	rel.CommissionRatioBps = commissionBps
	rel.UpdatedAt = ts
	return DB.Save(&rel).Error
}

// UnbindDistributorInvitee 解除分销商与下级用户的邀请绑定，写入解绑快照，不回滚历史流水和已产生收益。
func UnbindDistributorInvitee(inviterId, inviteeUserId, operatorId int, reason string) error {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return errors.New("invalid id")
	}
	inviter, err := GetUserById(inviterId, false)
	if err != nil || inviter == nil || !UserIsDistributor(inviter) {
		return errors.New("user is not distributor")
	}
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil || invitee == nil {
		return errors.New("user not found")
	}
	if invitee.InviterId != inviterId {
		return errors.New("not this distributor's invitee")
	}
	operatorUsername := ""
	if operatorId > 0 {
		if operator, err := GetUserById(operatorId, false); err == nil && operator != nil {
			operatorUsername = operator.Username
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var rel AffInviteRelation
		err := tx.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).First(&rel).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		log := DistributorInviteeUnbindLog{
			DistributorId:           inviterId,
			InviteeUserId:           inviteeUserId,
			OperatorId:              operatorId,
			InviteeUsername:         invitee.Username,
			InviteeDisplayName:      invitee.DisplayName,
			OperatorUsername:        operatorUsername,
			Reason:                  strings.TrimSpace(reason),
			CommissionRatioBps:      rel.CommissionRatioBps,
			CommissionEarnedQuota:   rel.CommissionEarnedQuota,
			ProfitShareEarnedQuota:  rel.ProfitShareEarnedQuota,
			ModelMarkupDiscountRate: rel.ModelMarkupDiscountRate,
			CreatedAt:               common.GetTimestamp(),
		}
		if log.CommissionRatioBps <= 0 {
			log.CommissionRatioBps = defaultCommissionBpsForNewInviteRelation(inviterId)
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}
		res := tx.Model(&User{}).
			Where("id = ? AND inviter_id = ?", inviteeUserId, inviterId).
			Update("inviter_id", 0)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("invitee binding changed")
		}
		if err := tx.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).
			Delete(&AffInviteRelation{}).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", inviterId).
			UpdateColumn("aff_count", gorm.Expr("CASE WHEN aff_count > 0 THEN aff_count - ? ELSE 0 END", 1)).
			Error
	})
}

// InviteeModelMarkupDiscountRateItem 定价页可见的模型×渠道加价折扣配置项（API 列表元素）。
type InviteeModelMarkupDiscountRateItem struct {
	ModelName                      string                            `json:"model_name"`
	ChannelID                      int                               `json:"channel_id"`
	ChannelPath                    string                            `json:"channel_path"` // 与定价页通道列表「复制」一致：model/route_slug 或 alias/model/channel_no
	SupplierType                   string                            `json:"supplier_type"`
	ChannelName                    string                            `json:"channel_name"`
	DefaultMarkupDiscountRate      float64                           `json:"default_markup_discount_rate"`      // 渠道默认官方价加价折扣率(%)
	CurrentMarkupDiscountRate      float64                           `json:"current_markup_discount_rate"`      // 对该被邀请用户生效的加价折扣率(%)
	OfficialCurrentDiscountPercent float64                           `json:"official_current_discount_percent"` // 定价首页当前展示折扣(%)
	ChannelPriceDiscountPercent    float64                           `json:"channel_price_discount_percent"`    // 最终成本率(%)
	OfficialBasePrice              float64                           `json:"official_base_price"`               // 官方根价/倍率
	ChannelBasePrice               float64                           `json:"channel_base_price"`                // 渠道原始价/倍率
	PricingQuotaType               int                               `json:"pricing_quota_type"`                // 当前渠道计费类型：0=按量，1=按次，3=阶梯
	OfficialPricingQuotaType       int                               `json:"-"`                                 // 全局官方计费类型，仅供导出：0=按量，1=按次，3=阶梯
	OfficialModelRatio             float64                           `json:"-"`                                 // 全局模型输入倍率，仅供媒体价格导出
	OfficialCompletionRatio        float64                           `json:"-"`                                 // 全局输出倍率，仅供按量官方价导出
	OfficialCacheRatio             *float64                          `json:"-"`                                 // 全局缓存读取倍率，仅供按量官方价导出
	OfficialCreateCacheRatio       *float64                          `json:"-"`                                 // 全局缓存创建倍率，仅供按量官方价导出
	OfficialImageRatio             *float64                          `json:"-"`                                 // 全局图片输入倍率，仅供导出
	OfficialImagePrice             *float64                          `json:"-"`                                 // 全局图片按张价，仅供导出
	OfficialImagePricingRules      *ratio_setting.ImagePricingRules  `json:"-"`                                 // 全局图片分辨率按张价，仅供导出
	OfficialVideoRatio             *float64                          `json:"-"`                                 // 全局视频输入倍率，仅供导出
	OfficialVideoCompletionRatio   float64                           `json:"-"`                                 // 全局视频输出倍率，仅供导出
	OfficialVideoPrice             *float64                          `json:"-"`                                 // 全局视频按条价，仅供导出
	OfficialVideoPricingRules      *ratio_setting.VideoPricingRules  `json:"-"`                                 // 全局视频专用规则，仅供导出
	OfficialRequestTierPricing     *ratio_setting.RequestTierPricing `json:"-"`                                 // 全局官方阶梯价，仅供导出
}

// inviteePricingChannelPath 与前端 ModelChannelList 复制通道路径格式一致。
func inviteePricingChannelPath(modelName string, ch PricingChannelItem) string {
	modelName = strings.TrimSpace(modelName)
	if slug := strings.TrimSpace(ch.RouteSlug); slug != "" && modelName != "" {
		return modelName + "/" + slug
	}
	return strings.TrimSpace(ch.SupplierAlias) + "/" + modelName + "/" + strings.TrimSpace(ch.ChannelNo)
}

type inviteeModelMarkupDiscountRateEntry struct {
	ModelName          string  `json:"model_name"`
	ChannelID          int     `json:"channel_id"`
	MarkupDiscountRate float64 `json:"markup_discount_rate"`
}

type inviteeModelMarkupDiscountRateEntryRaw struct {
	ModelName          string   `json:"model_name"`
	ChannelID          int      `json:"channel_id"`
	MarkupDiscountRate *float64 `json:"markup_discount_rate"`
	Discount           *float64 `json:"discount"`
}

func inviteeModelMarkupKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, strings.TrimSpace(modelName))
}

func parseInviteeModelMarkupDiscountRates(raw string) ([]inviteeModelMarkupDiscountRateEntry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "{}" {
		return nil, nil
	}
	var list []inviteeModelMarkupDiscountRateEntryRaw
	if err := common.UnmarshalJsonStr(raw, &list); err != nil {
		var legacy map[string]float64
		if legacyErr := common.UnmarshalJsonStr(raw, &legacy); legacyErr != nil {
			return nil, err
		}
		out := make([]inviteeModelMarkupDiscountRateEntry, 0, len(legacy))
		for key, rate := range legacy {
			key = strings.TrimSpace(key)
			modelName := ""
			channelID := 0
			if idx := strings.LastIndex(key, "/"); idx > 0 && idx < len(key)-1 {
				modelName = strings.TrimSpace(key[:idx])
				channelID, _ = strconv.Atoi(strings.TrimSpace(key[idx+1:]))
			}
			if channelID <= 0 {
				parts := strings.SplitN(key, ":", 2)
				if len(parts) == 2 {
					channelID, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
					modelName = strings.TrimSpace(parts[1])
				}
			}
			if channelID <= 0 || modelName == "" {
				continue
			}
			out = append(out, inviteeModelMarkupDiscountRateEntry{
				ModelName:          modelName,
				ChannelID:          channelID,
				MarkupDiscountRate: rate,
			})
		}
		return out, nil
	}
	out := make([]inviteeModelMarkupDiscountRateEntry, 0, len(list))
	for _, item := range list {
		modelName := strings.TrimSpace(item.ModelName)
		if item.ChannelID <= 0 || modelName == "" {
			continue
		}
		rate := 0.0
		if item.MarkupDiscountRate != nil {
			rate = *item.MarkupDiscountRate
		} else if item.Discount != nil {
			rate = *item.Discount
		}
		out = append(out, inviteeModelMarkupDiscountRateEntry{
			ModelName:          modelName,
			ChannelID:          item.ChannelID,
			MarkupDiscountRate: rate,
		})
	}
	return out, nil
}

func inviteeModelMarkupDiscountRateMap(list []inviteeModelMarkupDiscountRateEntry) map[string]float64 {
	m := make(map[string]float64, len(list))
	for _, item := range list {
		m[inviteeModelMarkupKey(item.ChannelID, item.ModelName)] = item.MarkupDiscountRate
	}
	return m
}

func inviteeModelActualDiscountPercent(officialBase, channelBase, costDiscountPercent, markupDiscountRate float64) float64 {
	if officialBase <= 0 || !isFiniteFloat(officialBase) || !isFiniteFloat(channelBase) {
		return 0
	}
	effective := channelBase*costDiscountPercent/100 + officialBase*markupDiscountRate/100
	if !isFiniteFloat(effective) || effective >= officialBase {
		return 0
	}
	discount := math.Round((1 - effective/officialBase) * 100)
	if discount < 0 {
		return 0
	}
	return discount
}

func isFiniteFloat(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func inviteeTierDiscountPercent(current, official float64) float64 {
	if official <= 0 || !isFiniteFloat(current) || !isFiniteFloat(official) || current >= official {
		return 0
	}
	discount := math.Round((1 - current/official) * 100)
	if discount < 0 {
		return 0
	}
	return discount
}

func inviteeTierPreviewBases(current, official, markupDiscountRate float64) (float64, float64) {
	costPart := current - official*markupDiscountRate/100
	if !isFiniteFloat(costPart) || costPart < 0 {
		costPart = 0
	}
	return official, costPart
}

func bestVideoFlatClipDiscountBases(hint *VideoFlatClipPricingHint, markupDiscountRate float64) (float64, float64, float64, bool) {
	if hint == nil {
		return 0, 0, 0, false
	}
	bestDiscount := 0.0
	bestCurrent := 0.0
	bestOfficial := 0.0
	for _, row := range hint.Tiers {
		discount := inviteeTierDiscountPercent(row.UsdAfterChannelDiscount, row.UsdOfficial)
		if discount > bestDiscount {
			bestDiscount = discount
			bestCurrent = row.UsdAfterChannelDiscount
			bestOfficial = row.UsdOfficial
		}
	}
	if bestDiscount <= 0 || bestOfficial <= 0 {
		return 0, 0, 0, false
	}
	officialBase, channelCostPart := inviteeTierPreviewBases(bestCurrent, bestOfficial, markupDiscountRate)
	return officialBase, channelCostPart, bestDiscount, true
}

func bestImagePerImageDiscountBases(hint *ImagePerImagePricingHint, markupDiscountRate float64) (float64, float64, float64, bool) {
	if hint == nil {
		return 0, 0, 0, false
	}
	bestDiscount := 0.0
	bestCurrent := 0.0
	bestOfficial := 0.0
	for _, row := range hint.Tiers {
		discount := inviteeTierDiscountPercent(row.UsdAfterChannelDiscount, row.UsdOfficial)
		if discount > bestDiscount {
			bestDiscount = discount
			bestCurrent = row.UsdAfterChannelDiscount
			bestOfficial = row.UsdOfficial
		}
	}
	if bestDiscount <= 0 || bestOfficial <= 0 {
		return 0, 0, 0, false
	}
	officialBase, channelCostPart := inviteeTierPreviewBases(bestCurrent, bestOfficial, markupDiscountRate)
	return officialBase, channelCostPart, bestDiscount, true
}

func officialVideoPricingRulesForExport(modelName string) *ratio_setting.VideoPricingRules {
	rules, ok := ratio_setting.GetVideoPricingRules(modelName)
	if !ok || (!ratio_setting.HasUsableVideoPerTokenRules(rules) &&
		!ratio_setting.HasUsableVideoPerSecondRules(rules) &&
		!ratio_setting.HasUsableVideoPerVideoRules(rules)) {
		return nil
	}
	rulesCopy := rules
	return &rulesCopy
}

func listPricingVisibleMarkupDiscountRateItems() ([]InviteeModelMarkupDiscountRateItem, map[string]float64, error) {
	pricing := GetPricing()
	filtered := make([]Pricing, 0, len(pricing))
	for _, p := range pricing {
		if ratio_setting.ModelHasConfiguredPricing(p.ModelName) {
			filtered = append(filtered, p)
		}
	}
	pricingChannels, err := ListChannelsForPricing()
	if err != nil {
		return nil, nil, err
	}
	visibleChannelIDs := make(map[int]struct{}, len(pricingChannels))
	channelNames := make(map[int]string, len(pricingChannels))
	channelSupplierTypes := make(map[int]string, len(pricingChannels))
	for _, pch := range pricingChannels {
		visibleChannelIDs[pch.ChannelID] = struct{}{}
		channelNames[pch.ChannelID] = strings.TrimSpace(pch.ChannelName)
		channelSupplierTypes[pch.ChannelID] = strings.TrimSpace(pch.SupplierType)
	}
	metas, err := ListChannelPricingMeta()
	if err != nil {
		return nil, nil, err
	}
	pricingItems := BuildPricingAPIItems(filtered, visibleChannelIDs, metas, false)

	items := make([]InviteeModelMarkupDiscountRateItem, 0, len(pricingItems))
	defaultRates := make(map[string]float64, len(pricingItems))
	for _, p := range pricingItems {
		if len(p.ChannelList) == 0 {
			continue
		}
		ch := p.ChannelList[0]
		modelName := strings.TrimSpace(p.ModelName)
		if modelName == "" || ch.ChannelID <= 0 {
			continue
		}
		key := inviteeModelMarkupKey(ch.ChannelID, modelName)
		if _, exists := defaultRates[key]; exists {
			continue
		}
		defaultRate := ch.MarkupDiscountRate
		quotaType := p.QuotaType
		if ch.QuotaType == 1 {
			quotaType = 1
		}
		officialBaseForDiscount := p.ModelRatio
		channelBase := ch.ModelRatio
		if quotaType == 1 {
			officialBaseForDiscount = p.ModelPrice
			channelBase = ch.ModelPrice
		}
		officialBase := p.ModelRatio
		if p.QuotaType == 1 {
			officialBase = p.ModelPrice
		}
		var officialTierPricing *ratio_setting.RequestTierPricing
		if p.QuotaType == 3 {
			if rule, ok := ratio_setting.GetModelRequestTierPricing(modelName); ok && len(rule.Tiers) > 0 {
				ruleCopy := rule
				officialTierPricing = &ruleCopy
			}
		}
		var officialImagePrice *float64
		if price, ok := ratio_setting.GetImagePrice(modelName); ok && price > 0 {
			priceCopy := price
			officialImagePrice = &priceCopy
		}
		var officialImagePricingRules *ratio_setting.ImagePricingRules
		if rules, ok := ratio_setting.GetImagePricingRules(modelName); ok && ratio_setting.HasUsableImagePerImageRules(rules) {
			rulesCopy := rules
			officialImagePricingRules = &rulesCopy
		}
		officialVideoPricingRules := officialVideoPricingRulesForExport(modelName)
		costDiscountPercent := ch.EffectiveCostPercent
		currentDiscount := inviteeModelActualDiscountPercent(officialBaseForDiscount, channelBase, costDiscountPercent, defaultRate)
		defaultRates[key] = defaultRate
		items = append(items, InviteeModelMarkupDiscountRateItem{
			ModelName:                      modelName,
			ChannelID:                      ch.ChannelID,
			ChannelPath:                    inviteePricingChannelPath(modelName, ch),
			SupplierType:                   channelSupplierTypes[ch.ChannelID],
			ChannelName:                    channelNames[ch.ChannelID],
			DefaultMarkupDiscountRate:      defaultRate,
			CurrentMarkupDiscountRate:      defaultRate,
			OfficialCurrentDiscountPercent: currentDiscount,
			ChannelPriceDiscountPercent:    costDiscountPercent,
			OfficialBasePrice:              officialBase,
			ChannelBasePrice:               channelBase,
			PricingQuotaType:               quotaType,
			OfficialPricingQuotaType:       p.QuotaType,
			OfficialModelRatio:             p.ModelRatio,
			OfficialCompletionRatio:        ratio_setting.GetCompletionRatio(modelName),
			OfficialCacheRatio:             p.CacheRatio,
			OfficialCreateCacheRatio:       p.CreateCacheRatio,
			OfficialImageRatio:             p.ImageRatio,
			OfficialImagePrice:             officialImagePrice,
			OfficialImagePricingRules:      officialImagePricingRules,
			OfficialVideoRatio:             p.VideoRatio,
			OfficialVideoCompletionRatio:   ratio_setting.GetVideoCompletionRatio(modelName),
			OfficialVideoPrice:             p.VideoPrice,
			OfficialVideoPricingRules:      officialVideoPricingRules,
			OfficialRequestTierPricing:     officialTierPricing,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ChannelPath != items[j].ChannelPath {
			return items[i].ChannelPath < items[j].ChannelPath
		}
		return items[i].ChannelID < items[j].ChannelID
	})
	return items, defaultRates, nil
}

// GetInviteeModelDiscounts 获取被邀请用户的模型加价折扣率配置（列表口径与 /api/pricing 一致）。
func GetInviteeModelDiscounts(inviterId, inviteeUserId int) ([]InviteeModelMarkupDiscountRateItem, float64, error) {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return nil, 0, errors.New("invalid id")
	}
	// 验证被邀请人确实属于当前邀请人
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil {
		return nil, 0, errors.New("user not found")
	}
	if invitee.InviterId != inviterId {
		return nil, 0, errors.New("not your invitee")
	}

	items, _, err := listPricingVisibleMarkupDiscountRateItems()
	if err != nil {
		return nil, 0, err
	}

	var rel AffInviteRelation
	err = DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).First(&rel).Error
	savedByKey := map[string]float64{}
	if err == nil {
		savedList, parseErr := parseInviteeModelMarkupDiscountRates(rel.ModelMarkupDiscountRate)
		if parseErr != nil {
			return nil, 0, parseErr
		}
		savedByKey = inviteeModelMarkupDiscountRateMap(savedList)
	}

	for i := range items {
		key := inviteeModelMarkupKey(items[i].ChannelID, items[i].ModelName)
		if rate, ok := savedByKey[key]; ok {
			items[i].CurrentMarkupDiscountRate = rate
		}
		items[i].CurrentMarkupDiscountRate = clampChannelMarkupDiscountRate(items[i].CurrentMarkupDiscountRate)
		maxRate := MaxInviteeMarkupDiscountRate(items[i])
		if items[i].CurrentMarkupDiscountRate > maxRate {
			items[i].CurrentMarkupDiscountRate = maxRate
		}
	}

	return items, 0, nil
}

// MaxInviteeMarkupDiscountRate returns the highest agent markup that keeps the
// invitee-facing sale rate at or below the official/base price.
func MaxInviteeMarkupDiscountRate(item InviteeModelMarkupDiscountRateItem) float64 {
	costPercent := item.ChannelPriceDiscountPercent
	if !isFiniteFloat(costPercent) {
		costPercent = 100
	}
	if costPercent < 0 {
		costPercent = 0
	}
	maxRate := maxInviteeSaleRatePercent - costPercent
	if maxRate < 0 {
		return 0
	}
	if maxRate > maxInviteeSaleRatePercent {
		return maxInviteeSaleRatePercent
	}
	return maxRate
}

// UpdateInviteeModelDiscounts 更新被邀请用户的模型加价折扣率（仅存与渠道默认不同的项，JSON 数组全量覆盖）。
type ModelMarkupDiscountRateUpdateRequest struct {
	ModelName          string  `json:"model_name"`
	ChannelID          int     `json:"channel_id"`
	MarkupDiscountRate float64 `json:"markup_discount_rate"`
}

func buildModelMarkupDiscountRateJSON(updates []ModelMarkupDiscountRateUpdateRequest) (string, error) {
	for _, u := range updates {
		if u.MarkupDiscountRate < 0 || u.MarkupDiscountRate > maxInviteeSaleRatePercent {
			return "", fmt.Errorf("markup_discount_rate for model %s must be between 0 and %.0f", u.ModelName, maxInviteeSaleRatePercent)
		}
	}

	items, defaultRates, err := listPricingVisibleMarkupDiscountRateItems()
	if err != nil {
		return "", err
	}
	maxRates := make(map[string]float64, len(items))
	for _, item := range items {
		maxRates[inviteeModelMarkupKey(item.ChannelID, item.ModelName)] = MaxInviteeMarkupDiscountRate(item)
	}

	ratesToSave := make([]inviteeModelMarkupDiscountRateEntry, 0)
	for _, u := range updates {
		modelName := strings.TrimSpace(u.ModelName)
		key := inviteeModelMarkupKey(u.ChannelID, modelName)
		defaultRate, hasModel := defaultRates[key]
		if !hasModel {
			continue
		}
		maxRate := maxRates[key]
		if u.MarkupDiscountRate > maxRate+0.000001 {
			return "", fmt.Errorf("markup_discount_rate for model %s must be between 0 and %.1f", modelName, maxRate)
		}
		if math.Abs(u.MarkupDiscountRate-defaultRate) > 0.000001 {
			ratesToSave = append(ratesToSave, inviteeModelMarkupDiscountRateEntry{
				ModelName:          modelName,
				ChannelID:          u.ChannelID,
				MarkupDiscountRate: u.MarkupDiscountRate,
			})
		}
	}

	discountsJSON, err := common.Marshal(ratesToSave)
	if err != nil {
		return "", err
	}
	return string(discountsJSON), nil
}

func UpdateInviteeModelDiscounts(inviterId, inviteeUserId int, updates []ModelMarkupDiscountRateUpdateRequest) error {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return errors.New("invalid id")
	}
	// 验证被邀请人确实属于当前邀请人
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil {
		return errors.New("user not found")
	}
	if invitee.InviterId != inviterId {
		return errors.New("not your invitee")
	}

	discountsJSON, err := buildModelMarkupDiscountRateJSON(updates)
	if err != nil {
		return err
	}

	// 更新或创建关系记录
	ts := common.GetTimestamp()
	var rel AffInviteRelation
	err = DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).First(&rel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rel = AffInviteRelation{
			InviterId:               inviterId,
			InviteeUserId:           inviteeUserId,
			ModelMarkupDiscountRate: discountsJSON,
			CreatedAt:               ts,
			UpdatedAt:               ts,
		}
		return DB.Create(&rel).Error
	}
	if err != nil {
		return err
	}

	rel.ModelMarkupDiscountRate = discountsJSON
	rel.UpdatedAt = ts
	return DB.Save(&rel).Error
}

func overlayModelMarkupDiscountItems(raw string) ([]InviteeModelMarkupDiscountRateItem, error) {
	items, _, err := listPricingVisibleMarkupDiscountRateItems()
	if err != nil {
		return nil, err
	}
	savedList, parseErr := parseInviteeModelMarkupDiscountRates(raw)
	if parseErr != nil {
		return nil, parseErr
	}
	savedByKey := inviteeModelMarkupDiscountRateMap(savedList)
	for i := range items {
		key := inviteeModelMarkupKey(items[i].ChannelID, items[i].ModelName)
		if rate, ok := savedByKey[key]; ok {
			items[i].CurrentMarkupDiscountRate = rate
		}
		items[i].CurrentMarkupDiscountRate = clampChannelMarkupDiscountRate(items[i].CurrentMarkupDiscountRate)
		maxRate := MaxInviteeMarkupDiscountRate(items[i])
		if items[i].CurrentMarkupDiscountRate > maxRate {
			items[i].CurrentMarkupDiscountRate = maxRate
		}
	}
	return items, nil
}

func GetDistributorModelDiscountTemplate(distributorId int) ([]InviteeModelMarkupDiscountRateItem, int64, bool, error) {
	if distributorId <= 0 {
		return nil, 0, false, errors.New("invalid distributor")
	}
	var user User
	if err := DB.Select("id", "distributor_model_markup_discount_template", "distributor_model_discount_auto_apply").Where("id = ?", distributorId).First(&user).Error; err != nil {
		return nil, 0, false, err
	}
	items, err := overlayModelMarkupDiscountItems(user.DistributorModelMarkupDiscountTemplate)
	if err != nil {
		return nil, 0, false, err
	}
	total, err := CountAffInvitees(distributorId)
	if err != nil {
		return nil, 0, false, err
	}
	return items, total, user.DistributorModelDiscountAutoApply == common.DistributorFlagYes, nil
}

func UpdateDistributorModelDiscountTemplate(distributorId int, updates []ModelMarkupDiscountRateUpdateRequest, autoApplyNewInvitees *bool) error {
	if distributorId <= 0 {
		return errors.New("invalid distributor")
	}
	discountsJSON, err := buildModelMarkupDiscountRateJSON(updates)
	if err != nil {
		return err
	}
	columns := map[string]interface{}{
		"distributor_model_markup_discount_template": discountsJSON,
	}
	if autoApplyNewInvitees != nil {
		if *autoApplyNewInvitees {
			columns["distributor_model_discount_auto_apply"] = common.DistributorFlagYes
		} else {
			columns["distributor_model_discount_auto_apply"] = common.DistributorFlagNo
		}
	}
	return DB.Model(&User{}).Where("id = ?", distributorId).Updates(columns).Error
}

func inviteeIDsForBatch(inviterId int, all bool, selectedIDs []int) ([]int, error) {
	if inviterId <= 0 {
		return nil, errors.New("invalid inviter")
	}
	if all {
		var ids []int
		if err := DB.Model(&User{}).Where("inviter_id = ?", inviterId).Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
		return ids, nil
	}
	seen := map[int]struct{}{}
	ids := make([]int, 0, len(selectedIDs))
	for _, id := range selectedIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New("no invitees selected")
	}
	var matched []int
	if err := DB.Model(&User{}).Where("inviter_id = ? AND id IN ?", inviterId, ids).Pluck("id", &matched).Error; err != nil {
		return nil, err
	}
	if len(matched) != len(ids) {
		return nil, errors.New("some invitees do not belong to this distributor")
	}
	return matched, nil
}

func updateInviteeModelDiscountsRawBatch(inviterId int, inviteeIDs []int, raw string) (int, error) {
	if len(inviteeIDs) == 0 {
		return 0, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "[]"
	}
	ts := common.GetTimestamp()
	return len(inviteeIDs), DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&AffInviteRelation{}).
			Where("inviter_id = ? AND invitee_user_id IN ?", inviterId, inviteeIDs).
			Updates(map[string]interface{}{
				"model_markup_discount_rate": raw,
				"updated_at":                 ts,
			}).Error; err != nil {
			return err
		}
		if raw == "[]" {
			return nil
		}
		var existing []int
		if err := tx.Model(&AffInviteRelation{}).
			Where("inviter_id = ? AND invitee_user_id IN ?", inviterId, inviteeIDs).
			Pluck("invitee_user_id", &existing).Error; err != nil {
			return err
		}
		existingSet := make(map[int]struct{}, len(existing))
		for _, id := range existing {
			existingSet[id] = struct{}{}
		}
		defaultBps := defaultCommissionBpsForNewInviteRelation(inviterId)
		toCreate := make([]AffInviteRelation, 0)
		for _, id := range inviteeIDs {
			if _, ok := existingSet[id]; ok {
				continue
			}
			toCreate = append(toCreate, AffInviteRelation{
				InviterId:               inviterId,
				InviteeUserId:           id,
				CommissionRatioBps:      defaultBps,
				CommissionEarnedQuota:   0,
				ProfitShareEarnedQuota:  0,
				ModelMarkupDiscountRate: raw,
				CreatedAt:               ts,
				UpdatedAt:               ts,
			})
		}
		if len(toCreate) == 0 {
			return nil
		}
		return tx.Create(&toCreate).Error
	})
}

func ApplyDistributorModelDiscountTemplate(inviterId int, all bool, selectedIDs []int) (int, error) {
	inviteeIDs, err := inviteeIDsForBatch(inviterId, all, selectedIDs)
	if err != nil {
		return 0, err
	}
	var user User
	if err := DB.Select("id", "distributor_model_markup_discount_template").Where("id = ?", inviterId).First(&user).Error; err != nil {
		return 0, err
	}
	raw := strings.TrimSpace(user.DistributorModelMarkupDiscountTemplate)
	if raw == "" {
		raw = "[]"
	}
	list, err := parseInviteeModelMarkupDiscountRates(raw)
	if err != nil {
		return 0, err
	}
	updates := make([]ModelMarkupDiscountRateUpdateRequest, 0, len(list))
	for _, item := range list {
		updates = append(updates, ModelMarkupDiscountRateUpdateRequest{
			ModelName:          item.ModelName,
			ChannelID:          item.ChannelID,
			MarkupDiscountRate: item.MarkupDiscountRate,
		})
	}
	raw, err = buildModelMarkupDiscountRateJSON(updates)
	if err != nil {
		return 0, err
	}
	return updateInviteeModelDiscountsRawBatch(inviterId, inviteeIDs, raw)
}

func ResetInviteeModelDiscountsBatch(inviterId int, all bool, selectedIDs []int) (int, error) {
	inviteeIDs, err := inviteeIDsForBatch(inviterId, all, selectedIDs)
	if err != nil {
		return 0, err
	}
	return updateInviteeModelDiscountsRawBatch(inviterId, inviteeIDs, "[]")
}

func affInviteRelationColumnExists(column string) bool {
	if DB == nil {
		return false
	}
	var count int64
	var err error
	switch {
	case common.UsingPostgreSQL:
		err = DB.Raw(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`,
			"aff_invite_relations", column).Scan(&count).Error
	case common.UsingSQLite:
		err = DB.Raw(`SELECT COUNT(*) FROM pragma_table_info('aff_invite_relations') WHERE name = ?`, column).Scan(&count).Error
	default:
		err = DB.Raw(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			"aff_invite_relations", column).Scan(&count).Error
	}
	return err == nil && count > 0
}

// migrateAffInviteRelationModelMarkupDiscountRateColumn 将废弃列 model_discounts 重命名为 model_markup_discount_rate。
func migrateAffInviteRelationModelMarkupDiscountRateColumn() error {
	if DB == nil || !DB.Migrator().HasTable(&AffInviteRelation{}) {
		return nil
	}
	hasNew := affInviteRelationColumnExists("model_markup_discount_rate")
	hasOld := affInviteRelationColumnExists("model_discounts")
	if !hasOld {
		return nil
	}
	if hasNew {
		// 两列并存时把旧数据拷到新列（新列为空时）
		if common.UsingPostgreSQL {
			_ = DB.Exec(`UPDATE aff_invite_relations SET model_markup_discount_rate = model_discounts
				WHERE (model_markup_discount_rate IS NULL OR TRIM(model_markup_discount_rate) = '' OR model_markup_discount_rate = '[]')
				AND model_discounts IS NOT NULL AND TRIM(model_discounts) <> '' AND model_discounts <> '[]'`).Error
		} else {
			_ = DB.Exec(`UPDATE aff_invite_relations SET model_markup_discount_rate = model_discounts
				WHERE (model_markup_discount_rate IS NULL OR TRIM(model_markup_discount_rate) = '' OR model_markup_discount_rate = '[]')
				AND model_discounts IS NOT NULL AND TRIM(model_discounts) <> '' AND model_discounts <> '[]'`).Error
		}
	} else {
		var stmt string
		switch {
		case common.UsingPostgreSQL:
			stmt = `ALTER TABLE aff_invite_relations RENAME COLUMN model_discounts TO model_markup_discount_rate`
		case common.UsingSQLite:
			stmt = `ALTER TABLE aff_invite_relations RENAME COLUMN model_discounts TO model_markup_discount_rate`
		default:
			stmt = `ALTER TABLE aff_invite_relations CHANGE COLUMN model_discounts model_markup_discount_rate TEXT`
		}
		if err := DB.Exec(stmt).Error; err != nil {
			return err
		}
		common.SysLog("migrate: renamed aff_invite_relations.model_discounts -> model_markup_discount_rate")
		return nil
	}
	var dropStmt string
	switch {
	case common.UsingPostgreSQL:
		dropStmt = `ALTER TABLE aff_invite_relations DROP COLUMN IF EXISTS model_discounts`
	default:
		dropStmt = `ALTER TABLE aff_invite_relations DROP COLUMN model_discounts`
	}
	err := DB.Exec(dropStmt).Error
	if err == nil {
		common.SysLog("migrate: dropped aff_invite_relations.model_discounts")
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unknown column") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "no such column") ||
		strings.Contains(msg, "does not exist") {
		return nil
	}
	if common.UsingSQLite &&
		(strings.Contains(msg, "syntax error") || strings.Contains(msg, "near \"drop\"")) {
		common.SysLog("migrate: skip DROP model_discounts (SQLite may not support DROP COLUMN): " + err.Error())
		return nil
	}
	return err
}
