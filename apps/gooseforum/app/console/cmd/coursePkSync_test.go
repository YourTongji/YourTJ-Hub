package cmd

import (
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/spf13/cobra"
)

func TestResolvePkCalendarId(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&pk.CalendarEntity{}); err != nil {
		t.Fatalf("migrate pk_calendar: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&pk.CalendarEntity{})
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 121, CalendarIdI18n: "2025-2026-1"}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	t.Cleanup(func() { conn.Unscoped().Where("1 = 1").Delete(&pk.CalendarEntity{}) })

	cases := []struct {
		name     string
		arg      string
		explicit uint64
		want     uint64
		wantErr  bool
	}{
		{name: "numeric calendarId", arg: "121", want: 121},
		{name: "term code resolved via pk_calendar", arg: "2025-2026-1", want: 121},
		{name: "explicit --calendar-id overrides", arg: "2025-2026-1", explicit: 999, want: 999},
		{name: "unknown term errors", arg: "2026-2027-1", wantErr: true},
		{name: "garbage errors", arg: "not-a-term", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolvePkCalendarId(c.arg, c.explicit)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for arg=%q", c.arg)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve(%q): %v", c.arg, err)
			}
			if got != c.want {
				t.Errorf("resolve(%q) = %d, want %d", c.arg, got, c.want)
			}
		})
	}
}

func TestCoursePkSyncCommandRegistered(t *testing.T) {
	var cmd *cobra.Command
	for _, c := range GetCommands() {
		if strings.HasPrefix(c.Use, "course-pk-sync") {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("course-pk-sync command not registered")
	}
	for _, flag := range []string{"depth", "onesystem-cookie", "calendar-id", "materialize"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("course-pk-sync missing --%s flag", flag)
		}
	}
}
