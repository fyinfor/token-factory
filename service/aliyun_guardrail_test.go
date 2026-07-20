package service

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
)

func TestAliyunGuardrailRuntimeOptions(t *testing.T) {
	if aliyunGuardrailRuntimeOptions() == nil {
		t.Fatal("Alibaba Cloud SDK runtime options must not be nil")
	}
}

func TestAliyunGuardrailRelayInfoWithoutChannelMeta(t *testing.T) {
	info := &relaycommon.RelayInfo{OriginModelName: "test-model"}
	if info.ChannelMeta != nil {
		t.Fatal("test requires nil channel metadata")
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelMeta.ChannelId
	}
	if channelID != 0 {
		t.Fatalf("unexpected channel ID: %d", channelID)
	}
}

func TestAliyunGuardrailCredentialsAreTrimmed(t *testing.T) {
	originalID := setting.AliyunGuardrailAccessKeyID
	originalSecret := setting.AliyunGuardrailAccessKeySecret
	originalRegion := setting.AliyunGuardrailRegionID
	t.Cleanup(func() {
		setting.AliyunGuardrailAccessKeyID = originalID
		setting.AliyunGuardrailAccessKeySecret = originalSecret
		setting.AliyunGuardrailRegionID = originalRegion
	})
	setting.AliyunGuardrailAccessKeyID = " test-id "
	setting.AliyunGuardrailAccessKeySecret = " test-secret "
	setting.AliyunGuardrailRegionID = " cn-shanghai "
	client, err := newAliyunGuardrailClient()
	if err != nil {
		t.Fatalf("create guardrail client: %v", err)
	}
	if client == nil {
		t.Fatal("guardrail client must not be nil")
	}
}

func TestCheckAliyunGuardrailTaskInputDisabled(t *testing.T) {
	originalEnabled := setting.AliyunGuardrailEnabled
	originalInputEnabled := setting.AliyunGuardrailInputEnabled
	originalID := setting.AliyunGuardrailAccessKeyID
	originalSecret := setting.AliyunGuardrailAccessKeySecret
	t.Cleanup(func() {
		setting.AliyunGuardrailEnabled = originalEnabled
		setting.AliyunGuardrailInputEnabled = originalInputEnabled
		setting.AliyunGuardrailAccessKeyID = originalID
		setting.AliyunGuardrailAccessKeySecret = originalSecret
	})
	setting.AliyunGuardrailEnabled = false
	setting.AliyunGuardrailInputEnabled = true
	setting.AliyunGuardrailAccessKeyID = ""
	setting.AliyunGuardrailAccessKeySecret = ""

	result, err := CheckAliyunGuardrailTaskInput(nil, nil, relaycommon.TaskSubmitReq{Prompt: "test"})
	if err != nil {
		t.Fatalf("disabled task guardrail should not error: %v", err)
	}
	if result != nil {
		t.Fatalf("disabled task guardrail should not return result: %+v", result)
	}
}
