package routes

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type routeNamedFile struct {
	name string
	data []byte
}

func serveMultipartNamed(router http.Handler, path string, files map[string]routeNamedFile, token string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for field, named := range files {
		part, err := writer.CreateFormFile(field, named.name)
		if err != nil {
			return httptest.NewRecorder()
		}
		_, _ = part.Write(named.data)
	}
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// routeTinyJPEG returns a valid 1x1 JPEG for legal-format upload assertions.
func routeTinyJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode tiny jpeg: %v", err)
	}
	return buf.Bytes()
}

// tinyGIF 是 1x1 透明 GIF89a（服务端 DecodeConfig 可解码）。
var tinyGIF = []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xFF\xFF\xFF\x21\xF9\x04\x01\x00\x00\x00\x00\x2C\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3B")

// tinyBMP 是 1x1 24bpp BMP（x/image/bmp 可解码）。注意像素偏移字段必须位于
// 第 10 字节（BITMAPFILEHEADER 保留字段之后），否则魔数注册不匹配、
// DecodeConfig 报 unknown format。
var tinyBMP = []byte{
	0x42, 0x4D, // "BM"
	0x3A, 0x00, 0x00, 0x00, // 文件总长 58
	0x00, 0x00, 0x00, 0x00, // reserved
	0x36, 0x00, 0x00, 0x00, // 像素数据偏移 54
	0x28, 0x00, 0x00, 0x00, // DIB 头长 40
	0x01, 0x00, 0x00, 0x00, // 宽 1
	0x01, 0x00, 0x00, 0x00, // 高 1
	0x01, 0x00, // planes 1
	0x18, 0x00, // 24bpp
	0x00, 0x00, 0x00, 0x00, // BI_RGB
	0x04, 0x00, 0x00, 0x00, // 图像数据长 4
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
	0xFF, 0x00, 0x00, 0x00, // BGR + 补齐
}

// readRepoAsset 读取仓库内资源（如 preset webp），供内容校验端到端断言使用。
func readRepoAsset(t *testing.T, rel string) []byte {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo asset path")
	}
	root := filepath.Join(filepath.Dir(testFile), "..", "..", "..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read repo asset %s: %v", rel, err)
	}
	return contents
}

// TestAdminSavePostingSettingsRejectsDangerousExtensions 验证保存端权威校验
// （issue #408）：危险扩展、双扩展、集合外条目整单拒绝并返回稳定错误码，
// 绝不落库。
func TestAdminSavePostingSettingsRejectsDangerousExtensions(t *testing.T) {
	path := "/api/admin/save-posting-settings"
	bodyFor := func(extensions []string) string {
		encoded, err := json.Marshal(map[string]any{
			"settings": map[string]any{
				"textControl": map[string]any{
					"minPostLength": 5, "maxPostLength": 1000, "minTitleLength": 2,
					"maxTitleLength": 50, "newUserPostCooldownMinutes": 0, "maxDailyTopicsPerUser": 7,
				},
				"uploadControl": map[string]any{
					"allowAttachments": true, "authorizedExtensions": extensions,
					"maxAttachmentSizeKb": 512, "maxDailyUploadsPerUser": 5,
					"newUserUploadCooldownMinutes": 0,
				},
				"llms": map[string]any{"enabled": false, "fullText": false, "files": false},
			},
		})
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		return string(encoded)
	}
	cases := []struct {
		name        string
		extensions  []string
		wantDropped string
	}{
		{name: "svg is rejected", extensions: []string{".png", ".svg"}, wantDropped: ".svg"},
		{name: "html and xml are rejected", extensions: []string{".jpg", ".html", ".xml"}, wantDropped: ".html, .xml"},
		{name: "dotless dangerous shorthand is rejected", extensions: []string{"png", "svg"}, wantDropped: "svg"},
		{name: "double extension is rejected", extensions: []string{".png", "avatar.png.exe"}, wantDropped: "avatar.png.exe"},
		{name: "bare text token is rejected", extensions: []string{"notanext"}, wantDropped: "notanext"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn, router := setupAdminSiteContractTest(t)
			recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, path, bodyFor(tc.extensions))
			envelope := decodeContractEnvelope(t, recorder)
			if envelope.Code == 0 {
				t.Fatalf("dangerous extensions accepted: %#v", envelope)
			}
			if envelope.MessageCode != "admin.upload.extNotAllowed" {
				t.Fatalf("messageCode = %q, want admin.upload.extNotAllowed", envelope.MessageCode)
			}
			if got := envelope.Params["extensions"]; got != tc.wantDropped {
				t.Fatalf("params.extensions = %#v, want %q", got, tc.wantDropped)
			}
			// 拒绝时不得改写已存配置（harness 初始为默认 posting 配置）。
			stored := pageConfig.GetConfigByPageType(pageConfig.PostingSettings, pageConfig.PostingContent{})
			want := defaultPostingAuthorizedExtensions()
			if !reflect.DeepEqual(stored.UploadControl.AuthorizedExtensions, want) {
				t.Fatalf("stored authorizedExtensions = %#v, want unchanged %#v", stored.UploadControl.AuthorizedExtensions, want)
			}
		})
	}
}

// defaultPostingAuthorizedExtensions 返回共享 harness 持久化的默认 posting 配置列表。
func defaultPostingAuthorizedExtensions() []string {
	return []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
}

// TestAdminSavePostingSettingsCanonicalizesLegalList 验证合法条目在保存端
// 规范化为小写带点并去重后落库（png → .png、.JPG → .jpg）。
func TestAdminSavePostingSettingsCanonicalizesLegalList(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	body := `{"settings":{"textControl":{"minPostLength":5,"maxPostLength":1000,"minTitleLength":2,"maxTitleLength":50,"newUserPostCooldownMinutes":0,"maxDailyTopicsPerUser":7},"uploadControl":{"allowAttachments":true,"authorizedExtensions":["webp","PNG","png",".JPG"],"maxAttachmentSizeKb":512,"maxDailyUploadsPerUser":5,"newUserUploadCooldownMinutes":0},"llms":{"enabled":false,"fullText":false,"files":false}}}`
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-posting-settings", body)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 {
		t.Fatalf("legal list rejected: %#v", envelope)
	}
	stored := pageConfig.GetConfigByPageType(pageConfig.PostingSettings, pageConfig.PostingContent{})
	if want := []string{".webp", ".png", ".jpg"}; !reflect.DeepEqual(stored.UploadControl.AuthorizedExtensions, want) {
		t.Fatalf("stored authorizedExtensions = %#v, want canonical %#v", stored.UploadControl.AuthorizedExtensions, want)
	}
}

// TestAdminPostingSettingsGETFiltersLegacyDangerousExtensions 验证读取路径
// 归一化（issue #408）：历史配置混入危险/双扩展条目时，GET 回显只保留合法条目。
func TestAdminPostingSettingsGETFiltersLegacyDangerousExtensions(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	persistContractPageConfig(t, conn, pageConfig.PostingSettings, map[string]any{
		"textControl": map[string]any{
			"minPostLength": 5, "maxPostLength": 20000,
			"minTitleLength": 4, "maxTitleLength": 120,
			"newUserPostCooldownMinutes": 10, "maxDailyTopicsPerUser": 10,
		},
		"uploadControl": map[string]any{
			"allowAttachments":     true,
			"authorizedExtensions": []string{"png", "jpg", ".svg", ".html", "avatar.png.exe"},
			"maxAttachmentSizeKb":  2048, "maxDailyUploadsPerUser": 20, "newUserUploadCooldownMinutes": 30,
		},
		"llms": map[string]any{"enabled": false, "fullText": false, "files": false},
	})
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/posting-settings", "")
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 {
		t.Fatalf("GET failed: %#v", envelope)
	}
	var result struct {
		UploadControl struct {
			AuthorizedExtensions []string `json:"authorizedExtensions"`
		} `json:"uploadControl"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode posting settings result: %v", err)
	}
	if want := []string{"png", "jpg"}; !reflect.DeepEqual(result.UploadControl.AuthorizedExtensions, want) {
		t.Fatalf("GET authorizedExtensions = %#v, want filtered %#v", result.UploadControl.AuthorizedExtensions, want)
	}
}

// TestAdminImgUploadContentGate 验证 multipart 管理端上传（issue #408）：
// 危险扩展名在内容解码前被拒；扩展名合法但字节伪造/解码失败返回稳定错误码。
func TestAdminImgUploadContentGate(t *testing.T) {
	path := "/api/admin/img-upload"
	t.Run("svg extension is rejected before content check", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipartNamed(t, conn, router, path, "evil.svg", contractTinyPNG)
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.extension.unsupported" {
			t.Fatalf("svg upload envelope = %#v, want upload.extension.unsupported", envelope)
		}
	})
	t.Run("png-named text bytes are rejected as invalid content", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipartNamed(t, conn, router, path, "fake.png", []byte("this is definitely not an image"))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.image.invalidContent" {
			t.Fatalf("forged upload envelope = %#v, want upload.image.invalidContent", envelope)
		}
	})
	t.Run("jpg-named png bytes are rejected as invalid content", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipartNamed(t, conn, router, path, "mismatch.jpg", contractTinyPNG)
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.image.invalidContent" {
			t.Fatalf("mismatched upload envelope = %#v, want upload.image.invalidContent", envelope)
		}
	})
	t.Run("gif webp bmp jpg uploads succeed with canonical extensions", func(t *testing.T) {
		webp := readRepoAsset(t, "apps/gooseforum/resource/static/pic/5.webp")
		for _, tc := range []struct {
			ext  string
			data []byte
		}{
			{ext: "gif", data: tinyGIF},
			{ext: "webp", data: webp},
			{ext: "bmp", data: tinyBMP},
			{ext: "jpg", data: routeTinyJPEG(t)},
		} {
			t.Run(tc.ext, func(t *testing.T) {
				conn, router := setupAdminDataContractTest(t)
				recorder := serveAdminDataMultipartNamed(t, conn, router, path, "legal."+tc.ext, tc.data)
				envelope := decodeContractEnvelope(t, recorder)
				if envelope.Code != 0 || envelope.MessageCode != "upload.success" {
					t.Fatalf("legal .%s upload envelope = %#v, want success", tc.ext, envelope)
				}
			})
		}
	})
}

// serveAdminDataMultipartNamed 以 SiteManager 身份提交带显式文件名的 multipart 表单。
func serveAdminDataMultipartNamed(t *testing.T, conn *gorm.DB, router *gin.Engine, path, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractSiteManager(t, conn)
	return serveMultipartNamed(router, path, map[string]routeNamedFile{"file": {name: filename, data: data}}, contractSessionToken(t, manager))
}

// TestDirectUploadInitExtensionGate 验证直传 init 与 multipart 同策略
// （issue #408）：集合外扩展名拒绝，大小写变体按规范化接受。
func TestDirectUploadInitExtensionGate(t *testing.T) {
	t.Run("svg filename is rejected", func(t *testing.T) {
		conn, router, _ := setupDirectUploadRouteTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/init",
			mustMarshalJSON(t, map[string]any{"filename": "evil.svg", "contentType": "image/svg+xml", "size": 100}),
			contractSessionToken(t, user))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.extension.unsupported" {
			t.Fatalf("init envelope = %#v, want upload.extension.unsupported", envelope)
		}
		if got := envelope.Params["extensions"]; got != ".jpg, .jpeg, .png, .gif, .webp, .bmp" {
			t.Fatalf("params.extensions = %#v, want the canonical allowlist joined", got)
		}
	})
	t.Run("double extension filename is rejected", func(t *testing.T) {
		conn, router, _ := setupDirectUploadRouteTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/init",
			mustMarshalJSON(t, map[string]any{"filename": "avatar.png.exe", "contentType": "image/png", "size": 100}),
			contractSessionToken(t, user))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.extension.unsupported" {
			t.Fatalf("init envelope = %#v, want upload.extension.unsupported", envelope)
		}
	})
	t.Run("upper-case canonical extension is accepted", func(t *testing.T) {
		conn, router, _ := setupDirectUploadRouteTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder, name := directUploadInitWithType(t, router, contractSessionToken(t, user), "avatar.PNG", "image/png", 4)
		if name == "" || recorder.Code != http.StatusOK {
			t.Fatalf("init status = %d, name %q; want direct acceptance", recorder.Code, name)
		}
		var envelope struct {
			Result struct {
				Mode string `json:"mode"`
			} `json:"result"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode init response: %v", err)
		}
		if envelope.Result.Mode != "direct" {
			t.Fatalf("init mode = %q, want direct", envelope.Result.Mode)
		}
	})
}

// TestAvatarUploadContentGate 验证头像上传与普通图片上传同策略（issue #408）。
func TestAvatarUploadContentGate(t *testing.T) {
	path := "/api/upload-avatar"
	t.Run("svg avatar is rejected", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveMultipartNamed(router, path, map[string]routeNamedFile{
			"avatar": {name: "avatar.svg", data: contractTinyPNG},
		}, contractSessionToken(t, user))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.extension.unsupported" {
			t.Fatalf("svg avatar envelope = %#v, want upload.extension.unsupported", envelope)
		}
	})
	t.Run("png-named text avatar is rejected as invalid content", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveMultipartNamed(router, path, map[string]routeNamedFile{
			"avatar": {name: "avatar.png", data: []byte("this is definitely not an image")},
		}, contractSessionToken(t, user))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code == 0 || envelope.MessageCode != "upload.image.invalidContent" {
			t.Fatalf("forged avatar envelope = %#v, want upload.image.invalidContent", envelope)
		}
	})
	t.Run("bmp avatar succeeds and keeps a canonical stored path", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveMultipartNamed(router, path, map[string]routeNamedFile{
			"avatar": {name: "avatar.bmp", data: tinyBMP},
		}, contractSessionToken(t, user))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code != 0 || envelope.MessageCode != "upload.success" {
			t.Fatalf("bmp avatar envelope = %#v, want success", envelope)
		}
		var result struct {
			AvatarUrl string `json:"avatarUrl"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			t.Fatalf("decode avatar result: %v", err)
		}
		if !strings.HasSuffix(result.AvatarUrl, ".bmp") {
			t.Fatalf("avatarUrl = %q, want a .bmp stored path", result.AvatarUrl)
		}
	})
}

// TestFileServeResponseHeaders 验证文件响应头（issue #408）：服务端按对象名的
// 规范化扩展名权威决定 Content-Type 并加 nosniff；图片内联，非图片强制附件。
func TestFileServeResponseHeaders(t *testing.T) {
	_, router, _ := setupDirectUploadRouteTest(t)
	seed := func(name, fileType string, data []byte) {
		t.Helper()
		if err := db4fileconnect.Connect().Create(&filedata.Entity{
			Name: name, Type: fileType, Data: data, Size: int64(len(data)),
			StorageStatus: filedata.StorageStatusReady,
		}).Error; err != nil {
			t.Fatalf("seed file row: %v", err)
		}
		t.Cleanup(func() {
			db4fileconnect.Connect().Where("name = ?", name).Delete(&filedata.Entity{})
		})
	}

	t.Run("known image extension serves canonical content type inline with nosniff", func(t *testing.T) {
		seed("2026/09/01/header.png", "image/png", contractTinyPNG)
		recorder := serveDirectUploadJSON(router, http.MethodGet, "/file/img/2026/09/01/header.png", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Type"); got != "image/png" {
			t.Fatalf("Content-Type = %q, want server-authoritative image/png", got)
		}
		if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := recorder.Header().Get("Content-Disposition"); got != "inline" {
			t.Fatalf("Content-Disposition = %q, want inline", got)
		}
	})

	t.Run("non-image legacy object is forced to attachment with octet-stream", func(t *testing.T) {
		seed("2026/09/01/legacy.bin", "application/octet-stream", []byte{0x00, 0x01, 0x02})
		recorder := serveDirectUploadJSON(router, http.MethodGet, "/file/img/2026/09/01/legacy.bin", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Type"); got != "application/octet-stream" {
			t.Fatalf("Content-Type = %q, want application/octet-stream", got)
		}
		if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename="legacy.bin"` {
			t.Fatalf("Content-Disposition = %q, want attachment", got)
		}
	})
}
