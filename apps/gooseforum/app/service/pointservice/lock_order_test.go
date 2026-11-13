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

	var queryTables []string
	if err := conn.Callback().Query().Before("gorm:query").Register("test:points-lock-read", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "users" {
			queryTables = append(queryTables, tx.Statement.Table)
			_, locked := tx.Statement.Clauses["FOR"]
			if locked {
				queryTables = append(queryTables, "users:for-update")
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:points-lock-read") })

	var updateTables []string
	if err := conn.Callback().Update().Before("gorm:update").Register("test:points-lock-update", func(tx *gorm.DB) {
		if tx.Statement != nil {
			updateTables = append(updateTables, tx.Statement.Table)
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

	lockIndex := -1
	for i, value := range queryTables {
		if value == "users:for-update" {
			lockIndex = i
			break
		}
	}
	if lockIndex < 0 {
		t.Fatalf("users FOR UPDATE read not observed: %v", queryTables)
	}
	balanceIndex := -1
	for i, value := range updateTables {
		if value == "user_points" {
			balanceIndex = i
			break
		}
	}
	if balanceIndex < 0 {
		t.Fatalf("user_points update not observed: %v", updateTables)
	}
	for _, value := range updateTables[:balanceIndex] {
		if value == "users" {
			t.Fatalf("prestige update preceded balance update: %v", updateTables)
		}
	}
	assertPointsState(t, conn, 1, PostCreatedReward, PostCreatedReward)
}
