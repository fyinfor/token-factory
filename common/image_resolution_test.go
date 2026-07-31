package common

import "testing"

func TestFormatImageResolutionLabel(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"512x512", "512P"},
		{"512x768", "512P"},
		{"480x480", "512P"},
		{"512p", "512P"},
		{"512P", "512P"},
		{"512", "512P"},
		{"1024x1024", "1K"},
		{"1024x1536", "1K"},
		{"1536x864", "1K"},
		{"1280x720", "1K"},
		{"720p", "1K"},
		{"513x513", "1K"},
		{"1K", "1K"},
		{"1k", "1K"},
		{"1080p", "1K"}, // 历史计费别名 ≡ 1K
		{"1080", "1K"},
		{"1920x1080", "2K"}, // 短边 1080 → 2K
		{"1080x1920", "2K"},
		{"1648x1232", "2K"}, // 短边 1232 → 2K
		{"1536x2048", "2K"},
		{"1080x1080", "2K"}, // 短边 1080 → 2K
		{"2048x2048", "2K"},
		{"2048x1152", "2K"},
		{"2560x1440", "2K"},
		{"2K", "2K"},
		{"2160x3840", "4K"},
		{"3840x2160", "4K"},
		{"4096x4096", "4K"},
		{"4K", "4K"},
		{"4097x100", "512P"}, // 短边 100 ≤512
		{"4097x800", "1K"},   // 短边 800 → 1K
	}
	for _, tc := range cases {
		if got := FormatImageResolutionLabel(tc.raw); got != tc.want {
			t.Fatalf("FormatImageResolutionLabel(%q)=%q, want %q", tc.raw, got, tc.want)
		}
	}
}
