package httpnotifyservice

import (
	"slices"
	"sync"
	"time"
)

// 审批投递去重（issue #1049）：进程内记忆“某 Endpoint 已投递某审批”，同一审批在
// dedupeTTL 内不重复提醒。失败时释放名额，由现有任务队列有限重试；进程重启会清空
// 去重记忆，最坏情况是重复一次提醒。
const (
	dedupeTTL        = 24 * time.Hour
	maxDedupeEntries = 4096
)

var recentDeliveries = &deliveryDeduper{seen: map[string]time.Time{}}

type deliveryDeduper struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

// claim 首次见到 key（或已过期）时登记并返回 true；TTL 内重复返回 false。
func (d *deliveryDeduper) claim(key string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if at, ok := d.seen[key]; ok && now.Sub(at) < dedupeTTL {
		return false
	}
	if len(d.seen) >= maxDedupeEntries {
		d.prune(now)
	}
	d.seen[key] = now
	return true
}

// release 撤销一次失败投递占用的名额；只删除 at 时刻登记的那一项，不会误删之后的登记。
func (d *deliveryDeduper) release(key string, at time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if seen, ok := d.seen[key]; ok && seen.Equal(at) {
		delete(d.seen, key)
	}
}

// prune 清理过期项；仍超上限时按登记时间淘汰最早的一半（按条数而非时间中点，
// 登记时间扎堆时也只淘汰一半，保证内存有界且去重不会一次失效）。
func (d *deliveryDeduper) prune(now time.Time) {
	for key, at := range d.seen {
		if now.Sub(at) >= dedupeTTL {
			delete(d.seen, key)
		}
	}
	if len(d.seen) < maxDedupeEntries {
		return
	}
	keys := make([]string, 0, len(d.seen))
	for key := range d.seen {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int { return d.seen[a].Compare(d.seen[b]) })
	for _, key := range keys[:len(keys)/2] {
		delete(d.seen, key)
	}
}
