package taskcommon

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// IsPassThroughBodyEnabled reports whether the video task submit should forward
// the client request body to upstream instead of adaptor-specific conversion.
func IsPassThroughBodyEnabled(info *relaycommon.RelayInfo) bool {
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled {
		return true
	}
	return info != nil && info.ChannelMeta != nil && info.ChannelSetting.PassThroughBodyEnabled
}

// gatewayOnlyTaskPassThroughStripFields are unified-gateway fields kept for
// validation/billing but omitted from upstream when body passthrough is on.
// Seedance/Doubao native API uses content[].text rather than prompt.
var gatewayOnlyTaskPassThroughStripFields = []string{
	"prompt",
}

// BuildPassThroughRequestBody returns the client body for upstream, rewriting
// model to the mapped upstream name and stripping gateway-only fields.
func BuildPassThroughRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if c == nil {
		return nil, errors.New("missing request context")
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_request_body_failed")
	}

	contentType := ""
	if c.Request != nil {
		contentType = c.Request.Header.Get("Content-Type")
	}
	upstreamModel := ""
	if info != nil && info.ChannelMeta != nil {
		upstreamModel = strings.TrimSpace(info.UpstreamModelName)
	}

	if strings.HasPrefix(contentType, "application/json") || looksLikeJSONObject(cachedBody) {
		body, err := rewriteJSONPassThroughBody(cachedBody, upstreamModel)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		return bytes.NewReader(body), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		return buildMultipartPassThroughBody(c, cachedBody, upstreamModel)
	}

	return common.ReaderOnly(storage), nil
}

func looksLikeJSONObject(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func rewriteJSONPassThroughBody(cachedBody []byte, upstreamModel string) ([]byte, error) {
	var bodyMap map[string]any
	if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
		return nil, err
	}
	if bodyMap == nil {
		bodyMap = map[string]any{}
	}
	for _, key := range gatewayOnlyTaskPassThroughStripFields {
		delete(bodyMap, key)
	}
	if upstreamModel != "" {
		bodyMap["model"] = upstreamModel
	}
	return common.Marshal(bodyMap)
}

func buildMultipartPassThroughBody(c *gin.Context, cachedBody []byte, upstreamModel string) (io.Reader, error) {
	formData, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return bytes.NewReader(cachedBody), nil
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	strip := make(map[string]struct{}, len(gatewayOnlyTaskPassThroughStripFields)+1)
	for _, key := range gatewayOnlyTaskPassThroughStripFields {
		strip[key] = struct{}{}
	}
	strip["model"] = struct{}{}

	if upstreamModel != "" {
		if err := writer.WriteField("model", upstreamModel); err != nil {
			return nil, err
		}
	}

	for key, values := range formData.Value {
		if _, skip := strip[key]; skip {
			continue
		}
		for _, v := range values {
			if err := writer.WriteField(key, v); err != nil {
				return nil, err
			}
		}
	}
	for fieldName, fileHeaders := range formData.File {
		for _, fh := range fileHeaders {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			fileCT := fh.Header.Get("Content-Type")
			if fileCT == "" || fileCT == "application/octet-stream" {
				buf512 := make([]byte, 512)
				n, _ := io.ReadFull(f, buf512)
				fileCT = http.DetectContentType(buf512[:n])
				_ = f.Close()
				f, err = fh.Open()
				if err != nil {
					continue
				}
			}
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
			h.Set("Content-Type", fileCT)
			part, err := writer.CreatePart(h)
			if err != nil {
				_ = f.Close()
				continue
			}
			_, _ = io.Copy(part, f)
			_ = f.Close()
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if c.Request != nil {
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	}
	return &buf, nil
}
