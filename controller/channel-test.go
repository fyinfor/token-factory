package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	taskopenaivideo "github.com/QuantumNous/new-api/relay/channel/task/openaivideo"
	tasktencentvod "github.com/QuantumNous/new-api/relay/channel/task/tencentvod"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"

	"github.com/gin-gonic/gin"
)

// testResult 渠道测试一次调用的结果；recordedModelName 为与模型元数据/操练场 model_name 对齐的用户侧名称（非上游 UpstreamModelName），仅成功路径会填充。
type testResult struct {
	context           *gin.Context
	localErr          error
	tokenFactoryError *types.TokenFactoryError
	recordedModelName string
}

func normalizeChannelTestEndpoint(channel *model.Channel, modelName, endpointType string) string {
	normalized := strings.TrimSpace(endpointType)
	if normalized != "" {
		return normalized
	}
	if strings.HasSuffix(modelName, ratio_setting.CompactModelSuffix) {
		return string(constant.EndpointTypeOpenAIResponseCompact)
	}
	if channel != nil && channel.Type == constant.ChannelTypeCodex {
		return string(constant.EndpointTypeOpenAIResponse)
	}
	if channel != nil && channel.Type == constant.ChannelTypeOpenAIVideo {
		return string(constant.EndpointTypeOpenAIVideoGW)
	}
	if channel != nil && channel.Type == constant.ChannelTypeVideoGenerator {
		return string(constant.EndpointTypeVideoGenerator)
	}
	if channel != nil && channel.Type == constant.ChannelTypeTencentCloudVideo {
		return string(constant.EndpointTypeTencentCloudVODVideo)
	}
	if channel != nil && channel.Type == constant.ChannelTypeTencentCloudImage {
		return string(constant.EndpointTypeTencentCloudVODImage)
	}
	return normalized
}

func testChannel(channel *model.Channel, testModel string, endpointType string, isStream bool) testResult {
	tik := time.Now()
	var unsupportedTestChannelTypes = []int{
		constant.ChannelTypeMidjourney,
		constant.ChannelTypeMidjourneyPlus,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu,
	}
	if lo.Contains(unsupportedTestChannelTypes, channel.Type) {
		channelTypeName := constant.GetChannelTypeName(channel.Type)
		return testResult{
			localErr: fmt.Errorf("%s channel test is not supported", channelTypeName),
		}
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testModel = strings.TrimSpace(testModel)
	if testModel == "" {
		if channel.TestModel != nil && *channel.TestModel != "" {
			testModel = strings.TrimSpace(*channel.TestModel)
		} else {
			models := channel.GetModels()
			if len(models) > 0 {
				testModel = strings.TrimSpace(models[0])
			}
			if testModel == "" {
				testModel = "gpt-4o-mini"
			}
		}
	}

	endpointType = normalizeChannelTestEndpoint(channel, testModel, endpointType)

	requestPath := "/v1/chat/completions"

	// 如果指定了端点类型，使用指定的端点类型
	if endpointType != "" {
		if endpointInfo, ok := common.GetDefaultEndpointInfo(constant.EndpointType(endpointType)); ok {
			requestPath = endpointInfo.Path
		}
	} else {
		// 如果没有指定端点类型，使用原有的自动检测逻辑

		if strings.Contains(strings.ToLower(testModel), "rerank") {
			requestPath = "/v1/rerank"
		}

		// 先判断是否为 Embedding 模型
		if strings.Contains(strings.ToLower(testModel), "embedding") ||
			strings.HasPrefix(testModel, "m3e") || // m3e 系列模型
			strings.Contains(testModel, "bge-") || // bge 系列模型
			strings.Contains(testModel, "embed") ||
			channel.Type == constant.ChannelTypeMokaAI { // 其他 embedding 模型
			requestPath = "/v1/embeddings" // 修改请求路径
		}

		// VolcEngine 图像生成模型
		if channel.Type == constant.ChannelTypeVolcEngine && strings.Contains(testModel, "seedream") {
			requestPath = "/v1/images/generations"
		}

		// responses-only models
		if strings.Contains(strings.ToLower(testModel), "codex") {
			requestPath = "/v1/responses"
		}

		// responses compaction models (must use /v1/responses/compact)
		if strings.HasSuffix(testModel, ratio_setting.CompactModelSuffix) {
			requestPath = "/v1/responses/compact"
		}
	}
	if strings.HasPrefix(requestPath, "/v1/responses/compact") {
		testModel = ratio_setting.WithCompactModelSuffix(testModel)
	}

	c.Request = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: requestPath}, // 使用动态路径
		Body:   nil,
		Header: make(http.Header),
	}

	cache, err := model.GetUserCache(1)
	if err != nil {
		return testResult{
			localErr:          err,
			tokenFactoryError: nil,
		}
	}
	cache.WriteContext(c)

	//c.Request.Header.Set("Authorization", "Bearer "+channel.Key)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("channel", channel.Type)
	c.Set("base_url", channel.GetBaseURL())
	group, _ := model.GetUserGroup(1, false)
	c.Set("group", group)

	tokenFactoryError := middleware.SetupContextForSelectedChannel(c, channel, testModel)
	if tokenFactoryError != nil {
		return testResult{
			context:           c,
			localErr:          tokenFactoryError,
			tokenFactoryError: tokenFactoryError,
		}
	}

	// 视频生成端点走任务式异步上游协议（Sora /v1/videos、OpenAI 视频网关 /v1/videos/generations 等），
	// 与同步 chat/embeddings/image 走的 relay.GetAdaptor 流程不兼容，因此在这里直接旁路：
	// 仅校验上游能正确接收任务创建请求并返回 task_id，不做轮询。
	if endpointType == string(constant.EndpointTypeOpenAIVideo) ||
		endpointType == string(constant.EndpointTypeOpenAIVideoGW) ||
		endpointType == string(constant.EndpointTypeVideoGenerator) ||
		endpointType == string(constant.EndpointTypeTencentCloudVODVideo) {
		return testChannelVideo(c, channel, testModel, endpointType, tik)
	}

	// Determine relay format based on endpoint type or request path
	var relayFormat types.RelayFormat
	if endpointType != "" {
		// 根据指定的端点类型设置 relayFormat
		switch constant.EndpointType(endpointType) {
		case constant.EndpointTypeOpenAI:
			relayFormat = types.RelayFormatOpenAI
		case constant.EndpointTypeOpenAIResponse:
			relayFormat = types.RelayFormatOpenAIResponses
		case constant.EndpointTypeOpenAIResponseCompact:
			relayFormat = types.RelayFormatOpenAIResponsesCompaction
		case constant.EndpointTypeAnthropic:
			relayFormat = types.RelayFormatClaude
		case constant.EndpointTypeGemini:
			relayFormat = types.RelayFormatGemini
		case constant.EndpointTypeJinaRerank:
			relayFormat = types.RelayFormatRerank
		case constant.EndpointTypeImageGeneration:
			relayFormat = types.RelayFormatOpenAIImage
		case constant.EndpointTypeTencentCloudVODImage:
			relayFormat = types.RelayFormatOpenAIImage
		case constant.EndpointTypeEmbeddings:
			relayFormat = types.RelayFormatEmbedding
		default:
			relayFormat = types.RelayFormatOpenAI
		}
	} else {
		// 根据请求路径自动检测
		relayFormat = types.RelayFormatOpenAI
		if c.Request.URL.Path == "/v1/embeddings" {
			relayFormat = types.RelayFormatEmbedding
		}
		if c.Request.URL.Path == "/v1/images/generations" {
			relayFormat = types.RelayFormatOpenAIImage
		}
		if c.Request.URL.Path == "/v1/messages" {
			relayFormat = types.RelayFormatClaude
		}
		if strings.Contains(c.Request.URL.Path, "/v1beta/models") {
			relayFormat = types.RelayFormatGemini
		}
		if c.Request.URL.Path == "/v1/rerank" || c.Request.URL.Path == "/rerank" {
			relayFormat = types.RelayFormatRerank
		}
		if c.Request.URL.Path == "/v1/responses" {
			relayFormat = types.RelayFormatOpenAIResponses
		}
		if strings.HasPrefix(c.Request.URL.Path, "/v1/responses/compact") {
			relayFormat = types.RelayFormatOpenAIResponsesCompaction
		}
	}

	request := buildTestRequest(testModel, endpointType, channel, isStream)

	info, err := relaycommon.GenRelayInfo(c, relayFormat, request, nil)

	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeGenRelayInfoFailed),
		}
	}

	info.IsChannelTest = true
	info.InitChannelMeta(c)

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeChannelModelMappedError),
		}
	}

	testModel = info.UpstreamModelName
	// 更新请求中的模型名称
	request.SetModelName(testModel)

	apiType, _ := common.ChannelType2APIType(channel.Type)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact &&
		apiType != constant.APITypeOpenAI &&
		apiType != constant.APITypeCodex {
		return testResult{
			context:           c,
			localErr:          fmt.Errorf("responses compaction test only supports openai/codex channels, got api type %d", apiType),
			tokenFactoryError: types.NewError(fmt.Errorf("unsupported api type: %d", apiType), types.ErrorCodeInvalidApiType),
		}
	}
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return testResult{
			context:           c,
			localErr:          fmt.Errorf("invalid api type: %d, adaptor is nil", apiType),
			tokenFactoryError: types.NewError(fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), types.ErrorCodeInvalidApiType),
		}
	}

	//// 创建一个用于日志的 info 副本，移除 ApiKey
	//logInfo := info
	//logInfo.ApiKey = ""
	common.SysLog(fmt.Sprintf("testing channel %d with model %s , info %+v ", channel.Id, testModel, info.ToString()))

	priceData, err := helper.ModelPriceHelper(c, info, 0, request.GetTokenCountMeta())
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeModelPriceError),
		}
	}

	adaptor.Init(info)

	var convertedRequest any
	// 根据 RelayMode 选择正确的转换函数
	switch info.RelayMode {
	case relayconstant.RelayModeEmbeddings:
		// Embedding 请求 - request 已经是正确的类型
		if embeddingReq, ok := request.(*dto.EmbeddingRequest); ok {
			convertedRequest, err = adaptor.ConvertEmbeddingRequest(c, info, *embeddingReq)
		} else {
			return testResult{
				context:           c,
				localErr:          errors.New("invalid embedding request type"),
				tokenFactoryError: types.NewError(errors.New("invalid embedding request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeImagesGenerations:
		// 图像生成请求 - request 已经是正确的类型
		if imageReq, ok := request.(*dto.ImageRequest); ok {
			convertedRequest, err = adaptor.ConvertImageRequest(c, info, *imageReq)
		} else {
			return testResult{
				context:           c,
				localErr:          errors.New("invalid image request type"),
				tokenFactoryError: types.NewError(errors.New("invalid image request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeRerank:
		// Rerank 请求 - request 已经是正确的类型
		if rerankReq, ok := request.(*dto.RerankRequest); ok {
			convertedRequest, err = adaptor.ConvertRerankRequest(c, info.RelayMode, *rerankReq)
		} else {
			return testResult{
				context:           c,
				localErr:          errors.New("invalid rerank request type"),
				tokenFactoryError: types.NewError(errors.New("invalid rerank request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeResponses:
		// Response 请求 - request 已经是正确的类型
		if responseReq, ok := request.(*dto.OpenAIResponsesRequest); ok {
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, *responseReq)
		} else {
			return testResult{
				context:           c,
				localErr:          errors.New("invalid response request type"),
				tokenFactoryError: types.NewError(errors.New("invalid response request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeResponsesCompact:
		// Response compaction request - convert to OpenAIResponsesRequest before adapting
		switch req := request.(type) {
		case *dto.OpenAIResponsesCompactionRequest:
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, dto.OpenAIResponsesRequest{
				Model:              req.Model,
				Input:              req.Input,
				Instructions:       req.Instructions,
				PreviousResponseID: req.PreviousResponseID,
			})
		case *dto.OpenAIResponsesRequest:
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, *req)
		default:
			return testResult{
				context:           c,
				localErr:          errors.New("invalid response compaction request type"),
				tokenFactoryError: types.NewError(errors.New("invalid response compaction request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	default:
		// Chat/Completion 等其他请求类型
		if generalReq, ok := request.(*dto.GeneralOpenAIRequest); ok {
			convertedRequest, err = adaptor.ConvertOpenAIRequest(c, info, generalReq)
		} else {
			return testResult{
				context:           c,
				localErr:          errors.New("invalid general request type"),
				tokenFactoryError: types.NewError(errors.New("invalid general request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	}

	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
		}
	}
	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeJsonMarshalFailed),
		}
	}

	//jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings)
	//if err != nil {
	//	return testResult{
	//		context:     c,
	//		localErr:    err,
	//		tokenFactoryError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
	//	}
	//}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
				return testResult{
					context:           c,
					localErr:          fixedErr,
					tokenFactoryError: relaycommon.TokenFactoryErrorFromParamOverride(fixedErr),
				}
			}
			return testResult{
				context:           c,
				localErr:          err,
				tokenFactoryError: types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid),
			}
		}
	}

	requestBody := bytes.NewBuffer(jsonData)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
		}
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			err := service.RelayErrorHandler(c.Request.Context(), httpResp, true)
			common.SysError(fmt.Sprintf(
				"channel test bad response: channel_id=%d name=%s type=%d model=%s endpoint_type=%s status=%d err=%v",
				channel.Id,
				channel.Name,
				channel.Type,
				testModel,
				endpointType,
				httpResp.StatusCode,
				err,
			))
			return testResult{
				context:           c,
				localErr:          err,
				tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError),
			}
		}
	}

	// 腾讯云图片模型测试：只校验是否成功提交任务（返回 TaskId），不等待任务完成与 URL 回填。
	// 这样可避免 DescribeTaskDetail/DescribeMediaInfos 带来的 30~40 秒测试时延。
	if endpointType == string(constant.EndpointTypeTencentCloudVODImage) {
		if httpResp == nil || httpResp.Body == nil {
			err := errors.New("empty upstream response")
			return testResult{
				context:           c,
				localErr:          err,
				tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
		raw, readErr := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if readErr != nil {
			return testResult{
				context:           c,
				localErr:          readErr,
				tokenFactoryError: types.NewOpenAIError(readErr, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
			}
		}
		taskID := strings.TrimSpace(gjson.GetBytes(raw, "Response.TaskId").String())
		if taskID == "" {
			errMsg := strings.TrimSpace(gjson.GetBytes(raw, "Response.Error.Message").String())
			if errMsg == "" {
				errMsg = strings.TrimSpace(gjson.GetBytes(raw, "Response.Error.Code").String())
			}
			if errMsg == "" {
				errMsg = fmt.Sprintf("submit succeeded but missing TaskId, body=%s", truncateForError(string(raw)))
			}
			err := errors.New(errMsg)
			return testResult{
				context:           c,
				localErr:          err,
				tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
		common.SysLog(fmt.Sprintf("tencent image test channel #%d accepted, task_id=%s", channel.Id, taskID))
		recordedName := strings.TrimSpace(info.OriginModelName)
		if recordedName == "" {
			recordedName = strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyOriginalModel))
		}
		return testResult{
			context:           c,
			localErr:          nil,
			tokenFactoryError: nil,
			recordedModelName: recordedName,
		}
	}

	usageA, respErr := adaptor.DoResponse(c, httpResp, info)
	if respErr != nil {
		return testResult{
			context:           c,
			localErr:          respErr,
			tokenFactoryError: respErr,
		}
	}
	usage, usageErr := coerceTestUsage(usageA, isStream, info.GetEstimatePromptTokens())
	if usageErr != nil {
		return testResult{
			context:           c,
			localErr:          usageErr,
			tokenFactoryError: types.NewOpenAIError(usageErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	result := w.Result()
	respBody, err := readTestResponseBody(result.Body, isStream)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
		}
	}
	if bodyErr := detectErrorFromTestResponseBody(respBody); bodyErr != nil {
		return testResult{
			context:           c,
			localErr:          bodyErr,
			tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	info.SetEstimatePromptTokens(usage.PromptTokens)

	quota := 0
	if !priceData.UsePrice {
		quota = usage.PromptTokens + int(math.Round(float64(usage.CompletionTokens)*priceData.CompletionRatio))
		quota = int(math.Round(float64(quota) * priceData.ModelRatio))
		if priceData.ModelRatio != 0 && quota <= 0 {
			quota = 1
		}
	} else {
		quota = int(priceData.ModelPrice * common.QuotaPerUnit)
	}
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	consumedTime := float64(milliseconds) / 1000.0
	other := service.GenerateTextOtherInfo(c, info, priceData.ModelRatio, priceData.GroupRatioInfo.GroupRatio, priceData.CompletionRatio,
		usage.PromptTokensDetails.CachedTokens, priceData.CacheRatio, priceData.ModelPrice, priceData.GroupRatioInfo.GroupSpecialRatio)
	model.RecordConsumeLog(c, 1, model.RecordConsumeLogParams{
		ChannelId:        channel.Id,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		ModelName:        info.OriginModelName,
		TokenName:        "模型测试",
		Quota:            quota,
		Content:          "模型测试",
		UseTimeSeconds:   int(consumedTime),
		IsStream:         info.IsStream,
		Group:            info.UsingGroup,
		Other:            other,
	})
	common.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))
	// 与 model_test_results、操练场 GetUserModels 的 model_name 一致，供 Upsert 使用
	recordedName := strings.TrimSpace(info.OriginModelName)
	if recordedName == "" {
		recordedName = strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyOriginalModel))
	}
	return testResult{
		context:           c,
		localErr:          nil,
		tokenFactoryError: nil,
		recordedModelName: recordedName,
	}
}

// truncateForError 把请求/响应内容截短到 800 字符以内，避免错误消息过长。
func truncateForError(s string) string {
	const maxLen = 800
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// testChannelVideo 处理视频生成类端点（OpenAI Sora /v1/videos、OpenAI 视频网关 /v1/videos/generations）的渠道测试。
// 视频生成是任务式异步接口，这里只验证上游能否正确创建任务（返回 task_id），不做轮询，
// 避免长时间阻塞和测试期间产生真实视频生成费用。
func testChannelVideo(c *gin.Context, channel *model.Channel, testModel string, endpointType string, tik time.Time) testResult {
	endpoint, ok := common.GetDefaultEndpointInfo(constant.EndpointType(endpointType))
	if !ok {
		err := fmt.Errorf("unsupported video endpoint type: %s", endpointType)
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeInvalidApiType),
		}
	}

	// 模型映射：保持与同步路径一致，使用渠道维度的 model_mapping 配置；支持链式映射，循环时退出。
	originModel := strings.TrimSpace(testModel)
	upstreamModel := originModel
	if mapping := strings.TrimSpace(c.GetString("model_mapping")); mapping != "" && mapping != "{}" {
		modelMap := make(map[string]string)
		if err := common.UnmarshalJsonStr(mapping, &modelMap); err == nil {
			current := upstreamModel
			visited := map[string]bool{current: true}
			for {
				next, exists := modelMap[current]
				if !exists || next == "" || next == current {
					break
				}
				if visited[next] {
					break
				}
				visited[next] = true
				current = next
			}
			upstreamModel = current
		}
	}

	apiKey := common.GetContextKeyString(c, constant.ContextKeyChannelKey)
	if apiKey == "" {
		apiKey = channel.Key
	}
	baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
	if baseURL == "" {
		err := fmt.Errorf("channel base_url is empty")
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeChannelBaseUrlEmpty),
		}
	}
	fullURL := baseURL + endpoint.Path

	var bodyMap map[string]any
	switch constant.EndpointType(endpointType) {
	case constant.EndpointTypeOpenAIVideo:
		// OpenAI Sora 风格：POST /v1/videos，body 字段参考 https://platform.openai.com/docs/api-reference/videos/create
		bodyMap = map[string]any{
			"model":   upstreamModel,
			"prompt":  "a cute cat dancing in a sunny garden",
			"size":    "720x1280",
			"seconds": "4",
		}
	case constant.EndpointTypeTencentCloudVODVideo:
		// 腾讯云官方 VOD 视频接口必须使用 TC3 签名和 X-TC-* 公共头，不能直接 Bearer 调上游。
		cred, credErr := tasktencentvod.ParseCredentials(apiKey)
		if credErr != nil {
			return testResult{
				context:           c,
				localErr:          credErr,
				tokenFactoryError: types.NewError(credErr, types.ErrorCodeChannelInvalidKey),
			}
		}
		modelName, modelVersion := tasktencentvod.SplitCombinedModel(upstreamModel)
		if strings.TrimSpace(modelName) == "" || strings.TrimSpace(modelVersion) == "" {
			invalidModelErr := fmt.Errorf("invalid tencent vod model %q, expected ModelName-ModelVersion", upstreamModel)
			return testResult{
				context:           c,
				localErr:          invalidModelErr,
				tokenFactoryError: types.NewError(invalidModelErr, types.ErrorCodeBadRequestBody),
			}
		}
		signedBody := map[string]any{
			"SubAppId":     cred.SubAppID,
			"ModelName":    modelName,
			"ModelVersion": modelVersion,
			"Prompt":       "a cute cat dancing in a sunny garden",
		}
		signedPayload, marshalErr := common.Marshal(signedBody)
		if marshalErr != nil {
			return testResult{
				context:           c,
				localErr:          marshalErr,
				tokenFactoryError: types.NewError(marshalErr, types.ErrorCodeJsonMarshalFailed),
			}
		}
		signedResp, reqErr := tasktencentvod.SignedPOSTJSON(strings.TrimSpace(channel.GetSetting().Proxy), baseURL, cred.Region, cred, "CreateAigcVideoTask", signedPayload)
		if reqErr != nil {
			return testResult{
				context:           c,
				localErr:          reqErr,
				tokenFactoryError: types.NewOpenAIError(reqErr, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
			}
		}
		defer func() { _ = signedResp.Body.Close() }()
		respBody, readErr := io.ReadAll(signedResp.Body)
		if readErr != nil {
			return testResult{
				context:           c,
				localErr:          readErr,
				tokenFactoryError: types.NewOpenAIError(readErr, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
			}
		}
		common.SysLog(fmt.Sprintf("video test channel #%d response: status=%d, body=%s", channel.Id, signedResp.StatusCode, string(respBody)))
		if signedResp.StatusCode != http.StatusOK {
			msg := detectErrorMessageFromJSONBytes(respBody)
			if msg == "" {
				msg = strings.TrimSpace(string(respBody))
			}
			if msg == "" {
				msg = fmt.Sprintf("upstream returned status %d", signedResp.StatusCode)
			}
			bodyErr := fmt.Errorf("status=%d, body=%s", signedResp.StatusCode, msg)
			return testResult{
				context:           c,
				localErr:          bodyErr,
				tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponse, http.StatusInternalServerError),
			}
		}
		taskID := strings.TrimSpace(gjson.GetBytes(respBody, "Response.TaskId").String())
		if taskID == "" {
			bodyErr := fmt.Errorf("upstream did not return task_id, body: %s", string(respBody))
			return testResult{
				context:           c,
				localErr:          bodyErr,
				tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
		common.SysLog(fmt.Sprintf("video test channel #%d ok, task_id=%s", channel.Id, taskID))
		return testResult{
			context:           c,
			localErr:          nil,
			tokenFactoryError: nil,
			recordedModelName: originModel,
		}
	case constant.EndpointTypeOpenAIVideoGW:
		// OpenAI 视频网关：根据 base URL 自动选 MaaS（Hidream 官方）或 ARK（ByteDance 兼容代理）。
		// 提交路径统一是 /v1/videos/generations，是否带 /api/maas/gw 前缀由用户在 base URL 中决定。
		// 两种协议的 body 结构其实一致（content 数组），仅模型字段名不同：MaaS 用 model_id，ARK 用 model。
		// 数字人等少数 MaaS 平铺字段模型（image+sound_file）需要在自定义参数里手动调整 body，
		// 测试入口只发主流 Seedance/Doubao 系列能通过校验的最小集合。
		modelKey := "model"
		if taskopenaivideo.DetectProtocol(baseURL) == taskopenaivideo.ProtocolMaaS {
			modelKey = "model_id"
		}
		bodyMap = map[string]any{
			modelKey: upstreamModel,
			"content": []map[string]any{
				{"type": "text", "text": "a cute cat dancing in a sunny garden  --duration 5"},
			},
		}
	case constant.EndpointTypeVideoGenerator:
		bodyMap = map[string]any{
			"model": upstreamModel,
			"content": []map[string]any{
				{"type": "text", "text": "a cute cat dancing in a sunny garden"},
			},
			"parameters": map[string]any{
				"duration":   5,
				"resolution": "720P",
				"ratio":      "16:9",
				"watermark":  false,
			},
		}
	default:
		err := fmt.Errorf("unsupported video endpoint type: %s", endpointType)
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeInvalidApiType),
		}
	}

	bodyBytes, err := common.Marshal(bodyMap)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewError(err, types.ErrorCodeJsonMarshalFailed),
		}
	}

	method := endpoint.Method
	if method == "" {
		method = http.MethodPost
	}
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), method, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
		}
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	common.SysLog(fmt.Sprintf(
		"video test channel #%d (%s) endpoint=%s url=%s model=%s -> upstream=%s, request body: %s",
		channel.Id, channel.Name, endpointType, fullURL, originModel, upstreamModel, string(bodyBytes),
	))

	client := service.GetHttpClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return testResult{
			context:           c,
			localErr:          err,
			tokenFactoryError: types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
		}
	}

	// 无论成功失败都打印完整响应，便于排查上游字段对不上的问题。
	common.SysLog(fmt.Sprintf(
		"video test channel #%d response: status=%d, body=%s",
		channel.Id, resp.StatusCode, string(respBody),
	))

	if resp.StatusCode != http.StatusOK {
		msg := detectErrorMessageFromJSONBytes(respBody)
		if msg == "" {
			msg = strings.TrimSpace(string(respBody))
		}
		if msg == "" {
			msg = fmt.Sprintf("upstream returned status %d", resp.StatusCode)
		}
		bodyErr := fmt.Errorf("status=%d, body=%s", resp.StatusCode, msg)
		return testResult{
			context:           c,
			localErr:          bodyErr,
			tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponse, http.StatusInternalServerError),
		}
	}

	var taskID string
	switch constant.EndpointType(endpointType) {
	case constant.EndpointTypeOpenAIVideo:
		// OpenAI Sora 风格：顶层 id（新接口）或 task_id（旧接口兼容）
		taskID = strings.TrimSpace(gjson.GetBytes(respBody, "id").String())
		if taskID == "" {
			taskID = strings.TrimSpace(gjson.GetBytes(respBody, "task_id").String())
		}
		if errMsg := strings.TrimSpace(gjson.GetBytes(respBody, "error.message").String()); errMsg != "" {
			bodyErr := fmt.Errorf("upstream error: %s, body: %s", errMsg, truncateForError(string(respBody)))
			return testResult{
				context:           c,
				localErr:          bodyErr,
				tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
	case constant.EndpointTypeOpenAIVideoGW:
		// OpenAI 视频网关：两种响应格式根据顶层字段自动判断。
		//   MaaS：{"code":0,"message":"","result":{"task_id":"..."}}
		//         失败时 code != 0，错误消息在 message/messasge 里。
		//   ARK： {"id":"cgt-...","model":"...","status":"queued",...}
		//         失败时 {"error":{"code":"...","message":"...","type":"..."}}
		if errMsg := strings.TrimSpace(gjson.GetBytes(respBody, "error.message").String()); errMsg != "" {
			// 把完整响应附在错误里返回给前端，方便用户直接看到上游对哪些字段不满。
			bodyErr := fmt.Errorf(
				"upstream error: %s | request: %s | response: %s",
				errMsg,
				truncateForError(string(bodyBytes)),
				truncateForError(string(respBody)),
			)
			return testResult{
				context:           c,
				localErr:          bodyErr,
				tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
		// MaaS：顶层 code 字段存在且 != 0 表示失败。注意 code=0 是合法的成功值，
		// 所以要先判断字段是否存在，再判断是否非零。
		if codeRes := gjson.GetBytes(respBody, "code"); codeRes.Exists() && codeRes.Int() != 0 {
			errMsg := strings.TrimSpace(gjson.GetBytes(respBody, "message").String())
			if errMsg == "" {
				errMsg = strings.TrimSpace(gjson.GetBytes(respBody, "messasge").String())
			}
			if errMsg == "" {
				errMsg = fmt.Sprintf("upstream returned code=%d", codeRes.Int())
			}
			bodyErr := fmt.Errorf(
				"upstream error: %s | request: %s | response: %s",
				errMsg,
				truncateForError(string(bodyBytes)),
				truncateForError(string(respBody)),
			)
			return testResult{
				context:           c,
				localErr:          bodyErr,
				tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
		// 优先取 ARK 风格的顶层 id，再取 MaaS 风格的 result.task_id，最后兜底顶层 task_id。
		taskID = strings.TrimSpace(gjson.GetBytes(respBody, "id").String())
		if taskID == "" {
			taskID = strings.TrimSpace(gjson.GetBytes(respBody, "result.task_id").String())
		}
		if taskID == "" {
			taskID = strings.TrimSpace(gjson.GetBytes(respBody, "task_id").String())
		}
	case constant.EndpointTypeVideoGenerator:
		if codeRes := gjson.GetBytes(respBody, "status"); codeRes.Exists() && codeRes.Int() != 0 {
			errMsg := strings.TrimSpace(gjson.GetBytes(respBody, "message").String())
			if errMsg == "" {
				errMsg = fmt.Sprintf("upstream returned status=%d", codeRes.Int())
			}
			bodyErr := fmt.Errorf(
				"upstream error: %s | request: %s | response: %s",
				errMsg,
				truncateForError(string(bodyBytes)),
				truncateForError(string(respBody)),
			)
			return testResult{
				context:           c,
				localErr:          bodyErr,
				tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
			}
		}
		taskID = strings.TrimSpace(gjson.GetBytes(respBody, "result.task_id").String())
		if taskID == "" {
			taskID = strings.TrimSpace(gjson.GetBytes(respBody, "task_id").String())
		}
	}

	if taskID == "" {
		bodyErr := fmt.Errorf("upstream did not return task_id, body: %s", string(respBody))
		return testResult{
			context:           c,
			localErr:          bodyErr,
			tokenFactoryError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}

	milliseconds := time.Since(tik).Milliseconds()
	common.SysLog(fmt.Sprintf("video test channel #%d ok, task_id=%s, took %dms, body: %s",
		channel.Id, taskID, milliseconds, string(respBody)))

	group, _ := model.GetUserGroup(1, false)
	model.RecordConsumeLog(c, 1, model.RecordConsumeLogParams{
		ChannelId:      channel.Id,
		ModelName:      originModel,
		TokenName:      "模型测试",
		Quota:          0,
		Content:        fmt.Sprintf("模型测试-视频生成(task_id=%s)", taskID),
		UseTimeSeconds: int(milliseconds / 1000),
		IsStream:       false,
		Group:          group,
	})

	return testResult{
		context:           c,
		localErr:          nil,
		tokenFactoryError: nil,
		recordedModelName: originModel,
	}
}

func coerceTestUsage(usageAny any, isStream bool, estimatePromptTokens int) (*dto.Usage, error) {
	switch u := usageAny.(type) {
	case *dto.Usage:
		return u, nil
	case dto.Usage:
		return &u, nil
	case nil:
		if !isStream {
			return nil, errors.New("usage is nil")
		}
		usage := &dto.Usage{
			PromptTokens: estimatePromptTokens,
		}
		usage.TotalTokens = usage.PromptTokens
		return usage, nil
	default:
		if !isStream {
			return nil, fmt.Errorf("invalid usage type: %T", usageAny)
		}
		usage := &dto.Usage{
			PromptTokens: estimatePromptTokens,
		}
		usage.TotalTokens = usage.PromptTokens
		return usage, nil
	}
}

func readTestResponseBody(body io.ReadCloser, isStream bool) ([]byte, error) {
	defer func() { _ = body.Close() }()
	const maxStreamLogBytes = 8 << 10
	if isStream {
		return io.ReadAll(io.LimitReader(body, maxStreamLogBytes))
	}
	return io.ReadAll(body)
}

func detectErrorFromTestResponseBody(respBody []byte) error {
	b := bytes.TrimSpace(respBody)
	if len(b) == 0 {
		return nil
	}
	if message := detectErrorMessageFromJSONBytes(b); message != "" {
		return fmt.Errorf("upstream error: %s", message)
	}

	for _, line := range bytes.Split(b, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}
		if message := detectErrorMessageFromJSONBytes(payload); message != "" {
			return fmt.Errorf("upstream error: %s", message)
		}
	}

	return nil
}

func detectErrorMessageFromJSONBytes(jsonBytes []byte) string {
	if len(jsonBytes) == 0 {
		return ""
	}
	if jsonBytes[0] != '{' && jsonBytes[0] != '[' {
		return ""
	}
	errVal := gjson.GetBytes(jsonBytes, "error")
	if !errVal.Exists() || errVal.Type == gjson.Null {
		return ""
	}

	message := gjson.GetBytes(jsonBytes, "error.message").String()
	if message == "" {
		message = gjson.GetBytes(jsonBytes, "error.error.message").String()
	}
	if message == "" && errVal.Type == gjson.String {
		message = errVal.String()
	}
	if message == "" {
		message = errVal.Raw
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return "upstream returned error payload"
	}
	return message
}

func buildTestRequest(model string, endpointType string, channel *model.Channel, isStream bool) dto.Request {
	testResponsesInput := json.RawMessage(`[{"role":"user","content":"hi"}]`)

	// 根据端点类型构建不同的测试请求
	if endpointType != "" {
		switch constant.EndpointType(endpointType) {
		case constant.EndpointTypeEmbeddings:
			// 返回 EmbeddingRequest
			return &dto.EmbeddingRequest{
				Model: model,
				Input: []any{"hello world"},
			}
		case constant.EndpointTypeImageGeneration, constant.EndpointTypeTencentCloudVODImage:
			// 返回 ImageRequest
			return &dto.ImageRequest{
				Model:  model,
				Prompt: "a cute cat",
				N:      lo.ToPtr(uint(1)),
				Size:   "1024x1024",
			}
		case constant.EndpointTypeJinaRerank:
			// 返回 RerankRequest
			return &dto.RerankRequest{
				Model:     model,
				Query:     "What is Deep Learning?",
				Documents: []any{"Deep Learning is a subset of machine learning.", "Machine learning is a field of artificial intelligence."},
				TopN:      lo.ToPtr(2),
			}
		case constant.EndpointTypeOpenAIResponse:
			// 返回 OpenAIResponsesRequest
			return &dto.OpenAIResponsesRequest{
				Model:  model,
				Input:  json.RawMessage(`[{"role":"user","content":"hi"}]`),
				Stream: lo.ToPtr(isStream),
			}
		case constant.EndpointTypeOpenAIResponseCompact:
			// 返回 OpenAIResponsesCompactionRequest
			return &dto.OpenAIResponsesCompactionRequest{
				Model: model,
				Input: testResponsesInput,
			}
		case constant.EndpointTypeAnthropic, constant.EndpointTypeGemini, constant.EndpointTypeOpenAI:
			// 返回 GeneralOpenAIRequest
			maxTokens := uint(16)
			if constant.EndpointType(endpointType) == constant.EndpointTypeGemini {
				maxTokens = 3000
			}
			req := &dto.GeneralOpenAIRequest{
				Model:  model,
				Stream: lo.ToPtr(isStream),
				Messages: []dto.Message{
					{
						Role:    "user",
						Content: "hi",
					},
				},
				MaxTokens: lo.ToPtr(maxTokens),
			}
			if isStream {
				req.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
			}
			return req
		}
	}

	// 自动检测逻辑（保持原有行为）
	if strings.Contains(strings.ToLower(model), "rerank") {
		return &dto.RerankRequest{
			Model:     model,
			Query:     "What is Deep Learning?",
			Documents: []any{"Deep Learning is a subset of machine learning.", "Machine learning is a field of artificial intelligence."},
			TopN:      lo.ToPtr(2),
		}
	}

	// 先判断是否为 Embedding 模型
	if strings.Contains(strings.ToLower(model), "embedding") ||
		strings.HasPrefix(model, "m3e") ||
		strings.Contains(model, "bge-") {
		// 返回 EmbeddingRequest
		return &dto.EmbeddingRequest{
			Model: model,
			Input: []any{"hello world"},
		}
	}

	// Responses compaction models (must use /v1/responses/compact)
	if strings.HasSuffix(model, ratio_setting.CompactModelSuffix) {
		return &dto.OpenAIResponsesCompactionRequest{
			Model: model,
			Input: testResponsesInput,
		}
	}

	// Responses-only models (e.g. codex series)
	if strings.Contains(strings.ToLower(model), "codex") {
		return &dto.OpenAIResponsesRequest{
			Model:  model,
			Input:  json.RawMessage(`[{"role":"user","content":"hi"}]`),
			Stream: lo.ToPtr(isStream),
		}
	}

	// Chat/Completion 请求 - 返回 GeneralOpenAIRequest
	testRequest := &dto.GeneralOpenAIRequest{
		Model:  model,
		Stream: lo.ToPtr(isStream),
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hi",
			},
		},
	}
	if isStream {
		testRequest.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}

	if strings.HasPrefix(model, "o") {
		testRequest.MaxCompletionTokens = lo.ToPtr(uint(16))
	} else if strings.Contains(model, "thinking") {
		if !strings.Contains(model, "claude") {
			testRequest.MaxTokens = lo.ToPtr(uint(50))
		}
	} else if strings.Contains(model, "gemini") {
		testRequest.MaxTokens = lo.ToPtr(uint(3000))
	} else {
		testRequest.MaxTokens = lo.ToPtr(uint(16))
	}

	return testRequest
}

func TestChannel(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		channel, err = model.GetChannelById(channelId, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	// 供应商仅允许测试自己归属的渠道。
	if c.GetInt("role") < common.RoleAdminUser && channel.OwnerUserID != c.GetInt("id") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无权测试其他供应商渠道",
		})
		return
	}
	//defer func() {
	//	if channel.ChannelInfo.IsMultiKey {
	//		go func() { _ = channel.SaveChannelInfo() }()
	//	}
	//}()
	testModel := c.Query("model")
	endpointType := c.Query("endpoint_type")
	isStream, _ := strconv.ParseBool(c.Query("stream"))
	tik := time.Now()
	result := testChannel(channel, testModel, endpointType, isStream)
	milliseconds := time.Since(tik).Milliseconds()
	consumedTime := float64(milliseconds) / 1000.0
	// 优先用 relay 解析后的用户侧名（与 models.model_name 一致）；否则与 testChannel 入参/渠道默认试测模型保持同一套兜底
	modelForRecord := strings.TrimSpace(result.recordedModelName)
	if modelForRecord == "" {
		modelForRecord = strings.TrimSpace(testModel)
		if modelForRecord == "" {
			if channel.TestModel != nil && strings.TrimSpace(*channel.TestModel) != "" {
				modelForRecord = strings.TrimSpace(*channel.TestModel)
			} else {
				models := channel.GetModels()
				if len(models) > 0 {
					modelForRecord = strings.TrimSpace(models[0])
				}
			}
		}
	}
	if result.localErr != nil {
		go channel.UpdateTestResult(false, milliseconds, result.localErr.Error(), modelForRecord)
		go func() {
			_ = model.UpsertModelTestResult(channel.Id, modelForRecord, false, milliseconds, result.localErr.Error())
		}()
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": result.localErr.Error(),
			"time":    consumedTime,
		})
		return
	}
	if result.tokenFactoryError != nil {
		go channel.UpdateTestResult(false, milliseconds, result.tokenFactoryError.Error(), modelForRecord)
		go func() {
			_ = model.UpsertModelTestResult(channel.Id, modelForRecord, false, milliseconds, result.tokenFactoryError.Error())
		}()
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": result.tokenFactoryError.Error(),
			"time":    consumedTime,
		})
		return
	}
	go channel.UpdateTestResult(true, milliseconds, "", modelForRecord)
	go func() { _ = model.UpsertModelTestResult(channel.Id, modelForRecord, true, milliseconds, "") }()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"time":    consumedTime,
	})
}

var testAllChannelsLock sync.Mutex
var testAllChannelsRunning bool = false

func testAllChannels(notify bool) error {

	testAllChannelsLock.Lock()
	if testAllChannelsRunning {
		testAllChannelsLock.Unlock()
		return errors.New("测试已在运行中")
	}
	testAllChannelsRunning = true
	testAllChannelsLock.Unlock()
	channels, getChannelErr := model.GetAllChannels(0, 0, true, false)
	if getChannelErr != nil {
		return getChannelErr
	}
	var disableThreshold = int64(common.ChannelDisableThreshold * 1000)
	if disableThreshold == 0 {
		disableThreshold = 10000000 // a impossible value
	}
	gopool.Go(func() {
		// 使用 defer 确保无论如何都会重置运行状态，防止死锁
		defer func() {
			testAllChannelsLock.Lock()
			testAllChannelsRunning = false
			testAllChannelsLock.Unlock()
		}()

		for _, channel := range channels {
			if channel.Status == common.ChannelStatusManuallyDisabled {
				continue
			}
			isChannelEnabled := channel.Status == common.ChannelStatusEnabled
			tik := time.Now()
			result := testChannel(channel, "", "", false)
			tok := time.Now()
			milliseconds := tok.Sub(tik).Milliseconds()
			testMessage := ""
			if result.localErr != nil {
				testMessage = result.localErr.Error()
			} else if result.tokenFactoryError != nil {
				testMessage = result.tokenFactoryError.Error()
			}
			testSuccess := result.localErr == nil && result.tokenFactoryError == nil
			channel.UpdateTestResult(testSuccess, milliseconds, testMessage, "")
			modelForRecord := ""
			if channel.TestModel != nil && strings.TrimSpace(*channel.TestModel) != "" {
				modelForRecord = strings.TrimSpace(*channel.TestModel)
			} else {
				models := channel.GetModels()
				if len(models) > 0 {
					modelForRecord = strings.TrimSpace(models[0])
				}
			}
			_ = model.UpsertModelTestResult(channel.Id, modelForRecord, testSuccess, milliseconds, testMessage)

			shouldBanChannel := false
			tokenFactoryError := result.tokenFactoryError
			// request error disables the channel
			if tokenFactoryError != nil {
				shouldBanChannel = service.ShouldDisableChannel(channel.Type, result.tokenFactoryError)
			}

			// 当错误检查通过，才检查响应时间
			if common.AutomaticDisableChannelEnabled && !shouldBanChannel {
				if milliseconds > disableThreshold {
					err := fmt.Errorf("响应时间 %.2fs 超过阈值 %.2fs", float64(milliseconds)/1000.0, float64(disableThreshold)/1000.0)
					tokenFactoryError = types.NewOpenAIError(err, types.ErrorCodeChannelResponseTimeExceeded, http.StatusRequestTimeout)
					shouldBanChannel = true
				}
			}

			// disable channel
			if isChannelEnabled && shouldBanChannel && channel.GetAutoBan() {
				processChannelError(result.context, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(result.context, constant.ContextKeyChannelKey), channel.GetAutoBan()), tokenFactoryError)
			}

			// enable channel
			if !isChannelEnabled && service.ShouldEnableChannel(tokenFactoryError, channel.Status) {
				service.EnableChannel(channel.Id, common.GetContextKeyString(result.context, constant.ContextKeyChannelKey), channel.Name)
			}

			time.Sleep(common.RequestInterval)
		}

		if notify {
			service.NotifyRootUser(dto.NotifyTypeChannelTest, "通道测试完成", "所有通道测试已完成")
		}
	})
	return nil
}

func TestAllChannels(c *gin.Context) {
	err := testAllChannels(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

var autoTestChannelsOnce sync.Once

func AutomaticallyTestChannels() {
	// 只在Master节点定时测试渠道
	if !common.IsMasterNode {
		return
	}
	autoTestChannelsOnce.Do(func() {
		for {
			if !operation_setting.GetMonitorSetting().AutoTestChannelEnabled {
				time.Sleep(1 * time.Minute)
				continue
			}
			for {
				frequency := operation_setting.GetMonitorSetting().AutoTestChannelMinutes
				time.Sleep(time.Duration(int(math.Round(frequency))) * time.Minute)
				common.SysLog(fmt.Sprintf("automatically test channels with interval %f minutes", frequency))
				common.SysLog("automatically testing all channels")
				_ = testAllChannels(false)
				common.SysLog("automatically channel test finished")
				if !operation_setting.GetMonitorSetting().AutoTestChannelEnabled {
					break
				}
			}
		}
	})
}
