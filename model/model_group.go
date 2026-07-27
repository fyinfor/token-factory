package model

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

// ── 模型归类（智能路由的地基）────────────────────────────────────
//
// 渠道上的「原始模型名」可能形态各异（带日期 / 版本后缀）。
// 路由模式（权重 / 价格优）面向的是「归类」而非每一个零散的原始模型名。
//
// 归类 key 的确定规则：
//  1. 默认按 NormalizeModelName 自动归一化。
//  2. 可通过 ModelGroupOverride 人工把某个原始模型名强制归到指定归类。
//  3. 可通过 ModelGroupMeta 给归类设置展示名 / 备注。

var (
	reTrailingDigits  = regexp.MustCompile(`^\d{4,}$`)
	reTrailingVersion = regexp.MustCompile(`^v\d+$`)
)

var strippableSuffixTokens = map[string]bool{
	"latest":  true,
	"preview": true,
}

// NormalizeModelName 将原始模型名归一化为「归类 key」（不区分大小写）。
func NormalizeModelName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	tokens := strings.Split(s, "-")
	end := len(tokens)
	for end > 1 {
		t := tokens[end-1]
		if reTrailingDigits.MatchString(t) || reTrailingVersion.MatchString(t) || strippableSuffixTokens[t] {
			end--
			continue
		}
		break
	}
	return strings.Join(tokens[:end], "-")
}

// ModelGroupOverride 模型归类的人工覆盖。RawModel 唯一。
type ModelGroupOverride struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RawModel  string    `gorm:"uniqueIndex;size:256;not null" json:"raw_model"`
	GroupKey  string    `gorm:"index;size:256;not null" json:"group_key"`
}

// ModelGroupMeta 归类的展示信息。GroupKey 唯一。
type ModelGroupMeta struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	GroupKey    string    `gorm:"uniqueIndex;size:256;not null" json:"group_key"`
	DisplayName string    `gorm:"size:256" json:"display_name"`
	Description string    `gorm:"size:512" json:"description"`
}

// ModelGroup 表示一个归类及其成员原始模型名（用于 UI 展示）。
type ModelGroup struct {
	GroupKey    string   `json:"group_key"`
	DisplayName string   `json:"display_name"`
	Models      []string `json:"models"`
}

// LoadModelGroupOverrides 加载全部人工覆盖，返回 rawModel(小写) -> groupKey。
func LoadModelGroupOverrides() (map[string]string, error) {
	var rows []ModelGroupOverride
	if err := DB.Find(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		key := strings.ToLower(strings.TrimSpace(r.RawModel))
		if key == "" || r.GroupKey == "" {
			continue
		}
		m[key] = r.GroupKey
	}
	return m, nil
}

// LoadAllModelGroupOverrides 加载全部人工覆盖（原始记录，按 raw_model 升序）。
func LoadAllModelGroupOverrides() ([]ModelGroupOverride, error) {
	var rows []ModelGroupOverride
	if err := DB.Order("raw_model ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ResolveModelGroupKey 返回某原始模型名最终归类 key：优先人工覆盖，否则自动归一化。
func ResolveModelGroupKey(raw string, overrides map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return ""
	}
	if ov, ok := overrides[key]; ok && ov != "" {
		return ov
	}
	return NormalizeModelName(raw)
}

// BuildModelGroups 把一批原始模型名聚合成归类列表。
func BuildModelGroups(rawModels []string) ([]ModelGroup, error) {
	overrides, err := LoadModelGroupOverrides()
	if err != nil {
		return nil, err
	}

	var metas []ModelGroupMeta
	_ = DB.Find(&metas).Error
	displayMap := make(map[string]string, len(metas))
	for _, m := range metas {
		if m.DisplayName != "" {
			displayMap[m.GroupKey] = m.DisplayName
		}
	}

	groupModels := make(map[string]map[string]bool)
	for _, raw := range rawModels {
		r := strings.TrimSpace(raw)
		if r == "" {
			continue
		}
		key := ResolveModelGroupKey(r, overrides)
		if key == "" {
			continue
		}
		if groupModels[key] == nil {
			groupModels[key] = make(map[string]bool)
		}
		groupModels[key][r] = true
	}

	groups := make([]ModelGroup, 0, len(groupModels))
	for key, set := range groupModels {
		models := make([]string, 0, len(set))
		for m := range set {
			models = append(models, m)
		}
		sort.Strings(models)
		display := displayMap[key]
		if display == "" {
			display = key
		}
		groups = append(groups, ModelGroup{
			GroupKey:    key,
			DisplayName: display,
			Models:      models,
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].GroupKey < groups[j].GroupKey })
	return groups, nil
}
