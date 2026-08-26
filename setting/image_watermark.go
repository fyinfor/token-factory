package setting

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	ImageWatermarkPolicyOff   = "off"
	ImageWatermarkPolicyAll   = "all"
	ImageWatermarkPolicyUsers = "users"

	ImageWatermarkTypeText = "text"
	ImageWatermarkTypeLogo = "logo"

	ImageWatermarkPositionBottomRight = "bottom-right"
	ImageWatermarkPositionBottomLeft  = "bottom-left"
	ImageWatermarkPositionTopRight    = "top-right"
	ImageWatermarkPositionTopLeft     = "top-left"
	ImageWatermarkPositionCenter      = "center"

	ImageWatermarkFailureModeBlock       = "block"
	ImageWatermarkFailureModePassthrough = "passthrough"
)

const (
	DefaultImageWatermarkText          = "TokenFactory"
	DefaultImageWatermarkOpacity       = 0.65
	DefaultImageWatermarkScalePercent  = 12
	DefaultImageWatermarkMarginPercent = 3
)

// ImageWatermarkPolicy controls output-side watermark processing for image responses.
// It defaults to off to preserve existing response behaviour.
var ImageWatermarkPolicy = ImageWatermarkPolicyOff
var ImageWatermarkType = ImageWatermarkTypeText
var ImageWatermarkText = DefaultImageWatermarkText
var ImageWatermarkLogoURL string
var ImageWatermarkOpacity = DefaultImageWatermarkOpacity
var ImageWatermarkPosition = ImageWatermarkPositionBottomRight
var ImageWatermarkScalePercent = DefaultImageWatermarkScalePercent
var ImageWatermarkMarginPercent = DefaultImageWatermarkMarginPercent
var ImageWatermarkFailureMode = ImageWatermarkFailureModeBlock

var imageWatermarkUserIDs = map[int]struct{}{}
var imageWatermarkUserIDsMutex sync.RWMutex

type ImageWatermarkConfig struct {
	Type          string
	Text          string
	LogoURL       string
	Opacity       float64
	Position      string
	ScalePercent  int
	MarginPercent int
	FailureMode   string
}

func ImageWatermarkUserIDsToString() string {
	imageWatermarkUserIDsMutex.RLock()
	defer imageWatermarkUserIDsMutex.RUnlock()
	ids := make([]string, 0, len(imageWatermarkUserIDs))
	for id := range imageWatermarkUserIDs {
		ids = append(ids, strconv.Itoa(id))
	}
	sort.Slice(ids, func(i, j int) bool {
		left, _ := strconv.Atoi(ids[i])
		right, _ := strconv.Atoi(ids[j])
		return left < right
	})
	return strings.Join(ids, ",")
}

func parseImageWatermarkUserIDs(value string) (map[int]struct{}, error) {
	next := make(map[int]struct{})
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' ' }) {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("图片水印指定用户 ID 必须为正整数")
		}
		next[id] = struct{}{}
	}
	return next, nil
}

func CheckImageWatermarkUserIDs(value string) error {
	_, err := parseImageWatermarkUserIDs(value)
	return err
}

func UpdateImageWatermarkUserIDs(value string) error {
	next, err := parseImageWatermarkUserIDs(value)
	if err != nil {
		return err
	}
	imageWatermarkUserIDsMutex.Lock()
	imageWatermarkUserIDs = next
	imageWatermarkUserIDsMutex.Unlock()
	return nil
}

func IsImageWatermarkForcedForUser(userID int) bool {
	switch strings.ToLower(strings.TrimSpace(ImageWatermarkPolicy)) {
	case ImageWatermarkPolicyAll:
		return true
	case ImageWatermarkPolicyUsers:
		imageWatermarkUserIDsMutex.RLock()
		defer imageWatermarkUserIDsMutex.RUnlock()
		_, ok := imageWatermarkUserIDs[userID]
		return ok
	default:
		return false
	}
}

func SetImageWatermarkPolicy(value string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != ImageWatermarkPolicyAll && value != ImageWatermarkPolicyUsers {
		value = ImageWatermarkPolicyOff
	}
	ImageWatermarkPolicy = value
}

func CheckImageWatermarkPolicy(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ImageWatermarkPolicyOff, ImageWatermarkPolicyAll, ImageWatermarkPolicyUsers:
		return nil
	default:
		return fmt.Errorf("图片水印策略必须为 off、all 或 users")
	}
}

func CheckImageWatermarkEnablement(value string) error {
	if err := CheckImageWatermarkPolicy(value); err != nil {
		return err
	}
	policy := strings.ToLower(strings.TrimSpace(value))
	if policy == ImageWatermarkPolicyOff {
		return nil
	}
	if policy == ImageWatermarkPolicyUsers {
		imageWatermarkUserIDsMutex.RLock()
		hasUsers := len(imageWatermarkUserIDs) > 0
		imageWatermarkUserIDsMutex.RUnlock()
		if !hasUsers {
			return fmt.Errorf("启用指定用户图片水印前，请至少选择一名用户")
		}
	}
	config := GetImageWatermarkConfig()
	switch config.Type {
	case ImageWatermarkTypeText:
		if strings.TrimSpace(config.Text) == "" {
			return fmt.Errorf("启用图片文字水印前，请填写水印文字")
		}
		return CheckImageWatermarkText(config.Text)
	case ImageWatermarkTypeLogo:
		if strings.TrimSpace(config.LogoURL) == "" {
			return fmt.Errorf("启用图片 Logo 水印前，请上传 Logo")
		}
		return CheckImageWatermarkLogoURL(config.LogoURL)
	default:
		return CheckImageWatermarkType(config.Type)
	}
}

func CheckImageWatermarkContentUpdate(key, value string) error {
	var err error
	switch key {
	case "ImageWatermarkType":
		err = CheckImageWatermarkType(value)
	case "ImageWatermarkText":
		err = CheckImageWatermarkText(value)
	case "ImageWatermarkLogoURL":
		err = CheckImageWatermarkLogoURL(value)
	default:
		return nil
	}
	if err != nil || strings.EqualFold(strings.TrimSpace(ImageWatermarkPolicy), ImageWatermarkPolicyOff) {
		return err
	}

	config := GetImageWatermarkConfig()
	switch key {
	case "ImageWatermarkType":
		config.Type = strings.ToLower(strings.TrimSpace(value))
	case "ImageWatermarkText":
		config.Text = strings.TrimSpace(value)
	case "ImageWatermarkLogoURL":
		config.LogoURL = strings.TrimSpace(value)
	}
	if config.Type == ImageWatermarkTypeText && config.Text == "" {
		return fmt.Errorf("已启用图片文字水印，水印文字不能为空")
	}
	if config.Type == ImageWatermarkTypeLogo && config.LogoURL == "" {
		return fmt.Errorf("已启用图片 Logo 水印，Logo 不能为空")
	}
	return nil
}

func CheckImageWatermarkUserIDsUpdate(value string) error {
	users, err := parseImageWatermarkUserIDs(value)
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(ImageWatermarkPolicy), ImageWatermarkPolicyUsers) && len(users) == 0 {
		return fmt.Errorf("已启用指定用户图片水印，请至少保留一名用户")
	}
	return nil
}

func SetImageWatermarkType(value string) {
	ImageWatermarkType = strings.ToLower(strings.TrimSpace(value))
}

func CheckImageWatermarkType(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ImageWatermarkTypeText, ImageWatermarkTypeLogo:
		return nil
	default:
		return fmt.Errorf("图片水印类型必须为 text 或 logo")
	}
}

func CheckImageWatermarkText(value string) error {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 100 {
		return fmt.Errorf("图片文字水印不能超过 100 个字符")
	}
	for _, char := range value {
		if char == '\n' || char == '\r' || char == '\t' {
			return fmt.Errorf("图片文字水印只能使用单行文字")
		}
		if char < 32 || char > 126 {
			return fmt.Errorf("图片文字水印当前仅支持 ASCII 字符；中文请使用 Logo 图片水印")
		}
	}
	return nil
}

func CheckImageWatermarkLogoURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "/") {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("图片 Logo 水印 URL 必须为 http 或 https 地址")
	}
	return nil
}

func SetImageWatermarkOpacity(value float64) { ImageWatermarkOpacity = value }

func CheckImageWatermarkOpacity(value string) error {
	opacity, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || opacity < 0.05 || opacity > 1 {
		return fmt.Errorf("图片水印透明度必须在 0.05 到 1 之间")
	}
	return nil
}

func SetImageWatermarkPosition(value string) {
	ImageWatermarkPosition = normalizeImageWatermarkPosition(value)
}

func CheckImageWatermarkPosition(value string) error {
	switch normalizeImageWatermarkPosition(value) {
	case ImageWatermarkPositionBottomRight, ImageWatermarkPositionBottomLeft, ImageWatermarkPositionTopRight, ImageWatermarkPositionTopLeft, ImageWatermarkPositionCenter:
		return nil
	default:
		return fmt.Errorf("图片水印位置无效")
	}
}

func normalizeImageWatermarkPosition(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "-")
}

func SetImageWatermarkFailureMode(value string) {
	ImageWatermarkFailureMode = strings.ToLower(strings.TrimSpace(value))
}

func CheckImageWatermarkFailureMode(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ImageWatermarkFailureModeBlock, ImageWatermarkFailureModePassthrough:
		return nil
	default:
		return fmt.Errorf("图片水印失败策略必须为 block 或 passthrough")
	}
}

func CheckImageWatermarkScalePercent(value string) error {
	scale, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || scale < 3 || scale > 40 {
		return fmt.Errorf("图片水印尺寸比例必须在 3 到 40 之间")
	}
	return nil
}

func CheckImageWatermarkMarginPercent(value string) error {
	margin, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || margin < 0 || margin > 20 {
		return fmt.Errorf("图片水印边距比例必须在 0 到 20 之间")
	}
	return nil
}

func GetImageWatermarkConfig() ImageWatermarkConfig {
	return ImageWatermarkConfig{
		Type:          strings.ToLower(strings.TrimSpace(ImageWatermarkType)),
		Text:          strings.TrimSpace(ImageWatermarkText),
		LogoURL:       strings.TrimSpace(ImageWatermarkLogoURL),
		Opacity:       ImageWatermarkOpacity,
		Position:      normalizeImageWatermarkPosition(ImageWatermarkPosition),
		ScalePercent:  ImageWatermarkScalePercent,
		MarginPercent: ImageWatermarkMarginPercent,
		FailureMode:   strings.ToLower(strings.TrimSpace(ImageWatermarkFailureMode)),
	}
}
