package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupInvoiceTestDB(t *testing.T) {
	t.Helper()
	previousDB := DB
	previousSQLite := common.UsingSQLite
	previousPostgres := common.UsingPostgreSQL
	t.Cleanup(func() {
		DB = previousDB
		common.UsingSQLite = previousSQLite
		common.UsingPostgreSQL = previousPostgres
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	DB = db
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	require.NoError(t, db.AutoMigrate(
		&TopUp{},
		&InvoiceProfile{},
		&InvoiceRequest{},
		&InvoiceRequestItem{},
		&TopUpConsumeAttribution{},
	))
}

func createInvoiceTestData(t *testing.T) (*TopUp, *InvoiceProfile) {
	t.Helper()
	topUp := &TopUp{
		UserId:          1,
		Money:           100,
		QuotaToAdd:      1000,
		TradeNo:         "invoice-test-topup",
		Status:          common.TopUpStatusSuccess,
		InvoiceEligible: true,
		CreateTime:      1,
	}
	require.NoError(t, DB.Create(topUp).Error)
	require.NoError(t, DB.Create(&TopUpConsumeAttribution{
		UserId:        1,
		TopUpId:       topUp.Id,
		ConsumedQuota: 800,
	}).Error)
	profile := &InvoiceProfile{
		UserId:    1,
		TitleType: InvoiceTitleTypeCompany,
		Title:     "Invoice Test Company",
		TaxNo:     "TEST-TAX-NO",
		Email:     "billing@example.com",
	}
	require.NoError(t, DB.Create(profile).Error)
	return topUp, profile
}

func reloadInvoiceTopUp(t *testing.T, id int) TopUp {
	t.Helper()
	var topUp TopUp
	require.NoError(t, DB.First(&topUp, id).Error)
	return topUp
}

func TestCreateInvoiceRequestMergesDuplicateTopUpsAndReservesAmount(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, profile := createInvoiceTestData(t)

	request, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{
		{TopUpId: topUp.Id, InvoiceAmount: 25},
		{TopUpId: topUp.Id, InvoiceAmount: 15},
	}, "", profile)
	require.NoError(t, err)
	assert.InDelta(t, 40, request.TotalAmount, 0.000001)

	items, err := GetInvoiceRequestItems(request.Id)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.InDelta(t, 40, items[0].InvoiceAmount, 0.000001)

	updated := reloadInvoiceTopUp(t, topUp.Id)
	assert.InDelta(t, 40, updated.PendingInvoiceAmount, 0.000001)
}

func TestCreateInvoiceRequestRejectsOverReservationWithoutPartialChanges(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, profile := createInvoiceTestData(t)

	_, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{
		{TopUpId: topUp.Id, InvoiceAmount: 101},
	}, "", profile)
	require.Error(t, err)

	updated := reloadInvoiceTopUp(t, topUp.Id)
	assert.Zero(t, updated.PendingInvoiceAmount)
	var requestCount int64
	require.NoError(t, DB.Model(&InvoiceRequest{}).Count(&requestCount).Error)
	assert.Zero(t, requestCount)
}

func TestSuccessfulTopUpSupportsPartialInvoicesUpToRechargeAmount(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, profile := createInvoiceTestData(t)
	topUp.Money = 10000
	require.NoError(t, DB.Save(topUp).Error)

	first, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 7000}}, "", profile)
	require.NoError(t, err)
	require.NoError(t, IssueInvoiceRequest(first.Id, "INV-7000", "/invoice-7000.pdf", ""))

	orders, err := ListInvoiceEligibleOrders(1, "")
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.InDelta(t, 3000, orders[0].InvoiceableAmount, 0.000001)

	second, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 3000}}, "", profile)
	require.NoError(t, err)
	require.NoError(t, IssueInvoiceRequest(second.Id, "INV-3000", "/invoice-3000.pdf", ""))

	updated := reloadInvoiceTopUp(t, topUp.Id)
	assert.InDelta(t, 10000, updated.InvoicedAmount, 0.000001)
	_, err = CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 0.01}}, "", profile)
	require.Error(t, err)
}

func TestSuccessfulTopUpIsInvoiceableWithoutConsumptionOrEligibilityFlag(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, profile := createInvoiceTestData(t)
	require.NoError(t, DB.Where("user_id = ?", 1).Delete(&TopUpConsumeAttribution{}).Error)
	require.NoError(t, DB.Model(topUp).Update("invoice_eligible", false).Error)

	request, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 100}}, "", profile)
	require.NoError(t, err)
	assert.InDelta(t, 100, request.TotalAmount, 0.000001)
}

func TestRejectAndCancelInvoiceRequestsReleaseReservations(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, profile := createInvoiceTestData(t)

	request, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 30}}, "", profile)
	require.NoError(t, err)
	require.NoError(t, RejectInvoiceRequest(request.Id, "invalid title"))
	assert.Zero(t, reloadInvoiceTopUp(t, topUp.Id).PendingInvoiceAmount)

	request, err = CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 20}}, "", profile)
	require.NoError(t, err)
	require.NoError(t, CancelInvoiceRequest(1, request.Id))
	assert.Zero(t, reloadInvoiceTopUp(t, topUp.Id).PendingInvoiceAmount)
	require.Error(t, CancelInvoiceRequest(1, request.Id))
}

func TestIssueInvoiceRequestConsumesReservationOnlyOnce(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, profile := createInvoiceTestData(t)

	request, err := CreateInvoiceRequest(1, []InvoiceRequestItemInput{{TopUpId: topUp.Id, InvoiceAmount: 50}}, "", profile)
	require.NoError(t, err)
	require.NoError(t, MarkInvoiceRequestProcessing(request.Id))
	require.NoError(t, IssueInvoiceRequest(request.Id, "INV-CODE", "/invoice.pdf", "done"))

	updated := reloadInvoiceTopUp(t, topUp.Id)
	assert.Zero(t, updated.PendingInvoiceAmount)
	assert.InDelta(t, 50, updated.InvoicedAmount, 0.000001)
	require.Error(t, IssueInvoiceRequest(request.Id, "INV-CODE", "/invoice.pdf", "done"))
	updated = reloadInvoiceTopUp(t, topUp.Id)
	assert.InDelta(t, 50, updated.InvoicedAmount, 0.000001)
}

func TestReleaseConsumeQuotaFromTopUpsUsesReverseFIFO(t *testing.T) {
	setupInvoiceTestDB(t)
	first := &TopUp{UserId: 1, TradeNo: "first", Status: common.TopUpStatusSuccess, InvoiceEligible: true}
	second := &TopUp{UserId: 1, TradeNo: "second", Status: common.TopUpStatusSuccess, InvoiceEligible: true}
	require.NoError(t, DB.Create(first).Error)
	require.NoError(t, DB.Create(second).Error)
	require.NoError(t, DB.Create(&TopUpConsumeAttribution{UserId: 1, TopUpId: first.Id, ConsumedQuota: 100, UpdatedAt: 10}).Error)
	require.NoError(t, DB.Create(&TopUpConsumeAttribution{UserId: 1, TopUpId: second.Id, ConsumedQuota: 80, UpdatedAt: 20}).Error)

	require.NoError(t, ReleaseConsumeQuotaFromTopUps(1, 120))
	var rows []TopUpConsumeAttribution
	require.NoError(t, DB.Where("user_id = ?", 1).Order("id asc").Find(&rows).Error)
	require.Len(t, rows, 2)
	assert.Equal(t, 60, rows[0].ConsumedQuota)
	assert.Zero(t, rows[1].ConsumedQuota)
}

func TestBackfillPendingInvoiceAmountsRebuildsActiveReservations(t *testing.T) {
	setupInvoiceTestDB(t)
	topUp, _ := createInvoiceTestData(t)
	topUp.PendingInvoiceAmount = 99
	require.NoError(t, DB.Save(topUp).Error)
	request := &InvoiceRequest{UserId: 1, RequestNo: "pending-backfill", Status: InvoiceRequestStatusPending, TotalAmount: 35, ProfileSnapshot: "{}"}
	require.NoError(t, DB.Create(request).Error)
	require.NoError(t, DB.Create(&InvoiceRequestItem{InvoiceRequestId: request.Id, TopUpId: topUp.Id, TradeNo: topUp.TradeNo, InvoiceAmount: 35}).Error)

	require.NoError(t, BackfillPendingInvoiceAmounts())
	assert.InDelta(t, 35, reloadInvoiceTopUp(t, topUp.Id).PendingInvoiceAmount, 0.000001)
}
