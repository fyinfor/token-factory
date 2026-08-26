package model

import "testing"

func TestChannelModelHasPassedConnectivityTest(t *testing.T) {
	idx := map[int][]string{
		1: {"gpt-4o"},
	}
	if ChannelModelHasPassedConnectivityTest(idx, 1, "gpt-4o") != true {
		t.Fatal("expected tested model to pass")
	}
	if ChannelModelHasPassedConnectivityTest(idx, 1, "gpt-4o-mini") {
		t.Fatal("expected untested model on same channel to fail")
	}
	if ChannelModelHasPassedConnectivityTest(idx, 2, "gpt-4o") {
		t.Fatal("expected other channel to fail")
	}
	if ChannelModelHasPassedConnectivityTest(nil, 1, "gpt-4o") {
		t.Fatal("expected nil index to fail")
	}
}
