package service

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	computePageFolder     = "compute"
	computePageHTMLFile   = "index.html"
	computePageConfigFile = "config.json"
)

var (
	ErrComputePageDisabled = errors.New("算力页面未启用")
	computePageMutex       sync.RWMutex
)

type ComputePageConfig struct {
	Enabled         bool   `json:"enabled"`
	AllowJavaScript bool   `json:"allow_javascript"`
	AllowPopups     bool   `json:"allow_popups"`
	ContentURL      string `json:"content_url"`
	RedirectToURL   bool   `json:"redirect_to_url"`
	FileName        string `json:"file_name"`
	UpdatedAt       int64  `json:"updated_at"`
	HasHTML         bool   `json:"-"`
	HasURL          bool   `json:"-"`
	HasContent      bool   `json:"-"`
}

func computePageDir() string {
	storageRoot := operation_setting.GetOssSetting().LocalStoragePath
	return filepath.Join(LocalUploadBaseDir(storageRoot), computePageFolder)
}

func computePageHTMLPath() string {
	return filepath.Join(computePageDir(), computePageHTMLFile)
}

func computePageConfigPath() string {
	return filepath.Join(computePageDir(), computePageConfigFile)
}

func readComputePageConfigLocked() (ComputePageConfig, error) {
	config := ComputePageConfig{}
	data, err := os.ReadFile(computePageConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			populateComputePageAvailability(&config)
			return config, nil
		}
		return config, fmt.Errorf("读取算力页面配置失败: %w", err)
	}
	if err := common.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("解析算力页面配置失败: %w", err)
	}
	config.FileName = strings.TrimSpace(config.FileName)
	config.ContentURL, _ = normalizeComputePageURL(config.ContentURL)
	populateComputePageAvailability(&config)
	return config, nil
}

func normalizeComputePageURL(rawURL string) (string, error) {
	contentURL := strings.TrimSpace(rawURL)
	if contentURL == "" {
		return "", nil
	}

	parsedURL, err := url.ParseRequestURI(contentURL)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", errors.New("网址必须是有效的 HTTP 或 HTTPS 地址")
	}
	return contentURL, nil
}

func populateComputePageAvailability(config *ComputePageConfig) {
	config.HasHTML = computePageHTMLExists()
	config.HasURL = config.ContentURL != ""
	config.HasContent = config.HasHTML || config.HasURL
}

func computePageHTMLExists() bool {
	info, err := os.Stat(computePageHTMLPath())
	return err == nil && !info.IsDir() && info.Size() > 0
}

func writeFileAtomic(targetPath string, data []byte, permission os.FileMode) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, "."+filepath.Base(targetPath)+"-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(permission); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, targetPath); err == nil {
		return nil
	}

	// Windows cannot rename over an existing file, so move the old file aside
	// before installing the new one and restore it if replacement fails.
	if _, err := os.Stat(targetPath); err != nil {
		return err
	}
	backupPath := targetPath + ".backup"
	_ = os.Remove(backupPath)
	if err := os.Rename(targetPath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func writeComputePageConfigLocked(config ComputePageConfig) error {
	config.HasHTML = false
	config.HasURL = false
	config.HasContent = false
	data, err := common.Marshal(config)
	if err != nil {
		return fmt.Errorf("生成算力页面配置失败: %w", err)
	}
	if err := writeFileAtomic(computePageConfigPath(), data, 0o644); err != nil {
		return fmt.Errorf("保存算力页面配置失败: %w", err)
	}
	return nil
}

func GetComputePageConfig() (ComputePageConfig, error) {
	computePageMutex.RLock()
	defer computePageMutex.RUnlock()
	return readComputePageConfigLocked()
}

func UpdateComputePageEnabled(enabled bool) (ComputePageConfig, error) {
	computePageMutex.Lock()
	defer computePageMutex.Unlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return config, err
	}
	if enabled && !config.HasContent {
		return config, errors.New("请先填写网址或上传 HTML 文件")
	}
	config.Enabled = enabled
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	populateComputePageAvailability(&config)
	return config, nil
}

func UpdateComputePageJavaScriptAllowed(allowed bool) (ComputePageConfig, error) {
	computePageMutex.Lock()
	defer computePageMutex.Unlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return config, err
	}
	config.AllowJavaScript = allowed
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	populateComputePageAvailability(&config)
	return config, nil
}

func UpdateComputePagePopupsAllowed(allowed bool) (ComputePageConfig, error) {
	computePageMutex.Lock()
	defer computePageMutex.Unlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return config, err
	}
	config.AllowPopups = allowed
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	populateComputePageAvailability(&config)
	return config, nil
}

func UpdateComputePageURL(rawURL string) (ComputePageConfig, error) {
	computePageMutex.Lock()
	defer computePageMutex.Unlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return config, err
	}
	contentURL, err := normalizeComputePageURL(rawURL)
	if err != nil {
		return config, err
	}
	config.ContentURL = contentURL
	if contentURL == "" {
		config.RedirectToURL = false
	}
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	populateComputePageAvailability(&config)
	return config, nil
}

func UpdateComputePageRedirectToURL(enabled bool) (ComputePageConfig, error) {
	computePageMutex.Lock()
	defer computePageMutex.Unlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return config, err
	}
	if enabled && !config.HasURL {
		return config, errors.New("请先填写网址")
	}
	config.RedirectToURL = enabled
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	populateComputePageAvailability(&config)
	return config, nil
}

func SaveComputePageHTML(fileName string, content []byte) (ComputePageConfig, error) {
	computePageMutex.Lock()
	defer computePageMutex.Unlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return config, err
	}
	if err := writeFileAtomic(computePageHTMLPath(), content, 0o644); err != nil {
		return config, fmt.Errorf("保存算力页面 HTML 失败: %w", err)
	}
	config.FileName = filepath.Base(strings.TrimSpace(fileName))
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	populateComputePageAvailability(&config)
	return config, nil
}

func ReadEnabledComputePageHTML() ([]byte, ComputePageConfig, error) {
	computePageMutex.RLock()
	defer computePageMutex.RUnlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return nil, config, err
	}
	if !config.Enabled || !config.HasHTML {
		return nil, config, ErrComputePageDisabled
	}
	data, err := os.ReadFile(computePageHTMLPath())
	if err != nil {
		return nil, config, fmt.Errorf("读取算力页面 HTML 失败: %w", err)
	}
	return data, config, nil
}
