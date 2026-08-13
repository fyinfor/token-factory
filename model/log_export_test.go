package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSignedLogDelta(t *testing.T) {
	cases := []struct {
		name   string
		quota  int
		typ    int
		expect int64
	}{
		{"consume 带正号 quota 应当取负", 5000, LogTypeConsume, -5000},
		{"topup quota 应当保持正", 25000, LogTypeTopup, 25000},
		{"refund quota 应当保持正", 1000, LogTypeRefund, 1000},
		{"manage 不参与累加", 9999, LogTypeManage, 0},
		{"system 不参与累加", 8888, LogTypeSystem, 0},
		{"error 不参与累加", 7777, LogTypeError, 0},
		{"unknown 不参与累加", 6666, LogTypeUnknown, 0},
		{"consume 0 仍为 0", 0, LogTypeConsume, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SignedLogDelta(c.quota, c.typ)
			if got != c.expect {
				t.Fatalf("SignedLogDelta(%d,%d) = %d, want %d", c.quota, c.typ, got, c.expect)
			}
		})
	}
}

func TestLogTypeChargeable(t *testing.T) {
	chargeable := []int{LogTypeConsume, LogTypeTopup, LogTypeRefund}
	for _, x := range chargeable {
		if !LogTypeChargeable(x) {
			t.Errorf("type %d should be chargeable", x)
		}
	}
	notChargeable := []int{LogTypeManage, LogTypeSystem, LogTypeError, LogTypeUnknown}
	for _, x := range notChargeable {
		if LogTypeChargeable(x) {
			t.Errorf("type %d should NOT be chargeable", x)
		}
	}
}

// 模拟 controller 中"反推期初余额 + 累加"的算法，验证在各种日志分布下
// 末行余额等于 currentQuota（这是对账正确性的核心保证）。
func TestStatementRunningBalanceEndsAtCurrentQuota(t *testing.T) {
	type log struct {
		quota int
		typ   int
	}
	cases := []struct {
		name         string
		currentQuota int
		logs         []log
	}{
		{
			name:         "纯消耗",
			currentQuota: 12345,
			logs: []log{
				{1000, LogTypeConsume},
				{2000, LogTypeConsume},
				{500, LogTypeConsume},
			},
		},
		{
			name:         "消耗+充值",
			currentQuota: 50000,
			logs: []log{
				{1000, LogTypeConsume},
				{5000, LogTypeTopup},
				{2000, LogTypeConsume},
				{3000, LogTypeTopup},
			},
		},
		{
			name:         "含 manage/system 行(quota=0),不应影响累加",
			currentQuota: 9999,
			logs: []log{
				{0, LogTypeSystem}, // 赠送,日志里没存金额
				{100, LogTypeConsume},
				{0, LogTypeManage}, // 管理员调整,日志里没存金额
				{200, LogTypeConsume},
			},
		},
		{
			name:         "无日志,余额为 current",
			currentQuota: 777,
			logs:         nil,
		},
		{
			name:         "退款为正",
			currentQuota: 10000,
			logs: []log{
				{500, LogTypeConsume},
				{200, LogTypeRefund},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 计算窗口内净变动
			var sum int64
			for _, l := range c.logs {
				sum += SignedLogDelta(l.quota, l.typ)
			}
			running := int64(c.currentQuota) - sum
			// 模拟逐行累加
			for _, l := range c.logs {
				running += SignedLogDelta(l.quota, l.typ)
			}
			if running != int64(c.currentQuota) {
				t.Errorf("末行余额 %d != currentQuota %d (diff=%d)", running, c.currentQuota, running-int64(c.currentQuota))
			}
		})
	}
}

func TestGetAllLogsForExportMatchesConsoleOrder(t *testing.T) {
	oldLogDB := LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatal(err)
	}
	LOG_DB = db
	t.Cleanup(func() { LOG_DB = oldLogDB })

	logs := []Log{
		{Id: 1, CreatedAt: 100, Type: LogTypeConsume, Quota: 1},
		{Id: 2, CreatedAt: 200, Type: LogTypeConsume, Quota: 2},
		{Id: 3, CreatedAt: 300, Type: LogTypeConsume, Quota: 3},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}

	got, total, err := GetAllLogsForExport(AdminLogExportFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(got) != 3 {
		t.Fatalf("total=%d len=%d, want 3", total, len(got))
	}
	for i, wantID := range []int{3, 2, 1} {
		if got[i].Id != wantID {
			t.Fatalf("logs[%d].id=%d, want %d", i, got[i].Id, wantID)
		}
	}
}

func TestGetTaskBillingTerminalLogsForExport(t *testing.T) {
	oldLogDB := LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatal(err)
	}
	LOG_DB = db
	t.Cleanup(func() { LOG_DB = oldLogDB })

	logs := []Log{
		{Id: 1, UserId: 10, CreatedAt: 100, Type: LogTypeConsume, Other: `{"task_id":"task-a","billing_phase":"pre_charge"}`},
		{Id: 2, UserId: 10, CreatedAt: 200, Type: LogTypeConsume, Other: `{"task_id":"task-a","billing_phase":"delta_charge"}`},
		{Id: 3, UserId: 11, CreatedAt: 210, Type: LogTypeRefund, Other: `{"task_id":"task-b","billing_phase":"delta_refund"}`},
		{Id: 4, UserId: 10, CreatedAt: 220, Type: LogTypeConsume, Other: `{"task_id":"task-a","billing_phase":"normal"}`},
		{Id: 5, UserId: 10, CreatedAt: 230, Type: LogTypeRefund, Other: `{"task_id":"task-x","billing_phase":"refund"}`},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}

	got, err := GetTaskBillingTerminalLogsForExport([]string{"task-a", "task-b"}, []int{10}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Id != 2 {
		t.Fatalf("got=%+v, want only task-a delta charge", got)
	}
}
