package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

// HeaderNoFailover is the client request header that disables channel failover/retry switching.
const HeaderNoFailover = "X-TF-No-Failover"

// HeaderRequestsNoFailover reports whether the request asks to disable channel-level failover.
// Truthy values: "1", "true", "yes", "on" (case-insensitive). Absent/empty/other → false.
func HeaderRequestsNoFailover(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return parseNoFailoverValue(c.GetHeader(HeaderNoFailover))
}

// ApplyNoFailoverHeader reads X-TF-No-Failover and, when truthy, stores ContextKeyNoFailover.
func ApplyNoFailoverHeader(c *gin.Context) {
	if !HeaderRequestsNoFailover(c) {
		return
	}
	common.SetContextKey(c, constant.ContextKeyNoFailover, true)
}

// ContextNoFailover reports whether the current request has disabled channel failover.
func ContextNoFailover(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if HeaderRequestsNoFailover(c) {
		return true
	}
	v, ok := common.GetContextKey(c, constant.ContextKeyNoFailover)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func parseNoFailoverValue(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
