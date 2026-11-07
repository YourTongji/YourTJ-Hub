//go:build miniointegration

package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// This opt-in suite only talks to the disposable loopback server started by
// apps/gooseforum/scripts/run-minio-upload-e2e.sh. It cannot target a production endpoint.
func TestDirectUploadMinIOReplay(t *testing.T) {
	if os.Getenv("YOURTJ_MINIO_E2E") != "1" || os.Getenv("MINIO_ROOT_USER") == "" || os.Getenv("MINIO_ROOT_PASSWORD") == "" {
		t.Fatal("run through apps/gooseforum/scripts/run-minio-upload-e2e.sh; disposable MinIO is required")
	}
	const endpoint = "127.0.0.1:19000"
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ROOT_USER"), os.Getenv("MINIO_ROOT_PASSWORD"), ""),
		Secure: false, Region: "us-east-1", BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	bucket := "yourtj-upload-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		for object := range client.ListObjects(cleanupCtx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				t.Error(object.Err)
				continue
			}
			if err := client.RemoveObject(cleanupCtx, bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Error(err)
			}
		}
		if err := client.RemoveBucket(cleanupCtx, bucket); err != nil {
			t.Error(err)
		}
	})
	provider, err := storageservice.NewS3Provider(pageConfig.StorageSettings{
		Endpoint: endpoint, Bucket: bucket, Region: "us-east-1", BucketLookup: "path",
		AccessKey: os.Getenv("MINIO_ROOT_USER"), SecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, filename, contentType string }{
		{"matching_png", "image.png", "image/png"},
		{"declared_jpeg_actual_png", "image.jpeg", "image/jpeg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, router, _ := setupDirectUploadRouteTest(t)
			// Keep the real route, middleware, metadata adapter and SQLite stores;
			// replace the helper's fake before making any upload request.
			storageservice.SetCurrentForTest(provider)
			user := createHTTPContractUser(t, conn, contractTestID())
			token := contractSessionToken(t, user)
			response, name := directUploadInitWithType(t, router, token, tc.filename, tc.contentType, int64(len(contractTinyPNG)))
			var init struct {
				Code   int `json:"code"`
				Result struct {
					Mode   string                      `json:"mode"`
					Upload storageservice.DirectUpload `json:"upload"`
				} `json:"result"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &init); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || init.Code != 0 || init.Result.Mode != "direct" || name == "" || init.Result.Upload.Fields["key"] != name {
				t.Fatal("init did not return a successful direct-upload policy")
			}
			post := func(key string, data []byte) (int, string) {
				t.Helper()
				upload := init.Result.Upload
				u, err := url.Parse(upload.URL)
				if err != nil || u.Scheme != "http" || u.Host != endpoint || strings.TrimSuffix(u.Path, "/") != "/"+bucket || u.RawQuery != "" || u.User != nil || u.Fragment != "" || upload.Method != http.MethodPost {
					t.Fatal("presigned POST escaped the disposable loopback bucket")
				}
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				for field, value := range upload.Fields {
					if field == "key" {
						value = key
					}
					if err := writer.WriteField(field, value); err != nil {
						t.Fatal(err)
					}
				}
				file, err := writer.CreateFormFile("file", tc.filename)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write(data); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				request, err := http.NewRequestWithContext(ctx, http.MethodPost, upload.URL, &body)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Content-Type", writer.FormDataContentType())
				httpClient := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
				result, err := httpClient.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = result.Body.Close() }()
				payload, err := io.ReadAll(io.LimitReader(result.Body, 8192))
				if err != nil {
					t.Fatal(err)
				}
				var failure struct {
					Code string `xml:"Code"`
				}
				if len(payload) > 0 {
					_ = xml.Unmarshal(payload, &failure)
				}
				return result.StatusCode, failure.Code
			}
			if status, code := post(name, contractTinyPNG); status != http.StatusNoContent {
				t.Fatalf("original presigned POST: status=%d code=%s", status, code)
			}
			data, _, err := provider.GetRange(ctx, name, 0, int64(len(contractTinyPNG))+1)
			if err != nil || !bytes.Equal(data, contractTinyPNG) {
				t.Fatalf("real MinIO size+1 range: bytes=%d error=%v", len(data), err)
			}
			type completed struct {
				Filename string `json:"filename"`
				URL      string `json:"url"`
				Size     int64  `json:"size"`
			}
			complete := func(key string) completed {
				t.Helper()
				r := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", mustMarshalJSON(t, map[string]string{"name": key}), token)
				var envelope struct {
					Code   int       `json:"code"`
					Result completed `json:"result"`
				}
				if err := json.Unmarshal(r.Body.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				if r.Code != http.StatusOK || envelope.Code != 0 || envelope.Result.Filename == "" {
					t.Fatalf("complete failed: HTTP %d, business code %d", r.Code, envelope.Code)
				}
				return envelope.Result
			}
			first := complete(name)
			if first.Filename == name || !strings.HasSuffix(first.Filename, ".png") || first.Size != int64(len(contractTinyPNG)) || first.URL != "/file/img/"+first.Filename {
				t.Fatalf("unexpected immutable result: %+v", first)
			}
			replayed := bytes.Clone(contractTinyPNG)
			replayed[len(replayed)-1] ^= 0xff
			if status, code := post(name, replayed); status != http.StatusNoContent {
				t.Fatalf("original credential replay: status=%d code=%s", status, code)
			}
			data, _, err = provider.Get(ctx, name)
			if err != nil || !bytes.Equal(data, replayed) {
				t.Fatal("replay did not replace the source object")
			}
			if status, code := post(first.Filename, replayed); status != http.StatusForbidden || code != "AccessDenied" {
				t.Fatalf("altered-key policy: status=%d code=%s; want 403 AccessDenied", status, code)
			}
			data, contentType, err := provider.Get(ctx, first.Filename)
			if err != nil || !bytes.Equal(data, contractTinyPNG) || contentType != "image/png" {
				t.Fatalf("published snapshot changed: type=%s error=%v", contentType, err)
			}
			for _, key := range []string{name, first.Filename} {
				if got := complete(key); got != first {
					t.Fatalf("retry changed result: %+v", got)
				}
			}
			if r := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+name, "", token); r.Code != http.StatusNotFound {
				t.Fatalf("mutable source readable: %d", r.Code)
			}
			if _, err := storageservice.CleanupPending(ctx, time.Now().Add(time.Hour), 100); err != nil {
				t.Fatal(err)
			}
			if exists, err := provider.Exists(ctx, name); err != nil || exists {
				t.Fatalf("source cleanup: exists=%t error=%v", exists, err)
			}
			if got := complete(name); got != first {
				t.Fatal("cleanup lost the completed result")
			}
			data, contentType, err = provider.Get(ctx, first.Filename)
			if err != nil || !bytes.Equal(data, contractTinyPNG) || contentType != "image/png" {
				t.Fatalf("cleanup changed published snapshot: type=%s error=%v", contentType, err)
			}
			read := serveDirectUploadJSON(router, http.MethodGet, first.URL, "", token)
			if read.Code != http.StatusOK || !bytes.Equal(read.Body.Bytes(), contractTinyPNG) || read.Header().Get("Content-Type") != "image/png" {
				t.Fatalf("published route after cleanup: HTTP %d, type=%s", read.Code, read.Header().Get("Content-Type"))
			}
			t.Log("real MinIO: POST and replay succeeded; altered final key rejected; snapshot and retry survived cleanup")
		})
	}
}
