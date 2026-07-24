package controller

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	seedanceCallbackProbeMaxSessions   = 200
	seedanceCallbackProbeMaxEvents     = 50
	seedanceCallbackProbeMaxBodyBytes  = 1 << 20 // 1 MiB
	seedanceCallbackProbeTTL           = 2 * time.Hour
	seedanceCallbackProbeTokenBytesLen = 24
)

type seedanceCallbackProbeEvent struct {
	ReceivedAt  int64             `json:"received_at"`
	Method      string            `json:"method"`
	ContentType string            `json:"content_type,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	BodyBytes   int               `json:"body_bytes"`
	RemoteAddr  string            `json:"remote_addr,omitempty"`
}

type seedanceCallbackProbeSession struct {
	Token     string                       `json:"token"`
	CreatedAt int64                        `json:"created_at"`
	ExpiresAt int64                        `json:"expires_at"`
	Events    []seedanceCallbackProbeEvent `json:"events"`
}

type seedanceCallbackProbeStore struct {
	mu       sync.Mutex
	sessions map[string]*seedanceCallbackProbeSession
}

var seedanceCallbackProbes = &seedanceCallbackProbeStore{
	sessions: make(map[string]*seedanceCallbackProbeSession),
}

func (s *seedanceCallbackProbeStore) create() (*seedanceCallbackProbeSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(time.Now())
	if len(s.sessions) >= seedanceCallbackProbeMaxSessions {
		return nil, errSeedanceCallbackProbeCapacity
	}
	token := common.GetRandomString(seedanceCallbackProbeTokenBytesLen)
	now := time.Now()
	sess := &seedanceCallbackProbeSession{
		Token:     token,
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(seedanceCallbackProbeTTL).Unix(),
		Events:    make([]seedanceCallbackProbeEvent, 0, 4),
	}
	s.sessions[token] = sess
	return cloneSeedanceCallbackProbeSession(sess), nil
}

func (s *seedanceCallbackProbeStore) get(token string) (*seedanceCallbackProbeSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(time.Now())
	sess, ok := s.sessions[token]
	if !ok {
		return nil, false
	}
	return cloneSeedanceCallbackProbeSession(sess), true
}

func (s *seedanceCallbackProbeStore) appendEvent(token string, event seedanceCallbackProbeEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(time.Now())
	sess, ok := s.sessions[token]
	if !ok {
		return false
	}
	if len(sess.Events) >= seedanceCallbackProbeMaxEvents {
		sess.Events = sess.Events[1:]
	}
	sess.Events = append(sess.Events, event)
	return true
}

func (s *seedanceCallbackProbeStore) delete(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[token]; !ok {
		return false
	}
	delete(s.sessions, token)
	return true
}

func (s *seedanceCallbackProbeStore) purgeExpiredLocked(now time.Time) {
	nowUnix := now.Unix()
	for token, sess := range s.sessions {
		if sess == nil || sess.ExpiresAt <= nowUnix {
			delete(s.sessions, token)
		}
	}
}

func cloneSeedanceCallbackProbeSession(sess *seedanceCallbackProbeSession) *seedanceCallbackProbeSession {
	if sess == nil {
		return nil
	}
	out := *sess
	if sess.Events != nil {
		out.Events = make([]seedanceCallbackProbeEvent, len(sess.Events))
		copy(out.Events, sess.Events)
	}
	return &out
}

var errSeedanceCallbackProbeCapacity = &seedanceCallbackProbeCapacityError{}

type seedanceCallbackProbeCapacityError struct{}

func (e *seedanceCallbackProbeCapacityError) Error() string {
	return "seedance callback probe capacity exceeded"
}

func seedanceCallbackProbeBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(service.GetCallbackAddress()), "/")
}

func seedanceCallbackProbeURLs(token string) (callbackURL, inspectURL string) {
	base := seedanceCallbackProbeBaseURL()
	path := "/api/debug/seedance/callback/" + token
	if base == "" {
		return path, path
	}
	return base + path, base + path
}

// CreateSeedanceCallbackProbe 创建一次性回调探测会话，返回可填入 callback_url 的地址。
// POST /api/debug/seedance/callback
func CreateSeedanceCallbackProbe(c *gin.Context) {
	sess, err := seedanceCallbackProbes.create()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	callbackURL, inspectURL := seedanceCallbackProbeURLs(sess.Token)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token":         sess.Token,
			"callback_url":  callbackURL,
			"inspect_url":   inspectURL,
			"expires_at":    sess.ExpiresAt,
			"ttl_seconds":   int(seedanceCallbackProbeTTL.Seconds()),
			"usage":         "将 callback_url 填入 Seedance 透传请求；任务状态变化后用 GET inspect_url 查看是否收到回调。",
			"called":        false,
			"call_count":    0,
		},
	})
}

// ReceiveSeedanceCallbackProbe 接收火山方舟/上游推送的回调（公开，无需鉴权）。
// POST /api/debug/seedance/callback/:token
func ReceiveSeedanceCallbackProbe(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "token required"})
		return
	}
	limited := io.LimitReader(c.Request.Body, seedanceCallbackProbeMaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "read body failed"})
		return
	}
	if len(body) > seedanceCallbackProbeMaxBodyBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "body too large"})
		return
	}

	headers := map[string]string{}
	for _, key := range []string{"Content-Type", "User-Agent", "X-Request-Id", "X-Ark-Request-Id"} {
		if v := strings.TrimSpace(c.GetHeader(key)); v != "" {
			headers[key] = v
		}
	}
	event := seedanceCallbackProbeEvent{
		ReceivedAt:  time.Now().Unix(),
		Method:      c.Request.Method,
		ContentType: c.GetHeader("Content-Type"),
		Headers:     headers,
		Body:        string(body),
		BodyBytes:   len(body),
		RemoteAddr:  c.ClientIP(),
	}
	if !seedanceCallbackProbes.appendEvent(token, event) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "probe session not found or expired"})
		return
	}
	// 上游要求 2xx 确认；保持轻量响应。
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "ok"})
}

// InspectSeedanceCallbackProbe 查询探测会话是否已被成功调用。
// GET /api/debug/seedance/callback/:token
func InspectSeedanceCallbackProbe(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "token required"})
		return
	}
	sess, ok := seedanceCallbackProbes.get(token)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "probe session not found or expired"})
		return
	}
	callbackURL, inspectURL := seedanceCallbackProbeURLs(sess.Token)
	var last *seedanceCallbackProbeEvent
	if n := len(sess.Events); n > 0 {
		last = &sess.Events[n-1]
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token":         sess.Token,
			"callback_url":  callbackURL,
			"inspect_url":   inspectURL,
			"created_at":    sess.CreatedAt,
			"expires_at":    sess.ExpiresAt,
			"called":        len(sess.Events) > 0,
			"call_count":    len(sess.Events),
			"last_event":    last,
			"events":        sess.Events,
		},
	})
}

// DeleteSeedanceCallbackProbe 删除探测会话。
// DELETE /api/debug/seedance/callback/:token
func DeleteSeedanceCallbackProbe(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "token required"})
		return
	}
	if !seedanceCallbackProbes.delete(token) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "probe session not found or expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "deleted"})
}
