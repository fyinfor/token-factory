package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidChannelSyncKey(t *testing.T) {
	require.False(t, IsValidChannelSyncKey(""))
	require.False(t, IsValidChannelSyncKey("   "))
	require.True(t, IsValidChannelSyncKey("abc123"))
	require.True(t, IsValidChannelSyncKey(NewChannelSyncKey()))
	require.False(t, IsValidChannelSyncKey("has space"))
	require.False(t, IsValidChannelSyncKey(strings.Repeat("a", 65)))
	require.True(t, IsValidChannelSyncKey(strings.Repeat("a", 64)))
}

func TestEnsureChannelSyncKey(t *testing.T) {
	ch := &Channel{}
	EnsureChannelSyncKey(ch)
	require.NotEmpty(t, ch.SyncKey)
	require.Len(t, ch.SyncKey, 6)
	require.True(t, IsValidChannelSyncKey(ch.SyncKey))

	ch2 := &Channel{SyncKey: "  custom-key  "}
	EnsureChannelSyncKey(ch2)
	require.Equal(t, "custom-key", ch2.SyncKey)
}

func TestNewChannelSyncKeyLength(t *testing.T) {
	key := NewChannelSyncKey()
	require.Len(t, key, 6)
	require.True(t, IsValidChannelSyncKey(key))
}
