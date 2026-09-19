package forum

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCampusPageHasNoPrivateBootstrapData(t *testing.T) {
	r := gin.New()
	r.GET("/campus", Campus)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/campus", nil)
	req.Header.Set("X-Goose-Page", "true")
	r.ServeHTTP(w, req)
	var payload struct {
		Component string         `json:"component"`
		Props     map[string]any `json:"props"`
	}
	if json.Unmarshal(w.Body.Bytes(), &payload) != nil || payload.Component != "campus.home" || len(payload.Props) != 0 {
		t.Fatal("campus bootstrap contract drift")
	}
}

func TestCampusLayoutExcludesSessionReplayAndCustomScripts(t *testing.T) {
	layout := campusLayout(LayoutPayload{UmamiEnabled: true, Site: SitePayload{ExternalLinks: `<script src="tracker"></script>`}})
	if layout.UmamiEnabled || layout.Site.ExternalLinks != "" {
		t.Fatal("private school data page allows analytics or custom scripts")
	}
}
