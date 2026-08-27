package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestApplyImageWatermarkToBase64Response(t *testing.T) {
	oldPolicy := setting.ImageWatermarkPolicy
	oldType := setting.ImageWatermarkType
	oldText := setting.ImageWatermarkText
	oldLogo := setting.ImageWatermarkLogoURL
	t.Cleanup(func() {
		setting.ImageWatermarkPolicy = oldPolicy
		setting.ImageWatermarkType = oldType
		setting.ImageWatermarkText = oldText
		setting.ImageWatermarkLogoURL = oldLogo
	})
	setting.SetImageWatermarkPolicy(setting.ImageWatermarkPolicyAll)
	setting.SetImageWatermarkType(setting.ImageWatermarkTypeText)
	setting.ImageWatermarkText = "TF"
	setting.ImageWatermarkLogoURL = ""

	source := image.NewNRGBA(image.Rect(0, 0, 160, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 160; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 20, G: 30, B: 40, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	body, err := common.Marshal(dto.ImageResponse{Data: []dto.ImageData{{
		B64Json: base64.StdEncoding.EncodeToString(encoded.Bytes()),
	}}, Metadata: json.RawMessage(`{"provider":"test"}`)})
	if err != nil {
		t.Fatal(err)
	}

	out, err := ApplyImageWatermarkToResponse(&relaycommon.RelayInfo{UserId: 7}, body)
	if err != nil {
		t.Fatal(err)
	}
	var response dto.ImageResponse
	if err := common.Unmarshal(out, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || response.Data[0].B64Json == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
	watermarked, err := base64.StdEncoding.DecodeString(response.Data[0].B64Json)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(watermarked))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 160 || decoded.Bounds().Dy() != 100 {
		t.Fatalf("size changed to %v", decoded.Bounds())
	}
	if bytes.Equal(encoded.Bytes(), watermarked) {
		t.Fatal("watermarked image should differ from the source")
	}
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(out, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["metadata"]) != `{"provider":"test"}` {
		t.Fatalf("metadata was not preserved: %s", raw["metadata"])
	}
}

func TestApplyImageWatermarkHonorsTextTypeWhenLogoConfigured(t *testing.T) {
	oldPolicy := setting.ImageWatermarkPolicy
	oldType := setting.ImageWatermarkType
	oldText := setting.ImageWatermarkText
	oldLogo := setting.ImageWatermarkLogoURL
	t.Cleanup(func() {
		setting.ImageWatermarkPolicy = oldPolicy
		setting.ImageWatermarkType = oldType
		setting.ImageWatermarkText = oldText
		setting.ImageWatermarkLogoURL = oldLogo
	})
	setting.SetImageWatermarkPolicy(setting.ImageWatermarkPolicyAll)
	setting.SetImageWatermarkType(setting.ImageWatermarkTypeText)
	setting.ImageWatermarkText = "TF"
	setting.ImageWatermarkLogoURL = "https://invalid.example/logo.png"

	source := image.NewNRGBA(image.Rect(0, 0, 160, 100))
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 20, G: 30, B: 40, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	body, err := common.Marshal(dto.ImageResponse{Data: []dto.ImageData{{
		B64Json: base64.StdEncoding.EncodeToString(encoded.Bytes()),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyImageWatermarkToResponse(&relaycommon.RelayInfo{UserId: 7}, body); err != nil {
		t.Fatalf("text mode unexpectedly loaded configured logo: %v", err)
	}
}

func TestValidateImageWatermarkConfigRejectsEmptyText(t *testing.T) {
	config := setting.GetImageWatermarkConfig()
	config.Type = setting.ImageWatermarkTypeText
	config.Text = "   "
	if err := validateImageWatermarkConfig(config); err == nil {
		t.Fatal("text watermark with empty content should be rejected")
	}
}

func TestStoreWatermarkedImageLocal(t *testing.T) {
	originalDB := model.DB
	originalSetting := *operation_setting.GetOssSetting()
	t.Cleanup(func() {
		model.DB = originalDB
		*operation_setting.GetOssSetting() = originalSetting
	})

	db, err := gorm.Open(sqlite.Open("file:image_watermark_store?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.TemporaryUpload{}); err != nil {
		t.Fatal(err)
	}
	model.DB = db
	root := t.TempDir()
	*operation_setting.GetOssSetting() = operation_setting.OssSetting{
		Enabled:              true,
		StorageType:          operation_setting.StorageTypeLocal,
		LocalStoragePath:     root,
		LocalObjectKeyPrefix: "tenant",
		LocalMaxFileSizeMB:   20,
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatal(err)
	}
	storedURL, err := StoreWatermarkedImage(encoded.Bytes(), 42)
	if err != nil {
		t.Fatal(err)
	}
	var record model.TemporaryUpload
	if err := db.Where("url = ?", storedURL).First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.UserID != 42 || record.Purpose != UploadPurposeWatermark || record.ExpiresAt != 0 {
		t.Fatalf("unexpected upload record: %+v", record)
	}
	storedPath := filepath.Join(root, LocalUploadFolder, filepath.FromSlash(record.ObjectKey))
	if _, err := os.Stat(storedPath); err != nil {
		t.Fatalf("stored image not found: %v", err)
	}
}

func TestRollbackWatermarkedImages(t *testing.T) {
	originalDB := model.DB
	originalSetting := *operation_setting.GetOssSetting()
	t.Cleanup(func() {
		model.DB = originalDB
		*operation_setting.GetOssSetting() = originalSetting
	})

	db, err := gorm.Open(sqlite.Open("file:image_watermark_rollback?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.TemporaryUpload{}); err != nil {
		t.Fatal(err)
	}
	model.DB = db
	root := t.TempDir()
	*operation_setting.GetOssSetting() = operation_setting.OssSetting{
		Enabled:              true,
		StorageType:          operation_setting.StorageTypeLocal,
		LocalStoragePath:     root,
		LocalObjectKeyPrefix: "tenant",
		LocalMaxFileSizeMB:   20,
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatal(err)
	}
	record, err := storeWatermarkedImage(encoded.Bytes(), 42)
	if err != nil {
		t.Fatal(err)
	}
	storedPath := filepath.Join(root, LocalUploadFolder, filepath.FromSlash(record.ObjectKey))
	if err := rollbackWatermarkedImages([]*model.TemporaryUpload{record}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storedPath); !os.IsNotExist(err) {
		t.Fatalf("rolled back image still exists: %v", err)
	}
	var count int64
	if err := db.Model(&model.TemporaryUpload{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled back upload index still exists: %d", count)
	}
}
