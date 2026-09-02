package storageservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

const (
	defaultPresignExpiry      = 10 * time.Minute
	maximumPresignExpiry      = time.Hour
	presignedUploadHTTPMethod = "POST"
)

// S3Provider stores files in an S3-compatible object store
// (MinIO, Tencent COS, Alibaba OSS, Cloudflare R2, ...).
type S3Provider struct {
	client *minio.Client
	bucket string
	now    func() time.Time
}

// NewS3Provider builds a minio-go client from storage settings.
//
// Addressing notes (vendor requirements):
//   - Alibaba OSS only supports virtual-hosted style; minio-go auto-detects
//     aliyuncs.com endpoints and switches to virtual-hosted automatically.
//   - Tencent COS buckets created after 2024-01-01 only support virtual-hosted
//     style; set BucketLookup to "dns" and the bucket region explicitly.
//   - Cloudflare R2 and MinIO accept both styles; "auto" defaults to path style
//     for non-Amazon endpoints.
func NewS3Provider(cfg pageConfig.StorageSettings) (*S3Provider, error) {
	if cfg.Bucket == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("s3 provider requires endpoint and bucket")
	}
	lookup := minio.BucketLookupAuto
	switch strings.ToLower(cfg.BucketLookup) {
	case "dns":
		lookup = minio.BucketLookupDNS
	case "path":
		lookup = minio.BucketLookupPath
	}
	// minio-go 的 New() 期望 endpoint 不含 scheme（Secure 选项控制协议）。
	// 用户配置可能带 scheme（如 https://cos.ap-shanghai.myqcloud.com），
	// 这里剥离 scheme 并据此确定 Secure。
	endpoint := cfg.Endpoint
	secure := cfg.Secure
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse s3 endpoint: %w", err)
		}
		endpoint = parsed.Host
		switch parsed.Scheme {
		case "http":
			secure = false
		case "https":
			secure = true
		}
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       secure,
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return &S3Provider{client: client, bucket: cfg.Bucket, now: time.Now}, nil
}

// Save uploads a single object. Files below 16MiB use one atomic PUT.
func (p *S3Provider) Save(ctx context.Context, name string, data []byte, contentType string) error {
	_, err := p.client.PutObject(ctx, p.bucket, name, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Get downloads an object and returns its bytes and content type.
func (p *S3Provider) Get(ctx context.Context, name string) ([]byte, string, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", mapStorageError(err)
	}
	defer func() { _ = obj.Close() }()

	info, err := obj.Stat()
	if err != nil {
		return nil, "", mapStorageError(err)
	}
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", mapStorageError(err)
	}
	return data, info.ContentType, nil
}

// GetRange reads a bounded byte range of an object via an S3 Range request so
// the whole body never crosses the network. The content type comes from the
// object stat.
func (p *S3Provider) GetRange(ctx context.Context, name string, offset, length int64) ([]byte, string, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(offset, offset+length-1); err != nil {
		return nil, "", fmt.Errorf("set s3 range %d-%d: %w", offset, offset+length-1, err)
	}
	obj, err := p.client.GetObject(ctx, p.bucket, name, opts)
	if err != nil {
		return nil, "", mapStorageError(err)
	}
	defer func() { _ = obj.Close() }()

	info, err := obj.Stat()
	if err != nil {
		return nil, "", mapStorageError(err)
	}
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", mapStorageError(err)
	}
	return data, info.ContentType, nil
}

// Delete removes an object. Removing a missing object is not an error.
func (p *S3Provider) Delete(ctx context.Context, name string) error {
	err := p.client.RemoveObject(ctx, p.bucket, name, minio.RemoveObjectOptions{})
	return mapStorageError(err)
}

// Exists reports whether the object exists.
func (p *S3Provider) Exists(ctx context.Context, name string) (bool, error) {
	_, err := p.client.StatObject(ctx, p.bucket, name, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return false, nil
	}
	return false, err
}

// mapStorageError normalizes S3 error responses: missing keys map to ErrNotFound.
func mapStorageError(err error) error {
	if err == nil {
		return nil
	}
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return ErrNotFound
	}
	return err
}

// TestConnection validates endpoint, credentials and bucket access with a
// single authenticated round trip. Local provider always succeeds.
func TestConnection(ctx context.Context, cfg pageConfig.StorageSettings) error {
	if cfg.Provider != ProviderS3 {
		return nil
	}
	p, err := NewS3Provider(cfg)
	if err != nil {
		return err
	}
	ok, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return fmt.Errorf("bucket check failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("bucket %q does not exist", p.bucket)
	}
	return nil
}

// PresignUpload builds a presigned POST policy that pins the object key,
// content type and exact byte size, so the browser can upload straight to the
// bucket without the app server seeing the bytes.
func (p *S3Provider) PresignUpload(ctx context.Context, request DirectUploadRequest) (*DirectUpload, error) {
	if err := validateObjectWrite(request.Name, request.ContentType, request.Size); err != nil {
		return nil, err
	}
	expiresIn := request.ExpiresIn
	if expiresIn == 0 {
		expiresIn = defaultPresignExpiry
	}
	if expiresIn < time.Minute || expiresIn > maximumPresignExpiry {
		return nil, fmt.Errorf("s3 upload expiry must be between %s and %s", time.Minute, maximumPresignExpiry)
	}
	expiresAt := p.now().UTC().Add(expiresIn)
	policy := minio.NewPostPolicy()
	setters := []func() error{
		func() error { return policy.SetBucket(p.bucket) },
		func() error { return policy.SetKey(request.Name) },
		func() error { return policy.SetContentType(request.ContentType) },
		func() error { return policy.SetContentLengthRange(request.Size, request.Size) },
		func() error { return policy.SetExpires(expiresAt) },
		func() error { return policy.SetSuccessStatusAction("204") },
	}
	for _, set := range setters {
		if err := set(); err != nil {
			return nil, err
		}
	}
	uploadURL, fields, err := p.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return nil, err
	}
	return &DirectUpload{URL: uploadURL.String(), Method: presignedUploadHTTPMethod, Fields: fields, ExpiresAt: expiresAt}, nil
}

// VerifyUpload checks that the uploaded object matches the expected size and
// content type recorded at init time.
func (p *S3Provider) VerifyUpload(ctx context.Context, request DirectUploadRequest) error {
	if err := validateObjectWrite(request.Name, request.ContentType, request.Size); err != nil {
		return err
	}
	info, err := p.client.StatObject(ctx, p.bucket, request.Name, minio.StatObjectOptions{})
	if err != nil {
		return err
	}
	if info.Size != request.Size {
		return fmt.Errorf("s3 object size mismatch: got %d, want %d", info.Size, request.Size)
	}
	if !strings.EqualFold(info.ContentType, request.ContentType) {
		return fmt.Errorf("s3 object content type mismatch: got %q, want %q", info.ContentType, request.ContentType)
	}
	return nil
}

func validateObjectWrite(name, contentType string, size int64) error {
	if name == "" || strings.TrimSpace(name) != name {
		return errors.New("s3 object name is required")
	}
	if strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") || path.Clean(name) != name {
		return errors.New("s3 object name must be a clean relative key")
	}
	if err := s3utils.CheckValidObjectName(name); err != nil {
		return fmt.Errorf("invalid s3 object name: %w", err)
	}
	if strings.TrimSpace(contentType) == "" {
		return errors.New("s3 content type is required")
	}
	if size <= 0 {
		return errors.New("s3 object size must be positive")
	}
	return nil
}

var _ Provider = (*S3Provider)(nil)
var _ DirectUploadProvider = (*S3Provider)(nil)
var _ ObjectRangeReader = (*S3Provider)(nil)
