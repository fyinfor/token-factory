package common

import "testing"

func TestParseVideoResolutionAndRatio_480p16x9(t *testing.T) {
	w, h, ok := ParseVideoResolutionAndRatio("480p", "16:9")
	if !ok {
		t.Fatal("expected ok")
	}
	if w != 854 || h != 480 {
		t.Fatalf("got %dx%d", w, h)
	}
}

func TestVideoDimensionsFromMetadata(t *testing.T) {
	w, h, ok := VideoDimensionsFromMetadata(map[string]interface{}{
		"resolution": "720p",
		"ratio":      "9:16",
	})
	if !ok {
		t.Fatal("expected ok")
	}
	if w >= h {
		t.Fatalf("portrait expected height > width, got %dx%d", w, h)
	}
}

func TestFormatVideoSpecLabel(t *testing.T) {
	if got := FormatVideoSpecLabel("1280x720", "16:9"); got != "720p 16:9" {
		t.Fatalf("got %q", got)
	}
	if got := FormatVideoSpecLabel("480p", "16:9"); got != "480p 16:9" {
		t.Fatalf("got %q", got)
	}
	if got := FormatVideoSpecLabel("1280x720", ""); got != "720p" {
		t.Fatalf("got %q", got)
	}
}
