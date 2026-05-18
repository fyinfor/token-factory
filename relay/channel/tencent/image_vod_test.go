package tencent

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestCapTencentOutputImageCount(t *testing.T) {
	if got := capTencentOutputImageCount("OG", 2); got != 2 {
		t.Fatalf("OG n=2: got %d", got)
	}
	if got := capTencentOutputImageCount("OG", 20); got != 8 {
		t.Fatalf("OG cap: got %d want 8", got)
	}
	if got := capTencentOutputImageCount("Kling", 9); got != 9 {
		t.Fatalf("Kling n=9: got %d", got)
	}
}

func TestNormalizeTencentImageSizeString(t *testing.T) {
	got, ok := normalizeTencentImageSizeString("854x480")
	if !ok || got != "848x480" {
		t.Fatalf("854x480: got %q ok=%v want 848x480", got, ok)
	}
	got, ok = normalizeTencentImageSizeString("1280x720")
	if !ok || got != "1280x720" {
		t.Fatalf("1280x720: got %q", got)
	}
}

func TestEnrichTencentVODImageBodyMapsNAndSize(t *testing.T) {
	n := uint(2)
	req := dto.ImageRequest{
		Prompt: "生成小猫图片",
		Size:   "1280x720",
		N:      &n,
	}
	body := map[string]any{}
	enrichTencentVODImageBody(body, "OG", req)
	oc, ok := body["OutputConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing OutputConfig: %#v", body)
	}
	if oc["OutputImageCount"] != 2 {
		t.Fatalf("OutputImageCount: %#v", oc["OutputImageCount"])
	}
	if oc["AspectRatio"] != "16:9" {
		t.Fatalf("AspectRatio: %#v", oc["AspectRatio"])
	}
	ext, ok := body["ExtInfo"].(string)
	if !ok || ext == "" {
		t.Fatalf("ExtInfo: %#v", body["ExtInfo"])
	}
}

func TestEnrichTencentVODImageBodyAlignsSize(t *testing.T) {
	req := dto.ImageRequest{
		Prompt: "test",
		Size:   "854x480",
	}
	body := map[string]any{}
	enrichTencentVODImageBody(body, "OG", req)
	ext, ok := body["ExtInfo"].(string)
	if !ok || !strings.Contains(ext, "848x480") {
		t.Fatalf("ExtInfo should contain 848x480, got %q", ext)
	}
}
