package operation_setting

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type MonitorSetting struct {
	AutoTestChannelEnabled bool     `json:"auto_test_channel_enabled"`
	AutoTestChannelMinutes float64  `json:"auto_test_channel_minutes"`
	AutoTestModelTags      []string `json:"auto_test_model_tags"`
}

// 默认配置
var monitorSetting = MonitorSetting{
	AutoTestChannelEnabled: false,
	AutoTestChannelMinutes: 10,
	AutoTestModelTags:      []string{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("monitor_setting", &monitorSetting)
}

func GetMonitorSetting() *MonitorSetting {
	if os.Getenv("CHANNEL_TEST_FREQUENCY") != "" {
		frequency, err := strconv.Atoi(os.Getenv("CHANNEL_TEST_FREQUENCY"))
		if err == nil && frequency > 0 {
			monitorSetting.AutoTestChannelEnabled = true
			monitorSetting.AutoTestChannelMinutes = float64(frequency)
		}
	}
	return &monitorSetting
}

const autoTestModelTagsMustBeArray = "monitor_setting.auto_test_model_tags 必须为 JSON 字符串数组，例如 [\"文本\"]"

// ParseAutoTestModelTagsJSON 只接受 JSON 数组。字符串、null、空串、非字符串元素一律拒绝。
func ParseAutoTestModelTagsJSON(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf(autoTestModelTagsMustBeArray)
	}
	if common.GetJsonType(json.RawMessage(trimmed)) != "array" {
		return nil, fmt.Errorf(autoTestModelTagsMustBeArray)
	}
	var tags []string
	if err := common.UnmarshalJsonStr(trimmed, &tags); err != nil {
		return nil, fmt.Errorf(autoTestModelTagsMustBeArray)
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return nil, fmt.Errorf("monitor_setting.auto_test_model_tags 不允许空字符串元素")
		}
		if _, ok := seen[tag]; ok {
			return nil, fmt.Errorf("monitor_setting.auto_test_model_tags 不允许重复标签: %s", tag)
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out, nil
}
