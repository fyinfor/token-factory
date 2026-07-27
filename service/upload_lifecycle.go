package service

import (
	"fmt"
	"mime"
	"mime/multipart"
	"path"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const PlaygroundUploadRetention = 24 * time.Hour

const maxPlaygroundUploadRetentionHours = 24 * 365

func playgroundUploadRetention(hours int) time.Duration {
	if hours <= 0 || hours > maxPlaygroundUploadRetentionHours {
		return PlaygroundUploadRetention
	}
	return time.Duration(hours) * time.Hour
}

const (
	UploadPurposeGeneral     = "general"
	UploadPurposeHomepage    = "homepage"
	UploadPurposeIcons       = "icons"
	UploadPurposeSupplier    = "supplier"
	UploadPurposeDistributor = "distributor"
	UploadPurposeChannel     = "channel"
	UploadPurposePlayground  = "playground"
	UploadPurposeLegacy      = "legacy"
)

type uploadPurposeSpec struct {
	directory string
	temporary bool
}

var uploadPurposeSpecs = map[string]uploadPurposeSpec{
	UploadPurposeGeneral:     {directory: "permanent/general"},
	UploadPurposeHomepage:    {directory: "permanent/homepage"},
	UploadPurposeIcons:       {directory: "permanent/icons"},
	UploadPurposeSupplier:    {directory: "permanent/suppliers"},
	UploadPurposeDistributor: {directory: "permanent/distributors"},
	UploadPurposeChannel:     {directory: "permanent/channels"},
	UploadPurposePlayground:  {directory: "temporary/playground", temporary: true},
}

type UploadResult struct {
	Purpose      string
	OriginalName string
	MimeType     string
	Size         int64
	URL          string
	StorageType  string
	ObjectKey    string
	StorageBase  string
	Endpoint     string
	Bucket       string
	ExpiresAt    int64
}

func resolveUploadPurpose(raw string) (string, uploadPurposeSpec, error) {
	purpose := strings.ToLower(strings.TrimSpace(raw))
	if purpose == "" {
		purpose = UploadPurposeGeneral
	}
	aliases := map[string]string{
		"icon":         UploadPurposeIcons,
		"supplier":     UploadPurposeSupplier,
		"suppliers":    UploadPurposeSupplier,
		"distributor":  UploadPurposeDistributor,
		"distributors": UploadPurposeDistributor,
		"channel":      UploadPurposeChannel,
		"channels":     UploadPurposeChannel,
	}
	if canonical, ok := aliases[purpose]; ok {
		purpose = canonical
	}
	spec, ok := uploadPurposeSpecs[purpose]
	if !ok {
		return "", uploadPurposeSpec{}, fmt.Errorf("不支持的上传用途")
	}
	return purpose, spec, nil
}

// UploadMultipartFileByPurpose stores a file in a fixed purpose namespace.
// Only Playground uploads receive an expiration record; all other purposes are permanent.
func UploadMultipartFileByPurpose(file *multipart.FileHeader, userID int, rawPurpose string) (*UploadResult, error) {
	purposeName, purpose, err := resolveUploadPurpose(rawPurpose)
	if err != nil {
		return nil, err
	}
	cfg := operation_setting.GetOssSetting()
	if !cfg.Enabled {
		return nil, fmt.Errorf("文件上传未启用，请先在运营设置中启用")
	}

	var result *UploadResult
	if cfg.StorageType == operation_setting.StorageTypeLocal {
		prefix, normalizeErr := NormalizeLocalUploadPrefix(joinUploadPrefix(cfg.LocalObjectKeyPrefix, purpose.directory))
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		result, err = localUploadMultipartFileWithPrefixResult(file, userID, prefix)
	} else {
		if !operation_setting.IsOssUploadReady() {
			return nil, ErrOssNotConfigured
		}
		prefix := joinUploadPrefix(cfg.ObjectKeyPrefix, purpose.directory)
		result, err = ossUploadMultipartFileWithPrefixResult(file, userID, prefix)
	}
	if err != nil {
		return nil, err
	}
	result.Purpose = purposeName
	result.OriginalName = uploadOriginalName(file.Filename)
	result.MimeType = uploadMimeType(file)
	result.Size = file.Size
	if purpose.temporary {
		result.ExpiresAt = time.Now().Add(playgroundUploadRetention(cfg.PlaygroundRetentionHours)).Unix()
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
		ExpiresAt:      result.ExpiresAt,
	}
	if err := model.UpsertUploadObject(record); err != nil {
		if cleanupErr := DeleteStoredUpload(result); cleanupErr != nil {
			return nil, fmt.Errorf("记录上传文件失败: %v；回滚文件失败: %w", err, cleanupErr)
		}
		return nil, fmt.Errorf("记录上传文件失败: %w", err)
	}
	return result, nil
}

func uploadOriginalName(filename string) string {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/"))
	if name == "." || name == "/" {
		return ""
	}
	runes := []rune(name)
	if len(runes) > 255 {
		name = string(runes[len(runes)-255:])
	}
	return name
}

func uploadMimeType(file *multipart.FileHeader) string {
	mimeType := strings.TrimSpace(file.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = mime.TypeByExtension(path.Ext(file.Filename))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if len(mimeType) > 128 {
		mimeType = mimeType[:128]
	}
	return mimeType
}
