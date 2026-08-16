package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupForumReadContractTest 注册站点统计与聚合搜索两条公开只读路由，
// 中间件链与 route4api.go 的生产注册保持一致（search 挂可选 JWTAuth）。
func setupForumReadContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	router.GET("/api/forum/get-site-statistics", ginUpNP(api.GetSiteStatistics))
	router.GET("/api/forum/search", middleware.JWTAuth, UpQueryReq(forum.SearchJSON))
	return conn, router
}

func TestGetSiteStatisticsHTTPContract(t *testing.T) {
	_, router := setupForumReadContractTest(t)
	recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/get-site-statistics", "", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("site statistics status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	fixture := contractFixture(t, "site-statistics-success.json")
	// 计数器值依赖进程级共享 DB 的累积状态（users/topics/posts max id），
	// 只钉住键集合与数值类型，不钉具体数值。
	assertResultObjectKeysMatchFixture(t, response, fixture)
	var result map[string]any
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode site statistics result %q: %v", response.Result, err)
	}
	for key, value := range result {
		if _, ok := value.(float64); !ok {
			t.Fatalf("result.%s = %#v, want numeric counter", key, value)
		}
	}
}

func TestSearchJSONHTTPContract(t *testing.T) {
	t.Run("empty query short-circuits to an empty aggregate payload", func(t *testing.T) {
		_, router := setupForumReadContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/search", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("search empty query status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "forum-search-empty-query-success.json"))
	})

	t.Run("unconfigured search backend degrades to searchUnavailable", func(t *testing.T) {
		_, router := setupForumReadContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/search?q=contract", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("search unavailable status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "forum-search-unavailable.json"))
	})

	t.Run("malformed page returns strict 400", func(t *testing.T) {
		_, router := setupForumReadContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/search?q=contract&page=abc", "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("search parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "forum-search-parse-failed.json"))
	})
}
