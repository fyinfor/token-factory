package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestModelTagsMatchMonitorFilter(t *testing.T) {
	textOnlyFilter := buildModelTagFilter([]string{"文本"})

	tests := []struct {
		name   string
		tags   string
		filter map[string]struct{}
		want   bool
	}{
		{
			name:   "explicit text tag matches",
			tags:   "文本,热门",
			filter: textOnlyFilter,
			want:   true,
		},
		{
			name:   "video only does not contain text",
			tags:   "视频",
			filter: textOnlyFilter,
			want:   false,
		},
		{
			name:   "image only does not contain text",
			tags:   "图片",
			filter: textOnlyFilter,
			want:   false,
		},
		{
			name:   "empty tags skipped",
			tags:   "",
			filter: textOnlyFilter,
			want:   false,
		},
		{
			name:   "text and video still matches text because text is explicit",
			tags:   "文本,视频",
			filter: textOnlyFilter,
			want:   true,
		},
		{
			name:   "empty filter never matches",
			tags:   "文本",
			filter: buildModelTagFilter(nil),
			want:   false,
		},
		{
			name:   "empty filter never matches video",
			tags:   "视频",
			filter: map[string]struct{}{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modelTagsMatchMonitorFilter(tt.tags, tt.filter); got != tt.want {
				t.Fatalf("modelTagsMatchMonitorFilter(%q) = %v, want %v", tt.tags, got, tt.want)
			}
		})
	}
}

func TestCollectModelsForScheduledChannelTestEmptyFilterSkipsAll(t *testing.T) {
	testModel := "gpt-4o"
	ch := &model.Channel{
		Models:    "gpt-4o,kling-v1,dall-e-3",
		TestModel: &testModel,
	}
	got := collectModelsForScheduledChannelTest(ch, buildModelTagFilter(nil))
	if len(got) != 0 {
		t.Fatalf("empty filter must skip all models, got %v", got)
	}
	got = collectModelsForScheduledChannelTest(ch, map[string]struct{}{})
	if len(got) != 0 {
		t.Fatalf("empty map filter must skip all models, got %v", got)
	}
	got = collectModelsForScheduledChannelTest(nil, buildModelTagFilter([]string{"文本"}))
	if len(got) != 0 {
		t.Fatalf("nil channel must skip, got %v", got)
	}
}

func TestModelMatchesTagFilterUsesExactNameWithoutNameRule(t *testing.T) {
	filter := buildModelTagFilter([]string{"文本"})
	// DB 未初始化时精确查询返回空标签，必须跳过，不能靠 name_rule 推断为文本。
	if modelMatchesTagFilter("gpt-4o", filter) {
		t.Fatal("without exact model_name row, gpt-4o must be skipped")
	}
	if modelMatchesTagFilter("Kling-3.0", filter) {
		t.Fatal("prefix video name must not be inferred as text")
	}
}
