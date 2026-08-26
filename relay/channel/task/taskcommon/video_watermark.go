package taskcommon

import (
	"bytes"
	"fmt"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

// ForceVideoWatermarkBody applies the administrator video watermark policy to
// a JSON task request. It preserves the original body when the policy does not
// apply or the channel has no supported watermark parameter.
func ForceVideoWatermarkBody(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (io.Reader, error) {
	if c == nil || info == nil || !setting.IsVideoWatermarkForcedForUser(info.UserId) {
		return body, nil
	}
	if !isWatermarkSupportedChannel(info.ChannelType) {
		return body, nil
	}
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return bytes.NewReader(raw), nil
	}
	var root map[string]any
	if err := common.Unmarshal(raw, &root); err != nil || root == nil {
		return bytes.NewReader(raw), nil
	}
	forceWatermark(root, info.ChannelType)
	out, err := common.Marshal(root)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(out), nil
}

func isWatermarkSupportedChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeAli, constant.ChannelTypeAliVideo, constant.ChannelTypeDoubaoVideo, constant.ChannelTypeSeedance, constant.ChannelTypeVolcEngine, constant.ChannelTypeMiniMax, constant.ChannelTypeMiniMaxH3Video:
		return true
	default:
		return false
	}
}

func forceWatermark(root map[string]any, channelType int) {
	if channelType == constant.ChannelTypeMiniMax || channelType == constant.ChannelTypeMiniMaxH3Video {
		root["aigc_watermark"] = true
		return
	}
	if params, ok := root["parameters"].(map[string]any); ok {
		params["watermark"] = true
		return
	}
	root["watermark"] = true
}

// ForceVideoWatermarkRequestBody is the reader-level integration helper used
// by RelayTaskSubmit after adapter conversion or passthrough construction.
func ForceVideoWatermarkRequestBody(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (io.Reader, error) {
	if body == nil {
		return nil, fmt.Errorf("request body is nil")
	}
	return ForceVideoWatermarkBody(c, info, body)
}
