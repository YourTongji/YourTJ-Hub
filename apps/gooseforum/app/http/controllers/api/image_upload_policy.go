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
	return &imageUploadPolicy{MaxSize: maxSize, AllowedExts: effectiveUploadExtensions()}, nil
}

// effectiveUploadExtensions 是读取生效 allowlist 的唯一入口：配置列表的规范子集为空
// 时回退内置全集，危险扩展永远进不来。
func effectiveUploadExtensions() []string {
	return imagepolicy.EffectiveAllowedExtensions(hotdataserve.GetPostingSettingsConfigCache().UploadControl.AuthorizedExtensions)
}

// validateDeclaredFilename 校验客户端声明的文件名与大小（不变的上传前置）。
func (policy imageUploadPolicy) validateDeclaredFilename(filename string, size int64) *imageUploadFailure {
	if strings.TrimSpace(filename) == "" {
		return uploadFailure(http.StatusBadRequest, component.MessageUploadFilenameRequired, nil)
	}
	if size <= 0 {
		return uploadFailure(http.StatusBadRequest, component.MessageUploadEmptyImage, nil)
	}
	if size > policy.MaxSize {
		return uploadFailure(http.StatusBadRequest, component.MessageUploadFileTooLarge, component.MessageParams{"maxSizeKb": policy.MaxSize / 1024})
	}
	return nil
}

// validateDeclaredExtension 仍按客户端文件名扩展名做 allowlist 前置拒绝：
// 危险扩展（evil.svg 等）在读取内容之前就被拒，安全强度不因「字节权威」回退。
func (policy imageUploadPolicy) validateDeclaredExtension(filename string) *imageUploadFailure {
	if imagepolicy.IsAllowedExt(filepath.Ext(filename), policy.AllowedExts) {
		return nil
	}
	return policy.extensionFailure()
}

// validateReportedContentType 只在尚无字节的 init 阶段使用：声明 MIME 必须与声明
// 扩展名推出的类型一致；此时没有字节可依据，仍按同一允许集预检。
func (policy imageUploadPolicy) validateReportedContentType(filename, reportedContentType string) *imageUploadFailure {
	if reportedContentType == "" {
		return nil
	}
	declared, ok := imagepolicy.ContentTypeForFilename(filename)
	if !ok {
		return policy.extensionFailure()
	}
	reported, _, err := mime.ParseMediaType(reportedContentType)
	if err != nil || !strings.EqualFold(reported, declared) {
		return uploadFailure(http.StatusBadRequest, component.MessageUploadInvalidImage, nil)
	}
	return nil
}

// validateDetectedContentType 是内容校验后的唯一判定：按字节判定的真实类型必须落在
// 生效 allowlist 内。真实类型与声明扩展名冲突本身不再算失败（issue #1393）。判定按
// 类型而非规范扩展名，所以只配 `.jpeg` 时真实 JPEG 仍通过；入库/命名的规范扩展名
// （ExtForContentType → `.jpg`）与 allowlist 解耦。
func (policy imageUploadPolicy) validateDetectedContentType(contentType string) *imageUploadFailure {
	if imagepolicy.IsAllowedContentType(contentType, policy.AllowedExts) {
		return nil
	}
	return policy.extensionFailure()
}

func (policy imageUploadPolicy) extensionFailure() *imageUploadFailure {
	return uploadFailure(http.StatusBadRequest, component.MessageUploadUnsupportedExt,
		component.MessageParams{"extensions": strings.Join(policy.AllowedExts, ", ")})
}

func uploadFailure(status int, code component.MessageCode, params component.MessageParams) *imageUploadFailure {
	return &imageUploadFailure{Status: status, Data: component.FailDataCode(code, params)}
}
