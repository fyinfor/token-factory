package controller

import "testing"

func TestModelTagsMatchMonitorFilter(t *testing.T) {
	textOnlyFilter := buildModelTagFilter([]string{"文本"})

	tests := []struct {
		name   string
		tags   string
		filter map[string]struct{}
		want   bool
	}{
		{
			name:   "text tag matches text filter",
			tags:   "文本,热门",
			filter: textOnlyFilter,
			want:   true,
		},
		{
			name:   "video only excluded by text filter",
			tags:   "视频",
			filter: textOnlyFilter,
			want:   false,
		},
		{
			name:   "text and video excluded when only text selected",
			tags:   "文本,视频",
			filter: textOnlyFilter,
			want:   false,
		},
		{
			name:   "empty tags excluded when filter set",
			tags:   "",
			filter: textOnlyFilter,
			want:   false,
		},
		{
			name:   "both categories selected allows multimodal",
			tags:   "文本,视频",
			filter: buildModelTagFilter([]string{"文本", "视频"}),
			want:   true,
		},
		{
			name:   "empty filter matches all",
			tags:   "视频",
			filter: buildModelTagFilter(nil),
			want:   true,
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
