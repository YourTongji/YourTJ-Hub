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
//   - /static/*filepath: HEAD == GET status and headers — including the
//     Cache-Control long-public header, which BrowserCache attaches to this
//     mount only (assertRouter registers it after the /assets StaticFS) —
//     with an empty body.
//   - /assets/*filepath (frontend build output): HEAD == GET status and
//     headers except Cache-Control, with an empty body; the mount carries no
//     cache header to promise. TestHeadContractAssetsMatchGet probes a file
//     the Vite manifest actually emits and skips when the build output is
//     missing.
//   - everything else (dynamic GET routes): HEAD 404 even when GET 200 —
//     load balancer and monitor probes must use GET. HEAD never executes a
//     write-side controller because no write route registers HEAD; that
//     "no business side effects" guarantee does not cover middleware
//     accounting — Any("/mcp") registers HEAD, and HEAD /mcp consumes the
//     mcp.auth per-IP rate-limit quota like any other /mcp request.
//
// See docs/architecture/contracts-and-data.md (HTTP method contract) for the
// authoritative statement of this matrix.

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/resource"
	"github.com/gin-gonic/gin"
)

// staticBadgeAsset is a committed static file (not frontend build output), so
// HEAD contract assertions run identically with and without a local
// `pnpm build` (CI never builds the frontend).
const staticBadgeAsset = "/static/badges/commenter-50.svg"

// setupHeadContractRouter assembles the production router exactly like main
// does (RegisterByGin) with app.env pinned to production so the Vite dev
// proxy is not registered and BrowserCache long-public headers apply. The
// gzip middleware is installed at router registration time (assertRouter
// reads server.gzip once), so server.gzip is pinned on here and restored on
// cleanup: the gzip negotiation assertions below must hold regardless of the
// developer's local [server].gzip setting.
func setupHeadContractRouter(t *testing.T) *gin.Engine {
	t.Helper()
	setupMcpRouteTestDB(t) // mcpRoute needs the page_config table migrated
	previousEnv := preferences.Get("app.env", "production")
	preferences.Set("app.env", "production")
	previousGzip := preferences.GetBool("server.gzip", true)
	preferences.Set("server.gzip", true)
	t.Cleanup(func() {
		preferences.Set("app.env", previousEnv)
		preferences.Set("server.gzip", previousGzip)
	})
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

// TestHeadContractStaticAssetsMatchGet asserts the /static mount contract:
// files served through gin StaticFS answer HEAD with the same status and
// headers as GET — Content-Length, Content-Type, and the BrowserCache
// long-public Cache-Control header that only this mount carries (assertRouter
// attaches the middleware after the /assets registration) — plus an empty
// body on the wire. server.gzip is pinned on by setupHeadContractRouter so
// the gzip negotiation assertions hold under any local [server].gzip config.
func TestHeadContractStaticAssetsMatchGet(t *testing.T) {
	router := setupHeadContractRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	client := headContractClient()

	// The badge is a committed static file, so GET must answer 200 wherever
	// the tests run (CI never builds the frontend but embeds /static).
	getResp := headContractRequest(t, client, http.MethodGet, srv.URL+staticBadgeAsset, "")
	_ = getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200 (committed static file)", staticBadgeAsset, getResp.StatusCode)
	}
	headResp := headContractRequest(t, client, http.MethodHead, srv.URL+staticBadgeAsset, "")
	headBody, err := io.ReadAll(headResp.Body)
	if err != nil {
		t.Fatalf("read HEAD %s body: %v", staticBadgeAsset, err)
	}
	_ = headResp.Body.Close()

	if headResp.StatusCode != getResp.StatusCode {
		t.Errorf("HEAD %s status = %d, GET status = %d; want equal", staticBadgeAsset, headResp.StatusCode, getResp.StatusCode)
	}
	if len(headBody) != 0 {
		t.Errorf("HEAD %s body = %d bytes, want empty (net/http strips HEAD bodies)", staticBadgeAsset, len(headBody))
	}
	if got := headResp.Header.Get("Content-Length"); got == "" || got != getResp.Header.Get("Content-Length") {
		t.Errorf("HEAD %s Content-Length = %q, GET Content-Length = %q; want equal and non-empty", staticBadgeAsset, got, getResp.Header.Get("Content-Length"))
	}
	for _, header := range []string{"Content-Type", "Cache-Control"} {
		if got := headResp.Header.Get(header); got == "" {
			t.Errorf("%s %s is empty on GET; want the /static mount to set it", staticBadgeAsset, header)
		} else if got != getResp.Header.Get(header) {
			t.Errorf("HEAD %s %s = %q, GET %s = %q; want equal", staticBadgeAsset, header, got, header, getResp.Header.Get(header))
		}
	}

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

// viteManifestEntryFile reads static/dist/.vite/manifest.json (the Vite
// multi-entry build output) and returns the emitted file of the named entry,
// mirroring assets_test.go. The build output is not committed and CI never
// builds the frontend, so the calling test is skipped when the manifest is
// missing instead of silently passing or failing.
func viteManifestEntryFile(t *testing.T, entry string) string {
	t.Helper()
	content, err := resource.GetTemplateFS().Open("static/dist/.vite/manifest.json")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skip("vite manifest is missing; run pnpm -C resource build to enable /assets HEAD contract checks")
		}
		t.Fatalf("open manifest: %v", err)
	}
	defer func() { _ = content.Close() }()

	var manifest map[string]struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(content).Decode(&manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	item, ok := manifest[entry]
	if !ok || item.File == "" {
		t.Fatalf("manifest entry %q missing or empty", entry)
	}
	return item.File
}

// TestHeadContractAssetsMatchGet asserts the /assets mount against a file the
// Vite build actually emits (the site entry from the manifest), so the
// successful-response path is exercised whenever the build output exists.
// /assets carries no BrowserCache Cache-Control header — the contract only
// promises HEAD == GET status and headers excluding Cache-Control plus an
// empty body. Without `pnpm build` the manifest is absent and the test skips
// (dist is never committed and CI never builds the frontend); a skip is not a
// pass — the 200 path simply stays unasserted until the frontend is built.
func TestHeadContractAssetsMatchGet(t *testing.T) {
	entryFile := viteManifestEntryFile(t, "src/site/main.ts")
	router := setupHeadContractRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	client := headContractClient()

	assetPath := "/assets/" + strings.TrimPrefix(entryFile, "/")
	getResp := headContractRequest(t, client, http.MethodGet, srv.URL+assetPath, "")
	_ = getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200 (manifest entry must resolve under /assets)", assetPath, getResp.StatusCode)
	}
	headResp := headContractRequest(t, client, http.MethodHead, srv.URL+assetPath, "")
	headBody, err := io.ReadAll(headResp.Body)
	if err != nil {
		t.Fatalf("read HEAD %s body: %v", assetPath, err)
	}
	_ = headResp.Body.Close()

	if headResp.StatusCode != getResp.StatusCode {
		t.Errorf("HEAD %s status = %d, GET status = %d; want equal", assetPath, headResp.StatusCode, getResp.StatusCode)
	}
	if len(headBody) != 0 {
		t.Errorf("HEAD %s body = %d bytes, want empty (net/http strips HEAD bodies)", assetPath, len(headBody))
	}
	if got := headResp.Header.Get("Content-Length"); got == "" || got != getResp.Header.Get("Content-Length") {
		t.Errorf("HEAD %s Content-Length = %q, GET Content-Length = %q; want equal and non-empty", assetPath, got, getResp.Header.Get("Content-Length"))
	}
	if got := headResp.Header.Get("Content-Type"); got == "" || got != getResp.Header.Get("Content-Type") {
		t.Errorf("HEAD %s Content-Type = %q, GET Content-Type = %q; want equal and non-empty", assetPath, got, getResp.Header.Get("Content-Type"))
	}
	// Cache-Control is deliberately not compared: BrowserCache only applies
	// to the /static mount, and /assets must not promise a cache header.
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
