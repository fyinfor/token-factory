package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayErrorHandler_IncludesBodyWhenMessageEmpty(t *testing.T) {
	t.Parallel()
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString(`{"code":"InvalidParameter","request_id":"r1"}`)),
	}
	err := RelayErrorHandler(context.Background(), resp, false)
	require.NotNil(t, err)
	msg := err.ToOpenAIError().Message
	require.Contains(t, msg, "InvalidParameter")
}

func TestRelayErrorHandler_PlainTextUpstreamBody(t *testing.T) {
	t.Parallel()
	raw := `<400> foo.bar.InvalidParameter: The item of content should be a message of a certain modal`
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString(raw)),
	}
	err := RelayErrorHandler(context.Background(), resp, false)
	require.NotNil(t, err)
	require.Contains(t, err.ToOpenAIError().Message, "InvalidParameter")
	require.Contains(t, err.ToOpenAIError().Message, "content")
}

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokenFactoryError := &types.TokenFactoryError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(tokenFactoryError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, tokenFactoryError.StatusCode)
		})
	}
}
