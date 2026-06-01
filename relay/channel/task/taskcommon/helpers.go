package taskcommon

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// UnmarshalMetadata converts a map[string]any metadata to a typed struct via JSON round-trip.
// This replaces the repeated pattern: json.Marshal(metadata) → json.Unmarshal(bytes, &target).
func UnmarshalMetadata(metadata map[string]any, target any) error {
	if metadata == nil {
		return nil
	}
	// Prevent metadata from overriding model fields to avoid billing bypass.
	delete(metadata, "model")
	metaBytes, err := common.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata failed: %w", err)
	}
	if err := common.Unmarshal(metaBytes, target); err != nil {
		return fmt.Errorf("unmarshal metadata failed: %w", err)
	}
	return nil
}

// DefaultString returns val if non-empty, otherwise fallback.
func DefaultString(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

// DefaultInt returns val if non-zero, otherwise fallback.
func DefaultInt(val, fallback int) int {
	if val == 0 {
		return fallback
	}
	return val
}

// EncodeLocalTaskID encodes an upstream operation name to a URL-safe base64 string.
// Used by Gemini/Vertex to store upstream names as task IDs.
func EncodeLocalTaskID(name string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(name))
}

// DecodeLocalTaskID decodes a base64-encoded upstream operation name.
func DecodeLocalTaskID(id string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecodeBase64Response attempts to decode a base64-encoded upstream response.
// It handles multiple formats:
// 1. JSON string containing base64: "eyJ...=="
// 2. Plain base64 string without quotes: eyJ...==
// 3. Plain JSON object (no decoding needed)
// Returns the decoded bytes or the original body if decoding fails.
func DecodeBase64Response(responseBody []byte) []byte {
	decodedBody := responseBody
	var strBody string
	if err := common.Unmarshal(responseBody, &strBody); err == nil {
		// Response is a JSON string, try to decode as base64
		if decoded, err := base64.StdEncoding.DecodeString(strBody); err == nil {
			decodedBody = decoded
		} else {
			// If base64 decode fails, the string might already be JSON
			// Try to unmarshal it directly
			var testObj map[string]any
			if err := common.Unmarshal([]byte(strBody), &testObj); err == nil {
				decodedBody = []byte(strBody)
			} else {
				// If that also fails, use the original body
				decodedBody = responseBody
			}
		}
	} else {
		// If unmarshal fails, try direct base64 decode
		// The body might be a base64 string without JSON quotes
		if decoded, err := base64.StdEncoding.DecodeString(string(responseBody)); err == nil {
			decodedBody = decoded
		} else {
			// If base64 decode fails, try removing quotes and decode
			trimmed := strings.Trim(string(responseBody), "\"")
			if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
				decodedBody = decoded
			} else {
				// If all attempts fail, use the original body as-is
				decodedBody = responseBody
			}
		}
	}

	return decodedBody
}

// WriteOpenAIVideoResponse writes OpenAIVideo JSON. On /v1/videos* routes timestamps
// are serialized as Unix int64 for new-api compatibility; other routes keep RFC3339.
func WriteOpenAIVideoResponse(c *gin.Context, video *dto.OpenAIVideo) {
	path := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	if dto.IsOpenAIVideosCompatPath(path) {
		data, err := dto.ConvertTimestampsToInt64(video)
		if err != nil {
			c.JSON(http.StatusOK, video)
			return
		}
		c.Data(http.StatusOK, "application/json", data)
		return
	}
	c.JSON(http.StatusOK, video)
}

// BuildProxyURL constructs the video proxy URL using the public task ID.
// e.g., "https://your-server.com/v1/videos/task_xxxx/content"
func BuildProxyURL(taskID string) string {
	return fmt.Sprintf("%s/v1/videos/%s/content", system_setting.ServerAddress, taskID)
}

// Status-to-progress mapping constants for polling updates.
const (
	ProgressSubmitted  = "10%"
	ProgressQueued     = "20%"
	ProgressInProgress = "30%"
	ProgressComplete   = "100%"
)

// ---------------------------------------------------------------------------
// BaseBilling — embeddable no-op implementations for TaskAdaptor billing methods.
// Adaptors that do not need custom billing can embed this struct directly.
// ---------------------------------------------------------------------------

type BaseBilling struct{}

// EstimateBilling returns nil (no extra ratios; use base model price).
func (BaseBilling) EstimateBilling(_ *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

// AdjustBillingOnSubmit returns nil (no submit-time adjustment).
func (BaseBilling) AdjustBillingOnSubmit(_ *relaycommon.RelayInfo, _ []byte) map[string]float64 {
	return nil
}

// AdjustBillingOnComplete returns 0 (keep pre-charged amount).
func (BaseBilling) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}
