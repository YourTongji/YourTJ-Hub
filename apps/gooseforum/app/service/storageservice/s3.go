package storageservice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Provider stores files in an S3-compatible object store
// (MinIO, Tencent COS, Alibaba OSS, Cloudflare R2, ...).
type S3Provider struct {
	client *minio.Client
	bucket string
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
	return &S3Provider{client: client, bucket: cfg.Bucket}, nil
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
