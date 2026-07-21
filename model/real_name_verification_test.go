package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRealNameVerificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&RealNameVerification{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateRealNameVerificationAllowsMultiplePendingRecords(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })
	DB = setupRealNameVerificationTestDB(t)

	first, err := CreateRealNameVerification(1, "launch-token-1", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("create first verification: %v", err)
	}
	second, err := CreateRealNameVerification(1, "launch-token-2", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("create second verification: %v", err)
	}
	if first.CertifyId != nil || second.CertifyId != nil {
		t.Fatalf("pending records must store a NULL certify ID")
	}
}

func TestNormalizeEmptyRealNameVerificationCertifyIDs(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })
	DB = setupRealNameVerificationTestDB(t)

	record := &RealNameVerification{
		UserId:      1,
		LaunchToken: "legacy-empty-certify-id",
		CertifyId:   stringPointer(""),
		Status:      RealNameVerificationStatusPending,
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := DB.Create(record).Error; err != nil {
		t.Fatalf("create legacy verification: %v", err)
	}
	if err := NormalizeEmptyRealNameVerificationCertifyIDs(); err != nil {
		t.Fatalf("normalize legacy certify ID: %v", err)
	}
	var result RealNameVerification
	if err := DB.First(&result, record.Id).Error; err != nil {
		t.Fatalf("reload verification: %v", err)
	}
	if result.CertifyId != nil {
		t.Fatalf("expected NULL certify ID after normalization, got %q", *result.CertifyId)
	}
}

func stringPointer(value string) *string {
	return &value
}
