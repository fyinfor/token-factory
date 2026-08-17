package relay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForASRTask_Success(t *testing.T) {
	task := &model.AsrTask{TaskID: "asr_ok", UserID: 1, Status: dto.ASRTaskStatusPending}
	polls := 0
	got, err := waitForASRTask(context.Background(), task, asrWaitOptions{
		Timeout:  time.Second,
		Interval: time.Millisecond,
		Poll: func(ctx context.Context, current *model.AsrTask) error {
			polls++
			current.Status = dto.ASRTaskStatusSucceeded
			current.ResultText = "你好"
			current.AudioSeconds = 1.5
			return nil
		},
		Reload: func(taskID string, userID int) (*model.AsrTask, error) {
			return task, nil
		},
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, dto.ASRTaskStatusSucceeded, got.Status)
	assert.Equal(t, "你好", got.ResultText)
	assert.Equal(t, 1.5, got.AudioSeconds)
	assert.GreaterOrEqual(t, polls, 1)
}

func TestWaitForASRTask_Failed(t *testing.T) {
	task := &model.AsrTask{TaskID: "asr_fail", UserID: 1, Status: dto.ASRTaskStatusPending}
	got, err := waitForASRTask(context.Background(), task, asrWaitOptions{
		Timeout:  time.Second,
		Interval: time.Millisecond,
		Poll: func(ctx context.Context, current *model.AsrTask) error {
			current.Status = dto.ASRTaskStatusFailed
			current.FailReason = "upstream AccessDenied"
			return nil
		},
		Reload: func(taskID string, userID int) (*model.AsrTask, error) {
			return task, nil
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AccessDenied")
	require.NotNil(t, got)
	assert.Equal(t, dto.ASRTaskStatusFailed, got.Status)
}

func TestWaitForASRTask_Timeout(t *testing.T) {
	task := &model.AsrTask{TaskID: "asr_to", UserID: 1, Status: dto.ASRTaskStatusPending}
	_, err := waitForASRTask(context.Background(), task, asrWaitOptions{
		Timeout:  20 * time.Millisecond,
		Interval: 5 * time.Millisecond,
		Poll:     func(context.Context, *model.AsrTask) error { return nil },
		Reload:   func(string, int) (*model.AsrTask, error) { return task, nil },
	})
	require.ErrorIs(t, err, errASRSyncWaitTimeout)
	assert.Equal(t, dto.ASRTaskStatusPending, task.Status)
}

func TestWaitForASRTask_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	task := &model.AsrTask{TaskID: "asr_cancel", UserID: 1, Status: dto.ASRTaskStatusPending}
	_, err := waitForASRTask(ctx, task, asrWaitOptions{
		Timeout:  time.Second,
		Interval: 50 * time.Millisecond,
		Poll: func(context.Context, *model.AsrTask) error {
			cancel()
			return nil
		},
		Reload: func(string, int) (*model.AsrTask, error) { return task, nil },
	})
	require.ErrorIs(t, err, errASRSyncWaitCancelled)
}

func TestWriteASRSyncWaitResponse_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", nil)

	speaker := 0
	transcripts, err := common.Marshal([]dto.ASRTranscript{
		{Sentences: []dto.ASRTranscriptSentence{{BeginTime: 0, EndTime: 1000, Text: "hi", SpeakerID: &speaker}}},
	})
	require.NoError(t, err)

	writeASRSyncWaitResponse(c, &model.AsrTask{
		TaskID:            "asr_json",
		ResultText:        "hi",
		AudioSeconds:      2,
		ResultTranscripts: string(transcripts),
	}, "json")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.ASRSyncResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "hi", resp.Text)
	assert.Equal(t, float64(2), resp.Duration)
	assert.Equal(t, "asr_json", resp.TaskID)
	require.Len(t, resp.Transcripts, 1)
}

func TestWriteASRSyncWaitResponse_Text(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", nil)

	writeASRSyncWaitResponse(c, &model.AsrTask{TaskID: "asr_text", ResultText: "plain"}, "text")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	assert.Equal(t, "plain", w.Body.String())
}
