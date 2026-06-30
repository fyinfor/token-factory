package common

import "testing"

func TestResolveVideoDimensionsFromRequest_ResolutionInMetadata(t *testing.T) {
	w, h, ok := ResolveVideoDimensionsFromRequest("", "", "", map[string]interface{}{
		"resolution": "480p",
		"ratio":      "16:9",
	})
	if !ok || w != 854 || h != 480 {
		t.Fatalf("got %dx%d ok=%v want 854x480", w, h, ok)
	}
}

func TestResolveVideoDimensionsFromRequest_TopLevelResolution(t *testing.T) {
	w, h, ok := ResolveVideoDimensionsFromRequest("", "720p", "16:9", nil)
	if !ok || w != 1280 || h != 720 {
		t.Fatalf("got %dx%d ok=%v want 1280x720", w, h, ok)
	}
}

func TestResolveVideoDimensionsFromRequest_SizePreset(t *testing.T) {
	w, h, ok := ResolveVideoDimensionsFromRequest("480p", "", "16:9", nil)
	if !ok || w != 854 || h != 480 {
		t.Fatalf("got %dx%d ok=%v want 854x480", w, h, ok)
	}
}

func TestResolveVideoDimensionsFromRequest_ResolutionBeforeSize(t *testing.T) {
	// metadata.resolution + ratio 优先于顶层 size（Seedance 2.0 常见请求形态）
	w, h, ok := ResolveVideoDimensionsFromRequest("1920x1080", "", "", map[string]interface{}{
		"resolution": "720p",
		"ratio":      "16:9",
	})
	if !ok || w != 1280 || h != 720 {
		t.Fatalf("got %dx%d ok=%v want 1280x720 from 720p+16:9", w, h, ok)
	}
}

func TestResolveVideoDimensionsFromRequest_Seedance21x9(t *testing.T) {
	w, h, ok := ResolveVideoDimensionsFromRequest("1680x720", "", "", map[string]interface{}{
		"resolution": "720p",
		"ratio":      "21:9",
	})
	if !ok || w != 1680 || h != 720 {
		t.Fatalf("got %dx%d ok=%v want 1680x720 from 720p+21:9", w, h, ok)
	}
}

func TestResolveVideoDimensionsFromRequest_FallsBackToSize(t *testing.T) {
	w, h, ok := ResolveVideoDimensionsFromRequest("1920x1080", "", "", nil)
	if !ok || w != 1920 || h != 1080 {
		t.Fatalf("got %dx%d ok=%v want 1920x1080 from size", w, h, ok)
	}
}
