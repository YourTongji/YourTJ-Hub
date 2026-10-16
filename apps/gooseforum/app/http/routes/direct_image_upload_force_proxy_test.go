package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func forceProxyStorageSettings(force bool) pageConfig.StorageSettings {
	return pageConfig.StorageSettings{
		Provider:         "s3",
		Endpoint:         "https://s3.contract.example.test",
		Bucket:           "contract-bucket",
		Region:           "contract-region",
		BucketLookup:     "auto",
		Secure:           true,
		ForceProxyUpload: force,
	}
}

func decodeInitMode(t *testing.T, body []byte) string {
	t.Helper()
	var envelope struct {
		Result struct {
			Mode string `json:"mode"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode init response %q: %v", string(body), err)
	}
	return envelope.Result.Mode
}

// 开关关闭（默认）：S3 提供方保持浏览器直传。
func TestDirectImageUploadS3DefaultStaysDirect(t *testing.T) {
	conn, router, _ := setupDirectUploadRouteTest(t)
	persistContractStorageSettings(t, conn, forceProxyStorageSettings(false))
	user := createHTTPContractUser(t, conn, contractTestID())

	recorder, name := directUploadInit(t, router, contractSessionToken(t, user), "a.png", int64(len(contractTinyPNG)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("init status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if mode := decodeInitMode(t, recorder.Body.Bytes()); mode != "direct" || name == "" {
		t.Fatalf("init mode = %q name = %q, want direct with a pending name", mode, name)
	}
}

// 开关开启：S3 提供方下 init 回 proxy，代理上传仍经 Provider.Save 写入 bucket。
func TestDirectImageUploadForceProxyKeepsS3Storage(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	persistContractStorageSettings(t, conn, forceProxyStorageSettings(true))
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	recorder, name := directUploadInit(t, router, token, "a.png", int64(len(contractTinyPNG)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("init status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if mode := decodeInitMode(t, recorder.Body.Bytes()); mode != "proxy" || name != "" {
		t.Fatalf("init mode = %q name = %q, want proxy without a pending object", mode, name)
	}
	if len(provider.objects) != 0 {
		t.Fatalf("force-proxy init must not create objects, got %d", len(provider.objects))
	}

	upload := serveMultipartNamed(router, "/file/img-upload", map[string]routeNamedFile{
		"file": {name: "a.jpg", data: routeTinyJPEG(t)},
	}, token)
	if upload.Code != http.StatusOK {
		t.Fatalf("proxy upload status = %d, want 200: %s", upload.Code, upload.Body.String())
	}
	var envelope struct {
		Result struct {
			URL string `json:"url"`
		} `json:"result"`
	}
	if err := json.Unmarshal(upload.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode proxy upload response: %v", err)
	}
	stored := ""
	for key := range provider.objects {
		if strings.HasSuffix(key, ".jpg") {
			stored = key
		}
	}
	if stored == "" {
		t.Fatalf("proxy upload did not write the original to the S3 provider; objects = %d", len(provider.objects))
	}
	if !strings.Contains(envelope.Result.URL, stored) {
		t.Fatalf("proxy upload url = %q, want it to reference stored object %q", envelope.Result.URL, stored)
	}
}
