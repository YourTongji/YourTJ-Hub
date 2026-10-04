package api

import (
	"bytes"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/httputil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/authsessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
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
	referenceName := filedata.ReferenceName(filename)
	privatePreview := false
	if fileusageservice.HasAnyReferences(referenceName) && !fileusageservice.HasActiveReferences(referenceName) {
		// 待审内容（issue #975）的图片对匿名与其他用户 fail-closed；上传者本人与
		// 有审核权限的站点管理员可授权预览（不进入任何共享缓存）。
		privatePreview = fileusageservice.HasPendingReferences(referenceName) && canPreviewPendingFile(c, referenceName)
		if !privatePreview {
			c.JSON(http.StatusNotFound, gin.H{
				"error":       "File not found",
				"messageCode": component.MessagePageNotFound,
			})
			return
		}
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
		base := path.Base(filename)
		if base == "." || base == "/" {
			base = "download"
		}
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": base}))
	}
	if privatePreview {
		c.Header("Cache-Control", "private, no-store")
	} else {
		httputil.SetLongPublic(c)
	}
	c.Data(http.StatusOK, contentType, entity.Data)
}

// canPreviewPendingFile 待审图片的授权预览：仅在文件已处于待审状态时才解析会话
// （不刷新令牌、不记活跃），普通公开图片读取路径不受影响。
func canPreviewPendingFile(c *gin.Context, referenceName string) bool {
	userID, _, _, ok := authsessionservice.ValidateToken(jwtopt.GetGinAccessToken(c))
	if !ok || userID == 0 {
		return false
	}
	if owner := filedata.GetByName(referenceName); owner.Id != 0 && owner.UserId == userID {
		return true
	}
	roleID, ok := userservice.GetUserRoleId(userID)
	if ok && permission.CheckRole(roleID, permission.SiteManager) {
		return true
	}
	if moderationservice.IsAdmin(userID) {
		return true
	}
	// 前台版主工作台同样审核待审内容（issue #975）：版主只能预览管辖分类内的
	// 待审图片，沿待审引用回溯到所属主题的分类逐一校验。
	for _, usage := range fileusageservice.ListPendingReferences(referenceName) {
		if categoryIDs := pendingUsageCategories(usage); len(categoryIDs) > 0 &&
			moderationservice.CanModerateAnyCategory(userID, categoryIDs) {
			return true
		}
	}
	return false
}

// pendingUsageCategories 返回待审引用所属主题的分类；无法回溯到主题时返回 nil。
func pendingUsageCategories(usage fileUsage.Entity) []uint64 {
	topicID := usage.TargetId
	switch usage.TargetType {
	case fileUsage.TargetTopic:
	case fileUsage.TargetPostRevision:
		revision := postRevisions.Get(usage.TargetId)
		post := posts.Get(revision.PostId)
		if post.VisibilityStatus != posts.VisibilityActive {
			return nil
		}
		if post.PostNo == 1 {
			return revision.CategoryIds
		}
		topicID = post.TopicId
	case fileUsage.TargetPost:
		topicID = posts.Get(usage.TargetId).TopicId
	default:
		return nil
	}
	if topicID == 0 {
		return nil
	}
	return topics.Get(topicID).CategoryIds
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
		c.JSON(http.StatusBadRequest, component.FailDataCode(imageContentFailureCode(err), nil))
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
	media, mediaErr := filedata.ProcessUploadedImage(entity.Name, fileData)
	if mediaErr != nil {
		slog.Warn("process uploaded image variants failed", "fileName", entity.Name, "error", mediaErr)
	}
	if adminUpload {
		fileusageservice.AddAdminUpload(userId, entity.Name)
	}

	result := map[string]any{
		"url":      entity.GetAccessPath(),
		"filename": file.Filename,
		"size":     len(fileData),
	}
	if media.Width > 0 && media.Height > 0 {
		result["imageMetadata"] = media
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(result, component.MessageUploadSuccess, nil))
}
