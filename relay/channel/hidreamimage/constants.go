package hidreamimage

import "strings"

const ChannelName = "hidream-image"

// ImageModelList 智象未来图像模型（MaaS 异步 /v1/images/generations 接口）
var ImageModelList = []string{
	"hidream-O1-Image",
	"HiDream-O1-Image-Dev-2604",
	"HiDream-O1-Image-1.5",
	"hidream-H4.5-image",
	"hidream-H4.0-image",
	"hidream-Q3-pro-image",
	"hidream-Q3-std-image",
	"hidream-Q1-image",
	"hidream-v2.1-standard",
	"hidream-v1-image",
	"hidream-outpainting-v1",
	"hidream-superresolution-v1",
	"hidream-image-erase-v1",
	"hidream-segment-v1",
	"hidream-translate-v1",
	"hidream-tryon-v1",
}

// modelIDMap maps user-facing model names to upstream model_id values.
var modelIDMap = map[string]string{
	"hidream-o1-image":           "HiDream-O1-Image-Dev-2604",
	"hidream-h4.5-image":           "Image-qyoyq2bi",
	"hidream-h4.0-image":           "Image-84977rs4",
	"hidream-q3-pro-image":         "Image-2zvskglc",
	"hidream-q3-std-image":         "Image-f3qa3lph",
	"hidream-q1-image":             "Image-aw47boi4",
	"hidream-tryon-v1":             "Image-zksa5by1",
	"hidream-translate-v1":         "Image-eccut7dd",
	"hidream-image-erase-v1":       "Image-q5580iqf",
	"hidream-outpainting-v1":       "Image-580g70sx",
	"hidream-superresolution-v1":   "Image-11lqtks8",
	"hidream-o1-image-dev-2604":    "HiDream-O1-Image-Dev-2604",
	"hidream-o1-image-1.5":         "HiDream-O1-Image-1.5",
}

func resolveModelID(modelName string) string {
	name := strings.TrimSpace(modelName)
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if id, ok := modelIDMap[lower]; ok {
		return id
	}
	// Allow channel model_mapping to map directly to upstream model_id.
	if strings.HasPrefix(name, "Image-") || strings.HasPrefix(name, "HiDream-") {
		return name
	}
	return ""
}

type modelSeries int

const (
	seriesUnknown modelSeries = iota
	seriesO
	seriesH
	seriesQ
	seriesTool
)

func detectSeries(modelName string) modelSeries {
	lower := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.Contains(lower, "hidream-o"):
		return seriesO
	case strings.Contains(lower, "hidream-h"):
		return seriesH
	case strings.Contains(lower, "hidream-q"):
		return seriesQ
	case strings.Contains(lower, "tryon"),
		strings.Contains(lower, "translate"),
		strings.Contains(lower, "erase"),
		strings.Contains(lower, "outpainting"),
		strings.Contains(lower, "superresolution"),
		strings.Contains(lower, "segment"):
		return seriesTool
	default:
		return seriesUnknown
	}
}

func isToolModel(modelName string) bool {
	return detectSeries(modelName) == seriesTool
}
