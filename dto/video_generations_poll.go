package dto

import "github.com/QuantumNous/new-api/common"

// VideoGenerationsPollContent 对应上游 Seedance/Ark 查询回包中的 content 节点。
type VideoGenerationsPollContent struct {
	VideoURL     string `json:"video_url,omitempty"`
	LastFrameURL string `json:"last_frame_url,omitempty"` // return_last_frame=true 时返回尾帧图
}

// VideoGenerationsPollExtra 对应上游 extra 元数据（计费/调试辅助，按需透传解析）。
type VideoGenerationsPollExtra struct {
	HasInputAudio bool   `json:"has_input_audio,omitempty"`
	HasInputImage bool   `json:"has_input_image,omitempty"`
	HasInputVideo bool   `json:"has_input_video,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
}

// VideoGenerationsPollUsage 对应上游 usage 节点。
type VideoGenerationsPollUsage struct {
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// VideoGenerationsPollUpstream 映射 GET /v1/video/generations/{task_id} 上游原始查询回包。
// 用于解析 task.Data / 实时拉取 body，并驱动字段透传与时间校正。
type VideoGenerationsPollUpstream struct {
	Content               *VideoGenerationsPollContent `json:"content,omitempty"`
	CreatedAt             int64                        `json:"created_at,omitempty"`
	Draft                 bool                         `json:"draft,omitempty"`
	Duration              int                          `json:"duration,omitempty"`
	ExecutionExpiresAfter int                          `json:"execution_expires_after,omitempty"`
	Extra                 *VideoGenerationsPollExtra   `json:"extra,omitempty"`
	FramesPerSecond       int                          `json:"framespersecond,omitempty"`
	GenerateAudio         bool                         `json:"generate_audio,omitempty"`
	ID                    string                       `json:"id,omitempty"`
	Model                 string                       `json:"model,omitempty"`
	Priority              int                          `json:"priority,omitempty"`
	Ratio                 string                       `json:"ratio,omitempty"`
	Resolution            string                       `json:"resolution,omitempty"`
	Seed                  int                          `json:"seed,omitempty"`
	ServiceTier           string                       `json:"service_tier,omitempty"`
	Status                string                       `json:"status,omitempty"`
	UpdatedAt             int64                        `json:"updated_at,omitempty"`
	Usage                 *VideoGenerationsPollUsage   `json:"usage,omitempty"`
}

// ParseVideoGenerationsPollUpstream 解析上游查询 JSON；解析失败返回 nil。
func ParseVideoGenerationsPollUpstream(raw []byte) *VideoGenerationsPollUpstream {
	if len(raw) == 0 {
		return nil
	}
	var upstream VideoGenerationsPollUpstream
	if err := common.Unmarshal(raw, &upstream); err != nil {
		return nil
	}
	return &upstream
}
