package routes

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestPkAdminRoutesRegistered 验证排课数据同步管理端 API（issue #248）正确挂载到
// SiteManager 权限路由组：sync-calendar 仅 POST，sync-status 仅 GET。
func TestPkAdminRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	if !registered[http.MethodPost+" /api/admin/pk/sync-calendar"] {
		t.Errorf("POST /api/admin/pk/sync-calendar was not registered")
	}
	if registered[http.MethodGet+" /api/admin/pk/sync-calendar"] {
		t.Errorf("GET /api/admin/pk/sync-calendar should not be registered")
	}
	if !registered[http.MethodGet+" /api/admin/pk/sync-status"] {
		t.Errorf("GET /api/admin/pk/sync-status was not registered")
	}
	if registered[http.MethodPost+" /api/admin/pk/sync-status"] {
		t.Errorf("POST /api/admin/pk/sync-status should not be registered")
	}
}
