package storageservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/google/uuid"
)

var (
	ErrDirectUploadUnsupported   = errors.New("file storage does not support direct uploads")
	ErrDirectUploadOwnerMismatch = errors.New("pending upload does not belong to user")
	ErrDirectUploadInvalidObject = errors.New("uploaded object failed validation")
	// ErrDirectUploadMetadataNotFound marks a complete/abort whose name has no
	// matching metadata row (pending or ready). It maps to 404 so unknown names
	// do not leak object existence the way a 500 would.
	ErrDirectUploadMetadataNotFound = errors.New("direct upload metadata not found")
)

// DirectUploadRequest describes the object a browser will upload straight to
// the storage provider.
type DirectUploadRequest struct {
	Name        string
	ContentType string
	Size        int64
	UserId      uint64
	ExpiresIn   time.Duration
}

// DirectUpload is the presigned POST payload handed to the browser.
type DirectUpload struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

// DirectUploadSession pairs the pending metadata row with the presigned upload.
type DirectUploadSession struct {
	Metadata FileMetadata
	Upload   DirectUpload
}

// CompleteDirectUploadRequest identifies the pending upload being completed or
// aborted. Validator, when set, reads the uploaded object and returns the
// byte-authoritative content type before the row is marked ready.
type CompleteDirectUploadRequest struct {
	Name      string
	UserId    uint64
	Validator func(io.Reader) (contentType string, err error)
}

// FileMetadata is the subset of file metadata rows the direct upload lifecycle
// needs. It mirrors the file model entity without importing it (the file model
// package already imports storageservice).
type FileMetadata struct {
	Id        uint64
	Name      string
	Type      string
	Size      int64
	UserId    uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DirectUploadMetadataStore abstracts the pending/ready file metadata rows used
// by the direct upload lifecycle. Implemented by the file model package and
// registered through RegisterDirectUploadMetadataStore to avoid an import cycle.
type DirectUploadMetadataStore interface {
	CreateFileMetadata(ctx context.Context, userId uint64, name, fileType string, size int64) (*FileMetadata, error)
	GetPendingFileMetadataByName(ctx context.Context, name string) (*FileMetadata, error)
	GetFileMetadataByName(ctx context.Context, name string) (*FileMetadata, error)
	// RenameFileMetadata rewrites a pending row's object name and content type
	// after the byte-detected type forced a rename.
	RenameFileMetadata(ctx context.Context, oldName, newName, fileType string) (*FileMetadata, error)
	MarkFileReady(ctx context.Context, name string) (*FileMetadata, error)
	ListPendingFilesBefore(ctx context.Context, before time.Time, limit int) ([]FileMetadata, error)
	DeleteByName(ctx context.Context, name string) error
}

// DirectUploadProvider is an optional capability implemented by remote stores.
type DirectUploadProvider interface {
	PresignUpload(context.Context, DirectUploadRequest) (*DirectUpload, error)
	VerifyUpload(context.Context, DirectUploadRequest) error
}

var directUploadMetadataStore DirectUploadMetadataStore

// RegisterDirectUploadMetadataStore installs the metadata store implementation.
// The file model package registers itself in init() to avoid an import cycle.
func RegisterDirectUploadMetadataStore(store DirectUploadMetadataStore) {
	mu.Lock()
	defer mu.Unlock()
	directUploadMetadataStore = store
}

func metadataStore() DirectUploadMetadataStore {
	mu.RLock()
	defer mu.RUnlock()
	return directUploadMetadataStore
}

// SupportsDirectUpload reports whether the active provider can presign direct
// uploads (currently only the S3 provider). The operator switch
// StorageSettings.ForceProxyUpload disables direct uploads while keeping the
// S3 provider: clients then use the server-proxied POST /file/img-upload,
// which still writes to the bucket through Provider.Save.
func SupportsDirectUpload() bool {
	if GetStorageSettings().ForceProxyUpload {
		return false
	}
	_, ok := Current().(DirectUploadProvider)
	return ok
}

// NewUploadName builds a unique object key under customPath for filename.
func NewUploadName(filename string, customPath string) string {
	return fmt.Sprintf("%s/%s%s", customPath, uuid.New().String(), strings.ToLower(path.Ext(filename)))
}

// BeginDirectUpload validates the request, creates a pending metadata row, then
// presigns the upload. On presign failure the pending row is rolled back.
func BeginDirectUpload(ctx context.Context, request DirectUploadRequest) (*DirectUploadSession, error) {
	store, ok := Current().(DirectUploadProvider)
	if !ok {
		return nil, ErrDirectUploadUnsupported
	}
	if request.UserId == 0 {
		return nil, errors.New("direct upload user is required")
	}
	if err := validateDirectUploadRequest(request.Name, request.ContentType, request.Size); err != nil {
		return nil, err
	}
	metadataRepo := metadataStore()
	if metadataRepo == nil {
		return nil, errors.New("direct upload metadata store is not registered")
	}
	metadata, err := metadataRepo.CreateFileMetadata(ctx, request.UserId, request.Name, request.ContentType, request.Size)
	if err != nil {
		return nil, err
	}
	upload, err := store.PresignUpload(ctx, request)
	if err != nil {
		_ = metadataRepo.DeleteByName(ctx, request.Name)
		return nil, err
	}
	return &DirectUploadSession{Metadata: *metadata, Upload: *upload}, nil
}

// CompleteDirectUpload verifies the uploaded object (size, content type and
// optional content validator) and flips the pending row to ready.
func CompleteDirectUpload(ctx context.Context, request CompleteDirectUploadRequest) (*FileMetadata, error) {
	if request.Name == "" || request.UserId == 0 {
		return nil, errors.New("direct upload name and user are required")
	}
	metadataRepo := metadataStore()
	if metadataRepo == nil {
		return nil, errors.New("direct upload metadata store is not registered")
	}
	metadata, err := metadataRepo.GetPendingFileMetadataByName(ctx, request.Name)
	if err != nil {
		// Idempotent duplicate complete: a row already flipped to ready by the
		// same user is returned as success instead of failing.
		ready, readyErr := metadataRepo.GetFileMetadataByName(ctx, request.Name)
		if readyErr == nil {
			if ready.UserId != request.UserId {
				return nil, ErrDirectUploadOwnerMismatch
			}
			return ready, nil
		}
		return nil, err
	}
	if metadata.UserId != request.UserId {
		return nil, ErrDirectUploadOwnerMismatch
	}
	store, ok := Current().(DirectUploadProvider)
	if !ok {
		return nil, ErrDirectUploadUnsupported
	}
	verifyRequest := DirectUploadRequest{Name: metadata.Name, ContentType: metadata.Type, Size: metadata.Size, UserId: metadata.UserId}
	if err := store.VerifyUpload(ctx, verifyRequest); err != nil {
		if errors.Is(err, ErrDirectUploadInvalidObject) {
			return nil, rollbackDirectUpload(ctx, metadataRepo, request.Name, err)
		}
		return nil, err
	}
	uploadName := metadata.Name
	if request.Validator != nil {
		// Validate only a bounded header: the object's full size was already
		// checked by VerifyUpload (StatObject), so the server never downloads
		// the whole (up to the upload cap) object during a direct upload
		// publish. Providers that support range reads are preferred; others
		// fall back to reading the whole object.
		headerReader, err := readObjectHeader(ctx, request.Name)
		if err != nil {
			return nil, err
		}
		detectedType, validationErr := request.Validator(headerReader)
		if validationErr != nil {
			if errors.Is(validationErr, ErrDirectUploadInvalidObject) {
				return nil, rollbackDirectUpload(ctx, metadataRepo, request.Name, validationErr)
			}
			return nil, validationErr
		}
		renamed, renameErr := normalizeDirectUploadType(ctx, metadataRepo, metadata, detectedType)
		if renameErr != nil {
			return nil, renameErr
		}
		uploadName = renamed.Name
	}
	return metadataRepo.MarkFileReady(ctx, uploadName)
}

// normalizeDirectUploadType renames the object to the canonical extension of the
// byte-detected type when the client-declared extension differs, so the stored
// name (and the Content-Type served from it) follows the bytes. Only the
// extension is replaced: the client's declared name is already gone from the
// object key (it is a path/uuid pair).
//
// shortcut: the rename downloads the whole object to copy it; it only runs on a
// declared/detected extension conflict, never for a matching upload, so a plain
// Get+Save is enough. Upgrade to a provider-side server-side copy when the
// Provider interface exposes one.
func normalizeDirectUploadType(ctx context.Context, metadataRepo DirectUploadMetadataStore, metadata *FileMetadata, contentType string) (*FileMetadata, error) {
	ext, ok := imagepolicy.ExtForContentType(contentType)
	if !ok {
		return nil, fmt.Errorf("detected content type %q has no canonical extension", contentType)
	}
	if declaredType, known := imagepolicy.ContentTypeForFilename(metadata.Name); known && declaredType == contentType {
		return metadata, nil
	}
	newName := strings.TrimSuffix(metadata.Name, path.Ext(metadata.Name)) + ext
	data, _, err := Current().Get(ctx, metadata.Name)
	if err != nil {
		return nil, fmt.Errorf("read direct upload object %q: %w", metadata.Name, err)
	}
	if err := Current().Save(ctx, newName, data, contentType); err != nil {
		return nil, fmt.Errorf("save renamed direct upload object %q: %w", newName, err)
	}
	renamed, err := metadataRepo.RenameFileMetadata(ctx, metadata.Name, newName, contentType)
	if err != nil {
		// The row is not ours any more: a concurrent complete of the same pending
		// name may already have renamed or published it, so the object under
		// newName can be the winner's. Never delete it here.
		// shortcut: a failed rename leaves the copy written above unreferenced; the
		// orphan reclamation (upload_owner rows) and CleanupPending (pending
		// metadata rows) both need a row, so nothing sweeps these objects on its
		// own - they need a later cleanup task or manual removal. Accepted because
		// it never blocks a publish and holds no over-privileged data.
		return nil, fmt.Errorf("rename direct upload metadata %q: %w", metadata.Name, err)
	}
	// The rename is committed, so a failed delete must not fail the publish.
	// shortcut: the leftover old object has no metadata row, and neither the
	// orphan reclamation nor CleanupPending collects row-less objects, so it stays
	// until a later cleanup task or manual removal. Accepted for the same reason.
	_ = Current().Delete(ctx, metadata.Name)
	return renamed, nil
}

// readObjectHeader returns a reader over a bounded prefix of the object. When
// the provider implements ObjectRangeReader the server only pulls the header
// bytes; otherwise it falls back to reading the whole object and truncating.
func readObjectHeader(ctx context.Context, name string) (io.Reader, error) {
	if rangeReader, ok := Current().(ObjectRangeReader); ok {
		data, _, err := rangeReader.GetRange(ctx, name, 0, imagepolicy.ImageHeaderSniffBytes)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}
	data, _, err := Current().Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(data) > imagepolicy.ImageHeaderSniffBytes {
		data = data[:imagepolicy.ImageHeaderSniffBytes]
	}
	return bytes.NewReader(data), nil
}

// AbortDirectUpload verifies ownership and removes both the object and the
// pending row.
func AbortDirectUpload(ctx context.Context, request CompleteDirectUploadRequest) error {
	if request.Name == "" || request.UserId == 0 {
		return errors.New("direct upload name and user are required")
	}
	metadataRepo := metadataStore()
	if metadataRepo == nil {
		return errors.New("direct upload metadata store is not registered")
	}
	metadata, err := metadataRepo.GetPendingFileMetadataByName(ctx, request.Name)
	if err != nil {
		return err
	}
	if metadata.UserId != request.UserId {
		return ErrDirectUploadOwnerMismatch
	}
	return rollbackDirectUpload(ctx, metadataRepo, request.Name, nil)
}

// DeleteDirectUpload removes both a completed upload's object and its metadata
// row. It is used when a complete succeeded but a later step (e.g. recording
// the upload owner) failed: deleting only the object would leave a ready
// metadata row whose object is gone, occupying the name forever.
func DeleteDirectUpload(ctx context.Context, name string) error {
	metadataRepo := metadataStore()
	if metadataRepo == nil {
		return errors.New("direct upload metadata store is not registered")
	}
	return rollbackDirectUpload(ctx, metadataRepo, name, nil)
}

// CleanupPending removes pending uploads older than before, deleting their
// objects and rows. Returns the number of uploads removed.
func CleanupPending(ctx context.Context, before time.Time, limit int) (int, error) {
	metadataRepo := metadataStore()
	if metadataRepo == nil {
		return 0, errors.New("direct upload metadata store is not registered")
	}
	items, err := metadataRepo.ListPendingFilesBefore(ctx, before, limit)
	if err != nil {
		return 0, err
	}
	removed := 0
	var cleanupErr error
	for _, metadata := range items {
		if err := rollbackDirectUpload(ctx, metadataRepo, metadata.Name, nil); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("cleanup pending file %q: %w", metadata.Name, err))
			continue
		}
		removed++
	}
	return removed, cleanupErr
}

func rollbackDirectUpload(ctx context.Context, metadataRepo DirectUploadMetadataStore, name string, cause error) error {
	if err := Current().Delete(ctx, name); err != nil && !errors.Is(err, ErrNotFound) {
		return errors.Join(cause, fmt.Errorf("delete direct upload object %q: %w", name, err))
	}
	if err := metadataRepo.DeleteByName(ctx, name); err != nil {
		return errors.Join(cause, fmt.Errorf("delete direct upload metadata %q: %w", name, err))
	}
	return cause
}

func validateDirectUploadRequest(name, contentType string, size int64) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("direct upload name is required")
	}
	if strings.TrimSpace(contentType) == "" {
		return errors.New("direct upload content type is required")
	}
	if size <= 0 {
		return errors.New("direct upload size must be positive")
	}
	return nil
}
