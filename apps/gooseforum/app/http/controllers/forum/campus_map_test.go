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
	})
}
