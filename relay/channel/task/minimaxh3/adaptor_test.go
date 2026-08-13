package minimaxh3

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestNormalizeBaseURLAndJoin(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", DefaultBaseURL},
		{"https://api.minimaxi.com/v2", "https://api.minimaxi.com/v2"},
		{"https://api.minimaxi.com/v2/", "https://api.minimaxi.com/v2"},
		{"https://api.minimaxi.com/v2/video_generation", "https://api.minimaxi.com/v2"},
		{"https://api.minimaxi.com/v2/query/video_generation", "https://api.minimaxi.com/v2"},
		{"https://api.minimaxi.com", "https://api.minimaxi.com/v2"},
		{"https://proxy.example.com/v2/", "https://proxy.example.com/v2"},
	}
	for _, tt := range tests {
		if got := NormalizeBaseURL(tt.in); got != tt.want {
			t.Fatalf("NormalizeBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	if got := SubmitURL("https://api.minimaxi.com/v2/"); got != "https://api.minimaxi.com/v2/video_generation" {
		t.Fatalf("SubmitURL = %q", got)
	}
	if got := QueryURL("https://api.minimaxi.com/v2", "424010985738629"); got != "https://api.minimaxi.com/v2/query/video_generation/424010985738629" {
		t.Fatalf("QueryURL = %q", got)
	}
}

func TestApplyAuthHeaders(t *testing.T) {
	h := make(http.Header)
	ApplyAuthHeaders(h, "sk-test")
	if got := h.Get("Authorization"); got != "Bearer sk-test" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := h.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestValidateNormalizesResolutionAliases(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"768p", Resolution768P},
		{"768P", Resolution768P},
		{"768", Resolution768P},
		{"1366x768", Resolution768P},
		{"2k", Resolution2K},
		{"2K", Resolution2K},
	}
	for _, tt := range cases {
		req := &VideoGenerationV2Req{
			Model:      ModelMiniMaxH3,
			Resolution: tt.in,
			Duration:   5,
			Ratio:      Ratio16x9,
			Content: []ContentItem{
				{Type: ContentTypeText, Text: "一个男孩在海边打篮球"},
			},
		}
		if err := ValidateVideoGenerationV2Req(req); err != nil {
			t.Fatalf("resolution %q should pass: %v", tt.in, err)
		}
		if req.Resolution != tt.want {
			t.Fatalf("resolution %q normalized to %q, want %q", tt.in, req.Resolution, tt.want)
		}
	}
}

func TestValidateT2VA(t *testing.T) {
	req := &VideoGenerationV2Req{
		Model:      ModelMiniMaxH3,
		Resolution: Resolution2K,
		Duration:   5,
		Ratio:      Ratio16x9,
		Content: []ContentItem{
			{Type: ContentTypeText, Text: "一个男孩在海边打篮球"},
		},
	}
	if err := ValidateVideoGenerationV2Req(req); err != nil {
		t.Fatalf("t2va should pass: %v", err)
	}
}

func TestValidateT2VARejectsAdaptiveRatio(t *testing.T) {
	req := &VideoGenerationV2Req{
		Model:      ModelMiniMaxH3,
		Resolution: Resolution768P,
		Duration:   5,
		Ratio:      RatioAdaptive,
		Content: []ContentItem{
			{Type: ContentTypeText, Text: "hello"},
		},
	}
	if err := ValidateVideoGenerationV2Req(req); err == nil {
		t.Fatal("t2va adaptive ratio should fail")
	}
}

func TestValidateI2VAForcesAdaptive(t *testing.T) {
	req := &VideoGenerationV2Req{
		Model:      ModelMiniMaxH3,
		Resolution: Resolution2K,
		Duration:   5,
		Ratio:      Ratio16x9,
		Content: []ContentItem{
			{Type: ContentTypeText, Text: "pull focus"},
			{Type: ContentTypeImageURL, ImageURL: &MediaURL{URL: "https://cdn.example.com/a.png"}, Role: RoleFirstFrame},
		},
	}
	if err := ValidateVideoGenerationV2Req(req); err != nil {
		t.Fatalf("i2va should pass: %v", err)
	}
	if req.Ratio != RatioAdaptive {
		t.Fatalf("i2va ratio = %q, want adaptive", req.Ratio)
	}
}

func TestValidateRejectsMixedRoles(t *testing.T) {
	req := &VideoGenerationV2Req{
		Model:      ModelMiniMaxH3,
		Resolution: Resolution2K,
		Duration:   5,
		Content: []ContentItem{
			{Type: ContentTypeText, Text: "mix"},
			{Type: ContentTypeImageURL, ImageURL: &MediaURL{URL: "https://cdn.example.com/a.png"}, Role: RoleFirstFrame},
			{Type: ContentTypeVideoURL, VideoURL: &MediaURL{URL: "https://cdn.example.com/a.mp4"}, Role: RoleReferenceVideo},
		},
	}
	if err := ValidateVideoGenerationV2Req(req); err == nil {
		t.Fatal("mixed i2va/r2va should fail")
	}
}

func TestValidateMissingText(t *testing.T) {
	req := &VideoGenerationV2Req{
		Model:      ModelMiniMaxH3,
		Resolution: Resolution2K,
		Duration:   5,
		Ratio:      Ratio16x9,
		Content: []ContentItem{
			{Type: ContentTypeImageURL, ImageURL: &MediaURL{URL: "https://cdn.example.com/a.png"}, Role: RoleFirstFrame},
		},
	}
	if err := ValidateVideoGenerationV2Req(req); err == nil {
		t.Fatal("missing text should fail")
	}
}

func TestConvertGatewayT2VA(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:      ModelMiniMaxH3,
		Prompt:     "一个男孩在海边打篮球",
		Duration:   5,
		Resolution: Resolution2K,
		Ratio:      Ratio16x9,
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeMiniMaxH3Video}}
	payload, err := convertToRequestPayload(&req, info)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if payload.Model != ModelMiniMaxH3 || payload.Duration != 5 || payload.Ratio != Ratio16x9 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Content) != 1 || payload.Content[0].Type != ContentTypeText {
		t.Fatalf("content = %+v", payload.Content)
	}
}

func TestParseTaskResultSucceeded(t *testing.T) {
	body := []byte(`{
		"task": {
			"id": "424010985738629",
			"model": "MiniMax-H3",
			"status": "succeeded",
			"content": {"url": "https://cdn.example.com/out.mp4"},
			"resolution": "2K",
			"duration": 5,
			"usage": {"total_seconds": 5, "output_seconds": 5},
			"ratio": "16:9"
		}
	}`)
	info, err := (&TaskAdaptor{}).ParseTaskResult(body)
	if err != nil {
		t.Fatalf("ParseTaskResult: %v", err)
	}
	if info.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %s", info.Status)
	}
	if info.Url != "https://cdn.example.com/out.mp4" {
		t.Fatalf("url = %s", info.Url)
	}
	if info.Duration != 5 || info.Resolution != "2K" || info.Ratio != "16:9" {
		t.Fatalf("spec duration=%d resolution=%s ratio=%s", info.Duration, info.Resolution, info.Ratio)
	}
}

func TestParseTaskResultFailedAndOaiError(t *testing.T) {
	failed := []byte(`{
		"task": {
			"id": "1",
			"status": "failed",
			"error": {"code": "1026", "message": "sensitive content"}
		}
	}`)
	info, err := (&TaskAdaptor{}).ParseTaskResult(failed)
	if err != nil {
		t.Fatalf("failed parse: %v", err)
	}
	if info.Status != model.TaskStatusFailure || info.Reason != "sensitive content" {
		t.Fatalf("failed result = %+v", info)
	}

	oai := []byte(`{
		"type": "error",
		"error": {"type": "bad_request_error", "message": "invalid task_id (2013)", "http_code": "400"}
	}`)
	info, err = (&TaskAdaptor{}).ParseTaskResult(oai)
	if err != nil {
		t.Fatalf("oai parse: %v", err)
	}
	if info.Status != model.TaskStatusFailure || info.Reason == "" {
		t.Fatalf("oai result = %+v", info)
	}
}
