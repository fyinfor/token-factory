package dto

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type AudioRequest struct {
	Model string `json:"model"`
	// Input TTS 为文本字符串；同步 ASR 透传上游原生 JSON（如 DashScope）时可为对象，故用 RawMessage。
	Input          json.RawMessage `json:"input,omitempty"`
	Voice          string          `json:"voice"`
	Instructions   string          `json:"instructions,omitempty"`
	ResponseFormat string          `json:"response_format,omitempty"`
	Speed          *float64        `json:"speed,omitempty"`
	StreamFormat   string          `json:"stream_format,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	// AudioURL / FileURL 同步 ASR OpenAI 兼容 JSON 字段（multipart 表单同名字段由 adaptor 另行读取）。
	AudioURL string `json:"audio_url,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
}

// GetInputText 返回 TTS 文本输入；input 为对象（透传上游原生体）时返回空字符串。
func (r *AudioRequest) GetInputText() string {
	if r == nil || len(r.Input) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(r.Input, &s); err == nil {
		return s
	}
	return ""
}

func (r *AudioRequest) GetTokenCountMeta() *types.TokenCountMeta {
	meta := &types.TokenCountMeta{
		CombineText: r.GetInputText(),
		TokenType:   types.TokenTypeTextNumber,
	}
	if strings.Contains(r.Model, "gpt") {
		meta.TokenType = types.TokenTypeTokenizer
	}
	return meta
}

func (r *AudioRequest) IsStream(c *gin.Context) bool {
	return r.StreamFormat == "sse"
}

func (r *AudioRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

type AudioResponse struct {
	Text     string  `json:"text"`
	Duration float64 `json:"duration,omitempty"` // 音频时长（秒）
}

type WhisperVerboseJSONResponse struct {
	Task     string    `json:"task,omitempty"`
	Language string    `json:"language,omitempty"`
	Duration float64   `json:"duration,omitempty"`
	Text     string    `json:"text,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
}

type Segment struct {
	Id               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// ============================== ASR 异步转录（OpenAI 兼容） ==============================

// ASRTaskSubmitRequest POST /v1/audio/transcriptions/async 提交异步转录任务请求体。
// 异步链路（filetrans）上游要求公网可访问的音频 URL：
// 可直接传 audio_url/file_url，或 multipart file（网关先上传到操练场附件库再取在线地址）。
// 支持 multipart 表单字段（model/audio_url/file）或 JSON body。
type ASRTaskSubmitRequest struct {
	Model          string `json:"model"`
	AudioURL       string `json:"audio_url"`
	FileURL        string `json:"file_url"`
	Language       string `json:"language,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

func (r *ASRTaskSubmitRequest) GetAudioURL() string {
	if r.AudioURL != "" {
		return r.AudioURL
	}
	return r.FileURL
}

func (r *ASRTaskSubmitRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

func (r *ASRTaskSubmitRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *ASRTaskSubmitRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTextNumber,
	}
}

// ASR 异步任务对外状态（映射上游 PENDING/RUNNING/SUCCEEDED/FAILED/CANCELED）
const (
	ASRTaskStatusPending   = "pending"
	ASRTaskStatusRunning   = "running"
	ASRTaskStatusSucceeded = "succeeded"
	ASRTaskStatusFailed    = "failed"
)

// ASRTaskSubmitResponse POST /v1/audio/transcriptions/async 提交响应。
type ASRTaskSubmitResponse struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	Model     string `json:"model"`
	CreatedAt int64  `json:"created_at"`
}

// ASRTaskFetchResponse GET /v1/audio/transcriptions/async/{task_id} 查询响应。
// 任务成功后 text/duration 有值；失败时 error 有值。
type ASRTaskFetchResponse struct {
	TaskID     string  `json:"task_id"`
	Status     string  `json:"status"`
	Model      string  `json:"model"`
	Text       string  `json:"text,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
	Error      string  `json:"error,omitempty"`
	CreatedAt  int64   `json:"created_at"`
	FinishedAt int64   `json:"finished_at,omitempty"`
}
