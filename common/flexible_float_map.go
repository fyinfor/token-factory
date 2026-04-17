package common

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseStringFloat64MapFlexible parses a JSON object whose values are either JSON numbers
// or objects with a numeric "ratio" field (e.g. mis-stored completion ratio metadata).
func ParseStringFloat64MapFlexible(jsonStr string) (map[string]float64, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" || jsonStr == "null" {
		return map[string]float64{}, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(raw))
	for k, v := range raw {
		var num float64
		if err := json.Unmarshal(v, &num); err == nil {
			out[k] = num
			continue
		}
		var wrapped struct {
			Ratio *float64 `json:"ratio"`
		}
		if err := json.Unmarshal(v, &wrapped); err != nil {
			return nil, fmt.Errorf("%q: %w", k, err)
		}
		if wrapped.Ratio != nil {
			out[k] = *wrapped.Ratio
			continue
		}
		return nil, fmt.Errorf("%q: expected number or object with numeric \"ratio\"", k)
	}
	return out, nil
}
