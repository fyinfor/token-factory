package minimaxh3

// MiniMax H3 视频生成 V2：https://platform.minimaxi.com/docs/api-reference/video-generation-v2-create
// 渠道仅填写 baseUrl（默认 https://api.minimaxi.com/v2），接口后缀由网关拼接。

const (
	ChannelName    = "minimax-h3-video"
	DefaultBaseURL = "https://api.minimaxi.com/v2"
	DefaultModel   = ModelMiniMaxH3
)

var ModelList = []string{
	ModelMiniMaxH3,
}

const (
	ModelMiniMaxH3 = "MiniMax-H3"
)

// 相对 baseUrl 的接口后缀。用户禁止填写完整地址。
const (
	VideoGenerationPath = "/video_generation"
	QueryTaskPathPrefix = "/query/video_generation/"
)

const (
	Resolution768P = "768P"
	Resolution2K   = "2K"

	RatioAdaptive = "adaptive"
	Ratio21x9     = "21:9"
	Ratio16x9     = "16:9"
	Ratio4x3      = "4:3"
	Ratio1x1      = "1:1"
	Ratio3x4      = "3:4"
	Ratio9x16     = "9:16"
)

const (
	ContentTypeText     = "text"
	ContentTypeImageURL = "image_url"
	ContentTypeVideoURL = "video_url"
	ContentTypeAudioURL = "audio_url"
)

const (
	RoleFirstFrame     = "first_frame"
	RoleLastFrame      = "last_frame"
	RoleReferenceImage = "reference_image"
	RoleReferenceVideo = "reference_video"
	RoleReferenceAudio = "reference_audio"
)

const (
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

const (
	SceneTextToVideo      = "t2va"
	SceneImageToVideo     = "i2va"
	SceneReferenceToVideo = "r2va"
)

const (
	MinDuration        = 4
	MaxDuration        = 15
	DefaultDuration    = 5
	DefaultResolution  = Resolution768P
	MaxTextChars       = 7000
	MaxFirstFrames     = 1
	MaxLastFrames      = 1
	MaxReferenceImages = 9
	MaxReferenceVideos = 3
	MaxReferenceAudios = 3
)

var supportedModels = map[string]struct{}{
	ModelMiniMaxH3: {},
}

var supportedResolutions = map[string]struct{}{
	Resolution768P: {},
	Resolution2K:   {},
}

var supportedRatios = map[string]struct{}{
	RatioAdaptive: {},
	Ratio21x9:     {},
	Ratio16x9:     {},
	Ratio4x3:      {},
	Ratio1x1:      {},
	Ratio3x4:      {},
	Ratio9x16:     {},
}

var t2vaRatios = map[string]struct{}{
	Ratio21x9: {},
	Ratio16x9: {},
	Ratio4x3:  {},
	Ratio1x1:  {},
	Ratio3x4:  {},
	Ratio9x16: {},
}

var contentTypes = map[string]struct{}{
	ContentTypeText:     {},
	ContentTypeImageURL: {},
	ContentTypeVideoURL: {},
	ContentTypeAudioURL: {},
}

var i2vaRoles = map[string]struct{}{
	RoleFirstFrame: {},
	RoleLastFrame:  {},
}

var r2vaRoles = map[string]struct{}{
	RoleReferenceImage: {},
	RoleReferenceVideo: {},
	RoleReferenceAudio: {},
}
