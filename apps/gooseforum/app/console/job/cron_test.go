package job

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/robfig/cron/v3"
)

func TestRefreshPkSyncCronAppliesSavedConfiguration(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	originalScheduler, originalRegistered, originalEntry := scheduler, registered, pkSyncEntry
	scheduler, registered, pkSyncEntry = cron.New(), false, 0
	t.Cleanup(func() {
		scheduler, registered, pkSyncEntry = originalScheduler, originalRegistered, originalEntry
		hotdataserve.ClearPkSyncScheduleConfigCache()
	})
	save := func(enabled bool, spec string) {
		t.Helper()
		pageConfig.UpdatePkSyncScheduleConfig(func(_ pageConfig.PkSyncScheduleConfig) pageConfig.PkSyncScheduleConfig {
			return pageConfig.PkSyncScheduleConfig{Enabled: enabled, Schedule: spec, Depth: 1, Audience: "undergraduate"}
		})
		hotdataserve.ClearPkSyncScheduleConfigCache()
		RefreshPkSyncCron()
	}
	// Saving before the scheduler starts must stay lazy (e.g. a CLI invocation).
	save(true, "30 2 * * *")
	if len(scheduler.Entries()) != 0 {
		t.Fatal("registered before scheduler initialization")
	}
	registered = true
	save(true, "30 2 * * *")
	first := pkSyncEntry
	if first == 0 || len(scheduler.Entries()) != 1 {
		t.Fatal("enabled schedule not registered")
	}
	save(true, "0 4 * * *")
	entries := scheduler.Entries()
	if len(entries) != 1 || entries[0].ID == first {
		t.Fatal("schedule update must replace the old job")
	}
	next := entries[0].Schedule.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if next.Hour() != 4 || next.Minute() != 0 {
		t.Fatalf("updated next run = %v", next)
	}
	save(false, "0 4 * * *")
	if pkSyncEntry != 0 || len(scheduler.Entries()) != 0 {
		t.Fatal("disabled schedule still registered")
	}
}
