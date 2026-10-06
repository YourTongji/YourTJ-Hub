package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
)

// A deployment already at v30 must still run the legacy pending adoption.
func TestVersionedMigrationRunsPublicationAdoptionFrom30(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncMigrationVersion(30); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetMigrationVersion(); got != 31 {
		t.Fatalf("migration stopped at %d; publication adoption requires version 31", got)
	}
}
