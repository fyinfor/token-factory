package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func setupDistributorModelLimitContext(t *testing.T, limits string, requestModel string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)

	token := &model.Token{
		ModelLimitsEnabled: limits != "",
		ModelLimits:        limits,
	}
	token.SyncModelLimits()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

	if token.ModelLimitsEnabled {
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, token.GetModelLimitsMap())
	} else {
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, false)
	}

	ctx.Set("request_model_for_test", requestModel)
	return ctx
}

func TestDistributorModelLimitAllowsConfiguredModel(t *testing.T) {
	t.Parallel()

	ctx := setupDistributorModelLimitContext(t, "gpt-4o", "gpt-4o")
	limits, _ := common.GetContextKey(ctx, constant.ContextKeyTokenModelLimit)
	limitsMap := limits.(map[string]bool)

	if !model.ModelLimitMapAllows(limitsMap, "gpt-4o") {
		t.Fatal("expected configured model gpt-4o to pass token model limit check")
	}
}

func TestDistributorModelLimitBlocksUnconfiguredModel(t *testing.T) {
	t.Parallel()

	ctx := setupDistributorModelLimitContext(t, "gpt-4o", "claude-3-5-sonnet-20241022")
	limits, _ := common.GetContextKey(ctx, constant.ContextKeyTokenModelLimit)
	limitsMap := limits.(map[string]bool)

	if model.ModelLimitMapAllows(limitsMap, "claude-3-5-sonnet-20241022") {
		t.Fatal("expected unconfigured model to be blocked by token model limit check")
	}
}

func TestDistributorModelLimitDisabledAllowsAnyModel(t *testing.T) {
	t.Parallel()

	ctx := setupDistributorModelLimitContext(t, "", "claude-3-5-sonnet-20241022")
	if common.GetContextKeyBool(ctx, constant.ContextKeyTokenModelLimitEnabled) {
		t.Fatal("expected model limit to be disabled when whitelist is empty")
	}
}
