package common

import "testing"

func TestNormalizeTaskVideoMetadata_FromSize(t *testing.T) {
	meta := NormalizeTaskVideoMetadata(map[string]interface{}{"duration": 5}, "864x480", nil, nil)
	if meta["resolution"] != "480p" {
		t.Fatalf("resolution=%v want 480p", meta["resolution"])
	}
	if meta["ratio"] != "16:9" {
		t.Fatalf("ratio=%v want 16:9", meta["ratio"])
	}
}

func TestNormalizeTaskVideoMetadata_KeepsExplicitResolution(t *testing.T) {
	meta := NormalizeTaskVideoMetadata(map[string]interface{}{
		"resolution": "1080p",
		"ratio":      "16:9",
	}, "864x480", nil, nil)
	if meta["resolution"] != "1080p" {
		t.Fatalf("resolution=%v", meta["resolution"])
	}
}

func TestPixelsToResolution_768p(t *testing.T) {
	if got := PixelsToResolution(1366, 768); got != "768p" {
		t.Fatalf("got %q want 768p", got)
	}
	if got := PixelsToResolution(1280, 720); got != "720p" {
		t.Fatalf("got %q want 720p", got)
	}
}
