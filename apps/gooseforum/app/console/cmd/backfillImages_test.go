package cmd

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func executeBackfillImages(t *testing.T) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
	for _, command := range GetCommands() {
		if command.Name() == "backfill-images" {
			root.AddCommand(command)
			defer root.RemoveCommand(command)
			break
		}
	}
	var output bytes.Buffer
	root.SetOut(&output)
	var errorOutput bytes.Buffer
	root.SetErr(&errorOutput)
	root.SetArgs([]string{"backfill-images"})
	err := root.Execute()
	return output.String(), err
}

func TestBackfillImagesQueryFailure(t *testing.T) {
	conn := dbconnect.Connect()
	sentinel := errors.New("injected topic query failure")
	const callback = "test:backfill_images_query"
	if err := conn.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "topics" {
			_ = tx.AddError(sentinel)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Query().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	output, err := executeBackfillImages(t)
	if !errors.Is(err, sentinel) {
		t.Fatalf("command error = %v, want query failure", err)
	}
	if strings.Contains(output, "backfilled") {
		t.Fatalf("query failure printed success summary: %q", output)
	}
}

func TestBackfillImagesPartialSaveFailure(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	ids := []uint64{9200000000001, 9200000000002}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", ids).Delete(&posts.Entity{})
		conn.Unscoped().Where("id IN ?", ids).Delete(&topics.Entity{})
	})
	for _, id := range ids {
		topic := topics.Entity{Id: id, FirstPostId: id, Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive}
		post := posts.Entity{Id: id, TopicId: id, PostNo: 1, Content: "![example](https://example.com/image.png)", ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
	}
	const callback = "test:backfill_images_save"
	failSave := true
	attempts := 0
	if err := conn.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "topics" {
			return
		}
		updates, ok := tx.Statement.Dest.(map[string]any)
		if !ok {
			return
		}
		if _, ok := updates["image_urls"]; !ok {
			return
		}
		whereClause, ok := tx.Statement.Clauses["WHERE"]
		if !ok {
			return
		}
		where, ok := whereClause.Expression.(clause.Where)
		if !ok {
			return
		}
		var topicID uint64
		for _, expression := range where.Exprs {
			switch expr := expression.(type) {
			case clause.Eq:
				column, ok := expr.Column.(clause.Column)
				if ok && column.Name == "id" {
					topicID, _ = expr.Value.(uint64)
				}
			case clause.Expr:
				if strings.EqualFold(strings.TrimSpace(expr.SQL), "id = ?") && len(expr.Vars) == 1 {
					topicID, _ = expr.Vars[0].(uint64)
				}
			}
			if topicID != 0 {
				break
			}
		}
		if topicID == ids[0] || topicID == ids[1] {
			attempts++
			if failSave && topicID == ids[1] {
				_ = tx.AddError(errors.New("injected image URL update failure"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Update().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	output, err := executeBackfillImages(t)
	if err == nil {
		t.Fatal("partial save failure must fail the command")
	}
	if !strings.Contains(output, "save failed topic 9200000000002") {
		t.Fatalf("missing failed topic diagnostic: %q", output)
	}
	var saved topics.Entity
	if err := conn.First(&saved, ids[0]).Error; err != nil {
		t.Fatal(err)
	}
	if len(saved.ImageUrls) != 1 {
		t.Fatalf("later topic was not processed: %#v", saved.ImageUrls)
	}
	failSave = false
	if _, err := executeBackfillImages(t); err != nil {
		t.Fatalf("successful retry: %v", err)
	}
	before := attempts
	if _, err := executeBackfillImages(t); err != nil {
		t.Fatalf("unchanged topics: %v", err)
	}
	if attempts != before {
		t.Fatal("unchanged images must not be saved again")
	}
}

func TestBackfillImagesFirstPostReadFailure(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	const valid, broken, missing uint64 = 9200000000011, 9200000000012, 9200000000013
	ids := []uint64{valid, broken, missing}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", ids).Delete(&posts.Entity{})
		conn.Unscoped().Where("id IN ?", ids).Delete(&topics.Entity{})
	})
	for _, id := range ids {
		topic := topics.Entity{Id: id, FirstPostId: id, Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		post := posts.Entity{Id: id, TopicId: id, PostNo: 1, Content: "![example](https://example.com/image.png)", ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive}
		if err := conn.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
	}
	failRead := true
	const callback = "test:backfill_images_first_post_read"
	if err := conn.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "posts" {
			return
		}
		where, _ := tx.Statement.Clauses["WHERE"].Expression.(clause.Where)
		for _, expr := range where.Exprs {
			in, ok := expr.(clause.IN)
			if !ok {
				continue
			}
			switch {
			case failRead && slices.Contains(in.Values, any(broken)):
				_ = tx.AddError(errors.New("injected first-post query failure"))
			case slices.Contains(in.Values, any(missing)):
				// The first post disappeared after the public topic scan.
				_ = tx.AddError(gorm.ErrRecordNotFound)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Query().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	output, err := executeBackfillImages(t)
	if err == nil {
		t.Fatalf("first-post read failure must fail the command, output=%q", output)
	}
	if !strings.Contains(output, "read first post failed topic 9200000000012") ||
		strings.Contains(output, "9200000000013") || strings.Contains(output, "backfilled image_urls") {
		t.Fatalf("wrong diagnostics for unreadable and missing posts: %q", output)
	}
	var saved topics.Entity
	if err := conn.First(&saved, valid).Error; err != nil {
		t.Fatal(err)
	}
	if len(saved.ImageUrls) != 1 {
		t.Fatalf("other topic was not processed: %#v", saved.ImageUrls)
	}
	failRead = false
	if _, err := executeBackfillImages(t); err != nil {
		t.Fatalf("retry must succeed while the genuinely missing post is skipped: %v", err)
	}
}

func TestBackfillImagesPreservesConcurrentTopicUpdates(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	const (
		topicID       = uint64(9200000000021)
		hiddenTopicID = uint64(9200000000022)
		firstPostID   = topicID
		hiddenPostID  = hiddenTopicID
		concurrentID  = uint64(9200000000031)
		followupID    = uint64(9200000000032)
	)
	topicIDs := []uint64{topicID, hiddenTopicID}
	postIDs := []uint64{firstPostID, hiddenPostID, concurrentID, followupID}
	conn.Unscoped().Where("id IN ?", topicIDs).Delete(&topics.Entity{})
	conn.Unscoped().Where("id IN ?", postIDs).Delete(&posts.Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", topicIDs).Delete(&topics.Entity{})
		conn.Unscoped().Where("id IN ?", postIDs).Delete(&posts.Entity{})
	})

	createdAt := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	content := "![one](https://example.com/1.png)\n![two](https://example.com/2.png)\n" +
		"![three](https://example.com/3.png)\n![four](https://example.com/4.png)"
	if err := conn.Create(&[]posts.Entity{
		{Id: firstPostID, TopicId: topicID, PostNo: 1, Content: content, ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityActive, CreatedAt: createdAt},
		{Id: hiddenPostID, TopicId: hiddenTopicID, PostNo: 1, Content: content, ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityActive, CreatedAt: createdAt},
	}).Error; err != nil {
		t.Fatalf("create first posts: %v", err)
	}
	if err := conn.Create(&[]topics.Entity{
		{
			Id: topicID, Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive,
			FirstPostId: firstPostID, LastPostId: firstPostID, LastPostedAt: &createdAt,
			PostCount: 1, ReplyCount: 0, PostSeq: 1, Posters: []topics.Poster{{UserID: 42}},
			LikeCount: 3, ViewCount: 10, FirstImageURL: "stale-first-image",
			CreatedAt: createdAt, UpdatedAt: createdAt,
		},
		{
			Id: hiddenTopicID, Status: 0, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive,
			FirstPostId: hiddenPostID, CreatedAt: createdAt, UpdatedAt: createdAt,
		},
	}).Error; err != nil {
		t.Fatalf("create topics: %v", err)
	}

	var (
		snapshotRead       bool
		snapshotPostSeq    uint64
		scannedLimit       int
		injectedErr        error
		concurrentTopic    topics.Entity
		imageUpdateCount   int
		imageUpdateColumns string
	)
	const readCallback = "test:backfill_images_concurrent_snapshot"
	if err := conn.Callback().Query().After("gorm:query").Register(readCallback, func(tx *gorm.DB) {
		if tx.Statement.Table != "topics" || snapshotRead {
			return
		}
		rows, ok := tx.Statement.Dest.(*[]*topics.Entity)
		if !ok {
			return
		}
		snapshotRead = true
		if limitClause, ok := tx.Statement.Clauses["LIMIT"]; ok {
			if limit, ok := limitClause.Expression.(clause.Limit); ok && limit.Limit != nil {
				scannedLimit = *limit.Limit
			}
		}
		for _, topic := range *rows {
			if topic.Id == topicID {
				snapshotPostSeq = topic.PostSeq
			}
			if topic.Id == hiddenTopicID {
				injectedErr = errors.New("non-public topic was included in the backfill snapshot")
				return
			}
		}

		// Simulate a committed reply after GetLatestPublished returned its old snapshot.
		postNo, err := topics.ReservePostSequence(topicID)
		if err != nil {
			injectedErr = err
			return
		}
		if postNo != 2 {
			injectedErr = errors.New("concurrent reply reserved an unexpected post number")
			return
		}
		if err := conn.Create(&posts.Entity{
			Id: concurrentID, TopicId: topicID, PostNo: postNo, UserId: 43,
			Content: "concurrent reply", ProcessStatus: posts.ProcessStatusNormal,
			VisibilityStatus: posts.VisibilityActive,
		}).Error; err != nil {
			injectedErr = err
			return
		}
		lastPostedAt := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
		if err := topics.IncrementPostFast(topicID, []topics.Poster{{UserID: 42}, {UserID: 43}}, concurrentID, lastPostedAt); err != nil {
			injectedErr = err
			return
		}
		if topics.IncrementLike(topics.Entity{Id: topicID}) != 1 {
			injectedErr = errors.New("concurrent like did not update the topic")
			return
		}
		if err := topics.IncrementViews(map[uint64]uint64{topicID: 5}); err != nil {
			injectedErr = err
			return
		}
		if err := conn.Table("topics").Where("id = ?", topicID).UpdateColumn("first_image_url", "concurrent-first-image").Error; err != nil {
			injectedErr = err
			return
		}
		concurrentUpdatedAt := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
		if err := conn.Table("topics").Where("id = ?", topicID).UpdateColumn("updated_at", concurrentUpdatedAt).Error; err != nil {
			injectedErr = err
			return
		}
		concurrentTopic, injectedErr = topics.GetWithError(topicID)
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Query().Remove(readCallback); err != nil {
			t.Error(err)
		}
	})

	const updateCallback = "test:backfill_images_update_columns"
	if err := conn.Callback().Update().Before("gorm:update").Register(updateCallback, func(tx *gorm.DB) {
		if tx.Statement.Table != "topics" {
			return
		}
		updates, ok := tx.Statement.Dest.(map[string]any)
		if ok {
			if _, ok := updates["image_urls"]; !ok {
				return
			}
			imageUpdateCount++
			if len(updates) == 1 {
				imageUpdateColumns = "image_urls"
			} else {
				imageUpdateColumns = "unexpected columns"
			}
			return
		}
		if topic, ok := tx.Statement.Dest.(*topics.Entity); ok && topic.Id == topicID {
			imageUpdateCount++
			imageUpdateColumns = "whole topic entity"
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Update().Remove(updateCallback); err != nil {
			t.Error(err)
		}
	})

	if output, err := executeBackfillImages(t); err != nil {
		t.Fatalf("backfill command: %v, output=%q", err, output)
	}
	if injectedErr != nil {
		t.Fatalf("concurrent write callback: %v", injectedErr)
	}
	if !snapshotRead || snapshotPostSeq != 1 {
		t.Fatalf("snapshot read=%v post_seq=%d, want stale snapshot post_seq=1", snapshotRead, snapshotPostSeq)
	}
	if scannedLimit != 10000 {
		t.Fatalf("public topic query limit = %d, want 10000", scannedLimit)
	}
	got, err := topics.GetWithError(topicID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PostSeq != concurrentTopic.PostSeq || got.PostCount != concurrentTopic.PostCount ||
		got.ReplyCount != concurrentTopic.ReplyCount || !slices.Equal(got.Posters, concurrentTopic.Posters) ||
		got.LastPostId != concurrentTopic.LastPostId || got.LastPostedAt == nil || concurrentTopic.LastPostedAt == nil ||
		!got.LastPostedAt.Equal(*concurrentTopic.LastPostedAt) || got.LikeCount != concurrentTopic.LikeCount ||
		got.ViewCount != concurrentTopic.ViewCount {
		t.Fatalf("concurrent topic fields regressed: got=%#v want=%#v", got, concurrentTopic)
	}
	if got.FirstImageURL != "concurrent-first-image" {
		t.Fatalf("first_image_url = %q, want concurrent value", got.FirstImageURL)
	}
	if !got.UpdatedAt.Equal(concurrentTopic.UpdatedAt) {
		t.Fatalf("updated_at changed: got=%s want=%s", got.UpdatedAt, concurrentTopic.UpdatedAt)
	}
	wantImages := []string{
		"https://example.com/1.png",
		"https://example.com/2.png",
		"https://example.com/3.png",
	}
	if !slices.Equal(got.ImageUrls, wantImages) {
		t.Fatalf("image_urls = %#v, want %#v", got.ImageUrls, wantImages)
	}
	if imageUpdateCount != 1 || imageUpdateColumns != "image_urls" {
		t.Fatalf("image URL updates=%d columns=%q, want exactly one image_urls-only update", imageUpdateCount, imageUpdateColumns)
	}
	hidden, err := topics.GetWithError(hiddenTopicID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden.ImageUrls) != 0 {
		t.Fatalf("non-public topic was backfilled: %#v", hidden.ImageUrls)
	}

	postNo, err := topics.ReservePostSequence(topicID)
	if err != nil {
		t.Fatalf("reserve follow-up post number: %v", err)
	}
	if postNo != 3 {
		t.Fatalf("follow-up post number = %d, want 3", postNo)
	}
	if err := conn.Create(&posts.Entity{
		Id: followupID, TopicId: topicID, PostNo: postNo, UserId: 44,
		Content: "follow-up reply", ProcessStatus: posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
	}).Error; err != nil {
		t.Fatalf("insert follow-up post after backfill: %v", err)
	}

	if output, err := executeBackfillImages(t); err != nil {
		t.Fatalf("unchanged rerun: %v, output=%q", err, output)
	}
	if imageUpdateCount != 1 {
		t.Fatalf("unchanged rerun performed %d image URL updates, want 0 additional", imageUpdateCount-1)
	}
}
