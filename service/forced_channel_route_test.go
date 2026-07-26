package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseModelRouteIndex_InvalidOrNotSlug(t *testing.T) {
	cases := []struct {
		raw string
	}{
		{""},
		{"glm-5.2"},
		{"openai/gpt-4o"},          // hyphen in last segment → not route_slug
		{"alias/c2"},               // cN reserved
		{"P5/claude-haiku-4-5"},    // hyphen model → not route_slug form for last segment
	}
	for _, tc := range cases {
		got, matched, err := ParseModelRouteIndex(tc.raw)
		require.NoError(t, err, tc.raw)
		require.False(t, matched, tc.raw)
		require.Nil(t, got, tc.raw)
	}
}
