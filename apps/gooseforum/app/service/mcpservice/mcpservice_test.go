package mcpservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// setupMCPServiceTestDB mirrors the route-level agent test setup: a shared
// sqlite in-memory DB with the tables the MCP tool handlers touch.
func setupMCPServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ratelimit.Default().ResetAll()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&agents.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
	); err != nil {
		t.Fatalf("migrate mcp tables: %v", err)
	}
	cleanMCPServiceTables(conn)
	hotdataserve.ClearRateLimitConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
	})
	return conn
}

func cleanMCPServiceTables(conn *gorm.DB) {
	conn.Where("1 = 1").Delete(&posts.Entity{})
	conn.Where("1 = 1").Delete(&topicCategoryIndex.Entity{})
	conn.Where("1 = 1").Delete(&topics.Entity{})
	conn.Where("1 = 1").Delete(&category.Entity{})
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func createMCPServiceAgent(t *testing.T, username string) (uint64, string) {
	t.Helper()
	result, err := agentservice.Create(agentservice.CreateParams{Username: username})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return result.Agent.UserId, result.Token
}

// connectMCPServer builds a server (per the writes flag) and returns a client
// session wired over in-memory transports so tests drive the real tool
// handlers without an HTTP round trip. A fixed agent identity is bound so tool
// handlers resolve a user even though no HTTP auth middleware runs here.
func connectMCPServer(t *testing.T, writes bool, agentID ...uint64) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	var svc *Service
	if len(agentID) > 0 && agentID[0] != 0 {
		svc = NewStdioService(agentID[0])
	} else {
		svc = NewService()
	}
	server := svc.buildServer(writes)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("connect server: %v", err)
	}
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func toolNames(t *testing.T, cs *mcp.ClientSession) map[string]bool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := make(map[string]bool, len(res.Tools))
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	return names
}

func TestToolSetDefaultReadOnly(t *testing.T) {
	cs := connectMCPServer(t, false)
	names := toolNames(t, cs)
	for _, want := range []string{"me", "list_topics", "get_posts", "search"} {
		if !names[want] {
			t.Errorf("read tool %q not registered", want)
		}
	}
	for _, unwanted := range []string{"create_topic", "create_post"} {
		if names[unwanted] {
			t.Errorf("write tool %q registered while writes disabled", unwanted)
		}
	}
}

func TestToolSetWritesEnabled(t *testing.T) {
	cs := connectMCPServer(t, true)
	names := toolNames(t, cs)
	for _, want := range []string{"me", "list_topics", "get_posts", "search", "create_topic", "create_post"} {
		if !names[want] {
			t.Errorf("tool %q not registered", want)
		}
	}
}

func TestGetServerHonorsWritesPreference(t *testing.T) {
	setupMCPServiceTestDB(t)
	svc := NewService()

	// Simulate a streamable session creation: getServer must pick the tool set
	// from the mcp.writes preference (default off).
	preferences.Set("mcp.writes", false)
	names := toolNames(t, connectMCPServer(t, false))
	if names["create_topic"] {
		t.Fatal("create_topic present when preference writes=false")
	}

	preferences.Set("mcp.writes", true)
	names = toolNames(t, connectMCPServer(t, true))
	if !names["create_topic"] || !names["create_post"] {
		t.Fatal("write tools missing when preference writes=true")
	}
	_ = svc
}

func TestMeTool(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-me")
	cs := connectMCPServer(t, true, agentID)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "me",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call me: %v", err)
	}
	if res.IsError {
		t.Fatalf("me returned error: %+v", res.Content)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("me structured content type %T", res.StructuredContent)
	}
	if id := asUint(out["agentId"]); id != agentID {
		t.Fatalf("me agentId = %d, want %d", id, agentID)
	}
}

func TestListTopicsTool(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-list")
	conn := db.Connect()
	conn.Create(&category.Entity{Id: 9001, Name: "general", Slug: "general"})
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 9001, Title: "MCP visible", UserId: agentID, Status: 1, ProcessStatus: 0, CategoryIds: []uint64{9001}, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	draft := topics.Entity{Id: 9002, Title: "Draft", UserId: agentID, Status: 0, ProcessStatus: 0, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&draft).Error; err != nil {
		t.Fatalf("create draft: %v", err)
	}

	cs := connectMCPServer(t, true, agentID)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_topics",
		Arguments: map[string]any{"page": 1, "pageSize": 10},
	})
	if err != nil {
		t.Fatalf("call list_topics: %v", err)
	}
	if res.IsError {
		t.Fatalf("list_topics returned error: %+v", res.Content)
	}
	out := res.StructuredContent.(map[string]any)
	list, _ := out["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("list_topics returned %d topics, want 1: %s", len(list), mustJSON(out))
	}
}

func TestCreateTopicTool(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-create-topic")
	conn := db.Connect()
	conn.Create(&category.Entity{Id: 9101, Name: "announce", Slug: "announce"})

	cs := connectMCPServer(t, true, agentID)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_topic",
		Arguments: map[string]any{
			"title":      "MCP published topic",
			"content":    "Topic body long enough for the default 5-char minimum.",
			"categoryId": []any{9101},
		},
	})
	if err != nil {
		t.Fatalf("call create_topic: %v", err)
	}
	if res.IsError {
		t.Fatalf("create_topic returned error: %+v", res.Content)
	}
	out := res.StructuredContent.(map[string]any)
	if id := asUint(out["topicId"]); id == 0 {
		t.Fatalf("create_topic returned empty topicId: %s", mustJSON(out))
	}
	topic := topics.Get(asUint(out["topicId"]))
	if topic.Id == 0 || topic.UserId != agentID || topic.Status != 1 {
		t.Fatalf("created topic = %#v", topic)
	}
}

func TestCreatePostTool(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-create-post")
	conn := db.Connect()
	conn.Create(&category.Entity{Id: 9102, Name: "meta", Slug: "meta"})
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 9201, Title: "Reply target", UserId: agentID, Status: 1, PostCount: 1, PostSeq: 1, CategoryIds: []uint64{9102}, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	first := posts.Entity{Id: 9201, TopicId: topic.Id, PostNo: 1, UserId: agentID, Content: "first", CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&first).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", first.Id)

	cs := connectMCPServer(t, true, agentID)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_post",
		Arguments: map[string]any{
			"topicId": 9201,
			"content": "A reply long enough for the default posting minimum.",
		},
	})
	if err != nil {
		t.Fatalf("call create_post: %v", err)
	}
	if res.IsError {
		t.Fatalf("create_post returned error: %+v", res.Content)
	}
	out := res.StructuredContent.(map[string]any)
	if id := asUint(out["id"]); id == 0 {
		t.Fatalf("create_post returned empty id: %s", mustJSON(out))
	}
	if no := asUint(out["postNo"]); no != 2 {
		t.Fatalf("create_post postNo = %d, want 2", no)
	}
}

func TestCreateTopicToolRateLimited(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-rate")
	conn := db.Connect()
	conn.Create(&category.Entity{Id: 9103, Name: "rl", Slug: "rl"})

	// topic.write default limitPerIp is 5; drive the shared quota until the
	// MCP handler refuses.
	cs := connectMCPServer(t, true, agentID)
	args := map[string]any{"title": "Rate test", "content": "Body long enough for the default minimum length.", "categoryId": []any{9103}}
	limited := false
	for i := 0; i < 20; i++ {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_topic", Arguments: args})
		if err != nil {
			t.Fatalf("call create_topic attempt %d: %v", i, err)
		}
		if res.IsError {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("create_topic never hit rate limit within 20 attempts")
	}
}

// HTTP authentication tests exercise the real streamable handler with the
// bearer-auth middleware.

func TestHTTPAuthMissingToken(t *testing.T) {
	setupMCPServiceTestDB(t)
	rec := serveMCPHTTP(t, http.MethodPost, "/mcp", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want 401", rec.Code)
	}
}

func TestHTTPAuthInvalidToken(t *testing.T) {
	setupMCPServiceTestDB(t)
	rec := serveMCPHTTP(t, http.MethodPost, "/mcp", "", "agt_invalid")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status = %d, want 401", rec.Code)
	}
}

func TestHTTPAuthValidToken(t *testing.T) {
	setupMCPServiceTestDB(t)
	_, token := createMCPServiceAgent(t, "mcp-http")
	// An initialize request with a valid token must not be rejected by auth.
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`
	rec := serveMCPHTTP(t, http.MethodPost, "/mcp", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid token initialize status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func serveMCPHTTP(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	svc := NewService()
	handler := svc.HTTPHandler()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestHTTPEndToEnd drives the full streamable HTTP path with a real agt_ token
// over an httptest server: initialize -> tools/list -> tools/call(me). It
// covers the verifier -> TokenInfo -> userID chain that the in-memory tests
// cannot reach.
func TestHTTPEndToEnd(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, token := createMCPServiceAgent(t, "mcp-e2e-http")

	svc := NewService()
	srv := httptest.NewServer(svc.HTTPHandler())
	defer srv.Close()

	// Inject the bearer token into every outgoing request via a RoundTripper.
	base := http.DefaultTransport
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+token)
		return base.RoundTrip(req)
	})
	client := &http.Client{Transport: rt}

	cs, err := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "1"}, nil).
		Connect(context.Background(), &mcp.StreamableClientTransport{
			Endpoint:             srv.URL + "/mcp",
			HTTPClient:           client,
			DisableStandaloneSSE: true,
		}, nil)
	if err != nil {
		t.Fatalf("connect over HTTP: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools over HTTP: %v", err)
	}
	found := false
	for _, tool := range res.Tools {
		if tool.Name == "me" {
			found = true
		}
	}
	if !found {
		t.Fatalf("me tool not listed over HTTP: %+v", res.Tools)
	}

	call, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "me", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call me over HTTP: %v", err)
	}
	if call.IsError {
		t.Fatalf("me over HTTP returned error: %+v", call.Content)
	}
	out, ok := call.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("me over HTTP structured content type %T", call.StructuredContent)
	}
	if id := asUint(out["agentId"]); id != agentID {
		t.Fatalf("me over HTTP agentId = %d, want %d", id, agentID)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestVerifierIgnoresSpoofedXForwardedFor asserts the auth verifier does not
// trust a raw X-Forwarded-For header when there is no gin TrustedProxies
// context: the IP used for rate limiting must fall back to the peer address.
func TestVerifierIgnoresSpoofedXForwardedFor(t *testing.T) {
	setupMCPServiceTestDB(t)
	_, token := createMCPServiceAgent(t, "mcp-spoof")
	svc := NewService()

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4") // spoofed, no trusted-proxy context
	ti, err := svc.verifier(context.Background(), token, req)
	if err != nil {
		t.Fatalf("verifier rejected valid token: %v", err)
	}
	if got := ti.Extra["clientIP"]; got != "203.0.113.5" {
		t.Fatalf("clientIP = %q, want peer address 203.0.113.5 (XFF must not be trusted)", got)
	}
}

// TestResultToMapPreservesLargeIDs guards against float64 rounding corrupting
// uint64 IDs above 2^53 when MCP output is marshaled.
func TestResultToMapPreservesLargeIDs(t *testing.T) {
	in := map[string]any{"id": uint64(9007199254740993)} // 2^53+1
	out, err := resultToMap(in)
	if err != nil {
		t.Fatalf("resultToMap: %v", err)
	}
	b, err := json.Marshal(out["id"])
	if err != nil {
		t.Fatalf("marshal id: %v", err)
	}
	if got := string(b); got != "9007199254740993" {
		t.Fatalf("large id = %s, want exact 9007199254740993 (float64 would corrupt it)", got)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
