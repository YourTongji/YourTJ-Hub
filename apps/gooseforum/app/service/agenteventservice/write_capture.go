package agenteventservice

import (
	"errors"
	"sort"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// WriteCapturePlan freezes the candidate set before an Agent write acquires any
// participant locks. Capture must not discover new participants after BeginTx
// has acquired the writer's credential lock. The current generation/eligibility
// is still checked under these locks when the content is captured.
type WriteCapturePlan struct {
	Subscribers []agents.Entity
	Depth       int
}

// PrepareWriteCaptureTx follows the same content -> users -> Agents lock order
// as public capture, lifecycle changes and source-event authorization. All
// candidate users and Agent rows are locked in ascending order exactly once;
// subsequent authorization and capture only reacquire members of that set.
func PrepareWriteCaptureTx(tx *gorm.DB, actorID, topicID uint64, sourceEventID string) (*WriteCapturePlan, error) {
	cfg := agentinstance.Current()
	plan := &WriteCapturePlan{}
	ids := []uint64{actorID}
	if sourceEventID != "" {
		e, err := agentEvents.EventTx(tx, cfg.ID, actorID, sourceEventID)
		if err != nil || e.TopicID != topicID {
			return nil, ErrInaccessible
		}
		p, _, err := PublicSourceTx(tx, e.PostID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, e.ActorID)
		// Saturation preserves the suppression decision without growing an integer
		// indefinitely when a runner continues writing beyond the broadcast cap.
		plan.Depth = min(p.AgentEventDepth+1, MaxBroadcastDepth+1)
	} else if topicID > 0 {
		if _, err := topics.GetUnscopedTx(tx, topicID); err != nil {
			return nil, err
		}
	}
	if cfg.ID != "" && cfg.Epoch != "" && cfg.ProducerEnabled {
		var err error
		plan.Subscribers, err = agents.ListEnabledTx(tx)
		if err != nil {
			return nil, err
		}
	}
	agentIDs := []uint64{actorID}
	for _, candidate := range plan.Subscribers {
		ids = append(ids, candidate.UserId)
		agentIDs = append(agentIDs, candidate.UserId)
	}
	if err := users.LockInteractionUserIDs(tx, ids); err != nil {
		return nil, err
	}
	sort.Slice(agentIDs, func(i, j int) bool { return agentIDs[i] < agentIDs[j] })
	for n, id := range agentIDs {
		if n > 0 && id == agentIDs[n-1] {
			continue
		}
		if _, err := agents.GetTx(tx, id, true); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return plan, nil
}
