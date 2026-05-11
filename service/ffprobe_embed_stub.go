//go:build !embed_ffprobe

package service

// getEmbeddedFFprobe 在非 embed_ffprobe 构建下不可用（返回 false）。
func getEmbeddedFFprobe(goos, goarch string) ([]byte, string, bool) {
	return nil, "", false
}
