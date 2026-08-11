package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputePageContentSecurityPolicy(t *testing.T) {
	t.Run("javascript disabled", func(t *testing.T) {
		policy := computePageContentSecurityPolicy(false)
		require.Contains(t, policy, "sandbox allow-same-origin")
		require.NotContains(t, policy, "allow-scripts")
		require.NotContains(t, policy, "script-src")
	})

	t.Run("javascript enabled", func(t *testing.T) {
		policy := computePageContentSecurityPolicy(true)
		require.Contains(t, policy, "sandbox allow-same-origin allow-scripts")
		require.Contains(t, policy, "script-src 'unsafe-inline' 'unsafe-eval'")
		require.Contains(t, policy, "connect-src https: http: ws: wss:")
		require.Equal(t, 1, strings.Count(policy, "script-src"))
	})
}
