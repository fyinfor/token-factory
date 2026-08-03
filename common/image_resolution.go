package common

import (
	"strconv"
	"strings"
)

// FormatImageResolutionLabel 将图片分辨率归一化为 Ai 绘图档位标识（512P / 1K / 2K / 4K）。
// 按实际输出图片短边像素判定（与视频分辨率规则相互独立）：
//
//	512P：短边 ≤ 512px
//	1K：短边 ≤ 1024px（且短边 > 512，即 512＜短边≤1024）
//	2K：1024px ＜ 短边 ≤ 2048px
//	4K：短边 ＞ 2048px
//
// 例如 512×512→512P，1024×1536→1K，1920×1080→2K，2160×3840→4K。
// 入参可为 WxH、512P/1K/2K/4K 等；历史计费别名「1080p」「1K」统一展示为 1K。
func FormatImageResolutionLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	compact := strings.ReplaceAll(s, " ", "")
	lower := strings.ToLower(compact)

	// 显式档位别名优先：历史「1080p」≡ 1K，避免把字面量「1080p」按短边 1080 误判为 2K。
	switch lower {
	case "512p", "512":
		return "512P"
	case "1k", "1080p", "1080":
		return "1K"
	}

	// 已是 2k / 4k 等。
	if strings.HasSuffix(lower, "k") && isDigits(strings.TrimSuffix(lower, "k")) {
		n, _ := strconv.Atoi(strings.TrimSuffix(lower, "k"))
		switch n {
		case 1:
			return "1K"
		case 2:
			return "2K"
		case 4:
			return "4K"
		default:
			return strings.ToUpper(lower)
		}
	}

	// 720p / 480 / 512p 等短边形式：按短边像素映射。
	if isDigits(strings.TrimSuffix(lower, "p")) && (strings.HasSuffix(lower, "p") || isDigits(lower)) {
		n, _ := strconv.Atoi(strings.TrimSuffix(lower, "p"))
		if n > 0 {
			return labelImageFromShortSide(n)
		}
	}

	// WxH 像素：严格按短边像素分档（1920×1080 短边 1080 → 2K）。
	if w, h, ok := parseVideoResolutionLiteral(lower); ok {
		short := w
		if h < short {
			short = h
		}
		return labelImageFromShortSide(short)
	}
	return compact
}

// labelImageFromShortSide 按 Ai 绘图短边像素映射分辨率档位。
func labelImageFromShortSide(short int) string {
	switch {
	case short <= 512:
		return "512P"
	case short <= 1024:
		return "1K"
	case short <= 2048:
		return "2K"
	default:
		return "4K"
	}
}
