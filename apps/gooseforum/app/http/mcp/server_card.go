package mcpservice

import (
	"encoding/json"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpServerName is the MCP Implementation name advertised at runtime and in
// the public server card.
const mcpServerName = "yourtj-hub"

const (
	serverCardSchema  = "https://static.modelcontextprotocol.io/schemas/mcp-server-card/v1.json"
	serverCardVersion = "1.0"
)

// serverCard is the SEP-1649 discovery document for the existing streamable
// HTTP server. version is the card schema version. serverInfo.version is the
// application build. Authentication lists the static Bearer scheme already
// enforced by HTTPHandler; the card never carries a credential.
type serverCard struct {
	Schema     string `json:"$schema"`
	Version    string `json:"version"`
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Description     string `json:"description"`
	ProtocolVersion string `json:"protocolVersion"`
	Transport       struct {
		Type     string `json:"type"`
		Endpoint string `json:"endpoint"`
	} `json:"transport"`
	Capabilities struct {
		Logging struct{} `json:"logging"`
		Tools   struct {
			ListChanged bool `json:"listChanged"`
		} `json:"tools"`
	} `json:"capabilities"`
	Authentication struct {
		Required bool     `json:"required"`
		Schemes  []string `json:"schemes"`
	} `json:"authentication"`
	Tools []string `json:"tools"`
}

// ServerCardJSON returns the public MCP server card. Transport matches
// NewStreamableHTTPHandler mounted at /mcp. Capabilities match the SDK
// defaults used by buildServer: logging, plus tools with listChanged once
// tools are registered. Protocol versions come from the same SDK.
func ServerCardJSON() ([]byte, error) {
	version := buildinfo.Get().Version
	versions := mcp.SupportedProtocolVersions()
	protocolVersion := ""
	if len(versions) > 0 {
		protocolVersion = versions[0]
	}
	card := serverCard{
		Schema:          serverCardSchema,
		Version:         serverCardVersion,
		Description:     "Forum agent tools served at /mcp with streamable HTTP. Authentication is the existing static Bearer scheme. The credential is not included in this card.",
		ProtocolVersion: protocolVersion,
		Tools:           []string{"dynamic"},
	}
	card.ServerInfo.Name = mcpServerName
	card.ServerInfo.Version = version
	card.Transport.Type = "streamable-http"
	card.Transport.Endpoint = "/mcp"
	card.Capabilities.Tools.ListChanged = true
	card.Authentication.Required = true
	card.Authentication.Schemes = []string{"bearer"}
	return json.Marshal(card)
}
