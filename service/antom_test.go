package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"testing"
)

func TestAntomSignAndVerify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	priv := base64.StdEncoding.EncodeToString(privDER)
	pub := base64.StdEncoding.EncodeToString(pubDER)
	content := antomSignContent("POST", "/ams/api/v1/payments/createPaymentSession", "CLIENT", "2026-01-01T12:00:00+08:00", `{"a":1}`)
	sig, err := signAntomRSA(priv, content)
	if err != nil {
		t.Fatal(err)
	}
	extracted := extractAntomSignature("algorithm=RSA256, keyVersion=1, signature=" + sig)
	if err := verifyAntomRSA(pub, content, extracted); err != nil {
		t.Fatal(err)
	}
}
