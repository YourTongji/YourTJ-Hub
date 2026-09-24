package pk

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func planItemTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	migratePlanItemTestDB(t, conn)
	return conn
}
func migratePlanItemTestDB(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.AutoMigrate(&ScheduleSnapshotEntity{}, &PlanSyncOwner{}, &PlanItem{}); err != nil {
		t.Fatal(err)
	}
}
func testPlan(id string) PlanPayload { p := samplePlans()[0]; p.Id = id; return p }

func TestPlanItemsMigrateOncePreservingLegacy(t *testing.T) {
	conn := planItemTestDB(t)
	legacy := ScheduleSnapshotEntity{UserId: 1, Plans: PlanList{testPlan("a"), testPlan("recovery")}, ActivePlanId: "a"}
	legacy.Plans[1].Name = "[本地自动恢复]方案 1"
	if err := conn.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	items, err := listPlanItems(conn, 1)
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v %v", items, err)
	}
	for i := range items {
		if items[i].Revision != 1 || !reflect.DeepEqual(items[i].Payload, legacy.Plans[i]) {
			t.Fatalf("lost legacy payload: %+v", items[i])
		}
	}
	if _, err := deletePlanItem(conn, 1, "a", 1); err != nil {
		t.Fatal(err)
	}
	items, err = listPlanItems(conn, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("migration resurrected deleted plan: %+v %v", items, err)
	}
	other, err := listPlanItems(conn, 2)
	if err != nil || len(other) != 0 {
		t.Fatalf("owner isolation: %+v %v", other, err)
	}
}
func TestPlanItemsIndependentCASAndDeletion(t *testing.T) {
	conn := planItemTestDB(t)
	for _, id := range []string{"a", "b"} {
		if item, err := savePlanItem(conn, 1, testPlan(id), 0); err != nil || item.Revision != 1 {
			t.Fatalf("create %+v %v", item, err)
		}
	}
	a, b := testPlan("a"), testPlan("b")
	a.Name = "Device A"
	b.Name = "Device B"
	for _, p := range []PlanPayload{a, b} {
		if item, err := savePlanItem(conn, 1, p, 1); err != nil || item.Revision != 2 {
			t.Fatalf("independent edit %+v %v", item, err)
		}
	}
	item, err := savePlanItem(conn, 1, testPlan("a"), 1)
	if !errors.Is(err, ErrPlanConflict) || item.Payload.Name != "Device A" {
		t.Fatalf("stale %+v %v", item, err)
	}
	if _, err := deletePlanItem(conn, 1, "a", 1); !errors.Is(err, ErrPlanConflict) {
		t.Fatal(err)
	}
	if _, err := deletePlanItem(conn, 1, "a", 2); err != nil {
		t.Fatal(err)
	}
	if item, err := savePlanItem(conn, 1, a, 2); !errors.Is(err, ErrPlanDeleted) || item != nil {
		t.Fatalf("resurrected: %+v %v", item, err)
	}
	if _, err := deletePlanItem(conn, 1, "a", 2); err != nil {
		t.Fatalf("repeat delete: %v", err)
	}
}
func TestPlanItemsQuotaAllowsUpdatesAndFreedCapacity(t *testing.T) {
	conn := planItemTestDB(t)
	for i := range MaxPlanItems {
		if _, err := savePlanItem(conn, 1, testPlan(fmt.Sprint(i)), 0); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := savePlanItem(conn, 1, testPlan("overflow"), 0); !errors.Is(err, ErrPlanQuota) {
		t.Fatal(err)
	}
	if _, err := savePlanItem(conn, 1, testPlan("0"), 1); err != nil {
		t.Fatal(err)
	}
	if _, err := deletePlanItem(conn, 1, "0", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := savePlanItem(conn, 1, testPlan("overflow"), 0); err != nil {
		t.Fatal(err)
	}
}
func TestPlanItemsConcurrentQuotaPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	migratePlanItemTestDB(t, conn)
	const owner uint64 = 7140714
	for _, model := range []any{&PlanItem{}, &PlanSyncOwner{}, &ScheduleSnapshotEntity{}} {
		if err := conn.Where("user_id = ?", owner).Delete(model).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, model := range []any{&PlanItem{}, &PlanSyncOwner{}, &ScheduleSnapshotEntity{}} {
			conn.Where("user_id = ?", owner).Delete(model)
		}
	})
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := savePlanItem(conn, owner, testPlan(fmt.Sprint(i)), 0); errs <- err }()
	}
	wg.Wait()
	close(errs)
	success, quota := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPlanQuota) {
			quota++
		} else {
			t.Fatal(err)
		}
	}
	if success != 10 || quota != 10 {
		t.Fatalf("concurrent quota: successes %d rejections %d", success, quota)
	}
	items, err := listPlanItems(conn, owner)
	if err != nil || len(items) != 10 {
		t.Fatalf("%d %v", len(items), err)
	}
	// All writers observed revision 1; exactly one may advance the same plan.
	errs = make(chan error, 8)
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plan := items[0].Payload
			plan.Name = fmt.Sprint(i)
			_, err := savePlanItem(conn, owner, plan, 1)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	success = 0
	stale := 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPlanConflict) {
			stale++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || stale != 7 {
		t.Fatalf("concurrent CAS successes %d stale %d", success, stale)
	}
}

func TestPlanItemsErasurePreventsInflightRecreation(t *testing.T) {
	conn := planItemTestDB(t)
	if _, err := savePlanItem(conn, 1, testPlan("private"), 0); err != nil {
		t.Fatal(err)
	}
	if err := deleteScheduleData(conn, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := savePlanItem(conn, 1, testPlan("new"), 0); !errors.Is(err, ErrPlanOwnerClosed) {
		t.Fatalf("closed owner recreated data: %v", err)
	}
	if err := deleteScheduleData(conn, 1); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&PlanItem{}).Where("user_id = ?", 1).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("erasure %d %v", count, err)
	}
}
