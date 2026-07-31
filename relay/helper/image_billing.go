package helper

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// FinalizeImagePerImageBilling adjusts billing from the upstream image response:
// actual image count and optional resolution inferred from output/input images.
func FinalizeImagePerImageBilling(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, responseBody []byte) {
	if info == nil || info.ImageBilling == nil || !info.PriceData.UsePrice {
		return
	}

	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	modelName := info.OriginModelName
	if !HasImageGenerationPricing(channelID, modelName) {
		return
	}

	actualCount := countImagesInResponseBody(responseBody)
	if actualCount <= 0 {
		if request != nil && request.N != nil && *request.N > 0 {
			actualCount = int(*request.N)
		} else if n, ok := info.PriceData.OtherRatios["n"]; ok && n > 0 {
			actualCount = int(math.Round(n))
		} else {
			actualCount = 1
		}
	}

	estimateCtx := estimateImageRequestContext(c, info)
	estimateCtx.Count = actualCount
	// Prefer upstream-reported pixel size (e.g. Tencent DescribeTaskDetail MetaData)
	// over request.Size so per-image tier billing can be reconciled after generation.
	if info.ImageBilling != nil && info.ImageBilling.DimensionsFromUpstream &&
		info.ImageBilling.Width > 0 && info.ImageBilling.Height > 0 {
		estimateCtx.Width = info.ImageBilling.Width
		estimateCtx.Height = info.ImageBilling.Height
	} else if w, h, ok := resolveImageDimensions(c, request, responseBody); ok {
		estimateCtx.Width = w
		estimateCtx.Height = h
	}

	if !SyncImagePerImagePriceData(c, info, estimateCtx) {
		info.PriceData.AddOtherRatio("n", float64(actualCount))
		if info.ImageBilling != nil {
			info.ImageBilling.Width = estimateCtx.Width
			info.ImageBilling.Height = estimateCtx.Height
			info.ImageBilling.Count = actualCount
			info.ImageBilling.Mode = string(estimateCtx.Mode)
		}
		return
	}

	if common.DebugEnabled && info.ImageBilling != nil {
		logger.LogDebug(c, fmt.Sprintf(
			"[image][finalize] model=%s mode=%s w=%d h=%d actualCount=%d channelUSD=%.6f globalUSD=%.6f effUSD=%.6f quota=%d",
			modelName, estimateCtx.Mode, estimateCtx.Width, estimateCtx.Height, actualCount,
			info.PriceData.ModelPrice, info.PriceData.GlobalModelPrice, info.ImageBilling.UsdPerImage, info.PriceData.Quota,
		))
	}
}

// ApplyActualImageDimensionsForBilling 用上游实际像素覆盖预扣尺寸，并立即重匹按张计费档位。
// 供渠道在写出响应 metadata 前调用，使 resolution 能反映结算档位（512P / 1K / 2K / 4K）。
func ApplyActualImageDimensionsForBilling(c *gin.Context, info *relaycommon.RelayInfo, width, height, count int) {
	if info == nil || info.ImageBilling == nil || !info.PriceData.UsePrice {
		return
	}
	if width <= 0 || height <= 0 {
		return
	}
	info.ImageBilling.Width = width
	info.ImageBilling.Height = height
	info.ImageBilling.DimensionsFromUpstream = true
	if count > 0 {
		info.ImageBilling.Count = count
	}

	estimateCtx := estimateImageRequestContext(c, info)
	estimateCtx.Width = width
	estimateCtx.Height = height
	if count > 0 {
		estimateCtx.Count = count
	} else if info.ImageBilling.Count > 0 {
		estimateCtx.Count = info.ImageBilling.Count
	}
	_ = SyncImagePerImagePriceData(c, info, estimateCtx)
}

func countImagesInResponseBody(body []byte) int {
	body = bytesTrimSpace(body)
	if len(body) == 0 {
		return 0
	}
	var imageResp dto.ImageResponse
	if err := common.Unmarshal(body, &imageResp); err != nil {
		return 0
	}
	count := 0
	for _, item := range imageResp.Data {
		if strings.TrimSpace(item.Url) != "" || strings.TrimSpace(item.B64Json) != "" {
			count++
		}
	}
	return count
}

func resolveImageDimensions(c *gin.Context, request *dto.ImageRequest, responseBody []byte) (int, int, bool) {
	if request != nil {
		if w, h, ok := parseResolutionFlexible(request.Size); ok {
			return w, h, true
		}
	}
	if w, h, ok := dimensionsFromImageResponseBody(c, responseBody); ok {
		return w, h, true
	}
	if request != nil {
		for _, url := range ExtractImageInputURLs(request.Image) {
			if w, h, ok := decodeImageURLDimensions(url); ok {
				return w, h, true
			}
		}
	}
	return 0, 0, false
}

func dimensionsFromImageResponseBody(c *gin.Context, body []byte) (int, int, bool) {
	body = bytesTrimSpace(body)
	if len(body) == 0 {
		return 0, 0, false
	}
	var imageResp dto.ImageResponse
	if err := common.Unmarshal(body, &imageResp); err != nil {
		return 0, 0, false
	}
	for _, item := range imageResp.Data {
		if w, h, ok := dimensionsFromImageData(c, item); ok {
			return w, h, true
		}
	}
	return 0, 0, false
}

func dimensionsFromImageData(c *gin.Context, item dto.ImageData) (int, int, bool) {
	if url := strings.TrimSpace(item.Url); url != "" {
		return decodeImageURLDimensions(url)
	}
	if b64 := strings.TrimSpace(item.B64Json); b64 != "" {
		if cfg, _, _, err := service.DecodeBase64ImageData(b64); err == nil && cfg.Width > 0 && cfg.Height > 0 {
			return cfg.Width, cfg.Height, true
		}
	}
	_ = c
	return 0, 0, false
}

func decodeImageURLDimensions(url string) (int, int, bool) {
	url = strings.TrimSpace(url)
	if url == "" {
		return 0, 0, false
	}
	cfg, _, err := service.DecodeUrlImageData(url)
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// ExtractImageInputURLs 从 ImageRequest.image 字段解析参考图 URL（字符串或数组）。
func ExtractImageInputURLs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return nil
	}
	var single string
	if err := common.Unmarshal(raw, &single); err == nil {
		single = strings.TrimSpace(single)
		if single != "" {
			return []string{single}
		}
	}
	var list []string
	if err := common.Unmarshal(raw, &list); err == nil {
		out := make([]string, 0, len(list))
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var objects []struct {
		URL string `json:"url"`
	}
	if err := common.Unmarshal(raw, &objects); err == nil {
		out := make([]string, 0, len(objects))
		for _, obj := range objects {
			u := strings.TrimSpace(obj.URL)
			if u != "" {
				out = append(out, u)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
