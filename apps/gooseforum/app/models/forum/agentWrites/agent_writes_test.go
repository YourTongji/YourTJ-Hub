package agentWrites

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestReservationRollbackReplayConflictAndIsolation(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.AutoMigrate(&Entry{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	entry := func() Entry {
		return Entry{InstanceID: "instance", AgentID: 1, Operation: "post", TargetID: 12, RequestKey: "reply:event:1", Digest: "digest", ExpiresAt: now.Add(7 * 24 * time.Hour)}
	}
	rolledBack := errors.New("simulated content failure")
	err = conn.Transaction(func(tx *gorm.DB) error {
		e := entry()
		if replay, err := ReserveTx(tx, &e, now); err != nil || replay != nil {
			t.Fatalf("reserve: %v %v", replay, err)
		}
		return rolledBack
	})
	if !errors.Is(err, rolledBack) {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&Entry{}).Count(&count)
	if count != 0 {
		t.Fatal("failed content creation permanently occupied key")
	}
	err = conn.Transaction(func(tx *gorm.DB) error {
		e := entry()
		if _, err := ReserveTx(tx, &e, now); err != nil {
			return err
		}
		return CompleteTx(tx, &e, 12, 45)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Transaction(func(tx *gorm.DB) error {
		e := entry()
		replay, err := ReserveTx(tx, &e, now)
		if err != nil {
			return err
		}
		if replay == nil || replay.PostID != 45 {
			t.Fatalf("lost committed result: %#v", replay)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	e := entry()
	e.Digest = "different"
	if _, err := LookupTx(conn, e, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected digest conflict: %v", err)
	}
	e = entry()
	e.AgentID = 2
	if replay, err := ReserveTx(conn, &e, now); err != nil || replay != nil {
		t.Fatalf("cross-agent collision: %v %v", replay, err)
	}
	e = entry()
	e.InstanceID = "other"
	if replay, err := ReserveTx(conn, &e, now); err != nil || replay != nil {
		t.Fatalf("cross-instance collision: %v %v", replay, err)
	}
}
