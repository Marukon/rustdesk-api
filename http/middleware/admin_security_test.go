package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

func TestAdminEntrySecurity(t *testing.T) {
	old := global.Config
	t.Cleanup(func() { global.Config = old })
	global.Config = config.Config{Admin: config.Admin{Path: "/test-admin"}}

	gin.SetMode(gin.TestMode)
	g := gin.New()
	g.GET("/test-admin", func(c *gin.Context) {
		SetAdminResponseHeaders(c)
		SetAdminGateCookie(c)
		c.String(http.StatusOK, AdminClientIP(c))
	})
	for _, tc := range []struct {
		name, url, proxy, wantIP string
		secureCookie, wantSecure bool
	}{
		{"direct HTTP", "http://example.test/test-admin", "", "192.0.2.10", false, false},
		{"direct HTTPS", "https://example.test/test-admin", "", "192.0.2.10", false, true},
		{"HTTPS termination", "http://example.test/test-admin", "192.0.2.10", "198.51.100.4", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			global.Config.Admin.SecureCookie = tc.secureCookie
			global.Config.Gin.TrustProxy = tc.proxy
			if tc.proxy != "" {
				if err := g.SetTrustedProxies([]string{tc.proxy}); err != nil {
					t.Fatal(err)
				}
			}
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			req.RemoteAddr = "192.0.2.10:43110"
			req.Header.Set("X-Forwarded-For", "198.51.100.4")
			w := httptest.NewRecorder()
			g.ServeHTTP(w, req)
			if w.Body.String() != tc.wantIP || w.Header().Get("Cache-Control") != "no-store" ||
				w.Header().Get("Referrer-Policy") != "no-referrer" {
				t.Fatalf("unexpected admin response: %s, %v", w.Body.String(), w.Header())
			}
			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected one gate cookie, got %d", len(cookies))
			}
			gate := cookies[0]
			if gate.Name != AdminGateCookieName || !gate.HttpOnly || gate.SameSite != http.SameSiteLaxMode ||
				gate.Secure != tc.wantSecure || !ValidateGateToken(gate.Value, GetGateSecret()) {
				t.Fatal("invalid gate cookie or scope")
			}
		})
	}
}
