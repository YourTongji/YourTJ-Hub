// Package agentwebhookservice owns bounded, persistent Agent webhook delivery.
package agentwebhookservice

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"gorm.io/gorm"
)

var (
	ErrSecretRequired = errors.New("agent webhook secret required")
	ErrConfigConflict = agents.ErrConfigConflict
	ErrInvalidConfig  = errors.New("agent webhook invalid config")
	ErrUnavailable    = errors.New("agent webhook unavailable")
	ErrNotReplayable  = errors.New("agent webhook event not replayable")
	ErrLeaseLost      = errors.New("agent webhook lease lost")
)

type ConfigParams struct {
	EventsEnabled   bool     `json:"eventsEnabled"`
	EventTypes      []string `json:"eventTypes"`
	WebhookEnabled  bool     `json:"webhookEnabled"`
	WebhookEndpoint string   `json:"webhookEndpoint"`
}
type SecretResult struct {
	Secret        string `json:"secret"`
	SecretVersion uint64 `json:"secretVersion"`
	ConfigVersion uint64 `json:"configVersion"`
}

func Configure(agentID, expectedVersion uint64, p ConfigParams) (*agents.Entity, error) {
	p.WebhookEndpoint = strings.TrimSpace(p.WebhookEndpoint)
	if len(p.WebhookEndpoint) > 512 || (p.WebhookEndpoint != "" && sender.CheckWebhookTarget(context.Background(), p.WebhookEndpoint) != nil) {
		return nil, ErrInvalidConfig
	}
	seen := map[string]bool{}
	types := make([]string, 0, 3)
	for _, kind := range p.EventTypes {
		switch kind {
		case "agent.mentioned", "agent.post_replied", "agent.topic_commented",
			"forum.topic_created", "forum.post_created":
			if !seen[kind] {
				seen[kind] = true
				types = append(types, kind)
			}
		default:
			return nil, ErrInvalidConfig
		}
	}
	if p.EventsEnabled && len(types) == 0 {
		return nil, ErrInvalidConfig
	}
	sort.Strings(types)
	encoded, err := json.Marshal(types)
	if err != nil {
		return nil, err
	}
	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		row, err := agents.GetTx(tx, agentID, true)
		if err != nil {
			return err
		}
		if row.ConfigVersion != expectedVersion {
			return ErrConfigConflict
		}
		if (p.EventsEnabled || p.WebhookEnabled) && row.Enabled != agents.StatusEnabled {
			return ErrUnavailable
		}
		if p.WebhookEnabled && row.SecretCiphertext == "" {
			return ErrSecretRequired
		}
		if p.WebhookEnabled && (!p.EventsEnabled || p.WebhookEndpoint == "") {
			return ErrInvalidConfig
		}
		changes := map[string]any{"events_enabled": p.EventsEnabled, "event_types": string(encoded), "webhook_enabled": p.WebhookEnabled, "webhook_endpoint": p.WebhookEndpoint, "webhook_paused_reason": ""}
		if row.EventsEnabled != p.EventsEnabled || row.EventTypes != string(encoded) {
			changes["subscription_generation"] = row.SubscriptionGeneration + 1
			if p.EventsEnabled {
				now := time.Now()
				changes["events_enabled_at"] = &now
			} else {
				changes["events_enabled_at"] = nil
			}
		}
		generation := row.EndpointGeneration
		if row.WebhookEndpoint != p.WebhookEndpoint || row.WebhookEnabled != p.WebhookEnabled {
			generation++
			changes["endpoint_generation"] = generation
		}
		if err := agents.UpdateConfigTx(tx, agentID, expectedVersion, changes); err != nil {
			return err
		}
		if _, changed := changes["subscription_generation"]; changed {
			if err := agenteventservice.WithdrawAgentTx(tx, agentID); err != nil {
				return err
			}
		}
		return agentWebhook.CancelGenerationTx(tx, agentinstance.Current().ID, agentID, generation, "configuration_changed")
	})
	if err != nil {
		return nil, err
	}
	return agents.GetByUserID(agentID), nil
}
func RotateSecret(agentID, expectedVersion uint64, emergency bool) (SecretResult, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return SecretResult{}, err
	}
	secret := "whsec_" + base64.StdEncoding.EncodeToString(random)
	encrypted, err := securestore.EncryptPurpose(secret, securestore.AgentWebhookSecretPurpose)
	if err != nil {
		return SecretResult{}, err
	}
	result := SecretResult{Secret: secret, ConfigVersion: expectedVersion + 1}
	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		row, err := agents.GetTx(tx, agentID, true)
		if err != nil {
			return err
		}
		if row.ConfigVersion != expectedVersion {
			return ErrConfigConflict
		}
		result.SecretVersion = row.SecretVersion + 1
		changes := map[string]any{"secret_ciphertext": encrypted, "secret_version": result.SecretVersion, "previous_secret_ciphertext": "", "previous_secret_expires_at": nil}
		if !emergency && row.SecretCiphertext != "" {
			until := time.Now().Add(24 * time.Hour)
			changes["previous_secret_ciphertext"] = row.SecretCiphertext
			changes["previous_secret_expires_at"] = &until
		}
		return agents.UpdateConfigTx(tx, agentID, expectedVersion, changes)
	})
	return result, err
}

type DeliveryPage struct {
	Items []agentWebhook.Delivery `json:"items"`
	Total int64                   `json:"total"`
	Page  int                     `json:"page"`
	Size  int                     `json:"size"`
}

func ListDeliveries(agentID uint64, page, size int) (DeliveryPage, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	items, total, err := agentWebhook.ListTx(db.Connect(), agentinstance.Current().ID, agentID, (page-1)*size, size)
	return DeliveryPage{Items: items, Total: total, Page: page, Size: size}, err
}
func GetAttempts(agentID, deliveryID uint64) ([]agentWebhook.Attempt, error) {
	state := agentinstance.Current()
	row, err := agentWebhook.GetTx(db.Connect(), state.ID, deliveryID, false)
	if err != nil || row.AgentID != agentID {
		return nil, ErrNotReplayable
	}
	return agentWebhook.AttemptsTx(db.Connect(), state.ID, deliveryID)
}

type AgentSummary struct {
	LatestAcceptedAt *time.Time `json:"latestAcceptedAt"`
	PendingCount     int64      `json:"pendingCount"`
	PauseReason      string     `json:"pauseReason"`
	SecretConfigured bool       `json:"secretConfigured"`
}

func Summary(agentID uint64) (AgentSummary, error) {
	row := agents.GetByUserID(agentID)
	if row == nil {
		return AgentSummary{}, gorm.ErrRecordNotFound
	}
	n, err := agentWebhook.PendingCountTx(db.Connect(), agentinstance.Current().ID, agentID)
	return AgentSummary{LatestAcceptedAt: row.LastWebhookAcceptedAt, PendingCount: n, PauseReason: row.WebhookPausedReason, SecretConfigured: row.SecretCiphertext != ""}, err
}

// Cleanup expires payloads without deleting the event-stream replay floor.
func Cleanup() error {
	state := agentinstance.Current()
	if state.ID == "" {
		return nil
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := agentWebhook.ExpireTx(tx, state.ID, now); err != nil {
			return err
		}
		if err := agentWebhook.PurgeAttemptsTx(tx, state.ID, now.Add(-30*24*time.Hour), 500); err != nil {
			return err
		}
		if err := agentWebhook.PurgeMetadataTx(tx, state.ID, now.Add(-30*24*time.Hour), 500); err != nil {
			return err
		}
		return agentWrites.PurgeExpiredTx(tx, state.ID, now, 500)
	})
}
