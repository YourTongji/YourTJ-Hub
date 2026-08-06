package storageservice

import (
	"testing"

	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
)

func TestIsLocalProviderDefault(t *testing.T) {
	// 默认存储配置为 local（pageConfig 无记录时返回默认值）
	if !IsLocalProvider() {
		t.Fatal("IsLocalProvider() = false with default config, want true")
	}
}

func TestPublicAccessPathWithConfig(t *testing.T) {
	// 无公开前缀时走本地代理路由
	cfg := pageConfig.StorageSettings{}
	if got := PublicAccessPathWithConfig(cfg, "images/a.png"); got != "/file/img/images/a.png" {
		t.Fatalf("PublicAccessPathWithConfig() = %q, want %q", got, "/file/img/images/a.png")
	}

	// 有公开前缀时使用前缀（处理斜杠边界）
	cfg.PublicUrlPrefix = "https://cdn.example.com/"
	if got := PublicAccessPathWithConfig(cfg, "images/a.png"); got != "https://cdn.example.com/images/a.png" {
		t.Fatalf("PublicAccessPathWithConfig() = %q, want %q", got, "https://cdn.example.com/images/a.png")
	}
}

func TestNewS3ProviderValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  pageConfig.StorageSettings
	}{
		{name: "empty bucket", cfg: pageConfig.StorageSettings{Provider: "s3", Endpoint: "https://minio.local", Bucket: ""}},
		{name: "empty endpoint", cfg: pageConfig.StorageSettings{Provider: "s3", Endpoint: "", Bucket: "bucket"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewS3Provider(tt.cfg); err == nil {
				t.Fatalf("NewS3Provider(%s) error = nil, want error", tt.name)
			}
		})
	}
}

func TestNewS3ProviderBucketLookupMapping(t *testing.T) {
	cfg := pageConfig.StorageSettings{
		Provider:     "s3",
		Endpoint:     "https://minio.local",
		Bucket:       "bucket",
		BucketLookup: "dns",
		Secure:       true,
	}
	p, err := NewS3Provider(cfg)
	if err != nil {
		t.Fatalf("NewS3Provider() error = %v", err)
	}
	if p == nil || p.bucket != "bucket" {
		t.Fatalf("NewS3Provider() returned %+v, want bucket %q", p, "bucket")
	}
}
