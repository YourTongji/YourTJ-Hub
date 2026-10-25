package api

import (
	"errors"
	"testing"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

func TestWriteTopicRollsBackWhenUploadReferenceFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	const owner = 9800400901
	const categoryID = 9800400902
	const name = "2026/10/topic-reference-rollback.webp"
	createTopicWriteUser(t, conn, owner, "upload-rollback")
	if err := conn.Create(&category.Entity{Id: categoryID, Name: "Upload atomicity", Slug: "upload-atomicity"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := fileDB.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatal(err)
	}
	if _, err := filedata.SaveFile(owner, name, "image/webp", []byte("image")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fileDB.Connect().Where("name = ?", name).Delete(&filedata.Entity{}) })
	const hook = "test:reject_topic_file_reference"
	attempted := false
	if err := conn.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "file_usages" {
			attempted = true
			_ = tx.AddError(errors.New("file usage storage unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove(hook) })
	response := WriteTopic(component.BetterRequest[WriteTopicReq]{UserId: owner, Params: WriteTopicReq{
		Title: "Do not publish without attachments", Content: "Body with image ![](/file/img/" + name + ")",
		CategoryId: []uint64{categoryID}, TopicStatus: 1,
	}})
	if !attempted {
		t.Fatalf("attachment insert was not reached: %#v", response)
	}
	var published int64
	if err := conn.Model(&topics.Entity{}).Where("user_id = ?", owner).Count(&published).Error; err != nil {
		t.Fatal(err)
	}
	if response.Data.Code != component.FAIL || published != 0 {
		t.Fatalf("attachment failure committed content: code=%v published=%d", response.Data.Code, published)
	}
}
