package model

import "testing"

func TestNormalizeRouteChannelWeight(t *testing.T) {
	w, on := NormalizeRouteChannelWeight(0, true)
	if w != 0 || on {
		t.Fatalf("weight 0 must be closed, got %d %v", w, on)
	}
	w, on = NormalizeRouteChannelWeight(80, false)
	if w != 0 || on {
		t.Fatalf("disabled must store weight 0, got %d %v", w, on)
	}
	w, on = NormalizeRouteChannelWeight(50, true)
	if w != 50 || !on {
		t.Fatalf("open weight should keep, got %d %v", w, on)
	}
}
