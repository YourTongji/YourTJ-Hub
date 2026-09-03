package routes

// HEAD route compatibility contract (issue #411).
//
// Gin v1.12 does NOT auto-register or auto-route HEAD for GET handlers: a
// HEAD request only matches a route that was explicitly registered for HEAD.
// In this codebase the only explicit HEAD registrations come from
// StaticFS("assets", ...) / StaticFS("static", ...) (gin registers GET+HEAD
// for StaticFS) and from Any("/mcp"). Every other route — SSR pages, /health,
// robots.txt, JSON APIs, file downloads — is GET-only, so a HEAD request to
// those paths falls through to NoRoute and answers 404 with the same JSON
// body the corresponding anonymous GET would produce for an unregistered
// method. The Go net/http server then strips the body on the wire (RFC 9110:
// HEAD responses never carry a body), which is what CDNs and reverse proxies
// observe.
//
// The contract is therefore:
//   - static asset paths (/assets/*filepath, /static/*filepath): HEAD == GET
//     status and headers, empty body — safe for cache/HEAD probes.
//   - everything else (dynamic GET routes): HEAD 404 even when GET 200 —
//     load balancer and monitor probes must use GET, and HEAD never executes
//     a write-side controller because no HEAD route is ever registered.
//
// See docs/architecture/contracts-and-data.md (HTTP method contract) for the
// authoritative statement of this matrix.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
)

// staticBadgeAsset is a committed static file (not frontend build output), so
// HEAD contract assertions run identically with and without a local
// `pnpm build` (CI never builds the frontend).
const staticBadgeAsset = "/static/badges/commenter-50.svg"

// setupHeadContractRouter assembles the production router exactly like main
// does (RegisterByGin) with app.env pinned to production so the Vite dev
// proxy is not registered and BrowserCache long-public headers apply.
func setupHeadContractRouter(t *testing.T) *gin.Engine {
	t.Helper()
	setupMcpRouteTestDB(t) // mcpRoute needs the page_config table migrated
	previousEnv := preferences.Get("app.env", "production")
	preferences.Set("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)
	return router
}

// headContractClient returns a client with transparent gzip disabled so the
// assertions observe the server's real headers (the net/http default client
// would auto-negotiate gzip and strip Content-Encoding/Content-Length).
func headContractClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DisableCompression: true}}
}

func headContractRequest(t *testing.T, client *http.Client, method, url string, acceptEncoding string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new %s request: %v", method, err)
	}
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// TestHeadRouteRegistrationContract pins the route-tree shape behind the
// HEAD behavior: gin registers HEAD only where the code explicitly does
// (StaticFS static assets and Any("/mcp")). No GET-only route — health, SSR
// pages, JSON APIs, file downloads — may grow an implicit HEAD. If a future
// change explicitly registers HEAD on a dynamic route, this test fails and
// the routes-snapshot gate (TestRoutesSnapshot) must be re-run deliberately.
func TestHeadRouteRegistrationContract(t *testing.T) {
	router := setupHeadContractRouter(t)

	headPaths := map[string]bool{}
	getPaths := map[string]bool{}
	for _, route := range router.Routes() {
		switch route.Method {
		case http.MethodHead:
			headPaths[route.Path] = true
		case http.MethodGet:
			getPaths[route.Path] = true
		}
	}

	// The complete HEAD surface: two StaticFS mounts + Any("/mcp").
	wantHead := map[string]bool{
		"/assets/*filepath": true,
		"/static/*filepath": true,
		"/mcp":              true,
	}
	if len(headPaths) != len(wantHead) {
		t.Errorf("HEAD routes = %v, want exactly %v", headPaths, wantHead)
	}
	for path := range wantHead {
		if !headPaths[path] {
			t.Errorf("HEAD route %q missing (registered by StaticFS/Any)", path)
		}
	}
	for path := range headPaths {
		if !wantHead[path] {
			t.Errorf("unexpected HEAD route %q: adding HEAD to a dynamic route changes routes-snapshot.json and the API contract; extend the contract deliberately instead", path)
		}
	}

	// Representative dynamic GET routes must stay GET-only.
	getOnly := []string{
		"/health",
		"/",
		"/robots.txt",
		"/sitemap.xml",
		"/api/login-public-key",
		"/api/forum/get-site-statistics",
		"/file/img/*filename",
	}
	for _, path := range getOnly {
		if !getPaths[path] {
			t.Errorf("GET route %q not registered (test matrix out of date?)", path)
		}
		if headPaths[path] {
			t.Errorf("HEAD route %q is registered: dynamic routes must stay HEAD-less (issue #411 contract); if intentional, update routes-snapshot.json and the contract docs", path)
		}
	}

	// POST-only write endpoints have neither GET nor HEAD; HEAD can never
	// reach the write controller.
	writeOnly := []string{"/api/forum/topics/write", "/api/login", "/file/img-upload"}
	for _, path := range writeOnly {
		if getPaths[path] {
			t.Errorf("GET route %q is registered for a write-only endpoint (test matrix out of date?)", path)
		}
		if headPaths[path] {
			t.Errorf("HEAD route %q is registered for a write-only endpoint: HEAD must never execute a write controller", path)
		}
	}
}

// TestHeadContractStaticAssetsMatchGet asserts the supported HEAD surface:
// static assets served through gin StaticFS answer HEAD with the same status
// and headers as GET (including Content-Length and browser cache headers)
// and an empty body. The frontend build output under /assets only exists
// after `pnpm build`, so /assets is asserted conditionally: whatever GET
// answers (200 with dist present, 404 NoRoute without), HEAD must match.
func TestHeadContractStaticAssetsMatchGet(t *testing.T) {
	router := setupHeadContractRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	client := headContractClient()

	probe := func(path string) {
		t.Helper()
		getResp := headContractRequest(t, client, http.MethodGet, srv.URL+path, "")
		_ = getResp.Body.Close()
		headResp := headContractRequest(t, client, http.MethodHead, srv.URL+path, "")
		headBody, err := io.ReadAll(headResp.Body)
		if err != nil {
			t.Fatalf("read HEAD %s body: %v", path, err)
		}
		_ = headResp.Body.Close()

		if headResp.StatusCode != getResp.StatusCode {
			t.Errorf("HEAD %s status = %d, GET status = %d; want equal", path, headResp.StatusCode, getResp.StatusCode)
		}
		if len(headBody) != 0 {
			t.Errorf("HEAD %s body = %d bytes, want empty (net/http strips HEAD bodies)", path, len(headBody))
		}
		if getResp.StatusCode != http.StatusOK {
			// Missing frontend build output (/assets) or similar: status
			// agreement and empty body are the whole contract.
			return
		}
		if got := headResp.Header.Get("Content-Length"); got == "" || got != getResp.Header.Get("Content-Length") {
			t.Errorf("HEAD %s Content-Length = %q, GET Content-Length = %q; want equal and non-empty", path, got, getResp.Header.Get("Content-Length"))
		}
		for _, header := range []string{"Content-Type", "Cache-Control"} {
			if got := headResp.Header.Get(header); got != getResp.Header.Get(header) {
				t.Errorf("HEAD %s %s = %q, GET %s = %q; want equal", path, header, got, header, getResp.Header.Get(header))
			}
		}
		if getResp.Header.Get("Content-Type") == "" {
			t.Errorf("GET %s Content-Type is empty; asset contract expects a media type", path)
		}
	}

	// Committed static file: full assertions above.
	probe(staticBadgeAsset)
	// Frontend build output: conditional (GET 200 only when dist exists).
	probe("/assets/index.js")

	// gzip negotiation: GET compresses, HEAD stays body-less. The gzip
	// middleware only decides compression after body writes, so a HEAD
	// response carries no Content-Encoding; assert the stable part of the
	// contract (200 + empty body + media type), not the CE header.
	getGzip := headContractRequest(t, client, http.MethodGet, srv.URL+staticBadgeAsset, "gzip")
	gzipBody, err := io.ReadAll(getGzip.Body)
	if err != nil {
		t.Fatalf("read GET gzip body: %v", err)
	}
	_ = getGzip.Body.Close()
	if getGzip.StatusCode != http.StatusOK || getGzip.Header.Get("Content-Encoding") != "gzip" || len(gzipBody) == 0 {
		t.Errorf("GET %s with Accept-Encoding: gzip = status %d, Content-Encoding %q, body %d bytes; want 200/gzip/non-empty", staticBadgeAsset, getGzip.StatusCode, getGzip.Header.Get("Content-Encoding"), len(gzipBody))
	}
	headGzip := headContractRequest(t, client, http.MethodHead, srv.URL+staticBadgeAsset, "gzip")
	headGzipBody, err := io.ReadAll(headGzip.Body)
	if err != nil {
		t.Fatalf("read HEAD gzip body: %v", err)
	}
	_ = headGzip.Body.Close()
	if headGzip.StatusCode != http.StatusOK {
		t.Errorf("HEAD %s with Accept-Encoding: gzip status = %d, want 200", staticBadgeAsset, headGzip.StatusCode)
	}
	if len(headGzipBody) != 0 {
		t.Errorf("HEAD %s with Accept-Encoding: gzip body = %d bytes, want empty", staticBadgeAsset, len(headGzipBody))
	}
	if got := headGzip.Header.Get("Content-Type"); got != "image/svg+xml" {
		t.Errorf("HEAD %s with Accept-Encoding: gzip Content-Type = %q, want image/svg+xml", staticBadgeAsset, got)
	}
}

// TestHeadContractDynamicRoutesAreGetOnly asserts the unsupported side of
// the matrix: dynamic GET routes (health, robots, JSON APIs, SSR pages) do
// not answer HEAD. A HEAD probe to those paths gets the stable NoRoute 404 —
// never the GET handler, never a write-side controller.
func TestHeadContractDynamicRoutesAreGetOnly(t *testing.T) {
	router := setupHeadContractRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	client := headContractClient()

	// Routes where GET succeeds (200) but HEAD must be 404.
	for _, path := range []string{"/health", "/robots.txt", "/api/login-public-key", "/"} {
		getResp := headContractRequest(t, client, http.MethodGet, srv.URL+path, "")
		_ = getResp.Body.Close()
		headResp := headContractRequest(t, client, http.MethodHead, srv.URL+path, "")
		headBody, err := io.ReadAll(headResp.Body)
		if err != nil {
			t.Fatalf("read HEAD %s body: %v", path, err)
		}
		_ = headResp.Body.Close()
		if getResp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200 (test premise)", path, getResp.StatusCode)
		}
		if headResp.StatusCode != http.StatusNotFound {
			t.Errorf("HEAD %s status = %d, want 404: gin does not route HEAD to GET handlers; probes must use GET", path, headResp.StatusCode)
		}
		if len(headBody) != 0 {
			t.Errorf("HEAD %s body = %d bytes, want empty on the wire", path, len(headBody))
		}
		if headResp.Header.Get("Content-Type") == "" {
			t.Errorf("HEAD %s Content-Type is empty, want the NoRoute JSON content type", path)
		}
	}

	// POST-only write endpoints: HEAD must never reach the controller.
	// HEAD /api/forum/topics/write is unregistered, so gin answers 404
	// (same as the anonymous GET for that path); the write controller is
	// only reachable via its registered POST route.
	writePath := "/api/forum/topics/write"
	getWrite := headContractRequest(t, client, http.MethodGet, srv.URL+writePath, "")
	_ = getWrite.Body.Close()
	headWrite := headContractRequest(t, client, http.MethodHead, srv.URL+writePath, "")
	headWriteBody, err := io.ReadAll(headWrite.Body)
	if err != nil {
		t.Fatalf("read HEAD %s body: %v", writePath, err)
	}
	_ = headWrite.Body.Close()
	if getWrite.StatusCode != http.StatusNotFound || headWrite.StatusCode != http.StatusNotFound {
		t.Errorf("%s statuses: GET = %d, HEAD = %d; want 404/404 (no HEAD route registered, no side effect)", writePath, getWrite.StatusCode, headWrite.StatusCode)
	}
	if len(headWriteBody) != 0 {
		t.Errorf("HEAD %s body = %d bytes, want empty on the wire", writePath, len(headWriteBody))
	}

	// File download route (GET registered): a missing file answers 404 for
	// both methods — GET from the file controller, HEAD from NoRoute — so
	// status stays stable while the body never reaches the wire.
	missingFile := "/file/img/definitely-not-a-real-file-411.png"
	getFile := headContractRequest(t, client, http.MethodGet, srv.URL+missingFile, "")
	_ = getFile.Body.Close()
	headFile := headContractRequest(t, client, http.MethodHead, srv.URL+missingFile, "")
	headFileBody, err := io.ReadAll(headFile.Body)
	if err != nil {
		t.Fatalf("read HEAD %s body: %v", missingFile, err)
	}
	_ = headFile.Body.Close()
	if getFile.StatusCode != http.StatusNotFound || headFile.StatusCode != http.StatusNotFound {
		t.Errorf("%s statuses: GET = %d, HEAD = %d; want 404/404", missingFile, getFile.StatusCode, headFile.StatusCode)
	}
	if len(headFileBody) != 0 {
		t.Errorf("HEAD %s body = %d bytes, want empty on the wire", missingFile, len(headFileBody))
	}
}
