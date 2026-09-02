package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

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
	c.Header("Content-Disposition", "inline")
	httputil.SetLongPublic(c)
	c.Data(http.StatusOK, entity.Type, entity.Data)
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

	_, failure = policy.Validate(file.Filename, file.Size, "")
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

	header := make([]byte, 512)
	n, _ := io.ReadFull(src, header)
	if n > 0 {
		if !isValidImageContent(header[:n]) {
			c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageUploadInvalidImage, nil))
			return
		}
	}

	remainingData, err := io.ReadAll(io.LimitReader(src, policy.MaxSize-int64(n)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageUploadContentReadFailed, nil))
		return
	}

	fileData := make([]byte, n+len(remainingData))
	copy(fileData, header[:n])
	copy(fileData[n:], remainingData)

	if int64(len(fileData)) > policy.MaxSize {
		c.JSON(http.StatusBadRequest, component.FailDataCode(
			component.MessageUploadFileTooLarge,

			component.MessageParams{"maxSizeKb": policy.MaxSize / 1024}))
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

// isValidImageContent checks common image file signatures.
func isValidImageContent(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	var imageSignatures = [][]byte{
		{0xFF, 0xD8, 0xFF}, // JPEG
		{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG
		{0x47, 0x49, 0x46, 0x38, 0x37, 0x61},             // GIF87a
		{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},             // GIF89a
		{0x52, 0x49, 0x46, 0x46},                         // WebP (RIFF)
		{0x42, 0x4D},                                     // BMP
	}

	for _, signature := range imageSignatures {
		if len(data) >= len(signature) && bytes.HasPrefix(data, signature) {
			if bytes.HasPrefix(signature, []byte{0x52, 0x49, 0x46, 0x46}) {
				if len(data) >= 12 && bytes.Equal(data[8:12], []byte("WEBP")) {
					return true
				}
				continue
			}
			return true
		}
	}

	return false
}

func isAllowedExtension(ext string, allowedExts []string) bool {
	for _, allowedExt := range allowedExts {
		if strings.ToLower(allowedExt) == ext {
			return true
		}
	}
	return false
}
