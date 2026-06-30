package model

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ModelVisibilityPublic = "public"
	ModelVisibilitySets   = "sets"
)

type ModelVisibilitySet struct {
	ID          int            `json:"id"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex:uk_model_visibility_set_name"`
	Description string         `json:"description,omitempty" gorm:"type:varchar(255)"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	UserIDs     []int    `json:"user_ids,omitempty" gorm:"-"`
	UserTags    []string `json:"user_tags,omitempty" gorm:"-"`
	UserGroups  []string `json:"user_groups,omitempty" gorm:"-"`
	MemberCount int64    `json:"member_count" gorm:"-"`
	RuleCount   int64    `json:"rule_count" gorm:"-"`
}

type ModelVisibilitySetUser struct {
	ID        int   `json:"id"`
	SetID     int   `json:"set_id" gorm:"not null;uniqueIndex:uk_model_visibility_set_user,priority:1;index"`
	UserID    int   `json:"user_id" gorm:"not null;uniqueIndex:uk_model_visibility_set_user,priority:2;index"`
	CreatedAt int64 `json:"created_at" gorm:"bigint"`
}

type ModelVisibilitySetRule struct {
	ID        int    `json:"id"`
	SetID     int    `json:"set_id" gorm:"not null;uniqueIndex:uk_model_visibility_set_rule,priority:1;index"`
	RuleType  string `json:"rule_type" gorm:"type:varchar(16);not null;uniqueIndex:uk_model_visibility_set_rule,priority:2"`
	RuleValue string `json:"rule_value" gorm:"type:varchar(128);not null;uniqueIndex:uk_model_visibility_set_rule,priority:3;index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

type ModelVisibilityBinding struct {
	ID        int   `json:"id"`
	ModelID   int   `json:"model_id" gorm:"not null;uniqueIndex:uk_model_visibility_binding,priority:1;index"`
	SetID     int   `json:"set_id" gorm:"not null;uniqueIndex:uk_model_visibility_binding,priority:2;index"`
	CreatedAt int64 `json:"created_at" gorm:"bigint"`
}

type ModelVisibilityUserSummary struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Group       string `json:"group,omitempty" gorm:"column:group"`
	Tags        string `json:"tags,omitempty"`
	Role        int    `json:"role"`
}

type ModelVisibilitySetDetail struct {
	ModelVisibilitySet
	Users []ModelVisibilityUserSummary `json:"users,omitempty"`
}

func normalizeVisibilityNames(values []string, maxLen int) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		if maxLen > 0 && len([]rune(name)) > maxLen {
			name = string([]rune(name)[:maxLen])
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func normalizeVisibilityUserIDs(ids []int) []int {
	out := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

func normalizeVisibilitySetIDs(ids []int) []int {
	return normalizeVisibilityUserIDs(ids)
}

func (s *ModelVisibilitySet) InsertWithMembers(userIDs []int, userTags []string, userGroups []string) error {
	now := common.GetTimestamp()
	s.Name = strings.TrimSpace(s.Name)
	s.Description = strings.TrimSpace(s.Description)
	if err := validateVisibilitySetMembers(userIDs, userTags, userGroups); err != nil {
		return err
	}
	s.CreatedTime = now
	s.UpdatedTime = now
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(s).Error; err != nil {
			return err
		}
		return replaceVisibilitySetMembersTx(tx, s.ID, userIDs, userTags, userGroups)
	})
}

func UpdateModelVisibilitySetWithMembers(s *ModelVisibilitySet, userIDs []int, userTags []string, userGroups []string) error {
	if s == nil || s.ID <= 0 {
		return errors.New("invalid visibility set")
	}
	s.Name = strings.TrimSpace(s.Name)
	s.Description = strings.TrimSpace(s.Description)
	s.UpdatedTime = common.GetTimestamp()
	if err := validateVisibilitySetMembers(userIDs, userTags, userGroups); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ModelVisibilitySet{}).Where("id = ?", s.ID).
			Select("name", "description", "updated_time").
			Updates(map[string]interface{}{
				"name":         s.Name,
				"description":  s.Description,
				"updated_time": s.UpdatedTime,
			}).Error; err != nil {
			return err
		}
		return replaceVisibilitySetMembersTx(tx, s.ID, userIDs, userTags, userGroups)
	})
}

func validateVisibilitySetMembers(userIDs []int, userTags []string, userGroups []string) error {
	users, err := ListUsersByVisibilityRules(userIDs, userTags, userGroups, 1)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errors.New("用户集至少需要命中一个用户")
	}
	return nil
}

func replaceVisibilitySetMembersTx(tx *gorm.DB, setID int, userIDs []int, userTags []string, userGroups []string) error {
	if setID <= 0 {
		return errors.New("invalid visibility set id")
	}
	if err := tx.Where("set_id = ?", setID).Delete(&ModelVisibilitySetUser{}).Error; err != nil {
		return err
	}
	if err := tx.Where("set_id = ?", setID).Delete(&ModelVisibilitySetRule{}).Error; err != nil {
		return err
	}
	now := common.GetTimestamp()
	normalizedUserIDs := normalizeVisibilityUserIDs(userIDs)
	if len(normalizedUserIDs) > 0 {
		rows := make([]ModelVisibilitySetUser, 0, len(normalizedUserIDs))
		for _, userID := range normalizedUserIDs {
			rows = append(rows, ModelVisibilitySetUser{
				SetID:     setID,
				UserID:    userID,
				CreatedAt: now,
			})
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return err
		}
	}
	rules := make([]ModelVisibilitySetRule, 0)
	for _, tag := range normalizeVisibilityNames(userTags, 64) {
		rules = append(rules, ModelVisibilitySetRule{
			SetID:     setID,
			RuleType:  "tag",
			RuleValue: tag,
			CreatedAt: now,
		})
	}
	for _, group := range normalizeVisibilityNames(userGroups, 64) {
		rules = append(rules, ModelVisibilitySetRule{
			SetID:     setID,
			RuleType:  "group",
			RuleValue: group,
			CreatedAt: now,
		})
	}
	if len(rules) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rules).Error; err != nil {
			return err
		}
	}
	return nil
}

func GetModelVisibilitySetDetail(id int) (*ModelVisibilitySetDetail, error) {
	if id <= 0 {
		return nil, errors.New("invalid visibility set id")
	}
	var set ModelVisibilitySet
	if err := DB.First(&set, id).Error; err != nil {
		return nil, err
	}
	detail := &ModelVisibilitySetDetail{ModelVisibilitySet: set}
	if err := fillVisibilitySetDetail(detail); err != nil {
		return nil, err
	}
	return detail, nil
}

func fillVisibilitySetDetail(detail *ModelVisibilitySetDetail) error {
	if detail == nil || detail.ID <= 0 {
		return nil
	}
	var userIDs []int
	if err := DB.Model(&ModelVisibilitySetUser{}).
		Where("set_id = ?", detail.ID).
		Order("user_id ASC").
		Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}
	detail.UserIDs = userIDs
	detail.MemberCount = int64(len(userIDs))
	if len(userIDs) > 0 {
		users := make([]ModelVisibilityUserSummary, 0, len(userIDs))
		if err := DB.Model(&User{}).
			Select(userVisibilitySummarySelect()).
			Where("id IN ?", userIDs).
			Order("id ASC").
			Scan(&users).Error; err != nil {
			return err
		}
		detail.Users = users
	}
	var rules []ModelVisibilitySetRule
	if err := DB.Where("set_id = ?", detail.ID).Order("rule_type ASC, rule_value ASC").Find(&rules).Error; err != nil {
		return err
	}
	for _, rule := range rules {
		switch rule.RuleType {
		case "tag":
			detail.UserTags = append(detail.UserTags, rule.RuleValue)
		case "group":
			detail.UserGroups = append(detail.UserGroups, rule.RuleValue)
		}
	}
	detail.RuleCount = int64(len(rules))
	return nil
}

func ListModelVisibilitySets(keyword string, offset int, limit int) ([]ModelVisibilitySetDetail, int64, error) {
	var (
		sets  []ModelVisibilitySet
		total int64
	)
	query := DB.Model(&ModelVisibilitySet{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&sets).Error; err != nil {
		return nil, 0, err
	}
	details := make([]ModelVisibilitySetDetail, 0, len(sets))
	for i := range sets {
		detail := ModelVisibilitySetDetail{ModelVisibilitySet: sets[i]}
		if err := fillVisibilitySetDetail(&detail); err != nil {
			return nil, 0, err
		}
		details = append(details, detail)
	}
	return details, total, nil
}

func DeleteModelVisibilitySet(id int) error {
	if id <= 0 {
		return errors.New("invalid visibility set id")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("set_id = ?", id).Delete(&ModelVisibilitySetUser{}).Error; err != nil {
			return err
		}
		if err := tx.Where("set_id = ?", id).Delete(&ModelVisibilitySetRule{}).Error; err != nil {
			return err
		}
		if err := tx.Where("set_id = ?", id).Delete(&ModelVisibilityBinding{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&ModelVisibilitySet{}, id).Error
	})
}

func ReplaceModelVisibilityBindings(modelID int, setIDs []int) error {
	if modelID <= 0 {
		return errors.New("invalid model id")
	}
	setIDs = normalizeVisibilitySetIDs(setIDs)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("model_id = ?", modelID).Delete(&ModelVisibilityBinding{}).Error; err != nil {
			return err
		}
		if len(setIDs) == 0 {
			return nil
		}
		var existingIDs []int
		if err := tx.Model(&ModelVisibilitySet{}).Where("id IN ?", setIDs).Pluck("id", &existingIDs).Error; err != nil {
			return err
		}
		if len(existingIDs) == 0 {
			return nil
		}
		existingIDs = normalizeVisibilitySetIDs(existingIDs)
		now := common.GetTimestamp()
		rows := make([]ModelVisibilityBinding, 0, len(existingIDs))
		for _, setID := range existingIDs {
			rows = append(rows, ModelVisibilityBinding{
				ModelID:   modelID,
				SetID:     setID,
				CreatedAt: now,
			})
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
	})
}

func GetModelVisibilitySetIDs(modelID int) ([]int, error) {
	if modelID <= 0 {
		return nil, nil
	}
	var ids []int
	err := DB.Model(&ModelVisibilityBinding{}).
		Where("model_id = ?", modelID).
		Order("set_id ASC").
		Pluck("set_id", &ids).Error
	return ids, err
}

func FillModelsVisibility(models []*Model) {
	if len(models) == 0 {
		return
	}
	modelIDs := make([]int, 0, len(models))
	byID := make(map[int]*Model, len(models))
	for _, item := range models {
		if item == nil || item.Id <= 0 {
			continue
		}
		item.Visibility = ModelVisibilityPublic
		item.VisibilitySetIDs = nil
		modelIDs = append(modelIDs, item.Id)
		byID[item.Id] = item
	}
	if len(modelIDs) == 0 {
		return
	}
	var rows []ModelVisibilityBinding
	if err := DB.Where("model_id IN ?", modelIDs).Order("set_id ASC").Find(&rows).Error; err != nil {
		return
	}
	for _, row := range rows {
		if item, ok := byID[row.ModelID]; ok {
			item.VisibilitySetIDs = append(item.VisibilitySetIDs, row.SetID)
			item.Visibility = ModelVisibilitySets
		}
	}
}

func GetAccessibleModelNamesForUser(userID int, modelNames []string) (map[string]bool, error) {
	out := make(map[string]bool, len(modelNames))
	cleaned := make([]string, 0, len(modelNames))
	seen := make(map[string]struct{}, len(modelNames))
	for _, name := range modelNames {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		cleaned = append(cleaned, n)
		out[n] = false
	}
	if len(cleaned) == 0 {
		return out, nil
	}
	if userID > 0 && IsAdmin(userID) {
		for _, name := range cleaned {
			out[name] = true
		}
		return out, nil
	}
	for _, name := range cleaned {
		allowed, err := UserCanAccessModel(userID, name)
		if err != nil {
			return nil, err
		}
		out[name] = allowed
	}
	return out, nil
}

func FilterModelsVisibleToUser(userID int, modelNames []string) []string {
	if len(modelNames) == 0 {
		return modelNames
	}
	out := make([]string, 0, len(modelNames))
	allowed, err := GetAccessibleModelNamesForUser(userID, modelNames)
	if err != nil {
		return out
	}
	for _, name := range modelNames {
		if allowed[strings.TrimSpace(name)] {
			out = append(out, name)
		}
	}
	return out
}

func UserCanAccessModel(userID int, modelName string) (bool, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return false, nil
	}
	if userID > 0 && IsAdmin(userID) {
		return true, nil
	}
	rows := getActiveModelRowsCached()
	modelID := findBestVisibilityModelID(modelName, rows)
	if modelID <= 0 {
		return true, nil
	}
	var setIDs []int
	if err := DB.Model(&ModelVisibilityBinding{}).Where("model_id = ?", modelID).Pluck("set_id", &setIDs).Error; err != nil {
		return false, err
	}
	if len(setIDs) == 0 {
		return true, nil
	}
	if userID <= 0 {
		return false, nil
	}
	var user User
	if err := DB.Select("id, "+commonGroupCol+", tags, role").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if user.Role >= common.RoleAdminUser {
		return true, nil
	}
	var count int64
	if err := DB.Model(&ModelVisibilitySetUser{}).Where("set_id IN ? AND user_id = ?", setIDs, userID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	rules := make([]ModelVisibilitySetRule, 0)
	if err := DB.Where("set_id IN ?", setIDs).Find(&rules).Error; err != nil {
		return false, err
	}
	userGroup := strings.TrimSpace(user.Group)
	userTags := make(map[string]struct{})
	for _, tag := range GetUserTagsList(user.Tags) {
		userTags[tag] = struct{}{}
	}
	for _, rule := range rules {
		switch rule.RuleType {
		case "group":
			if userGroup != "" && userGroup == strings.TrimSpace(rule.RuleValue) {
				return true, nil
			}
		case "tag":
			if _, ok := userTags[strings.TrimSpace(rule.RuleValue)]; ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func UserCanAccessAnyModelName(userID int, rawModelName string) (bool, error) {
	rawModelName = strings.TrimSpace(rawModelName)
	if rawModelName == "" {
		return false, nil
	}
	if allowed, err := UserCanAccessModel(userID, rawModelName); err != nil || allowed {
		return allowed, err
	}
	parts := strings.Split(rawModelName, "/")
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" || name == rawModelName {
			continue
		}
		if allowed, err := UserCanAccessModel(userID, name); err != nil || allowed {
			return allowed, err
		}
	}
	return false, nil
}

func findBestVisibilityModelID(modelName string, rows []Model) int {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0
	}
	bestIdx := -1
	for i := range rows {
		row := rows[i]
		if row.Id <= 0 {
			continue
		}
		if !modelNameMatchesRule(row.ModelName, modelName, row.NameRule) {
			continue
		}
		if bestIdx < 0 {
			bestIdx = i
			continue
		}
		cur := rows[bestIdx]
		curPriority := modelNameRulePriority(cur.NameRule)
		newPriority := modelNameRulePriority(row.NameRule)
		if newPriority < curPriority {
			bestIdx = i
			continue
		}
		if newPriority == curPriority && len(row.ModelName) > len(cur.ModelName) {
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return 0
	}
	return rows[bestIdx].Id
}

func QueryUsersForVisibility(keyword string, group string, tag string, limit int) ([]ModelVisibilityUserSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return QueryUsersForVisibilityPage(keyword, group, tag, 0, limit)
}

func CountUsersForVisibility(keyword string, group string, tag string) (int64, error) {
	db := buildUsersForVisibilityQuery(keyword, group, tag)
	var total int64
	err := db.Count(&total).Error
	return total, err
}

func QueryUsersForVisibilityPage(keyword string, group string, tag string, offset int, limit int) ([]ModelVisibilityUserSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	db := buildUsersForVisibilityQuery(keyword, group, tag).Select(userVisibilitySummarySelect())
	var users []ModelVisibilityUserSummary
	err := db.Order("id DESC").Offset(offset).Limit(limit).Scan(&users).Error
	return users, err
}

func buildUsersForVisibilityQuery(keyword string, group string, tag string) *gorm.DB {
	db := DB.Model(&User{}).Select(userVisibilitySummarySelect())
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		if id, err := strconv.Atoi(keyword); err == nil && id > 0 {
			db = db.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ? OR phone LIKE ? OR id = ?", like, like, like, like, id)
		} else {
			db = db.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ? OR phone LIKE ?", like, like, like, like)
		}
	}
	if group = strings.TrimSpace(group); group != "" {
		db = db.Where(commonGroupCol+" = ?", group)
	}
	if tag = strings.TrimSpace(tag); tag != "" {
		db = db.Where("tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?", tag, tag+",%", "%,"+tag+",%", "%,"+tag)
	}
	return db
}

func ListUsersByVisibilityRules(userIDs []int, userTags []string, userGroups []string, limit int) ([]ModelVisibilityUserSummary, error) {
	userIDs = normalizeVisibilityUserIDs(userIDs)
	userTags = normalizeVisibilityNames(userTags, 64)
	userGroups = normalizeVisibilityNames(userGroups, 64)
	if len(userIDs) == 0 && len(userTags) == 0 && len(userGroups) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	db := DB.Model(&User{}).Select(userVisibilitySummarySelect())
	db = db.Where("1 = 0")
	if len(userIDs) > 0 {
		db = db.Or("id IN ?", userIDs)
	}
	if len(userGroups) > 0 {
		db = db.Or(commonGroupCol+" IN ?", userGroups)
	}
	for _, tag := range userTags {
		db = db.Or("tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?", tag, tag+",%", "%,"+tag+",%", "%,"+tag)
	}
	var users []ModelVisibilityUserSummary
	err := db.Order("id DESC").Limit(limit).Scan(&users).Error
	return users, err
}

func RefreshModelVisibilityCache() {
	InvalidateActiveModelRowsCache()
	RefreshPricing()
}

func userVisibilitySummarySelect() string {
	return "id, username, display_name, email, phone, " + commonGroupCol + " AS " + commonGroupCol + ", tags, role"
}
