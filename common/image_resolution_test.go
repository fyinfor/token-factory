package common

import "testing"

func TestFormatImageResolutionLabel(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"1024x1024", "1080p"},
		{"1024x1536", "1080p"},
		{"1536x864", "1080p"},
		{"1280x720", "1080p"},
		{"720p", "1080p"},
		{"1K", "1080p"},
		{"1k", "1080p"},
		{"1080p", "1080p"},
		{"1080", "1080p"},
		{"1920x1080", "1080p"}, // 定价编辑器 1080p 存值，勿归为 2K
		{"1080x1920", "1080p"},
		{"1536x2048", "2K"},
		{"1080x1080", "2K"}, // 短边 1080 → 2K；非经典 1920x1080
		{"2048x2048", "2K"},
		{"2048x1152", "2K"},
		{"2560x1440", "2K"},
		{"2K", "2K"},
		{"2160x3840", "4K"},
		{"3840x2160", "4K"},
		{"4096x4096", "4K"},
		{"4K", "4K"},
		{"4097x100", "1080p"}, // 短边 100 ≤1024
	}
	for _, tc := range cases {
		if got := FormatImageResolutionLabel(tc.raw); got != tc.want {
			t.Fatalf("FormatImageResolutionLabel(%q)=%q, want %q", tc.raw, got, tc.want)
		}
	}
}
