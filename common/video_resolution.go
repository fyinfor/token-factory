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
