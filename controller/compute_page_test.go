package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputePageContentSecurityPolicy(t *testing.T) {
	t.Run("javascript disabled", func(t *testing.T) {
		policy := computePageContentSecurityPolicy(false, false)
		require.Contains(t, policy, "sandbox allow-same-origin")
		require.NotContains(t, policy, "allow-popups")
		require.NotContains(t, policy, "allow-scripts")
		require.NotContains(t, policy, "script-src")
	})

	t.Run("javascript enabled without external links", func(t *testing.T) {
		policy := computePageContentSecurityPolicy(true, false)
		require.Contains(t, policy, "sandbox allow-same-origin allow-scripts")
		require.NotContains(t, policy, "allow-popups")
		require.Contains(t, policy, "script-src 'unsafe-inline' 'unsafe-eval'")
		require.Contains(t, policy, "connect-src https: http: ws: wss:")
		require.Equal(t, 1, strings.Count(policy, "script-src"))
	})

	t.Run("external links enabled without javascript", func(t *testing.T) {
		policy := computePageContentSecurityPolicy(false, true)
		require.Contains(t, policy, "sandbox allow-same-origin allow-popups allow-popups-to-escape-sandbox")
		require.NotContains(t, policy, "allow-scripts")
		require.NotContains(t, policy, "script-src")
	})
}
