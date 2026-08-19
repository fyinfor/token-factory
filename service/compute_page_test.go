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
	require.False(t, config.AllowJavaScript)
	require.False(t, config.AllowPopups)
	require.False(t, config.HasHTML)

	_, err = UpdateComputePageEnabled(true)
	require.EqualError(t, err, "请先填写网址或上传 HTML 文件")

	firstHTML := []byte("<!doctype html><title>first</title>")
	config, err = SaveComputePageHTML("compute.html", firstHTML)
	require.NoError(t, err)
	require.True(t, config.HasHTML)
	require.Equal(t, "compute.html", config.FileName)

	config, err = UpdateComputePageURL(" https://example.com/compute ")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/compute", config.ContentURL)
	require.True(t, config.HasURL)
	require.True(t, config.HasHTML)
	require.True(t, config.HasContent)

	config, err = UpdateComputePageRedirectToURL(true)
	require.NoError(t, err)
	require.True(t, config.RedirectToURL)

	config, err = UpdateComputePageURL("")
	require.NoError(t, err)
	require.Empty(t, config.ContentURL)
	require.False(t, config.HasURL)
	require.True(t, config.HasContent)
	require.False(t, config.RedirectToURL)

	config, err = UpdateComputePageEnabled(true)
	require.NoError(t, err)
	require.True(t, config.Enabled)

	content, contentConfig, err := ReadEnabledComputePageHTML()
	require.NoError(t, err)
	require.Equal(t, firstHTML, content)
	require.False(t, contentConfig.AllowJavaScript)

	config, err = UpdateComputePageJavaScriptAllowed(true)
	require.NoError(t, err)
	require.True(t, config.AllowJavaScript)

	content, contentConfig, err = ReadEnabledComputePageHTML()
	require.NoError(t, err)
	require.Equal(t, firstHTML, content)
	require.True(t, contentConfig.AllowJavaScript)
	require.False(t, contentConfig.AllowPopups)

	config, err = UpdateComputePagePopupsAllowed(true)
	require.NoError(t, err)
	require.True(t, config.AllowPopups)

	content, contentConfig, err = ReadEnabledComputePageHTML()
	require.NoError(t, err)
	require.Equal(t, firstHTML, content)
	require.True(t, contentConfig.AllowPopups)

	secondHTML := []byte("<!doctype html><title>second</title>")
	config, err = SaveComputePageHTML("replacement.htm", secondHTML)
	require.NoError(t, err)
	require.True(t, config.Enabled)
	require.Equal(t, "replacement.htm", config.FileName)

	content, contentConfig, err = ReadEnabledComputePageHTML()
	require.NoError(t, err)
	require.Equal(t, secondHTML, content)
	require.True(t, contentConfig.AllowJavaScript)
	require.True(t, contentConfig.AllowPopups)

	config, err = UpdateComputePageEnabled(false)
	require.NoError(t, err)
	require.False(t, config.Enabled)

	_, _, err = ReadEnabledComputePageHTML()
	require.True(t, errors.Is(err, ErrComputePageDisabled))
}

func TestComputePageURLValidation(t *testing.T) {
	setting := operation_setting.GetOssSetting()
	originalStoragePath := setting.LocalStoragePath
	setting.LocalStoragePath = t.TempDir()
	t.Cleanup(func() {
		setting.LocalStoragePath = originalStoragePath
	})

	_, err := UpdateComputePageURL("javascript:alert(1)")
	require.EqualError(t, err, "网址必须是有效的 HTTP 或 HTTPS 地址")

	_, err = UpdateComputePageRedirectToURL(true)
	require.EqualError(t, err, "请先填写网址")

	config, err := UpdateComputePageURL("https://example.com/embed?plan=pro")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/embed?plan=pro", config.ContentURL)
	require.True(t, config.HasURL)
	require.True(t, config.HasContent)

	config, err = UpdateComputePageEnabled(true)
	require.NoError(t, err)
	require.True(t, config.Enabled)
}
