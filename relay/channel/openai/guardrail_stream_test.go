package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestStripReasoningFromStreamData(t *testing.T) {
	reasoning := "step by step"
	content := "final answer"
	raw, err := common.Marshal(&dto.ChatCompletionsStreamResponse{
		Id:    "chatcmpl-test",
		Model: "test-model",
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
				ReasoningContent: &reasoning,
				Content:          &content,
			},
		}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	stripped, ok := stripReasoningFromStreamData(string(raw))
	if !ok {
		t.Fatal("expected content-only chunk to be sendable")
	}
	var got dto.ChatCompletionsStreamResponse
	if err := common.UnmarshalJsonStr(stripped, &got); err != nil {
		t.Fatalf("unmarshal stripped: %v", err)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices=%d", len(got.Choices))
	}
	if got.Choices[0].Delta.GetReasoningContent() != "" {
		t.Fatalf("reasoning should be stripped, got %q", got.Choices[0].Delta.GetReasoningContent())
	}
	if got.Choices[0].Delta.GetContentString() != content {
		t.Fatalf("content=%q", got.Choices[0].Delta.GetContentString())
	}
}

func TestStripReasoningFromStreamDataReasoningOnly(t *testing.T) {
	reasoning := "only thinking"
	raw, err := common.Marshal(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
				ReasoningContent: &reasoning,
			},
		}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, ok := stripReasoningFromStreamData(string(raw)); ok {
		t.Fatal("reasoning-only chunk should be skipped on replay")
	}
}
