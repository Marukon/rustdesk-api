package router

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/controller/web"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
)

func getAdminResourcesDir() string {
	candidates := []string{
		filepath.Join(global.Config.Gin.ResourcesPath, "admin"),
		"/app/resources/admin",
		"./resources/admin",
		"resources/admin",
	}
	for _, dir := range candidates {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
		}
	}
	return filepath.Join(global.Config.Gin.ResourcesPath, "admin")
}

func WebInit(g *gin.Engine) {
	i := &web.Index{}
	g.GET("/", i.Index)

	if global.Config.App.WebClient == 1 {
		g.GET("/webclient-config/index.js", i.ConfigJs)
		g.StaticFS("/webclient", http.Dir(global.Config.Gin.ResourcesPath+"/web"))
		g.StaticFS("/webclient2", http.Dir(global.Config.Gin.ResourcesPath+"/web2"))
	}

	adminPath := global.Config.Admin.Path
	if adminPath != "" {
		adminDir := getAdminResourcesDir()
		global.Logger.Infof("[ROUTER] Admin secret path registered at: %s (resources dir: %s)", adminPath, adminDir)

		adminFS := http.Dir(adminDir)
		fileServer := http.StripPrefix(adminPath, http.FileServer(adminFS))

		serveAdmin := func(c *gin.Context) {
			secret := middleware.GetGateSecret()
			token := middleware.GenerateGateToken(secret)
			c.SetCookie(middleware.AdminGateCookieName, token, int(middleware.GateTokenTTL.Seconds()), "/", "", false, true)

			filepathParam := c.Param("filepath")
			if filepathParam == "" || filepathParam == "/" || filepathParam == "/index.html" {
				indexPath := filepath.Join(adminDir, "index.html")
				content, err := os.ReadFile(indexPath)
				if err != nil {
					global.Logger.Errorf("[ROUTER] Admin index.html not found at: %s (err: %v)", indexPath, err)
					c.String(http.StatusNotFound, "404 Admin frontend not found")
					return
				}
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.Data(http.StatusOK, "text/html; charset=utf-8", content)
				return
			}

			f, err := adminFS.Open(filepathParam)
			if err != nil {
				c.String(http.StatusNotFound, "404 not found")
				return
			}
			f.Close()

			fileServer.ServeHTTP(c.Writer, c.Request)
		}

		g.GET(adminPath, func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, adminPath+"/")
		})
		g.GET(adminPath+"/*filepath", serveAdmin)
		g.HEAD(adminPath+"/*filepath", serveAdmin)
	} else {
		global.Logger.Warn("[ROUTER] Admin secret path is not set (ADMIN_PATH or RUSTDESK_API_ADMIN_PATH is empty). Admin UI is disabled.")
	}

	g.Any("/_admin/*any", func(c *gin.Context) {
		c.String(http.StatusForbidden, "403 Forbidden")
	})
	g.GET("/_admin", func(c *gin.Context) {
		c.String(http.StatusForbidden, "403 Forbidden")
	})
}
