package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// seedancePixelSize 预扣费用的真实像素宽高（来自给定映射表，禁止按短边推算）。
type seedancePixelSize struct {
	W int
	H int
}

// seedancePreConsumeSizeTable 按 ratio + 档位读取真实宽高。
// 档位仅覆盖 480p / 720p / 1080p；adaptive 在查找前归一化为 1:1。
var seedancePreConsumeSizeTable = map[string]map[string]seedancePixelSize{
	"16:9": {
		"480p":  {W: 864, H: 496},
		"720p":  {W: 1280, H: 720},
		"1080p": {W: 1920, H: 1080},
	},
	"4:3": {
		"480p":  {W: 752, H: 560},
		"720p":  {W: 1112, H: 834},
		"1080p": {W: 1664, H: 1248},
	},
	"1:1": {
		"480p":  {W: 640, H: 640},
		"720p":  {W: 960, H: 960},
		"1080p": {W: 1440, H: 1440},
	},
	"3:4": {
		"480p":  {W: 560, H: 752},
		"720p":  {W: 834, H: 1112},
		"1080p": {W: 1248, H: 1664},
	},
	"9:16": {
		"480p":  {W: 496, H: 864},
		"720p":  {W: 720, H: 1280},
		"1080p": {W: 1080, H: 1920},
	},
	"21:9": {
		"480p":  {W: 992, H: 432},
		"720p":  {W: 1470, H: 630},
		"1080p": {W: 2206, H: 946},
	},
}

// LookupSeedancePreConsumeSize 按 ratio + 档位取预扣费用的真实宽高。
// ratio=adaptive 时统一使用 1:1 对应档位。
func LookupSeedancePreConsumeSize(ratio, quality string) (realW, realH int, ok bool) {
	ratioKey := normalizeSeedancePreConsumeRatio(ratio)
	qualityKey := normalizeSeedancePreConsumeQuality(quality)
	if ratioKey == "" || qualityKey == "" {
		return 0, 0, false
	}
	byQuality, found := seedancePreConsumeSizeTable[ratioKey]
	if !found {
		return 0, 0, false
	}
	size, found := byQuality[qualityKey]
	if !found || size.W <= 0 || size.H <= 0 {
		return 0, 0, false
	}
	return size.W, size.H, true
}

// SeedanceCalcToken 按真实宽高与时长计算预扣 token。
// 公式必须与给定算法保持一致：fps=24，total_frame = fps*duration+1（+1 参考帧不可删除）。
func SeedanceCalcToken(realW, realH, duration int) int {
	fps := 24
	totalFrame := fps*duration + 1 // 必须+1参考帧，这是硬性要求，不能删掉+1
	token := int64(realW) * int64(realH) * int64(totalFrame) / 1024
	return int(token)
}

// CalcSeedancePreConsumeTokens 视频按 token 预扣数量。
// 入参：ratio、quality(480p/720p/1080p)、duration（秒）；输出待预扣 token 整数。
// 映射不到宽高时回退 SeedanceTokenPreConsumeTokens，避免预扣为 0。
func CalcSeedancePreConsumeTokens(ratio, quality string, duration int) int {
	realW, realH, ok := LookupSeedancePreConsumeSize(ratio, quality)
	if !ok {
		return SeedanceTokenPreConsumeTokens
	}
	return SeedanceCalcToken(realW, realH, duration)
}

// CalcSeedancePreConsumeTokensFromRequest 从任务请求提取 ratio/档位/时长并计算预扣 token。
// 供现有 tryVideoPerTokenRulesPriceData 直接替换固定 50000。
func CalcSeedancePreConsumeTokensFromRequest(req relaycommon.TaskSubmitReq) int {
	return CalcSeedancePreConsumeTokens(
		videoRatioLabelFromRequest(req),
		videoQualityLabelForPreConsume(req),
		videoDurationFromTaskRequest(req),
	)
}

func normalizeSeedancePreConsumeRatio(ratio string) string {
	ratio = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(ratio), " ", ""))
	if ratio == "adaptive" {
		return "1:1"
	}
	return ratio
}

func normalizeSeedancePreConsumeQuality(quality string) string {
	return strings.ToLower(common.FormatVideoResolutionLabel(quality))
}

func videoRatioLabelFromRequest(req relaycommon.TaskSubmitReq) string {
	if r := strings.TrimSpace(req.Ratio); r != "" {
		return r
	}
	return submitMetadataString(req.Metadata, "ratio")
}

func videoQualityLabelForPreConsume(req relaycommon.TaskSubmitReq) string {
	if label := VideoBillingResolutionLabelFromRequest(req); label != "" {
		return label
	}
	if label := common.FormatVideoResolutionLabel(req.Size); label != "" {
		return label
	}
	if s := submitMetadataString(req.Metadata, "size"); s != "" {
		return common.FormatVideoResolutionLabel(s)
	}
	return ""
}
