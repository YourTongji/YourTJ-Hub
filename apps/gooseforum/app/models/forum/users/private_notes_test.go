package users

import (
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"testing"
)

func TestPrivateNoteQuotaAndRename(t *testing.T) {
	setupUserIsolationTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&PrivateNoteEntity{}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []*EntityComplete{MakeUser("note-owner", "password", "owner@notes.test"), MakeUser("note-target", "password", "target@notes.test")} {
		if err := Create(u); err != nil {
			t.Fatal(err)
		}
	}
	owner, _ := GetByUsername("note-owner")
	target, _ := GetByUsername("note-target")
	rows := make([]PrivateNoteEntity, MaxPrivateNotes)
	for i := range rows {
		rows[i] = PrivateNoteEntity{OwnerID: owner.Id, TargetUserID: uint64(100000 + i), Note: "n"}
	}
	if err := conn.CreateInBatches(rows, 100).Error; err != nil {
		t.Fatal(err)
	}
	if err := SetPrivateNote(owner.Id, target.Id, "too many"); !errors.Is(err, ErrPrivateNoteLimit) {
		t.Fatalf("quota: %v", err)
	}
	if err := SetPrivateNote(owner.Id, rows[0].TargetUserID, ""); err != nil {
		t.Fatal(err)
	}
	if err := SetPrivateNote(owner.Id, target.Id, "friend"); err != nil {
		t.Fatal(err)
	}
	if err := SetPrivateNote(owner.Id, target.Id, "updated"); err != nil {
		t.Fatal(err)
	}
	if err := UpdateFields(target.Id, map[string]any{"username": "renamed-target"}); err != nil {
		t.Fatal(err)
	}
	notes, err := ListPrivateNotes(owner.Id)
	if err != nil || len(notes) != 1 || notes[0].Username != "renamed-target" || notes[0].Note != "updated" {
		t.Fatalf("renamed note: %+v %v", notes, err)
	}
}
