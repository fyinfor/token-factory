package service

import (
	"testing"
)

func TestFilterTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "filter out context window size like 262.1K",
			input:    "Reasoning,Tools,Files,Vision,262.1K",
			expected: "Reasoning,Tools,Files,Vision",
		},
		{
			name:     "filter out 128K context size",
			input:    "Reasoning,128K",
			expected: "Reasoning",
		},
		{
			name:     "filter out 1M context size",
			input:    "Vision,1M,Chat",
			expected: "Vision,Chat",
		},
		{
			name:     "filter out numeric-only tags",
			input:    "Coding,32K,8K",
			expected: "Coding",
		},
		{
			name:     "all valid tags preserved",
			input:    "Reasoning,Tools,Files,Vision",
			expected: "Reasoning,Tools,Files,Vision",
		},
		{
			name:     "all invalid tags removed",
			input:    "262.1K,128K,1.5M",
			expected: "",
		},
		{
			name:     "case insensitive matching",
			input:    "REASONING,tools,VISION",
			expected: "REASONING,tools,VISION",
		},
		{
			name:     "preserve open weights tag",
			input:    "Open Weights,Vision,128K",
			expected: "Open Weights,Vision",
		},
		{
			name:     "whitespace handling",
			input:    "  Reasoning ,  Tools , 262.1K ,  Vision  ",
			expected: "Reasoning,Tools,Vision",
		},
		{
			name:     "mixed valid and invalid tags from official preset",
			input:    "Reasoning,Tools,Files,Vision,262.1K,Proprietary",
			expected: "Reasoning,Tools,Files,Vision,Proprietary",
		},
		{
			name:     "single valid tag",
			input:    "Embedding",
			expected: "Embedding",
		},
		{
			name:     "single invalid tag",
			input:    "200K",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterTags(tt.input)
			if result != tt.expected {
				t.Errorf("filterTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
