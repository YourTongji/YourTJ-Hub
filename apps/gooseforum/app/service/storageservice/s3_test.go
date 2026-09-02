package storageservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func newTestS3Provider(t *testing.T, endpoint string) *S3Provider {
	t.Helper()
	p, err := NewS3Provider(pageConfig.StorageSettings{
		Provider:  ProviderS3,
		Endpoint:  endpoint,
		Bucket:    "test-bucket",
		AccessKey: "test-access",
		SecretKey: "test-secret",
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("NewS3Provider() error = %v", err)
	}
	return p
}

func TestS3ProviderPresignUpload(t *testing.T) {
	p := newTestS3Provider(t, "http://127.0.0.1:1")
	p.now = func() time.Time { return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC) }

	upload, err := p.PresignUpload(context.Background(), DirectUploadRequest{
		Name: "2026/09/01/a.png", ContentType: "image/png", Size: 1024,
	})
	if err != nil {
		t.Fatalf("PresignUpload() error = %v", err)
	}
	if upload.Method != http.MethodPost {
		t.Fatalf("upload method = %q, want POST", upload.Method)
	}
	if upload.URL == "" {
		t.Fatal("upload URL is empty")
	}
	if upload.Fields["key"] != "2026/09/01/a.png" {
		t.Fatalf("upload fields key = %q, want object name", upload.Fields["key"])
	}
	if !upload.ExpiresAt.Equal(time.Date(2026, 9, 1, 12, 10, 0, 0, time.UTC)) {
		t.Fatalf("upload expiresAt = %v, want 10 minutes after now", upload.ExpiresAt)
	}
}

func TestS3ProviderPresignUploadValidation(t *testing.T) {
	p := newTestS3Provider(t, "http://127.0.0.1:1")
	tests := []struct {
		name        string
		contentType string
		size        int64
	}{
		{name: "", contentType: "image/png", size: 4},
		{name: "../escape.png", contentType: "image/png", size: 4},
		{name: "a.png", contentType: "", size: 4},
		{name: "a.png", contentType: "image/png", size: 0},
	}
	for _, tt := range tests {
		if _, err := p.PresignUpload(context.Background(), DirectUploadRequest{Name: tt.name, ContentType: tt.contentType, Size: tt.size}); err == nil {
			t.Fatalf("PresignUpload(%q, %q, %d) error = nil, want validation error", tt.name, tt.contentType, tt.size)
		}
	}
}

func TestS3ProviderPresignUploadExpiryBounds(t *testing.T) {
	p := newTestS3Provider(t, "http://127.0.0.1:1")
	for _, expiresIn := range []time.Duration{30 * time.Second, 2 * time.Hour} {
		if _, err := p.PresignUpload(context.Background(), DirectUploadRequest{
			Name: "a.png", ContentType: "image/png", Size: 4, ExpiresIn: expiresIn,
		}); err == nil {
			t.Fatalf("PresignUpload(expiresIn=%s) error = nil, want expiry bounds error", expiresIn)
		}
	}
}

func TestS3ProviderVerifyUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Content-Length", "1024")
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := newTestS3Provider(t, server.URL)
	ctx := context.Background()

	if err := p.VerifyUpload(ctx, DirectUploadRequest{Name: "a.png", ContentType: "image/png", Size: 1024}); err != nil {
		t.Fatalf("VerifyUpload() matching object error = %v", err)
	}
	if err := p.VerifyUpload(ctx, DirectUploadRequest{Name: "a.png", ContentType: "image/png", Size: 2048}); err == nil {
		t.Fatal("VerifyUpload() size mismatch error = nil, want error")
	}
	if err := p.VerifyUpload(ctx, DirectUploadRequest{Name: "a.png", ContentType: "image/jpeg", Size: 1024}); err == nil {
		t.Fatal("VerifyUpload() content type mismatch error = nil, want error")
	}
}

func TestValidateObjectWrite(t *testing.T) {
	valid := []struct {
		name        string
		contentType string
		size        int64
	}{
		{"2026/09/01/a.png", "image/png", 4},
		{"avatars/1/avatar.webp", "image/webp", 10},
	}
	for _, tt := range valid {
		if err := validateObjectWrite(tt.name, tt.contentType, tt.size); err != nil {
			t.Fatalf("validateObjectWrite(%q, %q, %d) error = %v, want nil", tt.name, tt.contentType, tt.size, err)
		}
	}
	invalid := []struct {
		name        string
		contentType string
		size        int64
	}{
		{"", "image/png", 4},
		{" a.png", "image/png", 4},
		{"/abs.png", "image/png", 4},
		{"../escape.png", "image/png", 4},
		{"a.png", "", 4},
		{"a.png", "image/png", 0},
	}
	for _, tt := range invalid {
		if err := validateObjectWrite(tt.name, tt.contentType, tt.size); err == nil {
			t.Fatalf("validateObjectWrite(%q, %q, %d) error = nil, want error", tt.name, tt.contentType, tt.size)
		}
	}
}
