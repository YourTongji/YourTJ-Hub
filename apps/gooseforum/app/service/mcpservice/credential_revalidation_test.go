package mcpservice

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStdioCredentialRotationInvalidatesOpenSession(t *testing.T) {
	setupMCPServiceTestDB(t)
	id, _ := createMCPServiceAgent(t, "stdio_rotation")
	svc := NewStdioService(id)
	if _, err := svc.userID(&mcp.CallToolRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := agentservice.RotateToken(id); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.userID(&mcp.CallToolRequest{}); err == nil {
		t.Fatal("open stdio session continued authorizing after token rotation")
	}
}

func TestHTTPToolRevalidatesBoundCredential(t *testing.T) {
	setupMCPServiceTestDB(t)
	id, token := createMCPServiceAgent(t, "http_rotation")
	svc := NewService()
	info, err := svc.verifier(context.Background(), token, httptest.NewRequest("POST", "/mcp", nil))
	if err != nil {
		t.Fatal(err)
	}
	req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{TokenInfo: info}}
	if _, err := svc.userID(req); err != nil {
		t.Fatal(err)
	}
	if _, err := agentservice.RotateToken(id); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.userID(req); err == nil {
		t.Fatal("cached HTTP tool identity survived token rotation")
	}
}
