// Package agentwriteservice shares Agent write authorization and idempotency
// between REST and MCP. Reservations and resource references commit with content.
package agentwriteservice

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"gorm.io/gorm"
)

const Retention = 7 * 24 * time.Hour

var ErrInvalidKey = errors.New("invalid agent idempotency key")
var ErrInstanceUnconfigured = errors.New("agent write idempotency requires instance identity")
var ErrReplay = errors.New("agent write already committed")

type Options struct {
	Key            string
	SourceEventID  string
	Operation      string
	TargetID       uint64
	Digest         string
	CredentialHash string
}

type contextKey struct{}
type CredentialContextKey struct{}

func WithOptions(ctx context.Context, options Options) context.Context {
	return context.WithValue(ctx, contextKey{}, options)
}
func FromContext(ctx context.Context) (Options, bool) {
	if ctx == nil {
		return Options{}, false
	}
	o, ok := ctx.Value(contextKey{}).(Options)
	return o, ok
}

func RequestDigest(request any) (string, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

func ValidateOptions(o Options) error {
	if o.Key != "" {
		if len(o.Key) > 256 || strings.TrimSpace(o.Key) != o.Key {
			return ErrInvalidKey
		}
		for _, r := range o.Key {
			if r < 33 || r > 126 {
				return ErrInvalidKey
			}
		}
		if agentinstance.Current().ID == "" {
			return ErrInstanceUnconfigured
		}
	}
	if len(o.SourceEventID) > 80 {
		return ErrInvalidKey
	}
	if o.SourceEventID != "" && o.Operation != "post" {
		return agenteventservice.ErrInaccessible
	}
	return nil
}

func entry(agentID uint64, o Options) agentWrites.Entry {
	return agentWrites.Entry{InstanceID: agentinstance.Current().ID, AgentID: agentID, Operation: o.Operation, TargetID: o.TargetID, RequestKey: o.Key, Digest: o.Digest, SourceEventID: o.SourceEventID, ExpiresAt: time.Now().UTC().Add(Retention)}
}

// BeginTx runs before content insertion. Existing target content is locked before
// users and Agent configuration, matching lifecycle and interaction authorization.
func BeginTx(ctx context.Context, tx *gorm.DB, agentID uint64) (reservation, replay *agentWrites.Entry, err error) {
	o, ok := FromContext(ctx)
	if !ok {
		return nil, nil, nil
	}
	if err = ValidateOptions(o); err != nil {
		return nil, nil, err
	}
	if o.SourceEventID != "" {
		if err = agenteventservice.ValidateSourceTx(tx, agentID, o.SourceEventID, o.TargetID); err != nil {
			return nil, nil, err
		}
	} else if o.TargetID > 0 {
		t, readErr := topics.GetUnscopedTx(tx, o.TargetID)
		if readErr != nil || t.DeletedAt.Valid || t.Status != 1 || t.ProcessStatus != topics.ProcessStatusNormal || t.VisibilityStatus != topics.VisibilityActive || t.TopicType != topics.TopicTypeForum {
			return nil, nil, agenteventservice.ErrInaccessible
		}
	}
	// 「Agent 评论策略」在控制器层（REST 与 MCP 共用的 createPost 入口）校验：
	// 全局开关是热配置，其冷加载会另开连接读库，不能在事务内执行
	// （单连接池的 SQLite 部署会自锁）。
	if err = users.LockInteractionUserIDs(tx, []uint64{agentID}); err != nil {
		return nil, nil, err
	}
	u, readErr := users.GetInteractionUserTx(tx, agentID)
	if readErr != nil || !u.IsBot() || u.IsFrozen == users.StatusFrozen {
		return nil, nil, agentservice.ErrAgentTokenInvalid
	}
	a, readErr := agents.LockTx(tx, agentID)
	if readErr != nil || a.Enabled != agents.StatusEnabled || a.TokenHash == "" || (o.CredentialHash != "" && subtle.ConstantTimeCompare([]byte(a.TokenHash), []byte(o.CredentialHash)) != 1) || !agentinstance.Current().APIEnabled {
		return nil, nil, agentservice.ErrAgentTokenInvalid
	}
	if o.Key == "" {
		return nil, nil, nil
	}
	e := entry(agentID, o)
	replay, err = agentWrites.ReserveTx(tx, &e, time.Now().UTC())
	if err != nil {
		return nil, nil, err
	}
	if replay != nil {
		return nil, replay, ErrReplay
	}
	return &e, nil, nil
}

// Lookup performs authorization again for a committed result before expensive
// moderation or daily creation quotas. It never occupies an unused request key.
func Lookup(ctx context.Context, agentID uint64) (*agentWrites.Entry, error) {
	o, ok := FromContext(ctx)
	if !ok || o.Key == "" {
		return nil, nil
	}
	if err := ValidateOptions(o); err != nil {
		return nil, err
	}
	var result *agentWrites.Entry
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		found, err := agentWrites.LookupTx(tx, entry(agentID, o), time.Now().UTC())
		if err != nil || found == nil {
			return err
		}
		_, replay, err := BeginTx(ctx, tx, agentID)
		if errors.Is(err, ErrReplay) {
			result = replay
			return nil
		}
		return err
	})
	return result, err
}

func FinishTx(ctx context.Context, tx *gorm.DB, agentID uint64, reservation *agentWrites.Entry, topicID, postID uint64) error {
	if err := agentWrites.CompleteTx(tx, reservation, topicID, postID); err != nil {
		return err
	}
	if o, ok := FromContext(ctx); ok && o.SourceEventID != "" {
		return agenteventservice.RecordResultTx(tx, agentID, o.SourceEventID, topicID, postID)
	}
	return nil
}
