package mcpservice

import (
	"encoding/json"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpServerName is the MCP Implementation name advertised at runtime and in
// the public server card.
const mcpServerName = "yourtj-hub"

// serverCard is the SEP-1649 discovery document for the existing streamable
// HTTP server. Authentication describes the static Bearer check already
// enforced by HTTPHandler; the card never carries a credential.
type serverCard struct {
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Name                      string   `json:"name"`
	Description               string   `json:"description"`
	Version                   string   `json:"version"`
	ProtocolVersion           string   `json:"protocolVersion"`
	SupportedProtocolVersions []string `json:"supportedProtocolVersions"`
	Transport                 struct {
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
		Required    bool   `json:"required"`
		Type        string `json:"type"`
		Scheme      string `json:"scheme"`
		Description string `json:"description"`
	} `json:"authentication"`
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
		Name:                      mcpServerName,
		Description:               "Forum agent tools served at /mcp with streamable HTTP. Authentication is the existing static Bearer scheme. The credential is not included in this card.",
		Version:                   version,
		ProtocolVersion:           protocolVersion,
		SupportedProtocolVersions: versions,
	}
	card.ServerInfo.Name = mcpServerName
	card.ServerInfo.Version = version
	card.Transport.Type = "streamable-http"
	card.Transport.Endpoint = "/mcp"
	card.Capabilities.Tools.ListChanged = true
	card.Authentication.Required = true
	card.Authentication.Type = "bearer"
	card.Authentication.Scheme = "Bearer"
	card.Authentication.Description = "Static Authorization Bearer credential already required by /mcp. This card does not include the credential."
	return json.Marshal(card)
}
