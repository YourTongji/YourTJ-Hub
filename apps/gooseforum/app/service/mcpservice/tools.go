package mcpservice

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/http/controllers/forum"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/permission"
	"github.com/leancodebox/GooseForum/app/service/userservice"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// schema helpers for handwritten tool input schemas. Values mirror the
// OpenAPI contract (packages/api-contract/paths/agent.yaml) so an LLM sees
// exactly the same constraints as the REST API.

func f(v float64) *float64 { return &v }
func intPtr(v int) *int    { return &v }

// objectSchema builds an object schema with the given properties. By default
// unknown properties are rejected (additionalProperties: false), matching the
// contract's strict objects.
func objectSchema(props map[string]*jsonschema.Schema, required []string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:                 "object",
		Properties:           props,
		Required:             required,
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}, // false
	}
}

func intProp(desc string, min, def *float64) *jsonschema.Schema {
	s := &jsonschema.Schema{Type: "integer", Description: desc, Minimum: min}
	if def != nil {
		s.Default = numJSON(*def)
	}
	return s
}

func intPropMax(desc string, min, max, def *float64) *jsonschema.Schema {
	s := intProp(desc, min, def)
	s.Maximum = max
	return s
}

// maxJSONSafeInt is the largest integer float64 represents exactly (2^53-1).
// ID fields use it as their schema Maximum so out-of-range values are rejected
// by JSON Schema validation instead of silently overflowing during conversion.
var maxJSONSafeInt = f(float64(1<<53 - 1))

func strProp(desc string, minLen *int) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "string", Description: desc, MinLength: minLen}
}

func uintArrayProp(desc string, minItems, maxItems int) *jsonschema.Schema {
	one := f(1)
	return &jsonschema.Schema{
		Type:        "array",
		Description: desc,
		Items:       &jsonschema.Schema{Type: "integer", Minimum: one, Maximum: maxJSONSafeInt},
		MinItems:    intPtr(minItems),
		MaxItems:    intPtr(maxItems),
	}
}

func numJSON(v float64) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// registerMe exposes GET /api/v1/agent/me.
func registerMe(s *mcp.Server, svc *Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "me",
		Description: "返回当前 Agent（机器人）自身的资料：agentId、用户名、昵称、头像、token 前缀、启用状态、创建与更新时间。token 本身及哈希永不返回。",
		InputSchema: objectSchema(nil, nil),
	}, func(_ context.Context, req *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		agentID, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		resp := api.AgentMe(component.BetterRequest[component.Null]{UserId: agentID})
		v, err := serviceResult(resp)
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(v)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, out, nil
	})
}

// registerListTopics exposes GET /api/v1/agent/topics.
func registerListTopics(s *mcp.Server, svc *Service) {
	props := map[string]*jsonschema.Schema{
		"page":       intProp("页码，从 1 开始", f(1), f(1)),
		"pageSize":   intProp("每页条数，最小 10", f(10), f(10)),
		"sort":       strProp("排序：latest / hot / popular / new", nil),
		"categoryId": intPropMax("分类 ID（单值过滤）", f(1), maxJSONSafeInt, nil),
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_topics",
		Description: "分页列出已发布主题（仅 status=1 且 processStatus=0），支持排序与分类过滤。",
		InputSchema: objectSchema(props, nil),
	}, func(_ context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		agentID, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		params := api.AgentTopicListReq{
			Page:       asInt(in["page"]),
			PageSize:   asInt(in["pageSize"]),
			Sort:       asString(in["sort"]),
			CategoryId: asUint(in["categoryId"]),
		}
		resp := api.AgentTopicList(component.BetterRequest[api.AgentTopicListReq]{Params: params, UserId: agentID})
		v, err := serviceResult(resp)
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(v)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, out, nil
	})
}

// registerCreateTopic exposes POST /api/v1/agent/topics (write, opt-in).
func registerCreateTopic(s *mcp.Server, svc *Service) {
	props := map[string]*jsonschema.Schema{
		"title":      strProp("主题标题", intPtr(1)),
		"content":    strProp("主题正文（Markdown）", intPtr(1)),
		"categoryId": uintArrayProp("分类 ID 数组（1-3 个）", 1, 3),
	}
	required := []string{"title", "content", "categoryId"}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_topic",
		Description: "以当前 Agent 身份发布一个主题（创建即发布）。受 topic.write 限流约束。",
		InputSchema: objectSchema(props, required),
	}, func(_ context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		agentID, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		if err := svc.checkWriteRateLimit("topic.write", req, agentID); err != nil {
			return nil, nil, err
		}
		params := api.AgentWriteTopicReq{
			Title:      asString(in["title"]),
			Content:    asString(in["content"]),
			CategoryId: asUintSlice(in["categoryId"]),
		}
		resp := api.AgentWriteTopic(component.BetterRequest[api.AgentWriteTopicReq]{Params: params, UserId: agentID})
		v, err := serviceResult(resp)
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(map[string]any{"topicId": v})
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, out, nil
	})
}

// registerGetPosts exposes GET /api/v1/agent/topics/{topicId}/posts.
func registerGetPosts(s *mcp.Server, svc *Service) {
	props := map[string]*jsonschema.Schema{
		"topicId":      intPropMax("主题 ID", f(1), maxJSONSafeInt, nil),
		"anchorPostId": intPropMax("锚点帖子 ID", f(1), maxJSONSafeInt, nil),
		"anchorPostNo": intPropMax("锚点楼号", f(1), maxJSONSafeInt, nil),
		"beforePostNo": intPropMax("取该楼之前的帖子", f(1), maxJSONSafeInt, nil),
		"afterPostNo":  intPropMax("取该楼之后的帖子", f(1), maxJSONSafeInt, nil),
		"limit":        intPropMax("返回条数（1-50）", f(1), f(50), nil),
	}
	required := []string{"topicId"}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_posts",
		Description: "获取指定主题的帖子窗口（可按锚点帖子或楼号，或 before/after 翻页）。",
		InputSchema: objectSchema(props, required),
	}, func(_ context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		agentID, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		params := forum.PostWindowReq{
			TopicID:      asUint(in["topicId"]),
			AnchorPostID: asUint(in["anchorPostId"]),
			AnchorPostNo: asUint(in["anchorPostNo"]),
			BeforePostNo: asUint(in["beforePostNo"]),
			AfterPostNo:  asUint(in["afterPostNo"]),
			Limit:        asInt(in["limit"]),
		}
		resp := forum.PostWindow(component.BetterRequest[forum.PostWindowReq]{Params: params, UserId: agentID})
		v, err := serviceResult(resp)
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(v)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, out, nil
	})
}

// registerCreatePost exposes POST /api/v1/agent/topics/{topicId}/posts (write, opt-in).
func registerCreatePost(s *mcp.Server, svc *Service) {
	props := map[string]*jsonschema.Schema{
		"topicId":       intPropMax("主题 ID", f(1), maxJSONSafeInt, nil),
		"content":       strProp("回复正文（Markdown）", intPtr(1)),
		"replyToPostId": intPropMax("要回复的帖子 ID（可选）", f(0), maxJSONSafeInt, nil),
	}
	required := []string{"topicId", "content"}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_post",
		Description: "以当前 Agent 身份在指定主题下发帖（回复）。受 post.create 限流约束。",
		InputSchema: objectSchema(props, required),
	}, func(_ context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		agentID, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		if err := svc.checkWriteRateLimit("post.create", req, agentID); err != nil {
			return nil, nil, err
		}
		params := api.AgentCreatePostReq{
			TopicId:       asUint(in["topicId"]),
			Content:       asString(in["content"]),
			ReplyToPostId: asUint(in["replyToPostId"]),
		}
		resp := api.AgentCreatePost(component.BetterRequest[api.AgentCreatePostReq]{Params: params, UserId: agentID})
		v, err := serviceResult(resp)
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(v)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, out, nil
	})
}

// registerSearch exposes GET /api/v1/agent/search.
func registerSearch(s *mcp.Server, svc *Service) {
	props := map[string]*jsonschema.Schema{
		"q":     strProp("搜索关键词", nil),
		"scope": strProp("搜索范围：all / topics / users / categories", nil),
		"page":  intProp("页码，从 1 开始", f(1), f(1)),
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search",
		Description: "站内搜索主题、用户与分类（bot 用户不会出现在用户搜索结果中）。",
		InputSchema: objectSchema(props, nil),
	}, func(_ context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		params := forum.SearchJSONReq{
			Q:     asString(in["q"]),
			Scope: asString(in["scope"]),
			Page:  asInt(in["page"]),
		}
		resp := forum.SearchJSON(component.BetterRequest[forum.SearchJSONReq]{Params: params})
		v, err := serviceResult(resp)
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(v)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, out, nil
	})
}

// checkWriteRateLimit enforces the same topic.write / post.create quota as the
// REST middleware, keyed identically (IP + bot userId) so MCP writes share the
// exact quota with REST writes. The IP comes from the auth verifier's
// TokenInfo.Extra["clientIP"], mirroring gin's ClientIP resolution.
func (s *Service) checkWriteRateLimit(action string, req *mcp.CallToolRequest, userID uint64) error {
	cfg := hotdataserve.GetRateLimitConfigCache()
	if !cfg.Enabled {
		return nil
	}
	rule := findRateLimitRule(cfg.Actions, action)
	if rule == nil || (rule.LimitPerIp <= 0 && rule.LimitPerUser <= 0) {
		return nil
	}

	// Mirror middleware/rateLimit.go: administrators are exempt when the
	// rate-limit config requests it, so an admin-owned bot behaves the same
	// through MCP and through the REST /api/v1/agent writes.
	if cfg.SkipAdmin && userID != 0 {
		if roleID, ok := userservice.GetUserRoleId(userID); ok && permission.CheckRole(roleID, permission.Admin) {
			return nil
		}
	}

	ip := ""
	if req.Extra != nil && req.Extra.TokenInfo != nil && req.Extra.TokenInfo.Extra != nil {
		if v, ok := req.Extra.TokenInfo.Extra["clientIP"].(string); ok {
			ip = v
		}
	}

	window := time.Duration(rule.WindowSeconds) * time.Second
	store := ratelimit.Default()

	var retryAfter time.Duration
	limited := false
	if rule.LimitPerIp > 0 {
		key := action + ":ip:" + ip
		ok, retry, _ := store.Allow(key, rule.LimitPerIp, window)
		if !ok {
			limited, retryAfter = true, retry
		}
	}
	if !limited && userID != 0 && rule.LimitPerUser > 0 {
		key := action + ":user:" + strconv.FormatUint(userID, 10)
		ok, retry, _ := store.Allow(key, rule.LimitPerUser, window)
		if !ok {
			limited, retryAfter = true, retry
		}
	}

	if limited {
		seconds := int(math.Ceil(retryAfter.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		return fmt.Errorf("rate limited (%s): retry after %ds", action, seconds)
	}
	return nil
}

// findRateLimitRule returns the quota for an action, or nil if none is
// configured. Mirrors the lookup in middleware/rateLimit.go.
func findRateLimitRule(rules []pageConfig.RateLimitRule, action string) *pageConfig.RateLimitRule {
	for i := range rules {
		if rules[i].Action == action {
			return &rules[i]
		}
	}
	return nil
}

// asInt / asUint / asString / asUintSlice coerce MCP argument values (JSON
// numbers/strings) into the typed request fields the REST controllers expect.
func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}

func asUint(v any) uint64 {
	switch n := v.(type) {
	case float64:
		if n < 0 || n >= float64(^uint64(0)) {
			return 0
		}
		return uint64(n)
	case int:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case int64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case json.Number:
		u, err := n.Int64()
		if err != nil || u < 0 {
			return 0
		}
		return uint64(u)
	case string:
		i, err := strconv.ParseUint(n, 10, 64)
		if err != nil {
			return 0
		}
		return i
	}
	return 0
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case json.Number:
		return s.String()
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

func asUintSlice(v any) []uint64 {
	arr, ok := v.([]any)
	if !ok {
		if arrNum, ok2 := v.([]float64); ok2 {
			out := make([]uint64, 0, len(arrNum))
			for _, n := range arrNum {
				out = append(out, uint64(n))
			}
			return out
		}
		return nil
	}
	out := make([]uint64, 0, len(arr))
	for _, item := range arr {
		out = append(out, asUint(item))
	}
	return out
}
