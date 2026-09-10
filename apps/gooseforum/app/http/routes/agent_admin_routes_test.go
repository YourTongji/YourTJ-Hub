package routes

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAgentAdminRoutesRegisteredAsPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, path := range []string{
		"/api/admin/agent-list",
		"/api/admin/agent-create",
		"/api/admin/agent-update",
		"/api/admin/agent-rotate-token",
		"/api/admin/agent-disable",
	} {
		if !registered[http.MethodPost+" "+path] {
			t.Errorf("POST %s was not registered", path)
		}
		if registered[http.MethodGet+" "+path] {
			t.Errorf("GET %s should not be registered", path)
		}
	}
}
