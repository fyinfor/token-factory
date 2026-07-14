package logger

import (
	"math"
	"testing"
)

func TestCeilToFixedDecimals(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		digits int
		want   float64
	}{
		{name: "exact six digits", value: 1.234567, digits: 6, want: 1.234567},
		{name: "ceil seventh digit", value: 1.2345671, digits: 6, want: 1.234568},
		{name: "tiny positive", value: 0.0000001, digits: 6, want: 0.000001},
		{name: "zero", value: 0, digits: 6, want: 0},
		{name: "negative ceil away from zero", value: -1.2345671, digits: 6, want: -1.234568},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CeilToFixedDecimals(tt.value, tt.digits)
			if math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("CeilToFixedDecimals(%v, %d) = %v, want %v", tt.value, tt.digits, got, tt.want)
			}
		})
	}
}
