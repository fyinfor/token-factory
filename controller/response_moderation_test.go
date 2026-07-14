package controller

import "testing"

func TestExtractModerationTextJSON(t *testing.T) {
	body := []byte("{\"choices\":[{\"message\":{\"role\":\"assistant\",\"content\":\"hello world\"}}]}")
	if got := extractModerationText(body, false); got != "hello world" {
		t.Fatalf("unexpected text: %q", got)
	}
}

func TestExtractModerationTextStream(t *testing.T) {
	body := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n")
	if got := extractModerationText(body, true); got != "hello world" {
		t.Fatalf("unexpected stream text: %q", got)
	}
}

func TestExtractModerationImages(t *testing.T) {
	body := []byte("{\"data\":[{\"url\":\"https://example.com/a.png\"},{\"b64_json\":\"aGVsbG8=\"}]}")
	files := extractModerationImages(body)
	if len(files) != 2 {
		t.Fatalf("expected 2 images, got %d", len(files))
	}
	if !files[0].Source.IsURL() || !files[1].Source.IsBase64() {
		t.Fatalf("unexpected image source types")
	}
}
