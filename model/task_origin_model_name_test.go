package model

import (
	"encoding/json"
	"testing"
)

func TestTaskGetOriginModelNamePrefersBillingContext(t *testing.T) {
	t.Parallel()

	task := &Task{
		Properties: Properties{OriginModelName: "from-properties"},
		PrivateData: TaskPrivateData{
			BillingContext: &TaskBillingContext{OriginModelName: "from-billing"},
		},
	}
	if got := task.GetOriginModelName(); got != "from-billing" {
		t.Fatalf("GetOriginModelName() = %q, want %q", got, "from-billing")
	}
}

func TestTaskGetFetchAccessModelNameFallsBackToDataModel(t *testing.T) {
	t.Parallel()

	task := &Task{
		Data: json.RawMessage(`{"model":"sora-2"}`),
	}
	if got := task.GetFetchAccessModelName(); got != "sora-2" {
		t.Fatalf("GetFetchAccessModelName() = %q, want %q", got, "sora-2")
	}
}

func TestTaskGetOriginModelNameNilSafe(t *testing.T) {
	t.Parallel()

	var task *Task
	if got := task.GetOriginModelName(); got != "" {
		t.Fatalf("nil task GetOriginModelName() = %q, want empty", got)
	}
	if got := task.GetFetchAccessModelName(); got != "" {
		t.Fatalf("nil task GetFetchAccessModelName() = %q, want empty", got)
	}
}
