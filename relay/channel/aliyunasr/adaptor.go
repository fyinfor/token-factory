package aliyunasr

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// Adaptor 阿里云百炼（DashScope）ASR 语音转写适配器。
// 对外暴露 OpenAI 兼容接口，内部完成 OpenAI 请求格式 ↔ DashScope 协议互转。
type Adaptor struct {
	ChannelType    int
	responseFormat string
	// audioSeconds 同步链路本地解析的音频时长（秒），按秒计费的核心依据；
	// 仅在上传文件时可本地解析，URL 模式下为 0，由上游 usage.duration 折算。
	audioSeconds float64
	audioFormat  string // 上游 parameters.format（必填）
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	base := strings.TrimSuffix(info.ChannelBaseUrl, "/")
	if base == "" {
		return "", errors.New("阿里云 ASR 渠道未配置上游基础地址（Base URL），请在渠道配置中填写，例如 https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api")
	}
	switch info.RelayMode {
	case relayconstant.RelayModeAudioTranscription:
		return base + SyncGenerationPath, nil
	case relayconstant.RelayModeAudioTranscriptionAsyncSubmit:
		return base + AsyncSubmitPath, nil
	default:
		return "", fmt.Errorf("ali asr adaptor: unsupported relay mode %d", info.RelayMode)
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	if info.RelayMode == relayconstant.RelayModeAudioTranscriptionAsyncSubmit {
		// DashScope 异步任务提交必需请求头
		req.Set("X-DashScope-Async", "enable")
	}
	if info.RelayMode == relayconstant.RelayModeAudioTranscription {
		// 同步非流式：显式关闭 SSE，与官方示例一致
		req.Set("X-DashScope-SSE", "disable")
	}
	// 网关到上游一律使用 JSON，与客户端提交的 multipart 无关
	req.Set("Content-Type", "application/json")
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

// ============================== 请求转换（OpenAI → DashScope） ==============================

// ConvertAudioRequest 同步转写请求转换。
// 支持两种输入：
//  1. multipart/form-data：file 字段上传音频文件（本地解析时长），或 audio_url/file_url 表单字段提供音频地址；
//  2. application/json：{"model": "...", "audio_url": "https://..."}。
func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	a.responseFormat = request.ResponseFormat
	var audioSource string
	var filename string

	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, fmt.Errorf("解析 multipart 表单失败: %w", err)
		}
		// 优先使用表单中的音频 URL，否则读取上传文件并转 base64 data-uri
		audioSource = firstFormValue(form, "audio_url", "file_url", "url")
		if audioSource == "" {
			fileHeaders := form.File["file"]
			if len(fileHeaders) == 0 {
				return nil, errors.New("请上传音频文件（file 字段）或通过 audio_url 提供音频地址")
			}
			src, err := readAudioFile(fileHeaders[0])
			if err != nil {
				return nil, err
			}
			a.audioSeconds = src.seconds
			audioSource = src.dataURI
			filename = fileHeaders[0].Filename
		} else {
			filename = firstFormValue(form, "filename")
		}
	} else {
		var jsonReq struct {
			AudioURL string `json:"audio_url"`
			FileURL  string `json:"file_url"`
			URL      string `json:"url"`
			Format   string `json:"format"`
		}
		if err := common.UnmarshalBodyReusable(c, &jsonReq); err != nil {
			return nil, fmt.Errorf("解析请求体失败: %w", err)
		}
		audioSource = jsonReq.AudioURL
		if audioSource == "" {
			audioSource = jsonReq.FileURL
		}
		if audioSource == "" {
			audioSource = jsonReq.URL
		}
		if audioSource == "" {
			return nil, errors.New("JSON 请求需提供 audio_url 音频地址（或使用 multipart 上传 file 文件）")
		}
		if f := strings.TrimSpace(jsonReq.Format); f != "" {
			a.audioFormat = strings.ToLower(f)
		}
	}

	if a.audioFormat == "" {
		a.audioFormat = InferAudioFormat(audioSource, filename)
	}

	modelName := strings.TrimSpace(request.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(info.UpstreamModelName)
	}
	reqBody := buildSyncRequest(modelName, audioSource, a.audioFormat)
	jsonBytes, err := common.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构造上游请求失败: %w", err)
	}
	return bytes.NewReader(jsonBytes), nil
}

// buildSyncRequest 按模型族构造 DashScope multimodal-generation 同步转写请求体。
//
// 两套协议不可混用：
//   - Fun-ASR-Flash / Qwen-Audio：type=input_audio + input_audio.data + format/sample_rate
//   - Qwen3-ASR-Flash：content 使用 {"audio": url}（无 type；MultiModalItem 不接受 input_audio）
//     + parameters.result_format/asr_options
func buildSyncRequest(model, audioSource, format string) *aliASRSyncRequest {
	reqBody := &aliASRSyncRequest{Model: model}
	if UsesFunASRFlashSyncProtocol(model) {
		if format == "" {
			format = InferAudioFormat(audioSource, "")
		}
		reqBody.Parameters = &aliASRSyncParameters{
			Format:     format,
			SampleRate: "16000",
		}
		reqBody.Input.Messages = []aliASRMessage{
			{
				Role: "user",
				Content: []aliASRContentItem{
					{
						Type: "input_audio",
						InputAudio: &aliASRInputAudio{
							Data: audioSource,
						},
					},
				},
			},
		}
		return reqBody
	}

	// Qwen3-ASR-Flash（DashScope 原生 multimodal）：{"audio": url}，不要带 type=input_audio
	reqBody.Parameters = &aliASRSyncParameters{
		ResultFormat: "message",
		AsrOptions:   &aliASROptions{},
	}
	reqBody.Input.Messages = []aliASRMessage{
		{
			Role: "user",
			Content: []aliASRContentItem{
				{Audio: audioSource},
			},
		},
	}
	return reqBody
}

func firstFormValue(form *multipart.Form, keys ...string) string {
	for _, key := range keys {
		if values, ok := form.Value[key]; ok && len(values) > 0 {
			if v := strings.TrimSpace(values[0]); v != "" {
				return v
			}
		}
	}
	return ""
}

type audioFileSource struct {
	dataURI string
	seconds float64
}

// readAudioFile 读取上传的音频文件：校验上游 10MB 限制、本地解析时长、转 base64 data-uri。
func readAudioFile(fileHeader *multipart.FileHeader) (*audioFileSource, error) {
	if fileHeader.Size > maxSyncAudioFileSize {
		return nil, fmt.Errorf("音频文件 %.1fMB 超过同步转写 10MB 上游限制，请改用异步转写接口", float64(fileHeader.Size)/1024/1024)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开音频文件失败: %w", err)
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(io.LimitReader(file, maxSyncAudioFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取音频文件失败: %w", err)
	}
	if len(fileBytes) > maxSyncAudioFileSize {
		return nil, errors.New("音频文件超过同步转写 10MB 上游限制，请改用异步转写接口")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	seconds, err := common.GetAudioDuration(context.TODO(), bytes.NewReader(fileBytes), ext)
	if err != nil {
		// 时长解析失败不阻断请求：后续 DoResponse 会用上游 usage.duration 折算秒数
		common.SysLog("aliyunasr: get audio duration failed: " + err.Error())
	}
	mime := AudioMIMEFromExt(ext)
	return &audioFileSource{
		dataURI: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(fileBytes),
		seconds: seconds,
	}, nil
}

// ============================== 响应转换（DashScope → OpenAI） ==============================

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, tokenFactoryError *types.TokenFactoryError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var aliResp aliASRSyncResponse
	if err := common.Unmarshal(responseBody, &aliResp); err != nil {
		return nil, types.NewOpenAIError(fmt.Errorf("解析上游响应失败: %w, body: %s", err, string(responseBody)), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	// DashScope 部分错误以 HTTP 200 + code/message 返回
	if aliResp.Code != "" {
		return nil, types.NewOpenAIError(
			fmt.Errorf("上游错误 [%s]: %s (request_id: %s)", aliResp.Code, aliResp.Message, aliResp.RequestID),
			types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}

	text := aliResp.Output.ResolveText()
	if text == "" {
		return nil, types.NewOpenAIError(
			fmt.Errorf("上游未返回识别文本 (request_id: %s, body: %s)", aliResp.RequestID, string(responseBody)),
			types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}

	usageDto := &dto.Usage{}
	if aliResp.Usage != nil {
		usageDto.CompletionTokens = aliResp.Usage.OutputTokens
		if aliResp.Usage.InputTokensDetails != nil {
			usageDto.PromptTokensDetails.AudioTokens = aliResp.Usage.InputTokensDetails.AudioTokens
		}
	}

	// 按秒计费：优先本地解析时长 → 上游 usage.duration/seconds → audio_tokens 折算 → 兜底 1 秒
	seconds := a.audioSeconds
	if seconds <= 0 {
		seconds = aliResp.Usage.AudioSeconds()
	}
	if seconds <= 0 && usageDto.PromptTokensDetails.AudioTokens > 0 {
		seconds = float64(usageDto.PromptTokensDetails.AudioTokens) / audioTokensPerSecond
	}
	if seconds <= 0 && aliResp.Output.Sentence != nil && aliResp.Output.Sentence.EndTime > 0 {
		seconds = float64(aliResp.Output.Sentence.EndTime) / 1000.0
	}
	if seconds <= 0 {
		seconds = 1
	}
	info.PriceData.AddOtherRatio("seconds", seconds)
	// 用量口径：PromptTokens 携带计费秒数（calculateTextQuotaSummary 要求 TotalTokens>0 才产生费用）
	usageDto.PromptTokens = int(math.Ceil(seconds))
	usageDto.TotalTokens = usageDto.PromptTokens + usageDto.CompletionTokens

	if a.responseFormat == "text" {
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.WriteHeader(resp.StatusCode)
		_, _ = c.Writer.Write([]byte(text))
		return usageDto, nil
	}

	// 同步转写 JSON 响应始终返回音频时长（秒），便于客户端展示与对账
	returnInfo := map[string]any{
		"text":     text,
		"duration": seconds,
	}
	if a.responseFormat == "verbose_json" {
		returnInfo["task"] = "transcribe"
	}
	jsonResponse, err := common.Marshal(returnInfo)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)
	return usageDto, nil
}

// ============================== 渠道元信息 ==============================

func (a *Adaptor) GetModelList() []string {
	if a.ChannelType == constant.ChannelTypeAliASRAsync {
		return AsyncModelList
	}
	return SyncModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

// ============================== 其余接口（ASR 渠道不支持，直接报错） ==============================

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support chat completions")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support rerank")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support embedding")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support image")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support responses")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support claude")
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("ali asr channel does not support gemini")
}
