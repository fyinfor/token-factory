package aliyunasr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
)

// UpstreamHTTPError 上游 HTTP 非 2xx 响应（含 DashScope code/message）。
type UpstreamHTTPError struct {
	StatusCode int
	Code       string
	Message    string
	Body       []byte
}

func (e *UpstreamHTTPError) Error() string {
	if e == nil {
		return "upstream http error"
	}
	return fmt.Sprintf("上游返回状态码 %d: [%s] %s", e.StatusCode, e.Code, e.Message)
}

// IsPermanentUpstreamHTTPError 判断是否应终止轮询并标记任务失败（鉴权/未开通/资源不存在等）。
// 429 与 5xx 视为可重试，保持 pending/running。
func IsPermanentUpstreamHTTPError(err error) bool {
	var u *UpstreamHTTPError
	if !errors.As(err, &u) || u == nil {
		return false
	}
	if u.StatusCode == http.StatusTooManyRequests || u.StatusCode >= 500 {
		return false
	}
	switch u.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return true
	}
	if u.StatusCode >= 400 && u.StatusCode < 500 {
		code := strings.ToLower(u.Code)
		msg := strings.ToLower(u.Message)
		if strings.Contains(code, "accessdenied") ||
			strings.Contains(code, "unpurchased") ||
			strings.Contains(code, "invalid") ||
			strings.Contains(msg, "access denied") ||
			strings.Contains(msg, "unpurchased") {
			return true
		}
	}
	return false
}

// 异步任务相关 HTTP 超时设置
const (
	asyncSubmitTimeout        = 30 * time.Second
	asyncFetchTimeout         = 30 * time.Second
	transcriptionFetchTimeout = 60 * time.Second
)

// SubmitAsyncTask 提交异步转写任务：POST {baseURL}/v1/services/audio/asr/transcription。
// DashScope 异步模式必须携带 X-DashScope-Async: enable 请求头。
// baseURL 为渠道配置的上游基础地址（如 https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api）。
func SubmitAsyncTask(baseURL, apiKey, proxy, model, fileURL string, parameters *aliASRAsyncParameters) (*ASRTaskResponse, []byte, error) {
	url := strings.TrimSuffix(baseURL, "/") + AsyncSubmitPath
	asyncParams := &aliASRAsyncParameters{ChannelID: []int{0}}
	if parameters != nil {
		if parameters.DiarizationEnabled != nil {
			asyncParams.DiarizationEnabled = parameters.DiarizationEnabled
		}
		if len(parameters.LanguageHints) > 0 {
			asyncParams.LanguageHints = parameters.LanguageHints
		}
		if len(parameters.ChannelID) > 0 {
			asyncParams.ChannelID = parameters.ChannelID
		}
	}
	reqBody := &aliASRAsyncSubmitRequest{
		Model:      model,
		Input:      BuildAsyncInput(model, fileURL),
		Parameters: asyncParams,
	}
	jsonBytes, err := common.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("构造异步任务请求失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), asyncSubmitTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-DashScope-Async", "enable")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, nil, fmt.Errorf("创建代理 HTTP 客户端失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取上游响应失败: %w", err)
	}

	var taskResp ASRTaskResponse
	if err := common.Unmarshal(respBody, &taskResp); err != nil {
		return nil, respBody, fmt.Errorf("解析上游响应失败: %w, body: %s", err, string(respBody))
	}
	if resp.StatusCode != http.StatusOK {
		return &taskResp, respBody, &UpstreamHTTPError{
			StatusCode: resp.StatusCode,
			Code:       taskResp.Code,
			Message:    taskResp.Message,
			Body:       respBody,
		}
	}
	if taskResp.Output.TaskID == "" {
		return &taskResp, respBody, fmt.Errorf("上游未返回 task_id: %s", string(respBody))
	}
	return &taskResp, respBody, nil
}

// FetchAsyncTask 查询异步任务状态：GET {baseURL}/v1/tasks/{task_id}。
func FetchAsyncTask(baseURL, apiKey, proxy, upstreamTaskID string) (*ASRTaskResponse, []byte, error) {
	url := strings.TrimSuffix(baseURL, "/") + AsyncTaskQueryPathPrefix + "/" + upstreamTaskID

	ctx, cancel := context.WithTimeout(context.Background(), asyncFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, nil, fmt.Errorf("创建代理 HTTP 客户端失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取上游响应失败: %w", err)
	}

	var taskResp ASRTaskResponse
	if err := common.Unmarshal(respBody, &taskResp); err != nil {
		return nil, respBody, fmt.Errorf("解析上游响应失败: %w, body: %s", err, string(respBody))
	}
	if resp.StatusCode != http.StatusOK {
		return &taskResp, respBody, &UpstreamHTTPError{
			StatusCode: resp.StatusCode,
			Code:       taskResp.Code,
			Message:    taskResp.Message,
			Body:       respBody,
		}
	}
	return &taskResp, respBody, nil
}

// DownloadTranscriptionResult 下载并解析 transcription_url 指向的识别结果 JSON 文件。
// 结果文件由上游临时托管（存在时效），网关拉取成功后应缓存到本地任务记录，
// 避免上游文件过期导致任务结果永久丢失。
func DownloadTranscriptionResult(transcriptionURL, proxy string) (*aliASRTranscriptionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), transcriptionFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, transcriptionURL, nil)
	if err != nil {
		return nil, err
	}

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("创建代理 HTTP 客户端失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载识别结果文件失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载识别结果文件返回状态码 %d（文件可能已过期）", resp.StatusCode)
	}
	// 结果文件上限保护：12 小时音频逐句结果通常在几十 MB 以内
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 128<<20))
	if err != nil {
		return nil, fmt.Errorf("读取识别结果文件失败: %w", err)
	}

	var result aliASRTranscriptionResult
	if err := common.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析识别结果文件失败: %w", err)
	}
	return &result, nil
}

// MergeTranscriptsText 合并多音轨识别文本，并取最大音轨时长作为计费时长（秒）。
func MergeTranscriptsText(result *aliASRTranscriptionResult) (text string, seconds float64) {
	if result == nil {
		return "", 0
	}
	texts := make([]string, 0, len(result.Transcripts))
	var maxDurationMs int64
	for _, t := range result.Transcripts {
		if strings.TrimSpace(t.Text) != "" {
			texts = append(texts, t.Text)
		}
		if t.ContentDurationInMilliseconds > maxDurationMs {
			maxDurationMs = t.ContentDurationInMilliseconds
		}
	}
	if maxDurationMs <= 0 && result.Properties != nil {
		maxDurationMs = result.Properties.OriginalDurationInMilliseconds
	}
	return strings.Join(texts, "\n"), float64(maxDurationMs) / 1000.0
}

// BuildAsyncSubmitParameters 构造异步提交 upstream parameters。
func BuildAsyncSubmitParameters(diarizationEnabled *bool) *aliASRAsyncParameters {
	params := &aliASRAsyncParameters{ChannelID: []int{0}}
	if diarizationEnabled != nil {
		params.DiarizationEnabled = diarizationEnabled
	}
	return params
}

// BuildUserTranscripts 将上游识别结果转为对外 transcripts 结构（含说话人分离 speaker_id）。
func BuildUserTranscripts(result *aliASRTranscriptionResult) []dto.ASRTranscript {
	if result == nil || len(result.Transcripts) == 0 {
		return nil
	}
	out := make([]dto.ASRTranscript, 0, len(result.Transcripts))
	for _, tr := range result.Transcripts {
		if len(tr.Sentences) == 0 {
			continue
		}
		sentences := make([]dto.ASRTranscriptSentence, 0, len(tr.Sentences))
		for _, s := range tr.Sentences {
			sentences = append(sentences, dto.ASRTranscriptSentence{
				BeginTime: s.BeginTime,
				EndTime:   s.EndTime,
				Text:      s.Text,
				SpeakerID: s.SpeakerID,
			})
		}
		out = append(out, dto.ASRTranscript{Sentences: sentences})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
