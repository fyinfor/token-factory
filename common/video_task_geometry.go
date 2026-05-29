package common

import (
	"math"
	"strings"
)

// NormalizeTaskVideoMetadata merges size (and optional width/height) into metadata.
// When resolution is absent, derives VolcEngine/Seedance resolution + ratio from pixels.
func NormalizeTaskVideoMetadata(metadata map[string]interface{}, size string, width, height *int) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if width != nil && *width > 0 {
		metadata["width"] = *width
	}
	if height != nil && *height > 0 {
		metadata["height"] = *height
	}
	if s := strings.TrimSpace(size); s != "" {
		metadata["size"] = s
	}

	w, h, ok := VideoDimensionsFromMetadata(metadata)
	if !ok || w <= 0 || h <= 0 {
		if pw, ph, parsed := parseVideoResolutionLiteral(strings.ToLower(strings.TrimSpace(size))); parsed {
			metadata["width"] = pw
			metadata["height"] = ph
			w, h = pw, ph
			ok = true
		}
	}
	if ok && w > 0 && h > 0 {
		ensureVolcEngineResolutionRatio(metadata, w, h)
	}
	return metadata
}

func ensureVolcEngineResolutionRatio(metadata map[string]interface{}, w, h int) {
	if metadata == nil || w <= 0 || h <= 0 {
		return
	}
	res := metadataString(metadata, "resolution")
	if res != "" {
		if rw, rh, ok := parseVideoResolutionLiteral(strings.ToLower(res)); ok {
			metadata["resolution"] = PixelsToResolution(rw, rh)
			if metadataString(metadata, "ratio") == "" {
				metadata["ratio"] = PixelsToRatio(rw, rh)
			}
		}
		return
	}
	metadata["resolution"] = PixelsToResolution(w, h)
	if metadataString(metadata, "ratio") == "" {
		metadata["ratio"] = PixelsToRatio(w, h)
	}
}

// PixelsToResolution maps pixel dimensions to Seedance-style resolution (480p, 720p, …).
func PixelsToResolution(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	short := width
	if height < short {
		short = height
	}
	switch {
	case short >= 1080:
		return "1080p"
	case short >= 720:
		return "720p"
	case short >= 540:
		return "540p"
	default:
		return "480p"
	}
}

// PixelsToRatio maps pixel dimensions to the nearest supported aspect ratio label.
func PixelsToRatio(width, height int) string {
	if width <= 0 || height <= 0 {
		return "16:9"
	}
	ratio := float64(width) / float64(height)
	candidates := []struct {
		value string
		ratio float64
	}{
		{"16:9", 16.0 / 9.0},
		{"9:16", 9.0 / 16.0},
		{"1:1", 1.0},
		{"4:3", 4.0 / 3.0},
		{"3:4", 3.0 / 4.0},
		{"21:9", 21.0 / 9.0},
	}
	best := "16:9"
	bestDiff := math.MaxFloat64
	for _, c := range candidates {
		if diff := math.Abs(ratio - c.ratio); diff < bestDiff {
			bestDiff = diff
			best = c.value
		}
	}
	return best
}

func metadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	v, ok := metadata[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
