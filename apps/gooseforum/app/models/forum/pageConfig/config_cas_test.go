package pageConfig

import (
	"context"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

func TestConfigCompareAndSwap(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&Entity{}); err != nil {
		t.Fatal(err)
	}
	key := "cas-calendar-test"
	db.Where("page_type = ?", key).Delete(&Entity{})
	t.Cleanup(func() { db.Where("page_type = ?", key).Delete(&Entity{}) })
	ctx := context.Background()
	if value, err := ReadConfig(ctx, key); err != nil || value != "" {
		t.Fatal("missing read")
	}
	if ok, err := CompareAndSwapConfig(ctx, key, "", "one"); err != nil || !ok {
		t.Fatal("create", err)
	}
	if ok, err := CompareAndSwapConfig(ctx, key, "", "two"); err != nil || ok {
		t.Fatal("duplicate create")
	}
	if ok, err := CompareAndSwapConfig(ctx, key, "wrong", "two"); err != nil || ok {
		t.Fatal("stale update")
	}
	if ok, err := CompareAndSwapConfig(ctx, key, "one", "two"); err != nil || !ok {
		t.Fatal("replace", err)
	}
	if value, err := ReadConfig(ctx, key); err != nil || value != "two" {
		t.Fatal("lost saved value")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ReadConfig(canceled, key); err == nil {
		t.Fatal("swallowed canceled read")
	}
}
