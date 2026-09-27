package routes

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/gin-gonic/gin"
)

func TestEventRouteRejectsQueryTokenAndForeignCookieBeforeStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/forum/events", middleware.StreamOriginProtection, middleware.JWTAuthCheck,
		middleware.NoUpdateUserActivity, api.StreamEvents)

	queryOnly := httptest.NewRequest(http.MethodGet, "/api/forum/events?access_token=secret", nil)
	queryReply := httptest.NewRecorder()
	engine.ServeHTTP(queryReply, queryOnly)
	if queryReply.Code != http.StatusUnauthorized || queryReply.Header().Get("Content-Type") == "text/event-stream" {
		t.Fatalf("query token response: %d %q", queryReply.Code, queryReply.Header().Get("Content-Type"))
	}

	foreign := httptest.NewRequest(http.MethodGet, "https://forum.example/api/forum/events", nil)
	foreign.Header.Set("Origin", "https://evil.example")
	foreign.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid"})
	foreignReply := httptest.NewRecorder()
	engine.ServeHTTP(foreignReply, foreign)
	if foreignReply.Code != http.StatusForbidden || foreignReply.Header().Get("New-Token") != "" {
		t.Fatalf("foreign cookie response: %d new-token=%q", foreignReply.Code, foreignReply.Header().Get("New-Token"))
	}
}

func TestAuthenticatedEventRouteEmitsHello(t *testing.T) {
	conn, engine := setupHTTPContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	engine.GET("/api/forum/events", middleware.StreamOriginProtection, middleware.JWTAuthCheck,
		middleware.NoUpdateUserActivity, api.StreamEvents)
	server := httptest.NewServer(engine)
	defer server.Close()
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/api/forum/events", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	reader := bufio.NewReader(response.Body)
	first, err := reader.ReadString('\n')
	if err != nil || !strings.Contains(first, "event: hello") {
		t.Fatalf("first line=%q err=%v", first, err)
	}
}
