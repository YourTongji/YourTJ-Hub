package mcpservice

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Event ACK acknowledges processing only; these tools never enable content writes.
func registerAgentEvents(server *mcp.Server, svc *Service) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_events", Description: "拉取本 Agent 的保留期内事件。读取不自动确认；持久化收件后推进 nextCursor。", InputSchema: objectSchema(map[string]*jsonschema.Schema{"after": strPropMax("本 Agent 不透明游标；失效时明确重置", nil, 2048), "limit": intPropMax("每页事件数量", f(1), f(100), f(50))}, nil)}, recoverToolHandler("list_events", func(ctx context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		id, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		value, err := serviceResult(api.AgentEvents(component.BetterRequest[api.AgentEventsReq]{UserId: id, Context: svc.credentialContext(ctx, req), Params: api.AgentEventsReq{After: asString(in["after"]), Limit: asInt(in["limit"])}}))
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(value)
		return &mcp.CallToolResult{}, out, err
	}))
	mcp.AddTool(server, &mcp.Tool{Name: "get_event", Description: "读取本 Agent 的事件；内容撤销后返回无内容 tombstone。", InputSchema: objectSchema(map[string]*jsonschema.Schema{"eventId": strPropMax("本 Agent 的事件 ID", intPtr(1), 80)}, []string{"eventId"})}, recoverToolHandler("get_event", func(ctx context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		id, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		value, err := serviceResult(api.AgentEvent(component.BetterRequest[api.AgentEventReq]{UserId: id, Context: svc.credentialContext(ctx, req), Params: api.AgentEventReq{EventID: asString(in["eventId"])}}))
		if err != nil {
			return nil, nil, err
		}
		out, err := resultToMap(value)
		return &mcp.CallToolResult{}, out, err
	}))
	mcp.AddTool(server, &mcp.Tool{Name: "ack_events", Description: "完成处理或明确跳过后幂等确认事件；不改变站内通知或发帖权限。", InputSchema: objectSchema(map[string]*jsonschema.Schema{"eventIds": {Type: "array", MinItems: intPtr(1), MaxItems: intPtr(100), Items: strPropMax("本 Agent 的事件 ID", intPtr(1), 80)}}, []string{"eventIds"})}, recoverToolHandler("ack_events", func(ctx context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		id, err := svc.userID(req)
		if err != nil {
			return nil, nil, err
		}
		ids := []string{}
		if values, ok := in["eventIds"].([]any); ok {
			for _, value := range values {
				ids = append(ids, asString(value))
			}
		}
		value, err := serviceResult(api.AgentAckEvents(component.BetterRequest[api.AgentAckEventsReq]{UserId: id, Context: svc.credentialContext(ctx, req), Params: api.AgentAckEventsReq{EventIDs: ids}}))
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{}, map[string]any{"acknowledged": value}, nil
	}))
}
