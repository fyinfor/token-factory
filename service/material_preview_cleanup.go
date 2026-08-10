package service

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	materialPreviewCleanupInterval  = 10 * time.Minute
	materialPreviewCleanupBatchSize = 100
)

var (
	materialPreviewCleanupOnce    sync.Once
	materialPreviewCleanupRunning atomic.Bool
)

// StartMaterialPreviewCleanupTask 定期清理到期的素材库预览文件。
// 只删除本地/OSS 预览对象并清空 material_assets.url，绝不删除素材行与上游 asset。
func StartMaterialPreviewCleanupTask() {
	materialPreviewCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			hours := operation_setting.GetMaterialPreviewRetentionHours()
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"material preview cleanup task started: tick=%s retention=%dh",
				materialPreviewCleanupInterval,
				hours,
			))
			runMaterialPreviewCleanupOnce()
			ticker := time.NewTicker(materialPreviewCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				runMaterialPreviewCleanupOnce()
			}
		})
	})
}

func runMaterialPreviewCleanupOnce() {
	if !materialPreviewCleanupRunning.CompareAndSwap(false, true) {
		return
	}
	defer materialPreviewCleanupRunning.Store(false)

	ctx := context.Background()
	assets, err := model.ListExpiredMaterialPreviews(time.Now().Unix(), materialPreviewCleanupBatchSize)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("material preview cleanup query failed: %v", err))
		return
	}
	cleared := 0
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		previewURL := strings.TrimSpace(asset.URL)
		if previewURL != "" {
			if err := CleanupManagedUploadByURL(previewURL); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf(
					"material preview file cleanup failed: asset_id=%s err=%v",
					asset.AssetId,
					err,
				))
				// 文件删失败仍清空 URL，避免接口继续指向失效预览；对象可由下次重试或 Bucket 生命周期兜底。
			}
		}
		if err := model.ClearMaterialAssetPreview(asset.Id); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf(
				"material preview url clear failed: id=%d asset_id=%s err=%v",
				asset.Id,
				asset.AssetId,
				err,
			))
			continue
		}
		cleared++
	}
	if cleared > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("material preview cleanup completed: cleared=%d", cleared))
	}
}

// IsManagedUploadURL 判断 URL 是否为本系统本地 uploads 或当前 OSS 配置下的对象地址。
func IsManagedUploadURL(publicURL string) bool {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return false
	}
	if strings.Contains(publicURL, "/"+LocalUploadFolder+"/") {
		return true
	}
	cfg := operation_setting.GetOssSetting()
	if !operation_setting.IsOssUploadReady() {
		return false
	}
	_, ok := parseOssObjectKeyFromURL(cfg, publicURL)
	return ok
}

// CleanupManagedUploadByURL 删除本地或 OSS 托管的预览/中转文件（best-effort）。
// 对外链（非本系统 uploads / OSS bucket URL）直接忽略。
func CleanupManagedUploadByURL(publicURL string) error {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return nil
	}
	_ = CleanupLocalUploadByURL(publicURL)
	return CleanupOssUploadByURL(publicURL)
}

// CleanupOssUploadByURL 若 URL 属于当前配置的 OSS 公网地址，则删除对应对象。
func CleanupOssUploadByURL(publicURL string) error {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return nil
	}
	cfg := operation_setting.GetOssSetting()
	if !operation_setting.IsOssUploadReady() {
		return nil
	}
	objectKey, ok := parseOssObjectKeyFromURL(cfg, publicURL)
	if !ok {
		return nil
	}
	return ossDeleteObject(cfg.Endpoint, cfg.Bucket, objectKey)
}

func parseOssObjectKeyFromURL(cfg *operation_setting.OssSetting, publicURL string) (string, bool) {
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	rawPath := strings.TrimLeft(parsed.EscapedPath(), "/")
	if rawPath == "" {
		return "", false
	}
	unescaped, err := url.PathUnescape(rawPath)
	if err == nil {
		rawPath = unescaped
	}
	rawPath = strings.TrimLeft(path.Clean("/"+rawPath), "/")
	if rawPath == "." || rawPath == "" || strings.HasPrefix(rawPath, "..") {
		return "", false
	}

	host := strings.ToLower(parsed.Host)
	bases := make([]string, 0, 2)
	if base := strings.TrimSpace(cfg.PublicBaseURL); base != "" {
		bases = append(bases, strings.TrimRight(base, "/"))
	}
	ep := strings.TrimSpace(cfg.Endpoint)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	bkt := strings.TrimSpace(cfg.Bucket)
	if ep != "" && bkt != "" {
		bases = append(bases, "https://"+bkt+"."+ep)
	}
	for _, base := range bases {
		bu, err := url.Parse(base)
		if err != nil || bu.Host == "" {
			continue
		}
		if !strings.EqualFold(bu.Host, host) {
			continue
		}
		basePath := strings.Trim(bu.EscapedPath(), "/")
		if basePath == "" {
			return rawPath, true
		}
		prefix := basePath + "/"
		if strings.HasPrefix(rawPath, prefix) {
			return strings.TrimPrefix(rawPath, prefix), true
		}
	}
	return "", false
}
