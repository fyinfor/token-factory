package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func RequireRealNameVerificationForTopUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setting.AliyunRealNameVerificationEnabled || !setting.AliyunRealNameVerificationRequiredForTopUp {
			c.Next()
			return
		}
		passed, err := model.HasPassedRealNameVerification(c.GetInt("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "message": "\u67e5\u8be2\u5b9e\u540d\u8ba4\u8bc1\u72b6\u6001\u5931\u8d25"})
			return
		}
		if !passed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "\u5145\u503c\u524d\u8bf7\u5148\u5b8c\u6210\u5b9e\u540d\u8ba4\u8bc1", "data": gin.H{"require_real_name_verification": true, "redirect": "/console/real-name-verification"}})
			return
		}
		c.Next()
	}
}
