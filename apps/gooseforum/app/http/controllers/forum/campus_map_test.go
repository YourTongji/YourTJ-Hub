package forum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCampusMapPublicPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/map", CampusMap)

	t.Run("payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/map", nil)
		req.Header.Set("X-Goose-Page", "true")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("anonymous map status = %d: %s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("public map payload Cache-Control = %q, want no-store", got)
		}
		var payload struct {
			Component PageComponent  `json:"component"`
			Props     map[string]any `json:"props"`
			Meta      PageMeta       `json:"meta"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Component != PageComponentCampusMap || len(payload.Props) != 0 || !strings.HasSuffix(payload.Meta.Canonical, "/map") {
			t.Fatalf("unexpected map payload: %#v", payload)
		}
	})
	t.Run("document", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/map", nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `id="goose-app"`) || !strings.Contains(recorder.Body.String(), `"component":"campus.map"`) {
			t.Fatalf("map document lacks its bootstrap payload: %s", recorder.Body.String())
		}
		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("public map document Cache-Control = %q, want no-store", got)
		}
	})
}

func TestCampusMapPrivateCourseIntent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/map", CampusMap)

	for _, tc := range []struct {
		name    string
		pageReq bool
	}{
		{name: "payload", pageReq: true},
		{name: "document"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/map?mine=1", nil)
			if tc.pageReq {
				req.Header.Set("X-Goose-Page", "true")
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("private map status = %d: %s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Cache-Control"); got != "private, no-store" {
				t.Fatalf("private map Cache-Control = %q, want private, no-store", got)
			}
			if !tc.pageReq {
				if strings.Contains(recorder.Body.String(), `"events"`) || strings.Contains(recorder.Body.String(), `"timetable"`) {
					t.Fatalf("private course data appeared in map HTML: %s", recorder.Body.String())
				}
				return
			}

			var payload struct {
				Component PageComponent  `json:"component"`
				Props     map[string]any `json:"props"`
				Meta      PageMeta       `json:"meta"`
				URL       string         `json:"url"`
				Layout    struct {
					UmamiEnabled bool        `json:"umamiEnabled"`
					Site         SitePayload `json:"site"`
				} `json:"layout"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Component != PageComponentCampusMap || len(payload.Props) != 0 || payload.Layout.UmamiEnabled || payload.Layout.Site.ExternalLinks != "" {
				t.Fatalf("private map payload contains private data or scripts: %#v", payload)
			}
			if !strings.HasSuffix(payload.Meta.Canonical, "/map") || !strings.HasSuffix(payload.URL, "/map?mine=1") {
				t.Fatalf("private map URL marker/canonical changed unexpectedly: canonical=%q url=%q", payload.Meta.Canonical, payload.URL)
			}
		})
	}
}
