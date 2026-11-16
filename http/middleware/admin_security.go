package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

// AdminClientIP only accepts forwarded addresses when Gin has a configured
// proxy allowlist. Keep the desktop client's existing IP policy separate.
func AdminClientIP(c *gin.Context) string {
	if global.Config.Gin.TrustProxy == "" {
		return c.RemoteIP()
	}
	return c.ClientIP()
}

func SetAdminResponseHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
}
