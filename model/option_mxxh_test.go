package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModelConsumptionDistributionMultiplierOption(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() {
		DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	DB = testDB
	if err := DB.AutoMigrate(&Option{}); err != nil {
		t.Fatalf("migrate options: %v", err)
	}

	multiplier, err := GetModelConsumptionDistributionMultiplier()
	if err != nil || multiplier != 1 {
		t.Fatalf("missing mxxh = %d, %v; want 1, nil", multiplier, err)
	}
	if err := ensureModelConsumptionDistributionMultiplierOption(); err != nil {
		t.Fatalf("initialize mxxh: %v", err)
	}

	for value, want := range map[string]int{
		"1":       1,
		"10":      10,
		"20":      20,
		"0":       1,
		"invalid": 1,
	} {
		if err := DB.Model(&Option{}).
			Where(&Option{Key: modelConsumptionDistributionMultiplierKey}).
			Update("value", value).Error; err != nil {
			t.Fatalf("set mxxh=%q: %v", value, err)
		}
		got, err := GetModelConsumptionDistributionMultiplier()
		if err != nil || got != want {
			t.Fatalf("mxxh=%q returned %d, %v; want %d, nil", value, got, err, want)
		}
	}
}
