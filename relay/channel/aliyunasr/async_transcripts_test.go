package aliyunasr

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAsyncSubmitParameters_DiarizationEnabled(t *testing.T) {
	enabled := true
	params := BuildAsyncSubmitParameters(&enabled)
	require.NotNil(t, params)
	require.NotNil(t, params.DiarizationEnabled)
	assert.True(t, *params.DiarizationEnabled)
	assert.Equal(t, []int{0}, params.ChannelID)

	assert.Nil(t, BuildAsyncSubmitParameters(nil).DiarizationEnabled)
}

func TestBuildUserTranscripts_WithSpeakerID(t *testing.T) {
	speaker0 := 0
	speaker1 := 1
	result := &aliASRTranscriptionResult{
		Transcripts: []aliASRTranscript{
			{
				Sentences: []aliASRSentence{
					{BeginTime: 100, EndTime: 3820, Text: "你好，我们今天讨论项目进度。", SpeakerID: &speaker0},
					{BeginTime: 3820, EndTime: 6500, Text: "好的，我先汇报一下。", SpeakerID: &speaker1},
				},
			},
		},
	}

	transcripts := BuildUserTranscripts(result)
	require.Len(t, transcripts, 1)
	require.Len(t, transcripts[0].Sentences, 2)
	assert.Equal(t, int64(100), transcripts[0].Sentences[0].BeginTime)
	assert.Equal(t, int64(3820), transcripts[0].Sentences[0].EndTime)
	require.NotNil(t, transcripts[0].Sentences[0].SpeakerID)
	assert.Equal(t, 0, *transcripts[0].Sentences[0].SpeakerID)
	require.NotNil(t, transcripts[0].Sentences[1].SpeakerID)
	assert.Equal(t, 1, *transcripts[0].Sentences[1].SpeakerID)

	b, err := common.Marshal(transcripts)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"speaker_id":0`)
	assert.Contains(t, string(b), `"speaker_id":1`)
}

func TestBuildUserTranscripts_Empty(t *testing.T) {
	assert.Nil(t, BuildUserTranscripts(nil))
	assert.Nil(t, BuildUserTranscripts(&aliASRTranscriptionResult{}))
}

func TestIsPermanentUpstreamHTTPError(t *testing.T) {
	assert.True(t, IsPermanentUpstreamHTTPError(&UpstreamHTTPError{
		StatusCode: 403,
		Code:       "AccessDenied.Unpurchased",
		Message:    "Access to model denied",
	}))
	assert.True(t, IsPermanentUpstreamHTTPError(&UpstreamHTTPError{StatusCode: 401, Code: "InvalidApiKey"}))
	assert.True(t, IsPermanentUpstreamHTTPError(&UpstreamHTTPError{StatusCode: 404, Code: "NotFound"}))
	assert.False(t, IsPermanentUpstreamHTTPError(&UpstreamHTTPError{StatusCode: 429, Code: "Throttling"}))
	assert.False(t, IsPermanentUpstreamHTTPError(&UpstreamHTTPError{StatusCode: 500, Code: "InternalError"}))
	assert.False(t, IsPermanentUpstreamHTTPError(assert.AnError))
}
