package users

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

const (
	blockedPeersCacheViewer uint64 = 880031
	blockedPeersCachePeer   uint64 = 880032
)

func seedBlockedPeersCacheUser(t *testing.T, id uint64) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&EntityComplete{}, &BlockEntity{}); err != nil {
		t.Fatalf("migrate blocked peers cache tables: %v", err)
	}
	if err := conn.Create(&EntityComplete{Id: id, Username: fmt.Sprintf("u%d", id)}).Error; err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
	t.Cleanup(func() {
		conn.Delete(&BlockEntity{}, "owner_id = ? OR target_user_id = ?", id, id)
		conn.Unscoped().Delete(&EntityComplete{}, id)
	})
}

// TestBlockedPeersCacheServesWithinTTLAndInvalidatesOnWrite 覆盖 issue #1129 的
// 语义验收：读者级屏蔽存在性结果按短 TTL 内存缓存（窗口内绕过写路径的库内
// 变更不可见，用于证明缓存确实生效），而屏蔽新增/解除经写路径立即双向失效，
// 保持 MADR 0079「屏蔽变更立即生效」的承诺。
func TestBlockedPeersCacheServesWithinTTLAndInvalidatesOnWrite(t *testing.T) {
	conn := dbconnect.Connect()
	seedBlockedPeersCacheUser(t, blockedPeersCacheViewer)
	seedBlockedPeersCacheUser(t, blockedPeersCachePeer)
	conn.Delete(&BlockEntity{}, "owner_id = ? AND target_user_id = ?", blockedPeersCacheViewer, blockedPeersCachePeer)
	conn.Delete(&BlockEntity{}, "owner_id = ? AND target_user_id = ?", blockedPeersCachePeer, blockedPeersCacheViewer)

	read := func(userID uint64) bool {
		t.Helper()
		has, err := HasBlockedPeersCached(userID)
		if err != nil {
			t.Fatalf("HasBlockedPeersCached(%d): %v", userID, err)
		}
		return has
	}

	// 无关系时首次查询写入缓存。
	InvalidateBlockedPeersCache(blockedPeersCacheViewer, blockedPeersCachePeer)
	if read(blockedPeersCacheViewer) || read(blockedPeersCachePeer) {
		t.Fatal("users without block relations must read false")
	}

	// TTL 窗口内缓存生效：直接写库（绕过写路径）在这个窗口内不可见。
	row := BlockEntity{OwnerID: blockedPeersCacheViewer, TargetUserID: blockedPeersCachePeer, CreatedAt: time.Now()}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatalf("seed direct block row: %v", err)
	}
	if read(blockedPeersCacheViewer) {
		t.Fatal("cached false must be served within the TTL window")
	}

	// 写路径解除必须同时失效双方缓存，随后读回真实状态。
	if err := SetBlockedUser(blockedPeersCacheViewer, blockedPeersCachePeer, false); err != nil {
		t.Fatalf("clear direct block row: %v", err)
	}
	if read(blockedPeersCacheViewer) || read(blockedPeersCachePeer) {
		t.Fatal("clear through the write path must invalidate both cached readers")
	}

	// 写路径新增必须同时失效双方缓存：owner 与 target 都立即读到 true，
	// 任一方向漏失效都会让另一方继续拿到缓存 false。
	if err := SetBlockedUser(blockedPeersCacheViewer, blockedPeersCachePeer, true); err != nil {
		t.Fatalf("block through the write path: %v", err)
	}
	if !read(blockedPeersCacheViewer) || !read(blockedPeersCachePeer) {
		t.Fatal("block through the write path must invalidate both cached readers")
	}

	// 再解除一次，验证反向。
	if err := SetBlockedUser(blockedPeersCacheViewer, blockedPeersCachePeer, false); err != nil {
		t.Fatalf("unblock through the write path: %v", err)
	}
	if read(blockedPeersCacheViewer) || read(blockedPeersCachePeer) {
		t.Fatal("unblock through the write path must invalidate both cached readers")
	}
}

func TestBlockedPeersCachedGuestShortCircuits(t *testing.T) {
	if has, err := HasBlockedPeersCached(0); err != nil || has {
		t.Fatalf("guest has=%v err=%v, want false/nil", has, err)
	}
}
