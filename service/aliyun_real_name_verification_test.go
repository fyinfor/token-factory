package service

import "testing"

func TestIsAliyunInvalidCertNoError(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		want    bool
	}{
		{name: "invalid cert number", code: "401", message: "参数非法(certNo)", want: true},
		{name: "case insensitive", code: "401", message: "invalid CERTNO", want: true},
		{name: "different parameter", code: "401", message: "参数非法(certName)", want: false},
		{name: "different code", code: "500", message: "参数非法(certNo)", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAliyunInvalidCertNoError(test.code, test.message); got != test.want {
				t.Fatalf("isAliyunInvalidCertNoError(%q, %q) = %v, want %v", test.code, test.message, got, test.want)
			}
		})
	}
}
