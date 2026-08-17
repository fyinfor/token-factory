package model

import (
	"regexp"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var volcEngineVideoTaskIDTestPattern = regexp.MustCompile(`^cgt-\d{14}-[a-z0-9]{5}$`)

func TestConvertToVolcEngineVideoTaskID_UsesCreatedAtTimestamp(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	createdAt := time.Date(2026, 8, 12, 10, 30, 22, 0, loc)

	got := ConvertToVolcEngineVideoTaskID("task_GefCKS3KVTD4ZMDJIAky3jiml87Uqtr7", createdAt)

	require.True(t, volcEngineVideoTaskIDTestPattern.MatchString(got), "got %q", got)
	assert.True(t, len(got) > len("cgt-20260812103022-"), "converted id too short: %q", got)
	assert.Equal(t, "cgt-20260812103022-", got[:len("cgt-20260812103022-")])
	assert.NotContains(t, got, "task_")
}

func TestConvertToVolcEngineVideoTaskID_IdempotentForStandardID(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	createdAt := time.Date(2026, 8, 12, 10, 30, 22, 0, loc)
	first := ConvertToVolcEngineVideoTaskID("task_abcdef0123456789ABCDEF01234567", createdAt)

	got := ConvertToVolcEngineVideoTaskID(first, createdAt.Add(time.Hour))
	assert.Equal(t, first, got)
}

func TestConvertToVolcEngineVideoTaskID_GenericIDsProduceValidFormat(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, loc)

	ids := []string{
		"task_GefCKS3KVTD4ZMDJIAky3jiml87Uqtr7",
		"task_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"",
	}
	seen := make(map[string]struct{}, len(ids))
	for _, generic := range ids {
		got := ConvertToVolcEngineVideoTaskID(generic, createdAt)
		require.True(t, volcEngineVideoTaskIDTestPattern.MatchString(got), "generic=%q got=%q", generic, got)
		assert.Equal(t, "cgt-20260102030405-", got[:len("cgt-20260102030405-")])
		seen[got] = struct{}{}
	}
	assert.Len(t, seen, len(ids), "each conversion should mint a distinct random suffix")
}

func TestIsVolcEngineVideoTaskID(t *testing.T) {
	assert.True(t, IsVolcEngineVideoTaskID("cgt-20260812103022-c6r5j"))
	assert.False(t, IsVolcEngineVideoTaskID("task_GefCKS3KVTD4ZMDJIAky3jiml87Uqtr7"))
	assert.False(t, IsVolcEngineVideoTaskID("cgt-20260812-c6r5j"))
	assert.False(t, IsVolcEngineVideoTaskID("CGT-20260812103022-c6r5j"))
}

func TestGeneratePublicTaskID_SeedanceUsesVolcFormat(t *testing.T) {
	got := GeneratePublicTaskID(constant.ChannelTypeSeedance)
	assert.True(t, volcEngineVideoTaskIDTestPattern.MatchString(got), "got %q", got)
}

func TestGeneratePublicTaskID_OtherChannelsKeepGenericFormat(t *testing.T) {
	got := GeneratePublicTaskID(constant.ChannelTypeOpenAIVideo)
	assert.Regexp(t, `^task_[0-9A-Za-z]{32}$`, got)
}
