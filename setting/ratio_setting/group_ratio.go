package ratio_setting

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

var groupRatioMap = types.NewRWMap[string, float64]()

var defaultGroupGroupRatio = map[string]map[string]float64{
	"vip": {
		"edit_this": 0.9,
	},
}

var groupGroupRatioMap = types.NewRWMap[string, map[string]float64]()
var groupModelPriceMap = types.NewRWMap[string, map[string]float64]()
var groupModelRatioMap = types.NewRWMap[string, map[string]float64]()
var channelModelPriceMap = types.NewRWMap[string, map[string]float64]()
var channelModelRatioMap = types.NewRWMap[string, map[string]float64]()
var channelCompletionRatioMap = types.NewRWMap[string, map[string]float64]()
var channelCacheRatioMap = types.NewRWMap[string, map[string]float64]()
var channelCreateCacheRatioMap = types.NewRWMap[string, map[string]float64]()
var channelImageRatioMap = types.NewRWMap[string, map[string]float64]()
var channelAudioRatioMap = types.NewRWMap[string, map[string]float64]()
var channelAudioCompletionRatioMap = types.NewRWMap[string, map[string]float64]()
var channelVideoRatioMap = types.NewRWMap[string, map[string]float64]()
var channelVideoCompletionRatioMap = types.NewRWMap[string, map[string]float64]()
var channelVideoPriceMap = types.NewRWMap[string, map[string]float64]()
var supplierModelPriceMap = types.NewRWMap[string, map[string]float64]()
var supplierModelRatioMap = types.NewRWMap[string, map[string]float64]()

var defaultGroupSpecialUsableGroup = map[string]map[string]string{
	"vip": {
		"append_1":   "vip_special_group_1",
		"-:remove_1": "vip_removed_group_1",
	},
}

type GroupRatioSetting struct {
	GroupRatio              *types.RWMap[string, float64]            `json:"group_ratio"`
	GroupGroupRatio         *types.RWMap[string, map[string]float64] `json:"group_group_ratio"`
	GroupSpecialUsableGroup *types.RWMap[string, map[string]string]  `json:"group_special_usable_group"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	groupSpecialUsableGroup := types.NewRWMap[string, map[string]string]()
	groupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)

	groupRatioMap.AddAll(defaultGroupRatio)
	groupGroupRatioMap.AddAll(defaultGroupGroupRatio)

	groupRatioSetting = GroupRatioSetting{
		GroupSpecialUsableGroup: groupSpecialUsableGroup,
		GroupRatio:              groupRatioMap,
		GroupGroupRatio:         groupGroupRatioMap,
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func GetGroupRatioSetting() *GroupRatioSetting {
	if groupRatioSetting.GroupSpecialUsableGroup == nil {
		groupRatioSetting.GroupSpecialUsableGroup = types.NewRWMap[string, map[string]string]()
		groupRatioSetting.GroupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)
	}
	return &groupRatioSetting
}

func GetGroupRatioCopy() map[string]float64 {
	return groupRatioMap.ReadAll()
}

func ContainsGroupRatio(name string) bool {
	_, ok := groupRatioMap.Get(name)
	return ok
}

func GroupRatio2JSONString() string {
	return groupRatioMap.MarshalJSONString()
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFloat64MapFromJSONStringFlexibleWithCallback(groupRatioMap, jsonStr, nil)
}

func GetGroupRatio(name string) float64 {
	ratio, ok := groupRatioMap.Get(name)
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func GetGroupGroupRatio(userGroup, usingGroup string) (float64, bool) {
	gp, ok := groupGroupRatioMap.Get(userGroup)
	if !ok {
		return -1, false
	}
	ratio, ok := gp[usingGroup]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func GetGroupModelPrice(group, model string) (float64, bool) {
	groupPrices, ok := groupModelPriceMap.Get(group)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	price, ok := groupPrices[model]
	if !ok {
		return -1, false
	}
	return price, true
}

func GroupModelPrice2JSONString() string {
	return groupModelPriceMap.MarshalJSONString()
}

func UpdateGroupModelPriceByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupModelPriceMap, jsonStr)
}

func GetGroupModelPriceCopy() map[string]map[string]float64 {
	return groupModelPriceMap.ReadAll()
}

func GetGroupModelRatio(group, model string) (float64, bool) {
	groupRatios, ok := groupModelRatioMap.Get(group)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	ratio, ok := groupRatios[model]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func GroupModelRatio2JSONString() string {
	return groupModelRatioMap.MarshalJSONString()
}

func UpdateGroupModelRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupModelRatioMap, jsonStr)
}

func GetGroupModelRatioCopy() map[string]map[string]float64 {
	return groupModelRatioMap.ReadAll()
}

func normalizeChannelID(channelID int) string {
	if channelID <= 0 {
		return ""
	}
	return strconv.Itoa(channelID)
}

func GetChannelModelPrice(channelID int, model string) (float64, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return -1, false
	}
	channelPrices, ok := channelModelPriceMap.Get(key)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	price, ok := channelPrices[model]
	if !ok {
		return -1, false
	}
	return price, true
}

func ChannelModelPrice2JSONString() string {
	return channelModelPriceMap.MarshalJSONString()
}

func UpdateChannelModelPriceByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelModelPriceMap, jsonStr)
}

func GetChannelModelPriceCopy() map[string]map[string]float64 {
	return channelModelPriceMap.ReadAll()
}

func GetChannelModelRatio(channelID int, model string) (float64, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return -1, false
	}
	channelRatios, ok := channelModelRatioMap.Get(key)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	ratio, ok := channelRatios[model]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func ChannelModelRatio2JSONString() string {
	return channelModelRatioMap.MarshalJSONString()
}

func UpdateChannelModelRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelModelRatioMap, jsonStr)
}

func GetChannelModelRatioCopy() map[string]map[string]float64 {
	return channelModelRatioMap.ReadAll()
}

func getChannelScopedValue(
	channelID int,
	model string,
	m *types.RWMap[string, map[string]float64],
) (float64, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return -1, false
	}
	channelMap, ok := m.Get(key)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	val, ok := channelMap[model]
	if !ok {
		return -1, false
	}
	return val, true
}

func GetChannelCompletionRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelCompletionRatioMap)
}
func ChannelCompletionRatio2JSONString() string {
	return channelCompletionRatioMap.MarshalJSONString()
}
func UpdateChannelCompletionRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelCompletionRatioMap, jsonStr)
}
func GetChannelCompletionRatioCopy() map[string]map[string]float64 {
	return channelCompletionRatioMap.ReadAll()
}

func GetChannelCacheRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelCacheRatioMap)
}
func ChannelCacheRatio2JSONString() string {
	return channelCacheRatioMap.MarshalJSONString()
}
func UpdateChannelCacheRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelCacheRatioMap, jsonStr)
}
func GetChannelCacheRatioCopy() map[string]map[string]float64 {
	return channelCacheRatioMap.ReadAll()
}

func GetChannelCreateCacheRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelCreateCacheRatioMap)
}
func ChannelCreateCacheRatio2JSONString() string {
	return channelCreateCacheRatioMap.MarshalJSONString()
}
func UpdateChannelCreateCacheRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelCreateCacheRatioMap, jsonStr)
}
func GetChannelCreateCacheRatioCopy() map[string]map[string]float64 {
	return channelCreateCacheRatioMap.ReadAll()
}

func GetChannelImageRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelImageRatioMap)
}
func ChannelImageRatio2JSONString() string {
	return channelImageRatioMap.MarshalJSONString()
}
func UpdateChannelImageRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelImageRatioMap, jsonStr)
}
func GetChannelImageRatioCopy() map[string]map[string]float64 {
	return channelImageRatioMap.ReadAll()
}

func GetChannelAudioRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelAudioRatioMap)
}
func ChannelAudioRatio2JSONString() string {
	return channelAudioRatioMap.MarshalJSONString()
}
func UpdateChannelAudioRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelAudioRatioMap, jsonStr)
}
func GetChannelAudioRatioCopy() map[string]map[string]float64 {
	return channelAudioRatioMap.ReadAll()
}

func GetChannelAudioCompletionRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelAudioCompletionRatioMap)
}
func ChannelAudioCompletionRatio2JSONString() string {
	return channelAudioCompletionRatioMap.MarshalJSONString()
}
func UpdateChannelAudioCompletionRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelAudioCompletionRatioMap, jsonStr)
}
func GetChannelAudioCompletionRatioCopy() map[string]map[string]float64 {
	return channelAudioCompletionRatioMap.ReadAll()
}

func GetChannelVideoRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelVideoRatioMap)
}
func ChannelVideoRatio2JSONString() string {
	return channelVideoRatioMap.MarshalJSONString()
}
func UpdateChannelVideoRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelVideoRatioMap, jsonStr)
}
func GetChannelVideoRatioCopy() map[string]map[string]float64 {
	return channelVideoRatioMap.ReadAll()
}

func GetChannelVideoCompletionRatio(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelVideoCompletionRatioMap)
}
func ChannelVideoCompletionRatio2JSONString() string {
	return channelVideoCompletionRatioMap.MarshalJSONString()
}
func UpdateChannelVideoCompletionRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelVideoCompletionRatioMap, jsonStr)
}
func GetChannelVideoCompletionRatioCopy() map[string]map[string]float64 {
	return channelVideoCompletionRatioMap.ReadAll()
}

func GetChannelVideoPrice(channelID int, model string) (float64, bool) {
	return getChannelScopedValue(channelID, model, channelVideoPriceMap)
}
func ChannelVideoPrice2JSONString() string {
	return channelVideoPriceMap.MarshalJSONString()
}
func UpdateChannelVideoPriceByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(channelVideoPriceMap, jsonStr)
}
func GetChannelVideoPriceCopy() map[string]map[string]float64 {
	return channelVideoPriceMap.ReadAll()
}

func normalizeSupplierID(supplierID int) string {
	if supplierID <= 0 {
		return ""
	}
	return strconv.Itoa(supplierID)
}

func GetSupplierModelPrice(supplierID int, model string) (float64, bool) {
	key := normalizeSupplierID(supplierID)
	if key == "" {
		return -1, false
	}
	supplierPrices, ok := supplierModelPriceMap.Get(key)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	price, ok := supplierPrices[model]
	if !ok {
		return -1, false
	}
	return price, true
}

func SupplierModelPrice2JSONString() string {
	return supplierModelPriceMap.MarshalJSONString()
}

func UpdateSupplierModelPriceByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(supplierModelPriceMap, jsonStr)
}

func GetSupplierModelPriceCopy() map[string]map[string]float64 {
	return supplierModelPriceMap.ReadAll()
}

func GetSupplierModelRatio(supplierID int, model string) (float64, bool) {
	key := normalizeSupplierID(supplierID)
	if key == "" {
		return -1, false
	}
	supplierRatios, ok := supplierModelRatioMap.Get(key)
	if !ok {
		return -1, false
	}
	model = FormatMatchingModelName(model)
	ratio, ok := supplierRatios[model]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func SupplierModelRatio2JSONString() string {
	return supplierModelRatioMap.MarshalJSONString()
}

func UpdateSupplierModelRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(supplierModelRatioMap, jsonStr)
}

func GetSupplierModelRatioCopy() map[string]map[string]float64 {
	return supplierModelRatioMap.ReadAll()
}

func GroupGroupRatio2JSONString() string {
	return groupGroupRatioMap.MarshalJSONString()
}

func UpdateGroupGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupGroupRatioMap, jsonStr)
}

func CheckGroupRatio(jsonStr string) error {
	checkGroupRatio := make(map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &checkGroupRatio)
	if err != nil {
		return err
	}
	for name, ratio := range checkGroupRatio {
		if ratio < 0 {
			return errors.New("group ratio must be not less than 0: " + name)
		}
	}
	return nil
}
