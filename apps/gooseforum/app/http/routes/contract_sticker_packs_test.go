package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"gorm.io/gorm"
)

// 官方表情包分组（sticker-packs / sticker-pack-save / sticker-pack-delete /
// sticker-batch 与公开 forum/sticker-packs）以及上传型市场表情包
// （sticker-market-pack-create / sticker-market-upload）的契约测试。

const contractCreatedPack = "p-1a2b3c4d"

// seedContractPack stores the custom pack the fixtures describe.
func seedContractPack(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Create(&sticker.OfficialPack{Key: contractCreatedPack, Name: "节日", SortOrder: 1000}).Error; err != nil {
		t.Fatalf("seed pack %s: %v", contractCreatedPack, err)
	}
}

func TestStickerPacksHTTPContract(t *testing.T) {
	t.Run("public packs list enabled official groups", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-000000000001.png", 1, true)
		seedContractSticker(t, conn, contractStickerSecondID, "hidden", "stickers/9f1c2d3e-0000-4000-8000-000000000002.png", 2, false)
		seedContractPack(t, conn)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/sticker-packs", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "forum-sticker-packs-success.json"))
	})

	t.Run("admin packs include empty created packs", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-000000000001.png", 1, true)
		seedContractSticker(t, conn, contractStickerSecondID, "梗图-冲", "", 2, false)
		seedContractPack(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodGet, "/api/admin/sticker-packs", "", "admin-sticker-packs-success.json")
	})

	t.Run("pack save edits a pack and validates input", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractPack(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-pack-save", `{"key":"p-1a2b3c4d","name":"节日表情","sortOrder":3}`, "admin-sticker-pack-save-success.json")
		var row sticker.OfficialPack
		if err := conn.Where("pack_key = ?", contractCreatedPack).First(&row).Error; err != nil || row.Name != "节日表情" || row.SortOrder != 3 {
			t.Fatalf("saved pack = %+v err=%v", row, err)
		}
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-pack-save", `{"name":"  "}`, "admin-sticker-name-required.json")
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-pack-save", `{"key":"missing","name":"x"}`, "invalid-params.json")

		recorder := serveAdminStickersRaw(t, conn, router, http.MethodPost, "/api/admin/sticker-pack-save", `{"name":"新建"}`)
		created := decodeContractEnvelope(t, recorder)
		assertResultObjectKeysMatchFixture(t, created, contractFixture(t, "admin-sticker-pack-save-success.json"))
		var result struct{ Key string }
		if err := json.Unmarshal(created.Result, &result); err != nil || len(result.Key) != len(contractCreatedPack) || result.Key[:2] != "p-" {
			t.Fatalf("created key = %q err=%v", result.Key, err)
		}
	})

	t.Run("pack delete moves stickers to the default pack", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractPack(t, conn)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/a.png", 1, true)
		seedContractSticker(t, conn, contractStickerSecondID, "wink", "stickers/b.png", 2, true)
		conn.Model(&sticker.Entity{}).Where("id IN ?", []uint64{contractStickerID, contractStickerSecondID}).Update("pack", contractCreatedPack)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-pack-delete", `{"key":"p-1a2b3c4d"}`, "admin-sticker-pack-delete-success.json")
		var count int64
		conn.Model(&sticker.Entity{}).Where("pack = ?", sticker.PackOfficial).Count(&count)
		if count != 2 {
			t.Fatalf("stickers in the default pack = %d, want 2", count)
		}
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-pack-delete", `{"key":"official"}`, "invalid-params.json")
	})

	t.Run("batch disables selected stickers", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/a.png", 1, true)
		seedContractSticker(t, conn, contractStickerSecondID, "wink", "stickers/b.png", 2, true)
		body := fmt.Sprintf(`{"ids":[%d,%d],"action":"disable"}`, contractStickerID, contractStickerSecondID)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-batch", body, "admin-sticker-batch-success.json")
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-batch", `{"ids":[1],"action":"archive"}`, "invalid-params.json")
	})

	t.Run("import accepts one image into a pack", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractPack(t, conn)
		manager := createContractSiteManager(t, conn)
		recorder := serveStickerUpload(router, "/api/admin/sticker-import", map[string]string{"pack": contractCreatedPack}, "party.png", contractTinyPNG, contractSessionToken(t, manager))
		envelope := decodeContractEnvelope(t, recorder)
		assertResultObjectKeysMatchFixture(t, envelope, contractFixture(t, "admin-sticker-import-success.json"))
		var row sticker.Entity
		if err := conn.Where("name = ?", "party").First(&row).Error; err != nil || row.Pack != contractCreatedPack {
			t.Fatalf("imported sticker = %+v err=%v", row, err)
		}
		t.Cleanup(func() { conn.Where("name = ?", "party").Delete(&sticker.Entity{}) })
	})

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/sticker-packs"},
		{http.MethodPost, "/api/admin/sticker-pack-save"},
		{http.MethodPost, "/api/admin/sticker-pack-delete"},
		{http.MethodPost, "/api/admin/sticker-batch"},
	} {
		t.Run(route.path, func(t *testing.T) {
			adminStickersGuardScenarios(t, route.method, route.path)
		})
	}
}

// serveStickerUpload posts a multipart upload with extra form fields.
func serveStickerUpload(router http.Handler, path string, fields map[string]string, fileName string, data []byte, token string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		_ = writer.WriteField(name, value)
	}
	if data != nil {
		part, _ := writer.CreateFormFile("file", fileName)
		_, _ = part.Write(data)
	}
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestUploadedMarketPackHTTPContract(t *testing.T) {
	conn, router := setupStickerMarketContractTest(t)
	token := contractSessionToken(t, createContractSiteManager(t, conn))

	recorder := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/sticker-market-pack-create", `{"name":"自制表情","topicLabel":"原创"}`, token)
	created := decodeContractEnvelope(t, recorder)
	assertResultObjectKeysMatchFixture(t, created, contractFixture(t, "admin-sticker-market-pack-create-success.json"))
	var pack struct {
		ID      uint64 `json:"id"`
		Source  string `json:"source"`
		Name    string `json:"name"`
		IconURL string `json:"iconUrl"`
	}
	if err := json.Unmarshal(created.Result, &pack); err != nil || pack.Source != "upload" || pack.Name != "自制表情" || pack.IconURL != "" {
		t.Fatalf("created pack = %+v err=%v", pack, err)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/sticker-market-pack-create", `{"name":" "}`, token)), contractFixture(t, "admin-sticker-name-required.json"))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/sticker-market-pack-create", `{"name":"x","topic":"missing"}`, token)), contractFixture(t, "invalid-params.json"))

	fields := map[string]string{"packId": fmt.Sprint(pack.ID)}
	archive := contractStickerZip(t, map[string][]byte{"happy.png": contractTinyPNG, "again.png": contractTinyPNG, "broken.png": []byte("not an image")})
	uploaded := decodeContractEnvelope(t, serveStickerUpload(router, "/api/admin/sticker-market-upload", fields, "pack.zip", archive, token))
	fixture := contractFixture(t, "admin-sticker-market-upload-success.json")
	assertResultObjectKeysMatchFixture(t, uploaded, fixture)
	var outcome struct {
		Result map[string]any `json:"result"`
		Pack   map[string]any `json:"pack"`
	}
	var want struct {
		Result map[string]any `json:"result"`
		Pack   map[string]any `json:"pack"`
	}
	if err := json.Unmarshal(uploaded.Result, &outcome); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(fixture.Result, &want)
	if fmt.Sprint(outcome.Result) != fmt.Sprint(want.Result) {
		t.Fatalf("upload outcome = %v, want %v", outcome.Result, want.Result)
	}
	if len(outcome.Pack) != len(want.Pack) || outcome.Pack["status"] != "ready" || outcome.Pack["itemCount"] != float64(1) || outcome.Pack["coverUrl"] == "" {
		t.Fatalf("uploaded pack = %v", outcome.Pack)
	}

	assertFixtureEnvelope(t, decodeContractEnvelope(t, serveStickerUpload(router, "/api/admin/sticker-market-upload", fields, "notes.txt", []byte("text"), token)), contractFixture(t, "admin-sticker-import-invalid-zip.json"))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, serveStickerUpload(router, "/api/admin/sticker-market-upload", fields, "a.png", nil, token)), contractFixture(t, "admin-img-upload-file-missing.json"))
	missing := map[string]string{"packId": "999999"}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, serveStickerUpload(router, "/api/admin/sticker-market-upload", missing, "a.png", contractTinyPNG, token)), contractFixture(t, "admin-sticker-not-found.json"))
}
