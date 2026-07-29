package common

import (
	"strconv"
	"strings"
)

// FormatImageResolutionLabel 将图片分辨率归一化为 Ai 绘图档位标识（1080p / 2K / 4K）。
// 短边规则：短边 ≤1024 → 1080p（与 1K 同档）；1024＜短边≤2048 → 2K；短边＞2048 → 4K。
// 入参可为 WxH、1K/2K/4K、1080p 等；1K 与 1080p 统一展示为 1080p。
func FormatImageResolutionLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	compact := strings.ReplaceAll(s, " ", "")
	lower := strings.ToLower(compact)

	// 1K ≡ 1080p：显式档位别名优先，避免把「1080p」按短边 1080 误判为 2K。
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

	// WxH 像素。定价编辑器将「1080p」存为 1920x1080，需与 1K/1080p 同档，
	// 不可按短边 1080 误判为 2K（否则操练场档位去重后会丢掉 1080p）。
	if w, h, ok := parseVideoResolutionLiteral(lower); ok {
		if isClassicImage1080pPixels(w, h) {
			return "1080p"
		}
		short := w
		if h < short {
			short = h
		}
		return labelImageFromShortSide(short)
	}
	return compact
}

// isClassicImage1080pPixels 识别定价侧常见的 1080p 像素写法（横/竖 16:9）。
func isClassicImage1080pPixels(w, h int) bool {
	return (w == 1920 && h == 1080) || (w == 1080 && h == 1920)
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
