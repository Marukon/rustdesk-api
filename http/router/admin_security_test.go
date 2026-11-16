package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/controller/api"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAdminAuthenticationBoundary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.UserToken{}, &model.LoginLog{}); err != nil {
		t.Fatal(err)
	}

	oldConfig, oldLogger, oldLimiter := global.Config, global.Logger, global.LoginLimiter
	oldLocalizer, oldValidator := global.Localizer, global.Validator.ValidStruct
	oldService, oldDB, oldServiceConfig, oldServiceJwt := service.AllService, service.DB, service.Config, service.Jwt
	t.Cleanup(func() {
		global.Config, global.Logger, global.LoginLimiter = oldConfig, oldLogger, oldLimiter
		global.Localizer, global.Validator.ValidStruct = oldLocalizer, oldValidator
		service.AllService, service.DB, service.Config, service.Jwt = oldService, oldDB, oldServiceConfig, oldServiceJwt
	})
	global.Config = config.Config{
		Admin: config.Admin{Path: "/test-admin", Secret: "test-only-gate-key"},
		App:   config.App{Register: true, TokenExpire: time.Hour},
	}
	global.Logger = logrus.New()
	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{CaptchaThreshold: -1})
	bundle := i18n.NewBundle(language.English)
	for _, id := range []string{"NoAccess", "NeedLogin", "OauthExpired", "UserDisabled"} {
		if err := bundle.AddMessages(language.English, &i18n.Message{ID: id, Other: id}); err != nil {
			t.Fatal(err)
		}
	}
	global.Localizer = func(string) *i18n.Localizer { return i18n.NewLocalizer(bundle, "en") }
	global.Validator.ValidStruct = func(*gin.Context, interface{}) []string { return nil }
	service.DB, service.Config, service.Jwt = db, &global.Config, &jwt.Jwt{}
	service.AllService = &service.Service{
		UserService: &service.UserService{}, OauthService: &service.OauthService{}, LdapService: &service.LdapService{},
	}

	const password = "test-only-password"
	admin, regular, disabled := model.User{Username: "operator", Status: model.COMMON_STATUS_ENABLE},
		model.User{Username: "member", Status: model.COMMON_STATUS_ENABLE},
		model.User{Username: "disabled", Status: model.COMMON_STATUS_DISABLED}
	isAdmin, isRegular := true, false
	admin.IsAdmin, regular.IsAdmin, disabled.IsAdmin = &isAdmin, &isRegular, &isAdmin
	passwordHash, err := utils.EncryptPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []*model.User{&admin, &regular, &disabled} {
		user.Password = passwordHash
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	gin.SetMode(gin.TestMode)
	g := gin.New()
	Init(g)
	g.POST("/api/login", (&api.Login{}).Login)
	g.GET("/api/oidc/auth-query", (&api.Oauth{}).OidcAuthQuery)
	gate := middleware.GenerateGateToken(global.Config.Admin.Secret)
	request := func(method, path, token, body string) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("api-token", token)
		req.AddCookie(&http.Cookie{Name: middleware.AdminGateCookieName, Value: gate})
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
		var data map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("%s %s: HTTP %d: %s", method, path, w.Code, w.Body.String())
		}
		return w, data
	}
	message := func(data map[string]any, expected string) {
		t.Helper()
		if data["message"] != expected {
			t.Fatalf("wanted %q, got %v", expected, data)
		}
	}
	loginBody := func(username string) string {
		return `{"username":"` + username + `","password":"` + password + `","deviceInfo":{"type":"app"}}`
	}

	w, data := request("POST", "/api/admin/login", "", loginBody(regular.Username))
	message(data, "NoAccess")
	if data["data"] != nil || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("ordinary account response: %v, headers: %v", data, w.Header())
	}
	_, data = request("POST", "/api/admin/login", "", loginBody(disabled.Username))
	message(data, "UserDisabled")
	_, data = request("POST", "/api/login", "", loginBody(regular.Username))
	desktopToken, _ := data["access_token"].(string)
	if desktopToken == "" {
		t.Fatalf("ordinary desktop login: %v", data)
	}
	_, data = request("GET", "/api/admin/user/current", desktopToken, "")
	message(data, "NoAccess")
	_, data = request("POST", "/api/admin/login", "", loginBody(admin.Username))
	message(data, "success")
	adminToken, _ := data["data"].(map[string]any)["token"].(string)
	if len(adminToken) != 32 {
		t.Fatal("opaque token no longer has the original format")
	}
	_, data = request("GET", "/api/admin/user/current", adminToken, "")
	message(data, "success")
	if err := db.Model(&admin).Update("is_admin", false).Error; err != nil {
		t.Fatal(err)
	}
	_, data = request("GET", "/api/admin/user/current", adminToken, "")
	message(data, "NoAccess")
	if err := db.Model(&admin).Update("is_admin", true).Error; err != nil {
		t.Fatal(err)
	}
	_, data = request("POST", "/api/admin/logout", adminToken, "")
	message(data, "success")
	_, data = request("GET", "/api/admin/user/current", adminToken, "")
	message(data, "NeedLogin")
	_, data = request("POST", "/api/admin/logout", adminToken, "")
	message(data, "success")

	_, data = request("POST", "/api/admin/user/register", "", loginBody("newmember"))
	message(data, "success")
	if payload, _ := data["data"].(map[string]any); payload["registered"] != true || payload["token"] != nil {
		t.Fatalf("registration must not issue a token: %v", data)
	}

	oauth := service.AllService.OauthService
	for _, item := range []struct {
		key   string
		id    uint
		admin bool
	}{
		{"admin-oidc", admin.Id, true},
		{"ordinary-admin-oidc", regular.Id, true},
		{"ordinary-desktop-oidc", regular.Id, false},
	} {
		oauth.SetOauthCache(item.key, &service.OauthCacheItem{
			UserId: item.id, AdminLogin: item.admin, Action: service.OauthActionTypeLogin,
		}, 0)
		t.Cleanup(func() { oauth.DeleteOauthCache(item.key) })
	}
	_, data = request("GET", "/api/oidc/auth-query?code=admin-oidc", "", "")
	if data["error"] != "OauthExpired" {
		t.Fatalf("admin OIDC code accepted by desktop endpoint: %v", data)
	}
	_, data = request("GET", "/api/admin/oidc/auth-query?code=ordinary-desktop-oidc", "", "")
	if data["error"] != "OauthExpired" {
		t.Fatalf("desktop OIDC code accepted by admin endpoint: %v", data)
	}
	_, data = request("GET", "/api/admin/oidc/auth-query?code=ordinary-admin-oidc", "", "")
	message(data, "NoAccess")
	_, data = request("GET", "/api/oidc/auth-query?code=ordinary-desktop-oidc", "", "")
	if data["access_token"] == nil {
		t.Fatalf("ordinary desktop OIDC failed: %v", data)
	}
	_, data = request("GET", "/api/admin/oidc/auth-query?code=admin-oidc", "", "")
	message(data, "success")
	_, data = request("GET", "/api/admin/oidc/auth-query?code=admin-oidc", "", "")
	if data["error"] != "OauthExpired" {
		t.Fatalf("consumed OIDC code should expire: %v", data)
	}
}
