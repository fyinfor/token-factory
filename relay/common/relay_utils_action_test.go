package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestResolveTaskActionToStore_VideoURLsAssetURISwitchesReferenceGenerate(t *testing.T) {
	info := &RelayInfo{}
	req := &TaskSubmitReq{
		Prompt: "续写参考视频",
		Metadata: map[string]interface{}{
			"video_urls": []interface{}{"asset://asset-2026xxxx"},
		},
	}
	got := ResolveTaskActionToStore(info, constant.TaskActionGenerate, req)
	if got != constant.TaskActionReferenceGenerate {
		t.Fatalf("expected %s when metadata.video_urls present, got %s", constant.TaskActionReferenceGenerate, got)
	}
}

func TestResolveTaskActionToStore_NoMediaDowngradesTextGenerate(t *testing.T) {
	info := &RelayInfo{}
	req := &TaskSubmitReq{Prompt: "一只猫在跑步"}
	got := ResolveTaskActionToStore(info, constant.TaskActionGenerate, req)
	if got != constant.TaskActionTextGenerate {
		t.Fatalf("expected %s for text-only request, got %s", constant.TaskActionTextGenerate, got)
	}
}

func TestResolveTaskActionToStore_ImageKeepsGenerate(t *testing.T) {
	info := &RelayInfo{}
	req := &TaskSubmitReq{
		Prompt: "让人物微笑",
		Images: []string{"https://example.com/a.jpg"},
	}
	got := ResolveTaskActionToStore(info, constant.TaskActionGenerate, req)
	if got != constant.TaskActionGenerate {
		t.Fatalf("expected %s for image-to-video, got %s", constant.TaskActionGenerate, got)
	}
}

func TestResolveTaskActionToStore_PresetRemixPreserved(t *testing.T) {
	info := &RelayInfo{TaskRelayInfo: &TaskRelayInfo{Action: constant.TaskActionRemix}}
	req := &TaskSubmitReq{Prompt: "remix"}
	got := ResolveTaskActionToStore(info, constant.TaskActionGenerate, req)
	if got != constant.TaskActionRemix {
		t.Fatalf("expected preset remix preserved, got %s", got)
	}
}
