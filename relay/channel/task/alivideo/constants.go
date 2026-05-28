package alivideo

// ModelList lists known DashScope video-synthesis model IDs (happyhorse family).
// Users may add custom model names in the channel editor.
var ModelList = []string{
	"happyhorse-1.0-t2v",
	"happyhorse-1.0-i2v",
	"happyhorse-1.0-r2v",
	"happyhorse-1.0-video-edit",
}

const ChannelName = "alivideo"

const (
	submitPath = "/v1/services/aigc/video-generation/video-synthesis"
	tasksPath  = "/v1/tasks"
)

// SubmitURL returns the full upstream video-synthesis submit URL for the given base.
func SubmitURL(baseURL string) string {
	return joinAPIPath(baseURL, submitPath)
}
