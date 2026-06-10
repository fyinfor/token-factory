package service

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const LocalUploadFolder = "uploads"

var localUploadPrefixSegmentRegexp = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func uploadFileExt(filename string) string {
	ext := path.Ext(strings.TrimSpace(filename))
	if ext != "" && len(ext) > 16 {
		return ""
	}
	return strings.ToLower(ext)
}

func LocalUploadBaseDir(storageRoot string) string {
	root := strings.TrimSpace(storageRoot)
	if root == "" || root == "." || root == LocalUploadFolder {
		root = "."
	}
	return filepath.Join(root, LocalUploadFolder)
}

func NormalizeLocalUploadPrefix(prefix string) (string, error) {
	p := strings.TrimSpace(prefix)
	if p == "" {
		return "", nil
	}
	if strings.Contains(p, "\\") || strings.Contains(p, ":") {
		return "", fmt.Errorf("本地文件夹前缀只能使用 / 分隔相对路径")
	}
	p = strings.Trim(p, "/")
	if p == "" {
		return "", nil
	}
	rawSegments := strings.Split(p, "/")
	for _, segment := range rawSegments {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("本地文件夹前缀不能包含上级目录")
		}
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		return "", nil
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." || path.IsAbs(cleaned) {
		return "", fmt.Errorf("本地文件夹前缀不能包含上级目录或绝对路径")
	}
	segments := strings.Split(cleaned, "/")
	validSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" || segment == "." {
			continue
		}
		if segment == ".." {
			return "", fmt.Errorf("本地文件夹前缀不能包含上级目录")
		}
		if !localUploadPrefixSegmentRegexp.MatchString(segment) {
			return "", fmt.Errorf("本地文件夹前缀只能包含字母、数字、点、下划线和短横线")
		}
		validSegments = append(validSegments, segment)
	}
	if len(validSegments) == 0 {
		return "", nil
	}
	return path.Join(validSegments...), nil
}

func BuildLocalUploadObjectPath(prefix string, ext string) (string, error) {
	p, err := NormalizeLocalUploadPrefix(prefix)
	if err != nil {
		return "", err
	}
	return BuildUploadObjectPath(p, ext), nil
}

func BuildUploadObjectPath(prefix string, ext string) string {
	p := strings.Trim(prefix, "/")
	if p != "" {
		p += "/"
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	datePath := time.Now().Format("2006/01/02")
	return fmt.Sprintf("%s%s/%s%s", p, datePath, uuid.NewString(), ext)
}
