package controller

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSupplierScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	model.DB = db
	if err := db.AutoMigrate(&model.User{}, &model.Channel{}, &model.SupplierApplication{}, &model.Model{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func channelIDsInScope(t *testing.T, scope supplierDashboardScope) map[int]struct{} {
	t.Helper()
	out := make(map[int]struct{}, len(scope.ChannelIDs))
	for _, id := range scope.ChannelIDs {
		out[id] = struct{}{}
	}
	return out
}

func TestCollectSupplierDashboardScope_includesChannelsByUserSupplierID(t *testing.T) {
	setupSupplierScopeTestDB(t)

	supplierUserID := 100
	otherOwnerID := 200
	supplierAppID := 5

	if err := model.DB.Create(&model.User{
		Id:         supplierUserID,
		Username:   "supplier-user",
		AffCode:    "aff-supplier-100",
		SupplierID: supplierAppID,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Create(&model.User{
		Id:       otherOwnerID,
		Username: "other-owner",
		AffCode:  "aff-other-200",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Create(&model.Channel{
		Id:                    13,
		OwnerUserID:           otherOwnerID,
		SupplierApplicationID: supplierAppID,
		Name:                  "shared-supplier-channel",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Create(&model.Channel{
		Id:          18,
		OwnerUserID: supplierUserID,
		Name:        "owned-channel",
	}).Error; err != nil {
		t.Fatal(err)
	}

	scope, err := collectSupplierDashboardScope(supplierUserID)
	if err != nil {
		t.Fatal(err)
	}
	ids := channelIDsInScope(t, scope)
	if _, ok := ids[13]; !ok {
		t.Fatalf("expected channel 13 in scope, got %v", scope.ChannelIDs)
	}
	if _, ok := ids[18]; !ok {
		t.Fatalf("expected channel 18 in scope, got %v", scope.ChannelIDs)
	}
}

func TestSupplierApplicationIDsForUser_prefersUsersSupplierID(t *testing.T) {
	setupSupplierScopeTestDB(t)

	userID := 100
	appID := 5
	if err := model.DB.Create(&model.User{
		Id: userID, Username: "u", AffCode: "aff-u-100", SupplierID: appID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	ids, err := supplierApplicationIDsForUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != appID {
		t.Fatalf("expected [%d], got %v", appID, ids)
	}
}
