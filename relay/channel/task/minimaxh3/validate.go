package minimaxh3

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValidateVideoGenerationV2Req 按官方文档校验创建任务请求。
func ValidateVideoGenerationV2Req(req *VideoGenerationV2Req) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return fmt.Errorf("model is required")
	}
	if _, ok := supportedModels[model]; !ok {
		return fmt.Errorf("model %q is not supported, current available: %s", model, ModelMiniMaxH3)
	}
	req.Model = model

	resolution := normalizeOfficialResolution(req.Resolution)
	if resolution == "" {
		return fmt.Errorf("resolution is required, available: %s, %s", Resolution768P, Resolution2K)
	}
	if _, ok := supportedResolutions[resolution]; !ok {
		return fmt.Errorf("resolution %q is invalid, available: %s, %s", strings.TrimSpace(req.Resolution), Resolution768P, Resolution2K)
	}
	req.Resolution = resolution

	if req.Duration < MinDuration || req.Duration > MaxDuration {
		return fmt.Errorf("duration must be an integer in [%d, %d]", MinDuration, MaxDuration)
	}

	if err := validateContent(req); err != nil {
		return err
	}

	scene := classifyScene(req.Content)
	req.Ratio = strings.TrimSpace(req.Ratio)
	if err := validateRatio(scene, req); err != nil {
		return err
	}
	return nil
}

func validateContent(req *VideoGenerationV2Req) error {
	if len(req.Content) == 0 {
		return fmt.Errorf("content must include a non-empty text item (prompt is required)")
	}

	hasText := false
	var firstFrames, lastFrames, refImages, refVideos, refAudios int
	hasI2VA := false
	hasR2VA := false

	for i, item := range req.Content {
		typ := strings.TrimSpace(item.Type)
		if _, ok := contentTypes[typ]; !ok {
			return fmt.Errorf("content[%d].type %q is invalid", i, item.Type)
		}
		req.Content[i].Type = typ
		role := strings.TrimSpace(item.Role)
		req.Content[i].Role = role

		switch typ {
		case ContentTypeText:
			text := strings.TrimSpace(item.Text)
			if text == "" {
				return fmt.Errorf("content[%d].text must be non-empty", i)
			}
			if utf8.RuneCountInString(item.Text) > MaxTextChars {
				return fmt.Errorf("content[%d].text exceeds %d characters", i, MaxTextChars)
			}
			hasText = true
			if role != "" {
				return fmt.Errorf("content[%d] text item must not set role", i)
			}
		case ContentTypeImageURL:
			if item.ImageURL == nil || strings.TrimSpace(item.ImageURL.URL) == "" {
				return fmt.Errorf("content[%d].image_url.url is required", i)
			}
			item.ImageURL.URL = strings.TrimSpace(item.ImageURL.URL)
			req.Content[i].ImageURL = item.ImageURL
			if err := validateMediaURL(item.ImageURL.URL, ContentTypeImageURL); err != nil {
				return fmt.Errorf("content[%d].image_url.url: %w", i, err)
			}
			switch role {
			case "", RoleFirstFrame:
				firstFrames++
				hasI2VA = true
			case RoleLastFrame:
				lastFrames++
				hasI2VA = true
			case RoleReferenceImage:
				refImages++
				hasR2VA = true
			default:
				return fmt.Errorf("content[%d].role %q is invalid for image_url", i, role)
			}
		case ContentTypeVideoURL:
			if item.VideoURL == nil || strings.TrimSpace(item.VideoURL.URL) == "" {
				return fmt.Errorf("content[%d].video_url.url is required", i)
			}
			item.VideoURL.URL = strings.TrimSpace(item.VideoURL.URL)
			req.Content[i].VideoURL = item.VideoURL
			if err := validateMediaURL(item.VideoURL.URL, ContentTypeVideoURL); err != nil {
				return fmt.Errorf("content[%d].video_url.url: %w", i, err)
			}
			if role == "" {
				role = RoleReferenceVideo
				req.Content[i].Role = role
			}
			if role != RoleReferenceVideo {
				return fmt.Errorf("content[%d].role must be %s for video_url", i, RoleReferenceVideo)
			}
			refVideos++
			hasR2VA = true
		case ContentTypeAudioURL:
			if item.AudioURL == nil || strings.TrimSpace(item.AudioURL.URL) == "" {
				return fmt.Errorf("content[%d].audio_url.url is required", i)
			}
			item.AudioURL.URL = strings.TrimSpace(item.AudioURL.URL)
			req.Content[i].AudioURL = item.AudioURL
			if err := validateMediaURL(item.AudioURL.URL, ContentTypeAudioURL); err != nil {
				return fmt.Errorf("content[%d].audio_url.url: %w", i, err)
			}
			if role == "" {
				role = RoleReferenceAudio
				req.Content[i].Role = role
			}
			if role != RoleReferenceAudio {
				return fmt.Errorf("content[%d].role must be %s for audio_url", i, RoleReferenceAudio)
			}
			refAudios++
			hasR2VA = true
		}
	}

	if !hasText {
		return fmt.Errorf("content must include a non-empty text item (prompt is required)")
	}
	if hasI2VA && hasR2VA {
		return fmt.Errorf("image-to-video roles (first_frame/last_frame) cannot be mixed with reference_* roles")
	}
	if firstFrames > MaxFirstFrames {
		return fmt.Errorf("first_frame image count must be <= %d", MaxFirstFrames)
	}
	if lastFrames > MaxLastFrames {
		return fmt.Errorf("last_frame image count must be <= %d", MaxLastFrames)
	}
	if refImages > MaxReferenceImages {
		return fmt.Errorf("reference_image count must be <= %d", MaxReferenceImages)
	}
	if refVideos > MaxReferenceVideos {
		return fmt.Errorf("reference_video count must be <= %d", MaxReferenceVideos)
	}
	if refAudios > MaxReferenceAudios {
		return fmt.Errorf("reference_audio count must be <= %d", MaxReferenceAudios)
	}
	return nil
}

func validateRatio(scene string, req *VideoGenerationV2Req) error {
	switch scene {
	case SceneTextToVideo:
		if req.Ratio == "" {
			return fmt.Errorf("ratio is required for text-to-video and cannot be %s", RatioAdaptive)
		}
		if _, ok := t2vaRatios[req.Ratio]; !ok {
			return fmt.Errorf("ratio %q is invalid for text-to-video, available: 21:9, 16:9, 4:3, 1:1, 3:4, 9:16", req.Ratio)
		}
	case SceneImageToVideo:
		// 官方：图生视频宽高比由输入图片决定，ratio 恒为 adaptive；传入其他合理值会被忽略。
		if req.Ratio != "" {
			if _, ok := supportedRatios[req.Ratio]; !ok {
				return fmt.Errorf("ratio %q is invalid", req.Ratio)
			}
		}
		req.Ratio = RatioAdaptive
	case SceneReferenceToVideo:
		if req.Ratio == "" {
			req.Ratio = RatioAdaptive
			return nil
		}
		if _, ok := supportedRatios[req.Ratio]; !ok {
			return fmt.Errorf("ratio %q is invalid, available: adaptive, 21:9, 16:9, 4:3, 1:1, 3:4, 9:16", req.Ratio)
		}
	}
	return nil
}

func classifyScene(items []ContentItem) string {
	for _, item := range items {
		role := strings.TrimSpace(item.Role)
		typ := strings.TrimSpace(item.Type)
		if _, ok := r2vaRoles[role]; ok {
			return SceneReferenceToVideo
		}
		if typ == ContentTypeVideoURL || typ == ContentTypeAudioURL {
			return SceneReferenceToVideo
		}
		if _, ok := i2vaRoles[role]; ok {
			return SceneImageToVideo
		}
		if typ == ContentTypeImageURL {
			return SceneImageToVideo
		}
	}
	return SceneTextToVideo
}

func firstNonEmptyText(items []ContentItem) string {
	for _, item := range items {
		if strings.TrimSpace(item.Type) == ContentTypeText {
			if t := strings.TrimSpace(item.Text); t != "" {
				return t
			}
		}
	}
	return ""
}

func validateMediaURL(raw, contentType string) error {
	u := strings.TrimSpace(raw)
	if u == "" {
		return fmt.Errorf("url is empty")
	}
	lower := strings.ToLower(u)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return nil
	case strings.HasPrefix(lower, "mm_file://"):
		if strings.TrimSpace(u[len("mm_file://"):]) == "" {
			return fmt.Errorf("mm_file:// file_id is empty")
		}
		return nil
	case strings.HasPrefix(lower, "data:"):
		return validateDataURI(lower, contentType)
	default:
		return fmt.Errorf("unsupported url scheme, use https URL, mm_file://{file_id}, or data URI")
	}
}

func validateDataURI(lower, contentType string) error {
	switch contentType {
	case ContentTypeImageURL:
		if !strings.HasPrefix(lower, "data:image/") || !strings.Contains(lower, ";base64,") {
			return fmt.Errorf("image data URI must be data:image/<format>;base64,<data>")
		}
	case ContentTypeVideoURL:
		if !strings.HasPrefix(lower, "data:video/") || !strings.Contains(lower, ";base64,") {
			return fmt.Errorf("video data URI must be data:video/mp4;base64,<data>")
		}
	case ContentTypeAudioURL:
		if !strings.HasPrefix(lower, "data:audio/") || !strings.Contains(lower, ";base64,") {
			return fmt.Errorf("audio data URI must be data:audio/<format>;base64,<data>")
		}
	}
	return nil
}
