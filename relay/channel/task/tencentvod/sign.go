package tencentvod

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

const vodService = "vod"
const vodAPIVersion = "2018-07-17"

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacSha256(s, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	_, _ = h.Write([]byte(s))
	return h.Sum(nil)
}

func tc3Authorization(secretID, secretKey, host, action string, timestamp int64, payloadJSON []byte) string {
	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\nx-tc-action:%s\n", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := fmt.Sprintf(
		"POST\n/\n\n%s\n%s\n%s",
		canonicalHeaders,
		signedHeaders,
		sha256hex(string(payloadJSON)),
	)
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, vodService)
	string2sign := fmt.Sprintf(
		"TC3-HMAC-SHA256\n%d\n%s\n%s",
		timestamp,
		credentialScope,
		sha256hex(canonicalRequest),
	)

	secretDate := hmacSha256(date, "TC3"+secretKey)
	secretService := hmacSha256(vodService, string(secretDate))
	signingKey := hmacSha256("tc3_request", string(secretService))
	signature := hex.EncodeToString(hmacSha256(string2sign, string(signingKey)))
	return fmt.Sprintf(
		"TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		secretID, credentialScope, signedHeaders, signature,
	)
}

func SignedPOSTJSON(proxy, endpoint, region string, cred Credentials, action string, payloadJSON []byte) (*http.Response, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid endpoint URL")
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	fullURL := u.Scheme + "://" + u.Host + "/"
	ts := common.GetTimestamp()

	req, err := http.NewRequest(http.MethodPost, fullURL, bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", tc3Authorization(cred.SecretID, cred.SecretKey, u.Host, action, ts, payloadJSON))
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", vodAPIVersion)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(ts, 10))
	if strings.TrimSpace(region) != "" {
		req.Header.Set("X-TC-Region", strings.TrimSpace(region))
	}

	client, err := service.GetHttpClientWithProxy(strings.TrimSpace(proxy))
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

package tencentvod

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

const vodService = "vod"
const vodAPIVersion = "2018-07-17"

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacSha256(s, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	_, _ = h.Write([]byte(s))
	return h.Sum(nil)
}

func tc3Authorization(secretID, secretKey, host, action string, timestamp int64, payloadJSON []byte) string {
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	actionLower := strings.ToLower(action)
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json", host, actionLower)
	signedHeaders := "content-type;host;x-tc-action"
	hashedRequestPayload := sha256hex(string(payloadJSON))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, hashedRequestPayload)

	algorithm := "TC3-HMAC-SHA256"
	requestTimestamp := strconv.FormatInt(timestamp, 10)
	ts, _ := strconv.ParseInt(requestTimestamp, 10, 64)
	t := time.Unix(ts, 0).UTC()
	date := t.Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, vodService)
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%s\n%s\n%s", algorithm, requestTimestamp, credentialScope, hashedCanonicalRequest)

	secretDate := hmacSha256(date, "TC3"+secretKey)
	secretService := hmacSha256(vodService, string(secretDate))
	signingKey := hmacSha256("tc3_request", string(secretService))
	signature := hex.EncodeToString(hmacSha256(string2sign, string(signingKey)))

	return fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, secretID, credentialScope, signedHeaders, signature)
}

func SignedPOSTJSON(proxy string, endpoint string, region string, cred Credentials, action string, payloadJSON []byte) (*http.Response, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid endpoint URL")
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	fullURL := u.Scheme + "://" + u.Host + "/"
	host := u.Host

	ts := common.GetTimestamp()
	auth := tc3Authorization(cred.SecretID, cred.SecretKey, host, action, ts, payloadJSON)

	req, err := http.NewRequest(http.MethodPost, fullURL, bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", vodAPIVersion)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(ts, 10))
	if strings.TrimSpace(region) != "" {
		req.Header.Set("X-TC-Region", strings.TrimSpace(region))
	}

	var client *http.Client
	if strings.TrimSpace(proxy) != "" {
		client, err = service.NewProxyHttpClient(strings.TrimSpace(proxy))
		if err != nil {
			return nil, err
		}
	} else {
		client = service.GetHttpClient()
	}
	return client.Do(req)
}

