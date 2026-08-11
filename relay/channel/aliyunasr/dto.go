package aliyunasr

import (
	"encoding/json"
	"strings"
)

// ============================== 同步转写（multimodal-generation） ==============================

// aliASRInputAudio 同步转写音频载荷：公网可访问的音频 URL（multipart file 会先上传附件库再填入）。
type aliASRInputAudio struct {
	Data string `json:"data"`
}

// aliASRContentItem 多模态消息内容项。
// Fun-ASR / Qwen-Audio：type=input_audio + input_audio.data
// Qwen3-ASR：{"audio": url}（不要带 type=input_audio，上游 MultiModalItem 会拒绝）
// Text 仅用于兼容解析部分旧响应。
type aliASRContentItem struct {
	Type       string            `json:"type,omitempty"`
	Text       string            `json:"text,omitempty"`
	InputAudio *aliASRInputAudio `json:"input_audio,omitempty"`
	Audio      string            `json:"audio,omitempty"`
}

type aliASRMessage struct {
	Role    string              `json:"role"`
	Content []aliASRContentItem `json:"content"`
}

type aliASROptions struct {
	Language  string `json:"language,omitempty"`
	EnableITN *bool  `json:"enable_itn,omitempty"`
}

// aliASRSyncParameters 同步转写 parameters。
// Fun-ASR-Flash / Qwen-Audio：format（必填）+ sample_rate；
// Qwen3-ASR-Flash：result_format + asr_options。
type aliASRSyncParameters struct {
	Format         string         `json:"format,omitempty"`
	SampleRate     string         `json:"sample_rate,omitempty"`
	ResultFormat   string         `json:"result_format,omitempty"`
	AsrOptions     *aliASROptions `json:"asr_options,omitempty"`
	VocabularyID   string         `json:"vocabulary_id,omitempty"`
	Vocabulary     map[string]int `json:"vocabulary,omitempty"`
	LanguageHints  []string       `json:"language_hints,omitempty"`
}

// aliASRSyncRequest POST /v1/services/aigc/multimodal-generation/generation 请求体。
type aliASRSyncRequest struct {
	Model      string `json:"model"`
	Input      struct {
		Messages []aliASRMessage `json:"messages"`
	} `json:"input"`
	Parameters *aliASRSyncParameters `json:"parameters,omitempty"`
}

// aliASRTokenDetails 上游 usage 的细分 token（text/audio），部分旧模型仍可能返回。
type aliASRTokenDetails struct {
	TextTokens  int `json:"text_tokens"`
	AudioTokens int `json:"audio_tokens"`
}

// aliASRUsage 同步/异步共用的用量结构。
// Fun-ASR / Qwen-Audio 系列以 duration（秒）计费；部分旧模型返回 token 字段。
type aliASRUsage struct {
	Duration            int                `json:"duration,omitempty"` // 音频时长（秒）
	Seconds             float64            `json:"seconds,omitempty"`  // Qwen3 filetrans 部分响应使用
	InputTokens         int                `json:"input_tokens,omitempty"`
	OutputTokens        int                `json:"output_tokens,omitempty"`
	TotalTokens         int                `json:"total_tokens,omitempty"`
	InputTokensDetails  *aliASRTokenDetails `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *aliASRTokenDetails `json:"output_tokens_details,omitempty"`
}

// AudioSeconds 取计费时长（秒），优先 duration，其次 seconds。
func (u *aliASRUsage) AudioSeconds() float64 {
	if u == nil {
		return 0
	}
	if u.Duration > 0 {
		return float64(u.Duration)
	}
	if u.Seconds > 0 {
		return u.Seconds
	}
	return 0
}

// aliASRSyncSentence 同步转写返回的句子详情。
type aliASRSyncSentence struct {
	BeginTime   int64  `json:"begin_time"`
	EndTime     int64  `json:"end_time"`
	Text        string `json:"text"`
	SentenceID  int    `json:"sentence_id,omitempty"`
	SentenceEnd bool   `json:"sentence_end,omitempty"`
	ChannelID   int    `json:"channel_id,omitempty"`
}

// aliASRSyncOutput 同步转写 output。
// Fun-ASR-Flash / Qwen-Audio-3.0-ASR-Flash 返回 text + sentence，无 choices。
// 旧版 multimodal 可能仍返回 choices，解析时做兼容。
type aliASRSyncOutput struct {
	Text     string              `json:"text,omitempty"`
	Sentence *aliASRSyncSentence `json:"sentence,omitempty"`
	Choices  []aliASRChoice      `json:"choices,omitempty"`
}

// ResolveText 提取识别文本：优先 output.text，其次 sentence.text，最后兼容 choices。
func (o aliASRSyncOutput) ResolveText() string {
	if t := strings.TrimSpace(o.Text); t != "" {
		return t
	}
	if o.Sentence != nil {
		if t := strings.TrimSpace(o.Sentence.Text); t != "" {
			return t
		}
	}
	var b strings.Builder
	for _, choice := range o.Choices {
		b.WriteString(choice.Message.Content.Text)
	}
	return strings.TrimSpace(b.String())
}

// aliASRMessageContent 响应消息 content：部分模型返回字符串，部分返回 [{"text": "..."}] 数组。
type aliASRMessageContent struct {
	Text string
}

func (m *aliASRMessageContent) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Text = s
		return nil
	}
	var items []aliASRContentItem
	if err := json.Unmarshal(data, &items); err == nil {
		for _, item := range items {
			m.Text += item.Text
		}
		return nil
	}
	return nil
}

type aliASRResponseMessage struct {
	Role    string               `json:"role"`
	Content aliASRMessageContent `json:"content"`
}

type aliASRChoice struct {
	FinishReason string                `json:"finish_reason"`
	Message      aliASRResponseMessage `json:"message"`
}

// aliASRSyncResponse POST multimodal-generation 响应体。
type aliASRSyncResponse struct {
	RequestID string          `json:"request_id"`
	Output    aliASRSyncOutput `json:"output"`
	Usage     *aliASRUsage    `json:"usage,omitempty"`
	Code      string          `json:"code,omitempty"`
	Message   string          `json:"message,omitempty"`
}

// ============================== 异步转写（filetrans 任务） ==============================

// aliASRAsyncInput 异步任务 input。
// Qwen-Audio-3.0 / Fun-ASR 使用 file_urls（数组）；Qwen3-ASR-Flash-Filetrans 使用 file_url（单值）。
type aliASRAsyncInput struct {
	FileURL  string   `json:"file_url,omitempty"`
	FileURLs []string `json:"file_urls,omitempty"`
}

type aliASRAsyncParameters struct {
	ChannelID            []int    `json:"channel_id,omitempty"`
	LanguageHints        []string `json:"language_hints,omitempty"`
	DiarizationEnabled   *bool    `json:"diarization_enabled,omitempty"`
}

// aliASRAsyncSubmitRequest POST /v1/services/audio/asr/transcription 请求体。
type aliASRAsyncSubmitRequest struct {
	Model      string                 `json:"model"`
	Input      aliASRAsyncInput       `json:"input"`
	Parameters *aliASRAsyncParameters `json:"parameters,omitempty"`
}

// ASRTaskResultItem 异步任务 results[] / result 单项。
type ASRTaskResultItem struct {
	FileURL          string `json:"file_url,omitempty"`
	SubtaskStatus    string `json:"subtask_status,omitempty"`
	TranscriptionURL string `json:"transcription_url,omitempty"`
	Code             string `json:"code,omitempty"`
	Message          string `json:"message,omitempty"`
}

// ASRTaskOutput 任务状态输出（提交/查询通用）。
type ASRTaskOutput struct {
	TaskID           string              `json:"task_id"`
	TaskStatus       string              `json:"task_status"` // PENDING / RUNNING / SUCCEEDED / FAILED / CANCELED
	TranscriptionURL string              `json:"transcription_url,omitempty"`
	FileURL          string              `json:"file_url,omitempty"`
	SubtaskStatus    string              `json:"subtask_status,omitempty"`
	Results          []ASRTaskResultItem `json:"results,omitempty"`
	Result           *ASRTaskResultItem  `json:"result,omitempty"` // Qwen3 filetrans 单对象形态
	Code             string              `json:"code,omitempty"`
	Message          string              `json:"message,omitempty"`
	SubmitTime       string              `json:"submit_time,omitempty"`
	ScheduledTime    string              `json:"scheduled_time,omitempty"`
	EndTime          string              `json:"end_time,omitempty"`
}

// ResolveTranscriptionURL 从多种上游响应形态中提取 transcription_url。
func (o ASRTaskOutput) ResolveTranscriptionURL() string {
	if u := strings.TrimSpace(o.TranscriptionURL); u != "" {
		return u
	}
	if o.Result != nil {
		if u := strings.TrimSpace(o.Result.TranscriptionURL); u != "" {
			return u
		}
	}
	for _, item := range o.Results {
		if u := strings.TrimSpace(item.TranscriptionURL); u != "" {
			return u
		}
	}
	return ""
}

// FailReason 提取失败原因（顶层 code/message，或 results 内失败项）。
func (o ASRTaskOutput) FailReason() string {
	code := strings.TrimSpace(o.Code)
	msg := strings.TrimSpace(o.Message)
	if code != "" || msg != "" {
		if code == "" {
			return msg
		}
		if msg == "" || msg == code {
			return "[" + code + "] " + code
		}
		return "[" + code + "] " + msg
	}
	for _, item := range o.Results {
		c := strings.TrimSpace(item.Code)
		m := strings.TrimSpace(item.Message)
		if c != "" || m != "" {
			if c == "" {
				return m
			}
			if m == "" || m == c {
				return "[" + c + "] " + c
			}
			return "[" + c + "] " + m
		}
	}
	if o.Result != nil {
		c := strings.TrimSpace(o.Result.Code)
		m := strings.TrimSpace(o.Result.Message)
		if c != "" || m != "" {
			if c == "" {
				return m
			}
			if m == "" || m == c {
				return "[" + c + "] " + c
			}
			return "[" + c + "] " + m
		}
	}
	return ""
}

// ASRTaskResponse 异步提交与任务查询的响应体（结构一致）。
type ASRTaskResponse struct {
	RequestID string        `json:"request_id"`
	Output    ASRTaskOutput `json:"output"`
	Usage     *aliASRUsage  `json:"usage,omitempty"`
	Code      string        `json:"code,omitempty"`
	Message   string        `json:"message,omitempty"`
}

// DashScope 异步任务状态枚举
const (
	AliASRTaskStatusPending   = "PENDING"
	AliASRTaskStatusRunning   = "RUNNING"
	AliASRTaskStatusSucceeded = "SUCCEEDED"
	AliASRTaskStatusFailed    = "FAILED"
	AliASRTaskStatusCanceled  = "CANCELED"
)

// ============================== 异步结果文件（transcription_url 下载内容） ==============================

type aliASRSentence struct {
	BeginTime int64  `json:"begin_time"` // 毫秒
	EndTime   int64  `json:"end_time"`   // 毫秒
	Text      string `json:"text"`
	SpeakerID *int   `json:"speaker_id,omitempty"`
}

type aliASRTranscript struct {
	ChannelID                     int              `json:"channel_id"`
	ContentDurationInMilliseconds int64            `json:"content_duration_in_milliseconds"`
	Text                          string           `json:"text"`
	Sentences                     []aliASRSentence `json:"sentences,omitempty"`
}

// aliASRTranscriptionResult transcription_url 指向的识别结果 JSON 文件。
type aliASRTranscriptionResult struct {
	FileURL     string             `json:"file_url"`
	Transcripts []aliASRTranscript `json:"transcripts"`
	// Properties 部分上游结果文件会带原始时长信息
	Properties *struct {
		OriginalDurationInMilliseconds int64 `json:"original_duration_in_milliseconds"`
	} `json:"properties,omitempty"`
}
