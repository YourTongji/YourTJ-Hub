package agenteventservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

const directedEventTypes = `["agent.mentioned","agent.post_replied","agent.topic_commented"]`
const broadcastEventTypes = `["agent.mentioned","agent.post_replied","agent.topic_commented","forum.topic_created","forum.post_created"]`

type broadcastFixture struct {
	t        *testing.T
	conn     *gorm.DB
	base     uint64
	human    users.EntityComplete
	botA     users.EntityComplete // Agent subscribed to the forum-wide types
	botB     users.EntityComplete // Agent subscribed only to directed interactions
	botC     users.EntityComplete // second forum-wide subscriber
	topic    topics.Entity
	first    posts.Entity
	nextPost uint64
	topics   []topics.Entity
	posts    []posts.Entity
}

func broadcastSetup(t *testing.T) *broadcastFixture {
	t.Helper()
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "test-instance")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch-1")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &agents.Entity{}, &topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &taskQueue.Entity{}, &agentEvents.ReplayState{}, &agentEvents.Publication{}, &agentEvents.Intent{}, &agentEvents.Entity{}); err != nil {
		t.Fatal(err)
	}
	base := uint64(time.Now().UnixNano()%100000000) + 800000000
	human := users.EntityComplete{Id: base, Username: fmt.Sprintf("bhuman%d", base), Email: fmt.Sprintf("bh%d@invalid.example", base), ActorType: users.ActorTypeHuman}
	botA := users.EntityComplete{Id: base + 1, Username: fmt.Sprintf("bbota%d", base), Email: fmt.Sprintf("ba%d@invalid.example", base), ActorType: users.ActorTypeBot}
	botB := users.EntityComplete{Id: base + 2, Username: fmt.Sprintf("bbotb%d", base), Email: fmt.Sprintf("bb%d@invalid.example", base), ActorType: users.ActorTypeBot}
	botC := users.EntityComplete{Id: base + 3, Username: fmt.Sprintf("bbotc%d", base), Email: fmt.Sprintf("bc%d@invalid.example", base), ActorType: users.ActorTypeBot}
	now := time.Now().Add(-time.Hour)
	agentA := agents.Entity{UserId: botA.Id, TokenPrefix: fmt.Sprintf("agtA%d", base), Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, SubscriptionGeneration: 1, EventTypes: broadcastEventTypes}
	agentB := agents.Entity{UserId: botB.Id, TokenPrefix: fmt.Sprintf("agtB%d", base), Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, SubscriptionGeneration: 1, EventTypes: directedEventTypes}
	agentC := agents.Entity{UserId: botC.Id, TokenPrefix: fmt.Sprintf("agtC%d", base), Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, SubscriptionGeneration: 1, EventTypes: broadcastEventTypes}
	topic := topics.Entity{Id: base + 10, UserId: human.Id, Status: 1, PostSeq: 1}
	first := posts.Entity{Id: base + 11, TopicId: topic.Id, PostNo: 1, UserId: human.Id, Content: "topic first post"}
	topic.FirstPostId = first.Id
	rows := []any{&human, &botA, &botB, &botC, &agentA, &agentB, &agentC, &topic, &first,
		&postRevisions.Entity{PostId: first.Id, Version: 1, EditorId: human.Id, Content: first.Content}}
	for _, row := range rows {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	f := &broadcastFixture{t: t, conn: conn, base: base, human: human, botA: botA, botB: botB, botC: botC, topic: topic, first: first, nextPost: base + 20}
	t.Cleanup(func() {
		conn.Where("instance_id = ?", "test-instance").Delete(&agentEvents.Entity{})
		conn.Where("instance_id = ?", "test-instance").Delete(&agentEvents.Intent{})
		conn.Where("instance_id = ?", "test-instance").Delete(&agentEvents.ReplayState{}, &agentEvents.Publication{})
		conn.Where("type = ?", TaskType).Delete(&taskQueue.Entity{})
		for _, post := range append(f.posts, first) {
			conn.Where("post_id = ?", post.Id).Delete(&postRevisions.Entity{})
		}
		conn.Unscoped().Where("topic_id IN ?", f.topicIDs()).Delete(&posts.Entity{})
		conn.Unscoped().Where("id IN ?", f.topicIDs()).Delete(&topics.Entity{})
		conn.Where("user_id IN ?", []uint64{botA.Id, botB.Id, botC.Id}).Delete(&agents.Entity{})
		conn.Unscoped().Where("id IN ?", []uint64{human.Id, botA.Id, botB.Id, botC.Id}).Delete(&users.EntityComplete{})
	})
	return f
}

func (f *broadcastFixture) topicIDs() []uint64 {
	ids := []uint64{f.topic.Id}
	for _, topic := range f.topics {
		ids = append(ids, topic.Id)
	}
	return ids
}

func (f *broadcastFixture) addTopic(authorID uint64) topics.Entity {
	f.t.Helper()
	f.base += 100
	topic := topics.Entity{Id: f.base, UserId: authorID, Status: 1}
	if err := f.conn.Create(&topic).Error; err != nil {
		f.t.Fatal(err)
	}
	f.topics = append(f.topics, topic)
	return topic
}

func (f *broadcastFixture) addPost(topic topics.Entity, authorID, postNo, replyTo uint64, content string) posts.Entity {
	f.t.Helper()
	f.nextPost++
	post := posts.Entity{Id: f.nextPost, TopicId: topic.Id, PostNo: postNo, UserId: authorID, ReplyToPostId: replyTo, Content: content}
	if err := f.conn.Create(&post).Error; err != nil {
		f.t.Fatal(err)
	}
	if err := f.conn.Create(&postRevisions.Entity{PostId: post.Id, Version: 1, EditorId: authorID, Content: content}).Error; err != nil {
		f.t.Fatal(err)
	}
	if postNo == 1 {
		if err := f.conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Updates(map[string]any{"first_post_id": post.Id, "post_seq": 1}).Error; err != nil {
			f.t.Fatal(err)
		}
	}
	f.posts = append(f.posts, post)
	return post
}

// capture publishes the post and materializes every recipient event.
func (f *broadcastFixture) capture(post posts.Entity) (bool, map[uint64]agentEvents.Entity) {
	f.t.Helper()
	if err := f.conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &post) }); err != nil {
		f.t.Fatal(err)
	}
	var intent agentEvents.Intent
	err := f.conn.Where("post_id = ?", post.Id).Take(&intent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
	if err != nil || !claimed {
		f.t.Fatalf("claim %t %v", claimed, err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		f.t.Fatal(err)
	}
	var rows []agentEvents.Entity
	if err := f.conn.Where("source_intent_id = ?", intent.ID).Find(&rows).Error; err != nil {
		f.t.Fatal(err)
	}
	events := map[uint64]agentEvents.Entity{}
	for _, row := range rows {
		events[row.AgentID] = row
	}
	return true, events
}

// linkResult emulates the accepted Agent write ledger recording that the event
// produced the given post.
func (f *broadcastFixture) linkResult(eventID string, postID uint64) {
	f.t.Helper()
	var event agentEvents.Entity
	if err := f.conn.Where("id = ?", eventID).Take(&event).Error; err != nil {
		f.t.Fatal(err)
	}
	var source posts.Entity
	if err := f.conn.Where("id = ?", event.PostID).Take(&source).Error; err != nil {
		f.t.Fatal(err)
	}
	if err := posts.SetAgentEventDepthTx(f.conn, postID, min(source.AgentEventDepth+1, MaxBroadcastDepth+1)); err != nil {
		f.t.Fatal(err)
	}
	if err := f.conn.Model(&agentEvents.Entity{}).Where("id = ?", eventID).Update("resulting_post_id", postID).Error; err != nil {
		f.t.Fatal(err)
	}
}

func TestBroadcastForwardsForumEventsToSubscribedAgents(t *testing.T) {
	f := broadcastSetup(t)
	_, events := f.capture(f.first)
	topicEvent := events[f.botA.Id]
	if topicEvent.Type != "forum.topic_created" || len(topicEvent.Reasons) != 1 || topicEvent.Reasons[0] != "topic_created" {
		t.Fatalf("topic event %#v", topicEvent)
	}
	if topicEvent.ActorType != "human" {
		t.Fatalf("human actor type %q", topicEvent.ActorType)
	}
	if _, ok := events[f.botB.Id]; ok {
		t.Fatal("directed-only subscriber received a broadcast event")
	}
	if events[f.botC.Id].ID == "" {
		t.Fatal("second broadcast subscriber missing topic event")
	}

	reply := f.addPost(f.topic, f.human.Id, 2, f.first.Id, "plain human reply")
	if _, events = f.capture(reply); events[f.botA.Id].Type != "forum.post_created" {
		t.Fatalf("reply event %#v", events[f.botA.Id])
	}

	agentPost := f.addPost(f.topic, f.botA.Id, 3, 0, "agent reply without an event source")
	_, events = f.capture(agentPost)
	if events[f.botA.Id].ID != "" {
		t.Fatal("author received its own broadcast event")
	}
	agentEvent := events[f.botC.Id]
	if agentEvent.Type != "forum.post_created" || agentEvent.ActorType != "bot" {
		t.Fatalf("agent content event %#v", agentEvent)
	}
	if envelope := Envelope(agentEvent); envelope.Data == nil || envelope.Data.ActorType != "bot" {
		t.Fatalf("envelope %#v", envelope)
	}
}

func TestBroadcastDepthCapStopsEventDrivenChains(t *testing.T) {
	f := broadcastSetup(t)
	_, events := f.capture(f.first)
	parent := events[f.botA.Id]
	if parent.ID == "" {
		t.Fatal("missing root broadcast event")
	}
	for hop := 1; hop <= MaxBroadcastDepth+1; hop++ {
		topic := f.addTopic(f.botA.Id)
		post := f.addPost(topic, f.botA.Id, 1, 0, fmt.Sprintf("agent hop %d", hop))
		f.linkResult(parent.ID, post.Id)
		_, produced := f.capture(post)
		event := produced[f.botC.Id]
		if hop <= MaxBroadcastDepth {
			if event.ID == "" {
				t.Fatalf("hop %d within the cap was not broadcast", hop)
			}
			parent = event
			continue
		}
		if event.ID != "" {
			t.Fatalf("hop %d beyond the cap was broadcast: %#v", hop, event)
		}
	}
}

func TestBroadcastSuppressesConsecutiveBotPosts(t *testing.T) {
	f := broadcastSetup(t)
	f.capture(f.first)
	for postNo := uint64(2); postNo <= uint64(MaxConsecutiveBotPosts+1); postNo++ {
		post := f.addPost(f.topic, f.botA.Id, postNo, 0, fmt.Sprintf("agent post %d", postNo))
		_, events := f.capture(post)
		observed := events[f.botC.Id]
		if postNo <= uint64(MaxConsecutiveBotPosts) {
			if observed.ID == "" {
				t.Fatalf("post %d in a mixed tail was suppressed", postNo)
			}
			continue
		}
		if observed.ID != "" {
			t.Fatalf("post %d extended a consecutive Agent run but was broadcast", postNo)
		}
	}
}

func TestBroadcastMergesWithDirectedReason(t *testing.T) {
	f := broadcastSetup(t)
	reply := f.addPost(f.topic, f.human.Id, 2, f.first.Id, "hi @"+f.botA.Username)
	_, events := f.capture(reply)
	merged := events[f.botA.Id]
	if merged.Type != "agent.mentioned" {
		t.Fatalf("directed type did not win: %#v", merged)
	}
	if len(merged.Reasons) != 2 || merged.Reasons[0] != "mention" || merged.Reasons[1] != "post_created" {
		t.Fatalf("merged reasons %#v", merged.Reasons)
	}
	if nonMentioned := events[f.botC.Id]; nonMentioned.Type != "forum.post_created" {
		t.Fatalf("plain subscriber %#v", nonMentioned)
	}
}

func TestBroadcastBacklogUsesSourceTail(t *testing.T) {
	f := broadcastSetup(t)
	var firstBot posts.Entity
	for n := uint64(2); n <= 6; n++ {
		p := f.addPost(f.topic, f.botA.Id, n, 0, "queued bot reply")
		if n == 2 {
			firstBot = p
		}
	}
	_, events := f.capture(firstBot)
	if events[f.botC.Id].ID == "" {
		t.Fatal("later bot posts suppressed the first bot publication")
	}
}

func TestBroadcastHiddenHumanDoesNotBreakBotRun(t *testing.T) {
	f := broadcastSetup(t)
	for n := uint64(2); n <= 5; n++ {
		f.addPost(f.topic, f.botA.Id, n, 0, "bot reply")
	}
	hidden := f.addPost(f.topic, f.human.Id, 6, 0, "pending human reply")
	if err := f.conn.Model(&hidden).Update("process_status", posts.ProcessStatusPending).Error; err != nil {
		t.Fatal(err)
	}
	last := f.addPost(f.topic, f.botA.Id, 7, 0, "fifth public bot reply")
	_, events := f.capture(last)
	if events[f.botC.Id].ID != "" {
		t.Fatal("non-public human post reset the broadcast loop bound")
	}
}

func TestBroadcastDepthSurvivesSourceResultReplacement(t *testing.T) {
	f := broadcastSetup(t)
	_, events := f.capture(f.first)
	parent := events[f.botA.Id]
	for hop := 1; hop <= MaxBroadcastDepth; hop++ {
		topic := f.addTopic(f.botA.Id)
		post := f.addPost(topic, f.botA.Id, 1, 0, "source-linked hop")
		f.linkResult(parent.ID, post.Id)
		_, produced := f.capture(post)
		parent = produced[f.botC.Id]
	}
	topic := f.addTopic(f.botA.Id)
	first := f.addPost(topic, f.botA.Id, 1, 0, "first accepted response")
	f.linkResult(parent.ID, first.Id)
	second := f.addPost(topic, f.botA.Id, 2, 0, "second accepted response")
	f.linkResult(parent.ID, second.Id)
	_, produced := f.capture(first)
	if produced[f.botC.Id].ID != "" {
		t.Fatal("replacing a source event result reset an already accepted reply depth")
	}
}

func TestBroadcastCaptureUsesPublishedRevisionNotPendingCandidate(t *testing.T) {
	f := broadcastSetup(t)
	var approved postRevisions.Entity
	if err := f.conn.Where("post_id = ? AND version = 1", f.first.Id).Take(&approved).Error; err != nil {
		t.Fatal(err)
	}
	pending := postRevisions.Entity{PostId: f.first.Id, Version: 2, EditorId: f.human.Id, Content: "private candidate @" + f.botB.Username, ProcessStatus: posts.ProcessStatusPending}
	if err := f.conn.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.conn.Model(&f.first).Updates(map[string]any{"published_revision_id": approved.Id, "latest_revision_id": pending.Id}).Error; err != nil {
		t.Fatal(err)
	}
	_, events := f.capture(f.first)
	if events[f.botB.Id].ID != "" {
		t.Fatal("private pending mention escaped into Agent inbox")
	}
	if events[f.botA.Id].Type != "forum.topic_created" {
		t.Fatalf("published first revision not broadcast: %#v", events)
	}
}
