package common

import (
	"net/url"
	"strings"
)

const (
	VideoBillingModeTextToVideo  = "text_to_video"
	VideoBillingModeImageToVideo = "image_to_video"
	VideoBillingModeVideoToVideo = "video_to_video"
)

// DetectVideoBillingMode classifies the user supplied source media for video
// billing. Keep it request-shape based: upstream adaptors may remap fields, but
// billing should reflect the user's image/video input lane.
func DetectVideoBillingMode(req *TaskSubmitReq) string {
	if req == nil {
		return VideoBillingModeTextToVideo
	}
	if metadataHasMedia(req.Metadata, "video_urls") {
		return VideoBillingModeVideoToVideo
	}
	if isVideoMediaRef(req.InputReference) {
		return VideoBillingModeVideoToVideo
	}
	for _, img := range req.Images {
		if isVideoMediaRef(img) {
			return VideoBillingModeVideoToVideo
		}
	}
	if strings.TrimSpace(req.InputReference) != "" ||
		strings.TrimSpace(req.Image) != "" ||
		hasNonEmptyString(req.Images) {
		return VideoBillingModeImageToVideo
	}
	return VideoBillingModeTextToVideo
}

func hasNonEmptyString(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func metadataHasMedia(metadata map[string]interface{}, key string) bool {
	if len(metadata) == 0 {
		return false
	}
	return metadataValueHasMedia(metadata[key])
}

func metadataValueHasMedia(value any) bool {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case []string:
		return hasNonEmptyString(v)
	case []interface{}:
		for _, item := range v {
			if metadataValueHasMedia(item) {
				return true
			}
		}
	}
	return false
}

func isVideoMediaRef(raw string) bool {
	ref := strings.TrimSpace(raw)
	if ref == "" {
		return false
	}
	lower := strings.ToLower(ref)
	if strings.HasPrefix(lower, "data:video/") {
		return true
	}
	if strings.HasPrefix(lower, "data:image/") {
		return false
	}
	path := ref
	if u, err := url.Parse(ref); err == nil && u.Path != "" {
		path = u.Path
	}
	path = strings.ToLower(strings.TrimSpace(path))
	for _, ext := range []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
