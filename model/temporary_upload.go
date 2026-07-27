package model

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// TemporaryUpload is the persisted upload index. The historical type/table name is
// retained for migration compatibility; ExpiresAt=0 represents a permanent file.
type TemporaryUpload struct {
	ID             int64   `json:"id" gorm:"primaryKey"`
	UserID         int     `json:"user_id" gorm:"not null;index"`
	Purpose        string  `json:"purpose" gorm:"type:varchar(32);not null;default:general;index"`
	OriginalName   string  `json:"original_name" gorm:"type:varchar(255);not null;default:''"`
	MimeType       string  `json:"mime_type" gorm:"type:varchar(128);not null;default:''"`
	Size           int64   `json:"size" gorm:"not null;default:0"`
	URL            string  `json:"url" gorm:"type:varchar(2048);not null;default:''"`
	StorageType    string  `json:"storage_type" gorm:"type:varchar(16);not null;index"`
	ObjectKey      string  `json:"-" gorm:"type:varchar(1024);not null"`
	StorageBase    string  `json:"-" gorm:"type:varchar(1024);not null;default:''"`
	Endpoint       string  `json:"-" gorm:"type:varchar(255);not null;default:''"`
	Bucket         string  `json:"-" gorm:"type:varchar(255);not null;default:''"`
	StorageKeyHash *string `json:"-" gorm:"type:char(64);uniqueIndex"`
	ExpiresAt      int64   `json:"expires_at" gorm:"not null;index"`
	RetryCount     int     `json:"-" gorm:"not null;default:0"`
	LastError      string  `json:"-" gorm:"type:text"`
	CreatedAt      int64   `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt      int64   `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TemporaryUpload) TableName() string { return "temporary_uploads" }

func CreateUploadObject(upload *TemporaryUpload) error {
	return DB.Create(upload).Error
}

func UpsertUploadObject(upload *TemporaryUpload) error {
	if upload.StorageKeyHash == nil || strings.TrimSpace(*upload.StorageKeyHash) == "" {
		return CreateUploadObject(upload)
	}

	var existing TemporaryUpload
	err := DB.Where("storage_key_hash = ?", *upload.StorageKeyHash).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = DB.Where(
			"storage_type = ? AND object_key = ? AND storage_base = ? AND endpoint = ? AND bucket = ?",
			upload.StorageType,
			upload.ObjectKey,
			upload.StorageBase,
			upload.Endpoint,
			upload.Bucket,
		).First(&existing).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CreateUploadObject(upload)
	}
	if err != nil {
		return err
	}
	return DB.Model(&existing).Updates(uploadObjectUpdateMap(upload)).Error
}

func uploadObjectUpdateMap(upload *TemporaryUpload) map[string]any {
	return map[string]any{
		"user_id":          upload.UserID,
		"purpose":          upload.Purpose,
		"original_name":    upload.OriginalName,
		"mime_type":        upload.MimeType,
		"size":             upload.Size,
		"url":              upload.URL,
		"storage_type":     upload.StorageType,
		"object_key":       upload.ObjectKey,
		"storage_base":     upload.StorageBase,
		"endpoint":         upload.Endpoint,
		"bucket":           upload.Bucket,
		"storage_key_hash": upload.StorageKeyHash,
	}
}

type UploadObjectFilter struct {
	Keyword     string
	Purpose     string
	Lifecycle   string
	StorageType string
	Page        int
	PageSize    int
}

func ListUploadObjects(filter UploadObjectFilter) ([]TemporaryUpload, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := DB.Model(&TemporaryUpload{})
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("original_name LIKE ? OR object_key LIKE ? OR url LIKE ?", like, like, like)
	}
	if filter.Purpose != "" {
		query = query.Where("purpose = ?", filter.Purpose)
	}
	switch filter.Lifecycle {
	case "permanent":
		query = query.Where("expires_at = ?", 0)
	case "temporary":
		query = query.Where("expires_at > ?", 0)
	}
	if filter.StorageType != "" {
		query = query.Where("storage_type = ?", filter.StorageType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var uploads []TemporaryUpload
	err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&uploads).Error
	return uploads, total, err
}

func GetUploadObjectByID(id int64) (*TemporaryUpload, error) {
	var upload TemporaryUpload
	if err := DB.First(&upload, id).Error; err != nil {
		return nil, err
	}
	return &upload, nil
}

func GetUploadObjectsByIDs(ids []int64) ([]TemporaryUpload, error) {
	if len(ids) == 0 {
		return []TemporaryUpload{}, nil
	}
	var uploads []TemporaryUpload
	err := DB.Where("id IN ?", ids).Find(&uploads).Error
	return uploads, err
}

func UpdateUploadObjectExpiration(id int64, expiresAt int64) error {
	return DB.Model(&TemporaryUpload{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"expires_at":  expiresAt,
			"retry_count": 0,
			"last_error":  "",
		}).Error
}

func ListExpiredUploadObjects(now int64, limit int) ([]TemporaryUpload, error) {
	if limit <= 0 {
		limit = 100
	}
	var uploads []TemporaryUpload
	err := DB.Where("expires_at > ? AND expires_at <= ?", 0, now).
		Order("retry_count ASC").
		Order("expires_at ASC").
		Limit(limit).
		Find(&uploads).Error
	return uploads, err
}

func DeleteUploadObject(id int64) error {
	return DB.Delete(&TemporaryUpload{}, id).Error
}

func MarkUploadObjectCleanupFailure(id int64, message string) error {
	message = strings.TrimSpace(message)
	messageRunes := []rune(message)
	if len(messageRunes) > 2000 {
		message = string(messageRunes[:2000])
	}
	return DB.Model(&TemporaryUpload{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"retry_count": gorm.Expr("retry_count + ?", 1),
			"last_error":  message,
		}).Error
}
