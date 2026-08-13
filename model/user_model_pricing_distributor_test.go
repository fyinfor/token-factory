package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserPricingDistributorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	dsn := fmt.Sprintf(
		"file:user_pricing_distributor_%d?mode=memory&cache=shared",
		time.Now().UnixNano(),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	DB = db
	t.Cleanup(func() {
		InvalidateUserModelPricingCache()
		DB = previousDB
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&User{}, &UserModelPricingOverride{}, &UserModelPricingChannel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	InvalidateUserModelPricingCache()
	return db
}

func TestUserPricingBillingAppliesSkipsDistributor(t *testing.T) {
	db := setupUserPricingDistributorTestDB(t)

	agent := User{
		Username:      "pricing-agent",
		Password:      "password",
		AffCode:       "pricing-agent-code",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
	}
	ordinary := User{
		Username: "pricing-ordinary",
		Password: "password",
		AffCode:  "pricing-ordinary-code",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Create(&ordinary).Error; err != nil {
		t.Fatalf("create ordinary: %v", err)
	}

	if UserPricingBillingApplies(agent.Id) {
		t.Fatalf("agent should not apply user-pricing billing override")
	}
	if !UserPricingBillingApplies(ordinary.Id) {
		t.Fatalf("ordinary user should apply user-pricing billing override")
	}
	if UserPricingBillingApplies(0) {
		t.Fatalf("userId=0 should not apply billing override")
	}
}

func TestGetEnabledUserModelPricingBillingOverrideSkipsDistributor(t *testing.T) {
	db := setupUserPricingDistributorTestDB(t)

	agent := User{
		Username:      "billing-ov-agent",
		Password:      "password",
		AffCode:       "billing-ov-agent-code",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
	}
	ordinary := User{
		Username: "billing-ov-ordinary",
		Password: "password",
		AffCode:  "billing-ov-ordinary-code",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Create(&ordinary).Error; err != nil {
		t.Fatalf("create ordinary: %v", err)
	}

	for _, uid := range []int{agent.Id, ordinary.Id} {
		ov := UserModelPricingOverride{
			UserId:               uid,
			ModelName:            "gpt-test",
			Mode:                 UserPricingModePriceCap,
			PriceDiscountPercent: 50,
			OperatingCostPercent: 0,
			MarkupDiscountRate:   5,
			Enabled:              true,
			CreatedTime:          time.Now().Unix(),
			UpdatedTime:          time.Now().Unix(),
		}
		if err := db.Create(&ov).Error; err != nil {
			t.Fatalf("create override for %d: %v", uid, err)
		}
	}
	InvalidateUserModelPricingCache()

	if _, ok := GetEnabledUserModelPricingOverride(agent.Id, "gpt-test"); !ok {
		t.Fatalf("agent should still have routing override config")
	}
	if _, ok := GetEnabledUserModelPricingBillingOverride(agent.Id, "gpt-test"); ok {
		t.Fatalf("agent must not get billing override")
	}
	if ov, ok := GetEnabledUserModelPricingBillingOverride(ordinary.Id, "gpt-test"); !ok {
		t.Fatalf("ordinary user should get billing override")
	} else if ov.TotalPercent() != 55 {
		t.Fatalf("ordinary total percent = %v, want 55", ov.TotalPercent())
	}
}

func TestApplyUserPricingOverrideToPricingAPI_DistributorKeepsChannelCost(t *testing.T) {
	db := setupUserPricingDistributorTestDB(t)

	prevModelRatio := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelRatioByJSONString(prevModelRatio)
	})
	if err := ratio_setting.UpdateModelRatioByJSONString(`{"gpt-test": 10}`); err != nil {
		t.Fatalf("set model ratio: %v", err)
	}

	agent := User{
		Username:      "display-agent",
		Password:      "password",
		AffCode:       "display-agent-code",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
	}
	ordinary := User{
		Username: "display-ordinary",
		Password: "password",
		AffCode:  "display-ordinary-code",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Create(&ordinary).Error; err != nil {
		t.Fatalf("create ordinary: %v", err)
	}

	for _, uid := range []int{agent.Id, ordinary.Id} {
		ov := UserModelPricingOverride{
			UserId:               uid,
			ModelName:            "gpt-test",
			Mode:                 UserPricingModePriceCap,
			PriceDiscountPercent: 50,
			OperatingCostPercent: 0,
			MarkupDiscountRate:   5, // 指定售价 55 折
			Enabled:              true,
			CreatedTime:          time.Now().Unix(),
			UpdatedTime:          time.Now().Unix(),
		}
		if err := db.Create(&ov).Error; err != nil {
			t.Fatalf("create override: %v", err)
		}
	}
	InvalidateUserModelPricingCache()

	baseItem := PricingAPIItem{
		ChannelList: []PricingChannelItem{{
			ChannelID:            101,
			ModelRatio:           8,
			EffectiveCostPercent: 50,
			PriceDiscountPercent: 50,
			MarkupDiscountRate:   5,
		}},
	}
	baseItem.ModelName = "gpt-test"

	agentOut := ApplyUserPricingOverrideToPricingAPI(agent.Id, []PricingAPIItem{clonePricingAPIItem(baseItem)})
	if len(agentOut) != 1 || len(agentOut[0].ChannelList) != 1 {
		t.Fatalf("agent pricing filter = %#v", agentOut)
	}
	ach := agentOut[0].ChannelList[0]
	if ach.ModelRatio != 8 {
		t.Fatalf("agent should keep channel model ratio, got %v", ach.ModelRatio)
	}
	if ach.EffectiveCostPercent != 50 {
		t.Fatalf("agent should keep channel cost percent, got %v", ach.EffectiveCostPercent)
	}
	if ach.MarkupDiscountRate != 0 {
		t.Fatalf("agent display markup should be 0, got %v", ach.MarkupDiscountRate)
	}

	ordinaryOut := ApplyUserPricingOverrideToPricingAPI(ordinary.Id, []PricingAPIItem{clonePricingAPIItem(baseItem)})
	if len(ordinaryOut) != 1 || len(ordinaryOut[0].ChannelList) != 1 {
		t.Fatalf("ordinary pricing filter = %#v", ordinaryOut)
	}
	och := ordinaryOut[0].ChannelList[0]
	if och.ModelRatio != 10 {
		t.Fatalf("ordinary should rewrite to global ratio 10, got %v", och.ModelRatio)
	}
	if och.EffectiveCostPercent != 50 || och.MarkupDiscountRate != 5 {
		t.Fatalf("ordinary discounts = cost %v markup %v, want 50/5", och.EffectiveCostPercent, och.MarkupDiscountRate)
	}
}

func clonePricingAPIItem(in PricingAPIItem) PricingAPIItem {
	out := in
	if in.ChannelList != nil {
		out.ChannelList = append([]PricingChannelItem(nil), in.ChannelList...)
	}
	return out
}
