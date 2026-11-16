package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
)

func Limiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		loginLimiter := global.LoginLimiter
		clientIp := c.ClientIP()
		adminRequest := strings.HasPrefix(c.Request.URL.Path, "/api/admin/")
		if adminRequest {
			clientIp = AdminClientIP(c)
		}
		banned, _ := loginLimiter.CheckSecurityStatus(clientIp)
		if banned {
			if adminRequest {
				SetAdminResponseHeaders(c)
			}
			response.Fail(c, http.StatusLocked, response.TranslateMsg(c, "Banned"))
			c.Abort()
			return
		}
		c.Next()
	}
}
