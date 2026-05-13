package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

const (
	defaultTimeoutSeconds       = 10
	defaultEndpoint             = "/api/ratio_config"
	maxConcurrentFetches        = 8
	maxRatioConfigBytes         = 10 << 20 // 10MB
	floatEpsilon                = 1e-9
	officialRatioPresetID       = -100
	officialRatioPresetName     = "官方倍率预设"
	officialRatioPresetBaseURL  = "https://basellm.github.io"
	modelsDevPresetID           = -101
	modelsDevPresetName         = "models.dev 价格预设"
	modelsDevPresetBaseURL      = "https://models.dev"
	modelsDevHost               = "models.dev"
	modelsDevPath               = "/api.json"
	modelsDevInputCostRatioBase = 1000.0
)

func nearlyEqual(a, b float64) bool {
	if a > b {
		return a-b < floatEpsilon
	}
	return b-a < floatEpsilon
}

func valuesEqual(a, b interface{}) bool {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		return nearlyEqual(af, bf)
	}
	return reflect.DeepEqual(a, b)
}

var ratioTypes = []string{"model_ratio", "completion_ratio", "cache_ratio", "create_cache_ratio", "model_price", "request_tier_pricing"}

func oldChannelValueOrNil(v float64) interface{} {
	if nearlyEqual(v, 0) {
		return nil
	}
	return v
}

func mapValueByModel(src any, modelName string) (any, bool) {
	switch m := src.(type) {
	case map[string]float64:
		v, ok := m[modelName]
		return v, ok
	case map[string]any:
		v, ok := m[modelName]
		return v, ok
	default:
		return nil, false
	}
}

func mapModelNames(src any, allModels map[string]struct{}) {
	switch m := src.(type) {
	case map[string]float64:
		for modelName := range m {
			allModels[modelName] = struct{}{}
		}
	case map[string]any:
		for modelName := range m {
			allModels[modelName] = struct{}{}
		}
	}
}

type upstreamResult struct {
	Name      string         `json:"name"`
	ChannelID int            `json:"channel_id"`
	Data      map[string]any `json:"data,omitempty"`
	Err       string         `json:"err,omitempty"`
}

type pricingChannelItem struct {
	ChannelID        int     `json:"channel_id"`
	QuotaType        int     `json:"quota_type"`
	ModelRatio       float64 `json:"model_ratio"`
	ModelPrice       float64 `json:"model_price"`
	CompletionRatio  float64 `json:"completion_ratio"`
	CacheRatio       float64 `json:"cache_ratio"`
	CreateCacheRatio float64 `json:"create_cache_ratio"`
}

type pricingItem struct {
	ModelName        string               `json:"model_name"`
	QuotaType        int                  `json:"quota_type"`
	ModelRatio       float64              `json:"model_ratio"`
	ModelPrice       float64              `json:"model_price"`
	CompletionRatio  float64              `json:"completion_ratio"`
	CacheRatio       float64              `json:"cache_ratio"`
	CreateCacheRatio float64              `json:"create_cache_ratio"`
	ChannelList      []pricingChannelItem `json:"channel_list"`
}

func upstreamChannelIDForLocalChannel(channelID int) int {
	if channelID <= 0 {
		return 0
	}
	ch, err := model.GetChannelById(channelID, false)
	if err != nil || ch == nil {
		return 0
	}
	otherInfo := ch.GetOtherInfo()
	return common.String2Int(common.Interface2String(otherInfo["upstream_channel_id"]))
}

func shouldPutPricingValue(values map[string]any, modelName string, value float64) bool {
	if _, ok := values[modelName]; !ok {
		return true
	}
	return !nearlyEqual(value, 0)
}

func putPricingValue(values map[string]any, modelName string, value float64) {
	if shouldPutPricingValue(values, modelName, value) {
		values[modelName] = value
	}
}

func putPricingValues(modelName string, modelRatio, completionRatio, cacheRatio, createCacheRatio, modelPrice float64, modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, modelPriceMap map[string]any) {
	putPricingValue(modelRatioMap, modelName, modelRatio)
	putPricingValue(completionRatioMap, modelName, completionRatio)
	putPricingValue(cacheRatioMap, modelName, cacheRatio)
	putPricingValue(createCacheRatioMap, modelName, createCacheRatio)
	putPricingValue(modelPriceMap, modelName, modelPrice)
}

func convertOfficialPricingItemsToRatioData(pricingItems []pricingItem) map[string]any {
	modelRatioMap := make(map[string]any)
	completionRatioMap := make(map[string]any)
	cacheRatioMap := make(map[string]any)
	createCacheRatioMap := make(map[string]any)
	modelPriceMap := make(map[string]any)

	for _, item := range pricingItems {
		modelName := strings.TrimSpace(item.ModelName)
		if modelName == "" {
			continue
		}
		putPricingValues(modelName, item.ModelRatio, item.CompletionRatio, item.CacheRatio, item.CreateCacheRatio, item.ModelPrice, modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, modelPriceMap)
	}

	return buildConvertedPricingData(modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, modelPriceMap)
}

func convertChannelPricingItemsToRatioData(pricingItems []pricingItem, upstreamChannelID int) map[string]any {
	if upstreamChannelID <= 0 {
		return map[string]any{}
	}

	modelRatioMap := make(map[string]any)
	completionRatioMap := make(map[string]any)
	cacheRatioMap := make(map[string]any)
	createCacheRatioMap := make(map[string]any)
	modelPriceMap := make(map[string]any)

	for _, item := range pricingItems {
		modelName := strings.TrimSpace(item.ModelName)
		if modelName == "" {
			continue
		}
		for _, channelItem := range item.ChannelList {
			if channelItem.ChannelID != upstreamChannelID {
				continue
			}
			putPricingValues(modelName, channelItem.ModelRatio, channelItem.CompletionRatio, channelItem.CacheRatio, channelItem.CreateCacheRatio, channelItem.ModelPrice, modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, modelPriceMap)
			break
		}
	}

	return buildConvertedPricingData(modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, modelPriceMap)
}

func buildConvertedPricingData(modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, modelPriceMap map[string]any) map[string]any {
	converted := make(map[string]any)
	if len(modelRatioMap) > 0 {
		converted["model_ratio"] = modelRatioMap
	}
	if len(completionRatioMap) > 0 {
		converted["completion_ratio"] = completionRatioMap
	}
	if len(cacheRatioMap) > 0 {
		converted["cache_ratio"] = cacheRatioMap
	}
	if len(createCacheRatioMap) > 0 {
		converted["create_cache_ratio"] = createCacheRatioMap
	}
	if len(modelPriceMap) > 0 {
		converted["model_price"] = modelPriceMap
	}
	return converted
}

func FetchUpstreamRatios(c *gin.Context) {
	var req dto.UpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.SysError("failed to bind upstream request: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = defaultTimeoutSeconds
	}

	var upstreams []dto.UpstreamDTO

	if len(req.Upstreams) > 0 {
		for _, u := range req.Upstreams {
			if strings.HasPrefix(u.BaseURL, "http") {
				if u.Endpoint == "" {
					u.Endpoint = defaultEndpoint
				}
				u.BaseURL = strings.TrimRight(u.BaseURL, "/")
				upstreams = append(upstreams, u)
			}
		}
	} else if len(req.ChannelIDs) > 0 {
		intIds := make([]int, 0, len(req.ChannelIDs))
		for _, id64 := range req.ChannelIDs {
			intIds = append(intIds, int(id64))
		}
		dbChannels, err := model.GetChannelsByIds(intIds)
		if err != nil {
			logger.LogError(c.Request.Context(), "failed to query channels: "+err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询渠道失败"})
			return
		}
		for _, ch := range dbChannels {
			if base := ch.GetBaseURL(); strings.HasPrefix(base, "http") {
				upstreams = append(upstreams, dto.UpstreamDTO{
					ID:       ch.Id,
					Name:     ch.Name,
					BaseURL:  strings.TrimRight(base, "/"),
					Endpoint: "",
				})
			}
		}
	}

	if len(upstreams) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无有效上游渠道"})
		return
	}

	var wg sync.WaitGroup
	ch := make(chan upstreamResult, len(upstreams))

	sem := make(chan struct{}, maxConcurrentFetches)

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{MaxIdleConns: 100, IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ExpectContinueTimeout: 1 * time.Second, ResponseHeaderTimeout: 10 * time.Second}
	if common.TLSInsecureSkipVerify {
		transport.TLSClientConfig = common.InsecureTLSConfig
	}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		// 对 github.io 优先尝试 IPv4，失败则回退 IPv6
		if strings.HasSuffix(host, "github.io") {
			if conn, err := dialer.DialContext(ctx, "tcp4", addr); err == nil {
				return conn, nil
			}
			return dialer.DialContext(ctx, "tcp6", addr)
		}
		return dialer.DialContext(ctx, network, addr)
	}
	client := &http.Client{Transport: transport}

	for _, chn := range upstreams {
		wg.Add(1)
		go func(chItem dto.UpstreamDTO) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			isOpenRouter := chItem.Endpoint == "openrouter"
			isTokenFactoryOpen := chItem.Endpoint == "tokenfactoryopen"

			endpoint := chItem.Endpoint
			var fullURL string
			if isOpenRouter {
				fullURL = chItem.BaseURL + "/v1/models"
			} else if isTokenFactoryOpen {
				fullURL = chItem.BaseURL + "/api/price_sync"
			} else if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
				fullURL = endpoint
			} else {
				if endpoint == "" {
					endpoint = defaultEndpoint
				} else if !strings.HasPrefix(endpoint, "/") {
					endpoint = "/" + endpoint
				}
				fullURL = chItem.BaseURL + endpoint
			}
			isModelsDev := isModelsDevAPIEndpoint(fullURL)

			uniqueName := chItem.Name
			if chItem.ID != 0 {
				uniqueName = fmt.Sprintf("%s(%d)", chItem.Name, chItem.ID)
			}

			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(req.Timeout)*time.Second)
			defer cancel()

			var openRouterAuthHeader string
			var tokenFactoryOpenBody []byte
			// OpenRouter requires Bearer token auth
			if isOpenRouter && chItem.ID != 0 {
				dbCh, err := model.GetChannelById(chItem.ID, true)
				if err != nil {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "failed to get channel key: " + err.Error()}
					return
				}
				key, _, apiErr := dbCh.GetNextEnabledKey()
				if apiErr != nil {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "failed to get enabled channel key: " + apiErr.Error()}
					return
				}
				if strings.TrimSpace(key) == "" {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "no API key configured for this channel"}
					return
				}
				openRouterAuthHeader = "Bearer " + strings.TrimSpace(key)
			} else if isOpenRouter {
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "OpenRouter requires a valid channel with API key"}
				return
			} else if isTokenFactoryOpen {
				if chItem.ID == 0 {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "TokenFactoryOpen requires a valid channel with API key"}
					return
				}
				dbCh, err := model.GetChannelById(chItem.ID, true)
				if err != nil {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "failed to get channel key: " + err.Error()}
					return
				}
				key, _, apiErr := dbCh.GetNextEnabledKey()
				if apiErr != nil {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "failed to get enabled channel key: " + apiErr.Error()}
					return
				}
				key = strings.TrimSpace(key)
				if key == "" {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "no API key configured for this channel"}
					return
				}
				tokenFactoryOpenBody, err = common.Marshal(map[string]string{"token": key})
				if err != nil {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: err.Error()}
					return
				}
			}

			// 简单重试：最多 3 次，指数退避
			var resp *http.Response
			var lastErr error
			for attempt := 0; attempt < 3; attempt++ {
				method := http.MethodGet
				var reqBody io.Reader
				if isTokenFactoryOpen {
					method = http.MethodPost
					reqBody = bytes.NewReader(tokenFactoryOpenBody)
				}
				httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
				if err != nil {
					logger.LogWarn(c.Request.Context(), "build request failed: "+err.Error())
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: err.Error()}
					return
				}
				if openRouterAuthHeader != "" {
					httpReq.Header.Set("Authorization", openRouterAuthHeader)
				}
				if isTokenFactoryOpen {
					httpReq.Header.Set("Content-Type", "application/json")
				}
				if isModelsDev {
					// models.dev occasionally hits TLS record corruption on keep-alive reuse.
					// Force fresh connection and browser-like UA to improve stability.
					httpReq.Close = true
					httpReq.Header.Set("User-Agent", "Mozilla/5.0")
				}
				resp, lastErr = client.Do(httpReq)
				if lastErr == nil {
					break
				}
				if isModelsDev && isTLSBadRecordMACError(lastErr) {
					transport.CloseIdleConnections()
				}
				time.Sleep(time.Duration(200*(1<<attempt)) * time.Millisecond)
			}
			if lastErr != nil {
				logger.LogWarn(c.Request.Context(), "http error on "+chItem.Name+": "+lastErr.Error())
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: lastErr.Error()}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				logger.LogWarn(c.Request.Context(), "non-200 from "+chItem.Name+": "+resp.Status)
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: resp.Status}
				return
			}

			// Content-Type 和响应体大小校验
			if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "application/json") {
				logger.LogWarn(c.Request.Context(), "unexpected content-type from "+chItem.Name+": "+ct)
			}
			limited := io.LimitReader(resp.Body, maxRatioConfigBytes)
			bodyBytes, err := io.ReadAll(limited)
			if err != nil {
				logger.LogWarn(c.Request.Context(), "read response failed from "+chItem.Name+": "+err.Error())
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: err.Error()}
				return
			}

			// type3: OpenRouter /v1/models -> convert per-token pricing to ratios
			if isOpenRouter {
				converted, err := convertOpenRouterToRatioData(bytes.NewReader(bodyBytes))
				if err != nil {
					logger.LogWarn(c.Request.Context(), "OpenRouter parse failed from "+chItem.Name+": "+err.Error())
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: err.Error()}
					return
				}
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Data: converted}
				return
			}

			// type4: models.dev /api.json -> convert provider model pricing to ratios
			if isModelsDev {
				converted, err := convertModelsDevToRatioData(bytes.NewReader(bodyBytes))
				if err != nil {
					logger.LogWarn(c.Request.Context(), "models.dev parse failed from "+chItem.Name+": "+err.Error())
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: err.Error()}
					return
				}
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Data: converted}
				return
			}

			// 兼容两种上游接口格式：
			//  type1: /api/ratio_config -> data 为 map[string]any，包含 model_ratio/completion_ratio/cache_ratio/model_price
			//  type2: /api/pricing      -> data 为 []Pricing 列表，需要转换为与 type1 相同的 map 格式
			var body struct {
				Success bool            `json:"success"`
				Data    json.RawMessage `json:"data"`
				Message string          `json:"message"`
			}

			if err := common.DecodeJson(bytes.NewReader(bodyBytes), &body); err != nil {
				logger.LogWarn(c.Request.Context(), "json decode failed from "+chItem.Name+": "+err.Error())
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: err.Error()}
				return
			}

			if !body.Success {
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: body.Message}
				return
			}

			// 若 Data 为空，将继续按 type1 尝试解析（与多数静态 ratio_config 兼容）

			// 尝试按 type1 解析
			var type1Data map[string]any
			if err := common.Unmarshal(body.Data, &type1Data); err == nil {
				// 如果包含至少一个 ratioTypes 字段，则认为是 type1
				isType1 := false
				for _, rt := range ratioTypes {
					if _, ok := type1Data[rt]; ok {
						isType1 = true
						break
					}
				}
				if isType1 {
					ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Data: type1Data}
					return
				}
			}

			// 如果不是 type1，则尝试按 type2 (/api/pricing) 解析
			var pricingItems []pricingItem
			if err := common.Unmarshal(body.Data, &pricingItems); err != nil {
				logger.LogWarn(c.Request.Context(), "unrecognized data format from "+chItem.Name+": "+err.Error())
				ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Err: "无法解析上游返回数据"}
				return
			}

			var converted map[string]any
			if req.SyncMode == "channel" {
				converted = convertChannelPricingItemsToRatioData(pricingItems, upstreamChannelIDForLocalChannel(chItem.ID))
			} else {
				converted = convertOfficialPricingItemsToRatioData(pricingItems)
			}

			ch <- upstreamResult{Name: uniqueName, ChannelID: chItem.ID, Data: converted}
		}(chn)
	}

	wg.Wait()
	close(ch)

	var testResults []dto.TestResult
	var successfulChannels []upstreamSyncSource

	for r := range ch {
		if r.Err != "" {
			testResults = append(testResults, dto.TestResult{
				Name:   r.Name,
				Status: "error",
				Error:  r.Err,
			})
		} else {
			testResults = append(testResults, dto.TestResult{
				Name:   r.Name,
				Status: "success",
			})
			successfulChannels = append(successfulChannels, upstreamSyncSource{
				name:        r.Name,
				channelID:   r.ChannelID,
				data:        r.Data,
				localModels: collectLocalChannelModels(r.ChannelID),
			})
		}
	}

	userID := c.GetInt("id")
	role := c.GetInt("role")
	includeAligned := true
	if req.IncludeAligned != nil {
		includeAligned = *req.IncludeAligned
	}
	var localData gin.H
	var differences map[string]map[string]dto.DifferenceItem
	// 非管理员：按已审核供应商身份，仅自有渠道模型参与与上游的差异对比
	if role < common.RoleAdminUser {
		app, err := model.GetApprovedSupplierApplicationByApplicant(userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "当前账号无供应商资质或未通过审核"})
			return
		}
		ownedRaw, err := collectSupplierOwnedModelNames(userID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		ownedNorm := model.NormalizeOwnedModelsForPricing(ownedRaw)
		localData = buildSupplierRatioSyncLocalMaps(app.ID, ownedNorm)
		differences = buildDifferences(localData, successfulChannels, includeAligned, app.ID, ownedNorm)
	} else {
		localData = ratio_setting.GetExposedData()
		differences = buildDifferences(localData, successfulChannels, includeAligned, 0, nil)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"differences":  differences,
			"test_results": testResults,
		},
	})
}

type upstreamSyncSource struct {
	name      string
	channelID int
	data      map[string]any
	// localModels 为本地渠道 channels.models 字段拆分并规范化后的模型集合（key 已 FormatMatchingModelName）。
	// 渠道同步时即使上游对某模型无定价，只要该模型属于本地渠道 models，也会被纳入对照（值为 nil）。
	localModels map[string]struct{}
}

// collectLocalChannelModels 读取本地渠道 channels.models 并拆分为规范化模型名集合。
// 渠道 ID 非正、模型为空或读取失败时返回 nil（让调用方按「未提供本地模型集合」处理）。
func collectLocalChannelModels(channelID int) map[string]struct{} {
	if channelID <= 0 {
		return nil
	}
	ch, err := model.GetChannelById(channelID, false)
	if err != nil || ch == nil {
		return nil
	}
	names := ch.GetModels()
	if len(names) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(names))
	for _, raw := range names {
		mn := strings.TrimSpace(raw)
		if mn == "" {
			continue
		}
		out[ratio_setting.FormatMatchingModelName(mn)] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// oldEffectiveForUpstream 返回该渠道列在同步前的生效价：正渠道 ID 只读取渠道覆盖，否则读取全局。
func oldEffectiveForUpstream(channelID int, ratioType string, modelName string, localData map[string]any) interface{} {
	if channelID > 0 {
		switch ratioType {
		case "model_ratio":
			if v, ok := ratio_setting.GetChannelModelRatio(channelID, modelName); ok {
				return oldChannelValueOrNil(v)
			}
		case "completion_ratio":
			if v, ok := ratio_setting.GetChannelCompletionRatio(channelID, modelName); ok {
				return oldChannelValueOrNil(v)
			}
		case "cache_ratio":
			if v, ok := ratio_setting.GetChannelCacheRatio(channelID, modelName); ok {
				return oldChannelValueOrNil(v)
			}
		case "create_cache_ratio":
			if v, ok := ratio_setting.GetChannelCreateCacheRatio(channelID, modelName); ok {
				return oldChannelValueOrNil(v)
			}
		case "model_price":
			if v, ok := ratio_setting.GetChannelModelPrice(channelID, modelName); ok {
				return oldChannelValueOrNil(v)
			}
		}
		return nil
	}
	if localRatioAny, ok := localData[ratioType]; ok {
		if val, exists := mapValueByModel(localRatioAny, modelName); exists {
			return val
		}
	}
	return nil
}

// buildSupplierRatioSyncLocalMaps 构建供应商全局视角下的本地倍率/价格映射（仅自有模型，值与 ResolveSupplierScoped* 一致）。
func buildSupplierRatioSyncLocalMaps(supplierApplicationID int, ownedNorm map[string]struct{}) gin.H {
	mr := make(map[string]float64)
	cr := make(map[string]float64)
	cache := make(map[string]float64)
	mp := make(map[string]float64)
	seen := make(map[string]struct{})
	ch0 := 0
	for name := range ownedNorm {
		mn := strings.TrimSpace(name)
		if mn == "" {
			continue
		}
		mn = ratio_setting.FormatMatchingModelName(mn)
		if _, ok := seen[mn]; ok {
			continue
		}
		seen[mn] = struct{}{}
		if v, ok := model.ResolveSupplierScopedFixedModelPrice(ch0, supplierApplicationID, mn); ok {
			mp[mn] = v
		}
		if vr, ok, _ := model.ResolveSupplierScopedModelRatio(ch0, supplierApplicationID, mn); ok {
			mr[mn] = vr
		}
		cr[mn] = model.ResolveSupplierScopedCompletionRatio(ch0, supplierApplicationID, mn)
		c0, _ := model.ResolveSupplierScopedCacheRatios(ch0, supplierApplicationID, mn)
		cache[mn] = c0
	}
	return gin.H{
		"model_ratio":             mr,
		"completion_ratio":        cr,
		"cache_ratio":             cache,
		"model_price":             mp,
		"model_tier_ratio":        map[string]ratio_setting.TierSegments{},
		"completion_tier_ratio":   map[string]ratio_setting.TierSegments{},
		"cache_tier_ratio":        map[string]ratio_setting.TierSegments{},
		"create_cache_tier_ratio": map[string]ratio_setting.TierSegments{},
	}
}

// modelNameOwnedForSupplierSync 判断差集模型名是否在供应商自有模型集合内（含规范化名）。
func modelNameOwnedForSupplierSync(modelName string, ownedNorm map[string]struct{}) bool {
	if len(ownedNorm) == 0 {
		return false
	}
	if _, ok := ownedNorm[modelName]; ok {
		return true
	}
	if _, ok := ownedNorm[ratio_setting.FormatMatchingModelName(modelName)]; ok {
		return true
	}
	return false
}

// oldEffectiveForUpstreamSupplier 供应商视角下，某上游列对应的「同步前」生效价。
func oldEffectiveForUpstreamSupplier(supplierApplicationID int, channelID int, ratioType string, modelName string, localData map[string]any) interface{} {
	mn := ratio_setting.FormatMatchingModelName(modelName)
	if channelID > 0 {
		var chRow *model.SupplierChannelModelPricing
		if supplierApplicationID > 0 {
			chRow, _ = model.GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, mn)
		}
		switch ratioType {
		case "model_ratio":
			if chRow != nil && chRow.ModelRatio != nil {
				return oldChannelValueOrNil(*chRow.ModelRatio)
			}
			if v, ok := ratio_setting.GetChannelModelRatio(channelID, mn); ok {
				return oldChannelValueOrNil(v)
			}
		case "completion_ratio":
			if chRow != nil && chRow.CompletionRatio != nil {
				return oldChannelValueOrNil(*chRow.CompletionRatio)
			}
			if v, ok := ratio_setting.GetChannelCompletionRatio(channelID, mn); ok {
				return oldChannelValueOrNil(v)
			}
		case "cache_ratio":
			if chRow != nil && chRow.CacheRatio != nil {
				return oldChannelValueOrNil(*chRow.CacheRatio)
			}
			if v, ok := ratio_setting.GetChannelCacheRatio(channelID, mn); ok {
				return oldChannelValueOrNil(v)
			}
		case "create_cache_ratio":
			if chRow != nil && chRow.CreateCacheRatio != nil {
				return oldChannelValueOrNil(*chRow.CreateCacheRatio)
			}
			if v, ok := ratio_setting.GetChannelCreateCacheRatio(channelID, mn); ok {
				return oldChannelValueOrNil(v)
			}
		case "model_price":
			if chRow != nil && chRow.ModelPrice != nil {
				return oldChannelValueOrNil(*chRow.ModelPrice)
			}
			if v, ok := ratio_setting.GetChannelModelPrice(channelID, mn); ok {
				return oldChannelValueOrNil(v)
			}
		}
		return nil
	}
	if localRatioAny, ok := localData[ratioType]; ok {
		if val, exists := mapValueByModel(localRatioAny, mn); exists {
			return val
		}
		if val, exists := mapValueByModel(localRatioAny, modelName); exists {
			return val
		}
	}
	return nil
}

func buildDifferences(localData map[string]any, successfulChannels []upstreamSyncSource, includeAligned bool, supplierAppID int, ownedNorm map[string]struct{}) map[string]map[string]dto.DifferenceItem {
	differences := make(map[string]map[string]dto.DifferenceItem)

	successUpstreamNames := make(map[string]struct{}, len(successfulChannels))
	for _, src := range successfulChannels {
		successUpstreamNames[src.name] = struct{}{}
	}

	allModels := make(map[string]struct{})

	for _, ratioType := range ratioTypes {
		if localRatioAny, ok := localData[ratioType]; ok {
			mapModelNames(localRatioAny, allModels)
		}
	}

	for _, channel := range successfulChannels {
		for _, ratioType := range ratioTypes {
			if upstreamRatio, ok := channel.data[ratioType].(map[string]any); ok {
				for modelName := range upstreamRatio {
					allModels[modelName] = struct{}{}
				}
			}
		}
		// 本地渠道 models 字段中的模型也并入：用于「即使上游缺值也展示本地渠道全部模型」的对照视图。
		for modelName := range channel.localModels {
			allModels[modelName] = struct{}{}
		}
	}

	if supplierAppID > 0 {
		filtered := make(map[string]struct{})
		for m := range allModels {
			if modelNameOwnedForSupplierSync(m, ownedNorm) {
				filtered[m] = struct{}{}
			}
		}
		allModels = filtered
	}

	confidenceMap := make(map[string]map[string]bool)

	// 预处理阶段：检查pricing接口的可信度
	for _, channel := range successfulChannels {
		confidenceMap[channel.name] = make(map[string]bool)

		modelRatios, hasModelRatio := channel.data["model_ratio"].(map[string]any)
		completionRatios, hasCompletionRatio := channel.data["completion_ratio"].(map[string]any)

		if hasModelRatio && hasCompletionRatio {
			// 遍历所有模型，检查是否满足不可信条件
			for modelName := range allModels {
				// 默认为可信
				confidenceMap[channel.name][modelName] = true

				// 检查是否满足不可信条件：model_ratio为37.5且completion_ratio为1
				if modelRatioVal, ok := modelRatios[modelName]; ok {
					if completionRatioVal, ok := completionRatios[modelName]; ok {
						// 转换为float64进行比较
						if modelRatioFloat, ok := modelRatioVal.(float64); ok {
							if completionRatioFloat, ok := completionRatioVal.(float64); ok {
								if modelRatioFloat == 37.5 && completionRatioFloat == 1.0 {
									confidenceMap[channel.name][modelName] = false
								}
							}
						}
					}
				}
			}
		} else {
			// 如果不是从pricing接口获取的数据，则全部标记为可信
			for modelName := range allModels {
				confidenceMap[channel.name][modelName] = true
			}
		}
	}

	for modelName := range allModels {
		for _, ratioType := range ratioTypes {
			var localValue interface{} = nil
			if localRatioAny, ok := localData[ratioType]; ok {
				if val, exists := mapValueByModel(localRatioAny, modelName); exists {
					localValue = val
				}
			}

			upstreamValues := make(map[string]interface{})
			upstreamOldVals := make(map[string]interface{})
			confidenceValues := make(map[string]bool)
			hasUpstreamValue := false

			for _, channel := range successfulChannels {
				var oldEff interface{}
				if supplierAppID > 0 {
					oldEff = oldEffectiveForUpstreamSupplier(supplierAppID, channel.channelID, ratioType, modelName, localData)
				} else {
					oldEff = oldEffectiveForUpstream(channel.channelID, ratioType, modelName, localData)
				}

				gotUpstream := false
				if upstreamRatio, ok := channel.data[ratioType].(map[string]any); ok {
					if val, exists := upstreamRatio[modelName]; exists {
						hasUpstreamValue = true
						upstreamValues[channel.name] = val
						upstreamOldVals[channel.name] = oldEff
						gotUpstream = true
					}
				}
				// 上游对该模型无该 ratioType 值，但模型属于本地渠道 models：保留该列为 nil，并标记
				// hasUpstreamValue=true，使「本地渠道全部模型」也能进入对照视图（与上游一致或上游缺失皆展示）。
				if !gotUpstream {
					if _, ok := channel.localModels[modelName]; ok {
						hasUpstreamValue = true
						upstreamValues[channel.name] = nil
						upstreamOldVals[channel.name] = oldEff
					}
				}

				confidenceValues[channel.name] = confidenceMap[channel.name][modelName]
			}

			if !hasUpstreamValue {
				continue
			}

			if differences[modelName] == nil {
				differences[modelName] = make(map[string]dto.DifferenceItem)
			}
			differences[modelName][ratioType] = dto.DifferenceItem{
				Current:     localValue,
				UpstreamOld: upstreamOldVals,
				Upstreams:   upstreamValues,
				Confidence:  confidenceValues,
			}
		}
	}

	channelHasDiff := make(map[string]bool)
	for _, ratioMap := range differences {
		for _, item := range ratioMap {
			for chName, newV := range item.Upstreams {
				if newV == nil {
					continue
				}
				oldV, _ := item.UpstreamOld[chName]
				if !valuesEqual(oldV, newV) {
					channelHasDiff[chName] = true
				}
			}
		}
	}

	for modelName, ratioMap := range differences {
		for ratioType, item := range ratioMap {
			for chName := range item.Upstreams {
				if !channelHasDiff[chName] {
					if includeAligned {
						if _, ok := successUpstreamNames[chName]; ok {
							continue
						}
					}
					delete(item.Upstreams, chName)
					delete(item.Confidence, chName)
					if item.UpstreamOld != nil {
						delete(item.UpstreamOld, chName)
					}
				}
			}

			allAligned := true
			for chName, newV := range item.Upstreams {
				if newV == nil {
					continue
				}
				oldV, _ := item.UpstreamOld[chName]
				if !valuesEqual(oldV, newV) {
					allAligned = false
					break
				}
			}
			if len(item.Upstreams) == 0 || (allAligned && !includeAligned) {
				delete(ratioMap, ratioType)
			} else {
				differences[modelName][ratioType] = item
			}
		}

		if len(ratioMap) == 0 {
			delete(differences, modelName)
		}
	}

	return differences
}

func roundRatioValue(value float64) float64 {
	return math.Round(value*1e6) / 1e6
}

func isModelsDevAPIEndpoint(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if strings.ToLower(parsedURL.Hostname()) != modelsDevHost {
		return false
	}
	path := strings.TrimSuffix(parsedURL.Path, "/")
	if path == "" {
		path = "/"
	}
	return path == modelsDevPath
}

func isTLSBadRecordMACError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "tls: bad record mac")
}

// convertOpenRouterToRatioData parses OpenRouter's /v1/models response and converts
// per-token USD pricing into the local ratio format.
// model_ratio = prompt_price_per_token * 1_000_000 * (USD / 1000)
//
//	since 1 ratio unit = $0.002/1K tokens and USD=500, the factor is 500_000
//
// completion_ratio = completion_price / prompt_price (output/input multiplier)
func convertOpenRouterToRatioData(reader io.Reader) (map[string]any, error) {
	var orResp struct {
		Data []struct {
			ID      string `json:"id"`
			Pricing struct {
				Prompt         string `json:"prompt"`
				Completion     string `json:"completion"`
				InputCacheRead string `json:"input_cache_read"`
			} `json:"pricing"`
		} `json:"data"`
	}

	if err := common.DecodeJson(reader, &orResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenRouter response: %w", err)
	}

	modelRatioMap := make(map[string]any)
	completionRatioMap := make(map[string]any)
	cacheRatioMap := make(map[string]any)

	for _, m := range orResp.Data {
		promptPrice, promptErr := strconv.ParseFloat(m.Pricing.Prompt, 64)
		completionPrice, compErr := strconv.ParseFloat(m.Pricing.Completion, 64)

		if promptErr != nil && compErr != nil {
			// Both unparseable — skip this model
			continue
		}

		// Treat parse errors as 0
		if promptErr != nil {
			promptPrice = 0
		}
		if compErr != nil {
			completionPrice = 0
		}

		// Negative values are sentinel values (e.g., -1 for dynamic/variable pricing) — skip
		if promptPrice < 0 || completionPrice < 0 {
			continue
		}

		if promptPrice == 0 && completionPrice == 0 {
			// Free model
			modelRatioMap[m.ID] = 0.0
			continue
		}
		if promptPrice <= 0 {
			// No meaningful prompt baseline, cannot derive ratios safely.
			continue
		}

		// Normal case: promptPrice > 0
		ratio := promptPrice * 1000 * ratio_setting.USD
		ratio = roundRatioValue(ratio)
		modelRatioMap[m.ID] = ratio

		compRatio := completionPrice / promptPrice
		compRatio = roundRatioValue(compRatio)
		completionRatioMap[m.ID] = compRatio

		// Convert input_cache_read to cache_ratio (= cache_read_price / prompt_price)
		if m.Pricing.InputCacheRead != "" {
			if cachePrice, err := strconv.ParseFloat(m.Pricing.InputCacheRead, 64); err == nil && cachePrice >= 0 {
				cacheRatio := cachePrice / promptPrice
				cacheRatio = roundRatioValue(cacheRatio)
				cacheRatioMap[m.ID] = cacheRatio
			}
		}
	}

	converted := make(map[string]any)
	if len(modelRatioMap) > 0 {
		converted["model_ratio"] = modelRatioMap
	}
	if len(completionRatioMap) > 0 {
		converted["completion_ratio"] = completionRatioMap
	}
	if len(cacheRatioMap) > 0 {
		converted["cache_ratio"] = cacheRatioMap
	}

	return converted, nil
}

type modelsDevProvider struct {
	ID     string                    `json:"id"`
	Name   string                    `json:"name"`
	API    string                    `json:"api"`
	Models map[string]modelsDevModel `json:"models"`
}

type modelsDevModel struct {
	Cost modelsDevCost `json:"cost"`
}

type modelsDevCost struct {
	Input     *float64 `json:"input"`
	Output    *float64 `json:"output"`
	CacheRead *float64 `json:"cache_read"`
}

type modelsDevCandidate struct {
	Provider  string
	Input     float64
	Output    *float64
	CacheRead *float64
}

var officialModelsDevProviderEndpointHosts = map[string]string{
	"anthropic":  "api.anthropic.com",
	"openai":     "api.openai.com",
	"google":     "generativelanguage.googleapis.com",
	"deepseek":   "api.deepseek.com",
	"xai":        "api.x.ai",
	"moonshotai": "api.moonshot.ai",
	// models.dev current id for Zhipu AI(BigModel)
	"zhipuai": "open.bigmodel.cn",
	// DashScope(OpenAI compatible endpoint)
	"alibaba": "dashscope-intl.aliyuncs.com",
}

// models.dev exposes Moonshot CN as a separate provider; only keep the global official endpoint.
var officialModelsDevProviderIDs = map[string]struct{}{
	"anthropic":  {},
	"openai":     {},
	"google":     {},
	"deepseek":   {},
	"xai":        {},
	"moonshotai": {},
	"zhipuai":    {},
	"alibaba":    {},
}

func resolveProviderAPIHost(providerID string, apiURL string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(apiURL))
	if err == nil && parsedURL.Hostname() != "" {
		return strings.ToLower(parsedURL.Hostname())
	}
	// For some official providers, models.dev omits `api`; use canonical official host fallback.
	return strings.ToLower(officialModelsDevProviderEndpointHosts[providerID])
}

func isOfficialModelsDevProvider(providerID string, provider modelsDevProvider) bool {
	if _, ok := officialModelsDevProviderIDs[providerID]; !ok {
		return false
	}
	expectedHost, ok := officialModelsDevProviderEndpointHosts[providerID]
	if !ok {
		return false
	}
	actualHost := resolveProviderAPIHost(providerID, provider.API)
	return actualHost == strings.ToLower(expectedHost)
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func isValidNonNegativeCost(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v >= 0
}

func buildModelsDevCandidate(provider string, cost modelsDevCost) (modelsDevCandidate, bool) {
	if cost.Input == nil {
		return modelsDevCandidate{}, false
	}

	input := *cost.Input
	if !isValidNonNegativeCost(input) {
		return modelsDevCandidate{}, false
	}

	var output *float64
	if cost.Output != nil {
		if !isValidNonNegativeCost(*cost.Output) {
			return modelsDevCandidate{}, false
		}
		output = cloneFloatPtr(cost.Output)
	}

	// input=0/output>0 cannot be transformed into local ratio.
	if input == 0 && output != nil && *output > 0 {
		return modelsDevCandidate{}, false
	}

	var cacheRead *float64
	if cost.CacheRead != nil && isValidNonNegativeCost(*cost.CacheRead) {
		cacheRead = cloneFloatPtr(cost.CacheRead)
	}

	return modelsDevCandidate{
		Provider:  provider,
		Input:     input,
		Output:    output,
		CacheRead: cacheRead,
	}, true
}

func shouldReplaceModelsDevCandidate(current, next modelsDevCandidate) bool {
	currentNonZero := current.Input > 0
	nextNonZero := next.Input > 0
	if currentNonZero != nextNonZero {
		// Prefer non-zero pricing data; this matches "cheapest non-zero" conflict policy.
		return nextNonZero
	}
	if nextNonZero && !nearlyEqual(next.Input, current.Input) {
		return next.Input < current.Input
	}
	// Stable tie-breaker for deterministic result.
	return next.Provider < current.Provider
}

// convertModelsDevToRatioData parses models.dev /api.json and converts
// provider pricing metadata into local ratio format.
// models.dev costs are USD per 1M tokens:
//
//	model_ratio = input_cost_per_1M / 2
//	completion_ratio = output_cost / input_cost
//	cache_ratio = cache_read_cost / input_cost
//
// Duplicate model keys across providers are resolved by selecting the
// cheapest non-zero input cost. If only zero-priced candidates exist,
// a zero ratio is kept.
func convertModelsDevToRatioData(reader io.Reader) (map[string]any, error) {
	var upstreamData map[string]modelsDevProvider
	if err := common.DecodeJson(reader, &upstreamData); err != nil {
		return nil, fmt.Errorf("failed to decode models.dev response: %w", err)
	}
	if len(upstreamData) == 0 {
		return nil, fmt.Errorf("empty models.dev response")
	}

	providers := make([]string, 0, len(upstreamData))
	for provider := range upstreamData {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	selectedCandidates := make(map[string]modelsDevCandidate)
	for _, provider := range providers {
		providerData := upstreamData[provider]
		providerID := strings.ToLower(strings.TrimSpace(providerData.ID))
		if providerID == "" {
			providerID = strings.ToLower(strings.TrimSpace(provider))
		}
		if !isOfficialModelsDevProvider(providerID, providerData) {
			continue
		}
		if len(providerData.Models) == 0 {
			continue
		}

		modelNames := make([]string, 0, len(providerData.Models))
		for modelName := range providerData.Models {
			modelNames = append(modelNames, modelName)
		}
		sort.Strings(modelNames)

		for _, modelName := range modelNames {
			candidate, ok := buildModelsDevCandidate(providerID, providerData.Models[modelName].Cost)
			if !ok {
				continue
			}
			current, exists := selectedCandidates[modelName]
			if !exists || shouldReplaceModelsDevCandidate(current, candidate) {
				selectedCandidates[modelName] = candidate
			}
		}
	}

	if len(selectedCandidates) == 0 {
		return nil, fmt.Errorf("no valid models.dev pricing entries found")
	}

	modelRatioMap := make(map[string]any)
	completionRatioMap := make(map[string]any)
	cacheRatioMap := make(map[string]any)

	for modelName, candidate := range selectedCandidates {
		if candidate.Input == 0 {
			modelRatioMap[modelName] = 0.0
			continue
		}

		modelRatio := candidate.Input * float64(ratio_setting.USD) / modelsDevInputCostRatioBase
		modelRatioMap[modelName] = roundRatioValue(modelRatio)

		if candidate.Output != nil {
			completionRatio := *candidate.Output / candidate.Input
			completionRatioMap[modelName] = roundRatioValue(completionRatio)
		}

		if candidate.CacheRead != nil {
			cacheRatio := *candidate.CacheRead / candidate.Input
			cacheRatioMap[modelName] = roundRatioValue(cacheRatio)
		}
	}

	converted := make(map[string]any)
	if len(modelRatioMap) > 0 {
		converted["model_ratio"] = modelRatioMap
	}
	if len(completionRatioMap) > 0 {
		converted["completion_ratio"] = completionRatioMap
	}
	if len(cacheRatioMap) > 0 {
		converted["cache_ratio"] = cacheRatioMap
	}
	return converted, nil
}

func GetSyncableChannels(c *gin.Context) {
	var syncableChannels []dto.SyncableChannel

	// 管理员可见全部渠道；已审核供应商仅可见自己归属渠道。
	if c.GetInt("role") >= common.RoleAdminUser {
		channels, err := model.GetAllChannels(0, 0, true, false)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		for _, channel := range channels {
			if channel.GetBaseURL() == "" {
				continue
			}
			syncableChannels = append(syncableChannels, dto.SyncableChannel{
				ID:      channel.Id,
				Name:    channel.Name,
				BaseURL: channel.GetBaseURL(),
				Status:  channel.Status,
				Type:    channel.Type,
			})
		}
	} else {
		ownerUserID := c.GetInt("id")
		ownedChannels, _, err := model.SearchSupplierChannels(&ownerUserID, 0, 100000, model.SupplierChannelSearchFilter{})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		for _, channel := range ownedChannels {
			if channel.GetBaseURL() == "" {
				continue
			}
			syncableChannels = append(syncableChannels, dto.SyncableChannel{
				ID:      channel.Id,
				Name:    channel.Name,
				BaseURL: channel.GetBaseURL(),
				Status:  channel.Status,
				Type:    channel.Type,
			})
		}
	}

	syncableChannels = append(syncableChannels, dto.SyncableChannel{
		ID:      officialRatioPresetID,
		Name:    officialRatioPresetName,
		BaseURL: officialRatioPresetBaseURL,
		Status:  1,
	})

	syncableChannels = append(syncableChannels, dto.SyncableChannel{
		ID:      modelsDevPresetID,
		Name:    modelsDevPresetName,
		BaseURL: modelsDevPresetBaseURL,
		Status:  1,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    syncableChannels,
	})
}
