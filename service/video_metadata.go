package service

import (
	"encoding/binary"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type VideoMetadata struct {
	DurationSec float64
	Width       int
	Height      int
	HasAudio    bool
}

type ffprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

var (
	ffprobeResolveOnce sync.Once
	ffprobePathCached  string
	ffprobeExtractOnce sync.Once
	ffprobeExtracted   string
)

func ProbeVideoMetadataFromURL(url string) (*VideoMetadata, error) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return nil, fmt.Errorf("empty video url")
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return nil, fmt.Errorf("unsupported video url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if meta, err := probeVideoMetadataByFFprobe(ctx, trimmed); err == nil {
		return meta, nil
	} else {
		// ffprobe 不可用或探测失败时，回退到纯 HTTP 头部解析（MP4/MOV）。
		if fallback, fbErr := probeVideoMetadataByHTTPRange(trimmed); fbErr == nil {
			return fallback, nil
		} else {
			if isFFprobeUnavailableError(err) {
				return nil, fmt.Errorf("ffprobe unavailable and HTTP fallback failed: %w", fbErr)
			}
			return nil, fmt.Errorf("ffprobe probe failed: %v; HTTP fallback failed: %w", err, fbErr)
		}
	}
}

func probeVideoMetadataByFFprobe(ctx context.Context, videoURL string) (*VideoMetadata, error) {
	ffprobeCmd := resolveFFprobeCommand()
	// ffprobe is required for robust remote MP4/MOV metadata parsing.
	cmd := exec.CommandContext(ctx, ffprobeCmd,
		"-v", "error",
		"-show_entries", "stream=codec_type,width,height:format=duration",
		"-of", "json",
		videoURL,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var parsed ffprobeOutput
	if err := common.Unmarshal(out, &parsed); err != nil {
		return nil, err
	}
	meta := &VideoMetadata{}
	for _, s := range parsed.Streams {
		switch strings.ToLower(strings.TrimSpace(s.CodecType)) {
		case "video":
			if s.Width > 0 {
				meta.Width = s.Width
			}
			if s.Height > 0 {
				meta.Height = s.Height
			}
		case "audio":
			meta.HasAudio = true
		}
	}
	if parsed.Format.Duration != "" {
		if d, err := strconv.ParseFloat(strings.TrimSpace(parsed.Format.Duration), 64); err == nil && d > 0 {
			meta.DurationSec = d
		}
	}
	if meta.DurationSec <= 0 || meta.Width <= 0 || meta.Height <= 0 {
		return nil, fmt.Errorf("insufficient video metadata")
	}
	meta.DurationSec = math.Ceil(meta.DurationSec*1000) / 1000
	return meta, nil
}

func isFFprobeUnavailableError(err error) bool {
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return true
	}
	if strings.Contains(strings.ToLower(err.Error()), "executable file not found") {
		return true
	}
	return false
}

type mp4Box struct {
	boxType string
	start   int
	size    int
	payload int
}

func probeVideoMetadataByHTTPRange(videoURL string) (*VideoMetadata, error) {
	if _, err := neturl.ParseRequestURI(videoURL); err != nil {
		return nil, fmt.Errorf("invalid video url: %w", err)
	}
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", "bytes=0-204800")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 205824))
	if err != nil {
		return nil, err
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("insufficient data for mp4 parsing")
	}
	meta := &VideoMetadata{}
	moov, ok := findFirstBox(data, 0, len(data), "moov")
	if !ok {
		return nil, fmt.Errorf("moov box not found in range payload")
	}
	if dur, ok := parseMvhdDuration(data, moov.payload, moov.start+moov.size); ok {
		meta.DurationSec = dur
	}
	tracks := findBoxes(data, moov.payload, moov.start+moov.size, "trak")
	for _, trak := range tracks {
		handlerType := parseTrackHandlerType(data, trak.payload, trak.start+trak.size)
		switch handlerType {
		case "soun":
			meta.HasAudio = true
		case "vide":
			if w, h, ok := parseTrackVideoSize(data, trak.payload, trak.start+trak.size); ok {
				meta.Width = w
				meta.Height = h
			}
		}
	}
	if meta.DurationSec <= 0 || meta.Width <= 0 || meta.Height <= 0 {
		return nil, fmt.Errorf("insufficient video metadata from HTTP range")
	}
	meta.DurationSec = math.Ceil(meta.DurationSec*1000) / 1000
	return meta, nil
}

func findFirstBox(data []byte, start, end int, target string) (mp4Box, bool) {
	boxes := findBoxes(data, start, end, target)
	if len(boxes) == 0 {
		return mp4Box{}, false
	}
	return boxes[0], true
}

func findBoxes(data []byte, start, end int, target string) []mp4Box {
	boxes := make([]mp4Box, 0)
	cursor := start
	for cursor+8 <= end && cursor+8 <= len(data) {
		size := int(binary.BigEndian.Uint32(data[cursor : cursor+4]))
		boxType := string(data[cursor+4 : cursor+8])
		headerLen := 8
		if size == 1 {
			if cursor+16 > len(data) || cursor+16 > end {
				break
			}
			size64 := binary.BigEndian.Uint64(data[cursor+8 : cursor+16])
			if size64 > uint64(^uint(0)>>1) {
				break
			}
			size = int(size64)
			headerLen = 16
		} else if size == 0 {
			size = end - cursor
		}
		if size < headerLen {
			break
		}
		boxEnd := cursor + size
		if boxEnd > end || boxEnd > len(data) {
			break
		}
		if boxType == target {
			boxes = append(boxes, mp4Box{
				boxType: boxType,
				start:   cursor,
				size:    size,
				payload: cursor + headerLen,
			})
		}
		cursor = boxEnd
	}
	return boxes
}

func parseMvhdDuration(data []byte, start, end int) (float64, bool) {
	mvhd, ok := findFirstBox(data, start, end, "mvhd")
	if !ok || mvhd.payload+20 > len(data) {
		return 0, false
	}
	version := data[mvhd.payload]
	switch version {
	case 1:
		if mvhd.payload+32 > len(data) {
			return 0, false
		}
		timescale := binary.BigEndian.Uint32(data[mvhd.payload+20 : mvhd.payload+24])
		duration := binary.BigEndian.Uint64(data[mvhd.payload+24 : mvhd.payload+32])
		if timescale == 0 {
			return 0, false
		}
		return float64(duration) / float64(timescale), true
	default:
		if mvhd.payload+20 > len(data) {
			return 0, false
		}
		timescale := binary.BigEndian.Uint32(data[mvhd.payload+12 : mvhd.payload+16])
		duration := binary.BigEndian.Uint32(data[mvhd.payload+16 : mvhd.payload+20])
		if timescale == 0 {
			return 0, false
		}
		return float64(duration) / float64(timescale), true
	}
}

func parseTrackHandlerType(data []byte, start, end int) string {
	mdia, ok := findFirstBox(data, start, end, "mdia")
	if !ok {
		return ""
	}
	hdlr, ok := findFirstBox(data, mdia.payload, mdia.start+mdia.size, "hdlr")
	if !ok || hdlr.payload+12 > len(data) {
		return ""
	}
	return string(data[hdlr.payload+8 : hdlr.payload+12])
}

func parseTrackVideoSize(data []byte, start, end int) (int, int, bool) {
	tkhd, ok := findFirstBox(data, start, end, "tkhd")
	if !ok || tkhd.payload+84 > len(data) {
		return 0, 0, false
	}
	version := data[tkhd.payload]
	var widthOffset int
	var heightOffset int
	if version == 1 {
		widthOffset = tkhd.payload + 88
		heightOffset = tkhd.payload + 92
	} else {
		widthOffset = tkhd.payload + 76
		heightOffset = tkhd.payload + 80
	}
	if heightOffset+4 > len(data) {
		return 0, 0, false
	}
	widthFixed := binary.BigEndian.Uint32(data[widthOffset : widthOffset+4])
	heightFixed := binary.BigEndian.Uint32(data[heightOffset : heightOffset+4])
	width := int(widthFixed >> 16)
	height := int(heightFixed >> 16)
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func resolveFFprobeCommand() string {
	ffprobeResolveOnce.Do(func() {
		cmdName := ffprobeCommandName()
		// 0) Embedded ffprobe (enabled by build tag: embed_ffprobe).
		if embeddedPath, ok := ensureEmbeddedFFprobe(cmdName); ok {
			ffprobePathCached = embeddedPath
			common.SysLog(fmt.Sprintf("video metadata: using embedded ffprobe: %s", embeddedPath))
			return
		}
		// 1) Prefer side-by-side bundled binaries.
		for _, candidate := range ffprobeLocalCandidates(cmdName) {
			if isExecutableFile(candidate) {
				ffprobePathCached = candidate
				common.SysLog(fmt.Sprintf("video metadata: using bundled ffprobe: %s", candidate))
				return
			}
		}
		// 2) Fallback to PATH.
		if path, err := exec.LookPath(cmdName); err == nil && path != "" {
			ffprobePathCached = path
			common.SysLog(fmt.Sprintf("video metadata: using PATH ffprobe: %s", path))
			return
		}
		// 3) Keep command name; execution will fail and caller will fallback to HTTP parsing.
		ffprobePathCached = cmdName
		common.SysLog("video metadata: ffprobe not found locally or in PATH, using HTTP fallback when needed")
	})
	return ffprobePathCached
}

func ffprobeCommandName() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}

func ffprobeLocalCandidates(cmdName string) []string {
	exePath, err := os.Executable()
	if err != nil || exePath == "" {
		return nil
	}
	exeDir := filepath.Dir(exePath)
	target := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	return []string{
		filepath.Join(exeDir, cmdName),
		filepath.Join(exeDir, "bin", "ffprobe", target, cmdName),
		filepath.Join(exeDir, "ffprobe", target, cmdName),
	}
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func ensureEmbeddedFFprobe(cmdName string) (string, bool) {
	ffprobeExtractOnce.Do(func() {
		blob, suggestedName, ok := getEmbeddedFFprobe(runtime.GOOS, runtime.GOARCH)
		if !ok || len(blob) == 0 {
			return
		}
		fileName := strings.TrimSpace(suggestedName)
		if fileName == "" {
			fileName = cmdName
		}
		targetDir := filepath.Join(os.TempDir(), "token-factory", "ffprobe", fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH))
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return
		}
		targetPath := filepath.Join(targetDir, fileName)
		if err := os.WriteFile(targetPath, blob, 0o755); err != nil {
			return
		}
		if runtime.GOOS != "windows" {
			_ = os.Chmod(targetPath, 0o755)
		}
		ffprobeExtracted = targetPath
	})
	if ffprobeExtracted == "" {
		return "", false
	}
	return ffprobeExtracted, true
}
