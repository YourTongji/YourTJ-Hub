// Package agentinstance keeps Agent isolation state outside business snapshots.
package agentinstance

import (
	"encoding/json"
	"os"
	"strings"
)

type State struct {
	ID              string `json:"instanceId"`
	Epoch           string `json:"streamEpoch"`
	APIEnabled      bool   `json:"apiEnabled"`
	ProducerEnabled bool   `json:"producerEnabled"`
	WebhookEnabled  bool   `json:"webhookEnabled"`
}

func enabled(name string, fallback bool) bool {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	return strings.EqualFold(raw, "true") || raw == "1"
}
func Current() State {
	s := State{APIEnabled: true}
	path := os.Getenv("YOURTJ_AGENT_STATE_FILE")
	if path == "" {
		path = "storage/agent-state/state.json"
	}
	if _, statErr := os.Stat(path); statErr == nil || os.Getenv("YOURTJ_AGENT_STATE_FILE") != "" {
		raw, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(raw, &s) != nil {
			return State{}
		}
	}
	if raw, ok := os.LookupEnv("YOURTJ_AGENT_INSTANCE_ID"); ok {
		s.ID = strings.TrimSpace(raw)
	}
	if raw, ok := os.LookupEnv("YOURTJ_AGENT_STREAM_EPOCH"); ok {
		s.Epoch = strings.TrimSpace(raw)
	}
	s.APIEnabled = enabled("YOURTJ_AGENT_API_ENABLED", s.APIEnabled)
	s.ProducerEnabled = enabled("YOURTJ_AGENT_PRODUCER_ENABLED", s.ProducerEnabled)
	s.WebhookEnabled = enabled("YOURTJ_AGENT_WEBHOOK_ENABLED", s.WebhookEnabled)
	if s.ID == "" || s.Epoch == "" {
		s.ProducerEnabled = false
		s.WebhookEnabled = false
	}
	return s
}
