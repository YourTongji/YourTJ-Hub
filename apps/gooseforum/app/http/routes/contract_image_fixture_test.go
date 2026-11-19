package routes

import (
	"bytes"
	"image/png"
	"testing"
	"testing/synctest"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
)

func TestContractImagesAreUniqueWithFrozenClock(t *testing.T) {
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatal(err)
	}
	// Initialize shared storage/cache workers outside the synctest bubble.
	saveContractImage(t, contractTestID())
	var names []string
	synctest.Test(t, func(t *testing.T) {
		ownerID := contractTestID()
		for range 2 {
			url, name := saveContractImage(t, ownerID)
			for _, previous := range names {
				if name == previous {
					t.Fatalf("duplicate image name: %s", name)
				}
			}
			names = append(names, name)
			if url != "/file/img/"+name {
				t.Fatalf("image URL = %q, name = %q", url, name)
			}
			file, err := filedata.GetFileByName(name)
			if err != nil {
				t.Fatal(err)
			}
			if file.UserId != ownerID || file.Type != "image/png" {
				t.Fatalf("unexpected image metadata: owner=%d type=%q", file.UserId, file.Type)
			}
			img, err := png.Decode(bytes.NewReader(file.Data))
			if err != nil || img.Bounds().Dx() != 3 || img.Bounds().Dy() != 3 {
				t.Fatalf("expected a 3x3 PNG: %v", err)
			}
		}
	})
	for _, name := range names {
		file, err := filedata.GetFileMetadataByName(name)
		if err != nil || file != nil {
			t.Fatalf("image cleanup failed for %q: file=%v err=%v", name, file, err)
		}
	}
}
