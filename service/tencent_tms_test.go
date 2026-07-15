package service

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitTextByRunes(t *testing.T) {
	text := strings.Repeat("测", 10001)
	chunks := splitTextByRunes(text, 10000)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if utf8.RuneCountInString(chunks[0]) != 10000 || utf8.RuneCountInString(chunks[1]) != 1 {
		t.Fatalf("unexpected chunk sizes: %d, %d", utf8.RuneCountInString(chunks[0]), utf8.RuneCountInString(chunks[1]))
	}
}

func TestTencentTMSAuthorization(t *testing.T) {
	authorization := tencentCloudAuthorization("secret-id", "secret-key", tencentTMSHost, tencentTMSService, tencentTMSAction, 1600000000, []byte("{\"Content\":\"dGVzdA==\",\"Type\":\"TEXT\"}"))
	if !strings.HasPrefix(authorization, "TC3-HMAC-SHA256 Credential=secret-id/2020-09-13/tms/tc3_request") {
		t.Fatalf("unexpected authorization: %s", authorization)
	}
	if !strings.Contains(authorization, "SignedHeaders=content-type;host;x-tc-action") {
		t.Fatalf("missing signed headers: %s", authorization)
	}
}

func TestNormalizeImageBase64(t *testing.T) {
	if got := normalizeImageBase64("data:image/png;base64,aGVsbG8="); got != "aGVsbG8=" {
		t.Fatalf("unexpected normalized base64: %s", got)
	}
}
