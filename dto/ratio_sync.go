package dto

type UpstreamDTO struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	Endpoint string `json:"endpoint"`
}

type UpstreamRequest struct {
	ChannelIDs []int64       `json:"channel_ids"`
	Upstreams  []UpstreamDTO `json:"upstreams"`
	Timeout    int           `json:"timeout"`
	SyncMode   string        `json:"sync_mode"`
	// IncludeAligned 为 true 时，即使本地已生效价与上游一致，仍在 differences 中返回该行
	//（便于应用同步后再次拉取仍能对照上游模型列表；默认 false 保持旧行为）
	IncludeAligned bool `json:"include_aligned"`
}

// TestResult 上游测试连通性结果
type TestResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// DifferenceItem 差异项
// Current 为系统全局默认（全局 ModelRatio / ModelPrice 等）
// UpstreamOld 为各渠道列「当前生效」旧值：有渠道覆盖则用渠道价，否则同 Current 语义（全局）
// Upstreams 为各渠道列上游拉取的新定价（数值），与 UpstreamOld 对照展示 旧/新

type DifferenceItem struct {
	Current     interface{}            `json:"current"`
	UpstreamOld map[string]interface{} `json:"upstream_old,omitempty"`
	Upstreams   map[string]interface{} `json:"upstreams"`
	Confidence  map[string]bool        `json:"confidence"`
}

type SyncableChannel struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Status  int    `json:"status"`
	Type    int    `json:"type"`
}
