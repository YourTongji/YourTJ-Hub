package api

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

type imageUploadPolicy struct {
	MaxSize     int64
	AllowedExts []string
}

type imageUploadFailure struct {
	Status int
	Data   component.ResultStruct
}

// resolveImageUploadPolicy applies the shared upload gate (auth, role,
// permission, cooldown, daily limit) and computes the effective size limit.
// It is used by both the multipart upload and the direct upload init path.
func resolveImageUploadPolicy(userId uint64) (*imageUploadPolicy, *imageUploadFailure) {
	if userId == 0 {
		return nil, uploadFailure(http.StatusUnauthorized, component.MessageAuthRequired, nil)
	}
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()
	userEntity, _ := users.Get(userId)
	isRoleUser := userEntity.RoleId > 0
	if !isRoleUser && !postingConfig.UploadControl.AllowAttachments {
		return nil, uploadFailure(http.StatusForbidden, component.MessageUploadAttachmentDisabled, nil)
	}
	if status, err := component.CheckUserPermission(&userEntity, component.PermissionActionUploadAttachment); err != nil {
		return nil, &imageUploadFailure{Status: status, Data: component.FailDataError(err)}
	}
	if !isRoleUser && postingConfig.UploadControl.NewUserUploadCooldownMinutes > 0 {
		cooldownTime := userEntity.CreatedAt.Add(time.Duration(postingConfig.UploadControl.NewUserUploadCooldownMinutes) * time.Minute)
		if time.Now().Before(cooldownTime) {
			return nil, uploadFailure(http.StatusBadRequest, component.MessageUploadCooldown, component.MessageParams{
				"minutes":     postingConfig.UploadControl.NewUserUploadCooldownMinutes,
				"availableAt": cooldownTime.Format(time.RFC3339),
			})
		}
	}
	if !isRoleUser && postingConfig.UploadControl.MaxDailyUploadsPerUser > 0 {
		count := filedata.CountDailyUploads(userId)
		if count >= int64(postingConfig.UploadControl.MaxDailyUploadsPerUser) {
			return nil, uploadFailure(http.StatusBadRequest, component.MessageUploadDailyLimit, component.MessageParams{"count": count})
		}
	}
	maxSize := int64(filedata.MaxFileSize)
	configMaxSize := int64(postingConfig.UploadControl.MaxAttachmentSizeKb) * 1024
	if !isRoleUser && configMaxSize > 0 && configMaxSize < maxSize {
		maxSize = configMaxSize
	}
	// 生效 allowlist 永远只可能是内置图片扩展集合（imagepolicy）的子集：
	// 配置里即使残留危险/非法项，也会在读取路径被过滤，空配置回退内置全集。
	return &imageUploadPolicy{MaxSize: maxSize, AllowedExts: imagepolicy.EffectiveAllowedExtensions(postingConfig.UploadControl.AuthorizedExtensions)}, nil
}

// Validate checks filename, size, extension and reported content type, and
// returns the canonical image content type for the file.
func (policy imageUploadPolicy) Validate(filename string, size int64, reportedContentType string) (string, *imageUploadFailure) {
	if strings.TrimSpace(filename) == "" {
		return "", uploadFailure(http.StatusBadRequest, component.MessageUploadFilenameRequired, nil)
	}
	if size <= 0 {
		return "", uploadFailure(http.StatusBadRequest, component.MessageUploadInvalidImage, nil)
	}
	if size > policy.MaxSize {
		return "", uploadFailure(http.StatusBadRequest, component.MessageUploadFileTooLarge, component.MessageParams{"maxSizeKb": policy.MaxSize / 1024})
	}
	if !imagepolicy.IsAllowedExt(filepath.Ext(filename), policy.AllowedExts) {
		return "", uploadFailure(http.StatusBadRequest, component.MessageUploadUnsupportedExt, component.MessageParams{"extensions": strings.Join(policy.AllowedExts, ", ")})
	}
	// IsAllowedExt 已保证扩展名属于生效 allowlist（内置图片集合的子集），
	// CheckImageType 在此不可能失败。
	contentType, _ := filedata.CheckImageType(filename)
	if reportedContentType != "" {
		reported, _, err := mime.ParseMediaType(reportedContentType)
		if err != nil || !strings.EqualFold(reported, contentType) {
			return "", uploadFailure(http.StatusBadRequest, component.MessageUploadInvalidImage, nil)
		}
	}
	return contentType, nil
}

func uploadFailure(status int, code component.MessageCode, params component.MessageParams) *imageUploadFailure {
	return &imageUploadFailure{Status: status, Data: component.FailDataCode(code, params)}
}
