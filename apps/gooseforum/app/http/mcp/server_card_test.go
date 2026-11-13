package mcpservice

import (
	"encoding/json"
	"testing"
)

// TestServerCardMatchesLiveServer guards the SEP-1649 discovery card against
// drift: serverInfo, capabilities and protocolVersion must equal what the
// real server reports at initialize, for both the read-only and write tool
// sets. If buildServer gains prompts/resources or custom capabilities, this
// test fails until the card is updated alongside it.
func TestServerCardMatchesLiveServer(t *testing.T) {
	body, err := ServerCardJSON()
	if err != nil {
		t.Fatalf("ServerCardJSON: %v", err)
	}
	var card struct {
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		ProtocolVersion string          `json:"protocolVersion"`
		Capabilities    json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(body, &card); err != nil {
		t.Fatalf("unmarshal card: %v", err)
	}
	var cardCaps map[string]any
	if err := json.Unmarshal(card.Capabilities, &cardCaps); err != nil {
		t.Fatalf("unmarshal card capabilities: %v", err)
	}
	cardCapsJSON, err := json.Marshal(cardCaps)
	if err != nil {
		t.Fatalf("normalize card capabilities: %v", err)
	}

	for _, writes := range []bool{false, true} {
		cs := connectMCPServer(t, writes)
		init := cs.InitializeResult()
		if init == nil {
			t.Fatalf("writes=%v: no initialize result", writes)
		}
		if init.ServerInfo == nil || init.ServerInfo.Name != card.ServerInfo.Name || init.ServerInfo.Version != card.ServerInfo.Version {
			t.Fatalf("writes=%v: live serverInfo = %#v, card advertises name=%q version=%q",
				writes, init.ServerInfo, card.ServerInfo.Name, card.ServerInfo.Version)
		}
		if init.ProtocolVersion != card.ProtocolVersion {
			t.Fatalf("writes=%v: negotiated protocolVersion = %q, card advertises %q",
				writes, init.ProtocolVersion, card.ProtocolVersion)
		}
		liveCaps, err := json.Marshal(init.Capabilities)
		if err != nil {
			t.Fatalf("writes=%v: marshal live capabilities: %v", writes, err)
		}
		var live map[string]any
		if err := json.Unmarshal(liveCaps, &live); err != nil {
			t.Fatalf("writes=%v: unmarshal live capabilities: %v", writes, err)
		}
		// json.Marshal sorts map keys, so both sides normalize identically.
		liveCapsJSON, err := json.Marshal(live)
		if err != nil {
			t.Fatalf("writes=%v: normalize live capabilities: %v", writes, err)
		}
		if string(liveCapsJSON) != string(cardCapsJSON) {
			t.Fatalf("writes=%v: live capabilities %s differ from card %s", writes, liveCapsJSON, cardCapsJSON)
		}
	}
}
