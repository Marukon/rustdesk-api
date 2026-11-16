package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/sirupsen/logrus"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		uri := c.Request.URL.RequestURI()
		path := c.Request.URL.Path
		adminPath := global.Config.Admin.Path
		if adminPath != "" && (path == adminPath || strings.HasPrefix(path, adminPath+"/")) {
			uri = "[admin entry]"
		} else if strings.HasPrefix(path, "/api/admin/") || strings.HasPrefix(path, "/api/oidc/") || strings.HasPrefix(path, "/api/oauth/") {
			uri = path
		}
		global.Logger.WithFields(
			logrus.Fields{
				"uri":    uri,
				"ip":     c.ClientIP(),
				"method": c.Request.Method,
			}).Debug("Request")
		c.Next()
	}
}
