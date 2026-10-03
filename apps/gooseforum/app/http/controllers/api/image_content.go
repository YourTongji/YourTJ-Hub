package api

import (
	"errors"
	"io"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

var errInvalidImageContent = imagepolicy.ErrInvalidImageContent

// Keep every upload entry on the same format validation policy.
func validateUploadedImage(reader io.Reader, expectedContentType string) error {
	return imagepolicy.ValidateContent(reader, expectedContentType)
}

// imageContentViolation 报告 err 是否来自图片内容校验（空内容 / 已知不支持格式 /
// 内容与扩展名不符），供直传路径决定是否按「对象内容无效」回滚。
func imageContentViolation(err error) bool {
	return errors.Is(err, imagepolicy.ErrEmptyImageContent) ||
		errors.Is(err, imagepolicy.ErrUnsupportedImageFormat) ||
		errors.Is(err, imagepolicy.ErrInvalidImageContent)
}

// imageContentFailureCode 把内容校验 sentinel 映射为稳定 messageCode：
// 空内容 / 已知不支持格式 / 其余不匹配或解码失败（含非 sentinel 错误）。
func imageContentFailureCode(err error) component.MessageCode {
	switch {
	case errors.Is(err, imagepolicy.ErrEmptyImageContent):
		return component.MessageUploadEmptyImage
	case errors.Is(err, imagepolicy.ErrUnsupportedImageFormat):
		return component.MessageUploadUnsupportedImage
	default:
		return component.MessageUploadInvalidImage
	}
}
