package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIImagesInformationalRecorder struct {
	header        http.Header
	body          bytes.Buffer
	informational []int
	status        int
	wroteFinal    bool
}

func newOpenAIImagesInformationalRecorder() *openAIImagesInformationalRecorder {
	return &openAIImagesInformationalRecorder{
		header: make(http.Header),
	}
}

func (r *openAIImagesInformationalRecorder) Header() http.Header {
	return r.header
}

func (r *openAIImagesInformationalRecorder) WriteHeader(code int) {
	if code >= 100 && code < 200 {
		r.informational = append(r.informational, code)
		return
	}
	if r.wroteFinal {
		return
	}
	r.status = code
	r.wroteFinal = true
}

func (r *openAIImagesInformationalRecorder) Write(data []byte) (int, error) {
	if !r.wroteFinal {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(data)
}

func (r *openAIImagesInformationalRecorder) Flush() {}

func TestOpenAIImagesJSONKeepaliveSendsInformationalStatusAndStops(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := newOpenAIImagesInformationalRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	keepalive := StartOpenAIImagesJSONKeepalive(c, time.Millisecond, time.Millisecond)
	require.NotNil(t, keepalive)
	defer keepalive.Stop()

	require.Eventually(t, func() bool {
		return OpenAIImagesJSONKeepaliveWasWritten(c) && len(rec.informational) > 0
	}, 100*time.Millisecond, time.Millisecond)
	require.Equal(t, http.StatusProcessing, rec.informational[0])
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.body.String())
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))

	keepalive.Stop()
	writtenCount := len(rec.informational)
	time.Sleep(5 * time.Millisecond)
	require.Equal(t, writtenCount, len(rec.informational))
}

func TestOpenAIImagesJSONKeepalivePreservesFinalJSONStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := newOpenAIImagesInformationalRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	keepalive := StartOpenAIImagesJSONKeepalive(c, time.Millisecond, time.Millisecond)
	require.NotNil(t, keepalive)
	require.Eventually(t, func() bool {
		return OpenAIImagesJSONKeepaliveWasWritten(c)
	}, 100*time.Millisecond, time.Millisecond)

	FinishOpenAIImagesJSONKeepalive(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})

	var payload map[string]bool
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(rec.body.Bytes()), &payload))
	require.True(t, payload["ok"])
	require.Equal(t, http.StatusOK, rec.status)
}
