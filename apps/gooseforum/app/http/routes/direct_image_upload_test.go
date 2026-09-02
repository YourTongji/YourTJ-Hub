package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// routeFakeObject is an in-memory object stored by the fake S3 provider.
type routeFakeObject struct {
	data        []byte
	contentType string
}

// routeFakeS3Provider simulates the S3 provider for route tests: presigned
// uploads are recorded in memory and verified/read back without network I/O.
type routeFakeS3Provider struct {
	objects map[string]routeFakeObject
}

func newRouteFakeS3Provider() *routeFakeS3Provider {
	return &routeFakeS3Provider{objects: map[string]routeFakeObject{}}
}

func (p *routeFakeS3Provider) Save(_ context.Context, name string, data []byte, contentType string) error {
	p.objects[name] = routeFakeObject{data: data, contentType: contentType}
	return nil
}

func (p *routeFakeS3Provider) Get(_ context.Context, name string) ([]byte, string, error) {
	obj, ok := p.objects[name]
	if !ok {
		return nil, "", storageservice.ErrNotFound
	}
	return obj.data, obj.contentType, nil
}

func (p *routeFakeS3Provider) Delete(_ context.Context, name string) error {
	delete(p.objects, name)
	return nil
}

func (p *routeFakeS3Provider) Exists(_ context.Context, name string) (bool, error) {
	_, ok := p.objects[name]
	return ok, nil
}

func (p *routeFakeS3Provider) PresignUpload(_ context.Context, request storageservice.DirectUploadRequest) (*storageservice.DirectUpload, error) {
	return &storageservice.DirectUpload{
		URL:       "https://fake.example.com/" + request.Name,
		Method:    "POST",
		Fields:    map[string]string{"key": request.Name},
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func (p *routeFakeS3Provider) VerifyUpload(_ context.Context, request storageservice.DirectUploadRequest) error {
	obj, ok := p.objects[request.Name]
	if !ok {
		return storageservice.ErrNotFound
	}
	if int64(len(obj.data)) != request.Size {
		return fmt.Errorf("size mismatch: got %d, want %d", len(obj.data), request.Size)
	}
	if !strings.EqualFold(obj.contentType, request.ContentType) {
		return fmt.Errorf("content type mismatch: got %q, want %q", obj.contentType, request.ContentType)
	}
	return nil
}

// setupDirectUploadRouteTest registers the file routes with production
// middleware and returns the router plus a fake S3 provider override.
func setupDirectUploadRouteTest(t *testing.T) (*gorm.DB, *gin.Engine, *routeFakeS3Provider) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate filedata contract table: %v", err)
	}
	fileAPI := router.Group("/file")
	fileAPI.POST("/img-upload/init", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.InitDirectImageUpload)
	fileAPI.POST("/img-upload/complete", middleware.JWTAuthCheck, middleware.CheckWritableAccount, api.CompleteDirectImageUpload)
	fileAPI.POST("/img-upload/abort", middleware.JWTAuthCheck, middleware.CheckWritableAccount, api.AbortDirectImageUpload)
	fileAPI.GET("/img/*filename", api.GetFileByFileName)

	provider := newRouteFakeS3Provider()
	storageservice.SetCurrentForTest(provider)
	t.Cleanup(func() {
		storageservice.SetCurrentForTest(nil)
		db4fileconnect.Connect().Where("1 = 1").Delete(&filedata.Entity{})
	})
	return conn, router, provider
}

func mustMarshalJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(b)
}

func serveDirectUploadJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func directUploadInit(t *testing.T, router http.Handler, token, filename string, size int64) (*httptest.ResponseRecorder, string) {
	t.Helper()
	body := mustMarshalJSON(t, map[string]any{
		"filename":    filename,
		"contentType": "image/png",
		"size":        size,
	})
	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/init", body, token)
	var envelope struct {
		Result struct {
			Mode string `json:"mode"`
			Name string `json:"name"`
		} `json:"result"`
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode init response %q: %v", recorder.Body.String(), err)
	}
	return recorder, envelope.Result.Name
}

func TestDirectImageUploadLocalModeReturnsProxy(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate filedata contract table: %v", err)
	}
	fileAPI := router.Group("/file")
	fileAPI.POST("/img-upload/init", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.InitDirectImageUpload)
	user := createHTTPContractUser(t, conn, contractTestID())

	recorder, _ := directUploadInit(t, router, contractSessionToken(t, user), "a.png", 100)
	if recorder.Code != http.StatusOK {
		t.Fatalf("init status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Result struct {
			Mode string `json:"mode"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode init response: %v", err)
	}
	if envelope.Result.Mode != "proxy" {
		t.Fatalf("init mode = %q, want proxy", envelope.Result.Mode)
	}
}

func TestDirectImageUploadInitRejectsOverLimitSize(t *testing.T) {
	conn, router, _ := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())

	recorder, _ := directUploadInit(t, router, contractSessionToken(t, user), "big.png", filedata.MaxFileSize+1)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("init status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode init response: %v", err)
	}
	if envelope.MessageCode != "upload.file.tooLarge" {
		t.Fatalf("init messageCode = %q, want upload.file.tooLarge", envelope.MessageCode)
	}
}

func TestDirectImageUploadCompleteRejectsForgedMIME(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	textBytes := []byte("this is definitely not an image")
	_, name := directUploadInit(t, router, token, "fake.png", int64(len(textBytes)))
	if name == "" {
		t.Fatal("init returned empty name")
	}
	// Simulate the browser uploading a text file with a forged image content type.
	provider.objects[name] = routeFakeObject{data: textBytes, contentType: "image/png"}

	body := mustMarshalJSON(t, map[string]string{"name": name})
	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", body, token)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("complete status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	if envelope.MessageCode != "upload.image.invalidContent" {
		t.Fatalf("complete messageCode = %q, want upload.image.invalidContent", envelope.MessageCode)
	}
	// The invalid object must be rolled back.
	if _, ok := provider.objects[name]; ok {
		t.Fatal("invalid object still present after rejected complete")
	}
}

func TestDirectImageUploadCompleteOwnerMismatch(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())

	_, name := directUploadInit(t, router, contractSessionToken(t, owner), "a.png", 4)
	if name == "" {
		t.Fatal("init returned empty name")
	}
	provider.objects[name] = routeFakeObject{data: []byte("png!"), contentType: "image/png"}

	body := mustMarshalJSON(t, map[string]string{"name": name})
	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", body, contractSessionToken(t, other))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("complete status = %d, want 404: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	if envelope.MessageCode != "page.notFound" {
		t.Fatalf("complete messageCode = %q, want page.notFound", envelope.MessageCode)
	}
}

func TestDirectImageUploadDuplicateCompleteIdempotent(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	_, name := directUploadInit(t, router, token, "a.png", int64(len(contractTinyPNG)))
	if name == "" {
		t.Fatal("init returned empty name")
	}
	provider.objects[name] = routeFakeObject{data: contractTinyPNG, contentType: "image/png"}

	body := mustMarshalJSON(t, map[string]string{"name": name})
	first := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", body, token)
	if first.Code != http.StatusOK {
		t.Fatalf("first complete status = %d, want 200: %s", first.Code, first.Body.String())
	}
	second := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", body, token)
	if second.Code != http.StatusOK {
		t.Fatalf("duplicate complete status = %d, want 200 (idempotent): %s", second.Code, second.Body.String())
	}
}

func TestDirectImageUploadAccessAfterAbort(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	_, name := directUploadInit(t, router, token, "a.png", 4)
	if name == "" {
		t.Fatal("init returned empty name")
	}
	provider.objects[name] = routeFakeObject{data: []byte("png!"), contentType: "image/png"}

	body := mustMarshalJSON(t, map[string]string{"name": name})
	abort := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/abort", body, token)
	if abort.Code != http.StatusOK {
		t.Fatalf("abort status = %d, want 200: %s", abort.Code, abort.Body.String())
	}
	if _, ok := provider.objects[name]; ok {
		t.Fatal("object still present after abort")
	}

	recorder := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+name, "", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET after abort status = %d, want 404: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDirectImageUploadCompleteSuccessRecordsOwnerUsage(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	_, name := directUploadInit(t, router, token, "a.png", int64(len(contractTinyPNG)))
	if name == "" {
		t.Fatal("init returned empty name")
	}
	provider.objects[name] = routeFakeObject{data: contractTinyPNG, contentType: "image/png"}

	body := mustMarshalJSON(t, map[string]string{"name": name})
	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", body, token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Result struct {
			URL      string `json:"url"`
			Filename string `json:"filename"`
			Size     int64  `json:"size"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	if envelope.Result.Filename != name || envelope.Result.Size != int64(len(contractTinyPNG)) {
		t.Fatalf("complete result = %+v, want filename %q size %d", envelope.Result, name, len(contractTinyPNG))
	}
	if !strings.Contains(envelope.Result.URL, name) {
		t.Fatalf("complete url = %q, want it to contain %q", envelope.Result.URL, name)
	}

	// The completed file is now publicly readable.
	read := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+name, "", "")
	if read.Code != http.StatusOK {
		t.Fatalf("GET completed file status = %d, want 200: %s", read.Code, read.Body.String())
	}
}

func TestDirectImageUploadAbortOwnerMismatch(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())

	_, name := directUploadInit(t, router, contractSessionToken(t, owner), "a.png", 4)
	if name == "" {
		t.Fatal("init returned empty name")
	}
	provider.objects[name] = routeFakeObject{data: []byte("png!"), contentType: "image/png"}

	body := mustMarshalJSON(t, map[string]string{"name": name})
	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/abort", body, contractSessionToken(t, other))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("abort status = %d, want 404: %s", recorder.Code, recorder.Body.String())
	}
	if _, ok := provider.objects[name]; !ok {
		t.Fatal("object removed despite owner mismatch")
	}
}

func TestDirectImageUploadRequiresAuth(t *testing.T) {
	_, router, _ := setupDirectUploadRouteTest(t)
	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/init", `{}`, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated init status = %d, want 401: %s", recorder.Code, recorder.Body.String())
	}
}
