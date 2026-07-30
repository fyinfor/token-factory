package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestComputePageFileLifecycle(t *testing.T) {
	setting := operation_setting.GetOssSetting()
	originalStoragePath := setting.LocalStoragePath
	setting.LocalStoragePath = t.TempDir()
	t.Cleanup(func() {
		setting.LocalStoragePath = originalStoragePath
	})

	config, err := GetComputePageConfig()
	require.NoError(t, err)
	require.False(t, config.Enabled)
	require.False(t, config.HasHTML)

	_, err = UpdateComputePageEnabled(true)
	require.EqualError(t, err, "请先上传 HTML 文件")

	firstHTML := []byte("<!doctype html><title>first</title>")
	config, err = SaveComputePageHTML("compute.html", firstHTML)
	require.NoError(t, err)
	require.True(t, config.HasHTML)
	require.Equal(t, "compute.html", config.FileName)

	config, err = UpdateComputePageEnabled(true)
	require.NoError(t, err)
	require.True(t, config.Enabled)

	content, err := ReadEnabledComputePageHTML()
	require.NoError(t, err)
	require.Equal(t, firstHTML, content)

	secondHTML := []byte("<!doctype html><title>second</title>")
	config, err = SaveComputePageHTML("replacement.htm", secondHTML)
	require.NoError(t, err)
	require.True(t, config.Enabled)
	require.Equal(t, "replacement.htm", config.FileName)

	content, err = ReadEnabledComputePageHTML()
	require.NoError(t, err)
	require.Equal(t, secondHTML, content)

	config, err = UpdateComputePageEnabled(false)
	require.NoError(t, err)
	require.False(t, config.Enabled)

	_, err = ReadEnabledComputePageHTML()
	require.True(t, errors.Is(err, ErrComputePageDisabled))
}
