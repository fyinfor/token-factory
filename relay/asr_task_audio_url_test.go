package relay

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveASRAsyncAudioURL_FromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions/async", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	url, err := resolveASRAsyncAudioURL(c, &dto.ASRTaskSubmitRequest{
		AudioURL: "https://cdn.example.com/a.mp3",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/a.mp3", url)
}

func TestResolveASRAsyncAudioURL_RejectsNonHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions/async", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	_, err := resolveASRAsyncAudioURL(c, &dto.ASRTaskSubmitRequest{FileURL: "ftp://x/a.mp3"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http/https")
}

func TestResolveASRAsyncAudioURL_RequiresSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions/async", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	_, err := resolveASRAsyncAudioURL(c, &dto.ASRTaskSubmitRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio_url")
}

func TestResolveASRAsyncAudioURL_MultipartFormURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "fun-asr"))
	require.NoError(t, writer.WriteField("audio_url", "https://cdn.example.com/from-form.mp3"))
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions/async", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, common.ReplaceRequestBody(c, body.Bytes()))

	url, err := resolveASRAsyncAudioURL(c, &dto.ASRTaskSubmitRequest{Model: "fun-asr"})
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/from-form.mp3", url)
}

func TestResolveASRAsyncAudioURL_MultipartMissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "fun-asr"))
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions/async", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, common.ReplaceRequestBody(c, body.Bytes()))

	_, err := resolveASRAsyncAudioURL(c, &dto.ASRTaskSubmitRequest{Model: "fun-asr"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file")
}

func TestFirstASRMultipartFormValue(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="a.mp3"`)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("file_url", " https://cdn.example.com/b.wav "))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(1<<20))

	assert.Equal(t, "https://cdn.example.com/b.wav", firstASRMultipartFormValue(req.MultipartForm, "audio_url", "file_url", "url"))
	assert.Equal(t, "", firstASRMultipartFormValue(nil, "audio_url"))
}
