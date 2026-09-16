package api

import (
	"io"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
)

var errInvalidImageContent = imagepolicy.ErrInvalidImageContent

// Keep every upload entry on the same format validation policy.
func validateUploadedImage(reader io.Reader, expectedContentType string) error {
	return imagepolicy.ValidateContent(reader, expectedContentType)
}
