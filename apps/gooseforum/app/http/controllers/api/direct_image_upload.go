package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"github.com/gin-gonic/gin"
)

type directImageUploadInitRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type directImageUploadCompleteRequest struct {
	Name string `json:"name"`
}

type directImageUploadInitResult struct {
	Mode   string                       `json:"mode"`
	Name   string                       `json:"name,omitempty"`
	Upload *storageservice.DirectUpload `json:"upload,omitempty"`
}

// InitDirectImageUpload starts a direct upload: when the active provider
// supports presigned uploads it returns the presigned POST payload, otherwise
// it reports proxy mode so the client falls back to the multipart path.
func InitDirectImageUpload(c *gin.Context) {
	var request directImageUploadInitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestParseFailed, component.MessageParams{"error": err.Error()}))
		return
	}
	if !storageservice.SupportsDirectUpload() {
		c.JSON(http.StatusOK, component.SuccessDataCode(directImageUploadInitResult{Mode: "proxy"}, component.MessageOperationSuccess, nil))
		return
	}
	userId := c.GetUint64("userId")
	policy, failure := resolveImageUploadPolicy(userId)
	if failure != nil {
		// 业务失败走 200 信封（仓库约定：400 仅留给请求解析失败）。
		c.JSON(http.StatusOK, failure.Data)
		return
	}
	contentType, failure := policy.Validate(request.Filename, request.Size, request.ContentType)
	if failure != nil {
		c.JSON(http.StatusOK, failure.Data)
		return
	}
	name := storageservice.NewUploadName(request.Filename, time.Now().Format("2006/01/02"))
	session, err := storageservice.BeginDirectUpload(c.Request.Context(), storageservice.DirectUploadRequest{
		Name: name, ContentType: contentType, Size: request.Size, UserId: userId,
	})
	if err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageUploadSaveFailed, component.MessageParams{"error": err.Error()}))
		return
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(directImageUploadInitResult{
		Mode: "direct", Name: session.Metadata.Name, Upload: &session.Upload,
	}, component.MessageOperationSuccess, nil))
}

// CompleteDirectImageUpload verifies the uploaded object, marks the row ready
// and records the upload owner. On owner-usage failure the object is deleted.
func CompleteDirectImageUpload(c *gin.Context) {
	var request directImageUploadCompleteRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Name == "" {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	userId := c.GetUint64("userId")
	metadata, err := storageservice.CompleteDirectUpload(c.Request.Context(), storageservice.CompleteDirectUploadRequest{
		Name: request.Name, UserId: userId,
		Validator: func(reader io.Reader, contentType string) error {
			err := validateUploadedImage(reader, contentType)
			if errors.Is(err, errInvalidImageContent) {
				return errors.Join(storageservice.ErrDirectUploadInvalidObject, err)
			}
			return err
		},
	})
	if err != nil {
		writeDirectUploadError(c, err)
		return
	}
	if err := fileusageservice.AddUploadOwner(userId, metadata.Name); err != nil {
		// 失败时同时删除对象与 ready 元数据行，避免遗留无对象的孤儿行。
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
		cleanupErr := storageservice.DeleteDirectUpload(cleanupCtx, metadata.Name)
		cancel()
		if cleanupErr != nil {
			slog.Error("delete direct upload after owner usage failure", "fileName", metadata.Name, "err", cleanupErr)
		}
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageUploadSaveFailed, component.MessageParams{"error": err.Error()}))
		return
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(map[string]any{
		"url": storageservice.PublicAccessPath(metadata.Name), "filename": metadata.Name, "size": metadata.Size,
	}, component.MessageUploadSuccess, nil))
}

// AbortDirectImageUpload cancels a pending direct upload, removing the object
// and the pending row.
func AbortDirectImageUpload(c *gin.Context) {
	var request directImageUploadCompleteRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Name == "" {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	err := storageservice.AbortDirectUpload(c.Request.Context(), storageservice.CompleteDirectUploadRequest{
		Name: request.Name, UserId: c.GetUint64("userId"),
	})
	if err != nil {
		writeDirectUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(true, component.MessageOperationSuccess, nil))
}

func writeDirectUploadError(c *gin.Context, err error) {
	// 业务失败统一走 200 信封（仓库约定）：未知/越权 name 返回 page.notFound，
	// 伪造对象返回 upload.image.invalidContent，存储失败返回 upload.saveFailed。
	// 400 仅保留给请求解析失败（ShouldBindJSON）。
	switch {
	case errors.Is(err, storageservice.ErrDirectUploadMetadataNotFound):
		c.JSON(http.StatusOK, component.FailDataCode(component.MessagePageNotFound, nil))
	case errors.Is(err, storageservice.ErrDirectUploadOwnerMismatch):
		c.JSON(http.StatusOK, component.FailDataCode(component.MessagePageNotFound, nil))
	case errors.Is(err, storageservice.ErrDirectUploadInvalidObject):
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageUploadInvalidImage, nil))
	case errors.Is(err, storageservice.ErrDirectUploadUnsupported):
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageUploadSaveFailed, component.MessageParams{"error": err.Error()}))
	default:
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageUploadSaveFailed, component.MessageParams{"error": err.Error()}))
	}
}
