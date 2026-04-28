package service

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Cloudflare can close idle HTTP requests after ~120s. Use 1xx informational
	// responses so keepalive does not commit the final HTTP status.
	OpenAIImagesJSONKeepaliveInitialDelay = 75 * time.Second
	OpenAIImagesJSONKeepaliveInterval     = 25 * time.Second

	openAIImagesJSONKeepaliveContextKey = "openai_images_json_keepalive"
)

type OpenAIImagesJSONKeepalive struct {
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once

	written  atomic.Bool
	terminal atomic.Bool
}

func StartOpenAIImagesJSONKeepalive(c *gin.Context, initialDelay, interval time.Duration) *OpenAIImagesJSONKeepalive {
	if c == nil || c.Request == nil || c.Writer == nil || initialDelay <= 0 || interval <= 0 {
		return nil
	}
	if existing := getOpenAIImagesJSONKeepalive(c); existing != nil {
		return existing
	}

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	keepalive := &OpenAIImagesJSONKeepalive{
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	c.Set(openAIImagesJSONKeepaliveContextKey, keepalive)

	go keepalive.run(c, initialDelay, interval)
	return keepalive
}

func FinishOpenAIImagesJSONKeepalive(c *gin.Context) {
	keepalive := getOpenAIImagesJSONKeepalive(c)
	if keepalive == nil {
		return
	}
	keepalive.terminal.Store(true)
	keepalive.Stop()
}

func StopOpenAIImagesJSONKeepalive(c *gin.Context) {
	keepalive := getOpenAIImagesJSONKeepalive(c)
	if keepalive == nil {
		return
	}
	keepalive.Stop()
}

func OpenAIImagesJSONKeepaliveWasWritten(c *gin.Context) bool {
	keepalive := getOpenAIImagesJSONKeepalive(c)
	return keepalive != nil && keepalive.written.Load()
}

func (k *OpenAIImagesJSONKeepalive) Stop() {
	if k == nil {
		return
	}
	k.stopOnce.Do(func() {
		close(k.stopCh)
		<-k.doneCh
	})
}

func (k *OpenAIImagesJSONKeepalive) run(c *gin.Context, initialDelay, interval time.Duration) {
	defer close(k.doneCh)

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-c.Request.Context().Done():
		return
	case <-k.stopCh:
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if !k.write(c) {
			return
		}

		select {
		case <-ticker.C:
		case <-c.Request.Context().Done():
			return
		case <-k.stopCh:
			return
		}
	}
}

func (k *OpenAIImagesJSONKeepalive) write(c *gin.Context) bool {
	if k.terminal.Load() || c == nil || c.Writer == nil {
		return false
	}

	writer := openAIImagesKeepaliveResponseWriter(c)
	if writer == nil {
		return false
	}

	writer.WriteHeader(http.StatusProcessing)
	k.written.Store(true)
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return true
}

func openAIImagesKeepaliveResponseWriter(c *gin.Context) http.ResponseWriter {
	type responseWriterUnwrapper interface {
		Unwrap() http.ResponseWriter
	}
	if c == nil || c.Writer == nil {
		return nil
	}
	if unwrapper, ok := c.Writer.(responseWriterUnwrapper); ok {
		return unwrapper.Unwrap()
	}
	return nil
}

func getOpenAIImagesJSONKeepalive(c *gin.Context) *OpenAIImagesJSONKeepalive {
	if c == nil {
		return nil
	}
	value, ok := c.Get(openAIImagesJSONKeepaliveContextKey)
	if !ok {
		return nil
	}
	keepalive, _ := value.(*OpenAIImagesJSONKeepalive)
	return keepalive
}
