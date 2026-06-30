package common

import (
	"math"
	"strconv"
	"strings"
)

// ParseVideoResolutionAndRatio maps Seedance/VolcEngine resolution + aspect ratio to pixel size.
// resolution examples: 480p, 720p, 854x480; ratio examples: 16:9, 9:16.
func ParseVideoResolutionAndRatio(resolution, ratio string) (width, height int, ok bool) {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	ratio = strings.TrimSpace(ratio)
	if resolution != "" {
		if w, h, parsed := parseVideoResolutionLiteral(resolution); parsed {
			return w, h, true
		}
		if r := parseAspectRatioFloat(ratio); r > 0 {
			if w, h, parsed := parseVideoResolutionWithAspect(resolution, r); parsed {
				return w, h, true
			}
		}
		if w, h, parsed := parseVideoResolutionPreset(resolution); parsed {
			return w, h, true
		}
	}
	return 0, 0, false
}

// ResolveVideoDimensionsFromRequest 从任务请求字段解析像素尺寸（供预扣/结算按分辨率匹配 token 单价）。
// 优先级：metadata（含 resolution/ratio）> 顶层 resolution > size（支持 480p 或 WxH）。
func ResolveVideoDimensionsFromRequest(size, resolution, ratio string, metadata map[string]interface{}) (width, height int, ok bool) {
	if w, h, parsed := VideoDimensionsFromMetadata(metadata); parsed {
		return w, h, true
	}
	r := strings.TrimSpace(ratio)
	if r == "" && metadata != nil {
		r = metadataString(metadata, "ratio")
	}
	if res := strings.TrimSpace(resolution); res != "" {
		if w, h, parsed := ParseVideoResolutionAndRatio(res, r); parsed {
			return w, h, true
		}
	}
	if s := strings.TrimSpace(size); s != "" {
		if w, h, parsed := ParseVideoResolutionAndRatio(s, r); parsed {
			return w, h, true
		}
		if w, h, parsed := parseVideoResolutionLiteral(strings.ToLower(s)); parsed {
			return w, h, true
		}
	}
	return 0, 0, false
}

// VideoDimensionsFromMetadata reads width/height or resolution[/ratio] from task metadata.
func VideoDimensionsFromMetadata(metadata map[string]interface{}) (width, height int, ok bool) {
	if metadata == nil {
		return 0, 0, false
	}
	w := coercePositiveInt(metadata["width"])
	h := coercePositiveInt(metadata["height"])
	if w > 0 && h > 0 {
		return w, h, true
	}
	if size, _ := metadata["size"].(string); strings.TrimSpace(size) != "" {
		if pw, ph, parsed := parseVideoResolutionLiteral(strings.ToLower(strings.TrimSpace(size))); parsed {
			return pw, ph, true
		}
	}
	resolution, _ := metadata["resolution"].(string)
	ratio, _ := metadata["ratio"].(string)
	return ParseVideoResolutionAndRatio(resolution, ratio)
}

func parseVideoResolutionLiteral(s string) (int, int, bool) {
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

func parseVideoResolutionPreset(s string) (int, int, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "480", "480p":
		return 854, 480, true
	case "540", "540p":
		return 960, 540, true
	case "720", "720p":
		return 1280, 720, true
	case "1080", "1080p":
		return 1920, 1080, true
	case "2k":
		return 2560, 1440, true
	case "4k":
		return 3840, 2160, true
	default:
		return 0, 0, false
	}
}

func parseVideoResolutionWithAspect(resolution string, aspect float64) (int, int, bool) {
	shortSide := 0
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "480", "480p":
		shortSide = 480
	case "540", "540p":
		shortSide = 540
	case "720", "720p":
		shortSide = 720
	case "1080", "1080p":
		shortSide = 1080
	case "2k":
		shortSide = 1440
	case "4k":
		shortSide = 2160
	default:
		return 0, 0, false
	}
	if aspect <= 0 {
		aspect = 16.0 / 9.0
	}
	longSide := int(math.Ceil(float64(shortSide) * aspect))
	return longSide, shortSide, true
}

func parseAspectRatioFloat(ratio string) float64 {
	ratio = strings.TrimSpace(ratio)
	if ratio == "" {
		return 16.0 / 9.0
	}
	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return 16.0 / 9.0
	}
	w, errW := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	h, errH := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return 16.0 / 9.0
	}
	return w / h
}

// FormatVideoResolutionLabel 将任意分辨率输入归一化为「分辨率标识」展示值（如 480p / 720p / 2K / 4K）。
// 业务约束：日志/前端展示禁止渲染像素尺寸（如 1280x720），统一转为分辨率档位标识。
// 入参可为：480p、720、854x480 等；无法识别时原样去空格返回。
func FormatVideoResolutionLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	compact := strings.ReplaceAll(s, " ", "")
	lower := strings.ToLower(compact)

	// 已是 720p / 480 等短边形式：规范化为「数字 + p」。
	if isDigits(strings.TrimSuffix(lower, "p")) && (strings.HasSuffix(lower, "p") || isDigits(lower)) {
		n, _ := strconv.Atoi(strings.TrimSuffix(lower, "p"))
		if n > 0 {
			return labelFromShortSide(n)
		}
	}
	// 2k / 4k / 8k 形式。
	if strings.HasSuffix(lower, "k") && isDigits(strings.TrimSuffix(lower, "k")) {
		return strings.ToUpper(lower)
	}
	// WxH 像素形式：取短边换算为分辨率档位标识。
	if w, h, ok := parseVideoResolutionLiteral(lower); ok {
		short := w
		if h < short {
			short = h
		}
		return labelFromShortSide(short)
	}
	return compact
}

// labelFromShortSide 依据短边像素映射到主流分辨率档位标识。
func labelFromShortSide(short int) string {
	switch {
	case short >= 4320:
		return "8K"
	case short >= 2160:
		return "4K"
	case short >= 1440:
		return "2K"
	case short >= 1080:
		return "1080p"
	case short >= 720:
		return "720p"
	case short >= 540:
		return "540p"
	case short >= 480:
		return "480p"
	case short >= 360:
		return "360p"
	case short >= 240:
		return "240p"
	default:
		return strconv.Itoa(short) + "p"
	}
}

// FormatVideoSpecLabel 组合「分辨率标识 + 画面比例」展示值，如 480p 16:9。
// 业务约束：禁止渲染像素尺寸；分辨率走 FormatVideoResolutionLabel 归一化，
// 画面比例原样展示（如 16:9）。任一缺失时仅展示另一项。
func FormatVideoSpecLabel(resolution, ratio string) string {
	label := FormatVideoResolutionLabel(resolution)
	ratio = strings.TrimSpace(ratio)
	switch {
	case label != "" && ratio != "":
		return label + " " + ratio
	case label != "":
		return label
	default:
		return ratio
	}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func coercePositiveInt(v any) int {
	switch x := v.(type) {
	case int:
		if x > 0 {
			return x
		}
	case int64:
		if x > 0 {
			return int(x)
		}
	case float64:
		if x > 0 {
			return int(math.Ceil(x))
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(x)); err == nil && i > 0 {
			return i
		}
	}
	return 0
}
