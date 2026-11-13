package publicationservice_test

import (
	"context"
	"fmt"
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
	memberBalanceLocked := make(chan struct{})
	personaProceed := make(chan struct{})
	memberCtx := context.WithValue(ctx, lockOrderContextKey{}, "member")

	if err := conn.Callback().Update().Before("gorm:update").Register("test:points-member-balance-barrier", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Context.Value(lockOrderContextKey{}) != "member" || tx.Statement.Table != "user_points" {
			return
		}
		close(memberBalanceLocked)
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Update().Remove("test:points-member-balance-barrier") })

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
			// Hold the participant lock until member has reached its balance UPDATE.
			// Old implementation: persona then waits for user_points while member
			// waits for users -> PostgreSQL deadlock. Fixed implementation: persona
			// already owns users, so member's participant lock waits and both
			// transactions serialize.
			select {
			case <-memberBalanceLocked:
			case <-ctx.Done():
				return ctx.Err()
			}
			close(personaProceed)
			return pointservice.RewardPublicationTx(tx, 10, 101)
		})
	}()
	select {
	case <-personaOwnerLocked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	go func() {
		results <- conn.WithContext(memberCtx).Transaction(func(tx *gorm.DB) error {
			if _, err := topics.GetForUpdateTx(tx, 20); err != nil {
				return err
			}
			// Hold user_points before RewardPublicationTx on both implementations.
			// The fixed path has already locked users; the old path then waits for
			// persona's users row while persona waits for this balance row.
			result := tx.Table("user_points").Where("user_id = ?", owner.Id).
				Update("current_points", gorm.Expr("current_points + 0"))
			if result.Error != nil {
				return result.Error
			}
			select {
			case <-personaProceed:
			case <-ctx.Done():
				return ctx.Err()
			}
			return pointservice.RewardPublicationTx(tx, 20, 201)
		})
	}()
	select {
	case <-memberBalanceLocked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
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
