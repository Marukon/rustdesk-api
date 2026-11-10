package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

// ClientUserAgentAuth 仅允许官方 PC (Windows 等) 和 Android 客户端的 User-Agent 访问客户端 API
func ClientUserAgentAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 1. 管理端 API (/api/admin/*) 由独立的 AdminGate 门禁保护，此处直接放行
		if strings.HasPrefix(path, "/api/admin") {
			c.Next()
			return
		}

		// 2. 本地回环地址内部通信放行
		clientIP := c.ClientIP()
		if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
			c.Next()
			return
		}

		ua := c.GetHeader("User-Agent")
		uaLower := strings.ToLower(strings.TrimSpace(ua))

		// 官方客户端 User-Agent 特征：
		// - 官方桌面端(Windows)与 Android 核心通信库: reqwest/...
		// - 官方 Android (Flutter/Dart) 通信库: dart/...
		// - 官方客户端自定义标识: 包含 rustdesk
		isOfficialClient := strings.Contains(uaLower, "reqwest") ||
			strings.Contains(uaLower, "dart") ||
			strings.Contains(uaLower, "rustdesk")

		if !isOfficialClient {
			global.Logger.Warn(fmt.Sprintf("Client UA blocked access on %s from %s (UA: %s)", path, clientIP, ua))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Forbidden",
			})
			return
		}

		c.Next()
	}
}
