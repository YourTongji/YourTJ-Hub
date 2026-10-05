package agentwebhookservice

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const TaskType = "agent-webhook.deliver"
const permitTTL = 15 * time.Second
const MaxPendingPerAgent = 1000

var sender = safefetch.New(safefetch.Config{ConnectTimeout: 2 * time.Second, TotalTimeout: 10 * time.Second, MaxBodyBytes: 16 << 10, MaxConcurrency: 8})

// Register binds the independent event box to optional generation-frozen delivery.
func Register() {
	agenteventservice.RegisterDeliveryHook(EnqueueEventTx)
	agenteventservice.RegisterWithdrawalHook(func(tx *gorm.DB, instance string, eventIDs []string) error {
		if err := agentWebhook.RedactBySourceEventsTx(tx, instance, eventIDs); err != nil {
			return err
		}
		tasks, err := agentWebhook.TasksForEventsTx(tx, instance, eventIDs)
		if err != nil {
			return err
		}
		return taskQueue.CancelPendingIDsTx(tx, tasks, "withdrawn")
	})
}
func EnqueueEventTx(tx *gorm.DB, event agentEvents.Entity, generation uint64) error {
	if generation == 0 {
		return nil
	}
	agent, err := agents.GetTx(tx, event.AgentID, true)
	if err != nil {
		return err
	}
	if !agent.WebhookEnabled || agent.EndpointGeneration != generation || agent.Enabled != agents.StatusEnabled || agent.SecretCiphertext == "" {
		return nil
	}
	body, err := json.Marshal(agenteventservice.Envelope(event))
	if err != nil {
		return err
	}
	row := agentWebhook.Delivery{InstanceID: event.InstanceID, EventID: event.ID, AgentID: event.AgentID, EndpointGeneration: generation, SchemaVersion: 1, Status: agentWebhook.Pending, Body: string(body), Round: 1, Deadline: minTime(time.Now().Add(24*time.Hour), event.ExpiresAt), ExpiresAt: event.ExpiresAt}
	pending, err := agentWebhook.PendingCountTx(tx, event.InstanceID, event.AgentID)
	if err != nil {
		return err
	}
	if pending >= MaxPendingPerAgent {
		row.Status = agentWebhook.Dead
		row.Reason = "queue_capacity"
		_, err := agentWebhook.CreateTx(tx, &row)
		return err
	}
	return enqueueTx(tx, &row)
}
func enqueueTx(tx *gorm.DB, row *agentWebhook.Delivery) error {
	created, err := agentWebhook.CreateTx(tx, row)
	if err != nil || !created {
		return err
	}
	return scheduleTx(tx, row)
}
func scheduleTx(tx *gorm.DB, row *agentWebhook.Delivery) error {
	payload, err := json.Marshal(struct {
		InstanceID string `json:"instanceId"`
		DeliveryID uint64 `json:"deliveryId"`
	}{row.InstanceID, row.ID})
	if err != nil {
		return err
	}
	task := taskQueue.Entity{Type: TaskType, TaskJson: string(payload), ScheduleGroup: strconv.FormatUint(row.AgentID, 10)}
	if err := taskQueue.CreateTx(tx, &task); err != nil {
		return err
	}
	row.TaskID = task.Id
	return agentWebhook.UpdateTx(tx, row.InstanceID, row.ID, map[string]any{"task_id": task.Id})
}
func Test(agentID, adminID uint64) (agentWebhook.Delivery, error) {
	state := agentinstance.Current()
	if !state.WebhookEnabled {
		return agentWebhook.Delivery{}, ErrUnavailable
	}
	var row agentWebhook.Delivery
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		agent, err := agents.GetTx(tx, agentID, true)
		if err != nil {
			return err
		}
		if agent.Enabled != agents.StatusEnabled || !agent.WebhookEnabled || agent.WebhookEndpoint == "" {
			return ErrUnavailable
		}
		if agent.SecretCiphertext == "" {
			return ErrSecretRequired
		}
		now := time.Now()
		eventID := "evt_test_" + uuid.NewString()
		body, err := json.Marshal(map[string]any{"id": eventID, "instanceId": state.ID, "schemaVersion": 1, "type": "agent.webhook_test", "occurredAt": now, "agentId": agentID, "test": true})
		if err != nil {
			return err
		}
		row = agentWebhook.Delivery{InstanceID: state.ID, EventID: eventID, AgentID: agentID, EndpointGeneration: agent.EndpointGeneration, SchemaVersion: 1, Status: agentWebhook.Pending, Body: string(body), Round: 1, Deadline: now.Add(24 * time.Hour), ExpiresAt: now.Add(agenteventservice.Retention), CreatedBy: adminID}
		return enqueueTx(tx, &row)
	})
	return row, err
}
func eventTx(tx *gorm.DB, instance string, agentID uint64, id string) (agentEvents.Entity, error) {
	return agentEvents.EventTx(tx, instance, agentID, id)
}
func Redeliver(agentID, deliveryID, adminID uint64) error {
	state := agentinstance.Current()
	if !state.WebhookEnabled {
		return ErrUnavailable
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		row, err := agentWebhook.GetTx(tx, state.ID, deliveryID, false)
		if err != nil || row.AgentID != agentID {
			return ErrNotReplayable
		}
		if row.Status != agentWebhook.Dead && row.Status != agentWebhook.Accepted {
			return ErrNotReplayable
		}
		if !strings.HasPrefix(row.EventID, "evt_test_") {
			event, err := eventTx(tx, state.ID, row.AgentID, row.EventID)
			if err != nil {
				return ErrNotReplayable
			}
			allowed, err := agenteventservice.AuthorizeEventTx(tx, &event)
			if err != nil {
				return err
			}
			if !allowed {
				return ErrNotReplayable
			}
		}
		agent, err := agents.GetTx(tx, agentID, true)
		if err != nil {
			return err
		}
		if !agent.WebhookEnabled || agent.Enabled != agents.StatusEnabled || agent.EndpointGeneration != row.EndpointGeneration {
			return ErrNotReplayable
		}
		row, err = agentWebhook.GetTx(tx, state.ID, deliveryID, true)
		if err != nil {
			return err
		}
		if row.Status != agentWebhook.Dead && row.Status != agentWebhook.Accepted {
			return ErrNotReplayable
		}
		now := time.Now()
		if !now.Before(row.ExpiresAt) || row.Body == "" {
			return ErrNotReplayable
		}
		pending, err := agentWebhook.PendingCountTx(tx, state.ID, agentID)
		if err != nil {
			return err
		}
		if pending >= MaxPendingPerAgent {
			return ErrUnavailable
		}
		if row.TaskID == 0 {
			if err := scheduleTx(tx, row); err != nil {
				return err
			}
		} else if err := taskQueue.ResetTerminalTx(tx, row.TaskID, now); err != nil {
			return err
		}
		return agentWebhook.UpdateTx(tx, state.ID, row.ID, map[string]any{"status": agentWebhook.Pending, "reason": "", "round": row.Round + 1, "attempt_count": 0, "deadline": minTime(now.Add(24*time.Hour), row.ExpiresAt), "next_run_at": now, "last_redelivered_by": adminID})
	})
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Sign preserves the exact raw payload and implements Standard Webhooks v1.
func Sign(secret, eventID string, timestamp int64, body []byte) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil || !strings.HasPrefix(secret, "whsec_") || len(decoded) < 32 {
		return "", ErrSecretRequired
	}
	mac := hmac.New(sha256.New, decoded)
	_, _ = mac.Write([]byte(eventID + "." + strconv.FormatInt(timestamp, 10) + "."))
	_, _ = mac.Write(body)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

type permit struct {
	delivery  agentWebhook.Delivery
	endpoint  string
	secret    string
	previous  string
	attemptID string
	until     time.Time
}

func authorizeTx(tx *gorm.DB, task *taskQueue.Entity, id uint64) (permit, error) {
	state := agentinstance.Current()
	p := permit{}
	ok, err := taskQueue.OwnedTx(tx, task.Id, task.LeaseToken)
	if err != nil {
		return p, err
	}
	if !ok {
		return p, ErrLeaseLost
	}
	row, err := agentWebhook.GetTx(tx, state.ID, id, false)
	if err != nil {
		return p, err
	}
	reason := ""
	if !state.WebhookEnabled {
		reason = "instance_disabled"
	}
	if row.Status != agentWebhook.Pending && row.Status != agentWebhook.RetryWait && row.Status != agentWebhook.Running {
		reason = "terminal"
	}
	now := time.Now()
	if !now.Before(row.ExpiresAt) {
		reason = "expired"
	}
	if reason == "" && (!now.Before(row.Deadline) || row.AttemptCount >= 9) {
		reason = "retry_budget_exhausted"
	}
	if row.Body == "" {
		reason = "withdrawn"
	}
	if reason == "" && !strings.HasPrefix(row.EventID, "evt_test_") {
		event, err := eventTx(tx, state.ID, row.AgentID, row.EventID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reason = "withdrawn"
		} else if err != nil {
			return p, err
		} else {
			allowed, err := agenteventservice.AuthorizeEventTx(tx, &event)
			if err != nil {
				return p, err
			}
			if !allowed {
				reason = "withdrawn"
			}
		}
	}
	agent, err := agents.GetTx(tx, row.AgentID, true)
	if err != nil {
		return p, err
	}
	// Configuration owns Agent before delivery/task rows. Take the fenced task
	// write only after source/participant/Agent authorization locks, otherwise
	// completion or disablement can form a task -> Agent -> task cycle.
	ok, err = taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusRunning, nil, "")
	if err != nil {
		return p, err
	}
	if !ok {
		return p, ErrLeaseLost
	}
	row, err = agentWebhook.GetTx(tx, state.ID, id, true)
	if err != nil {
		return p, err
	}
	if row.Body == "" {
		reason = "withdrawn"
	}
	if row.Status != agentWebhook.Pending && row.Status != agentWebhook.RetryWait && row.Status != agentWebhook.Running {
		reason = "terminal"
	}
	if reason == "" && (agent.Enabled != agents.StatusEnabled || !agent.WebhookEnabled || agent.EndpointGeneration != row.EndpointGeneration) {
		reason = "configuration_changed"
	}
	if reason == "" && agent.WebhookPausedReason != "" {
		reason = agent.WebhookPausedReason
	}
	if reason == "" && safefetch.ValidateWebhookURL(agent.WebhookEndpoint) != nil {
		reason = "target_blocked"
	}
	if reason != "" {
		if reason == "instance_disabled" {
			due := time.Now().Add(5 * time.Minute)
			_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusRetrying, &due, reason)
			return p, err
		}
		if reason == "terminal" {
			_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusSuccess, nil, reason)
			return p, err
		}
		if reason == "retry_budget_exhausted" {
			if err := agentWebhook.UpdateTx(tx, state.ID, row.ID, map[string]any{"status": agentWebhook.Dead, "reason": reason}); err != nil {
				return p, err
			}
			_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusFailed, nil, reason)
			return p, err
		}
		if err := agentWebhook.UpdateTx(tx, state.ID, row.ID, map[string]any{"status": agentWebhook.Cancelled, "reason": reason, "body": ""}); err != nil {
			return p, err
		}
		_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusSuccess, nil, reason)
		return p, err
	}
	secret, err := securestore.DecryptPurpose(agent.SecretCiphertext, securestore.AgentWebhookSecretPurpose)
	if err != nil {
		return p, finishBlockedTx(tx, task, row, "secret_invalid")
	}
	if _, err = Sign(secret, row.EventID, now.Unix(), []byte(row.Body)); err != nil {
		return p, finishBlockedTx(tx, task, row, "secret_invalid")
	}
	previous := ""
	if agent.PreviousSecretCiphertext != "" && agent.PreviousSecretExpiresAt != nil && now.Before(*agent.PreviousSecretExpiresAt) {
		previous, err = securestore.DecryptPurpose(agent.PreviousSecretCiphertext, securestore.AgentWebhookSecretPurpose)
		if err != nil {
			return p, finishBlockedTx(tx, task, row, "secret_invalid")
		}
	}
	p = permit{delivery: *row, endpoint: agent.WebhookEndpoint, secret: secret, previous: previous, attemptID: uuid.NewString(), until: now.Add(permitTTL)}
	attempt := agentWebhook.Attempt{ID: p.attemptID, InstanceID: state.ID, DeliveryID: row.ID, Round: row.Round, Number: row.AttemptCount + 1, AuthorizedAt: now}
	if err := agentWebhook.AddAttemptTx(tx, &attempt); err != nil {
		return p, err
	}
	err = agentWebhook.UpdateTx(tx, state.ID, row.ID, map[string]any{"status": agentWebhook.Running, "attempt_count": row.AttemptCount + 1, "total_attempts": row.TotalAttempts + 1, "permit_expires_at": p.until, "last_attempt_id": p.attemptID})
	p.delivery.AttemptCount++
	return p, err
}
func finishBlockedTx(tx *gorm.DB, task *taskQueue.Entity, row *agentWebhook.Delivery, reason string) error {
	if err := agentWebhook.UpdateTx(tx, row.InstanceID, row.ID, map[string]any{"status": agentWebhook.Dead, "reason": reason}); err != nil {
		return err
	}
	if err := agents.UpdateColumns(tx, row.AgentID, map[string]any{"webhook_paused_reason": reason}); err != nil {
		return err
	}
	_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusFailed, nil, reason)
	return err
}

var limits = struct {
	sync.Mutex
	keys map[string]int
}{keys: map[string]int{}}

func acquireKeys(agentID uint64, endpoint string) (func(), bool) {
	u, _ := urlParse(endpoint)
	keys := []string{"agent:" + strconv.FormatUint(agentID, 10), "host:" + u}
	limits.Lock()
	defer limits.Unlock()
	for _, key := range keys {
		if limits.keys[key] >= 2 {
			return nil, false
		}
	}
	for _, key := range keys {
		limits.keys[key]++
	}
	return func() {
		limits.Lock()
		defer limits.Unlock()
		for _, key := range keys {
			limits.keys[key]--
			if limits.keys[key] == 0 {
				delete(limits.keys, key)
			}
		}
	}, true
}
func urlParse(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return strings.ToLower(u.Hostname()), nil
}
func HandleTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload struct {
		InstanceID string `json:"instanceId"`
		DeliveryID uint64 `json:"deliveryId"`
	}
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return err
	}
	state := agentinstance.Current()
	if payload.InstanceID != state.ID {
		return db.Connect().Transaction(func(tx *gorm.DB) error {
			_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusSuccess, nil, "foreign_instance")
			return err
		})
	}
	// Reserve fair Agent/host capacity before creating a send permit.
	row, err := agentWebhook.GetTx(db.Connect(), state.ID, payload.DeliveryID, false)
	if err != nil {
		return err
	}
	agent := agents.GetByUserID(row.AgentID)
	if agent == nil {
		return ErrUnavailable
	}
	release, ok := acquireKeys(row.AgentID, agent.WebhookEndpoint)
	if !ok {
		due := time.Now().Add(time.Second)
		return db.Connect().Transaction(func(tx *gorm.DB) error {
			_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusRetrying, &due, "capacity_wait")
			return err
		})
	}
	defer release()
	var p permit
	err = db.Connect().Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, task, payload.DeliveryID); return err })
	if errors.Is(err, ErrLeaseLost) {
		return nil
	}
	if err != nil {
		return err
	}
	if p.attemptID == "" {
		return nil
	}
	if ctx.Err() != nil || !time.Now().Before(p.until) {
		return complete(task, p, safefetch.Result{}, context.DeadlineExceeded)
	}
	stamp := time.Now().Unix()
	signature, err := Sign(p.secret, p.delivery.EventID, stamp, []byte(p.delivery.Body))
	if err != nil {
		return complete(task, p, safefetch.Result{}, err)
	}
	if p.previous != "" {
		old, err := Sign(p.previous, p.delivery.EventID, stamp, []byte(p.delivery.Body))
		if err != nil {
			return complete(task, p, safefetch.Result{}, err)
		}
		signature += " " + old
	}
	headers := http.Header{}
	headers.Set("Webhook-Id", p.delivery.EventID)
	headers.Set("Webhook-Timestamp", strconv.FormatInt(stamp, 10))
	headers.Set("Webhook-Signature", signature)
	headers.Set("Webhook-Attempt-Id", p.attemptID)
	headers.Set("Webhook-Delivery-Id", strconv.FormatUint(p.delivery.ID, 10))
	requestCtx, cancel := context.WithDeadline(ctx, p.until)
	defer cancel()
	result, sendErr := sender.PostJSON(requestCtx, p.endpoint, []byte(p.delivery.Body), headers)
	return complete(task, p, result, sendErr)
}
func retryDelay(attempt uint32, retryAfter string, now, deadline time.Time) time.Time {
	seconds := time.Duration(1<<min(attempt, 8)) * time.Minute
	random := make([]byte, 1)
	if _, err := rand.Read(random); err == nil {
		seconds += time.Duration(random[0]) * seconds / 1024
	}
	due := now.Add(seconds)
	if parsed, err := strconv.Atoi(retryAfter); err == nil && parsed >= 0 {
		due = maxTime(due, now.Add(time.Duration(min(parsed, 86400))*time.Second))
	} else if parsed, err := http.ParseTime(retryAfter); err == nil {
		due = maxTime(due, parsed)
	}
	return minTime(due, deadline)
}
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
func complete(task *taskQueue.Entity, p permit, result safefetch.Result, sendErr error) error {
	state := agentinstance.Current()
	now := time.Now()
	status := agentWebhook.Dead
	reason := "receiver_rejected"
	taskStatus := uint8(taskQueue.StatusFailed)
	var due *time.Time
	switch {
	case sendErr == nil && result.StatusCode >= 200 && result.StatusCode < 300:
		status = agentWebhook.Accepted
		reason = ""
		taskStatus = taskQueue.StatusSuccess
	case safefetch.IsClass(sendErr, safefetch.ErrorBlocked) || safefetch.IsClass(sendErr, safefetch.ErrorInvalid):
		reason = "target_blocked"
	case errors.Is(sendErr, ErrSecretRequired):
		reason = "secret_invalid"
	case result.StatusCode == http.StatusGone:
		reason = "receiver_gone"
	case sendErr != nil || result.StatusCode == http.StatusRequestTimeout || result.StatusCode == http.StatusTooManyRequests || result.StatusCode >= 500:
		reason = "network_unavailable"
		if sendErr == nil {
			reason = "receiver_retryable"
		}
		if p.delivery.AttemptCount < 9 && now.Before(p.delivery.Deadline) {
			status = agentWebhook.RetryWait
			taskStatus = taskQueue.StatusRetrying
			next := retryDelay(p.delivery.AttemptCount, result.RetryAfter, now, p.delivery.Deadline)
			due = &next
		} else {
			reason = "retry_budget_exhausted"
		}
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		// Keep Agent -> task -> delivery ordering consistent with permits and
		// configuration withdrawal. No source locks are needed for diagnostics.
		if _, err := agents.GetTx(tx, p.delivery.AgentID, true); err != nil {
			return err
		}
		owned, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskStatus, due, reason)
		if err != nil {
			return err
		}
		if !owned {
			return ErrLeaseLost
		}
		row, err := agentWebhook.GetTx(tx, state.ID, p.delivery.ID, true)
		if err != nil {
			return err
		}
		if row.LastAttemptID != p.attemptID {
			return ErrLeaseLost
		}
		if err := agentWebhook.CompleteAttemptTx(tx, state.ID, p.attemptID, map[string]any{"http_status": result.StatusCode, "error_class": reason, "duration_ms": result.Duration.Milliseconds(), "completed_at": now}); err != nil {
			return err
		}
		updates := map[string]any{"status": status, "reason": reason, "next_run_at": due, "permit_expires_at": nil}
		if status == agentWebhook.Accepted {
			updates["accepted_at"] = now
		}
		if row.Status == agentWebhook.Cancelled || row.Body == "" {
			updates["status"] = agentWebhook.Cancelled
			updates["reason"] = "withdrawn"
		} else if status == agentWebhook.Accepted {
			updates["accepted_at"] = now
		}
		if err := agentWebhook.UpdateTx(tx, state.ID, row.ID, updates); err != nil {
			return err
		}
		if status == agentWebhook.Accepted {
			return agents.UpdateColumns(tx, row.AgentID, map[string]any{"last_webhook_accepted_at": now})
		}
		if reason == "receiver_gone" || reason == "target_blocked" || reason == "secret_invalid" {
			return agents.UpdateWebhookGenerationTx(tx, row.AgentID, row.EndpointGeneration, map[string]any{"webhook_paused_reason": reason})
		}
		return nil
	})
}
