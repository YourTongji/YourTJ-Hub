package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// A deployment already at v30 must still run the legacy pending adoption.
func TestVersionedMigrationRunsPublicationAdoptionFrom30(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &badges.Entity{}, &userBadges.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncMigrationVersion(30); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetMigrationVersion(); got != 32 {
		t.Fatalf("migration stopped at %d; publication adoption and robot badges require version 32", got)
	}
}
