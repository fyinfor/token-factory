package tencent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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
	oc, ok := body["OutputConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing OutputConfig: %#v", body)
	}
	if oc["AspectRatio"] != "16:9" {
		t.Fatalf("AspectRatio for aligned 854x480 should be 16:9, got %#v", oc["AspectRatio"])
	}
	if _, hasRatio := body["ratio"]; hasRatio {
		t.Fatalf("top-level ratio must not be forwarded to Tencent VOD body: %#v", body["ratio"])
	}
}

func TestEnrichTencentVODImageBodyUsesExplicitRatioAndSkipsTopLevel(t *testing.T) {
	req := dto.ImageRequest{
		Prompt: "test",
		Size:   "854x480",
		Extra: map[string]json.RawMessage{
			"ratio": json.RawMessage(`"16:9"`),
		},
	}
	body := map[string]any{}
	enrichTencentVODImageBody(body, "OG", req)
	if _, hasRatio := body["ratio"]; hasRatio {
		t.Fatalf("top-level ratio must not be forwarded: %#v", body["ratio"])
	}
	oc, ok := body["OutputConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing OutputConfig: %#v", body)
	}
	if oc["AspectRatio"] != "16:9" {
		t.Fatalf("AspectRatio: %#v", oc["AspectRatio"])
	}
}

func TestParseTencentDescribeImageTaskExtractsCommonFields(t *testing.T) {
	body := []byte(`{
  "Response": {
    "TaskType": "AigcImageTask",
    "Status": "FINISH",
    "CreateTime": "2026-07-27T08:21:50Z",
    "FinishTime": "2026-07-27T08:22:30Z",
    "RequestId": "d8d37989-2082-4956-95f6-13ef7fc330b4",
    "AigcImageTask": {
      "Status": "FINISH",
      "ErrCode": 0,
      "Progress": 100,
      "Input": {
        "OutputConfig": {
          "StorageMode": "Temporary",
          "Resolution": "1k",
          "AspectRatio": "16:9",
          "OutputImageCount": 1
        }
      },
      "Output": {
        "FileInfos": [{
          "FileUrl": "http://example.com/aigcImageGenFile.png",
          "MetaData": { "Height": 768, "Width": 1360 }
        }]
      }
    }
  }
}`)
	parsed, err := parseTencentDescribeImageTask(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.taskErr != "" {
		t.Fatalf("unexpected taskErr: %s", parsed.taskErr)
	}
	r := parsed.result
	if r.Status != "FINISH" || r.Progress != 100 {
		t.Fatalf("status/progress: %+v", r)
	}
	if r.CreateTime != "2026-07-27T08:21:50Z" || r.FinishTime != "2026-07-27T08:22:30Z" {
		t.Fatalf("times: %+v", r)
	}
	if r.StorageMode != "Temporary" || r.Resolution != "1k" || r.AspectRatio != "16:9" || r.OutputImageCount != 1 {
		t.Fatalf("output config: %+v", r)
	}
	if r.Width != 1360 || r.Height != 768 {
		t.Fatalf("dimensions: %+v", r)
	}
	if r.RequestId != "d8d37989-2082-4956-95f6-13ef7fc330b4" {
		t.Fatalf("request id: %+v", r)
	}
	if len(r.URLs) != 1 || r.URLs[0] == "" {
		t.Fatalf("urls: %+v", r.URLs)
	}

	meta, err := buildTencentImageCommonMetadata(&r, nil)
	if err != nil || len(meta) == 0 {
		t.Fatalf("metadata: %v %s", err, string(meta))
	}
	var m map[string]any
	if err := json.Unmarshal(meta, &m); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if m["request_id"] != "d8d37989-2082-4956-95f6-13ef7fc330b4" {
		t.Fatalf("meta request_id: %#v", m["request_id"])
	}
	if int(m["width"].(float64)) != 1360 || int(m["height"].(float64)) != 768 {
		t.Fatalf("meta wh: %#v", m)
	}
	if m["size"] != "1360x768" {
		t.Fatalf("meta size: %#v", m["size"])
	}
	// 无计费匹配时，按实际像素短边归一：768 → 1K（512＜短边≤1024）
	if m["resolution"] != "1K" {
		t.Fatalf("meta resolution tier: %#v", m["resolution"])
	}
}

func TestBuildTencentImageCommonMetadataUsesMatchedBillingTier(t *testing.T) {
	r := &tencentImagePollResult{
		Width:  1360,
		Height: 768,
	}
	info := &relaycommon.RelayInfo{
		ImageBilling: &relaycommon.ImageBillingSnapshot{
			RuleRes:    "1920x1080",
			RuleWidth:  1920,
			RuleHeight: 1080,
		},
	}
	meta, err := buildTencentImageCommonMetadata(r, info)
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(meta, &m); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	// 1920x1080 短边 1080 → 图片档位 2K（与视频 1080p 标识相互独立）
	if m["resolution"] != "2K" {
		t.Fatalf("want matched billing tier 2K, got %#v", m["resolution"])
	}
	if m["size"] != "1360x768" {
		t.Fatalf("size: %#v", m["size"])
	}
}

func TestParseTencentDescribeImageTaskUsesVideoStreamDimensions(t *testing.T) {
	body := []byte(`{
  "Response": {
    "Status": "FINISH",
    "RequestId": "rid-1",
    "AigcImageTask": {
      "ErrCode": 0,
      "Progress": 100,
      "Output": {
        "FileInfos": [{
          "FileUrl": "http://example.com/a.png",
          "MetaData": {
            "Height": 0,
            "Width": 0,
            "VideoStreamSet": [{ "Height": 768, "Width": 1360 }]
          }
        }]
      }
    }
  }
}`)
	parsed, err := parseTencentDescribeImageTask(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.result.Width != 1360 || parsed.result.Height != 768 {
		t.Fatalf("want 1360x768 from VideoStreamSet, got %+v", parsed.result)
	}
}

func TestApplyTencentImageActualBillingSetsUpstreamDims(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ImageBilling: &relaycommon.ImageBillingSnapshot{
			Width:  1280,
			Height: 720,
			Count:  1,
		},
	}
	applyTencentImageActualBilling(info, &tencentImagePollResult{
		URLs:      []string{"http://example.com/a.png"},
		Width:     1360,
		Height:    768,
		RequestId: "upstream-rid",
		OutputImageCount: 1,
	})
	if info.UpstreamRequestId != "upstream-rid" {
		t.Fatalf("UpstreamRequestId: %q", info.UpstreamRequestId)
	}
	if !info.ImageBilling.DimensionsFromUpstream {
		t.Fatal("expected DimensionsFromUpstream")
	}
	if info.ImageBilling.Width != 1360 || info.ImageBilling.Height != 768 {
		t.Fatalf("billing dims: %+v", info.ImageBilling)
	}
}

func TestApplyTencentImageActualBillingSkipsMissingDims(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ImageBilling: &relaycommon.ImageBillingSnapshot{
			Width:  1280,
			Height: 720,
		},
	}
	applyTencentImageActualBilling(info, &tencentImagePollResult{
		URLs:      []string{"http://example.com/a.png"},
		RequestId: "rid",
	})
	if info.UpstreamRequestId != "rid" {
		t.Fatalf("UpstreamRequestId: %q", info.UpstreamRequestId)
	}
	if info.ImageBilling.DimensionsFromUpstream {
		t.Fatal("should not mark DimensionsFromUpstream without pixels")
	}
	if info.ImageBilling.Width != 1280 || info.ImageBilling.Height != 720 {
		t.Fatalf("precharge dims should remain: %+v", info.ImageBilling)
	}
}
