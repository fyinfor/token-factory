package common

import "testing"

func TestDetectVideoBillingMode_InputReferenceImage(t *testing.T) {
	req := &TaskSubmitReq{InputReference: "https://example.com/ref.png"}
	if got := DetectVideoBillingMode(req); got != VideoBillingModeImageToVideo {
		t.Fatalf("expected %s, got %s", VideoBillingModeImageToVideo, got)
	}
}

func TestDetectVideoBillingMode_InputReferenceVideo(t *testing.T) {
	req := &TaskSubmitReq{InputReference: "https://example.com/ref.mp4?token=abc"}
	if got := DetectVideoBillingMode(req); got != VideoBillingModeVideoToVideo {
		t.Fatalf("expected %s, got %s", VideoBillingModeVideoToVideo, got)
	}
}

func TestDetectVideoBillingMode_MetadataVideoURLs(t *testing.T) {
	req := &TaskSubmitReq{Metadata: map[string]interface{}{
		"video_urls": []interface{}{"https://example.com/ref.mov"},
	}}
	if got := DetectVideoBillingMode(req); got != VideoBillingModeVideoToVideo {
		t.Fatalf("expected %s, got %s", VideoBillingModeVideoToVideo, got)
	}
}

func TestDetectVideoBillingMode_MetadataVideoURLsAssetURI(t *testing.T) {
	req := &TaskSubmitReq{Metadata: map[string]interface{}{
		"video_urls": []interface{}{"asset://asset-2026xxxx"},
	}}
	if got := DetectVideoBillingMode(req); got != VideoBillingModeVideoToVideo {
		t.Fatalf("expected %s, got %s", VideoBillingModeVideoToVideo, got)
	}
}

func TestDetectVideoBillingMode_MetadataVideoURLSingular(t *testing.T) {
	req := &TaskSubmitReq{Metadata: map[string]interface{}{
		"video_url": "asset://asset-2026xxxx",
	}}
	if got := DetectVideoBillingMode(req); got != VideoBillingModeVideoToVideo {
		t.Fatalf("expected %s, got %s", VideoBillingModeVideoToVideo, got)
	}
}

func TestDetectVideoBillingMode_Images(t *testing.T) {
	req := &TaskSubmitReq{Images: []string{"https://example.com/ref.jpg"}}
	if got := DetectVideoBillingMode(req); got != VideoBillingModeImageToVideo {
		t.Fatalf("expected %s, got %s", VideoBillingModeImageToVideo, got)
	}
}
