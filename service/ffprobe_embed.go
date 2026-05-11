//go:build embed_ffprobe

package service

import _ "embed"

var (
	//go:embed ffprobe-bin/windows-amd64/ffprobe.exe
	ffprobeWindowsAmd64 []byte

	//go:embed ffprobe-bin/linux-amd64/ffprobe
	ffprobeLinuxAmd64 []byte

	//go:embed ffprobe-bin/linux-arm64/ffprobe
	ffprobeLinuxArm64 []byte
)

// getEmbeddedFFprobe 返回 embed_ffprobe 构建下打进二进制内的 ffprobe 字节（按 GOOS/GOARCH）。
func getEmbeddedFFprobe(goos, goarch string) ([]byte, string, bool) {
	switch goos + "-" + goarch {
	case "windows-amd64":
		if len(ffprobeWindowsAmd64) == 0 {
			return nil, "", false
		}
		return ffprobeWindowsAmd64, "ffprobe.exe", true
	case "linux-amd64":
		if len(ffprobeLinuxAmd64) == 0 {
			return nil, "", false
		}
		return ffprobeLinuxAmd64, "ffprobe", true
	case "linux-arm64":
		if len(ffprobeLinuxArm64) == 0 {
			return nil, "", false
		}
		return ffprobeLinuxArm64, "ffprobe", true
	default:
		return nil, "", false
	}
}
