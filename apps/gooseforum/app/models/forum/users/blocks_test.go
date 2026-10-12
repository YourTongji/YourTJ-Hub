package users

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

func blockQueryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&BlockEntity{}); err != nil {
		t.Fatalf("migrate user_blocks: %v", err)
	}
	return conn
}

func seedBlockRelation(t *testing.T, conn *gorm.DB, owner, target uint64) {
	t.Helper()
	row := BlockEntity{OwnerID: owner, TargetUserID: target, CreatedAt: time.Now()}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatalf("create block %d->%d: %v", owner, target, err)
	}
	t.Cleanup(func() { conn.Delete(&BlockEntity{}, "owner_id = ? AND target_user_id = ?", owner, target) })
}

func TestBlockedPeerIDsQueryCoversBothDirections(t *testing.T) {
	conn := blockQueryTestDB(t)
	const viewer, blocked, blocker, stranger uint64 = 880001, 880002, 880003, 880004
	seedBlockRelation(t, conn, viewer, blocked) // 观看者拉黑对方
	seedBlockRelation(t, conn, blocker, viewer) // 对方拉黑观看者

	var peers []uint64
	if err := BlockedPeerIDsQuery(viewer).Find(&peers).Error; err != nil {
		t.Fatalf("query blocked peers: %v", err)
	}
	seen := map[uint64]bool{}
	for _, id := range peers {
		seen[id] = true
	}
	if len(peers) != 2 || !seen[blocked] || !seen[blocker] || seen[stranger] {
		t.Fatalf("peers=%v, want exactly both directional relations", peers)
	}
}

func TestHasBlockedPeersDetectsEitherDirection(t *testing.T) {
	conn := blockQueryTestDB(t)
	const viewer, blocked, blocker, stranger uint64 = 880011, 880012, 880013, 880014
	seedBlockRelation(t, conn, viewer, blocked)
	seedBlockRelation(t, conn, blocker, viewer)

	cases := []struct {
		name string
		uid  uint64
		want bool
	}{
		{"owner", viewer, true},
		{"target", blocked, true},
		{"reverse target", blocker, true},
		{"unrelated", stranger, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			has, err := HasBlockedPeers(test.uid)
			if err != nil {
				t.Fatalf("HasBlockedPeers(%d): %v", test.uid, err)
			}
			if has != test.want {
				t.Fatalf("HasBlockedPeers(%d)=%v, want %v", test.uid, has, test.want)
			}
		})
	}
}
