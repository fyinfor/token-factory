package model

import (
	"math"
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
		t.Fatalf("missing mxxh = %v, %v; want 1, nil", multiplier, err)
	}
	if err := ensureModelConsumptionDistributionMultiplierOption(); err != nil {
		t.Fatalf("initialize mxxh: %v", err)
	}

	for value, want := range map[string]float64{
		"1":        1,
		"10":       10,
		"20":       20,
		"99.99":    99.99,
		"0":        1,
		"-2.5":     1,
		"NaN":      1,
		"Infinity": 1,
		"invalid":  1,
	} {
		if err := DB.Model(&Option{}).
			Where(&Option{Key: modelConsumptionDistributionMultiplierKey}).
			Update("value", value).Error; err != nil {
			t.Fatalf("set mxxh=%q: %v", value, err)
		}
		got, err := GetModelConsumptionDistributionMultiplier()
		if err != nil || math.Abs(got-want) > 1e-9 {
			t.Fatalf("mxxh=%q returned %v, %v; want %v, nil", value, got, err, want)
		}
	}
}
