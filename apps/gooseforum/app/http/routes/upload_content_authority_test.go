package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// storedNameFromURL 从上传响应 URL 反解存储对象名（测试环境未配置 CDN 前缀，
// 公开路径固定为 /file/img/<name>）。
func storedNameFromURL(t *testing.T, url string) string {
	t.Helper()
	name, ok := strings.CutPrefix(url, "/file/img/")
	if !ok || name == "" {
		t.Fatalf("upload url = %q, want a /file/img/<name> path", url)
	}
	return name
}

type uploadedFileResult struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

func decodeUploadedFileResult(t *testing.T, envelope contractEnvelope) uploadedFileResult {
	t.Helper()
	var result uploadedFileResult
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode upload result %s: %v", envelope.Result, err)
	}
	return result
}

// TestMultipartImageUploadStoresDetectedType is AC-1 of issue #1393: a `foo.png`
// name with JPEG bytes is accepted, stored under the canonical `.jpg` extension
// with the detected MIME, and served as image/jpeg. The client's filename is
// still echoed back unchanged.
func TestMultipartImageUploadStoresDetectedType(t *testing.T) {
	conn, router, _ := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	jpegData := routeTinyJPEG(t)

	recorder := serveMultipartNamed(router, "/file/img-upload", map[string]routeNamedFile{
		"file": {name: "foo.png", data: jpegData},
	}, token)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 || envelope.MessageCode != "upload.success" {
		t.Fatalf("conflicting upload envelope = %#v, want success", envelope)
	}
	result := decodeUploadedFileResult(t, envelope)
	if result.Filename != "foo.png" {
		t.Fatalf("result.filename = %q, want the client filename foo.png", result.Filename)
	}
	name := storedNameFromURL(t, result.URL)
	if !strings.HasSuffix(name, ".jpg") {
		t.Fatalf("stored name = %q, want the detected type's .jpg extension", name)
	}
	// Only the extension follows the bytes; the generated uuid object stem is
	// untouched (uuid.New().String() is 36 chars with four dashes).
	if stem := strings.TrimSuffix(path.Base(name), ".jpg"); len(stem) != 36 || strings.Count(stem, "-") != 4 {
		t.Fatalf("stored basename stem = %q, want the generated uuid unchanged", stem)
	}
	if row := filedata.GetByName(name); row.Type != "image/jpeg" {
		t.Fatalf("stored assert_type = %q, want image/jpeg", row.Type)
	}

	read := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+name, "", token)
	if read.Code != http.StatusOK {
		t.Fatalf("GET stored upload status = %d, want 200: %s", read.Code, read.Body.String())
	}
	if got := read.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("GET stored upload Content-Type = %q, want image/jpeg", got)
	}
	if !bytes.Equal(read.Body.Bytes(), jpegData) {
		t.Fatal("GET stored upload returned different bytes than were uploaded")
	}
}

// TestMultipartImageUploadAcceptsJPEGWithJpegOnlyAllowlist is the issue #1393
// review regression: the detected-type allowlist check is by MIME, so a
// `.jpeg`-only allowlist accepts every real JPEG (both `.jpeg` and `.jpg`
// spellings) instead of rejecting image/jpeg for lacking the `.jpg` token. The
// stored name still uses the canonical `.jpg` extension.
func TestMultipartImageUploadAcceptsJPEGWithJpegOnlyAllowlist(t *testing.T) {
	conn, router, _ := setupDirectUploadRouteTest(t)
	persistContractPageConfig(t, conn, pageConfig.PostingSettings, map[string]any{
		"textControl": map[string]any{
			"minPostLength": 5, "maxPostLength": 1000, "minTitleLength": 2,
			"maxTitleLength": 50, "newUserPostCooldownMinutes": 0, "maxDailyTopicsPerUser": 7,
		},
		"uploadControl": map[string]any{
			"allowAttachments": true, "authorizedExtensions": []string{".jpeg"},
			"maxAttachmentSizeKb": 512, "maxDailyUploadsPerUser": 5,
			"newUserUploadCooldownMinutes": 0,
		},
		"llms": map[string]any{"enabled": false, "fullText": false, "files": false},
	})
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	jpegData := routeTinyJPEG(t)

	// The declared spelling must still be listed (unchanged pre-check); it is the
	// detected-type check that must not demand the canonical `.jpg` token.
	recorder := serveMultipartNamed(router, "/file/img-upload", map[string]routeNamedFile{
		"file": {name: "photo.jpeg", data: jpegData},
	}, token)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 || envelope.MessageCode != "upload.success" {
		t.Fatalf("photo.jpeg upload envelope = %#v, want success with a .jpeg-only allowlist", envelope)
	}
	name := storedNameFromURL(t, decodeUploadedFileResult(t, envelope).URL)
	if !strings.HasSuffix(name, ".jpg") {
		t.Fatalf("stored name = %q, want the canonical .jpg extension", name)
	}
}

// TestAvatarUploadStoresDetectedType is AC-1 on the avatar exit.
func TestAvatarUploadStoresDetectedType(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveMultipartNamed(router, "/api/upload-avatar", map[string]routeNamedFile{
		"avatar": {name: "avatar.png", data: routeTinyJPEG(t)},
	}, contractSessionToken(t, user))
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 || envelope.MessageCode != "upload.success" {
		t.Fatalf("conflicting avatar envelope = %#v, want success", envelope)
	}
	var result struct {
		AvatarURL string `json:"avatarUrl"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode avatar result: %v", err)
	}
	name := storedNameFromURL(t, result.AvatarURL)
	if !strings.HasSuffix(name, ".jpg") {
		t.Fatalf("avatar name = %q, want the detected type's .jpg extension", name)
	}
	if row := filedata.GetByName(name); row.Type != "image/jpeg" {
		t.Fatalf("avatar assert_type = %q, want image/jpeg", row.Type)
	}
}

// TestDirectUploadCompleteRenamesToDetectedType is AC-1/AC-4 on the direct
// upload exit: the object is renamed to the detected type's extension, the
// response carries the new name, the old name is no longer readable and the row
// type follows the bytes.
func TestDirectUploadCompleteRenamesToDetectedType(t *testing.T) {
	conn, router, provider := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	jpegData := routeTinyJPEG(t)

	_, declaredName := directUploadInit(t, router, token, "foo.png", int64(len(jpegData)))
	if declaredName == "" {
		t.Fatal("init returned empty name")
	}
	if !strings.HasSuffix(declaredName, ".png") {
		t.Fatalf("init name = %q, want the declared .png extension", declaredName)
	}
	provider.objects[declaredName] = routeFakeObject{data: jpegData, contentType: "image/png"}

	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete",
		mustMarshalJSON(t, map[string]string{"name": declaredName}), token)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 || envelope.MessageCode != "upload.success" {
		t.Fatalf("complete envelope = %#v, want success", envelope)
	}
	result := decodeUploadedFileResult(t, envelope)
	if !strings.HasSuffix(result.Filename, ".jpg") || !strings.Contains(result.URL, result.Filename) {
		t.Fatalf("complete result = %+v, want the detected .jpg name in filename and url", result)
	}
	// Only the extension is replaced: the object stem the presigned policy pinned
	// at init stays byte-identical.
	if before, after := strings.TrimSuffix(declaredName, path.Ext(declaredName)), strings.TrimSuffix(result.Filename, ".jpg"); before != after {
		t.Fatalf("rename changed the object stem: %q -> %q, want only the extension replaced", before, after)
	}
	if row := filedata.GetByName(result.Filename); row.Type != "image/jpeg" {
		t.Fatalf("completed assert_type = %q, want image/jpeg", row.Type)
	}
	if _, ok := provider.objects[declaredName]; ok {
		t.Fatal("object still stored under the client-declared name after the rename")
	}
	if read := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+declaredName, "", token); read.Code != http.StatusNotFound {
		t.Fatalf("GET old name status = %d, want 404 after the rename", read.Code)
	}
	read := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+result.Filename, "", token)
	if read.Code != http.StatusOK || read.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("GET renamed upload status=%d Content-Type=%q, want 200 image/jpeg", read.Code, read.Header().Get("Content-Type"))
	}
}

// uploadVerdict is one upload exit's answer for a given input, plus the user
// whose storage rows the caller can inspect for leftovers.
type uploadVerdict struct {
	envelope contractEnvelope
	userId   uint64
}

func multipartUploadVerdict(t *testing.T, filename string, data []byte) uploadVerdict {
	t.Helper()
	conn, router, _ := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveMultipartNamed(router, "/file/img-upload", map[string]routeNamedFile{
		"file": {name: filename, data: data},
	}, contractSessionToken(t, user))
	return uploadVerdict{envelope: decodeContractEnvelope(t, recorder), userId: user.Id}
}

func adminUploadVerdict(t *testing.T, filename string, data []byte) uploadVerdict {
	t.Helper()
	conn, router := setupAdminDataContractTest(t)
	manager := createContractSiteManager(t, conn)
	recorder := serveMultipartNamed(router, "/api/admin/img-upload", map[string]routeNamedFile{
		"file": {name: filename, data: data},
	}, contractSessionToken(t, manager))
	return uploadVerdict{envelope: decodeContractEnvelope(t, recorder), userId: manager.Id}
}

func avatarUploadVerdict(t *testing.T, filename string, data []byte) uploadVerdict {
	t.Helper()
	conn, router := setupAccountContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveMultipartNamed(router, "/api/upload-avatar", map[string]routeNamedFile{
		"avatar": {name: filename, data: data},
	}, contractSessionToken(t, user))
	return uploadVerdict{envelope: decodeContractEnvelope(t, recorder), userId: user.Id}
}

// directUploadVerdict drives init + complete. A zero-byte payload cannot pass
// init (it reports empty itself), so the pending row is created directly to
// exercise the complete-stage decision.
func directUploadVerdict(t *testing.T, filename string, data []byte) uploadVerdict {
	t.Helper()
	conn, router, provider := setupDirectUploadRouteTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	declaredType := "image/png"
	if declared, ok := imagepolicy.ContentTypeForFilename(filename); ok {
		declaredType = declared
	}
	name := storageservice.NewUploadName(filename, "2026/10/01")
	if len(data) > 0 {
		recorder, initiatedName := directUploadInitWithType(t, router, token, filename, declaredType, int64(len(data)))
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code != 0 {
			return uploadVerdict{envelope: envelope, userId: user.Id}
		}
		name = initiatedName
	} else if _, err := filedata.CreateFileMetadata(context.Background(), user.Id, name, declaredType, 0); err != nil {
		t.Fatalf("create pending metadata: %v", err)
	}
	provider.objects[name] = routeFakeObject{data: data, contentType: declaredType}

	recorder := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete",
		mustMarshalJSON(t, map[string]string{"name": name}), token)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 {
		// A rejected complete must remove both the object and the pending row.
		if _, ok := provider.objects[name]; ok {
			t.Fatal("object still present after rejected complete")
		}
		if row := filedata.GetByName(name); row.Id != 0 {
			t.Fatal("pending row still present after rejected complete")
		}
	}
	return uploadVerdict{envelope: envelope, userId: user.Id}
}

// TestUploadExitsShareContentVerdicts is AC-2/AC-3/AC-4 of issue #1393: every
// upload exit reports the same stable code for the same bytes, and a rejected
// upload never leaves a stored file behind.
func TestUploadExitsShareContentVerdicts(t *testing.T) {
	exits := []struct {
		name string
		send func(*testing.T, string, []byte) uploadVerdict
	}{
		{name: "multipart", send: multipartUploadVerdict},
		{name: "admin multipart", send: adminUploadVerdict},
		{name: "avatar", send: avatarUploadVerdict},
		{name: "direct complete", send: directUploadVerdict},
	}
	tests := []struct {
		name     string
		filename string
		data     []byte
		want     string
	}{
		{
			name: "text bytes are not a usable image", filename: "fake.png",
			data: []byte("this is definitely not an image"), want: "upload.image.invalidContent",
		},
		{
			name: "empty bytes are empty content", filename: "empty.png",
			data: nil, want: "upload.image.empty",
		},
		{
			name: "heic bytes are a known unsupported format", filename: "photo.png",
			data: routeHEICHeader, want: "upload.image.unsupported",
		},
		{
			name: "dangerous extensions are rejected before the content check", filename: "evil.svg",
			data: contractTinyPNG, want: "upload.extension.unsupported",
		},
	}
	for _, exit := range exits {
		for _, tt := range tests {
			t.Run(exit.name+"/"+tt.name, func(t *testing.T) {
				verdict := exit.send(t, tt.filename, tt.data)
				if verdict.envelope.Code == 0 || verdict.envelope.MessageCode != tt.want {
					t.Fatalf("envelope = %#v, want messageCode %q", verdict.envelope, tt.want)
				}
				if stored := filedata.CountUserUploadsToday(verdict.userId); stored != 0 {
					t.Fatalf("user uploads today = %d, want 0 after a rejected upload", stored)
				}
			})
		}
	}
}
