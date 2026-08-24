package common

import "testing"

func TestMaskKeyHeadTail(t *testing.T) {
	raw := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtERCOmPp2NIDAQABXX"
	got := MaskKeyHeadTail(raw)
	want := raw[:8] + "..." + raw[len(raw)-8:]
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if !IsMaskedKeyDisplay(got, raw) {
		t.Fatalf("masked display should be treated as unchanged")
	}
	if IsMaskedKeyDisplay("brand-new-private-key-material", raw) {
		t.Fatalf("full new key should not be treated as masked")
	}
}
