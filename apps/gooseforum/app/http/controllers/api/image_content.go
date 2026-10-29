package api

import (
	"errors"
	"io"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// uploadFailureError carries a ready failure envelope through the
// storageservice error contract, so a direct-upload complete returns exactly the
// same code as the multipart and avatar paths for the same input.
type uploadFailureError struct{ failure *imageUploadFailure }

func (e uploadFailureError) Error() string {
	return string(e.failure.Data.MessageCode)
}

// Unwrap makes storageservice treat the object as invalid and roll it back.
func (e uploadFailureError) Unwrap() error { return storageservice.ErrDirectUploadInvalidObject }

// detectUploadedImageContent 是四个上传出口共用的字节判定入口：类型只由文件头
// 决定，客户端声明的扩展名与 MIME 不参与。返回规范化 MIME 供存储名/入库使用。
func detectUploadedImageContent(reader io.Reader) (string, error) {
	return imagepolicy.DetectContentType(reader)
}

// imageContentFailureCode 把内容校验 sentinel 映射为稳定 messageCode：
// 空内容 / 已知不支持格式 / 其余字节不是可用图片（含非 sentinel 错误）。
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
