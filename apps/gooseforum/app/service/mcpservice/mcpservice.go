// Package mcpservice exposes the existing Agent Bot API (6 operations behind
// /api/v1/agent/*, Bearer agt_ token) as a standard MCP server.
//
// Design goals (issue #93):
//   - thin wrapper, never reimplements business logic: every tool handler
//     calls the same controller/service functions the REST Agent API uses;
//   - read operations available by default, write operations opt-in via the
//     writes setting in the admin panel (mirrors Discourse MCP's --allow_writes);
//   - authentication reuses agentservice.ResolveByToken so the agt_ token is
//     the single credential, with the same "all failures collapse to 401"
//     policy as the REST middleware;
//   - one process, one binary: the streamable HTTP handler is mounted at /mcp
//     on the main gin engine, and a mcp-stdio subcommand serves local CLIs;
//   - write tools reuse the existing topic.write / post.create rate limits,
//     keyed identically to the REST middleware so quotas are shared.
package mcpservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service builds and serves the MCP server. A single instance is used for
// both the streamable HTTP endpoint and the stdio subcommand.
type Service struct {
	// fixedUserID, when non-zero, short-circuits per-request authentication.
	// It is used by the stdio subcommand, which resolves the agt_ token once
	// at startup (a local CLI has a single identity for its lifetime).
	fixedUserID uint64

	// writesOverride, when non-nil, pins the write-tool availability for this
	// service instead of reading the admin-panel mcp.writes setting. This lets
	// the mcp-stdio subcommand decide write access independently of the public
	// /mcp endpoint.
	writesOverride *bool

	// sessionTimeout, when non-zero, overrides the idle-session cleanup timeout
	// applied by HTTPHandler. Production uses defaultMCPSessionTimeout (15m);
	// tests set it to a short value to observe session reclamation without
	// waiting.
	sessionTimeout time.Duration

	// buildMu guards the cached per-writes *mcp.Server instances. Building a
	// server registers tools and reflects their schemas, so the result is
	// cached and reused across sessions until the writes preference flips.
	buildMu   sync.Mutex
	readOnly  *mcp.Server // writes=false tool set
	readWrite *mcp.Server // writes=true tool set
}

// defaultMCPSessionTimeout is how long an idle streamable-HTTP session may
// hold no active requests before the SDK closes it. Sessions are created on
// the first POST (initialize) and only removed on a client DELETE or timeout;
// without a timeout, an idle client that never sends DELETE would pin its
// session + connection + read-loop goroutine in memory forever, and repeated
// initialize calls would accumulate unboundedly. 15 minutes of inactivity is
// far beyond any real MCP interaction and lets us reap those sessions.
const defaultMCPSessionTimeout = 15 * time.Minute

// defaultMCPPostWriteTimeout bounds a single /mcp POST (JSON-RPC) write. The
// SDK writes the response body only after the tool handler finishes, so the
// deadline must be generous; it exists to break clients that stop reading,
// which would otherwise pin the session forever (the SDK pauses the
// idle-session timer while a POST is in flight). See HTTPHandler.
const defaultMCPPostWriteTimeout = 60 * time.Second

// NewService returns a Service that authenticates every request via the
// agt_ bearer token (fixedUserID == 0).
func NewService() *Service {
	return &Service{}
}

// NewStdioService returns a Service bound to a single agent identity,
// resolved once at startup from an agt_ token. Used by the mcp-stdio
// subcommand for local CLI clients. writes, if non-nil, overrides the
// admin-panel mcp.writes setting for this service (independent of the public
// endpoint).
func NewStdioService(agentUserID uint64, writes ...bool) *Service {
	s := &Service{fixedUserID: agentUserID}
	if len(writes) > 0 {
		w := writes[0]
		s.writesOverride = &w
	}
	return s
}

// writesEnabled returns whether write tools should be registered for this
// service, honoring the per-service override if set. Without an override the
// admin-panel MCP write setting is used, so a panel toggle takes effect
// without restarting the process.
func (s *Service) writesEnabled() bool {
	if s.writesOverride != nil {
		return *s.writesOverride
	}
	return hotdataserve.GetMCPSettingsConfigCache().Writes
}

// userID resolves the authenticated agent's user id for a tool call.
// The stdio path uses the fixed identity; the streamable HTTP path reads the
// TokenInfo the auth middleware attached to the request.
func (s *Service) userID(req *mcp.CallToolRequest) (uint64, error) {
	if s.fixedUserID != 0 {
		return s.fixedUserID, nil
	}
	if req.Extra == nil || req.Extra.TokenInfo == nil || req.Extra.TokenInfo.UserID == "" {
		return 0, errors.New("mcpservice: unauthenticated request")
	}
	id, err := strconv.ParseUint(req.Extra.TokenInfo.UserID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("mcpservice: invalid user id %q: %w", req.Extra.TokenInfo.UserID, err)
	}
	return id, nil
}

// ClientIPContextKey is the context key the gin mounting layer uses to pass the
// trusted-proxy-resolved client IP into the MCP auth verifier. The REST stack
// resolves the client IP via gin's ClientIP() (which honors server.trusted_proxies
// before trusting X-Forwarded-For); MCP must reuse that same resolution so the
// IP dimension of the write rate limits cannot be spoofed by a raw
// X-Forwarded-For header on a direct connection.
type ClientIPContextKey struct{}

// verifier authenticates an agt_ token against the Agent store. It mirrors
// middleware.AgentAuth: only the exact "Bearer <token>" form is accepted and
// every failure resolves to the same auth.ErrInvalidToken so the SDK answers
// with a single 401. Agent tokens never expire, so a far-future expiration is
// attached to satisfy the SDK's required-expiration check.
func (s *Service) verifier(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
	agent, _, err := agentservice.ResolveByToken(token)
	if err != nil || agent == nil {
		return nil, auth.ErrInvalidToken
	}
	ip := ""
	if v, ok := ctx.Value(ClientIPContextKey{}).(string); ok {
		ip = v
	} else {
		// Direct handler use (tests): no gin TrustedProxies context, fall back
		// to the peer address without ever trusting X-Forwarded-For.
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = req.RemoteAddr
		}
	}
	return &auth.TokenInfo{
		UserID:     strconv.FormatUint(agent.UserId, 10),
		Expiration: time.Now().Add(100 * 365 * 24 * time.Hour),
		Extra: map[string]any{
			"clientIP": ip,
		},
	}, nil
}

// getServer returns the *mcp.Server appropriate for the current writes
// setting. Read tools are always registered; write tools only when the
// admin-panel MCP write setting is enabled. The setting is re-read on every
// new session (with a 5s hotdataserve cache) so a panel change takes effect
// without restarting the process. Built servers are cached per writes value
// and reused across sessions.
func (s *Service) getServer(*http.Request) *mcp.Server {
	writes := s.writesEnabled()
	s.buildMu.Lock()
	defer s.buildMu.Unlock()
	if writes {
		if s.readWrite == nil {
			s.readWrite = s.buildServer(true)
		}
		return s.readWrite
	}
	if s.readOnly == nil {
		s.readOnly = s.buildServer(false)
	}
	return s.readOnly
}

// buildServer constructs an MCP server with the curated 6-tool set. Tool
// definitions are handwritten (see tools.go) rather than generated from
// OpenAPI: the set is deliberately minimal so an LLM sees only what it can
// use, and no hallucinated endpoints leak into its context.
//
// Every tool handler is wrapped by recoverToolHandler: the go-sdk runs tool
// handlers on its own goroutines (internal/jsonrpc2 conn.handleAsync spawns
// `go func(){ handler.Handle(...) }()` and the SDK never recovers), so a panic
// inside a handler chain (a reused REST controller, resultToMap's json.Marshal,
// a future nil dereference) would crash the whole process. The gin
// middleware.Recovery() only protects HTTP request goroutines and cannot see
// these. Converting the panic to a tool error keeps the process alive and
// surfaces it to the MCP client instead.
func (s *Service) buildServer(writes bool) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "yourtj-hub",
		Version: buildinfo.Get().Version,
	}, nil)

	registerMe(server, s)
	registerListTopics(server, s)
	registerGetPosts(server, s)
	registerSearch(server, s)
	if writes {
		registerCreateTopic(server, s)
		registerCreatePost(server, s)
	}
	return server
}

// HTTPHandler returns the streamable HTTP MCP handler, wrapped with bearer
// authentication. Mount at /mcp with gin.WrapH.
//
// The surrounding http.Server (see console/serve.go) sets WriteTimeout to 10s,
// which would kill the SSE/streamable long-lived connections MCP clients rely
// on. Per request we therefore adjust the write deadline; this only affects
// /mcp and leaves the 10s slow-writer protection in place for every other
// endpoint.
//
//   - GET (SSE stream): the connection lives as long as the session, so the
//     write deadline is lifted to "no timeout". The stream is still bounded by
//     the session timeout (defaultMCPSessionTimeout, 15m), which reaps idle
//     sessions and closes their streams.
//   - POST (JSON-RPC): the SDK writes the response body (text/event-stream by
//     default) only after the tool handler has finished, so a slow handler
//     (e.g. search over meilisearch, or a write tool hitting the DB) must not
//     be cut off by a 10s deadline. A finite generous deadline
//     (defaultMCPPostWriteTimeout, 60s) keeps that headroom while still
//     bounding the request: the SDK pauses the idle-session timer for the
//     duration of an in-flight POST, so without any write deadline a client
//     that never reads its response could pin the connection, the session, and
//     their goroutines forever. 60s is far beyond any real handler time; once
//     the write fails the request completes and the idle timer resumes, so the
//     session can be reaped normally.
func (s *Service) HTTPHandler() http.Handler {
	timeout := defaultMCPSessionTimeout
	if s.sessionTimeout > 0 {
		timeout = s.sessionTimeout
	}
	handler := mcp.NewStreamableHTTPHandler(s.getServer, &mcp.StreamableHTTPOptions{
		Logger:         slog.Default(),
		SessionTimeout: timeout,
		// The SDK's DNS-rebinding protection rejects requests arriving via a
		// loopback address whose Host header is not loopback (403 "invalid
		// Host header"). The documented production topology is exactly that:
		// openresty forwards forum.yourtj.de → 127.0.0.1:5234 with a public
		// Host. The protection targets unauthenticated local services; /mcp is
		// bearer-authenticated and the gin layer already resolves the client IP
		// through the same trusted-proxies policy as the REST stack, so we
		// disable it explicitly.
		DisableLocalhostProtection: true,
	})
	authed := auth.RequireBearerToken(s.verifier, nil)(handler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rc := http.NewResponseController(w); rc != nil {
			deadline := time.Time{} // GET SSE streams are long-lived
			if r.Method != http.MethodGet {
				deadline = time.Now().Add(defaultMCPPostWriteTimeout)
			}
			_ = rc.SetWriteDeadline(deadline)
		}
		authed.ServeHTTP(w, r)
	})
}

// RunStdio serves the MCP protocol over stdin/stdout for local CLI clients.
func (s *Service) RunStdio(ctx context.Context) error {
	return s.getServer(nil).Run(ctx, &mcp.StdioTransport{})
}

// serviceResult unwraps the shared component.Response envelope. A business
// failure (REST HTTP 200 + code:1) is surfaced as an error carrying the stable
// messageCode, matching the semantics MCP clients expect from a tool error.
func serviceResult(resp component.Response) (any, error) {
	if resp.Data.Code != component.SUCCESS {
		return nil, fmt.Errorf("business error %s %v", resp.Data.MessageCode, resp.Data.Params)
	}
	return resp.Data.Result, nil
}

// resultToMap converts an arbitrary JSON-serializable value to a map so MCP
// structured output is a clean JSON object. Numbers are decoded with UseNumber
// so uint64 IDs above 2^53 survive the round trip without losing precision
// (float64 would silently corrupt large IDs such as topic/post/user ids).
func resultToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}
