package notificationservice

import (
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"gorm.io/gorm"
)

// redactAnonymousActors runs after private interaction eligibility checks and
// before persistence. PrivateActorID is never part of any notification DTO.
func redactAnonymousActors(tx *gorm.DB, rows []*eventNotification.Entity) error {
	ids := make([]uint64, 0, len(rows))
	for _, n := range rows {
		if n.Payload.PostId != 0 {
			ids = append(ids, n.Payload.PostId)
		}
	}
	topicIDs := []uint64{}
	for _, n := range rows {
		if n.EventType == eventNotification.EventTypeLike && n.Payload.PostId == 0 {
			topicIDs = append(topicIDs, n.Payload.TopicId)
		}
	}
	var topicRows []topics.Entity
	if len(topicIDs) > 0 {
		if err := tx.Unscoped().Where("id IN ?", topicIDs).Find(&topicRows).Error; err != nil {
			return err
		}
	}
	anonymousTopics := map[uint64]bool{}
	for _, t := range topicRows {
		anonymousTopics[t.Id] = t.PersonaUID != ""
	}
	var postRows []posts.Entity
	if len(ids) > 0 {
		if err := tx.Unscoped().Where("id IN ?", ids).Find(&postRows).Error; err != nil {
			return err
		}
	}
	byID := make(map[uint64]posts.Entity, len(postRows))
	for _, post := range postRows {
		byID[post.Id] = post
	}
	uids := make([]string, 0, len(postRows))
	for _, p := range postRows {
		if p.PersonaUID != "" {
			uids = append(uids, p.PersonaUID)
		}
	}
	personas, err := identity.GetMap(tx, uids)
	if err != nil {
		return err
	}
	for _, n := range rows {
		p, ok := byID[n.Payload.PostId]
		if n.EventType == eventNotification.EventTypeLike && anonymousTopics[n.Payload.TopicId] {
			n.PrivateActorID = n.Payload.ActorId
			n.Payload.ActorId = 0
			n.Payload.ActorName = ""
			continue
		}
		if !ok || p.PersonaUID == "" {
			continue
		}
		n.PrivateActorID = n.Payload.ActorId
		if n.EventType == eventNotification.EventTypeLike {
			n.Payload.ActorId = 0
			n.Payload.ActorName = ""
			continue
		}
		row, exists := personas[p.PersonaUID]
		public := anonymousidentityservice.PublicPersona{Name: "匿名同学", PublicUID: p.PersonaUID, ProfileURL: "/a/" + p.PersonaUID}
		if exists {
			public = anonymousidentityservice.Public(row)
		}
		n.Payload.ActorId = 0
		n.Payload.ActorPersonaUID = p.PersonaUID
		n.Payload.ActorName = public.Name
		n.Payload.Extra.ProfileURL = public.ProfileURL
	}
	return nil
}
