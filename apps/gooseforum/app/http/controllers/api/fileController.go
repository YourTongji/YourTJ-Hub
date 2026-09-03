package api

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/httputil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/gin-gonic/gin"
)

func GetFileByFileName(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Invalid filename",
			"messageCode": component.MessageRequestInvalidParams,
		})
		return
	}
	filename = strings.TrimPrefix(filename, "/")
	// 附件引用被标记为 RECOVERING（内容删除后 30 天窗口）或 PURGED 时不再允许公开下载。
	// 已删除内容的附件只应在恢复（回 ACTIVE）后重新可见；RECOVERING 只是为清理协调保留引用，
	// 不构成公开访问授权。
	if fileusageservice.HasAnyReferences(filename) && !fileusageservice.HasActiveReferences(filename) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":       "File not found",
			"messageCode": component.MessagePageNotFound,
		})
		return
	}

	entity, err := filedata.GetFileByName(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":       "File not found",
			"messageCode": component.MessagePageNotFound,
		})
		return
	}
	// 响应类型由存储对象名的规范化扩展名权威决定（issue #408），不采信
	// 客户端声明或行内 assert_type——合法图片对象名必然带可映射扩展名。
	contentType, ok := imagepolicy.ContentTypeForFilename(filename)
	if !ok {
		// 未知/危险对象（历史残留、无扩展名等）：octet-stream + 附件下载，
		// 绝不按行内类型内联渲染；nosniff 兜底防类型混淆。
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("X-Content-Type-Options", "nosniff")
	if strings.HasPrefix(contentType, "image/") {
		c.Header("Content-Disposition", "inline")
	} else {
		base := strings.ReplaceAll(path.Base(filename), `"`, "")
		if base == "." || base == "/" {
			base = "download"
		}
		c.Header("Content-Disposition", `attachment; filename="`+base+`"`)
	}
	httputil.SetLongPublic(c)
	c.Data(http.StatusOK, contentType, entity.Data)
}

// SaveImgByGinContext handles image uploads with size and content checks.
func SaveImgByGinContext(c *gin.Context) {
	saveImgByGinContext(c, false)
}

func SaveAdminImgByGinContext(c *gin.Context) {
	saveImgByGinContext(c, true)
}

func saveImgByGinContext(c *gin.Context, adminUpload bool) {
	userId := c.GetUint64(`userId`)
	policy, failure := resolveImageUploadPolicy(userId)
	if failure != nil {
		c.JSON(failure.Status, failure.Data)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageUploadFileMissing, nil))
		return
	}

	contentType, failure := policy.Validate(file.Filename, file.Size, "")
	if failure != nil {
		c.JSON(failure.Status, failure.Data)
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageUploadReadFailed, nil))
		return
	}
	defer func() { _ = src.Close() }()

	fileData, err := io.ReadAll(io.LimitReader(src, policy.MaxSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageUploadContentReadFailed, nil))
		return
	}
	if int64(len(fileData)) > policy.MaxSize {
		c.JSON(http.StatusBadRequest, component.FailDataCode(
			component.MessageUploadFileTooLarge,
			component.MessageParams{"maxSizeKb": policy.MaxSize / 1024}))
		return
	}
	// 内容校验与直传完成同口径：sniff 类型 + 解码格式都必须与扩展名推出的类型一致，
	// 伪造 MIME/扩展与字节不符在此拒绝，错误只回稳定 messageCode，不回解析细节。
	if err := validateUploadedImage(bytes.NewReader(fileData), contentType); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageUploadInvalidImage, nil))
		return
	}

	folderName := time.Now().Format("2006/01/02")

	entity, err := filedata.SaveFileFromUpload(userId, fileData, file.Filename, folderName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(
			component.MessageUploadSaveFailed,

			component.MessageParams{"error": err.Error()}))
		return
	}
	if adminUpload {
		fileusageservice.AddAdminUpload(userId, entity.Name)
	}

	c.JSON(http.StatusOK, component.SuccessDataCode(map[string]any{
		"url":      entity.GetAccessPath(),
		"filename": file.Filename,
		"size":     len(fileData),
	}, component.MessageUploadSuccess, nil))
}
