package relay

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestImageResponseCaptureWriterSuppressesFlushWhileCapturing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	var captured bytes.Buffer
	writer := &imageResponseCaptureWriter{
		ResponseWriter: ctx.Writer,
		buf:            &captured,
		captureOnly:    true,
	}

	writer.WriteHeader(200)
	if _, err := writer.Write([]byte(`{"data":[]}`)); err != nil {
		t.Fatal(err)
	}
	writer.Flush()

	if recorder.Code != 200 || recorder.Body.Len() != 0 || recorder.Flushed {
		t.Fatalf("capture-only writer leaked response: code=%d bytes=%d flushed=%v", recorder.Code, recorder.Body.Len(), recorder.Flushed)
	}
	if captured.String() != `{"data":[]}` {
		t.Fatalf("unexpected captured body: %s", captured.String())
	}
}

func TestClearCapturedImageEntityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Writer.Header().Set("Content-Encoding", "gzip")
	ctx.Writer.Header().Set("Content-Length", "99")
	ctx.Writer.Header().Set("ETag", "old")
	ctx.Writer.Header().Set("Content-Type", "application/json")

	clearCapturedImageEntityHeaders(ctx.Writer)

	for _, key := range []string{"Content-Encoding", "Content-Length", "ETag"} {
		if value := ctx.Writer.Header().Get(key); value != "" {
			t.Fatalf("header %s was not cleared: %q", key, value)
		}
	}
	if value := ctx.Writer.Header().Get("Content-Type"); value != "application/json" {
		t.Fatalf("content type should be preserved: %q", value)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("header cleanup unexpectedly changed status: %d", recorder.Code)
	}
}

func TestRestoreCapturedContentEncodingPreservesGatewayCompression(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Writer.Header().Set("Content-Encoding", "gzip")

	clearCapturedImageEntityHeaders(ctx.Writer)
	if got := ctx.Writer.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("content encoding should be cleared before restore: %q", got)
	}
	restoreCapturedContentEncoding(ctx.Writer, "gzip")
	if got := ctx.Writer.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("gateway content encoding was not restored: %q", got)
	}
}
