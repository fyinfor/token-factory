package controller

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const outputModerationChunkRunes = 500

type moderationResponseWriter struct {
	gin.ResponseWriter
	ctx          *gin.Context
	textEnabled  bool
	imageEnabled bool
	stream       bool
	pending      bytes.Buffer
	err          error
	status       int
	size         int
	wroteHeader  bool
}

func newModerationResponseWriter(c *gin.Context, textEnabled, imageEnabled, stream bool) *moderationResponseWriter {
	return &moderationResponseWriter{
		ResponseWriter: c.Writer,
		ctx:            c,
		textEnabled:    textEnabled,
		imageEnabled:   imageEnabled,
		stream:         stream,
		status:         http.StatusOK,
		size:           -1,
	}
}

func (w *moderationResponseWriter) WriteHeader(code int) {
	if code > 0 && !w.wroteHeader {
		w.status = code
	}
}

func (w *moderationResponseWriter) WriteHeaderNow() {}

func (w *moderationResponseWriter) Status() int {
	return w.status
}

func (w *moderationResponseWriter) Size() int {
	return w.size
}

func (w *moderationResponseWriter) Written() bool {
	return w.size >= 0
}

func (w *moderationResponseWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return len(data), w.err
	}
	_, _ = w.pending.Write(data)
	if w.size < 0 {
		w.size = 0
	}
	w.size += len(data)
	if w.stream && w.textEnabled {
		text := extractModerationText(w.pending.Bytes(), true)
		if utf8.RuneCountInString(text) >= outputModerationChunkRunes {
			if err := w.moderateText(text); err != nil {
				w.err = err
				w.pending.Reset()
				return len(data), err
			}
			if err := w.flushPending(); err != nil {
				return len(data), err
			}
		}
	}
	return len(data), nil
}

func (w *moderationResponseWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

func (w *moderationResponseWriter) Flush() {}

func (w *moderationResponseWriter) Finalize() error {
	if w.err != nil {
		return w.err
	}
	data := w.pending.Bytes()
	if w.textEnabled {
		if err := w.moderateText(extractModerationText(data, w.stream)); err != nil {
			w.pending.Reset()
			return err
		}
	}
	if w.imageEnabled {
		files := extractModerationImages(data)
		if len(files) > 0 {
			result, err := service.CheckImagesWithTencentIMS(w.ctx.Request.Context(), files)
			if err != nil {
				w.pending.Reset()
				return err
			}
			if result.Blocked {
				w.pending.Reset()
				return errors.New("生成图片审核未通过 / Generated image failed moderation")
			}
		}
	}
	return w.flushPending()
}

func (w *moderationResponseWriter) moderateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	result, err := service.CheckTextWithTencentTMS(w.ctx.Request.Context(), text)
	if err != nil {
		return err
	}
	if result.Blocked {
		return errors.New("上游返回内容审核未通过 / Upstream response failed moderation")
	}
	return nil
}

func (w *moderationResponseWriter) flushPending() error {
	if w.pending.Len() == 0 {
		return nil
	}
	data := append([]byte(nil), w.pending.Bytes()...)
	w.pending.Reset()
	if !w.wroteHeader {
		w.ResponseWriter.WriteHeader(w.status)
		w.ResponseWriter.WriteHeaderNow()
		w.wroteHeader = true
	}
	if _, err := w.ResponseWriter.Write(data); err != nil {
		return err
	}
	if w.stream {
		w.ResponseWriter.Flush()
	}
	return nil
}

func extractModerationText(data []byte, stream bool) string {
	if stream {
		var texts []string
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
			var value any
			if common.Unmarshal([]byte(payload), &value) == nil {
				collectModerationText(value, "", &texts)
			}
		}
		return strings.Join(texts, "")
	}
	var value any
	if common.Unmarshal(data, &value) != nil {
		return ""
	}
	var texts []string
	collectModerationText(value, "", &texts)
	return strings.Join(texts, "\n")
}

func collectModerationText(value any, key string, texts *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			collectModerationText(child, strings.ToLower(childKey), texts)
		}
	case []any:
		for _, child := range typed {
			collectModerationText(child, key, texts)
		}
	case string:
		switch key {
		case "content", "text", "output_text", "completion", "reasoning_content", "delta":
			if strings.TrimSpace(typed) != "" {
				*texts = append(*texts, typed)
			}
		}
	}
}

func extractModerationImages(data []byte) []*types.FileMeta {
	var value any
	if common.Unmarshal(data, &value) != nil {
		return nil
	}
	var files []*types.FileMeta
	collectModerationImages(value, "", &files)
	return files
}

func collectModerationImages(value any, key string, files *[]*types.FileMeta) {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			collectModerationImages(child, strings.ToLower(childKey), files)
		}
	case []any:
		for _, child := range typed {
			collectModerationImages(child, key, files)
		}
	case string:
		var source *types.FileSource
		switch key {
		case "url":
			if strings.HasPrefix(typed, "http://") || strings.HasPrefix(typed, "https://") {
				source = types.NewURLFileSource(typed)
			}
		case "b64_json":
			if typed != "" {
				source = types.NewBase64FileSource(typed, "")
			}
		}
		if source != nil {
			*files = append(*files, types.NewImageFileMeta(source, ""))
		}
	}
}

func outputModerationError(err error) *types.TokenFactoryError {
	return types.NewErrorWithStatusCode(
		errors.New("上游返回内容审核失败，请稍后重试 / Upstream response moderation failed; please try again later: "+err.Error()),
		types.ErrorCodeContentModerationFailed,
		http.StatusBadGateway,
	)
}
