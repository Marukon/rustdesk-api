package admin

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/lib/upload"
	"os"
	"path/filepath"
	"time"
)

type File struct {
}

// OssToken 文件
// @Tags 文件
// @Summary 获取ossToken
// @Description 获取ossToken
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/file/oss_token [get]
// @Security token
func (f *File) OssToken(c *gin.Context) {
	token := global.Oss.GetPolicyToken("")
	response.Success(c, token)
}

type FileBack struct {
	upload.CallbackBaseForm
	Url string `json:"url"`
}

// Notify 上传成功后回调
func (f *File) Notify(c *gin.Context) {

	res := global.Oss.Verify(c.Request)
	if !res {
		response.Fail(c, 101, response.TranslateMsg(c, "NoAccess"))
		return
	}
	fm := &FileBack{}
	if err := c.ShouldBind(fm); err != nil {
		fmt.Println(err)
	}
	fm.Url = global.Config.Oss.Host + "/" + fm.Filename
	response.Success(c, fm)

}

// Upload 上传文件到本地
// @Tags 文件
// @Summary 上传文件到本地
// @Description 上传文件到本地
// @Accept  multipart/form-data
// @Produce  json
// @Param file formData file true "上传文件示例"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/file/upload [post]
// @Security token
func (f *File) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	baseFilename := filepath.Base(filepath.Clean(file.Filename))
	if baseFilename == "" || baseFilename == "." || baseFilename == "/" || baseFilename == "\\" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	dateDir := time.Now().Format("20060102")
	webPath := "/upload/" + dateDir + "/"
	dirPath := filepath.Join(global.Config.Gin.ResourcesPath, "upload", dateDir)
	dst := filepath.Join(dirPath, baseFilename)
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	// 上传文件至指定目录
	err = c.SaveUploadedFile(file, dst)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	// 返回文件web地址
	response.Success(c, gin.H{
		"url": webPath + baseFilename,
	})
}
