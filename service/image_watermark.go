package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	stdraw "image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/webp"
)

// ApplyImageWatermarkToResponse applies the configured output watermark to a
// standard OpenAI image response. A matching policy with an unprocessable
// output returns an error so callers can block rather than leak an unmarked image.
func ApplyImageWatermarkToResponse(info *relaycommon.RelayInfo, responseBody []byte) ([]byte, error) {
	if info == nil || !setting.IsImageWatermarkForcedForUser(info.UserId) {
		return responseBody, nil
	}
	if len(bytes.TrimSpace(responseBody)) == 0 {
		return nil, fmt.Errorf("图片水印处理失败: 上游响应为空")
	}

	var response dto.ImageResponse
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("图片水印处理失败: 无法解析标准图片响应: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("图片水印处理失败: 上游未返回图片")
	}

	config := setting.GetImageWatermarkConfig()
	if err := validateImageWatermarkConfig(config); err != nil {
		return nil, err
	}

	var logo image.Image
	var logoErr error
	if config.Type == setting.ImageWatermarkTypeLogo {
		logo, logoErr = loadImageWatermarkLogo(config.LogoURL)
		if logoErr != nil {
			return nil, fmt.Errorf("图片水印处理失败: 加载 Logo: %w", logoErr)
		}
	}

	storedUploads := make([]*model.TemporaryUpload, 0, len(response.Data))
	for index := range response.Data {
		upload, err := applyImageWatermarkToData(&response.Data[index], info.UserId, config, logo)
		if err != nil {
			rollbackErr := rollbackWatermarkedImages(storedUploads)
			if rollbackErr != nil {
				return nil, fmt.Errorf("图片水印处理失败: 第 %d 张图片: %v；回滚已托管图片失败: %w", index+1, err, rollbackErr)
			}
			return nil, fmt.Errorf("图片水印处理失败: 第 %d 张图片: %w", index+1, err)
		}
		if upload != nil {
			storedUploads = append(storedUploads, upload)
		}
	}
	watermarkedData, err := common.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("图片水印处理失败: 序列化图片结果: %w", err)
	}
	var rawResponse map[string]json.RawMessage
	if err := common.Unmarshal(responseBody, &rawResponse); err != nil {
		return nil, fmt.Errorf("图片水印处理失败: 保留上游响应字段: %w", err)
	}
	rawResponse["data"] = watermarkedData
	return common.Marshal(rawResponse)
}

func validateImageWatermarkConfig(config setting.ImageWatermarkConfig) error {
	if config.Type == setting.ImageWatermarkTypeLogo {
		if err := setting.CheckImageWatermarkLogoURL(config.LogoURL); err != nil || strings.TrimSpace(config.LogoURL) == "" {
			return fmt.Errorf("图片水印处理失败: Logo 模式需要有效的 Logo URL")
		}
	} else if config.Type == setting.ImageWatermarkTypeText {
		if strings.TrimSpace(config.Text) == "" {
			return fmt.Errorf("图片水印处理失败: 文字模式需要非空水印文字")
		}
		if err := setting.CheckImageWatermarkText(config.Text); err != nil {
			return fmt.Errorf("图片水印处理失败: %w", err)
		}
	} else {
		return fmt.Errorf("图片水印处理失败: 不支持的水印类型")
	}
	if config.Opacity < 0.05 || config.Opacity > 1 || config.ScalePercent < 3 || config.ScalePercent > 40 || config.MarginPercent < 0 || config.MarginPercent > 20 {
		return fmt.Errorf("图片水印处理失败: 水印配置无效")
	}
	return setting.CheckImageWatermarkPosition(config.Position)
}

func applyImageWatermarkToData(data *dto.ImageData, userID int, config setting.ImageWatermarkConfig, logo image.Image) (*model.TemporaryUpload, error) {
	if data == nil {
		return nil, fmt.Errorf("图片数据为空")
	}
	if strings.TrimSpace(data.B64Json) != "" {
		raw, err := decodeBase64Image(data.B64Json)
		if err != nil {
			return nil, err
		}
		watermarked, err := renderImageWatermark(raw, config, logo)
		if err != nil {
			return nil, err
		}
		data.B64Json = base64.StdEncoding.EncodeToString(watermarked)
		data.Url = ""
		return nil, nil
	}
	if strings.TrimSpace(data.Url) == "" {
		return nil, fmt.Errorf("图片 URL 与 b64_json 均为空")
	}
	mimeType, encoded, err := GetImageFromUrl(strings.TrimSpace(data.Url))
	if err != nil {
		return nil, err
	}
	raw, err := decodeBase64Image(encoded)
	if err != nil {
		return nil, err
	}
	watermarked, err := renderImageWatermark(raw, config, logo)
	if err != nil {
		return nil, err
	}
	_ = mimeType
	upload, err := storeWatermarkedImage(watermarked, userID)
	if err != nil {
		return nil, err
	}
	data.Url = upload.URL
	data.B64Json = ""
	return upload, nil
}

func rollbackWatermarkedImages(uploads []*model.TemporaryUpload) error {
	var rollbackErr error
	for index := len(uploads) - 1; index >= 0; index-- {
		if err := DeleteIndexedUploadObject(uploads[index]); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}

func decodeBase64Image(value string) (image.Image, error) {
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[comma+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("解码 b64_json: %w", err)
	}
	return decodeImage(raw)
}

func loadImageWatermarkLogo(logoURL string) (image.Image, error) {
	if localPath, ok := ResolveLocalUploadFilePath(logoURL); ok {
		raw, err := os.ReadFile(localPath)
		if err != nil {
			return nil, err
		}
		return decodeImage(raw)
	}
	_, encoded, err := GetImageFromUrl(logoURL)
	if err != nil {
		return nil, err
	}
	return decodeBase64Image(encoded)
}

func decodeImage(raw []byte) (image.Image, error) {
	config, format, err := getImageConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("读取图片尺寸: %w", err)
	}
	const maxWatermarkPixels = 25_000_000
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxWatermarkPixels {
		return nil, fmt.Errorf("图片像素超过水印处理上限")
	}
	if format == "gif" {
		return nil, fmt.Errorf("暂不支持 GIF 图片水印，请使用 PNG、JPEG 或 WebP")
	}
	if decoded, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
		return decoded, nil
	}
	decoded, err := webp.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("解码图片: %w", err)
	}
	return decoded, nil
}

func renderImageWatermark(source image.Image, config setting.ImageWatermarkConfig, logo image.Image) ([]byte, error) {
	bounds := source.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil, fmt.Errorf("图片尺寸无效")
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	stdraw.Draw(canvas, canvas.Bounds(), source, bounds.Min, stdraw.Src)

	if config.Type == setting.ImageWatermarkTypeLogo {
		drawLogoWatermark(canvas, logo, config)
	} else {
		if err := drawTextWatermark(canvas, config); err != nil {
			return nil, err
		}
	}

	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("编码水印图片: %w", err)
	}
	return output.Bytes(), nil
}

func drawTextWatermark(canvas *image.NRGBA, config setting.ImageWatermarkConfig) error {
	shortSide := min(canvas.Bounds().Dx(), canvas.Bounds().Dy())
	fontSize := float64(shortSide*config.ScalePercent) / 100
	if fontSize < 10 {
		fontSize = 10
	}
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer face.Close()

	text := config.Text
	metrics := face.Metrics()
	textWidth := font.MeasureString(face, text).Ceil()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()
	x, y := imageWatermarkOrigin(canvas.Bounds(), textWidth, textHeight, metrics.Ascent.Ceil(), config)
	mask := image.NewAlpha(canvas.Bounds())
	drawer := &font.Drawer{Dst: mask, Src: image.NewUniform(color.Alpha{A: 255}), Face: face, Dot: fixedPoint(x, y)}
	drawer.DrawString(text)
	alpha := uint8(math.Round(config.Opacity * 255))
	stdraw.DrawMask(
		canvas,
		canvas.Bounds(),
		image.NewUniform(color.NRGBA{R: 255, G: 255, B: 255, A: alpha}),
		image.Point{},
		mask,
		mask.Bounds().Min,
		stdraw.Over,
	)
	return nil
}

func fixedPoint(x, y int) fixed.Point26_6 {
	return fixed.P(x, y)
}

func drawLogoWatermark(canvas *image.NRGBA, logo image.Image, config setting.ImageWatermarkConfig) {
	shortSide := min(canvas.Bounds().Dx(), canvas.Bounds().Dy())
	targetWidth := max(1, shortSide*config.ScalePercent/100)
	logoBounds := logo.Bounds()
	targetHeight := max(1, int(math.Round(float64(targetWidth*logoBounds.Dy())/float64(max(1, logoBounds.Dx())))))
	positioned := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	xdraw.CatmullRom.Scale(positioned, positioned.Bounds(), logo, logoBounds, stdraw.Over, nil)
	x, y := imageWatermarkOrigin(canvas.Bounds(), targetWidth, targetHeight, 0, config)
	alpha := uint8(math.Round(config.Opacity * 255))
	stdraw.DrawMask(
		canvas,
		image.Rect(x, y, x+targetWidth, y+targetHeight),
		positioned,
		positioned.Bounds().Min,
		image.NewUniform(color.Alpha{A: alpha}),
		image.Point{},
		stdraw.Over,
	)
}

func imageWatermarkOrigin(canvas image.Rectangle, width, height, baseline int, config setting.ImageWatermarkConfig) (int, int) {
	margin := min(canvas.Dx(), canvas.Dy()) * config.MarginPercent / 100
	x := margin
	y := margin
	if config.Position == setting.ImageWatermarkPositionCenter {
		x = (canvas.Dx() - width) / 2
		y = (canvas.Dy() - height) / 2
	}
	if config.Position == setting.ImageWatermarkPositionBottomRight || config.Position == setting.ImageWatermarkPositionTopRight {
		x = canvas.Dx() - width - margin
	}
	if config.Position == setting.ImageWatermarkPositionBottomRight || config.Position == setting.ImageWatermarkPositionBottomLeft {
		y = canvas.Dy() - height - margin
	}
	x = max(0, x)
	y = max(0, y)
	if baseline > 0 {
		y += baseline
	}
	return x, y
}

// StoreWatermarkedImage persists rewritten URL outputs via the existing local/OSS upload settings.
func StoreWatermarkedImage(data []byte, userID int) (string, error) {
	record, err := storeWatermarkedImage(data, userID)
	if err != nil {
		return "", err
	}
	return record.URL, nil
}

func storeWatermarkedImage(data []byte, userID int) (*model.TemporaryUpload, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("水印图片为空")
	}
	if !operation_setting.IsUploadReady() {
		return nil, fmt.Errorf("水印图片托管不可用，请先启用本地存储或 OSS 上传")
	}
	cfg := operation_setting.GetOssSetting()
	maxFileSizeMB := cfg.MaxFileSizeMB
	if cfg.StorageType == operation_setting.StorageTypeLocal && cfg.LocalMaxFileSizeMB > 0 {
		maxFileSizeMB = cfg.LocalMaxFileSizeMB
	}
	if cfg.StorageType == operation_setting.StorageTypeOSS && cfg.OssMaxFileSizeMB > 0 {
		maxFileSizeMB = cfg.OssMaxFileSizeMB
	}
	if maxFileSizeMB <= 0 {
		maxFileSizeMB = 20
	}
	if int64(len(data)) > int64(maxFileSizeMB)*1024*1024 {
		return nil, fmt.Errorf("水印图片超过存储大小限制（最大 %d MB）", maxFileSizeMB)
	}
	purpose := uploadPurposeSpecs[UploadPurposeWatermark]
	result := &UploadResult{
		Purpose:      UploadPurposeWatermark,
		OriginalName: "watermark.png",
		MimeType:     "image/png",
		Size:         int64(len(data)),
	}
	prefix := joinUploadPrefix(cfg.ObjectKeyPrefix, purpose.directory)
	if cfg.StorageType == operation_setting.StorageTypeLocal {
		localPrefix, err := NormalizeLocalUploadPrefix(joinUploadPrefix(cfg.LocalObjectKeyPrefix, purpose.directory))
		if err != nil {
			return nil, err
		}
		relPath, err := BuildLocalUploadObjectPath(localPrefix, ".png")
		if err != nil {
			return nil, err
		}
		fullPath := filepath.Join(LocalUploadBaseDir(cfg.LocalStoragePath), filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return nil, err
		}
		result.URL = localObjectURL(cfg.LocalURLPrefix, filepath.ToSlash(filepath.Join(LocalUploadFolder, relPath)))
		result.StorageType = operation_setting.StorageTypeLocal
		result.ObjectKey = relPath
		result.StorageBase = cfg.LocalStoragePath
	} else {
		if !operation_setting.IsOssUploadReady() {
			return nil, ErrOssNotConfigured
		}
		objectKey := BuildUploadObjectPath(prefix, ".png")
		if err := ossPutObject(cfg, objectKey, "image/png", data); err != nil {
			return nil, err
		}
		result.URL = publicObjectURL(cfg, objectKey)
		result.StorageType = operation_setting.StorageTypeOSS
		result.ObjectKey = objectKey
		result.Endpoint = cfg.Endpoint
		result.Bucket = cfg.Bucket
	}
	storageKeyHash := uploadStorageKeyHash(
		result.StorageType,
		result.ObjectKey,
		result.StorageBase,
		result.Endpoint,
		result.Bucket,
	)
	record := &model.TemporaryUpload{
		UserID:         userID,
		Purpose:        result.Purpose,
		OriginalName:   result.OriginalName,
		MimeType:       result.MimeType,
		Size:           result.Size,
		URL:            result.URL,
		StorageType:    result.StorageType,
		ObjectKey:      result.ObjectKey,
		StorageBase:    result.StorageBase,
		Endpoint:       result.Endpoint,
		Bucket:         result.Bucket,
		StorageKeyHash: &storageKeyHash,
	}
	if err := model.UpsertUploadObject(record); err != nil {
		if cleanupErr := DeleteStoredUpload(result); cleanupErr != nil {
			return nil, fmt.Errorf("记录水印图片失败: %v；回滚文件失败: %w", err, cleanupErr)
		}
		return nil, fmt.Errorf("记录水印图片失败: %w", err)
	}
	return record, nil
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
