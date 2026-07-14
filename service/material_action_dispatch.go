package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// MaterialActionHandler 终端用户 Action 业务处理函数。
type MaterialActionHandler func(userId int, body []byte) (any, error)

var materialActionRegistry map[string]MaterialActionHandler

func init() {
	materialActionRegistry = map[string]MaterialActionHandler{
		MaterialActionCreateAssetGroup:            dispatchCreateAssetGroup,
		MaterialActionGetAssetGroup:               dispatchGetAssetGroup,
		MaterialActionCreateAsset:                 dispatchCreateAsset,
		MaterialActionGetAsset:                    dispatchGetAsset,
		MaterialActionUpdateAssetGroup:            dispatchUpdateAssetGroup,
		MaterialActionUpdateAsset:                 dispatchUpdateAsset,
		MaterialActionDeleteAsset:                 dispatchDeleteAsset,
		MaterialActionDeleteAssetGroup:            dispatchDeleteAssetGroup,
		MaterialActionCreateVisualValidateSession: dispatchCreateVisualValidateSession,
		MaterialActionGetVisualValidateResult:     dispatchGetVisualValidateResult,
	}
}

// NormalizeMaterialAction 规范化 Action 名称（去空白）。
func NormalizeMaterialAction(action string) string {
	return strings.TrimSpace(action)
}

// IsSupportedMaterialAction 判断 Action 是否已注册。
func IsSupportedMaterialAction(action string) bool {
	_, ok := materialActionRegistry[NormalizeMaterialAction(action)]
	return ok
}

// SupportedMaterialActions 返回已注册 Action 列表（用于日志/调试）。
func SupportedMaterialActions() []string {
	actions := make([]string, 0, len(materialActionRegistry))
	for action := range materialActionRegistry {
		actions = append(actions, action)
	}
	return actions
}

// DispatchMaterialAction 按 Action 分发到对应业务处理函数。
func DispatchMaterialAction(userId int, action string, body []byte) (any, error) {
	action = NormalizeMaterialAction(action)
	if action == "" {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "Action 参数不能为空")
	}
	handler, ok := materialActionRegistry[action]
	if !ok {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "不支持的 Action: "+action)
	}
	return handler(userId, body)
}

func dispatchCreateAssetGroup(userId int, body []byte) (any, error) {
	var input CreateAssetGroupInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionCreateAssetGroup(userId, input)
}

func dispatchGetAssetGroup(userId int, body []byte) (any, error) {
	var input AssetGroupIdInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionGetAssetGroup(userId, input)
}

func dispatchCreateAsset(userId int, body []byte) (any, error) {
	var input CreateAssetInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionCreateAsset(userId, input)
}

func dispatchGetAsset(userId int, body []byte) (any, error) {
	var input AssetGroupIdInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionGetAsset(userId, input)
}

func dispatchUpdateAssetGroup(userId int, body []byte) (any, error) {
	var input UpdateAssetGroupInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionUpdateAssetGroup(userId, input)
}

func dispatchUpdateAsset(userId int, body []byte) (any, error) {
	var input UpdateAssetInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionUpdateAsset(userId, input)
}

func dispatchDeleteAsset(userId int, body []byte) (any, error) {
	var input AssetGroupIdInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionDeleteAsset(userId, input)
}

func dispatchDeleteAssetGroup(userId int, body []byte) (any, error) {
	var input AssetGroupIdInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionDeleteAssetGroup(userId, input)
}

func dispatchCreateVisualValidateSession(userId int, body []byte) (any, error) {
	return ActionCreateVisualValidateSession(userId)
}

func dispatchGetVisualValidateResult(userId int, body []byte) (any, error) {
	var input VisualValidateResultInput
	if err := common.Unmarshal(body, &input); err != nil {
		return nil, materialActionErr(common.MaterialCodeInvalidParameter, "请求参数 JSON 格式无效")
	}
	return ActionGetVisualValidateResult(userId, input)
}

// LogMaterialActionRegistry 启动时打印已注册 Action（便于排查部署版本）。
func LogMaterialActionRegistry() {
	common.SysLog(fmt.Sprintf("[material-action] registered actions: %s", strings.Join(SupportedMaterialActions(), ", ")))
}
