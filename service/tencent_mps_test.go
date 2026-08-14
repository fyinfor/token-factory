package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestParseVideoUpscaleOutputPath(t *testing.T) {
	out, err := parseVideoUpscaleOutputPath("https://bucket-125000.cos.ap-guangzhou.myqcloud.com/upscale/")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if out.Bucket != "bucket-125000" || out.Region != "ap-guangzhou" {
		t.Fatalf("unexpected bucket/region: %+v", out)
	}
	if out.OutputDir != "/upscale/" {
		t.Fatalf("unexpected output dir: %s", out.OutputDir)
	}
	got := joinCOSObjectURL(out.BaseURL, "/upscale/a.mp4")
	want := "https://bucket-125000.cos.ap-guangzhou.myqcloud.com/upscale/a.mp4"
	if got != want {
		t.Fatalf("join url got %s want %s", got, want)
	}
}

func TestBuildVideoUpscalePublicURL_OutputPathPlusFileName(t *testing.T) {
	configured := "https://bucket-125000.cos.ap-guangzhou.myqcloud.com/upscale/"
	got := buildVideoUpscalePublicURL(configured, "/video/output.mp4")
	want := "https://bucket-125000.cos.ap-guangzhou.myqcloud.com/video/output.mp4"
	if got != want {
		t.Fatalf("full object path got %s want %s", got, want)
	}
	got = buildVideoUpscalePublicURL(configured, "video_only.mp4")
	want = "https://bucket-125000.cos.ap-guangzhou.myqcloud.com/video_only.mp4"
	if got != want {
		t.Fatalf("filename-only got %s want %s", got, want)
	}
}

func TestSanitizeChannelVideoUpscaleRulesDedup(t *testing.T) {
	rules := SanitizeChannelVideoUpscaleRules([]dto.ChannelVideoUpscaleRule{
		{SourceResolution: "480p", TargetResolution: "720p", TemplateId: 1},
		{SourceResolution: "540p", TargetResolution: "720P", TemplateId: 2},
		{SourceResolution: "720p", TargetResolution: "1080p", TemplateId: 0},
	})
	if len(rules) != 1 {
		t.Fatalf("want 1 unique target rule, got %d: %+v", len(rules), rules)
	}
	if rules[0].TargetResolution != "720p" || rules[0].TemplateId != 1 {
		t.Fatalf("unexpected kept rule: %+v", rules[0])
	}
}

func TestVideoUpscalePricePerSecond_MatchSourceAndTarget(t *testing.T) {
	modelName := "upscale-price-source-target"
	prev := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prev) })
	cfg := `{"` + modelName + `":{"video_upscale_per_second":[{"resolution":"720p","source_resolution":"480p","price":0.02},{"resolution":"1080p","source_resolution":"720p","price":0.05}]}}`
	if err := ratio_setting.UpdateVideoPricingRulesByJSONString(cfg); err != nil {
		t.Fatalf("update rules: %v", err)
	}
	price, ok := VideoUpscalePricePerSecond(0, modelName, "720p", "480p")
	if !ok || price != 0.02 {
		t.Fatalf("480p->720p price = %v ok=%v, want 0.02", price, ok)
	}
	price, ok = VideoUpscalePricePerSecond(0, modelName, "1080p", "720p")
	if !ok || price != 0.05 {
		t.Fatalf("720p->1080p price = %v ok=%v, want 0.05", price, ok)
	}
	if _, ok = VideoUpscalePricePerSecond(0, modelName, "720p", "1080p"); ok {
		t.Fatal("unexpected match for 1080p->720p")
	}
}

func TestExtractDurationFromTaskData_SeedanceResult(t *testing.T) {
	body := []byte(`{"status":"succeeded","resolution":"480p","duration":5,"usage":{"total_tokens":38830}}`)
	if got := extractDurationFromTaskData(body); got != 5 {
		t.Fatalf("duration = %d, want 5", got)
	}
	if got := extractTotalTokensFromTaskData(body); got != 38830 {
		t.Fatalf("tokens = %d, want 38830", got)
	}
}
