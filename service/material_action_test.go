package service

import (
	"errors"
	"testing"
)
func TestMaterialActionRegistryIncludesAssetAPIs(t *testing.T) {
	required := []string{
		MaterialActionCreateAssetGroup,
		MaterialActionGetAssetGroup,
		MaterialActionCreateAsset,
		MaterialActionGetAsset,
		MaterialActionUpdateAssetGroup,
		MaterialActionUpdateAsset,
		MaterialActionDeleteAsset,
		MaterialActionDeleteAssetGroup,
		MaterialActionCreateVisualValidateSession,
		MaterialActionGetVisualValidateResult,
	}
	for _, action := range required {
		if !IsSupportedMaterialAction(action) {
			t.Fatalf("action %q not registered", action)
		}
	}
}

func TestDispatchMaterialActionRejectsUnknown(t *testing.T) {
	_, err := DispatchMaterialAction(1, "NotExists", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	var bizErr *MaterialActionError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected MaterialActionError, got %T", err)
	}
	if bizErr.Code != 400 {
		t.Fatalf("expected code 400, got %d", bizErr.Code)
	}
}
