package pointservice

import (
	"testing"

	"gorm.io/gorm"
)

// The actor row lock must be a FOR UPDATE read before any points ledger or
// balance write. A plain actor snapshot cannot prevent the PostgreSQL
// users <-> user_points deadlock with persona publication.
func TestApplyPointsTxLocksParticipantBeforeBalance(t *testing.T) {
	conn := newPointsTestDB(t)
	createPointsUser(t, conn, 1, 0)
	createPointsPost(t, conn, 2, 1, 1, 0)

	var events []string
	if err := conn.Callback().Query().Before("gorm:query").Register("test:points-lock-read", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "users" {
			return
		}
		if _, locked := tx.Statement.Clauses["FOR"]; locked {
			events = append(events, "users:for-update")
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:points-lock-read") })

	if err := conn.Callback().Update().Before("gorm:update").Register("test:points-lock-update", func(tx *gorm.DB) {
		if tx.Statement == nil {
			return
		}
		switch tx.Statement.Table {
		case "user_points", "users":
			events = append(events, tx.Statement.Table)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Update().Remove("test:points-lock-update") })

	if err := conn.Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, 1, PostCreatedReward, PointsActionPostCreated, "post:2", "")
	}); err != nil {
		t.Fatal(err)
	}

	lockIndex, balanceIndex := -1, -1
	for i, event := range events {
		if event == "users:for-update" && lockIndex < 0 {
			lockIndex = i
		}
		if event == "user_points" && balanceIndex < 0 {
			balanceIndex = i
		}
	}
	if lockIndex < 0 {
		t.Fatalf("users FOR UPDATE read not observed: %v", events)
	}
	if balanceIndex < 0 {
		t.Fatalf("user_points update not observed: %v", events)
	}
	if lockIndex > balanceIndex {
		t.Fatalf("participant lock followed balance update: %v", events)
	}
	for _, event := range events[lockIndex+1 : balanceIndex] {
		if event == "users" {
			t.Fatalf("prestige update preceded balance update: %v", events)
		}
	}
	assertPointsState(t, conn, 1, PostCreatedReward, PostCreatedReward)
}
