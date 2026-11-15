package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
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
		if topic, ok := tx.Statement.Dest.(*topics.Entity); ok && (topic.Id == ids[0] || topic.Id == ids[1]) {
			attempts++
			if failSave && topic.Id == ids[1] {
				_ = tx.AddError(errors.New("injected save failure"))
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
