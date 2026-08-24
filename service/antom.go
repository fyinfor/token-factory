package service

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting"
)

const (
	antomCreatePaymentSessionURI = "/ams/api/v1/payments/createPaymentSession"
)

type antomAmount struct {
	Currency string `json:"currency"`
	Value    string `json:"value"`
}

type antomPaymentMethodOption struct {
	PaymentMethodType  string `json:"paymentMethodType"`
	PaymentMethodOrder string `json:"paymentMethodOrder,omitempty"`
}

type antomCreateSessionRequest struct {
	ProductCode             string `json:"productCode"`
	ProductScene            string `json:"productScene"`
	PaymentRequestId        string `json:"paymentRequestId"`
	PaymentAmount           antomAmount `json:"paymentAmount"`
	PaymentNotifyUrl        string `json:"paymentNotifyUrl"`
	PaymentRedirectUrl      string `json:"paymentRedirectUrl"`
	SettlementStrategy      *struct {
		SettlementCurrency string `json:"settlementCurrency"`
	} `json:"settlementStrategy,omitempty"`
	Order struct {
		ReferenceOrderId  string      `json:"referenceOrderId"`
		OrderDescription  string      `json:"orderDescription"`
		OrderAmount       antomAmount `json:"orderAmount"`
		Buyer             struct {
			ReferenceBuyerId string `json:"referenceBuyerId"`
		} `json:"buyer"`
	} `json:"order"`
	Env struct {
		TerminalType string `json:"terminalType"`
	} `json:"env"`
	AvailablePaymentMethod *struct {
		PaymentMethodTypeList []antomPaymentMethodOption `json:"paymentMethodTypeList"`
	} `json:"availablePaymentMethod,omitempty"`
}

type antomAPIResult struct {
	ResultCode    string `json:"resultCode"`
	ResultStatus  string `json:"resultStatus"`
	ResultMessage string `json:"resultMessage"`
}

type antomCreateSessionResponse struct {
	Result        antomAPIResult `json:"result"`
	NormalUrl     string         `json:"normalUrl"`
	PaymentSessionId string     `json:"paymentSessionId"`
}

type AntomNotifyPayload struct {
	NotifyType        string         `json:"notifyType"`
	Result            antomAPIResult `json:"result"`
	PaymentRequestId  string         `json:"paymentRequestId"`
	PaymentId         string         `json:"paymentId"`
	PaymentAmount     antomAmount    `json:"paymentAmount"`
}

func CreateAntomCheckoutSession(paymentRequestId string, payAmountMinor string, currency string, notifyURL string, redirectURL string, buyerId string, terminalType string) (string, error) {
	if !setting.AntomConfigured() {
		return "", fmt.Errorf("Antom 未配置")
	}
	if terminalType != "WAP" && terminalType != "APP" {
		terminalType = "WEB"
	}
	req := antomCreateSessionRequest{
		ProductCode:        "CASHIER_PAYMENT",
		ProductScene:       "CHECKOUT_PAYMENT",
		PaymentRequestId:   paymentRequestId,
		PaymentAmount:      antomAmount{Currency: currency, Value: payAmountMinor},
		PaymentNotifyUrl:   notifyURL,
		PaymentRedirectUrl: redirectURL,
	}
	req.Order.ReferenceOrderId = paymentRequestId
	req.Order.OrderDescription = "Top Up"
	req.Order.OrderAmount = antomAmount{Currency: currency, Value: payAmountMinor}
	req.Order.Buyer.ReferenceBuyerId = buyerId
	req.Env.TerminalType = terminalType
	settle := setting.GetAntomSettlementCurrency()
	if settle != "" {
		req.SettlementStrategy = &struct {
			SettlementCurrency string `json:"settlementCurrency"`
		}{SettlementCurrency: settle}
	}
	methods := setting.GetAntomPaymentMethodTypes()
	if len(methods) > 0 {
		list := make([]antomPaymentMethodOption, 0, len(methods))
		for i, m := range methods {
			list = append(list, antomPaymentMethodOption{
				PaymentMethodType:  m,
				PaymentMethodOrder: fmt.Sprintf("%d", i),
			})
		}
		req.AvailablePaymentMethod = &struct {
			PaymentMethodTypeList []antomPaymentMethodOption `json:"paymentMethodTypeList"`
		}{PaymentMethodTypeList: list}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	respBody, err := antomPOST(antomCreatePaymentSessionURI, body)
	if err != nil {
		return "", err
	}
	var parsed antomCreateSessionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("解析 Antom 响应失败: %w", err)
	}
	if parsed.Result.ResultStatus != "S" {
		msg := parsed.Result.ResultMessage
		if msg == "" {
			msg = parsed.Result.ResultCode
		}
		return "", fmt.Errorf("Antom 创建收银台失败: %s", msg)
	}
	if strings.TrimSpace(parsed.NormalUrl) == "" {
		return "", fmt.Errorf("Antom 未返回收银台链接")
	}
	return parsed.NormalUrl, nil
}

func VerifyAntomNotification(httpURI, clientId, requestTime, signatureHeader, body string) error {
	if !setting.AntomConfigured() {
		return fmt.Errorf("Antom 未配置")
	}
	sig := extractAntomSignature(signatureHeader)
	if sig == "" {
		return fmt.Errorf("缺少 Signature")
	}
	content := antomSignContent("POST", httpURI, clientId, requestTime, body)
	return verifyAntomRSA(setting.AntomPublicKey, content, sig)
}

func ParseAntomNotify(body []byte) (*AntomNotifyPayload, error) {
	var payload AntomNotifyPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func AntomSuccessAck() map[string]any {
	return map[string]any{
		"result": map[string]string{
			"resultCode":    "SUCCESS",
			"resultStatus":  "S",
			"resultMessage": "success",
		},
	}
}

func AntomFailAck(msg string) map[string]any {
	if msg == "" {
		msg = "error"
	}
	return map[string]any{
		"result": map[string]string{
			"resultCode":    "UNKNOWN_EXCEPTION",
			"resultStatus":  "F",
			"resultMessage": msg,
		},
	}
}

func antomPOST(uri string, body []byte) ([]byte, error) {
	clientId := strings.TrimSpace(setting.AntomClientId)
	requestTime := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02T15:04:05-07:00")
	content := antomSignContent("POST", uri, clientId, requestTime, string(body))
	sig, err := signAntomRSA(setting.AntomMerchantPrivateKey, content)
	if err != nil {
		return nil, err
	}
	endpoint := setting.GetAntomGatewayURL() + uri
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=UTF-8")
	httpReq.Header.Set("Client-Id", clientId)
	httpReq.Header.Set("Request-Time", requestTime)
	httpReq.Header.Set("Signature", "algorithm=RSA256, keyVersion=1, signature="+sig)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respClientId := resp.Header.Get("Client-Id")
	if respClientId == "" {
		respClientId = clientId
	}
	respTime := resp.Header.Get("Response-Time")
	respSig := resp.Header.Get("Signature")
	if respSig != "" && respTime != "" {
		if err := verifyAntomRSA(setting.AntomPublicKey, antomSignContent("POST", uri, respClientId, respTime, string(respBody)), extractAntomSignature(respSig)); err != nil {
			return nil, fmt.Errorf("Antom 响应验签失败: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Antom HTTP %d: %s", resp.StatusCode, truncateAntomErr(string(respBody)))
	}
	return respBody, nil
}

func antomSignContent(method, uri, clientId, ts, body string) string {
	return fmt.Sprintf("%s %s\n%s.%s.%s", method, uri, clientId, ts, body)
}

func extractAntomSignature(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToLower(p), "signature=") {
			raw := strings.TrimSpace(p[len("signature="):])
			decoded, err := url.QueryUnescape(raw)
			if err != nil {
				return raw
			}
			return decoded
		}
	}
	return ""
}

func signAntomRSA(privateKeyPEM, content string) (string, error) {
	key, err := parseAntomPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	hashed := sha256.Sum256([]byte(content))
	signed, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return url.QueryEscape(base64.StdEncoding.EncodeToString(signed)), nil
}

func verifyAntomRSA(publicKeyPEM, content, signatureB64 string) error {
	key, err := parseAntomPublicKey(publicKeyPEM)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("签名解码失败")
	}
	hashed := sha256.Sum256([]byte(content))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], raw); err != nil {
		return fmt.Errorf("验签失败")
	}
	return nil
}

func parseAntomPrivateKey(raw string) (*rsa.PrivateKey, error) {
	der, err := decodeAntomKeyBytes(raw)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("私钥不是 RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(der)
}

func parseAntomPublicKey(raw string) (*rsa.PublicKey, error) {
	der, err := decodeAntomKeyBytes(raw)
	if err != nil {
		return nil, err
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("解析公钥失败")
	}
	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("公钥不是 RSA")
	}
	return rsaKey, nil
}

func decodeAntomKeyBytes(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("密钥为空")
	}
	if block, _ := pem.Decode([]byte(raw)); block != nil {
		return block.Bytes, nil
	}
	cleaned := strings.ReplaceAll(raw, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return base64.StdEncoding.DecodeString(cleaned)
}

func truncateAntomErr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
