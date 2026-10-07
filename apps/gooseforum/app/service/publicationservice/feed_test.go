package publicationservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestDelayedPublicationCannotRecreateClosedAuthorAfterRawFenceExpiry(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(append(feed.Models(), &users.EntityComplete{})...); err != nil {
		t.Fatal(err)
	}
	preferences.Set("feed.metrics.enabled", true)
	t.Cleanup(func() { preferences.Set("feed.metrics.enabled", false) })
	const closed, active uint64 = 957801, 957802
	rows := []users.EntityComplete{{Id: closed, Username: "closed-feed-author"}, {Id: active, Username: "active-feed-author"}}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Delete(&rows[0]).Error; err != nil {
		t.Fatal(err)
	}
	// Physical cleanup has removed the expired 30-day fence. The identity owner
	// still retains the authoritative account-closed lifecycle state.
	fence := feed.Owner{UserID: closed, Closed: true, ExpiresAt: time.Now().Add(-time.Hour)}
	if err := conn.Create(&fence).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Delete(&fence).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", []uint64{closed, active}).Delete(&users.EntityComplete{})
		conn.Where("user_id IN ?", []uint64{closed, active}).Delete(&feed.Owner{})
		conn.Where("user_id IN ?", []uint64{closed, active}).Delete(&feed.Event{})
	})
	for _, uid := range []uint64{closed, active} {
		if err := conn.Transaction(func(tx *gorm.DB) error { return capturePublicContributionTx(tx, uid, 1, uid, "public_reply") }); err != nil {
			t.Fatal(err)
		}
	}
	for _, model := range []any{&feed.Owner{}, &feed.Event{}} {
		var n int64
		if err := conn.Model(model).Where("user_id = ?", closed).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("recreated closed author's %T after fence expiry", model)
		}
	}
	var n int64
	if err := conn.Model(&feed.Event{}).Where("user_id = ?", active).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("eligible author publication was not captured")
	}
}
