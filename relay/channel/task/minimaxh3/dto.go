package minimaxh3

// VideoGenerationV2Req 对应官方 VideoGenerationV2Req。
// duration 为必填整数；aigc_watermark 使用指针以保留显式 false。
type VideoGenerationV2Req struct {
	Model         string        `json:"model"`
	Content       []ContentItem `json:"content"`
	Resolution    string        `json:"resolution"`
	Duration      int           `json:"duration"`
	Ratio         string        `json:"ratio,omitempty"`
	CallbackURL   string        `json:"callback_url,omitempty"`
	AigcWatermark *bool         `json:"aigc_watermark,omitempty"`
}

// ContentItem 多模态输入项（text / image_url / video_url / audio_url）。
type ContentItem struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

// MediaURL 官方 { "url": "..." } 对象，支持公网 URL、mm_file://、data URI。
type MediaURL struct {
	URL string `json:"url"`
}

// VideoGenerationV2Resp 创建任务成功响应。
type VideoGenerationV2Resp struct {
	TaskID string `json:"task_id"`
	OaiErrorEnvelope
}

// QueryTaskResponse 查询任务响应。
type QueryTaskResponse struct {
	Task VideoTask `json:"task"`
	OaiErrorEnvelope
}

// VideoTask 官方 VideoTask。
type VideoTask struct {
	ID         string            `json:"id"`
	Model      string            `json:"model"`
	Status     string            `json:"status"`
	Error      *VideoTaskError   `json:"error,omitempty"`
	CreatedAt  int64             `json:"created_at,omitempty"`
	UpdatedAt  int64             `json:"updated_at,omitempty"`
	Content    *VideoTaskContent `json:"content,omitempty"`
	Resolution string            `json:"resolution,omitempty"`
	Duration   int               `json:"duration,omitempty"`
	Usage      *VideoTaskUsage   `json:"usage,omitempty"`
	Ratio      string            `json:"ratio,omitempty"`
	TaskType   string            `json:"task_type,omitempty"`
	Modality   string            `json:"modality,omitempty"`
}

type VideoTaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type VideoTaskContent struct {
	URL    string `json:"url,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

type VideoTaskUsage struct {
	TotalSeconds     int `json:"total_seconds,omitempty"`
	InputSeconds     int `json:"input_seconds,omitempty"`
	OutputSeconds    int `json:"output_seconds,omitempty"`
	InputImageCount  int `json:"input_image_count,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
}

// OaiErrorEnvelope MiniMax V2 OpenAI 风格错误（HTTP 状态码为真实错误码）。
type OaiErrorEnvelope struct {
	Type      string          `json:"type,omitempty"`
	Error     *OaiErrorDetail `json:"error,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

type OaiErrorDetail struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	HTTPCode string `json:"http_code,omitempty"`
}

func (e OaiErrorEnvelope) HasError() bool {
	return e.Error != nil && (e.Error.Message != "" || e.Error.Type != "")
}

func (e OaiErrorEnvelope) ErrorMessage() string {
	if e.Error == nil {
		return ""
	}
	if e.Error.Message != "" {
		return e.Error.Message
	}
	return e.Error.Type
}

func (e OaiErrorEnvelope) ErrorCode() string {
	if e.Error == nil {
		return ""
	}
	if e.Error.Type != "" {
		return e.Error.Type
	}
	return e.Error.HTTPCode
}
