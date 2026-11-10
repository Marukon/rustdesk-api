package web

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
)

type Index struct {
}

func (i *Index) Index(c *gin.Context) {
	c.String(http.StatusForbidden, "403 Forbidden")
}

func (i *Index) AdminEntry(c *gin.Context) {
	adminPath := global.Config.Admin.Path
	if adminPath == "" {
		c.String(http.StatusForbidden, "403 Forbidden: Admin path not configured")
		return
	}

	secret := middleware.GetGateSecret()
	token := middleware.GenerateGateToken(secret)
	c.SetCookie(middleware.AdminGateCookieName, token, int(middleware.GateTokenTTL.Seconds()), "/", "", false, true)

	indexPath := filepath.Join(global.Config.Gin.ResourcesPath, "admin", "index.html")
	c.File(indexPath)
}

func (i *Index) ConfigJs(c *gin.Context) {
	apiServer := global.Config.Rustdesk.ApiServer
	magicQueryonline := global.Config.Rustdesk.WebclientMagicQueryonline
	tmp := fmt.Sprintf(`localStorage.setItem('api-server', '%v');
const ws2_prefix = 'wc-';
localStorage.setItem(ws2_prefix+'api-server', '%v');

window.webclient_magic_queryonline = %d;
window.ws_host = '%v';
`, apiServer, apiServer, magicQueryonline, global.Config.Rustdesk.WsHost)
	//	tmp := `
	//localStorage.setItem('api-server', "` + apiServer + `")
	//const ws2_prefix = 'wc-'
	//localStorage.setItem(ws2_prefix+'api-server', "` + apiServer + `")
	//
	//window.webclient_magic_queryonline = ` + magicQueryonline + ``

	c.Header("Content-Type", "application/javascript")
	c.String(200, tmp)
}
