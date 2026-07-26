package logger

import (
	"testing"
)

func TestLogHelperNilContextDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("LogError(nil) panicked: %v", r)
		}
	}()
	LogError(nil, "nil context should be safe")
	LogWarn(nil, "nil context should be safe")
	LogInfo(nil, "nil context should be safe")
}
