package common

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
)

func TestTaskSubmitReqGetModerationMeta(t *testing.T) {
	req := TaskSubmitReq{
		Prompt:         "main prompt",
		NegativePrompt: "negative prompt",
		Image:          "https://example.com/a.png",
		Images: []string{
			"https://example.com/a.png",
			"data:image/png;base64,aGVsbG8=",
		},
	}
	meta := req.GetModerationMeta()
	if meta.CombineText != "main prompt\nnegative prompt" {
		t.Fatalf("unexpected combined text: %q", meta.CombineText)
	}
	if len(meta.Files) != 2 {
		t.Fatalf("expected 2 unique images, got %d", len(meta.Files))
	}
	if meta.Files[0].FileType != types.FileTypeImage || !meta.Files[0].Source.IsURL() {
		t.Fatalf("expected first file to be a URL image")
	}
	if !meta.Files[1].Source.IsBase64() {
		t.Fatalf("expected second file to be a base64 image")
	}
}
