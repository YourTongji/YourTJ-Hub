package storageservice

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

// newDualTestProvider 构造双 endpoint provider：public 作为公网 presign 端点，
// internal 作为服务端数据面端点，二者由两个独立 httptest server 提供并自动清理。
func newDualTestProvider(t *testing.T, publicH, internalH http.Handler) *S3Provider {
	t.Helper()
	publicServer := httptest.NewServer(publicH)
	internalServer := httptest.NewServer(internalH)
	t.Cleanup(func() {
		publicServer.Close()
		internalServer.Close()
	})
	p, err := NewS3Provider(pageConfig.StorageSettings{
		Provider:         ProviderS3,
		Endpoint:         publicServer.URL,
		InternalEndpoint: internalServer.URL,
		Bucket:           "test-bucket",
		AccessKey:        "test-access",
		SecretKey:        "test-secret",
		Region:           "us-east-1",
	})
	if err != nil {
		t.Fatalf("NewS3Provider(dual) error = %v", err)
	}
	if p.client == p.presignClient {
		t.Fatal("dual provider should build two distinct clients")
	}
	return p
}

// headOKHandler 模拟 S3 HEAD 对象成功（StatObject/VerifyUpload 用）。
func headOKHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "1024")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// s3EmulationHandler 模拟一个含单个对象 a.png（1024B image/png）的 S3 bucket。
// HEAD 一律 200（minio-go BucketExists/StatObject 用，path 可能是 /bucket 或 /bucket/a.png）；
// GET /<bucket>/a.png → 200 对象数据；其余 GET → 404。reqLog 非空时记录每个请求
// "METHOD path"，供断言请求落在哪个端点。
func s3EmulationHandler(reqLog *[]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reqLog != nil {
			*reqLog = append(*reqLog, r.Method+" "+r.URL.Path)
		}
		path := strings.TrimSuffix(r.URL.Path, "/") // BucketExists 发 HEAD /bucket/
		objectPath := strings.HasSuffix(path, "/a.png")
		switch r.Method {
		case http.MethodHead:
			if objectPath {
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", "1024")
			} else {
				w.Header().Set("Content-Type", "application/xml")
				w.Header().Set("Content-Length", "0")
			}
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if objectPath {
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", "1024")
				w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(bytes.Repeat([]byte{0x01}, 1024))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func TestS3ProviderDualEndpointPresignUsesPublic(t *testing.T) {
	p := newDualTestProvider(t,
		http.HandlerFunc(headOKHandler),
		http.HandlerFunc(headOKHandler),
	)

	upload, err := p.PresignUpload(context.Background(), DirectUploadRequest{
		Name: "2026/09/01/a.png", ContentType: "image/png", Size: 1024,
	})
	if err != nil {
		t.Fatalf("PresignUpload() error = %v", err)
	}
	u, err := url.Parse(upload.URL)
	if err != nil {
		t.Fatalf("parse presigned URL %q: %v", upload.URL, err)
	}
	publicHost := p.presignClient.EndpointURL().Host
	if u.Host != publicHost {
		t.Fatalf("presign URL host = %q, want public %q (url=%q)", u.Host, publicHost, upload.URL)
	}
}

func TestS3ProviderDualEndpointVerifyAndGetHitInternal(t *testing.T) {
	var publicLog, internalLog []string
	p := newDualTestProvider(t,
		s3EmulationHandler(&publicLog),
		s3EmulationHandler(&internalLog),
	)
	ctx := context.Background()

	if err := p.VerifyUpload(ctx, DirectUploadRequest{Name: "a.png", ContentType: "image/png", Size: 1024}); err != nil {
		t.Fatalf("VerifyUpload() error = %v", err)
	}
	data, ct, err := p.Get(ctx, "a.png")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(data) != 1024 || ct != "image/png" {
		t.Fatalf("Get() = %d bytes %q, want 1024 image/png", len(data), ct)
	}
	if len(internalLog) == 0 {
		t.Fatalf("VerifyUpload/Get never hit internal endpoint (log=%v)", internalLog)
	}
	if len(publicLog) != 0 {
		t.Fatalf("data plane hit public endpoint %d times (log=%v); must use internal", len(publicLog), publicLog)
	}
	t.Logf("internal endpoint requests: %v", internalLog)
}

func TestS3ProviderSingleEndpointWhenInternalEmpty(t *testing.T) {
	p := newTestS3Provider(t, "http://127.0.0.1:1")
	if p.client != p.presignClient {
		t.Fatal("empty internalEndpoint must keep a single client (historical behavior)")
	}
}

func TestS3ProviderSingleEndpointWhenInternalSameHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(headOKHandler))
	defer server.Close()
	p, err := NewS3Provider(pageConfig.StorageSettings{
		Provider:         ProviderS3,
		Endpoint:         server.URL,
		InternalEndpoint: server.URL + "/", // 同主机（仅尾斜杠差异）→ 单 client
		Bucket:           "test-bucket",
		AccessKey:        "test-access",
		SecretKey:        "test-secret",
		Region:           "us-east-1",
	})
	if err != nil {
		t.Fatalf("NewS3Provider(same host) error = %v", err)
	}
	if p.client != p.presignClient {
		t.Fatal("internalEndpoint with same host as endpoint must keep a single client")
	}
}

func TestS3ProviderInvalidInternalEndpointFails(t *testing.T) {
	_, err := NewS3Provider(pageConfig.StorageSettings{
		Provider:         ProviderS3,
		Endpoint:         "http://127.0.0.1:1",
		InternalEndpoint: "://bad-scheme",
		Bucket:           "test-bucket",
		AccessKey:        "test-access",
		SecretKey:        "test-secret",
	})
	if err == nil {
		t.Fatal("NewS3Provider(bad internal) error = nil, want error")
	}
}

func TestS3ProviderTestConnectionDualProbe(t *testing.T) {
	t.Run("both endpoints healthy", func(t *testing.T) {
		publicServer := httptest.NewServer(s3EmulationHandler(nil))
		internalServer := httptest.NewServer(s3EmulationHandler(nil))
		defer publicServer.Close()
		defer internalServer.Close()
		cfg := pageConfig.StorageSettings{
			Provider:         ProviderS3,
			Endpoint:         publicServer.URL,
			InternalEndpoint: internalServer.URL, // 独立主机（不同端口）→ 真双 client
			Bucket:           "test-bucket",
			Region:           "us-east-1",
			AccessKey:        "test-access",
			SecretKey:        "test-secret",
		}
		if err := TestConnection(context.Background(), cfg); err != nil {
			t.Fatalf("TestConnection(dual healthy) error = %v", err)
		}
	})

	t.Run("internal down fails fast with internal prefix", func(t *testing.T) {
		publicServer := httptest.NewServer(s3EmulationHandler(nil))
		defer publicServer.Close()
		cfg := pageConfig.StorageSettings{
			Provider:         ProviderS3,
			Endpoint:         publicServer.URL,
			InternalEndpoint: "http://127.0.0.1:1",
			Bucket:           "test-bucket",
			Region:           "us-east-1",
			AccessKey:        "test-access",
			SecretKey:        "test-secret",
		}
		err := TestConnection(context.Background(), cfg)
		if err == nil {
			t.Fatal("TestConnection(internal down) error = nil, want error")
		}
		if !strings.Contains(err.Error(), "internal endpoint bucket check failed") {
			t.Fatalf("TestConnection error = %v, want internal endpoint prefix", err)
		}
	})
}

func TestS3ProviderTestConnectionSingleProbeWhenNoInternal(t *testing.T) {
	server := httptest.NewServer(s3EmulationHandler(nil))
	cfg := pageConfig.StorageSettings{
		Provider:  ProviderS3,
		Endpoint:  server.URL,
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		AccessKey: "test-access",
		SecretKey: "test-secret",
	}
	if err := TestConnection(context.Background(), cfg); err != nil {
		t.Fatalf("TestConnection(single) error = %v", err)
	}
}

func TestS3ProviderSameHostNormalization(t *testing.T) {
	cases := []struct {
		a, b string
		same bool
	}{
		{"https://oss-ap-northeast-2.aliyuncs.com", "https://oss-ap-northeast-2.aliyuncs.com/", true},
		{"oss-ap-northeast-2.aliyuncs.com", "https://oss-ap-northeast-2.aliyuncs.com", true},
		{"https://oss-ap-northeast-2.aliyuncs.com", "https://oss-ap-northeast-2-internal.aliyuncs.com", false},
		{"http://a.example.com", "HTTP://A.EXAMPLE.COM", true},
	}
	for _, tt := range cases {
		if got := sameS3EndpointHost(tt.a, tt.b); got != tt.same {
			t.Errorf("sameS3EndpointHost(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.same)
		}
	}
}
