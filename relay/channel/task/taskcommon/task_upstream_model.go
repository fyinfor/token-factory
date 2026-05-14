package taskcommon

import (
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// RelayTaskUpstreamModel returns the model string to send in upstream task requests after
// relay/helper.ModelMappedHelper (covers channel model_mapping and TokenFactoryOpen route prefix).
// New video task adaptors should use this when building upstream payloads instead of only
// checking info.IsModelMapped.
func RelayTaskUpstreamModel(info *relaycommon.RelayInfo, reqModel string) string {
	if info != nil && info.UseRelayTaskUpstreamModel() {
		if s := strings.TrimSpace(info.UpstreamModelName); s != "" {
			return s
		}
	}
	return strings.TrimSpace(reqModel)
}
