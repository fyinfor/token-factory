package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

type Channel struct {
	Id                 int     `json:"id"`
	Type               int     `json:"type" gorm:"default:0"`
	Key                string  `json:"key" gorm:"not null"`
	OpenAIOrganization *string `json:"openai_organization"`
	TestModel          *string `json:"test_model"`
	Status             int     `json:"status" gorm:"default:1"`
	Name               string  `json:"name" gorm:"index"`
	Weight             *uint   `json:"weight" gorm:"default:0"`
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`
	TestTime           int64   `json:"test_time" gorm:"bigint"` // 最近一次渠道测试时间（Unix 秒级时间戳）
	ResponseTime       int     `json:"response_time"`           // 最近一次渠道测试响应耗时（毫秒）
	BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`
	Other              string  `json:"other"`
	Balance            float64 `json:"balance"` // in USD
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
	Models             string  `json:"models"`
	Group              string  `json:"group" gorm:"type:varchar(64);default:'default'"`
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string `json:"model_mapping" gorm:"type:text"`
	//MaxInputTokens     *int    `json:"max_input_tokens" gorm:"default:0"`
	StatusCodeMapping *string `json:"status_code_mapping" gorm:"type:varchar(1024);default:''"`
	Priority          *int64  `json:"priority" gorm:"bigint;default:0"`
	AutoBan           *int    `json:"auto_ban" gorm:"default:1"`
	OtherInfo         string  `json:"other_info"` // 渠道扩展信息（JSON），测试相关键：last_test_success/last_test_message/last_test_model/last_test_time
	Tag               *string `json:"tag" gorm:"index"`
	Setting           *string `json:"setting" gorm:"type:text"` // 渠道额外设置
	ParamOverride     *string `json:"param_override" gorm:"type:text"`
	HeaderOverride    *string `json:"header_override" gorm:"type:text"`
	Remark            *string `json:"remark" gorm:"type:varchar(255)" validate:"max=255"`
	// add after v0.8.5
	ChannelInfo ChannelInfo `json:"channel_info" gorm:"type:json"`

	OtherSettings         string `json:"settings" gorm:"column:settings"`                         // 其他设置，存储azure版本等不需要检索的信息，详见dto.ChannelOtherSettings
	OwnerUserID           int    `json:"owner_user_id" gorm:"type:int;index;default:0"`           // 渠道归属用户ID（供应商场景）
	SupplierApplicationID int    `json:"supplier_application_id" gorm:"type:int;index;default:0"` // 关联 supplier_applications.id
	ChannelNo             string `json:"channel_no" gorm:"type:varchar(32);default:'';index;comment:供应商渠道编号 c1,c2 递增"`
	SupplierName          string `json:"supplier_name,omitempty" gorm:"-"` // 供应商用户名（由控制器回填，不落库）

	// 渠道计费折扣（百分数，100=原价无折扣，60=六折/按原价×0.6 计费）。nil=数据库默认/未设，按 100 处理。使用指针以便 GORM Updates 时可将 0% 写回。
	PriceDiscountPercent *float64 `json:"price_discount_percent" gorm:"type:double precision;default:100"`

	// cache info
	Keys []string `json:"-" gorm:"-"`
}

type ChannelInfo struct {
	IsMultiKey             bool                  `json:"is_multi_key"`                        // 是否多Key模式
	MultiKeySize           int                   `json:"multi_key_size"`                      // 多Key模式下的Key数量
	MultiKeyStatusList     map[int]int           `json:"multi_key_status_list"`               // key状态列表，key index -> status
	MultiKeyDisabledReason map[int]string        `json:"multi_key_disabled_reason,omitempty"` // key禁用原因列表，key index -> reason
	MultiKeyDisabledTime   map[int]int64         `json:"multi_key_disabled_time,omitempty"`   // key禁用时间列表，key index -> time
	MultiKeyPollingIndex   int                   `json:"multi_key_polling_index"`             // 多Key模式下轮询的key索引
	MultiKeyMode           constant.MultiKeyMode `json:"multi_key_mode"`
}

// Value implements driver.Valuer interface
func (c ChannelInfo) Value() (driver.Value, error) {
	return common.Marshal(&c)
}

// Scan implements sql.Scanner interface
func (c *ChannelInfo) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return common.Unmarshal(bytesValue, c)
}

func (channel *Channel) GetKeys() []string {
	if channel.Key == "" {
		return []string{}
	}
	if len(channel.Keys) > 0 {
		return channel.Keys
	}
	trimmed := strings.TrimSpace(channel.Key)
	// If the key starts with '[', try to parse it as a JSON array (e.g., for Vertex AI scenarios)
	if strings.HasPrefix(trimmed, "[") {
		var arr []json.RawMessage
		if err := common.Unmarshal([]byte(trimmed), &arr); err == nil {
			res := make([]string, len(arr))
			for i, v := range arr {
				res[i] = string(v)
			}
			return res
		}
	}
	// Otherwise, fall back to splitting by newline
	keys := strings.Split(strings.Trim(channel.Key, "\n"), "\n")
	return keys
}

func (channel *Channel) GetNextEnabledKey() (string, int, *types.TokenFactoryError) {
	// If not in multi-key mode, return the original key string directly.
	if !channel.ChannelInfo.IsMultiKey {
		return channel.Key, 0, nil
	}

	// Obtain all keys (split by \n)
	keys := channel.GetKeys()
	if len(keys) == 0 {
		// No keys available, return error, should disable the channel
		return "", 0, types.NewError(errors.New("no keys available"), types.ErrorCodeChannelNoAvailableKey)
	}

	lock := GetChannelPollingLock(channel.Id)
	lock.Lock()
	defer lock.Unlock()

	statusList := channel.ChannelInfo.MultiKeyStatusList
	// helper to get key status, default to enabled when missing
	getStatus := func(idx int) int {
		if statusList == nil {
			return common.ChannelStatusEnabled
		}
		if status, ok := statusList[idx]; ok {
			return status
		}
		return common.ChannelStatusEnabled
	}

	// Collect indexes of enabled keys
	enabledIdx := make([]int, 0, len(keys))
	for i := range keys {
		if getStatus(i) == common.ChannelStatusEnabled {
			enabledIdx = append(enabledIdx, i)
		}
	}
	// If no specific status list or none enabled, return an explicit error so caller can
	// properly handle a channel with no available keys (e.g. mark channel disabled).
	// Returning the first key here caused requests to keep using an already-disabled key.
	if len(enabledIdx) == 0 {
		return "", 0, types.NewError(errors.New("no enabled keys"), types.ErrorCodeChannelNoAvailableKey)
	}

	switch channel.ChannelInfo.MultiKeyMode {
	case constant.MultiKeyModeRandom:
		// Randomly pick one enabled key
		selectedIdx := enabledIdx[rand.Intn(len(enabledIdx))]
		return keys[selectedIdx], selectedIdx, nil
	case constant.MultiKeyModePolling:
		// Use channel-specific lock to ensure thread-safe polling

		channelInfo, err := CacheGetChannelInfo(channel.Id)
		if err != nil {
			return "", 0, types.NewError(err, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
		}
		//println("before polling index:", channel.ChannelInfo.MultiKeyPollingIndex)
		defer func() {
			if common.DebugEnabled {
				println(fmt.Sprintf("channel %d polling index: %d", channel.Id, channel.ChannelInfo.MultiKeyPollingIndex))
			}
			if !common.MemoryCacheEnabled {
				_ = channel.SaveChannelInfo()
			} else {
				// CacheUpdateChannel(channel)
			}
		}()
		// Start from the saved polling index and look for the next enabled key
		start := channelInfo.MultiKeyPollingIndex
		if start < 0 || start >= len(keys) {
			start = 0
		}
		for i := 0; i < len(keys); i++ {
			idx := (start + i) % len(keys)
			if getStatus(idx) == common.ChannelStatusEnabled {
				// update polling index for next call (point to the next position)
				channel.ChannelInfo.MultiKeyPollingIndex = (idx + 1) % len(keys)
				return keys[idx], idx, nil
			}
		}
		// Fallback – should not happen, but return first enabled key
		return keys[enabledIdx[0]], enabledIdx[0], nil
	default:
		// Unknown mode, default to first enabled key (or original key string)
		return keys[enabledIdx[0]], enabledIdx[0], nil
	}
}

func (channel *Channel) SaveChannelInfo() error {
	return DB.Model(channel).Update("channel_info", channel.ChannelInfo).Error
}

func (channel *Channel) GetModels() []string {
	if channel.Models == "" {
		return []string{}
	}
	return strings.Split(strings.Trim(channel.Models, ","), ",")
}

func (channel *Channel) GetGroups() []string {
	if channel.Group == "" {
		return []string{}
	}
	groups := strings.Split(strings.Trim(channel.Group, ","), ",")
	for i, group := range groups {
		groups[i] = strings.TrimSpace(group)
	}
	return groups
}

func (channel *Channel) GetOtherInfo() map[string]interface{} {
	otherInfo := make(map[string]interface{})
	if channel.OtherInfo != "" {
		err := common.Unmarshal([]byte(channel.OtherInfo), &otherInfo)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal other info: channel_id=%d, tag=%s, name=%s, error=%v", channel.Id, channel.GetTag(), channel.Name, err))
		}
	}
	return otherInfo
}

func (channel *Channel) SetOtherInfo(otherInfo map[string]interface{}) {
	otherInfoBytes, err := json.Marshal(otherInfo)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal other info: channel_id=%d, tag=%s, name=%s, error=%v", channel.Id, channel.GetTag(), channel.Name, err))
		return
	}
	channel.OtherInfo = string(otherInfoBytes)
}

func (channel *Channel) GetTag() string {
	if channel.Tag == nil {
		return ""
	}
	return *channel.Tag
}

func (channel *Channel) SetTag(tag string) {
	channel.Tag = &tag
}

func (channel *Channel) GetAutoBan() bool {
	if channel.AutoBan == nil {
		return false
	}
	return *channel.AutoBan == 1
}

func (channel *Channel) Save() error {
	return DB.Save(channel).Error
}

func (channel *Channel) SaveWithoutKey() error {
	if channel.Id == 0 {
		return errors.New("channel ID is 0")
	}
	return DB.Omit("key").Save(channel).Error
}

func GetAllChannels(startIdx int, num int, selectAll bool, idSort bool) ([]*Channel, error) {
	var channels []*Channel
	var err error
	order := "priority desc"
	if idSort {
		order = "id desc"
	}
	if selectAll {
		err = DB.Order(order).Find(&channels).Error
	} else {
		err = DB.Order(order).Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	}
	return channels, err
}

// ListChannelsByOwnerUser 分页查询指定归属用户创建的渠道。
func ListChannelsByOwnerUser(ownerUserID int, startIdx int, num int) ([]*Channel, int64, error) {
	var (
		channels []*Channel
		total    int64
	)
	query := DB.Model(&Channel{}).Where("owner_user_id = ?", ownerUserID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error; err != nil {
		return nil, 0, err
	}
	return channels, total, nil
}

// ListAllSupplierChannels 分页查询所有供应商归属渠道（管理员视角）。
func ListAllSupplierChannels(startIdx int, num int) ([]*Channel, int64, error) {
	var (
		channels []*Channel
		total    int64
	)
	query := DB.Model(&Channel{}).Where("owner_user_id > ? AND supplier_application_id > ?", 0, 0)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error; err != nil {
		return nil, 0, err
	}
	return channels, total, nil
}

// SupplierChannelSearchFilter 供应商渠道搜索过滤参数。
type SupplierChannelSearchFilter struct {
	ChannelID    int
	Keyword      string
	Supplier     string
	Name         string
	Key          string
	BaseURL      string
	ModelKeyword string
	Group        string
}

// ChannelSimplePricingItem pricing 页面使用的渠道精简信息。
type ChannelSimplePricingItem struct {
	ChannelID     int    `json:"channel_id"`
	ChannelName   string `json:"channel_name"`
	ChannelNo     string `json:"channel_no"`
	SupplierAlias string `json:"supplier_alias"`
}

// ChannelPricingMeta 定价接口计算渠道维度价格所需的渠道行（含供应商别名）。
type ChannelPricingMeta struct {
	ChannelID              int      `gorm:"column:channel_id"`
	SupplierApplicationID  int      `gorm:"column:supplier_application_id"`
	ChannelNo              string   `gorm:"column:channel_no"`
	Models                 string   `gorm:"column:models"`
	SupplierAlias          *string  `gorm:"column:supplier_alias"`
	PriceDiscountPercent   *float64 `gorm:"column:price_discount_percent"`
}

// ListChannelsForPricing 查询定价页使用的渠道列表。
func ListChannelsForPricing() ([]ChannelSimplePricingItem, error) {
	items := make([]ChannelSimplePricingItem, 0)
	err := DB.Model(&Channel{}).
		Select("channels.id AS channel_id, channels.name AS channel_name, channels.channel_no, COALESCE(supplier_applications.supplier_alias, '') AS supplier_alias").
		Joins("LEFT JOIN supplier_applications ON supplier_applications.id = channels.supplier_application_id").
		Order("channels.id ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ListChannelPricingMeta 查询全部渠道的定价元数据（用于按模型汇总渠道价）。
func ListChannelPricingMeta() ([]ChannelPricingMeta, error) {
	items := make([]ChannelPricingMeta, 0)
	err := DB.Model(&Channel{}).
		Select("channels.id AS channel_id, channels.supplier_application_id, channels.channel_no, channels.models, channels.price_discount_percent, supplier_applications.supplier_alias").
		Joins("LEFT JOIN supplier_applications ON supplier_applications.id = channels.supplier_application_id").
		Order("channels.id ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ChannelModelsRawContains 判断 channels.models 逗号列表是否包含指定模型名（去空格精确匹配）。
func ChannelModelsRawContains(modelsRaw string, modelName string) bool {
	if strings.TrimSpace(modelsRaw) == "" || strings.TrimSpace(modelName) == "" {
		return false
	}
	for _, m := range strings.Split(modelsRaw, ",") {
		if strings.TrimSpace(m) == modelName {
			return true
		}
	}
	return false
}

// SearchSupplierChannels 搜索供应商渠道（供应商只查自己，管理员可查全部供应商渠道）。
func SearchSupplierChannels(ownerUserID *int, startIdx int, num int, filter SupplierChannelSearchFilter) ([]*Channel, int64, error) {
	var (
		channels []*Channel
		total    int64
	)
	query := DB.Model(&Channel{})
	if ownerUserID != nil {
		query = query.Where("owner_user_id = ?", *ownerUserID)
	} else {
		query = query.Where("owner_user_id > ? AND supplier_application_id > ?", 0, 0)
	}
	if filter.ChannelID > 0 {
		query = query.Where("id = ?", filter.ChannelID)
	}
	if filter.Keyword != "" {
		keywordLike := "%" + filter.Keyword + "%"
		query = query.Where("(name LIKE ? OR "+commonKeyCol+" LIKE ? OR base_url LIKE ?)", keywordLike, keywordLike, keywordLike)
	}
	if filter.Supplier != "" {
		query = query.Joins("LEFT JOIN users ON users.id = channels.owner_user_id").Where("users.username LIKE ?", "%"+filter.Supplier+"%")
	}
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Key != "" {
		// commonKeyCol 兼容不同数据库对保留字 key 的转义差异
		query = query.Where(commonKeyCol+" = ? OR "+commonKeyCol+" LIKE ?", filter.Key, "%"+filter.Key+"%")
	}
	if filter.BaseURL != "" {
		query = query.Where("base_url LIKE ?", "%"+filter.BaseURL+"%")
	}
	if filter.ModelKeyword != "" {
		query = query.Where("models LIKE ?", "%"+filter.ModelKeyword+"%")
	}
	if filter.Group != "" && filter.Group != "null" {
		var groupCondition string
		if common.UsingMySQL {
			groupCondition = "CONCAT(',', " + commonGroupCol + ", ',') LIKE ?"
		} else {
			groupCondition = "(',' || " + commonGroupCol + " || ',') LIKE ?"
		}
		query = query.Where(groupCondition, "%,"+filter.Group+",%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error; err != nil {
		return nil, 0, err
	}
	return channels, total, nil
}

// ParseSupplierChannelIDFilter 解析渠道ID筛选参数（支持空值）。
func ParseSupplierChannelIDFilter(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetChannelsByTag(tag string, idSort bool, selectAll bool) ([]*Channel, error) {
	var channels []*Channel
	order := "priority desc"
	if idSort {
		order = "id desc"
	}
	query := DB.Where("tag = ?", tag).Order(order)
	if !selectAll {
		query = query.Omit("key")
	}
	err := query.Find(&channels).Error
	return channels, err
}

func SearchChannels(keyword string, group string, model string, idSort bool) ([]*Channel, error) {
	var channels []*Channel
	modelsCol := "`models`"

	// 如果是 PostgreSQL，使用双引号
	if common.UsingPostgreSQL {
		modelsCol = `"models"`
	}

	baseURLCol := "`base_url`"
	// 如果是 PostgreSQL，使用双引号
	if common.UsingPostgreSQL {
		baseURLCol = `"base_url"`
	}

	order := "priority desc"
	if idSort {
		order = "id desc"
	}

	// 构造基础查询
	baseQuery := DB.Model(&Channel{}).Omit("key")

	// 构造WHERE子句
	var whereClause string
	var args []interface{}
	if group != "" && group != "null" {
		var groupCondition string
		if common.UsingMySQL {
			groupCondition = `CONCAT(',', ` + commonGroupCol + `, ',') LIKE ?`
		} else {
			// sqlite, PostgreSQL
			groupCondition = `(',' || ` + commonGroupCol + ` || ',') LIKE ?`
		}
		whereClause = "(id = ? OR name LIKE ? OR " + commonKeyCol + " = ? OR " + baseURLCol + " LIKE ?) AND " + modelsCol + ` LIKE ? AND ` + groupCondition
		args = append(args, common.String2Int(keyword), "%"+keyword+"%", keyword, "%"+keyword+"%", "%"+model+"%", "%,"+group+",%")
	} else {
		whereClause = "(id = ? OR name LIKE ? OR " + commonKeyCol + " = ? OR " + baseURLCol + " LIKE ?) AND " + modelsCol + " LIKE ?"
		args = append(args, common.String2Int(keyword), "%"+keyword+"%", keyword, "%"+keyword+"%", "%"+model+"%")
	}

	// 执行查询
	err := baseQuery.Where(whereClause, args...).Order(order).Find(&channels).Error
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func GetChannelById(id int, selectAll bool) (*Channel, error) {
	channel := &Channel{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(channel, "id = ?", id).Error
	} else {
		err = DB.Omit("key").First(channel, "id = ?", id).Error
	}
	if err != nil {
		return nil, err
	}
	return channel, nil
}

// ResolveSupplierApplicationIDByAlias 根据供应商别名返回 supplier_application_id。
//
//   - "P0"（不区分大小写）返回 0，代表未归属任何 supplier_applications 的渠道；
//   - 其他别名到 supplier_applications.supplier_alias 精确匹配后返回其 id；
//
// 未找到别名时返回 (0, false, err)；找到 P0 或匹配记录时 found=true。
func ResolveSupplierApplicationIDByAlias(alias string) (supplierApplicationID int, found bool, err error) {
	aliasTrim := strings.TrimSpace(alias)
	if aliasTrim == "" {
		return 0, false, fmt.Errorf("alias 不能为空")
	}
	if strings.EqualFold(aliasTrim, "P0") {
		return 0, true, nil
	}
	var app SupplierApplication
	if err := DB.Select("id").Where("supplier_alias = ?", aliasTrim).First(&app).Error; err != nil {
		return 0, false, fmt.Errorf("供应商别名未找到: %s", aliasTrim)
	}
	return app.ID, true, nil
}

// FindChannelIDBySupplierAliasAndNo 根据「供应商别名」+「渠道编号」定位具体渠道 ID。
//
// 支持两种别名形式：
//  1. "P0"：特指未归属供应商申请（supplier_application_id = 0）的渠道；
//  2. 其他：先到 supplier_applications.supplier_alias 精确匹配，取得 id 后再按
//     (supplier_application_id, channel_no) 查找渠道。
//
// 该方法仅返回启用状态的渠道。未找到时返回 0 与具体错误信息。
func FindChannelIDBySupplierAliasAndNo(alias string, channelNo string) (int, error) {
	noTrim := strings.TrimSpace(channelNo)
	if noTrim == "" {
		return 0, fmt.Errorf("channel_no 不能为空")
	}
	supplierApplicationID, _, err := ResolveSupplierApplicationIDByAlias(alias)
	if err != nil {
		return 0, err
	}

	var channel Channel
	if err := DB.Select("id, status").
		Where("supplier_application_id = ? AND channel_no = ?", supplierApplicationID, noTrim).
		First(&channel).Error; err != nil {
		return 0, fmt.Errorf("未找到渠道: %s/%s", strings.TrimSpace(alias), noTrim)
	}
	if channel.Status != common.ChannelStatusEnabled {
		return 0, fmt.Errorf("渠道已禁用: %s/%s", strings.TrimSpace(alias), noTrim)
	}
	return channel.Id, nil
}

// ValidateSupplierChannelNoUnique 校验同一 supplier_application_id 下 channel_no 不重复（空编号不校验）。
// excludeChannelID 大于 0 时排除自身，用于更新；新建时传 0。
func ValidateSupplierChannelNoUnique(excludeChannelID int, supplierApplicationID int, channelNo string) error {
	no := strings.TrimSpace(channelNo)
	if supplierApplicationID <= 0 || no == "" {
		return nil
	}
	q := DB.Model(&Channel{}).Where("supplier_application_id = ? AND channel_no = ?", supplierApplicationID, no)
	if excludeChannelID > 0 {
		q = q.Where("id <> ?", excludeChannelID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return fmt.Errorf("该供应商下已存在相同渠道编号")
	}
	return nil
}

func maxChannelNoNumericSuffixForSupplier(tx *gorm.DB, supplierApplicationID int) (int, error) {
	var existing []string
	if err := tx.Model(&Channel{}).Where("supplier_application_id = ?", supplierApplicationID).Pluck("channel_no", &existing).Error; err != nil {
		return 0, err
	}
	maxN := 0
	for _, no := range existing {
		no = strings.TrimSpace(no)
		if len(no) >= 2 && no[0] == 'c' {
			if n, err := strconv.Atoi(no[1:]); err == nil && n > maxN {
				maxN = n
			}
		}
	}
	return maxN, nil
}

// allocateSupplierChannelNosInBatch 为同一事务批次内待插入的渠道分配 channel_no（supplier_application_id 相同则 c1,c2… 连续不重复）。
func allocateSupplierChannelNosInBatch(tx *gorm.DB, batch []Channel) error {
	maxCache := make(map[int]int)
	assigned := make(map[int]int)
	for i := range batch {
		sid := batch[i].SupplierApplicationID
		if sid <= 0 {
			continue
		}
		if strings.TrimSpace(batch[i].ChannelNo) != "" {
			continue
		}
		m, ok := maxCache[sid]
		if !ok {
			var err error
			m, err = maxChannelNoNumericSuffixForSupplier(tx, sid)
			if err != nil {
				return err
			}
			maxCache[sid] = m
		}
		assigned[sid]++
		batch[i].ChannelNo = "c" + strconv.Itoa(m+assigned[sid])
	}
	return nil
}

// BackfillSupplierChannelNo 为历史数据补全 channel_no：按供应商分组、渠道 id 升序，已有非空编号的不覆盖，空编号接续已有最大 cN。
func BackfillSupplierChannelNo() error {
	type row struct {
		ID                    int
		SupplierApplicationID int
		ChannelNo             string
	}
	var rows []row
	if err := DB.Model(&Channel{}).
		Select("id", "supplier_application_id", "channel_no").
		Where("supplier_application_id > 0").
		Order("supplier_application_id asc, id asc").
		Scan(&rows).Error; err != nil {
		return err
	}
	bySupplier := make(map[int][]row)
	for _, r := range rows {
		bySupplier[r.SupplierApplicationID] = append(bySupplier[r.SupplierApplicationID], r)
	}
	for _, list := range bySupplier {
		maxN := 0
		for _, r := range list {
			no := strings.TrimSpace(r.ChannelNo)
			if len(no) >= 2 && no[0] == 'c' {
				if n, err := strconv.Atoi(no[1:]); err == nil && n > maxN {
					maxN = n
				}
			}
		}
		for _, r := range list {
			if strings.TrimSpace(r.ChannelNo) != "" {
				continue
			}
			maxN++
			no := "c" + strconv.Itoa(maxN)
			if err := DB.Model(&Channel{}).Where("id = ?", r.ID).Update("channel_no", no).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func BatchInsertChannels(channels []Channel) error {
	if len(channels) == 0 {
		return nil
	}
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	createdChannels := make([]Channel, 0, len(channels))
	for _, chunk := range lo.Chunk(channels, 50) {
		if err := allocateSupplierChannelNosInBatch(tx, chunk); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Create(&chunk).Error; err != nil {
			tx.Rollback()
			return err
		}
		createdChannels = append(createdChannels, chunk...)
		for _, channel_ := range chunk {
			if err := channel_.AddAbilities(tx); err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	// Best effort: initialize channel-level model pricing entries so newly imported
	// channels are visible in channel pricing editor without blank mappings.
	ensureChannelModelPricingDefaults(createdChannels)
	return nil
}

func ensureChannelModelPricingDefaults(channels []Channel) {
	if len(channels) == 0 {
		return
	}
	channelModelPrice := ratio_setting.GetChannelModelPriceCopy()
	channelModelRatio := ratio_setting.GetChannelModelRatioCopy()
	changed := false

	for _, ch := range channels {
		if ch.Id <= 0 {
			continue
		}
		channelID := strconv.Itoa(ch.Id)
		if _, ok := channelModelPrice[channelID]; !ok {
			channelModelPrice[channelID] = make(map[string]float64)
		}
		if _, ok := channelModelRatio[channelID]; !ok {
			channelModelRatio[channelID] = make(map[string]float64)
		}
		seen := make(map[string]struct{})
		for _, rawModel := range ch.GetModels() {
			modelName := strings.TrimSpace(rawModel)
			if modelName == "" {
				continue
			}
			modelKey := ratio_setting.FormatMatchingModelName(modelName)
			if _, ok := seen[modelKey]; ok {
				continue
			}
			seen[modelKey] = struct{}{}

			if _, exists := channelModelPrice[channelID][modelKey]; exists {
				continue
			}
			if _, exists := channelModelRatio[channelID][modelKey]; exists {
				continue
			}

			if modelPrice, ok := ratio_setting.GetModelPrice(modelKey, false); ok {
				channelModelPrice[channelID][modelKey] = modelPrice
				changed = true
				continue
			}
			if modelRatio, ok, _ := ratio_setting.GetModelRatio(modelKey); ok {
				channelModelRatio[channelID][modelKey] = modelRatio
				changed = true
			}
		}
	}

	if !changed {
		return
	}
	priceJSONBytes, err := common.Marshal(channelModelPrice)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal ChannelModelPrice: %v", err))
		return
	}
	ratioJSONBytes, err := common.Marshal(channelModelRatio)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal ChannelModelRatio: %v", err))
		return
	}
	if err := UpdateOption("ChannelModelPrice", string(priceJSONBytes)); err != nil {
		common.SysLog(fmt.Sprintf("failed to update ChannelModelPrice option: %v", err))
	}
	if err := UpdateOption("ChannelModelRatio", string(ratioJSONBytes)); err != nil {
		common.SysLog(fmt.Sprintf("failed to update ChannelModelRatio option: %v", err))
	}
}

func BatchDeleteChannels(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	// 使用事务 分批删除channel表和abilities表
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	for _, chunk := range lo.Chunk(ids, 200) {
		if err := tx.Where("id in (?)", chunk).Delete(&Channel{}).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Where("channel_id in (?)", chunk).Delete(&Ability{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (channel *Channel) GetPriority() int64 {
	if channel.Priority == nil {
		return 0
	}
	return *channel.Priority
}

func (channel *Channel) GetWeight() int {
	if channel.Weight == nil {
		return 0
	}
	return int(*channel.Weight)
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	url := *channel.BaseURL
	if url == "" {
		url = constant.ChannelBaseURLs[channel.Type]
	}
	return url
}

func (channel *Channel) GetModelMapping() string {
	if channel.ModelMapping == nil {
		return ""
	}
	return *channel.ModelMapping
}

func (channel *Channel) GetStatusCodeMapping() string {
	if channel.StatusCodeMapping == nil {
		return ""
	}
	return *channel.StatusCodeMapping
}

func (channel *Channel) Insert() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		batch := []Channel{*channel}
		if err := allocateSupplierChannelNosInBatch(tx, batch); err != nil {
			return err
		}
		*channel = batch[0]
		if err := tx.Create(channel).Error; err != nil {
			return err
		}
		return channel.AddAbilities(tx)
	})
}

func (channel *Channel) Update() error {
	// If this is a multi-key channel, recalculate MultiKeySize based on the current key list to avoid inconsistency after editing keys
	if channel.ChannelInfo.IsMultiKey {
		var keyStr string
		if channel.Key != "" {
			keyStr = channel.Key
		} else {
			// If key is not provided, read the existing key from the database
			if existing, err := GetChannelById(channel.Id, true); err == nil {
				keyStr = existing.Key
			}
		}
		// Parse the key list (supports newline separation or JSON array)
		keys := []string{}
		if keyStr != "" {
			trimmed := strings.TrimSpace(keyStr)
			if strings.HasPrefix(trimmed, "[") {
				var arr []json.RawMessage
				if err := common.Unmarshal([]byte(trimmed), &arr); err == nil {
					keys = make([]string, len(arr))
					for i, v := range arr {
						keys[i] = string(v)
					}
				}
			}
			if len(keys) == 0 { // fallback to newline split
				keys = strings.Split(strings.Trim(keyStr, "\n"), "\n")
			}
		}
		channel.ChannelInfo.MultiKeySize = len(keys)
		// Clean up status data that exceeds the new key count to prevent index out of range
		if channel.ChannelInfo.MultiKeyStatusList != nil {
			for idx := range channel.ChannelInfo.MultiKeyStatusList {
				if idx >= channel.ChannelInfo.MultiKeySize {
					delete(channel.ChannelInfo.MultiKeyStatusList, idx)
				}
			}
		}
	}
	var err error
	err = DB.Model(channel).Updates(channel).Error
	if err != nil {
		return err
	}
	DB.Model(channel).First(channel, "id = ?", channel.Id)
	err = channel.UpdateAbilities(nil)
	return err
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("response_time", "test_time").Updates(Channel{
		TestTime:     common.GetTimestamp(),
		ResponseTime: int(responseTime),
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update response time: channel_id=%d, error=%v", channel.Id, err))
	}
}

// UpdateTestResult 持久化渠道测试结果（成功/失败）、响应时间与测试时间。
// 同时会把最近一次测试的状态信息写入 other_info，供前端与运维排查使用。
func (channel *Channel) UpdateTestResult(success bool, responseTime int64, message string, modelName string) {
	err := DB.Transaction(func(tx *gorm.DB) error {
		var dbChannel Channel
		if err := tx.Select("id", "other_info").First(&dbChannel, "id = ?", channel.Id).Error; err != nil {
			return err
		}

		otherInfo := dbChannel.GetOtherInfo()
		otherInfo["last_test_success"] = success
		otherInfo["last_test_message"] = message
		otherInfo["last_test_model"] = modelName
		otherInfo["last_test_time"] = common.GetTimestamp()
		dbChannel.SetOtherInfo(otherInfo)

		return tx.Model(&Channel{}).Where("id = ?", channel.Id).Select("response_time", "test_time", "other_info").Updates(Channel{
			TestTime:     common.GetTimestamp(),
			ResponseTime: int(responseTime),
			OtherInfo:    dbChannel.OtherInfo,
		}).Error
	})
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update test result: channel_id=%d, error=%v", channel.Id, err))
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_time", "balance").Updates(Channel{
		BalanceUpdatedTime: common.GetTimestamp(),
		Balance:            balance,
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update balance: channel_id=%d, error=%v", channel.Id, err))
	}
}

func (channel *Channel) Delete() error {
	var err error
	err = DB.Delete(channel).Error
	if err != nil {
		return err
	}
	err = channel.DeleteAbilities()
	return err
}

var channelStatusLock sync.Mutex

// channelPollingLocks stores locks for each channel.id to ensure thread-safe polling
var channelPollingLocks sync.Map

// GetChannelPollingLock returns or creates a mutex for the given channel ID
func GetChannelPollingLock(channelId int) *sync.Mutex {
	if lock, exists := channelPollingLocks.Load(channelId); exists {
		return lock.(*sync.Mutex)
	}
	// Create new lock for this channel
	newLock := &sync.Mutex{}
	actual, _ := channelPollingLocks.LoadOrStore(channelId, newLock)
	return actual.(*sync.Mutex)
}

// CleanupChannelPollingLocks removes locks for channels that no longer exist
// This is optional and can be called periodically to prevent memory leaks
func CleanupChannelPollingLocks() {
	var activeChannelIds []int
	DB.Model(&Channel{}).Pluck("id", &activeChannelIds)

	activeChannelSet := make(map[int]bool)
	for _, id := range activeChannelIds {
		activeChannelSet[id] = true
	}

	channelPollingLocks.Range(func(key, value interface{}) bool {
		channelId := key.(int)
		if !activeChannelSet[channelId] {
			channelPollingLocks.Delete(channelId)
		}
		return true
	})
}

func handlerMultiKeyUpdate(channel *Channel, usingKey string, status int, reason string) {
	keys := channel.GetKeys()
	if len(keys) == 0 {
		channel.Status = status
	} else {
		var keyIndex int
		for i, key := range keys {
			if key == usingKey {
				keyIndex = i
				break
			}
		}
		if channel.ChannelInfo.MultiKeyStatusList == nil {
			channel.ChannelInfo.MultiKeyStatusList = make(map[int]int)
		}
		if status == common.ChannelStatusEnabled {
			delete(channel.ChannelInfo.MultiKeyStatusList, keyIndex)
		} else {
			channel.ChannelInfo.MultiKeyStatusList[keyIndex] = status
			if channel.ChannelInfo.MultiKeyDisabledReason == nil {
				channel.ChannelInfo.MultiKeyDisabledReason = make(map[int]string)
			}
			if channel.ChannelInfo.MultiKeyDisabledTime == nil {
				channel.ChannelInfo.MultiKeyDisabledTime = make(map[int]int64)
			}
			channel.ChannelInfo.MultiKeyDisabledReason[keyIndex] = reason
			channel.ChannelInfo.MultiKeyDisabledTime[keyIndex] = common.GetTimestamp()
		}
		if len(channel.ChannelInfo.MultiKeyStatusList) >= channel.ChannelInfo.MultiKeySize {
			channel.Status = common.ChannelStatusAutoDisabled
			info := channel.GetOtherInfo()
			info["status_reason"] = "All keys are disabled"
			info["status_time"] = common.GetTimestamp()
			channel.SetOtherInfo(info)
		}
	}
}

func UpdateChannelStatus(channelId int, usingKey string, status int, reason string) bool {
	if common.MemoryCacheEnabled {
		channelStatusLock.Lock()
		defer channelStatusLock.Unlock()

		channelCache, _ := CacheGetChannel(channelId)
		if channelCache == nil {
			return false
		}
		if channelCache.ChannelInfo.IsMultiKey {
			// Use per-channel lock to prevent concurrent map read/write with GetNextEnabledKey
			pollingLock := GetChannelPollingLock(channelId)
			pollingLock.Lock()
			// 如果是多Key模式，更新缓存中的状态
			handlerMultiKeyUpdate(channelCache, usingKey, status, reason)
			pollingLock.Unlock()
			//CacheUpdateChannel(channelCache)
			//return true
		} else {
			// 如果缓存渠道存在，且状态已是目标状态，直接返回
			if channelCache.Status == status {
				return false
			}
			CacheUpdateChannelStatus(channelId, status)
		}
	}

	shouldUpdateAbilities := false
	defer func() {
		if shouldUpdateAbilities {
			err := UpdateAbilityStatus(channelId, status == common.ChannelStatusEnabled)
			if err != nil {
				common.SysLog(fmt.Sprintf("failed to update ability status: channel_id=%d, error=%v", channelId, err))
			}
		}
	}()
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return false
	} else {
		if channel.Status == status {
			return false
		}

		if channel.ChannelInfo.IsMultiKey {
			beforeStatus := channel.Status
			// Protect map writes with the same per-channel lock used by readers
			pollingLock := GetChannelPollingLock(channelId)
			pollingLock.Lock()
			handlerMultiKeyUpdate(channel, usingKey, status, reason)
			pollingLock.Unlock()
			if beforeStatus != channel.Status {
				shouldUpdateAbilities = true
			}
		} else {
			info := channel.GetOtherInfo()
			info["status_reason"] = reason
			info["status_time"] = common.GetTimestamp()
			channel.SetOtherInfo(info)
			channel.Status = status
			shouldUpdateAbilities = true
		}
		err = channel.SaveWithoutKey()
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to update channel status: channel_id=%d, status=%d, error=%v", channel.Id, status, err))
			return false
		}
	}
	return true
}

func EnableChannelByTag(tag string) error {
	err := DB.Model(&Channel{}).Where("tag = ?", tag).Update("status", common.ChannelStatusEnabled).Error
	if err != nil {
		return err
	}
	err = UpdateAbilityStatusByTag(tag, true)
	return err
}

func DisableChannelByTag(tag string) error {
	err := DB.Model(&Channel{}).Where("tag = ?", tag).Update("status", common.ChannelStatusManuallyDisabled).Error
	if err != nil {
		return err
	}
	err = UpdateAbilityStatusByTag(tag, false)
	return err
}

func EditChannelByTag(tag string, newTag *string, modelMapping *string, models *string, group *string, priority *int64, weight *uint, paramOverride *string, headerOverride *string) error {
	updateData := Channel{}
	shouldReCreateAbilities := false
	updatedTag := tag
	// 如果 newTag 不为空且不等于 tag，则更新 tag
	if newTag != nil && *newTag != tag {
		updateData.Tag = newTag
		updatedTag = *newTag
	}
	if modelMapping != nil && *modelMapping != "" {
		updateData.ModelMapping = modelMapping
	}
	if models != nil && *models != "" {
		shouldReCreateAbilities = true
		updateData.Models = *models
	}
	if group != nil && *group != "" {
		shouldReCreateAbilities = true
		updateData.Group = *group
	}
	if priority != nil {
		updateData.Priority = priority
	}
	if weight != nil {
		updateData.Weight = weight
	}
	if paramOverride != nil {
		updateData.ParamOverride = paramOverride
	}
	if headerOverride != nil {
		updateData.HeaderOverride = headerOverride
	}

	err := DB.Model(&Channel{}).Where("tag = ?", tag).Updates(updateData).Error
	if err != nil {
		return err
	}
	if shouldReCreateAbilities {
		channels, err := GetChannelsByTag(updatedTag, false, false)
		if err == nil {
			for _, channel := range channels {
				err = channel.UpdateAbilities(nil)
				if err != nil {
					common.SysLog(fmt.Sprintf("failed to update abilities: channel_id=%d, tag=%s, error=%v", channel.Id, channel.GetTag(), err))
				}
			}
		}
	} else {
		err := UpdateAbilityByTag(tag, newTag, priority, weight)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateChannelUsedQuota(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeChannelUsedQuota, id, quota)
		return
	}
	updateChannelUsedQuota(id, quota)
}

func updateChannelUsedQuota(id int, quota int) {
	err := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update channel used quota: channel_id=%d, delta_quota=%d, error=%v", id, quota, err))
	}
}

func DeleteChannelByStatus(status int64) (int64, error) {
	result := DB.Where("status = ?", status).Delete(&Channel{})
	return result.RowsAffected, result.Error
}

func DeleteDisabledChannel() (int64, error) {
	result := DB.Where("status = ? or status = ?", common.ChannelStatusAutoDisabled, common.ChannelStatusManuallyDisabled).Delete(&Channel{})
	return result.RowsAffected, result.Error
}

func GetPaginatedTags(offset int, limit int) ([]*string, error) {
	var tags []*string
	err := DB.Model(&Channel{}).Select("DISTINCT tag").Where("tag != ''").Offset(offset).Limit(limit).Find(&tags).Error
	return tags, err
}

func SearchTags(keyword string, group string, model string, idSort bool) ([]*string, error) {
	var tags []*string
	modelsCol := "`models`"

	// 如果是 PostgreSQL，使用双引号
	if common.UsingPostgreSQL {
		modelsCol = `"models"`
	}

	baseURLCol := "`base_url`"
	// 如果是 PostgreSQL，使用双引号
	if common.UsingPostgreSQL {
		baseURLCol = `"base_url"`
	}

	order := "priority desc"
	if idSort {
		order = "id desc"
	}

	// 构造基础查询
	baseQuery := DB.Model(&Channel{}).Omit("key")

	// 构造WHERE子句
	var whereClause string
	var args []interface{}
	if group != "" && group != "null" {
		var groupCondition string
		if common.UsingMySQL {
			groupCondition = `CONCAT(',', ` + commonGroupCol + `, ',') LIKE ?`
		} else {
			// sqlite, PostgreSQL
			groupCondition = `(',' || ` + commonGroupCol + ` || ',') LIKE ?`
		}
		whereClause = "(id = ? OR name LIKE ? OR " + commonKeyCol + " = ? OR " + baseURLCol + " LIKE ?) AND " + modelsCol + ` LIKE ? AND ` + groupCondition
		args = append(args, common.String2Int(keyword), "%"+keyword+"%", keyword, "%"+keyword+"%", "%"+model+"%", "%,"+group+",%")
	} else {
		whereClause = "(id = ? OR name LIKE ? OR " + commonKeyCol + " = ? OR " + baseURLCol + " LIKE ?) AND " + modelsCol + " LIKE ?"
		args = append(args, common.String2Int(keyword), "%"+keyword+"%", keyword, "%"+keyword+"%", "%"+model+"%")
	}

	subQuery := baseQuery.Where(whereClause, args...).
		Select("tag").
		Where("tag != ''").
		Order(order)

	err := DB.Table("(?) as sub", subQuery).
		Select("DISTINCT tag").
		Find(&tags).Error

	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (channel *Channel) ValidateSettings() error {
	channelParams := &dto.ChannelSettings{}
	if channel.Setting != nil && *channel.Setting != "" {
		err := common.Unmarshal([]byte(*channel.Setting), channelParams)
		if err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) GetSetting() dto.ChannelSettings {
	setting := dto.ChannelSettings{}
	if channel.Setting != nil && *channel.Setting != "" {
		err := common.Unmarshal([]byte(*channel.Setting), &setting)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal setting: channel_id=%d, error=%v", channel.Id, err))
			channel.Setting = nil // 清空设置以避免后续错误
			_ = channel.Save()    // 保存修改
		}
	}
	return setting
}

func (channel *Channel) SetSetting(setting dto.ChannelSettings) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal setting: channel_id=%d, error=%v", channel.Id, err))
		return
	}
	channel.Setting = common.GetPointer[string](string(settingBytes))
}

func (channel *Channel) GetOtherSettings() dto.ChannelOtherSettings {
	setting := dto.ChannelOtherSettings{}
	if channel.OtherSettings != "" {
		err := common.UnmarshalJsonStr(channel.OtherSettings, &setting)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal setting: channel_id=%d, error=%v", channel.Id, err))
			channel.OtherSettings = "{}" // 清空设置以避免后续错误
			_ = channel.Save()           // 保存修改
		}
	}
	return setting
}

func (channel *Channel) SetOtherSettings(setting dto.ChannelOtherSettings) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal setting: channel_id=%d, error=%v", channel.Id, err))
		return
	}
	channel.OtherSettings = string(settingBytes)
}

func (channel *Channel) GetParamOverride() map[string]interface{} {
	paramOverride := make(map[string]interface{})
	if channel.ParamOverride != nil && *channel.ParamOverride != "" {
		err := common.Unmarshal([]byte(*channel.ParamOverride), &paramOverride)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal param override: channel_id=%d, error=%v", channel.Id, err))
		}
	}
	return paramOverride
}

func (channel *Channel) GetHeaderOverride() map[string]interface{} {
	headerOverride := make(map[string]interface{})
	if channel.HeaderOverride != nil && *channel.HeaderOverride != "" {
		err := common.Unmarshal([]byte(*channel.HeaderOverride), &headerOverride)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal header override: channel_id=%d, error=%v", channel.Id, err))
		}
	}
	return headerOverride
}

func GetChannelsByIds(ids []int) ([]*Channel, error) {
	var channels []*Channel
	err := DB.Where("id in (?)", ids).Find(&channels).Error
	return channels, err
}

func BatchSetChannelTag(ids []int, tag *string) error {
	// 开启事务
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 更新标签
	err := tx.Model(&Channel{}).Where("id in (?)", ids).Update("tag", tag).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// update ability status
	channels, err := GetChannelsByIds(ids)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, channel := range channels {
		err = channel.UpdateAbilities(tx)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// CountAllChannels returns total channels in DB
func CountAllChannels() (int64, error) {
	var total int64
	err := DB.Model(&Channel{}).Count(&total).Error
	return total, err
}

// CountAllTags returns number of non-empty distinct tags
func CountAllTags() (int64, error) {
	var total int64
	err := DB.Model(&Channel{}).Where("tag is not null AND tag != ''").Distinct("tag").Count(&total).Error
	return total, err
}

// Get channels of specified type with pagination
func GetChannelsByType(startIdx int, num int, idSort bool, channelType int) ([]*Channel, error) {
	var channels []*Channel
	order := "priority desc"
	if idSort {
		order = "id desc"
	}
	err := DB.Where("type = ?", channelType).Order(order).Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	return channels, err
}

// Count channels of specific type
func CountChannelsByType(channelType int) (int64, error) {
	var count int64
	err := DB.Model(&Channel{}).Where("type = ?", channelType).Count(&count).Error
	return count, err
}

// Return map[type]count for all channels
func CountChannelsGroupByType() (map[int64]int64, error) {
	type result struct {
		Type  int64 `gorm:"column:type"`
		Count int64 `gorm:"column:count"`
	}
	var results []result
	err := DB.Model(&Channel{}).Select("type, count(*) as count").Group("type").Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int64)
	for _, r := range results {
		counts[r.Type] = r.Count
	}
	return counts, nil
}
