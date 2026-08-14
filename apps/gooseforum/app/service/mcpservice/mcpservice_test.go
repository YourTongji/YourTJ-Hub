package mcpservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
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
		&postRevisions.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&pageConfig.Entity{},
	); err != nil {
		t.Fatalf("migrate mcp tables: %v", err)
	}
	cleanMCPServiceTables(conn)
	hotdataserve.ClearRateLimitConfigCache()
	hotdataserve.ClearMCPSettingsConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
		hotdataserve.ClearMCPSettingsConfigCache()
	})
	return conn
}

func cleanMCPServiceTables(conn *gorm.DB) {
	conn.Where("1 = 1").Delete(&posts.Entity{})
	conn.Where("1 = 1").Delete(&topicCategoryIndex.Entity{})
	conn.Where("1 = 1").Delete(&postRevisions.Entity{})
	conn.Where("1 = 1").Delete(&topics.Entity{})
	conn.Where("1 = 1").Delete(&category.Entity{})
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
	conn.Where("page_type = ?", pageConfig.MCPSettings).Delete(&pageConfig.Entity{})
}

// setMCPSettings writes the admin-panel MCP settings row and clears the
// hotdataserve cache so the next read observes the new value.
func setMCPSettings(t *testing.T, conn *gorm.DB, cfg pageConfig.MCPSettingsConfig) {
	t.Helper()
	conn.Where("page_type = ?", pageConfig.MCPSettings).Delete(&pageConfig.Entity{})
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.MCPSettings, Config: jsonopt.Encode(cfg)}).Error; err != nil {
		t.Fatalf("write mcp settings: %v", err)
	}
	hotdataserve.ClearMCPSettingsConfigCache()
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
	conn := setupMCPServiceTestDB(t)
	svc := NewService()

	// Simulate a streamable session creation: getServer must pick the tool set
	// from the admin-panel MCP writes setting (default off).
	setMCPSettings(t, conn, pageConfig.MCPSettingsConfig{Enabled: true, Writes: false})
	names := toolNames(t, connectMCPServer(t, false))
	if names["create_topic"] {
		t.Fatal("create_topic present when writes=false")
	}

	setMCPSettings(t, conn, pageConfig.MCPSettingsConfig{Enabled: true, Writes: true})
	names = toolNames(t, connectMCPServer(t, true))
	if !names["create_topic"] || !names["create_post"] {
		t.Fatal("write tools missing when writes=true")
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

// TestSessionTimeoutReclaimsIdleSession asserts an idle streamable-HTTP session
// is closed by the SDK after the configured SessionTimeout, so repeated
// initialize calls without DELETE do not accumulate sessions/goroutines
// forever. A short sessionTimeout is injected; after waiting past it, a
// request carrying the old session ID must no longer be accepted.
func TestSessionTimeoutReclaimsIdleSession(t *testing.T) {
	setupMCPServiceTestDB(t)
	_, token := createMCPServiceAgent(t, "mcp-timeout")

	svc := NewService()
	svc.sessionTimeout = 200 * time.Millisecond
	srv := httptest.NewServer(svc.HTTPHandler())
	defer srv.Close()

	base := http.DefaultTransport
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+token)
		return base.RoundTrip(req)
	})
	client := &http.Client{Transport: rt}

	// initialize -> captures the Mcp-Session-Id the server assigned.
	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`
	initReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", strings.NewReader(initBody))
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("Accept", "application/json, text/event-stream")
	initResp, err := client.Do(initReq)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	initResp.Body.Close()
	sessionID := initResp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("initialize returned no Mcp-Session-Id")
	}

	// Wait past the session timeout, then reuse the same session ID: the SDK
	// must have reaped the session, so the request is answered with 404 rather
	// than being routed to the (now-closed) session.
	time.Sleep(500 * time.Millisecond)

	ping := `{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}`
	pingReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", strings.NewReader(ping))
	pingReq.Header.Set("Content-Type", "application/json")
	pingReq.Header.Set("Accept", "application/json, text/event-stream")
	pingReq.Header.Set("Mcp-Session-Id", sessionID)
	pingResp, err := client.Do(pingReq)
	if err != nil {
		t.Fatalf("ping after timeout: %v", err)
	}
	defer pingResp.Body.Close()
	if pingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("request with stale session ID after timeout: status = %d, want 404 (session reclaimed)", pingResp.StatusCode)
	}
}

// TestRecoverToolHandlerConvertsPanic asserts a panic inside a tool handler is
// converted to a tool error instead of crashing the process. The go-sdk runs
// handlers on its own goroutines with no recover(); without the wrapper the
// whole process (including the REST server) would die.
func TestRecoverToolHandlerConvertsPanic(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "boom",
		Description: "panics",
		InputSchema: objectSchema(nil, nil),
	}, recoverToolHandler("boom", func(_ context.Context, _ *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		panic("simulated nil dereference")
	}))

	ctx := context.Background()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil).
		Connect(ctx, mustInMemoryTransport(t, server), nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "boom", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call boom: %v", err)
	}
	if !res.IsError {
		t.Fatalf("boom returned success; want tool error after panic: %+v", res.Content)
	}
}

func mustInMemoryTransport(t *testing.T, server *mcp.Server) *mcp.InMemoryTransport {
	t.Helper()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), t1, nil); err != nil {
		t.Fatalf("connect server: %v", err)
	}
	return t2
}

// TestAsIntClampsOutOfRange asserts float64/int64 values outside the int range
// are clamped rather than left to Go's implementation-defined float→int
// overflow, which on amd64 would otherwise yield max-int64 and later wrap an
// OFFSET computation negative.
func TestAsIntClampsOutOfRange(t *testing.T) {
	if got := asInt(1e30); got != intMax {
		t.Fatalf("asInt(1e30) = %d, want clamped intMax %d", got, intMax)
	}
	if got := asInt(-1e30); got != intMin {
		t.Fatalf("asInt(-1e30) = %d, want clamped intMin %d", got, intMin)
	}
	if got := asInt(json.Number("1e30")); got != intMax {
		t.Fatalf("asInt(json.Number(\"1e30\")) = %d, want intMax %d", got, intMax)
	}
	// Normal values pass through untouched.
	if got := asInt(42); got != 42 {
		t.Fatalf("asInt(42) = %d, want 42", got)
	}
	// A large but in-range int64 is preserved (no clamp triggered).
	if got := asInt(int64(1 << 40)); got != 1<<40 {
		t.Fatalf("asInt(1<<40) = %d, want %d", got, 1<<40)
	}
}

// TestSchemaBindsPageBounds asserts page/pageSize carry a Maximum so out-of-range
// pagination values are rejected by schema validation before reaching asInt.
func TestSchemaBindsPageBounds(t *testing.T) {
	cs := connectMCPServer(t, false)
	// Ask the client to invoke list_topics with page far beyond any Maximum; the
	// SDK's schema validation must reject it as an invalid argument, not pass it
	// through to a wrapping OFFSET computation.
	callRes, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_topics", Arguments: map[string]any{"page": 1e30}})
	if err != nil {
		t.Fatalf("call list_topics with page=1e30: %v", err)
	}
	if !callRes.IsError {
		t.Fatalf("list_topics with page=1e30 succeeded; want schema rejection, got %+v", callRes.Content)
	}
}

// TestStdioWriteRateLimitedWithIPOnlyConfig asserts that in the mcp-stdio
// transport (no HTTP layer, no client IP) writes are still rate limited when
// the config relies on the IP dimension alone (limitPerUser=0). The fix charges
// the IP quota per-agent (action+":ip:agent:<id>") when ip is empty, so a
// config that sets only limitPerIp does not leave stdio writes unlimited.
func TestStdioWriteRateLimitedWithIPOnlyConfig(t *testing.T) {
	conn := setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-rate-iponly")
	conn.Create(&category.Entity{Id: 9104, Name: "rl2", Slug: "rl2"})

	// Only the IP dimension is configured for topic.write; the per-user
	// dimension is explicitly zero.
	custom := pageConfig.RateLimitConfig{
		Enabled:   true,
		SkipAdmin: false,
		Actions: []pageConfig.RateLimitRule{
			{Action: "topic.write", WindowSeconds: 60, LimitPerIp: 1, LimitPerUser: 0},
		},
	}
	conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
	conn.Create(&pageConfig.Entity{PageType: pageConfig.RateLimitSettings, Config: jsonopt.Encode(custom)})
	hotdataserve.ClearRateLimitConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearRateLimitConfigCache()
	})

	cs := connectMCPServer(t, true, agentID)
	args := map[string]any{"title": "Rate test", "content": "Body long enough for the default minimum length.", "categoryId": []any{9104}}
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
		t.Fatal("create_topic never hit rate limit with IP-only config; stdio writes should still be bounded")
	}
}

// deadlineRecordingWriter records SetWriteDeadline calls made through
// http.ResponseController so tests can assert the handler's per-method
// deadline policy.
type deadlineRecordingWriter struct {
	http.ResponseWriter
	deadlines []time.Time
}

func (w *deadlineRecordingWriter) SetWriteDeadline(t time.Time) error {
	w.deadlines = append(w.deadlines, t)
	return nil
}

func (w *deadlineRecordingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// TestHTTPWriteDeadlineByMethod asserts the /mcp handler applies a finite
// write deadline to POSTs and none to GET SSE streams. The SDK pauses the
// idle-session timer for the duration of an in-flight POST, so a client that
// stops reading its response would otherwise pin the connection, session, and
// goroutines forever; the finite deadline bounds that. GET streams are
// long-lived and bounded by the session timeout instead.
func TestHTTPWriteDeadlineByMethod(t *testing.T) {
	setupMCPServiceTestDB(t)
	handler := NewService().HTTPHandler()

	postRec := httptest.NewRecorder()
	postW := &deadlineRecordingWriter{ResponseWriter: postRec}
	handler.ServeHTTP(postW, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`)))
	if len(postW.deadlines) == 0 {
		t.Fatal("POST /mcp: no SetWriteDeadline call recorded")
	}
	if d := postW.deadlines[0]; d.IsZero() {
		t.Fatal("POST /mcp: write deadline is zero, want a finite deadline")
	} else if got := time.Until(d); got < 30*time.Second || got > 90*time.Second {
		t.Fatalf("POST /mcp: write deadline in %v, want ~60s", got)
	}

	getRec := httptest.NewRecorder()
	getW := &deadlineRecordingWriter{ResponseWriter: getRec}
	handler.ServeHTTP(getW, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if len(getW.deadlines) == 0 {
		t.Fatal("GET /mcp: no SetWriteDeadline call recorded")
	}
	if !getW.deadlines[0].IsZero() {
		t.Fatalf("GET /mcp: write deadline is finite (%v), want zero (unlimited SSE stream)", getW.deadlines[0])
	}
}

// TestSchemaSortEnum asserts the list_topics sort parameter is constrained to
// the supported values (latest/hot/popular/new), so an LLM cannot invent a
// sort key that silently falls back to the default ordering.
func TestSchemaSortEnum(t *testing.T) {
	setupMCPServiceTestDB(t)
	agentID, _ := createMCPServiceAgent(t, "mcp-sort-enum")
	conn := db.Connect()
	// Use a high ID: earlier tests create topics through the real service with
	// auto-increment IDs (which continue from the explicit IDs the tests
	// insert), and cleanup soft-deletes, so a reused low ID would collide with
	// a soft-deleted row.
	conn.Create(&category.Entity{Id: 9502, Name: "enum", Slug: "enum"})
	now := time.Now().Add(-time.Hour)
	if err := conn.Create(&topics.Entity{Id: 9501, Title: "Enum topic", UserId: agentID, Status: 1, ProcessStatus: 0, CategoryIds: []uint64{9502}, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	cs := connectMCPServer(t, true, agentID)

	// A valid sort value must pass schema validation.
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_topics",
		Arguments: map[string]any{"page": 1, "pageSize": 10, "sort": "latest"},
	})
	if err != nil {
		t.Fatalf("call list_topics sort=latest: %v", err)
	}
	if res.IsError {
		t.Fatalf("list_topics sort=latest returned error: %+v", res.Content)
	}

	// An out-of-enum value must be rejected by schema validation before the
	// handler runs.
	res, err = cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_topics",
		Arguments: map[string]any{"page": 1, "pageSize": 10, "sort": "bogus"},
	})
	if err != nil {
		t.Fatalf("call list_topics sort=bogus: %v", err)
	}
	if !res.IsError {
		t.Fatalf("list_topics sort=bogus succeeded; want schema enum rejection: %+v", res.Content)
	}
}
