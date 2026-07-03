package constant

import "testing"

func TestUsesRelayVideoPricing_IncludesTokenFactoryOpenOnlyForVideoRelay(t *testing.T) {
	if !UsesRelayVideoPricing(ChannelTypeTokenFactoryOpen) {
		t.Fatal("TokenFactoryOpen(60) should use relay video pricing")
	}
	if UsesRelayVideoPricing(ChannelTypeOpenAI) {
		t.Fatal("plain OpenAI channel should not use relay video pricing")
	}
	if !UsesRelayVideoPricing(ChannelTypeOpenAIVideo) {
		t.Fatal("OpenAIVideo should use relay video pricing")
	}
	if IsVideoTaskChannel(ChannelTypeTokenFactoryOpen) {
		t.Fatal("TokenFactoryOpen must stay out of IsVideoTaskChannel to preserve text chat relay")
	}
}
