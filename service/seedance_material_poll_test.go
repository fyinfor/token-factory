package service

import "testing"

func TestShouldContinueMaterialPoll_LocalUpload(t *testing.T) {
	source := "https://example.com/api/uploads/user/2026/01/01/a.jpg"
	info := &MaterialAssetResult{
		Status: MaterialStatusActive,
		URL:    source,
	}
	if !shouldContinueMaterialPoll(info, source) {
		t.Fatal("expected continue when upstream still echoes local temp URL")
	}
	info.URL = "https://cdn.example.com/permanent/a.jpg"
	if shouldContinueMaterialPoll(info, source) {
		t.Fatal("expected stop when permanent URL differs from local temp URL")
	}
}

func TestShouldContinueMaterialPoll_OnlineUpload(t *testing.T) {
	source := "https://other.com/image.jpg"
	info := &MaterialAssetResult{Status: MaterialStatusPending}
	if !shouldContinueMaterialPoll(info, source) {
		t.Fatal("expected continue while pending")
	}
	info.Status = MaterialStatusActive
	info.URL = source
	if shouldContinueMaterialPoll(info, source) {
		t.Fatal("expected stop when active with URL for online upload")
	}
}

func TestNormalizeMaterialStatus(t *testing.T) {
	if NormalizeMaterialStatus("active") != MaterialStatusActive {
		t.Fatal("expected Active")
	}
}
