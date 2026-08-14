package minimaxh3

import (
	"strings"

	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*VideoGenerationV2Req, error) {
	modelName := taskcommon.RelayTaskUpstreamModel(info, req.Model)
	if modelName == "" {
		modelName = DefaultModel
	}

	payload := &VideoGenerationV2Req{
		Model:      modelName,
		Duration:   req.Duration,
		Resolution: strings.TrimSpace(req.Resolution),
		Ratio:      strings.TrimSpace(req.Ratio),
	}
	if err := req.UnmarshalMetadata(payload); err != nil {
		return nil, err
	}
	payload.Model = modelName

	if payload.Duration <= 0 {
		payload.Duration = DefaultDuration
	}
	if mapped := normalizeOfficialResolution(payload.Resolution); mapped != "" {
		payload.Resolution = mapped
	} else {
		payload.Resolution = resolveResolution(req)
	}
	if strings.TrimSpace(payload.Ratio) == "" {
		payload.Ratio = strings.TrimSpace(req.Ratio)
	}

	if len(payload.Content) == 0 {
		payload.Content = buildContentFromTaskReq(req)
	}
	if err := ValidateVideoGenerationV2Req(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func resolveResolution(req *relaycommon.TaskSubmitReq) string {
	if req == nil {
		return DefaultResolution
	}
	if r := strings.TrimSpace(req.Resolution); r != "" {
		if mapped := mapResolution(r); mapped != "" {
			return mapped
		}
	}
	if s := strings.TrimSpace(req.Size); s != "" {
		if mapped := mapResolution(s); mapped != "" {
			return mapped
		}
	}
	if req.Metadata != nil {
		if v, ok := req.Metadata["resolution"].(string); ok {
			if mapped := mapResolution(v); mapped != "" {
				return mapped
			}
		}
	}
	return DefaultResolution
}

// normalizeOfficialResolution 将客户端/计费侧别名归一为官方枚举：768P、2K。
// 兼容 768p、768、1366x768、2k 等写法。
func normalizeOfficialResolution(raw string) string {
	return mapResolution(raw)
}

func mapResolution(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "×", "X")
	if s == "" {
		return ""
	}
	switch {
	case s == Resolution2K || s == "2K" || strings.HasPrefix(s, "2K") || strings.Contains(s, "2048") || strings.Contains(s, "1440") || strings.Contains(s, "2560"):
		return Resolution2K
	case s == Resolution768P || s == "768P" || s == "768" || strings.Contains(s, "768"):
		return Resolution768P
	default:
		if _, ok := supportedResolutions[s]; ok {
			return s
		}
		return ""
	}
}

func buildContentFromTaskReq(req *relaycommon.TaskSubmitReq) []ContentItem {
	items := make([]ContentItem, 0, 8)
	prompt := strings.TrimSpace(req.Prompt)
	if prompt != "" {
		items = append(items, ContentItem{Type: ContentTypeText, Text: prompt})
	}

	videoURLs := collectMetadataURLs(req, "video_urls", "video_url")
	audioURLs := collectMetadataURLs(req, "audio_urls", "audio_url")
	refImages := collectMetadataURLs(req, "reference_images", "reference_image")

	first := strings.TrimSpace(req.ResolveFirstFrameURL())
	last := strings.TrimSpace(req.ResolveLastFrameURL())
	images := compactStrings(req.Images)
	if first == "" && last == "" && len(refImages) == 0 && len(videoURLs) == 0 && len(audioURLs) == 0 {
		if len(images) > 0 {
			first = images[0]
			if len(images) > 1 {
				last = images[1]
			}
		}
	}

	isR2VA := len(videoURLs) > 0 || len(audioURLs) > 0 || len(refImages) > 0 ||
		metadataHasR2VARole(req)

	if isR2VA {
		for _, u := range refImages {
			items = append(items, imageItem(u, RoleReferenceImage))
		}
		for _, u := range images {
			if u == first || u == last {
				continue
			}
			items = append(items, imageItem(u, RoleReferenceImage))
		}
		if first != "" {
			items = append(items, imageItem(first, RoleReferenceImage))
		}
		if last != "" && last != first {
			items = append(items, imageItem(last, RoleReferenceImage))
		}
		for _, u := range videoURLs {
			items = append(items, ContentItem{
				Type:     ContentTypeVideoURL,
				VideoURL: &MediaURL{URL: u},
				Role:     RoleReferenceVideo,
			})
		}
		for _, u := range audioURLs {
			items = append(items, ContentItem{
				Type:     ContentTypeAudioURL,
				AudioURL: &MediaURL{URL: u},
				Role:     RoleReferenceAudio,
			})
		}
		return items
	}

	if first != "" {
		items = append(items, imageItem(first, RoleFirstFrame))
	}
	if last != "" {
		items = append(items, imageItem(last, RoleLastFrame))
	}
	return items
}

func imageItem(url, role string) ContentItem {
	return ContentItem{
		Type:     ContentTypeImageURL,
		ImageURL: &MediaURL{URL: strings.TrimSpace(url)},
		Role:     role,
	}
}

func metadataHasR2VARole(req *relaycommon.TaskSubmitReq) bool {
	if req == nil || req.Metadata == nil {
		return false
	}
	for _, key := range []string{"role", "scene"} {
		if v, ok := req.Metadata[key].(string); ok {
			s := strings.ToLower(strings.TrimSpace(v))
			if strings.Contains(s, "reference") || s == SceneReferenceToVideo || s == "r2v" {
				return true
			}
		}
	}
	return false
}

func collectMetadataURLs(req *relaycommon.TaskSubmitReq, listKey, singleKey string) []string {
	if req == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		u := strings.TrimSpace(raw)
		if u == "" {
			return
		}
		key := strings.ToLower(u)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, u)
	}
	if req.Metadata != nil {
		appendAnyURLs(req.Metadata[listKey], add)
		if v, ok := req.Metadata[singleKey].(string); ok {
			add(v)
		}
	}
	if singleKey == "video_url" && req.InputReference != "" && looksLikeVideoURL(req.InputReference) {
		add(req.InputReference)
	}
	return out
}

func appendAnyURLs(raw any, add func(string)) {
	switch arr := raw.(type) {
	case []string:
		for _, s := range arr {
			add(s)
		}
	case []any:
		for _, it := range arr {
			if s, ok := it.(string); ok {
				add(s)
			}
		}
	case string:
		add(arr)
	}
}

func compactStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{})
	for _, s := range in {
		u := strings.TrimSpace(s)
		if u == "" {
			continue
		}
		key := strings.ToLower(u)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, u)
	}
	return out
}

func looksLikeVideoURL(raw string) bool {
	u := strings.ToLower(strings.TrimSpace(raw))
	if u == "" {
		return false
	}
	if strings.HasPrefix(u, "data:video/") {
		return true
	}
	for _, ext := range []string{".mp4", ".mov"} {
		if strings.Contains(u, ext) {
			return true
		}
	}
	return false
}

func taskSubmitFromNative(req *VideoGenerationV2Req) relaycommon.TaskSubmitReq {
	out := relaycommon.TaskSubmitReq{
		Model:      req.Model,
		Prompt:     firstNonEmptyText(req.Content),
		Duration:   req.Duration,
		Resolution: req.Resolution,
		Ratio:      req.Ratio,
		Metadata:   map[string]any{},
	}
	var images, videos, audios []string
	for _, item := range req.Content {
		switch strings.TrimSpace(item.Type) {
		case ContentTypeImageURL:
			if item.ImageURL != nil {
				u := strings.TrimSpace(item.ImageURL.URL)
				if u == "" {
					continue
				}
				images = append(images, u)
				switch strings.TrimSpace(item.Role) {
				case RoleLastFrame:
					out.LastFrameURL = u
				case RoleFirstFrame, "":
					if out.FirstFrameURL == "" {
						out.FirstFrameURL = u
					}
				}
			}
		case ContentTypeVideoURL:
			if item.VideoURL != nil {
				if u := strings.TrimSpace(item.VideoURL.URL); u != "" {
					videos = append(videos, u)
				}
			}
		case ContentTypeAudioURL:
			if item.AudioURL != nil {
				if u := strings.TrimSpace(item.AudioURL.URL); u != "" {
					audios = append(audios, u)
				}
			}
		}
	}
	out.Images = images
	if len(videos) > 0 {
		out.Metadata["video_urls"] = videos
	}
	if len(audios) > 0 {
		out.Metadata["audio_urls"] = audios
	}
	if req.CallbackURL != "" {
		out.Metadata["callback_url"] = req.CallbackURL
	}
	if req.AigcWatermark != nil {
		out.Metadata["aigc_watermark"] = *req.AigcWatermark
	}
	return out
}
