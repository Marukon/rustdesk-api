package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

const (
	AdminGateCookieName = "admin_gate_token"
	AdminGateHeaderName = "X-Admin-Gate-Token"
	GateTokenTTL        = 7 * 24 * time.Hour
)

// GenerateGateToken 生成带 HMAC-SHA256 签名的门禁令牌
func GenerateGateToken(secret string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	msg := fmt.Sprintf("%s:%s", global.Config.Admin.Path, ts)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(msg))
	sig := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", ts, sig)
}

// ValidateGateToken 验证门禁令牌是否合法有效
func ValidateGateToken(token string, secret string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	tsStr, sig := parts[0], parts[1]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}

	tokenTime := time.Unix(ts, 0)
	if time.Since(tokenTime) > GateTokenTTL || time.Until(tokenTime) > 5*time.Minute {
		return false
	}

	msg := fmt.Sprintf("%s:%s", global.Config.Admin.Path, tsStr)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(msg))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

func GetGateSecret() string {
	if global.Config.Admin.Secret != "" {
		return global.Config.Admin.Secret
	}
	if global.Config.Jwt.Key != "" {
		return global.Config.Jwt.Key
	}
	return "rustdesk-admin-gate-default-secret-salt"
}

// AdminGateAuth 管理后台防爆破门禁验证中间件
func AdminGateAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if global.Config.Admin.Path == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Admin interface is disabled. Please set ADMIN_PATH environment variable.",
			})
			return
		}

		secret := GetGateSecret()

		token, err := c.Cookie(AdminGateCookieName)
		if err != nil || token == "" {
			token = c.GetHeader(AdminGateHeaderName)
		}

		if token == "" || !ValidateGateToken(token, secret) {
			global.Logger.Warn(fmt.Sprintf("Admin gate blocked unauthorized access from IP: %s to %s", c.ClientIP(), c.Request.RequestURI))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Forbidden: Admin access restricted",
			})
			return
		}

		c.Next()
	}
}
