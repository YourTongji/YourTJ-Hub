package publicationservice_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"gorm.io/gorm"
)

// Persona publication locks the owner users row before the reward path. Every
// other points writer must acquire that same participant row before user_points;
// otherwise PostgreSQL can abort one of two concurrent transactions for the same
// owner with a deadlock.
func TestPostgreSQLPublicationPointsUseOneParticipantLockOrder(t *testing.T) {
	conn := setupAgentPolicyPostgreSQL(t)
	conn.TranslateError = true
	owner := users.EntityComplete{Username: fmt.Sprintf("points-lock-%d", time.Now().UnixNano())}
	if err := conn.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&userPoints.Entity{UserId: owner.Id, CurrentPoints: 0}).Error; err != nil {
		t.Fatal(err)
	}
	personaUID := "points-lock-persona"
	rows := []any{
		&identity.Persona{UID: personaUID, Name: "Points lock persona", AvatarSeed: "seed"},
		&identity.Binding{OwnerID: owner.Id, PersonaUID: personaUID},
		// Two published topics let the two transactions hold distinct content locks
		// while contending on the same participant row.
		&topics.Entity{Id: 10, UserId: owner.Id, Status: 1, FirstPostId: 100, PostCount: 2, PostSeq: 2, VisibilityStatus: topics.VisibilityActive, ProcessStatus: topics.ProcessStatusNormal},
		&topics.Entity{Id: 20, UserId: owner.Id, Status: 1, FirstPostId: 200, PostCount: 2, PostSeq: 2, VisibilityStatus: topics.VisibilityActive, ProcessStatus: topics.ProcessStatusNormal},
		&posts.Entity{Id: 100, TopicId: 10, UserId: owner.Id, PostNo: 1, VisibilityStatus: posts.VisibilityActive, ProcessStatus: posts.ProcessStatusNormal},
		&posts.Entity{Id: 200, TopicId: 20, UserId: owner.Id, PostNo: 1, VisibilityStatus: posts.VisibilityActive, ProcessStatus: posts.ProcessStatusNormal},
		&posts.Entity{Id: 201, TopicId: 20, UserId: owner.Id, PostNo: 2, VisibilityStatus: posts.VisibilityActive, ProcessStatus: posts.ProcessStatusNormal, Content: "Concurrent member reply"},
		&posts.Entity{Id: 101, TopicId: 10, UserId: owner.Id, PostNo: 2, PersonaUID: personaUID, IsAnonymous: true, VisibilityStatus: posts.VisibilityActive, ProcessStatus: posts.ProcessStatusNormal, Content: "Concurrent persona reply"},
	}
	for _, row := range rows {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	personaOwnerLocked := make(chan struct{})
	memberAtReward := make(chan struct{})
	var memberOnce sync.Once

	if err := conn.Callback().Query().Before("gorm:query").Register("test:points-member-reward-barrier", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "posts" {
			return
		}
		if tx.Statement.Context.Value(lockOrderContextKey{}) != "member" {
			return
		}
		memberOnce.Do(func() { close(memberAtReward) })
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:points-member-reward-barrier") })

	results := make(chan error, 2)
	go func() {
		results <- conn.Transaction(func(tx *gorm.DB) error {
			if _, err := topics.GetForUpdateTx(tx, 10); err != nil {
				return err
			}
			if err := identity.ValidateWriterTx(tx, owner.Id, personaUID); err != nil {
				return err
			}
			close(personaOwnerLocked)
			select {
			case <-memberAtReward:
			case <-ctx.Done():
				return ctx.Err()
			}
			// Both implementations now enter the reward path together. The fixed
			// implementation gives the member transaction the participant lock
			// before user_points; the old implementation lets member take
			// user_points first and then wait for persona's users row.
			return pointservice.RewardPublicationTx(tx, 10, 101)
		})
	}()
	select {
	case <-personaOwnerLocked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	go func() {
		results <- conn.WithContext(context.WithValue(ctx, lockOrderContextKey{}, "member")).Transaction(func(tx *gorm.DB) error {
			if _, err := topics.GetForUpdateTx(tx, 20); err != nil {
				return err
			}
			return pointservice.RewardPublicationTx(tx, 20, 201)
		})
	}()
	for i := 0; i < 2; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("publication/points lock order: %v", err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	var user users.EntityComplete
	if err := conn.First(&user, owner.Id).Error; err != nil {
		t.Fatal(err)
	}
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", owner.Id).Take(&balance).Error; err != nil {
		t.Fatal(err)
	}
	if user.Prestige != 2*pointservice.PostCreatedReward || balance.CurrentPoints != 2*pointservice.PostCreatedReward {
		t.Fatalf("prestige=%d balance=%d, want %d", user.Prestige, balance.CurrentPoints, 2*pointservice.PostCreatedReward)
	}
}

type lockOrderContextKey struct{}
