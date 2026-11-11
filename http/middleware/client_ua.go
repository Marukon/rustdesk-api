package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

// ClientUserAgentAuth 保护 /api 接口：
// 1. 严格阻断常规网页浏览器（Chrome/Edge/Safari/Firefox/WebKit 等）直接探测
// 2. 严格阻断自动化漏洞扫描与抓取工具（curl/python/postman/sqlmap 等）
// 3. 放行官方客户端（桌面端 Windows 与移动端 Android，包含底层 Rust reqwest/Dart/系统通信库）
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

		// 3. 拦截浏览器特征标头（Sec-* 头部）
		secChUa := c.GetHeader("Sec-Ch-Ua")
		secFetchMode := c.GetHeader("Sec-Fetch-Mode")
		secFetchDest := c.GetHeader("Sec-Fetch-Dest")
		if secChUa != "" || secFetchMode != "" || secFetchDest == "document" {
			global.Logger.Warn(fmt.Sprintf("[CLIENT-AUTH] Blocked browser request on %s from %s (UA: '%s', Sec-Ch-Ua: '%s')", path, clientIP, ua, secChUa))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Forbidden",
			})
			return
		}

		// 4. 浏览器 User-Agent 关键词黑名单拦截
		browserKeywords := []string{
			"mozilla", "chrome", "safari", "firefox", "edge", "edg/", "webkit", "opera", "opr/", "msie", "trident",
		}
		for _, kw := range browserKeywords {
			if strings.Contains(uaLower, kw) {
				global.Logger.Warn(fmt.Sprintf("[CLIENT-AUTH] Blocked browser UA on %s from %s (UA: '%s')", path, clientIP, ua))
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Forbidden",
				})
				return
			}
		}

		// 5. 常见扫描工具 / 爬虫黑名单拦截
		scannerKeywords := []string{
			"curl", "wget", "python", "postman", "httpie", "sqlmap", "nikto", "nmap", "masscan", "httpx", "go-http-client",
		}
		for _, kw := range scannerKeywords {
			if strings.Contains(uaLower, kw) {
				global.Logger.Warn(fmt.Sprintf("[CLIENT-AUTH] Blocked scanner/tool UA on %s from %s (UA: '%s')", path, clientIP, ua))
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Forbidden",
				})
				return
			}
		}

		// 6. 官方客户端放行逻辑：
		// 官方 RustDesk 客户端（无论是 Windows 桌面端还是 Android 移动端）：
		// - 部分版本/请求使用底层 Rust reqwest 发起，未设置 User-Agent 请求头，表现为空字符串 ""
		// - 部分版本携带 "reqwest", "dart", "rustdesk"
		// - Android 原生组件可能携带 "okhttp" 或 "dalvik"
		isOfficialClient := uaLower == "" ||
			strings.Contains(uaLower, "reqwest") ||
			strings.Contains(uaLower, "dart") ||
			strings.Contains(uaLower, "rustdesk") ||
			strings.Contains(uaLower, "okhttp") ||
			strings.Contains(uaLower, "dalvik")

		if !isOfficialClient {
			global.Logger.Warn(fmt.Sprintf("[CLIENT-AUTH] Blocked unknown UA on %s from %s (UA: '%s')", path, clientIP, ua))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Forbidden",
			})
			return
		}

		global.Logger.Infof("[CLIENT-AUTH] Allowed %s %s from %s (UA: '%s')", c.Request.Method, path, clientIP, ua)
		c.Next()
	}
}
