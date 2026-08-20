package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func setupTokenModelContext(t *testing.T, limits string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)

	token := &model.Token{
		ModelLimitsEnabled: limits != "",
		ModelLimits:        limits,
	}
	token.SyncModelLimits()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)

	if token.ModelLimitsEnabled {
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, token.GetModelLimitsMap())
	} else {
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, false)
	}
	return ctx
}

func TestTokenAllowsModelWhenLimitDisabled(t *testing.T) {
	t.Parallel()

	ctx := setupTokenModelContext(t, "")
	if !TokenAllowsModel(ctx, "sora-2") {
		t.Fatal("expected any model to pass when token model limit is disabled")
	}
}

func TestTokenAllowsModelWhenConfigured(t *testing.T) {
	t.Parallel()

	ctx := setupTokenModelContext(t, "sora-2,whisper-1")
	if !TokenAllowsModel(ctx, "sora-2") {
		t.Fatal("expected configured video model to pass token whitelist")
	}
	if TokenAllowsModel(ctx, "gpt-4o") {
		t.Fatal("expected unconfigured model to be blocked")
	}
}

func TestTokenAllowsModelRejectsEmptyNameWhenEnabled(t *testing.T) {
	t.Parallel()

	ctx := setupTokenModelContext(t, "sora-2")
	if TokenAllowsModel(ctx, "") {
		t.Fatal("empty model name must not pass an enabled whitelist")
	}
}

func TestFetchOwnerUserIDAdminBypassesOwnership(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 2)
	ctx.Set("role", common.RoleAdminUser)

	if got := FetchOwnerUserID(ctx); got != 0 {
		t.Fatalf("admin FetchOwnerUserID() = %d, want 0 (no owner filter)", got)
	}
}

func TestFetchOwnerUserIDRegularUserKeepsOwnership(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 42)
	ctx.Set("role", common.RoleCommonUser)

	if got := FetchOwnerUserID(ctx); got != 42 {
		t.Fatalf("regular FetchOwnerUserID() = %d, want 42", got)
	}
}
