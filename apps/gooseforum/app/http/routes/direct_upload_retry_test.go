package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

func TestDirectImageUploadCompleteRecoversLostResponse(t *testing.T) {
	for _, declared := range []struct{ filename, contentType string }{{"image.png", "image/png"}, {"image.jpeg", "image/jpeg"}} {
		t.Run(declared.filename, func(t *testing.T) {
			conn, router, provider := setupDirectUploadRouteTest(t)
			user := createHTTPContractUser(t, conn, contractTestID())
			other := createHTTPContractUser(t, conn, contractTestID())
			token := contractSessionToken(t, user)
			_, name := directUploadInitWithType(t, router, token, declared.filename, declared.contentType, int64(len(contractTinyPNG)))
			if name == "" {
				t.Fatal("init returned empty name")
			}
			provider.objects[name] = routeFakeObject{data: contractTinyPNG, contentType: declared.contentType}
			type completion struct {
				Code        int    `json:"code"`
				MessageCode string `json:"messageCode"`
				Result      struct {
					Filename string `json:"filename"`
					URL      string `json:"url"`
					Size     int64  `json:"size"`
				} `json:"result"`
			}
			complete := func(key, session string) completion {
				t.Helper()
				response := serveDirectUploadJSON(router, http.MethodPost, "/file/img-upload/complete", mustMarshalJSON(t, map[string]string{"name": key}), session)
				if response.Code != http.StatusOK {
					t.Fatalf("complete status %d: %s", response.Code, response.Body.String())
				}
				var got completion
				if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				return got
			}
			first := complete(name, token)
			if first.Code != 0 || first.Result.Filename == "" || first.Result.Filename == name {
				t.Fatalf("first complete failed: %+v", first)
			}
			foreign := complete(name, contractSessionToken(t, other))
			if foreign.MessageCode != "page.notFound" || foreign.Result.Filename != "" || foreign.Result.URL != "" {
				t.Fatalf("foreign retry leaked result: %+v", foreign)
			}
			// The client knows only the init name after a lost response. Replaying its
			// still-valid upload policy must not change the result recovered by retry.
			provider.objects[name] = routeFakeObject{data: []byte("invalid replacement"), contentType: declared.contentType}
			for _, key := range []string{name, first.Result.Filename} {
				got := complete(key, token)
				if got.Code != 0 || got.Result != first.Result {
					t.Fatalf("retry %q = %+v, want %+v", key, got, first)
				}
			}
			read := serveDirectUploadJSON(router, http.MethodGet, "/file/img/"+name, "", token)
			if read.Code != http.StatusNotFound {
				t.Fatalf("temporary key became readable: %d", read.Code)
			}
			if !bytes.Equal(provider.objects[first.Result.Filename].data, contractTinyPNG) {
				t.Fatal("retry changed validated snapshot")
			}
			if _, err := storageservice.CleanupPending(context.Background(), time.Now().Add(time.Hour), 100); err != nil {
				t.Fatal(err)
			}
			if _, exists := provider.objects[name]; exists {
				t.Fatal("temporary source was not cleaned up")
			}
			got := complete(name, token)
			if got.Code != 0 || got.Result != first.Result {
				t.Fatalf("source cleanup lost result: %+v", got)
			}
		})
	}
}
