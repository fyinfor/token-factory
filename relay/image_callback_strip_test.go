package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripCallbackURLFromRequestBody(t *testing.T) {
	in := []byte(`{"model":"dall-e-3","prompt":"a cat","callback_url":"https://example.com/cb","n":1}`)
	out, err := stripCallbackURLFromRequestBody(in, "application/json")
	require.NoError(t, err)
	assert.NotContains(t, string(out), "callback_url")
	assert.Contains(t, string(out), `"model":"dall-e-3"`)
	assert.Contains(t, string(out), `"prompt":"a cat"`)
}

func TestStripCallbackURLFromRequestBodyNoField(t *testing.T) {
	in := []byte(`{"model":"dall-e-3","prompt":"a cat"}`)
	out, err := stripCallbackURLFromRequestBody(in, "application/json")
	require.NoError(t, err)
	assert.JSONEq(t, string(in), string(out))
}
