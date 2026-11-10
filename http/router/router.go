package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/controller/web"
)

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
		adminFS := http.Dir(global.Config.Gin.ResourcesPath + "/admin")
		fileServer := http.StripPrefix(adminPath, http.FileServer(adminFS))

		g.GET(adminPath, func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, adminPath+"/")
		})

		g.GET(adminPath+"/*filepath", func(c *gin.Context) {
			param := c.Param("filepath")
			if param == "" || param == "/" || param == "/index.html" {
				i.AdminEntry(c)
				return
			}
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	g.Any("/_admin/*any", func(c *gin.Context) {
		c.String(http.StatusForbidden, "403 Forbidden")
	})
	g.GET("/_admin", func(c *gin.Context) {
		c.String(http.StatusForbidden, "403 Forbidden")
	})
}
