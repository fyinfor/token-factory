package service

import (
	"errors"
	"fmt"
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
	Enabled   bool   `json:"enabled"`
	FileName  string `json:"file_name"`
	UpdatedAt int64  `json:"updated_at"`
	HasHTML   bool   `json:"-"`
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
			config.HasHTML = computePageHTMLExists()
			return config, nil
		}
		return config, fmt.Errorf("读取算力页面配置失败: %w", err)
	}
	if err := common.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("解析算力页面配置失败: %w", err)
	}
	config.FileName = strings.TrimSpace(config.FileName)
	config.HasHTML = computePageHTMLExists()
	return config, nil
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
	if enabled && !config.HasHTML {
		return config, errors.New("请先上传 HTML 文件")
	}
	config.Enabled = enabled
	config.UpdatedAt = time.Now().Unix()
	if err := writeComputePageConfigLocked(config); err != nil {
		return config, err
	}
	config.HasHTML = computePageHTMLExists()
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
	config.HasHTML = true
	return config, nil
}

func ReadEnabledComputePageHTML() ([]byte, error) {
	computePageMutex.RLock()
	defer computePageMutex.RUnlock()

	config, err := readComputePageConfigLocked()
	if err != nil {
		return nil, err
	}
	if !config.Enabled || !config.HasHTML {
		return nil, ErrComputePageDisabled
	}
	data, err := os.ReadFile(computePageHTMLPath())
	if err != nil {
		return nil, fmt.Errorf("读取算力页面 HTML 失败: %w", err)
	}
	return data, nil
}
