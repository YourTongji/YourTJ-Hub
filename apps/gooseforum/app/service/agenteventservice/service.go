// Package agenteventservice orchestrates human-to-Agent public interactions.
// Lock order: source post, topic, participant users (ascending), Agents (ascending).
// Event sequence allocation remains under the Agent lock through commit.
package agenteventservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const Retention = 7 * 24 * time.Hour
const TaskType = "agent-interaction.materialize"
const MaxMentionRecipients = 20

// MaxBroadcastDepth bounds event-driven Agent chains: a post answering an
// event deeper than this many hops is not broadcast again.
const MaxBroadcastDepth = 3

// MaxConsecutiveBotPosts suppresses broadcasts once a topic's newest posts are
// all Agent-authored, which stops loops that ignore sourceEventId.
const MaxConsecutiveBotPosts = 5

type Error struct {
	Code        string `json:"code"`
	ReplayFloor string `json:"replayFloor,omitempty"`
}

func (e *Error) Error() string { return e.Code }

var ErrInaccessible = &Error{Code: "inaccessible"}

type EventData struct {
	TopicID       uint64   `json:"topicId"`
	PostID        uint64   `json:"postId"`
	PostNo        uint64   `json:"postNo"`
	ReplyToPostID uint64   `json:"replyToPostId"`
	ActorID       uint64   `json:"actorId"`
	ActorType     string   `json:"actorType"`
	Reasons       []string `json:"reasons"`
	URL           string   `json:"url"`
}
type Event struct {
	ID               string     `json:"id"`
	InstanceID       string     `json:"instanceId"`
	SchemaVersion    int        `json:"schemaVersion"`
	Type             string     `json:"type"`
	OccurredAt       time.Time  `json:"occurredAt"`
	AgentID          uint64     `json:"agentId"`
	State            string     `json:"state"`
	Data             *EventData `json:"data,omitempty"`
	AckedAt          *time.Time `json:"ackedAt,omitempty"`
	ResultingTopicID uint64     `json:"resultingTopicId,omitempty"`
	ResultingPostID  uint64     `json:"resultingPostId,omitempty"`
}
type Page struct {
	Events      []Event `json:"events"`
	NextCursor  string  `json:"nextCursor"`
	HasMore     bool    `json:"hasMore"`
	ReplayFloor string  `json:"replayFloor"`
}
type cursor struct {
	Instance string `json:"i"`
	Epoch    string `json:"e"`
	Agent    uint64 `json:"a"`
	Seq      uint64 `json:"s"`
}

func encodeCursor(instance, epoch string, agent, seq uint64) string {
	b, err := json.Marshal(cursor{instance, epoch, agent, seq})
	if err != nil {
		panic(err) // This fixed struct contains only JSON-supported primitive fields.
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

var deliveryHook func(*gorm.DB, agentEvents.Entity, uint64) error
var withdrawalHook func(*gorm.DB, string, []string) error

func RegisterWithdrawalHook(h func(*gorm.DB, string, []string) error) { withdrawalHook = h }

func RegisterDeliveryHook(h func(*gorm.DB, agentEvents.Entity, uint64) error) { deliveryHook = h }

// PublicSourceTx provides the common revocation/source-write authorization lock.
// Every current visibility and block check is performed on transaction owner APIs.
func PublicSourceTx(tx *gorm.DB, postID uint64) (posts.Entity, topics.Entity, error) {
	p, err := posts.GetUnscopedTx(tx, postID)
	if err != nil {
		return p, topics.Entity{}, err
	}
	t, err := topics.GetUnscopedTx(tx, p.TopicId)
	if err != nil {
		return p, t, err
	}
	if p.DeletedAt.Valid || p.IsAnonymous || p.ProcessStatus != posts.ProcessStatusNormal || p.VisibilityStatus != posts.VisibilityActive || p.RetentionStatus == posts.RetentionPurged || t.DeletedAt.Valid || t.Status != 1 || t.ProcessStatus != topics.ProcessStatusNormal || t.VisibilityStatus != topics.VisibilityActive || t.TopicType != topics.TopicTypeForum || t.RetentionStatus == topics.RetentionPurged {
		return p, t, ErrInaccessible
	}
	if t.FirstPostId != 0 && t.FirstPostId != p.Id {
		first, err := posts.GetCurrentTx(tx, t.FirstPostId)
		if err != nil {
			return p, t, err
		}
		if first.DeletedAt.Valid || first.IsAnonymous || first.ProcessStatus != posts.ProcessStatusNormal || first.VisibilityStatus != posts.VisibilityActive {
			return p, t, ErrInaccessible
		}
	}
	return p, t, nil
}

// BaselineTx records a genuinely previously public revision for legacy content.
// It runs before overwriting an existing revision, never in the async worker.
func BaselineTx(tx *gorm.DB, postID, version uint64) error {
	cfg := agentinstance.Current()
	if cfg.ID == "" || cfg.Epoch == "" {
		return nil
	}
	_, err := agentEvents.PublicationTx(tx, cfg.ID, postID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return agentEvents.SetPublicationTx(tx, agentEvents.Publication{InstanceID: cfg.ID, PostID: postID, Version: version})
}

// CapturePublicTx advances publication state and freezes numeric recipients and
// subscription/endpoint generations in the same transaction as the content.
func CapturePublicTx(tx *gorm.DB, post *posts.Entity) error {
	cfg := agentinstance.Current()
	if cfg.ID == "" || cfg.Epoch == "" {
		return nil
	}
	if err := validateConfig(cfg); err != nil {
		return err
	}
	p, t, err := PublicSourceTx(tx, post.Id)
	if errors.Is(err, ErrInaccessible) || errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	rev, err := postRevisions.LatestTx(tx, p.Id)
	if err != nil {
		return err
	}
	state, err := agentEvents.PublicationTx(tx, cfg.ID, p.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if state.Version >= rev.Version {
		return nil
	}
	previous := state.Version
	if previous == 0 && rev.Version > 1 {
		prior, found, err := PreviousPublicTx(tx, p.Id)
		if err != nil {
			return err
		}
		if found {
			previous = prior.Version
		}
	}
	if err := agentEvents.SetPublicationTx(tx, agentEvents.Publication{InstanceID: cfg.ID, PostID: p.Id, Version: rev.Version}); err != nil {
		return err
	}
	// Pausing production suppresses new occurrences, while the last public
	// revision must keep advancing so re-enabling cannot adopt historical edits.
	if !cfg.ProducerEnabled {
		return nil
	}
	actor, err := users.GetInteractionUserTx(tx, p.UserId)
	if err != nil {
		return err
	}
	if actor.IsFrozen != users.StatusNormal {
		return nil
	}
	reasons := map[uint64][]string{}
	if actor.ActorType == users.ActorTypeHuman {
		// Directed interaction reasons stay human-only; Agent-authored content
		// reaches Agents only through explicit broadcast subscriptions.
		currentIDs, err := users.ResolveMentionIDsTx(tx, rev.Content, p.UserId, MaxMentionRecipients)
		if err != nil {
			return err
		}
		old := map[uint64]bool{}
		if previous > 0 {
			oldRev, err := postRevisions.VersionTx(tx, p.Id, previous)
			if err != nil {
				return err
			}
			ids, err := users.ResolveMentionIDsTx(tx, oldRev.Content, p.UserId, 0)
			if err != nil {
				return err
			}
			for _, id := range ids {
				old[id] = true
			}
		}
		if previous == 0 && p.PostNo > 1 && p.ReplyToPostId != 0 {
			parent, err := posts.GetCurrentTx(tx, p.ReplyToPostId)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil && parent.TopicId == p.TopicId && !parent.IsAnonymous && !parent.DeletedAt.Valid && parent.VisibilityStatus == posts.VisibilityActive && parent.ProcessStatus == posts.ProcessStatusNormal {
				reasons[parent.UserId] = append(reasons[parent.UserId], "post_reply")
			}
		}
		for _, id := range currentIDs {
			if !old[id] {
				reasons[id] = append(reasons[id], "mention")
			}
		}
		if previous == 0 && p.PostNo > 1 {
			reasons[t.UserId] = append(reasons[t.UserId], "comment")
		}
	}
	if err := appendBroadcastReasonsTx(tx, reasons, p, previous); err != nil {
		return err
	}
	ids := make([]uint64, 0, len(reasons))
	for id := range reasons {
		if id != p.UserId && id != 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if err := users.LockInteractionUserIDs(tx, append(ids, p.UserId)); err != nil {
		return err
	}
	allowed, err := users.FilterInteractionRecipientsTx(tx, p.UserId, ids)
	if err != nil {
		return err
	}
	recipients := make([]agentEvents.Recipient, 0, len(allowed))
	now := time.Now().UTC()
	for _, id := range allowed {
		u, err := users.GetInteractionUserTx(tx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		if u.ActorType != users.ActorTypeBot || u.IsFrozen != users.StatusNormal {
			continue
		}
		a, err := agents.GetTx(tx, id, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		if a.Enabled != agents.StatusEnabled || !a.EventsEnabled || a.EventsEnabledAt == nil || a.EventsEnabledAt.After(now) {
			continue
		}
		filtered := filterReasons(reasons[id], a.EventTypes)
		if len(filtered) == 0 {
			continue
		}
		endpoint := uint64(0)
		if a.WebhookEnabled && a.WebhookEndpoint != "" {
			endpoint = a.EndpointGeneration
		}
		recipients = append(recipients, agentEvents.Recipient{AgentID: id, Reasons: filtered, SubscriptionGeneration: a.SubscriptionGeneration, EndpointGeneration: endpoint})
	}
	if len(recipients) == 0 {
		return nil
	}
	i := agentEvents.Intent{ID: "src_" + uuid.NewString(), InstanceID: cfg.ID, PostID: p.Id, Version: rev.Version, PreviousVersion: previous, ActorID: p.UserId, Recipients: recipients, Status: "pending", CreatedAt: now, ExpiresAt: now.Add(Retention)}
	if err := agentEvents.CreateIntentTx(tx, &i); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]string{"instanceId": cfg.ID, "intentId": i.ID})
	if err != nil {
		return err
	}
	task := taskQueue.Entity{Type: TaskType, TaskJson: string(payload)}
	if err := taskQueue.CreateTx(tx, &task); err != nil {
		return err
	}
	return agentEvents.BindTaskTx(tx, i.ID, task.Id)
}
func eventType(reason string) string {
	switch reason {
	case "post_reply":
		return "agent.post_replied"
	case "mention":
		return "agent.mentioned"
	case "topic_created":
		return "forum.topic_created"
	case "post_created":
		return "forum.post_created"
	default:
		return "agent.topic_commented"
	}
}

func isBroadcastReason(reason string) bool {
	return reason == "topic_created" || reason == "post_created"
}

func broadcastOnly(reasons []string) bool {
	if len(reasons) == 0 {
		return false
	}
	for _, reason := range reasons {
		if !isBroadcastReason(reason) {
			return false
		}
	}
	return true
}

func dropBroadcastReasons(reasons []string) []string {
	kept := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if !isBroadcastReason(reason) {
			kept = append(kept, reason)
		}
	}
	return kept
}

func intentBroadcastOnly(i agentEvents.Intent) bool {
	if len(i.Recipients) == 0 {
		return false
	}
	for _, recipient := range i.Recipients {
		if !broadcastOnly(recipient.Reasons) {
			return false
		}
	}
	return true
}

func actorTypeLabel(actorType int8) string {
	if actorType == users.ActorTypeBot {
		return "bot"
	}
	return "human"
}

// appendBroadcastReasonsTx adds the forum-wide types to every enabled Agent
// subscription. Directed reasons are appended first so an interaction keeps its
// more specific type when both apply to the same Agent.
func appendBroadcastReasonsTx(tx *gorm.DB, reasons map[uint64][]string, p posts.Entity, previous uint64) error {
	if previous != 0 {
		return nil
	}
	reason := "post_created"
	if p.PostNo == 1 {
		reason = "topic_created"
	}
	subscribers, err := agents.ListEnabledTx(tx)
	if err != nil {
		return err
	}
	for _, subscriber := range subscribers {
		if subscriber.UserId == 0 || subscriber.UserId == p.UserId {
			continue
		}
		reasons[subscriber.UserId] = append(reasons[subscriber.UserId], reason)
	}
	return nil
}

// botBroadcastAllowedTx bounds event-driven Agent chatter. A post answering an
// event deeper than MaxBroadcastDepth hops is not broadcast again, and neither
// is a post that extends a run of MaxConsecutiveBotPosts Agent posts in its
// topic; both rules together break chains even when a client ignores
// sourceEventId.
func botBroadcastAllowedTx(tx *gorm.DB, instanceID string, p posts.Entity) (bool, error) {
	parent, found, err := agentEvents.ResultingEventForPostTx(tx, instanceID, p.Id)
	if err != nil {
		return false, err
	}
	if found {
		depth, err := chainDepthTx(tx, instanceID, parent, MaxBroadcastDepth)
		if err != nil {
			return false, err
		}
		if depth+1 > MaxBroadcastDepth {
			return false, nil
		}
	}
	authorIDs, err := posts.TailAuthorIDsTx(tx, p.TopicId, MaxConsecutiveBotPosts)
	if err != nil {
		return false, err
	}
	if len(authorIDs) < MaxConsecutiveBotPosts {
		return true, nil
	}
	for _, authorID := range authorIDs {
		author, err := users.GetInteractionUserTx(tx, authorID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if author.ActorType != users.ActorTypeBot {
			return true, nil
		}
	}
	return false, nil
}

// chainDepthTx walks the resulting-post links between events; withdrawn events
// have redacted their result, which ends the walk and stays permissive.
func chainDepthTx(tx *gorm.DB, instanceID string, event agentEvents.Entity, limit int) (int, error) {
	depth := 0
	current := event
	for depth < limit {
		parent, found, err := agentEvents.ResultingEventForPostTx(tx, instanceID, current.PostID)
		if err != nil {
			return 0, err
		}
		if !found {
			break
		}
		depth++
		current = parent
	}
	return depth, nil
}
func filterReasons(reasons []string, typesJSON string) []string {
	var types []string
	_ = json.Unmarshal([]byte(typesJSON), &types)
	result := make([]string, 0)
	for _, r := range reasons {
		for _, t := range types {
			if eventType(r) == t {
				result = append(result, r)
				break
			}
		}
	}
	return result
}
func stableID(source string, agent uint64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", source, agent)))
	return fmt.Sprintf("evt_%x", sum[:24])
}

// HandleTask materializes and marks its owned task in one transaction. A stale
// worker cannot commit events after its task lease has been reclaimed.
func HandleTask(ctx context.Context, task *taskQueue.Entity) error {
	err := handleTask(ctx, task)
	if err != nil {
		// Domain diagnostics share fencing with the scheduler. Retain only a stable
		// error class: neither source text nor raw database errors belong here.
		var payload struct {
			InstanceID string `json:"instanceId"`
			IntentID   string `json:"intentId"`
		}
		if json.Unmarshal([]byte(task.TaskJson), &payload) == nil {
			diagnosticErr := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
				owned, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusRunning, nil, "")
				if err != nil || !owned {
					return err
				}
				status := "retry_wait"
				if task.RetryCount+1 >= 9 {
					status = "failed"
				}
				return agentEvents.UpdateIntentTx(tx, payload.IntentID, status, "materialization_error")
			})
			if diagnosticErr != nil {
				return errors.Join(err, diagnosticErr)
			}
		}
	}
	return err
}
func handleTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload struct {
		InstanceID string `json:"instanceId"`
		IntentID   string `json:"intentId"`
	}
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return err
	}
	cfg := agentinstance.Current()
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if !cfg.ProducerEnabled && payload.InstanceID == cfg.ID {
			due := time.Now().Add(time.Minute)
			_, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusRetrying, &due, "producer_paused")
			return err
		}
		owned, err := taskQueue.TransitionOwnedTx(tx, task.Id, task.LeaseToken, taskQueue.StatusSuccess, nil, "")
		if err != nil {
			return err
		}
		if !owned {
			return nil
		}
		if !cfg.ProducerEnabled || payload.InstanceID != cfg.ID {
			return nil
		}
		i, err := agentEvents.IntentTx(tx, cfg.ID, payload.IntentID)
		if err != nil {
			return err
		}
		if i.Status == "materialized" || i.Status == "cancelled" || i.Status == "expired" {
			return nil
		}
		if time.Now().After(i.ExpiresAt) {
			return agentEvents.UpdateIntentTx(tx, i.ID, "expired", "retention_expired")
		}
		p, t, err := PublicSourceTx(tx, i.PostID)
		if errors.Is(err, ErrInaccessible) || errors.Is(err, gorm.ErrRecordNotFound) {
			return agentEvents.UpdateIntentTx(tx, i.ID, "cancelled", "source_withdrawn")
		}
		if err != nil {
			return err
		}
		if _, err := postRevisions.VersionTx(tx, i.PostID, i.Version); err != nil {
			return err
		}
		ids := []uint64{i.ActorID}
		for _, r := range i.Recipients {
			ids = append(ids, r.AgentID)
		}
		if err := users.LockInteractionUserIDs(tx, ids); err != nil {
			return err
		}
		actor, err := users.GetInteractionUserTx(tx, i.ActorID)
		if err != nil {
			return err
		}
		actorHuman := actor.ActorType == users.ActorTypeHuman
		if actor.IsFrozen != users.StatusNormal || (!actorHuman && !intentBroadcastOnly(i)) {
			return agentEvents.UpdateIntentTx(tx, i.ID, "cancelled", "actor_unavailable")
		}
		broadcastAllowed := true
		if !actorHuman {
			broadcastAllowed, err = botBroadcastAllowedTx(tx, cfg.ID, p)
			if err != nil {
				return err
			}
		}
		targets := i.Recipients
		sort.Slice(targets, func(a, b int) bool { return targets[a].AgentID < targets[b].AgentID })
		for _, r := range targets {
			a, err := agents.GetTx(tx, r.AgentID, true)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			u, err := users.GetInteractionUserTx(tx, r.AgentID)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if a.Enabled != agents.StatusEnabled || !a.EventsEnabled || a.SubscriptionGeneration != r.SubscriptionGeneration || u.ActorType != users.ActorTypeBot || u.IsFrozen != users.StatusNormal {
				continue
			}
			allowed, err := users.FilterInteractionRecipientsTx(tx, i.ActorID, []uint64{r.AgentID})
			if err != nil {
				return err
			}
			if len(allowed) == 0 {
				continue
			}
			reasons := r.Reasons
			if !broadcastAllowed {
				reasons = dropBroadcastReasons(reasons)
				if len(reasons) == 0 {
					continue
				}
			}
			id := stableID(i.ID, r.AgentID)
			_, err = agentEvents.EventTx(tx, cfg.ID, r.AgentID, id)
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			seq, err := agents.ReserveEventSeqTx(tx, r.AgentID)
			if err != nil {
				return err
			}
			e := agentEvents.Entity{ID: id, InstanceID: cfg.ID, AgentID: r.AgentID, Seq: seq, SourceIntentID: i.ID, SubscriptionGeneration: r.SubscriptionGeneration, Type: eventType(reasons[0]), TopicID: t.Id, PostID: p.Id, PostNo: p.PostNo, ReplyToPostID: p.ReplyToPostId, ActorID: i.ActorID, ActorType: actorTypeLabel(actor.ActorType), Reasons: reasons, OccurredAt: i.CreatedAt, ExpiresAt: i.ExpiresAt}
			if err := agentEvents.CreateEventTx(tx, &e); err != nil {
				return err
			}
			if deliveryHook != nil && r.EndpointGeneration > 0 && a.WebhookEnabled && a.EndpointGeneration == r.EndpointGeneration {
				if err := deliveryHook(tx, e, r.EndpointGeneration); err != nil {
					return err
				}
			}
		}
		return agentEvents.UpdateIntentTx(tx, i.ID, "materialized", "")
	})
}

func Envelope(e agentEvents.Entity) Event {
	out := Event{ID: e.ID, InstanceID: e.InstanceID, SchemaVersion: 1, Type: e.Type, OccurredAt: e.OccurredAt, AgentID: e.AgentID, State: "active", AckedAt: e.AckedAt, ResultingPostID: e.ResultingPostID, ResultingTopicID: e.ResultingTopicID}
	if e.WithdrawnAt != nil {
		out.State = "withdrawn"
		return out
	}
	if time.Now().After(e.ExpiresAt) {
		out.State = "expired"
		return out
	}
	actorType := e.ActorType
	if actorType == "" {
		actorType = "human"
	}
	out.Data = &EventData{TopicID: e.TopicID, PostID: e.PostID, PostNo: e.PostNo, ReplyToPostID: e.ReplyToPostID, ActorID: e.ActorID, ActorType: actorType, Reasons: e.Reasons, URL: fmt.Sprintf("/p/post/%d/%d", e.TopicID, e.PostNo)}
	return out
}

// ValidateEventTx is shared by pull, ACK, sends and event-linked writes. Taking
// the source locks here serializes it with deletion and moderation commits.
func ValidateEventTx(tx *gorm.DB, e *agentEvents.Entity) error {
	if e.WithdrawnAt != nil || time.Now().After(e.ExpiresAt) {
		return ErrInaccessible
	}
	_, _, err := PublicSourceTx(tx, e.PostID)
	if err != nil {
		return err
	}
	if err := users.LockInteractionUserIDs(tx, []uint64{e.ActorID, e.AgentID}); err != nil {
		return err
	}
	actor, err := users.GetInteractionUserTx(tx, e.ActorID)
	if err != nil {
		return err
	}
	if actor.IsFrozen != users.StatusNormal || (actor.ActorType != users.ActorTypeHuman && !broadcastOnly(e.Reasons)) {
		return ErrInaccessible
	}
	if err := users.CheckInteractionAllowed(tx, e.ActorID, e.AgentID); err != nil {
		return err
	}
	a, err := agents.GetTx(tx, e.AgentID, true)
	if err != nil {
		return err
	}
	u, err := users.GetInteractionUserTx(tx, e.AgentID)
	if err != nil {
		return err
	}
	if a.Enabled != agents.StatusEnabled || !a.EventsEnabled || a.SubscriptionGeneration != e.SubscriptionGeneration || u.ActorType != users.ActorTypeBot || u.IsFrozen != users.StatusNormal {
		return ErrInaccessible
	}
	// The caller may have read this row before obtaining the source fences.
	// Withdrawal is permanent even when the source later becomes public again.
	current, err := agentEvents.EventTx(tx, e.InstanceID, e.AgentID, e.ID)
	if err != nil {
		return err
	}
	*e = current
	if e.WithdrawnAt != nil || time.Now().After(e.ExpiresAt) {
		return ErrInaccessible
	}

	return nil
}
func refreshTx(tx *gorm.DB, e *agentEvents.Entity) error {
	err := ValidateEventTx(tx, e)
	if errors.Is(err, ErrInaccessible) || errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, users.ErrInteractionBlocked) {
		if err := agentEvents.WithdrawTx(tx, e, time.Now().UTC()); err != nil {
			return err
		}
		if withdrawalHook != nil {
			return withdrawalHook(tx, e.InstanceID, []string{e.ID})
		}
		return nil
	}
	return err
}
func List(userID uint64, after string, limit int) (Page, error) {
	return ListWithCredential(userID, after, limit, "")
}
func ListWithCredential(userID uint64, after string, limit int, expectedTokenHash string) (Page, error) {
	cfg := agentinstance.Current()
	if !cfg.APIEnabled {
		return Page{}, ErrInaccessible
	}
	if err := validateConfig(cfg); err != nil {
		return Page{}, err
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	var c cursor
	if after != "" {
		b, err := base64.RawURLEncoding.DecodeString(after)
		if err != nil || json.Unmarshal(b, &c) != nil {
			return Page{}, &Error{Code: "cursor_invalid"}
		}
		if c.Agent != userID {
			return Page{}, ErrInaccessible
		}
		if c.Instance != cfg.ID || c.Epoch != cfg.Epoch {
			return Page{}, &Error{Code: "cursor_reset", ReplayFloor: encodeCursor(cfg.ID, cfg.Epoch, userID, 0)}
		}
	}
	page := Page{Events: make([]Event, 0)}
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		floor, err := agentEvents.ReplayFloorTx(tx, cfg.ID, userID, time.Now().UTC())
		if err != nil {
			return err
		}
		page.ReplayFloor = encodeCursor(cfg.ID, cfg.Epoch, userID, floor)
		if after != "" && c.Seq < floor {
			return &Error{Code: "cursor_expired", ReplayFloor: page.ReplayFloor}
		}
		if after == "" {
			c.Seq = floor
		}
		rows, err := agentEvents.ListTx(tx, cfg.ID, userID, c.Seq, limit+1)
		if err != nil {
			return err
		}
		page.HasMore = len(rows) > limit
		if page.HasMore {
			rows = rows[:limit]
		}
		if err := lockEventBatchTx(tx, rows); err != nil {
			return err
		}
		if err := validateCredentialTx(tx, userID, expectedTokenHash); err != nil {
			return err
		}
		for _, e := range rows {
			if err := refreshTx(tx, &e); err != nil {
				return err
			}
			page.Events = append(page.Events, Envelope(e))
			c.Seq = e.Seq
		}
		page.NextCursor = encodeCursor(cfg.ID, cfg.Epoch, userID, c.Seq)
		return nil
	})
	return page, err
}
func Get(userID uint64, eventID string) (Event, error) { return GetWithCredential(userID, eventID, "") }
func GetWithCredential(userID uint64, eventID, expectedTokenHash string) (Event, error) {
	cfg := agentinstance.Current()
	if !cfg.APIEnabled {
		return Event{}, ErrInaccessible
	}
	var result Event
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		e, err := agentEvents.EventTx(tx, cfg.ID, userID, eventID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInaccessible
		}
		if err != nil {
			return err
		}
		if err := lockEventBatchTx(tx, []agentEvents.Entity{e}); err != nil {
			return err
		}
		if err := validateCredentialTx(tx, userID, expectedTokenHash); err != nil {
			return err
		}
		if err := refreshTx(tx, &e); err != nil {
			return err
		}
		result = Envelope(e)
		return nil
	})
	return result, err
}
func Ack(userID uint64, ids []string) error { return AckWithCredential(userID, ids, "") }
func AckWithCredential(userID uint64, ids []string, expectedTokenHash string) error {
	cfg := agentinstance.Current()
	if !cfg.APIEnabled || len(ids) == 0 || len(ids) > 100 {
		return ErrInaccessible
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		rows := make([]agentEvents.Entity, 0, len(ids))
		for _, id := range ids {
			e, err := agentEvents.EventTx(tx, cfg.ID, userID, id)
			if err != nil {
				return ErrInaccessible
			}
			rows = append(rows, e)
		}
		if err := lockEventBatchTx(tx, rows); err != nil {
			return err
		}
		if err := validateCredentialTx(tx, userID, expectedTokenHash); err != nil {
			return err
		}
		for _, e := range rows {
			if err := ValidateEventTx(tx, &e); err != nil {
				return ErrInaccessible
			}
		}

		return agentEvents.AckTx(tx, cfg.ID, userID, ids, time.Now().UTC())
	})
}
func ValidateSourceTx(tx *gorm.DB, agentID uint64, eventID string, topicID uint64) error {
	if eventID == "" {
		return nil
	}
	cfg := agentinstance.Current()
	if !cfg.APIEnabled {
		return ErrInaccessible
	}
	e, err := agentEvents.EventTx(tx, cfg.ID, agentID, eventID)
	if err != nil {
		return ErrInaccessible
	}
	if e.TopicID != topicID {
		return ErrInaccessible
	}
	return ValidateEventTx(tx, &e)
}
func RecordResultTx(tx *gorm.DB, agentID uint64, eventID string, topicID, postID uint64) error {
	if eventID == "" {
		return nil
	}
	return agentEvents.RecordResultTx(tx, agentinstance.Current().ID, agentID, eventID, topicID, postID)
}
func ListIntents(limit int) ([]agentEvents.Intent, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return agentEvents.ListIntentsTx(db.Connect(), agentinstance.Current().ID, limit)
}
func ReplayIntent(id string) error {
	cfg := agentinstance.Current()
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		i, err := agentEvents.IntentTx(tx, cfg.ID, id)
		if err != nil {
			return err
		}
		if time.Now().After(i.ExpiresAt) || i.Status == "cancelled" || i.Status == "expired" {
			return ErrInaccessible
		}
		payload, err := json.Marshal(map[string]string{"instanceId": cfg.ID, "intentId": i.ID})
		if err != nil {
			return err
		}
		task := taskQueue.Entity{Type: TaskType, TaskJson: string(payload)}
		if err := taskQueue.CreateTx(tx, &task); err != nil {
			return err
		}
		if err := agentEvents.BindTaskTx(tx, id, task.Id); err != nil {
			return err
		}
		return agentEvents.UpdateIntentTx(tx, id, "pending", "")
	})
}

func validateConfig(cfg agentinstance.State) error {
	if cfg.ID == "" || cfg.Epoch == "" {
		return &Error{Code: "instance_unconfigured"}
	}
	return nil
}
func AuthorizeEventTx(tx *gorm.DB, e *agentEvents.Entity) (bool, error) {
	err := ValidateEventTx(tx, e)
	if errors.Is(err, ErrInaccessible) || errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, users.ErrInteractionBlocked) {
		return false, nil
	}
	return err == nil, err
}

type IntentView struct {
	IntentID           string    `json:"intentId"`
	SourceOccurrenceID string    `json:"sourceOccurrenceId"`
	PostID             uint64    `json:"postId"`
	Revision           uint64    `json:"revision"`
	Status             string    `json:"status"`
	ErrorCode          string    `json:"errorCode"`
	RetryCount         uint8     `json:"retryCount"`
	CreatedAt          time.Time `json:"createdAt"`
	ExpiresAt          time.Time `json:"expiresAt"`
}
type IntentPage struct {
	List     []IntentView `json:"list"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

func ListAgentIntents(agentID uint64, page, pageSize int) (IntentPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := agentEvents.PageIntentsForAgentTx(db.Connect(), agentinstance.Current().ID, agentID, page, pageSize)
	if err != nil {
		return IntentPage{}, err
	}
	views := make([]IntentView, 0)
	for _, i := range rows {
		found := false
		for _, r := range i.Recipients {
			if r.AgentID == agentID {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		retry := uint8(0)
		task, err := taskQueue.GetByID(i.TaskID)
		if err == nil {
			retry = task.RetryCount
		}
		views = append(views, IntentView{IntentID: i.ID, SourceOccurrenceID: i.ID, PostID: i.PostID, Revision: i.Version, Status: i.Status, ErrorCode: i.LastError, RetryCount: retry, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt})
	}
	return IntentPage{List: views, Total: int(total), Page: page, PageSize: pageSize}, nil
}
func ReplayAgentIntent(agentID uint64, intentID string) error {
	i, err := agentEvents.IntentTx(db.Connect(), agentinstance.Current().ID, intentID)
	if err != nil {
		return ErrInaccessible
	}
	found := false
	for _, r := range i.Recipients {
		if r.AgentID == agentID {
			found = true
			break
		}
	}
	if !found {
		return ErrInaccessible
	}
	if i.Status != "failed" && i.Status != "retry_wait" {
		return ErrInaccessible
	}
	return ReplayIntent(intentID)
}

// PreviousPublicTx is the shared durable publication baseline used by approval
// and mention differences. Legacy pending edits can recover a prior normal
// revision; new occurrences never consult the mutable current body or activity.
func PreviousPublicTx(tx *gorm.DB, postID uint64) (postRevisions.Entity, bool, error) {
	cfg := agentinstance.Current()
	if cfg.ID == "" {
		return postRevisions.Entity{}, false, nil
	}
	state, err := agentEvents.PublicationTx(tx, cfg.ID, postID)
	if err == nil {
		rev, err := postRevisions.VersionTx(tx, postID, state.Version)
		return rev, err == nil, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return postRevisions.Entity{}, false, err
	}
	latest, err := postRevisions.LatestTx(tx, postID)
	if err != nil {
		return postRevisions.Entity{}, false, err
	}
	rev, err := postRevisions.PreviousNormalTx(tx, postID, latest.Version)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return rev, false, nil
	}
	return rev, err == nil, err
}

// WithdrawContentTx runs in the lifecycle transaction. Current permissions and
// copies cannot disagree after commit, including accepted webhook payloads.
func WithdrawContentTx(tx *gorm.DB, topicID, postID uint64) error {
	cfg := agentinstance.Current()
	if cfg.ID == "" {
		return nil
	}
	if postID != 0 {
		if _, err := posts.GetUnscopedTx(tx, postID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if _, err := topics.GetUnscopedTx(tx, topicID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	rows, err := agentEvents.ContentEventsTx(tx, cfg.ID, topicID, postID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(rows))
	for _, e := range rows {
		ids = append(ids, e.ID)
		sourcePostID := e.PostID
		if err := agentEvents.WithdrawTx(tx, &e, time.Now().UTC()); err != nil {
			return err
		}
		if err := agentEvents.CancelPostIntentsTx(tx, cfg.ID, sourcePostID); err != nil {
			return err
		}
	}
	postIDs := []uint64{postID}
	if postID == 0 {
		postIDs, err = posts.TopicPostIDsTx(tx, topicID)
		if err != nil {
			return err
		}
	}
	for _, id := range postIDs {
		if err := agentEvents.CancelPostIntentsTx(tx, cfg.ID, id); err != nil {
			return err
		}
	}
	if len(ids) > 0 && withdrawalHook != nil {
		return withdrawalHook(tx, cfg.ID, ids)
	}
	return nil
}

// Batch reads/ACK acquire every source before any Agent row. Otherwise a page
// spanning two topics could lock Agent A, then wait on a producer that already
// owns its next source and is waiting for A. Stable order prevents that cycle.
func lockEventBatchTx(tx *gorm.DB, rows []agentEvents.Entity) error {
	postsSet := map[uint64]bool{}
	topicSet := map[uint64]bool{}
	participants := []uint64{}
	for _, e := range rows {
		if e.WithdrawnAt != nil || time.Now().After(e.ExpiresAt) {
			continue
		}
		postsSet[e.PostID] = true
		topicSet[e.TopicID] = true
		participants = append(participants, e.ActorID, e.AgentID)
	}
	postIDs := make([]uint64, 0, len(postsSet))
	for id := range postsSet {
		postIDs = append(postIDs, id)
	}
	sort.Slice(postIDs, func(i, j int) bool { return postIDs[i] < postIDs[j] })
	for _, id := range postIDs {
		if _, err := posts.GetUnscopedTx(tx, id); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	topicIDs := make([]uint64, 0, len(topicSet))
	for id := range topicSet {
		topicIDs = append(topicIDs, id)
	}
	sort.Slice(topicIDs, func(i, j int) bool { return topicIDs[i] < topicIDs[j] })
	for _, id := range topicIDs {
		if _, err := topics.GetUnscopedTx(tx, id); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return users.LockInteractionUserIDs(tx, participants)
}

func validateCredentialTx(tx *gorm.DB, userID uint64, expectedTokenHash string) error {
	if expectedTokenHash == "" {
		return nil
	}
	if err := users.LockInteractionUserIDs(tx, []uint64{userID}); err != nil {
		return err
	}
	agent, err := agents.GetTx(tx, userID, true)
	if err != nil {
		return ErrInaccessible
	}
	user, err := users.GetInteractionUserTx(tx, userID)
	if err != nil {
		return ErrInaccessible
	}
	if agent.Enabled != agents.StatusEnabled || agent.TokenHash == "" || agent.TokenHash != expectedTokenHash || user.ActorType != users.ActorTypeBot || user.IsFrozen != users.StatusNormal {
		return ErrInaccessible
	}
	return nil
}

// Cleanup keeps identity/sequence tombstones for explicit replay-floor errors,
// removing source identifiers and retained copies in bounded transactions.
func Cleanup() error {
	cfg := agentinstance.Current()
	if cfg.ID == "" {
		return nil
	}
	now := time.Now().UTC()
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		rows, err := agentEvents.ExpiringEventsTx(tx, cfg.ID, now, 500)
		if err != nil {
			return err
		}
		if err := lockEventBatchTx(tx, rows); err != nil {
			return err
		}
		ids := make([]string, 0, len(rows))
		for _, e := range rows {
			if err := agentEvents.WithdrawTx(tx, &e, now); err != nil {
				return err
			}
			ids = append(ids, e.ID)
		}
		if withdrawalHook != nil && len(ids) > 0 {
			if err := withdrawalHook(tx, cfg.ID, ids); err != nil {
				return err
			}
		}
		if err := agentEvents.ExpireIntentsTx(tx, cfg.ID, now, 500); err != nil {
			return err
		}
		return agentEvents.PurgeMetadataTx(tx, cfg.ID, now.Add(-30*24*time.Hour), 500)
	})
}

// WithdrawActorTx uses the account's participant fence and takes no content
// locks. It may run after CloseAccountTx locks the user: senders always acquire
// participant locks before delivery rows, so closure cannot form a lock ring.
func WithdrawActorTx(tx *gorm.DB, userID uint64) error {
	cfg := agentinstance.Current()
	if cfg.ID == "" {
		return nil
	}
	if err := users.LockInteractionUserIDs(tx, []uint64{userID}); err != nil {
		return err
	}
	rows, err := agentEvents.ActorEventsTx(tx, cfg.ID, userID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(rows))
	for _, e := range rows {
		if err := agentEvents.WithdrawTx(tx, &e, time.Now().UTC()); err != nil {
			return err
		}
		ids = append(ids, e.ID)
	}
	if err := agentEvents.CancelActorIntentsTx(tx, cfg.ID, userID); err != nil {
		return err
	}
	if len(ids) > 0 && withdrawalHook != nil {
		return withdrawalHook(tx, cfg.ID, ids)
	}
	return nil
}

// WithdrawAgentTx is called after the caller owns the Agent configuration row.
// It takes no content/user locks, preserving configuration->delivery ordering.
func WithdrawAgentTx(tx *gorm.DB, agentID uint64) error {
	cfg := agentinstance.Current()
	if cfg.ID == "" {
		return nil
	}
	rows, err := agentEvents.AgentEventsTx(tx, cfg.ID, agentID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(rows))
	for _, e := range rows {
		if err := agentEvents.WithdrawTx(tx, &e, time.Now().UTC()); err != nil {
			return err
		}
		ids = append(ids, e.ID)
	}
	if len(ids) > 0 && withdrawalHook != nil {
		return withdrawalHook(tx, cfg.ID, ids)
	}
	return nil
}
