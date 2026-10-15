package botcontentservice

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrUnavailable = errors.New("bot content unavailable")
var ErrConflict = errors.New("bot content version conflict")

// This private write context never becomes a credential or a public author capability.
type Maintenance struct {
	ActorID            uint64
	Post               posts.Entity
	Topic              topics.Entity
	ExpectedRevisionNo uint64
	CategoryIDs        []uint64
	Capture            *agenteventservice.WriteCapturePlan
	SourceRevision     *postRevisions.Entity
}

type SourceRequest struct {
	TopicId uint64 `json:"topicId"`
	PostId  uint64 `json:"postId"`
}

func Load(actorID uint64, source SourceRequest) (*Maintenance, error) {
	if (source.PostId == 0) == (source.TopicId == 0) {
		return nil, ErrUnavailable
	}
	m := &Maintenance{ActorID: actorID}
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"})
		if source.TopicId != 0 {
			query = query.Where("topic_id = ? AND post_no = 1", source.TopicId)
		} else {
			query = query.Where("id = ?", source.PostId)
		}
		if err := query.First(&m.Post).Error; err != nil {
			return ErrUnavailable
		}
		var err error
		m.Topic, err = topics.GetForUpdateTx(tx, m.Post.TopicId)
		if err != nil {
			return ErrUnavailable
		}
		if err := m.authorizeTx(tx); err != nil {
			return err
		}
		r, err := postRevisions.LatestTx(tx, m.Post.Id)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		m.ExpectedRevisionNo = r.Version
		if m.Post.LatestRevisionId != 0 {
			if r.Id != m.Post.LatestRevisionId {
				return ErrConflict
			}
			m.SourceRevision = &r
		}
		return nil
	})
	return m, err
}

func (m *Maintenance) authorizeTx(tx *gorm.DB) error {
	if err := users.LockInteractionUserIDs(tx, []uint64{m.ActorID, m.Post.UserId}); err != nil {
		return err
	}
	if _, err := agents.LockTx(tx, m.Post.UserId); err != nil {
		return ErrUnavailable
	}
	if !agentservice.CanManageTx(tx, m.ActorID, m.Post.UserId) {
		return ErrUnavailable
	}
	return m.checkAvailability(tx)
}

func (m *Maintenance) checkAvailability(tx *gorm.DB) error {
	bot, err := users.GetInteractionUserTx(tx, m.Post.UserId)
	if err != nil || !bot.IsBot() || bot.IsFrozen != users.StatusNormal {
		return ErrUnavailable
	}
	a := agents.GetByUserIDWithDB(tx, bot.Id)
	if a == nil || a.Enabled != agents.StatusEnabled {
		return ErrUnavailable
	}
	if m.Post.Id == 0 || m.Topic.Id == 0 || m.Post.DeletedAt.Valid || m.Topic.DeletedAt.Valid ||
		m.Post.VisibilityStatus != posts.VisibilityActive || m.Topic.VisibilityStatus != topics.VisibilityActive ||
		m.Post.ProcessStatus != posts.ProcessStatusNormal || m.Topic.ProcessStatus != topics.ProcessStatusNormal || m.Topic.Status != 1 ||
		m.Post.IsAnonymous || m.Post.PersonaUID != "" || m.Topic.PersonaUID != "" || m.Topic.TopicType != topics.TopicTypeForum ||
		(m.Post.PostNo == 1 && m.Post.UserId != m.Topic.UserId) {
		return ErrUnavailable
	}
	return nil
}

// Lock order matches Agent writes: content, topic, users, then Agent configuration.
func (m *Maintenance) CheckTx(tx *gorm.DB) error {
	var livePost posts.Entity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&livePost, m.Post.Id).Error; err != nil {
		return ErrUnavailable
	}
	liveTopic, err := topics.GetForUpdateTx(tx, livePost.TopicId)
	if err != nil {
		return ErrUnavailable
	}
	oldPost, oldTopic := m.Post, m.Topic
	m.Post, m.Topic = livePost, liveTopic
	m.Capture, err = agenteventservice.PrepareWriteCaptureTx(tx, m.Post.UserId, m.Topic.Id, "", m.ActorID)
	if err != nil {
		return err
	}
	if err := m.authorizeTx(tx); err != nil {
		return err
	}
	r, err := postRevisions.LatestTx(tx, livePost.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if r.Version != m.ExpectedRevisionNo || livePost.Content != oldPost.Content || (livePost.PostNo == 1 && (liveTopic.Title != oldTopic.Title || !slices.Equal(liveTopic.CategoryIds, oldTopic.CategoryIds))) || livePost.UserId != oldPost.UserId {
		return ErrConflict
	}
	if m.CategoryIDs != nil {
		var count int64
		if err := tx.Model(&category.Entity{}).Where("id IN ?", m.CategoryIDs).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(m.CategoryIDs)) {
			return ErrUnavailable
		}
	}
	return nil
}

func (m *Maintenance) AuditTx(tx *gorm.DB) error {
	revision, err := postRevisions.LatestTx(tx, m.Post.Id)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(optlogger.MessagePayload{MessageCode: "admin.opt.agent.contentEdited", Params: optlogger.MessageParams{
		"agentId": m.Post.UserId, "topicId": m.Topic.Id, "postId": m.Post.Id, "revisionNo": revision.Version,
	}})
	if err != nil {
		return err
	}
	return tx.Create(&optRecord.Entity{OptUserId: m.ActorID, OptType: int(optlogger.EditTopic), TargetType: int(optlogger.Topic), TargetId: strconv.FormatUint(m.Topic.Id, 10), OptInfo: string(payload), CreatedAt: time.Now()}).Error
}

// Author and operator eligibility is rechecked when a delayed maintenance revision is published.
func CheckPublicationTx(tx *gorm.DB, operatorID uint64, post posts.Entity, topic topics.Entity) (*agenteventservice.WriteCapturePlan, error) {
	m := Maintenance{ActorID: operatorID, Post: post, Topic: topic}
	plan, err := agenteventservice.PrepareWriteCaptureTx(tx, post.UserId, topic.Id, "", operatorID)
	if err != nil {
		return nil, err
	}
	if err := m.authorizeTx(tx); err != nil {
		return nil, err
	}
	return plan, nil
}
