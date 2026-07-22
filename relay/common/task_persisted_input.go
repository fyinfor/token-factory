package common

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextKeyTaskPersistedInput 存放任务入库用的原始请求 JSON（渠道可覆盖默认 TaskSubmitReq 序列化）。
const ContextKeyTaskPersistedInput = "task_persisted_input"

// SetTaskPersistedInput 写入任务 Properties.Input 的最终落库字符串。
func SetTaskPersistedInput(c *gin.Context, input string) {
	if c == nil {
		return
	}
	c.Set(ContextKeyTaskPersistedInput, input)
}

// GetTaskPersistedInput 读取渠道预先写入的落库 Input；未设置时 ok=false。
func GetTaskPersistedInput(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(ContextKeyTaskPersistedInput)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}
