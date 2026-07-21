package model

import (
	"errors"
	"strings"
	"time"
)

const (
	RealNameVerificationStatusPending = "pending"
	RealNameVerificationStatusPassed  = "passed"
	RealNameVerificationStatusFailed  = "failed"
	RealNameVerificationStatusExpired = "expired"
)

type RealNameVerification struct {
	Id           int        `json:"id"`
	UserId       int        `json:"user_id" gorm:"column:user_id;index;not null"`
	LaunchToken  string     `json:"-" gorm:"column:launch_token;type:varchar(64);uniqueIndex;not null"`
	CertifyId    *string    `json:"-" gorm:"column:certify_id;type:varchar(128);uniqueIndex"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(16);index;not null"`
	ProviderCode string     `json:"-" gorm:"column:provider_code;type:varchar(64)"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"column:expires_at;index;not null"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty" gorm:"column:verified_at"`
	RewardedAt   *time.Time `json:"rewarded_at,omitempty" gorm:"column:rewarded_at"`
	RewardQuota  int        `json:"reward_quota" gorm:"column:reward_quota;not null;default:0"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (RealNameVerification) TableName() string { return "real_name_verifications" }

func CreateRealNameVerification(userId int, launchToken string, expiresAt time.Time) (*RealNameVerification, error) {
	if userId <= 0 || strings.TrimSpace(launchToken) == "" {
		return nil, errors.New("invalid real-name verification request")
	}
	record := &RealNameVerification{UserId: userId, LaunchToken: launchToken, Status: RealNameVerificationStatusPending, ExpiresAt: expiresAt}
	if err := DB.Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func GetRealNameVerificationByLaunchToken(launchToken string) (*RealNameVerification, error) {
	var record RealNameVerification
	if err := DB.Where("launch_token = ?", launchToken).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func GetLatestRealNameVerificationByUserId(userId int) (*RealNameVerification, error) {
	var record RealNameVerification
	if err := DB.Where("user_id = ?", userId).Order("id DESC").First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func GetLatestActiveRealNameVerificationByUserId(userId int) (*RealNameVerification, error) {
	var record RealNameVerification
	if err := DB.
		Where(
			"user_id = ? AND (status = ? OR (status = ? AND certify_id IS NOT NULL))",
			userId,
			RealNameVerificationStatusPassed,
			RealNameVerificationStatusPending,
		).
		Order("id DESC").
		First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func RevokeRealNameVerification(userId int) error {
	return DB.Model(&RealNameVerification{}).Where("user_id = ? AND status = ?", userId, RealNameVerificationStatusPassed).Update("status", RealNameVerificationStatusExpired).Error
}

func HasReceivedRealNameVerificationReward(userId int) (bool, error) {
	var count int64
	err := DB.Model(&RealNameVerification{}).Where("user_id = ? AND rewarded_at IS NOT NULL", userId).Count(&count).Error
	return count > 0, err
}

func HasPassedRealNameVerification(userId int) (bool, error) {
	var count int64
	err := DB.Model(&RealNameVerification{}).Where("user_id = ? AND status = ?", userId, RealNameVerificationStatusPassed).Count(&count).Error
	return count > 0, err
}

func FillRealNameVerificationForUsers(users []*User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]int, 0, len(users))
	byID := make(map[int]*User, len(users))
	for _, user := range users {
		if user != nil {
			ids = append(ids, user.Id)
			byID[user.Id] = user
		}
	}
	var records []RealNameVerification
	if err := DB.Where("user_id IN ? AND status = ?", ids, RealNameVerificationStatusPassed).Order("verified_at DESC, id DESC").Find(&records).Error; err != nil {
		return err
	}
	for _, record := range records {
		user := byID[record.UserId]
		if user != nil && !user.RealNameVerified {
			user.RealNameVerified = true
			user.RealNameVerifiedAt = record.VerifiedAt
		}
	}
	return nil
}

func NormalizeEmptyRealNameVerificationCertifyIDs() error {
	return DB.Model(&RealNameVerification{}).
		Where("certify_id = ?", "").
		Update("certify_id", nil).Error
}
