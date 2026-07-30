package common

import (
	"strconv"
	"strings"
)

// FormatImageResolutionLabel 将图片分辨率归一化为 Ai 绘图档位标识（1080p / 2K / 4K）。
// 按实际输出图片短边像素：短边 ≤1024 → 1080p（≡1K）；1024＜短边≤2048 → 2K；短边＞2048 → 4K。
// 例如 1024×1536→1080p，1536×2048→2K，1920×1080→2K，2160×3840→4K。
// 入参可为 WxH、1K/2K/4K、1080p 等；显式「1K」「1080p」统一展示为 1080p。
func FormatImageResolutionLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	compact := strings.ReplaceAll(s, " ", "")
	lower := strings.ToLower(compact)

	// 显式档位别名优先：1K ≡ 1080p，避免把字面量「1080p」按短边 1080 误判为 2K。
	if lower == "1k" || lower == "1080p" || lower == "1080" {
		return "1080p"
	}

	// 已是 2k / 4k 等。
	if strings.HasSuffix(lower, "k") && isDigits(strings.TrimSuffix(lower, "k")) {
		n, _ := strconv.Atoi(strings.TrimSuffix(lower, "k"))
		switch n {
		case 2:
			return "2K"
		case 4:
			return "4K"
		default:
			return strings.ToUpper(lower)
		}
	}

	// 720p / 480 等短边形式：按短边像素映射。
	if isDigits(strings.TrimSuffix(lower, "p")) && (strings.HasSuffix(lower, "p") || isDigits(lower)) {
		n, _ := strconv.Atoi(strings.TrimSuffix(lower, "p"))
		if n > 0 {
			return labelImageFromShortSide(n)
		}
	}

	// WxH 像素：严格按短边阈值分档（1920×1080 短边 1080 → 2K）。
	if w, h, ok := parseVideoResolutionLiteral(lower); ok {
		short := w
		if h < short {
			short = h
		}
		return labelImageFromShortSide(short)
	}
	return compact
}

// labelImageFromShortSide 按 Ai 绘图短边阈值映射分辨率档位（低档统一展示为 1080p）。
func labelImageFromShortSide(short int) string {
	switch {
	case short <= 1024:
		return "1080p"
	case short <= 2048:
		return "2K"
	default:
		return "4K"
	}
}
