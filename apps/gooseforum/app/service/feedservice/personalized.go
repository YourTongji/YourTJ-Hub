package feedservice

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

var ErrSnapshotExpired = errors.New("feed_snapshot_expired")
var ErrInvalidCursor = errors.New("invalid feed cursor")
var ErrUnavailable = errors.New("personalized feed unavailable")
var builds = make(chan struct{}, 2)

type snapshot struct {
	Config       feedconfig.Config
	EntryVariant string
	ID           string
	User         uint64
	Hash         string
	Variant      string
	Items        []Candidate
	Created      time.Time
	Expires      time.Time
	Bytes        int
	Seed         uint64
}

var snapshotsMu sync.Mutex
var snapshots = map[string]*snapshot{}
var snapshotsBytes int

type cursor struct {
	Version int    `json:"v"`
	User    uint64 `json:"u"`
	ID      string `json:"i"`
	Offset  int    `json:"o"`
	Hash    string `json:"h"`
	Expires int64  `json:"x"`
}
type Page struct {
	SnapshotID   string
	ContentAt    time.Time
	Config       feedconfig.Config
	EntryVariant string
	Topics       []topics.Entity
	Items        []Candidate
	NextCursor   string
	Trace        string
	Variant      string
	Hash         string
}

func invalidateViewer(uid uint64) {
	snapshotsMu.Lock()
	for id, s := range snapshots {
		if s.User == uid {
			delete(snapshots, id)
			snapshotsBytes -= s.Bytes
		}
	}
	snapshotsMu.Unlock()
	profileMu.Lock()
	delete(profiles, uid)
	profileMu.Unlock()
}
func storeSnapshot(s *snapshot) bool {
	data, err := json.Marshal(s)
	if err != nil {
		return false
	}
	s.Bytes = len(data)
	if s.Bytes > 64<<10 {
		return false
	}
	snapshotsMu.Lock()
	defer snapshotsMu.Unlock()
	now := time.Now()
	for id, old := range snapshots {
		if !old.Expires.After(now) {
			delete(snapshots, id)
			snapshotsBytes -= old.Bytes
		}
	}
	for {
		same := 0
		var oldest *snapshot
		for _, old := range snapshots {
			if old.User == s.User {
				same++
				if oldest == nil || old.Created.Before(oldest.Created) {
					oldest = old
				}
			}
		}
		if same < 2 {
			break
		}
		delete(snapshots, oldest.ID)
		snapshotsBytes -= oldest.Bytes
	}
	for len(snapshots) >= 512 || snapshotsBytes+s.Bytes > 32<<20 {
		var oldest *snapshot
		for _, old := range snapshots {
			if oldest == nil || old.Created.Before(oldest.Created) {
				oldest = old
			}
		}
		if oldest == nil {
			return false
		}
		delete(snapshots, oldest.ID)
		snapshotsBytes -= oldest.Bytes
	}
	snapshots[s.ID] = s
	snapshotsBytes += s.Bytes
	return true
}

func ForYou(ctx context.Context, uid uint64, token string) (Page, error) {
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	cfg := feedconfig.Current()
	if uid == 0 || !cfg.Enabled || !feedconfig.RankReady() {
		return Page{}, ErrUnavailable
	}
	var s *snapshot
	offset := 0
	if token != "" {
		var c cursor
		if verify(token, cursorKey, &c, 512) != nil || c.User != uid || c.Version != 1 || c.Offset < 0 || c.Offset > 120 {
			return Page{}, ErrInvalidCursor
		}
		snapshotsMu.Lock()
		s = snapshots[c.ID]
		snapshotsMu.Unlock()
		if s == nil || s.User != uid || s.Hash != c.Hash || s.Hash != cfg.Hash || !s.Expires.After(time.Now()) || c.Expires != s.Expires.Unix() {
			return Page{}, ErrSnapshotExpired
		}
		offset = c.Offset
	} else {
		select {
		case builds <- struct{}{}:
			defer func() { <-builds }()
		default:
			return Page{}, ErrUnavailable
		}
		var err error
		s, err = buildSnapshot(ctx, uid, cfg)
		if err != nil {
			return Page{}, err
		}
	}

	return snapshotPage(ctx, s, offset)
}

func snapshotPage(ctx context.Context, s *snapshot, offset int) (Page, error) {
	contentAt := time.Now()
	if offset > len(s.Items) {
		return Page{}, ErrInvalidCursor
	}
	ids := []uint64{}
	authors := []uint64{}
	for _, item := range s.Items[offset:] {
		ids = append(ids, item.ID)
		authors = append(authors, item.Author)
	}
	rows, err := topics.RankTopics(ctx, ids)
	if err != nil {
		return Page{}, err
	}
	authors = authors[:0]
	for _, r := range rows {
		authors = append(authors, r.UserId)
	}
	allowed, err := users.FilterEligibleFeedAuthors(ctx, s.User, authors)
	if err != nil {
		return Page{}, err
	}
	excluded, _, err := seenEligibility(ctx, s.User, ids)
	if err != nil {
		return Page{}, err
	}
	present := map[uint64]uint64{}
	for _, row := range rows {
		present[row.Id] = row.UserId
	}
	items := []Candidate{}
	scan := offset
	for scan < len(s.Items) && len(items) < 20 {
		item := s.Items[scan]
		scan++
		if author, exists := present[item.ID]; exists && allowed[author] && author != s.User && !excluded[item.ID] {
			item.Author = author
			items = append(items, item)
		}
	}
	ids = ids[:0]
	for _, c := range items {
		ids = append(ids, c.ID)
	}
	hydrated, err := topics.PublicTopics(ctx, ids)
	if err != nil {
		return Page{}, err
	}
	byID := map[uint64]topics.Entity{}
	for _, r := range hydrated {
		byID[r.Id] = r
	}
	ordered := []topics.Entity{}
	actual := []Candidate{}
	for _, i := range items {
		if r, ok := byID[i.ID]; ok {
			ordered = append(ordered, r)
			actual = append(actual, i)
		}
	}
	page := Page{SnapshotID: s.ID, ContentAt: contentAt, Config: s.Config, EntryVariant: s.EntryVariant, Topics: ordered, Items: actual, Variant: s.Variant, Hash: s.Hash}
	if scan < len(s.Items) {
		page.NextCursor = sign(cursor{1, s.User, s.ID, scan, s.Hash, s.Expires.Unix()}, cursorKey)
	}
	return page, nil
}

type profile struct {
	Categories map[uint64]float64
	Authors    map[uint64]float64
	Created    time.Time
	Bytes      int
	Count      int
}

var profileMu sync.Mutex
var profiles = map[uint64]profile{}

func getProfile(ctx context.Context, uid uint64) (profile, error) {
	profileMu.Lock()
	cached, ok := profiles[uid]
	profileMu.Unlock()
	if ok && time.Since(cached.Created) < 10*time.Minute {
		return cached, nil
	}
	now := time.Now()
	after := now.Add(-60 * 24 * time.Hour)
	type activity struct {
		id     uint64
		at     time.Time
		weight float64
	}
	stream := []activity{}
	for _, kind := range []string{"like", "bookmark"} {
		rows, err := topicUserAction.ProfileActions(ctx, uid, kind, after)
		if err != nil {
			return profile{}, err
		}
		weight := 1.
		if kind == "bookmark" {
			weight = 1.5
		}
		for _, r := range rows {
			stream = append(stream, activity{r.TopicID, r.At, weight})
		}
	}
	replies, err := posts.ProfileReplies(ctx, uid, after)
	if err != nil {
		return profile{}, err
	}
	for _, r := range replies {
		stream = append(stream, activity{r.ID, r.FirstPublicAt, 2})
	}
	sort.Slice(stream, func(i, j int) bool { return stream[i].at.After(stream[j].at) })
	if len(stream) > 100 {
		stream = stream[:100]
	}
	ids := []uint64{}
	for _, a := range stream {
		ids = append(ids, a.id)
	}
	rows, err := topics.RankTopics(ctx, ids)
	if err != nil {
		return profile{}, err
	}
	byID := map[uint64]topics.Entity{}
	for _, r := range rows {
		byID[r.Id] = r
	}
	p := profile{Categories: map[uint64]float64{}, Authors: map[uint64]float64{}, Created: now}
	for _, a := range stream {
		r, ok := byID[a.id]
		if !ok || r.UserId == uid {
			continue
		}
		p.Count++
		weight := a.weight * math.Pow(.5, now.Sub(a.at).Hours()/(14*24))
		if r.PersonaUID == "" {
			p.Authors[r.UserId] += weight
		}
		for _, cat := range r.CategoryIds {
			p.Categories[cat] += weight
		}
	}
	trim := func(m map[uint64]float64, limit int) {
		keys := topAffinity(m, limit)
		kept := map[uint64]bool{}
		maxValue := 0.
		for _, id := range keys {
			kept[id] = true
			maxValue = max(maxValue, m[id])
		}
		for id, v := range m {
			if !kept[id] {
				delete(m, id)
			} else if maxValue > 0 {
				m[id] = v / maxValue
			}
		}
	}
	trim(p.Categories, 5)
	trim(p.Authors, 50)
	if p.Count < 5 {
		p.Categories = map[uint64]float64{}
		p.Authors = map[uint64]float64{}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return p, err
	}
	p.Bytes = len(data)
	profileMu.Lock()
	size := p.Bytes
	for id, old := range profiles {
		if time.Since(old.Created) >= 10*time.Minute {
			delete(profiles, id)
		} else {
			size += old.Bytes
		}
	}
	if size <= 4<<20 {
		profiles[uid] = p
	}
	profileMu.Unlock()
	return p, nil
}
func topAffinity(m map[uint64]float64, limit int) []uint64 {
	ids := make([]uint64, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if m[ids[i]] == m[ids[j]] {
			return ids[i] < ids[j]
		}
		return m[ids[i]] > m[ids[j]]
	})
	return ids[:min(limit, len(ids))]
}

func buildSnapshot(ctx context.Context, uid uint64, cfg feedconfig.Config) (*snapshot, error) {
	p, err := getProfile(ctx, uid)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	sources := map[uint64]uint8{}
	queries := []topics.Recall{{Viewer: uid, Source: "following", After: now.Add(-7 * 24 * time.Hour), Limit: 70}, {Source: "hot", Hash: cfg.RankHash, Limit: 60}, {Source: "daily", Hash: cfg.RankHash, Limit: 30}, {Source: "author", Authors: topAffinity(p.Authors, 50), After: now.Add(-14 * 24 * time.Hour), Limit: 20}, {Source: "explore", After: now.Add(-24 * time.Hour), Limit: 20}}
	for _, q := range queries {
		ids, e := recallPool(ctx, q)
		if e != nil {
			return nil, e
		}
		bit := uint8(1)
		switch q.Source {
		case "hot":
			bit = 2
		case "daily":
			bit = 4
		case "author":
			bit = 16
		case "explore":
			bit = 32
		}
		for _, id := range ids {
			sources[id] |= bit
		}
	}
	for _, cat := range topAffinity(p.Categories, 2) {
		ids, e := recallPool(ctx, topics.Recall{Source: "category", Hash: cfg.RankHash, Category: cat, Limit: 20})
		if e != nil {
			return nil, e
		}
		for _, id := range ids {
			sources[id] |= 8
		}
	}
	if len(sources) > 240 {
		return nil, ErrUnavailable
	}
	ids := []uint64{}
	for id := range sources {
		ids = append(ids, id)
	}
	fallbackIDs, err := topics.RecallRankIDs(ctx, topics.Recall{Source: "latest", After: now.Add(-30 * 24 * time.Hour), Limit: 60})
	if err != nil {
		return nil, err
	}
	for _, id := range fallbackIDs {
		if _, ok := sources[id]; !ok {
			sources[id] = 0
			ids = append(ids, id)
		}
	}
	rows, err := topics.RankTopics(ctx, ids)
	if err != nil {
		return nil, err
	}
	authors := []uint64{}
	ids = ids[:0]
	for _, r := range rows {
		authors = append(authors, r.UserId)
		ids = append(ids, r.Id)
	}
	allowed, err := users.FilterEligibleFeedAuthors(ctx, uid, authors)
	if err != nil {
		return nil, err
	}
	followed, err := userFollow.FollowingAmong(ctx, uid, authors)
	if err != nil {
		return nil, err
	}
	excluded, replies, err := seenEligibility(ctx, uid, ids)
	if err != nil {
		return nil, err
	}
	actions, err := topicUserAction.GetByTopicIDsContext(ctx, uid, ids)
	if err != nil {
		return nil, err
	}
	pool := []Candidate{}
	for _, r := range rows {
		if r.UserId == uid || !allowed[r.UserId] || r.FirstPublicAt == nil || excluded[r.Id] {
			continue
		}
		age := max(0, now.Sub(*r.FirstPublicAt).Hours())
		c := Candidate{Repeat: !replies[r.Id] && wasRepeated(uid, r.Id, r.LastPublicReplyAt, now), ID: r.Id, Author: r.UserId, Sources: sources[r.Id], Reason: "recent", Fallback: sources[r.Id] == 0}
		if r.PersonaUID == "" && followed[r.UserId] {
			c.Features[0] = 10000
			c.Reason = "following"
		}
		c.Features[1] = Quantize(math.Pow(.5, age/12))
		c.Features[2] = Quantize(float64(r.RankScore) / (1000 * cfg.Rules.HotMax()))
		c.Features[3] = Quantize(float64(r.DailyScore) / 100000)
		for _, cat := range r.CategoryIds {
			c.Features[4] = max(c.Features[4], Quantize(p.Categories[cat]))
		}
		if c.Features[4] > 0 && c.Reason == "recent" {
			c.Reason = "category"
		}
		if r.PersonaUID == "" && !followed[r.UserId] {
			c.Features[5] = Quantize(p.Authors[r.UserId])
		}
		if replies[r.Id] {
			c.Features[6] = 10000
			c.Reason = "newreply"
		}
		if age > 7*24 && r.DailyScore == 0 && !replies[r.Id] {
			c.SoftFiltered = true
		}
		state := actions[r.Id]
		if !replies[r.Id] && (state.LikedAt != nil || state.BookmarkedAt != nil) {
			c.SoftFiltered = true
		}
		pool = append(pool, c)
	}
	seedHash := sha256.Sum256([]byte(feed.NewID()))
	seed := binary.BigEndian.Uint64(seedHash[:8]) & math.MaxInt64
	weights := cfg.Weights
	variant := "A"
	if cfg.Comparison && bucket(uid, cfg.Salt+":weights") >= 50 {
		weights = cfg.Alternative
		variant = "B"
	}
	ranked := SelectCandidates(pool, weights, seed)
	if cfg.Interleaving {
		ranked = InterleaveCandidates(pool, cfg.Weights, cfg.Alternative, seed)
		variant = "interleaved"
	}
	_, entryVariant, err := assignDefault(ctx, uid)
	if err != nil {
		return nil, err
	}
	s := &snapshot{Config: cfg, EntryVariant: entryVariant, ID: feed.NewID(), User: uid, Hash: cfg.Hash, Variant: variant, Items: ranked, Created: now, Expires: now.Add(30 * time.Minute), Seed: seed}
	if !storeSnapshot(s) {
		return nil, ErrUnavailable
	}
	captureCandidateSample(uid, s, pool, cfg)
	return s, nil
}

func bucket(uid uint64, salt string) int {
	sum := sha256.Sum256([]byte(salt + ":" + strconv.FormatUint(uid, 10)))
	return int(binary.BigEndian.Uint64(sum[:8]) % 100)
}

// AssignDefault is shared by default and explicit home visits. Assignment is
// independent of the current tab and ranking comparison.
func AssignDefault(ctx context.Context, uid uint64) (bool, string, error) {
	return assignDefault(ctx, uid)
}

var publicPools = boundedMemory{limit: 1 << 20}

func recallPool(ctx context.Context, q topics.Recall) ([]uint64, error) {
	if q.Source != "hot" && q.Source != "daily" && q.Source != "category" {
		return topics.RecallRankIDs(ctx, q)
	}
	// Cache only public ID pools; per-viewer metadata and access checks follow.
	key := memoryKey{Name: fmt.Sprintf("%s:%s:%d:%d:%d", q.Hash, q.Source, q.Category, q.Limit, q.After.UnixNano())}
	now := time.Now()
	if e, ok := publicPools.get(key, now); ok {
		var ids []uint64
		if json.Unmarshal([]byte(e.Trace), &ids) == nil {
			return ids, nil
		}
	}
	ids, err := topics.RecallRankIDs(ctx, q)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	publicPools.put(key, memoryEntry{At: now, Expires: now.Add(30 * time.Second), Trace: string(data)})
	return ids, nil
}

// Fallback is finite and applies exactly the same hard owner filters. It is a
// first page with an explicit latest URL; stale FY cursors never become offsets.
func Fallback(ctx context.Context, uid uint64) (Page, error) {
	ids, err := topics.RecallRankIDs(ctx, topics.Recall{Source: "latest", After: time.Now().Add(-30 * 24 * time.Hour), Limit: 60})
	if err != nil {
		return Page{}, err
	}
	rows, err := topics.RankTopics(ctx, ids)
	if err != nil {
		return Page{}, err
	}
	authors := []uint64{}
	for _, r := range rows {
		authors = append(authors, r.UserId)
	}
	allowed, err := users.FilterEligibleFeedAuthors(ctx, uid, authors)
	if err != nil {
		return Page{}, err
	}
	items := []Candidate{}
	for _, r := range rows {
		if allowed[r.UserId] && r.UserId != uid {
			items = append(items, Candidate{ID: r.Id, Author: r.UserId, Reason: "recent"})
			if len(items) == 20 {
				break
			}
		}
	}
	s := &snapshot{Config: feedconfig.Current(), User: uid, Hash: feedconfig.Current().Hash, Items: items, Variant: "fallback"}
	return snapshotPage(ctx, s, 0)
}
