package aliyunasr

import (
	"path/filepath"
	"strings"
)

// 阿里云百炼（DashScope）ASR 语音转写渠道。
// 上游基础地址由渠道配置填写，例如：https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api
// 以下路径由后端自动拼接，渠道配置无需填写完整 URL。

const (
	// SyncGenerationPath 同步转写（短音频，≤5 分钟 / ≤10MB）：多模态生成接口，一次性返回识别文本。
	SyncGenerationPath = "/v1/services/aigc/multimodal-generation/generation"
	// AsyncSubmitPath 异步转写任务提交（长音频，≤12 小时 / ≤2GB）：需要请求头 X-DashScope-Async: enable。
	AsyncSubmitPath = "/v1/services/audio/asr/transcription"
	// AsyncTaskQueryPathPrefix 异步任务查询前缀，完整路径为 {prefix}/{task_id}。
	AsyncTaskQueryPathPrefix = "/v1/tasks"
)

// SyncModelList 同步转写适用模型（POST /v1/services/aigc/multimodal-generation/generation）。
var SyncModelList = []string{
	"qwen3-asr-flash",
	"qwen-audio-3.0-asr-flash",
	"fun-asr-flash",
	"fun-asr-flash-2026-06-15",
}

// AsyncModelList 异步转写适用模型（POST /v1/services/audio/asr/transcription + GET /v1/tasks/{task_id}）。
var AsyncModelList = []string{
	"qwen3-asr-flash-filetrans",
	"qwen-audio-3.0-asr-flash-filetrans",
	"Fun-ASR",
	"fun-asr",
}

var ChannelName = "aliyunasr"

// SampleAudioURL 渠道连通性测试使用的官方短音频样例（公网可访问，约数秒）。
// 同步/异步转写均可使用；仅用于验证密钥与上游可达，不做完整识别结果断言。
const SampleAudioURL = "https://dashscope.oss-cn-beijing.aliyuncs.com/audios/welcome.mp3"

const (
	// maxSyncAudioFileSize 同步转写上游文件大小限制：10MB。
	maxSyncAudioFileSize = 10 << 20
	// audioTokensPerSecond DashScope ASR 上游 usage 中 audio token 与实际音频时长的换算关系（约 25 token/秒），
	// 用于同步链路在无法本地解析音频时长时（URL 模式）按上游计费信息折算秒数。
	audioTokensPerSecond = 25.0
	// AsyncPreConsumeSeconds 异步转写提交时预扣的音频时长（秒），成功后按 usage.duration 补差价。
	AsyncPreConsumeSeconds = 60
)

// UsesFunASRFlashSyncProtocol 判断同步链路是否使用 Fun-ASR / Qwen-Audio 协议：
// content=type=input_audio + parameters.format/sample_rate。
// 否则走 Qwen3-ASR-Flash 协议：content={"audio":url} + parameters.result_format/asr_options。
func UsesFunASRFlashSyncProtocol(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" || strings.Contains(m, "filetrans") {
		return false
	}
	// Qwen3-ASR（非 Qwen-Audio）走原生 multimodal {"audio":url}
	if strings.Contains(m, "qwen3") && strings.Contains(m, "asr") {
		return false
	}
	return strings.Contains(m, "fun-asr") || strings.Contains(m, "qwen-audio")
}

// InferAudioFormat 从音频 URL / 文件名推断上游 parameters.format（必填）。
func InferAudioFormat(audioSource, filename string) string {
	ext := ""
	if filename != "" {
		ext = strings.ToLower(filepath.Ext(filename))
	}
	if ext == "" {
		u := audioSource
		// 去掉 data URI / query
		if strings.HasPrefix(strings.ToLower(u), "data:") {
			// data:audio/mpeg;base64,... → mpeg
			if i := strings.Index(u, ";"); i > 5 {
				mime := strings.ToLower(u[5:i])
				switch mime {
				case "audio/wav", "audio/x-wav", "audio/wave":
					return "wav"
				case "audio/mpeg", "audio/mp3":
					return "mp3"
				case "audio/opus":
					return "opus"
				case "audio/aac":
					return "aac"
				case "audio/flac":
					return "flac"
				case "audio/ogg":
					return "ogg"
				case "audio/mp4", "audio/m4a":
					return "m4a"
				}
			}
			return "mp3"
		}
		if i := strings.Index(u, "?"); i >= 0 {
			u = u[:i]
		}
		if i := strings.Index(u, "#"); i >= 0 {
			u = u[:i]
		}
		ext = strings.ToLower(filepath.Ext(u))
	}
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "mp3", "wav", "opus", "aac", "flac", "m4a", "ogg", "pcm", "amr", "webm":
		return ext
	case "mpeg", "mpga":
		return "mp3"
	default:
		return "mp3"
	}
}

// UsesSingularFileURL 判断异步提交是否应使用 input.file_url（单值）。
// Qwen3-ASR-Flash-Filetrans 使用 file_url；Qwen-Audio-3.0 / Fun-ASR 使用 file_urls 数组。
func UsesSingularFileURL(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(m, "qwen3") && strings.Contains(m, "filetrans")
}

// BuildAsyncInput 按模型协议构造异步 input。
func BuildAsyncInput(model, fileURL string) aliASRAsyncInput {
	if UsesSingularFileURL(model) {
		return aliASRAsyncInput{FileURL: fileURL}
	}
	return aliASRAsyncInput{FileURLs: []string{fileURL}}
}
