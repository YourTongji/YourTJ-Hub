package decorationservice

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"sync"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/decoration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testDB(t *testing.T, postgresDSN string) *gorm.DB {
	t.Helper()
	var conn *gorm.DB
	var err error
	if postgresDSN == "" {
		conn, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	} else {
		conn, err = gorm.Open(postgres.Open(postgresDSN), &gorm.Config{TranslateError: true})
		if err == nil {
			err = conn.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	if postgresDSN == "" {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := conn.AutoMigrate(&users.EntityComplete{}, &decoration.Definition{}, &decoration.Entitlement{}, &decoration.Equipment{}, &decoration.Audit{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: 1, Username: "decorated"}).Error; err != nil {
		t.Fatal(err)
	}
	return conn
}
func adjustment(code, key string) Adjustment {
	return Adjustment{UserID: 1, Code: code, RequestKey: key, Reason: "test grant"}
}
func testStyle(code, kind string) decoration.Definition {
	style := decoration.Definition{Code: code, Name: code, Kind: kind, LightColors: []string{"#065F46", "#115E59"}, DarkColors: []string{"#6EE7B7", "#99F6E4"}, IsEnabled: true}
	if kind == AvatarFrame {
		style.LightColors = []string{"#166534", "#0D9488"}
		style.DarkColors = []string{"#86EFAC", "#2DD4BF"}
	}
	return style
}
func transact(t *testing.T, conn *gorm.DB, fn func(*gorm.DB) error) {
	t.Helper()
	if err := conn.Transaction(fn); err != nil {
		t.Fatal(err)
	}
}

func TestDecorationStylesRejectUnsafeAndUnreadableColors(t *testing.T) {
	for _, s := range []Style{testStyle("frame", AvatarFrame), testStyle("name", NicknameStyle)} {
		if err := ValidateStyle(s); err != nil {
			t.Fatalf("valid style %s: %v", s.Code, err)
		}
	}
	for _, colors := range [][]string{nil, {"red"}, {"url(https://example.com)"}, {"#FFFFFF"}, {"#fff"}, {"#123456", "#234567", "#345678"}} {
		s := testStyle("name", NicknameStyle)
		s.LightColors = colors
		if ValidateStyle(s) == nil {
			t.Fatalf("accepted unsafe/unreadable colors %v", colors)
		}
	}
	s := testStyle("frame", AvatarFrame)
	s.Code = "../style"
	if ValidateStyle(s) == nil {
		t.Fatal("invalid code accepted")
	}
	for _, asset := range []string{"https://example.com/frame.png", "/static/other.png", "/static/decorations/../other.png", "/static/decorations/frame.svg", "data:image/svg+xml,<svg/>"} {
		s := testStyle("frame", AvatarFrame)
		s.AssetURL = asset
		if ValidateStyle(s) == nil {
			t.Fatalf("accepted arbitrary frame asset %q", asset)
		}
	}
	s = testStyle("frame", AvatarFrame)
	s.AssetURL = "/static/decorations/custom-frame.webp"
	if err := ValidateStyle(s); err != nil {
		t.Fatalf("valid frame asset rejected: %v", err)
	}
	s = testStyle("name", NicknameStyle)
	s.AssetURL = "/static/decorations/custom-frame.png"
	if err := ValidateStyle(s); err == nil {
		t.Fatal("nickname accepted frame artwork")
	}
}

func assertLifecycle(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, style := range []Style{testStyle("frame-primary", AvatarFrame), testStyle("frame-secondary", AvatarFrame), testStyle("name-primary", NicknameStyle)} {
		if err := conn.Create(&style).Error; err != nil {
			t.Fatal(err)
		}
	}
	state, err := current(conn, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.OwnedCodes) != 0 || state.Equipped != (Equipped{}) {
		t.Fatal("catalog automatically granted styles")
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return equipTx(tx, 1, Equipped{AvatarFrame: "frame-primary"}) }); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unowned equip: %v", err)
	}
	grant := adjustment("frame-primary", "manual-frame-primary-1")
	transact(t, conn, func(tx *gorm.DB) error { return AdjustTx(tx, 99, "manual", grant, false) })
	transact(t, conn, func(tx *gorm.DB) error { return AdjustTx(tx, 99, "manual", grant, false) })
	transact(t, conn, func(tx *gorm.DB) error {
		return AdjustTx(tx, 99, "manual", adjustment("name-primary", "manual-name-primary-1"), false)
	})
	if err := conn.Transaction(func(tx *gorm.DB) error { return equipTx(tx, 1, Equipped{AvatarFrame: "name-primary"}) }); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("wrong kind equip: %v", err)
	}
	transact(t, conn, func(tx *gorm.DB) error {
		return equipTx(tx, 1, Equipped{AvatarFrame: "frame-primary", NicknameStyle: "name-primary"})
	})
	d := effectiveBatch(conn, []uint64{1})[1]
	if d == nil || d.AvatarFrame.Code != "frame-primary" || d.NicknameStyle.Code != "name-primary" {
		t.Fatalf("display=%+v", d)
	}
	revoke := adjustment("frame-primary", "manual-revoke-1")
	transact(t, conn, func(tx *gorm.DB) error { return AdjustTx(tx, 99, "manual", revoke, true) })
	// Replaying the original grant after a later revoke must not resurrect it.
	transact(t, conn, func(tx *gorm.DB) error { return AdjustTx(tx, 99, "manual", grant, false) })
	d = effectiveBatch(conn, []uint64{1})[1]
	if d == nil || d.AvatarFrame != nil || d.NicknameStyle == nil {
		t.Fatalf("revoke/replay display=%+v", d)
	}
	grant.Code = "frame-secondary"
	if err := conn.Transaction(func(tx *gorm.DB) error { return AdjustTx(tx, 99, "manual", grant, false) }); !errors.Is(err, ErrConflict) {
		t.Fatalf("key conflict: %v", err)
	}
	// Disabled definitions hide equipped styles without destroying ownership.
	style := testStyle("name-primary", NicknameStyle)
	style.IsEnabled = false
	if err := conn.Model(&decoration.Definition{}).Where("code = ?", style.Code).Update("is_enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if d := effectiveBatch(conn, []uint64{1})[1]; d != nil {
		t.Fatalf("disabled style shown: %+v", d)
	}
	state, err = current(conn, 1)
	if err != nil || len(state.OwnedCodes) != 1 || state.Equipped.NicknameStyle != "" {
		t.Fatalf("disabled state=%+v err=%v", state, err)
	}
	transact(t, conn, func(tx *gorm.DB) error { return equipTx(tx, 1, Equipped{}) })
	// XP/trust qualifications and permanent Credit ownership can coexist.
	for _, source := range []string{"xp", "trust", "credit"} {
		transact(t, conn, func(tx *gorm.DB) error {
			return AdjustTx(tx, 0, source, adjustment("frame-secondary", source+"-secondary-1"), false)
		})
	}
	transact(t, conn, func(tx *gorm.DB) error { return equipTx(tx, 1, Equipped{AvatarFrame: "frame-secondary"}) })
	for _, source := range []string{"xp", "trust"} {
		transact(t, conn, func(tx *gorm.DB) error {
			return AdjustTx(tx, 0, source, adjustment("frame-secondary", source+"-revoke-1"), true)
		})
	}
	if d := effectiveBatch(conn, []uint64{1})[1]; d == nil || d.AvatarFrame.Code != "frame-secondary" {
		t.Fatalf("permanent purchase lost: %+v", d)
	}
	// External caller rollback includes both entitlement and idempotency receipt.
	want := errors.New("rollback")
	_ = conn.Transaction(func(tx *gorm.DB) error {
		if err := AdjustTx(tx, 0, "credit", adjustment("frame-primary", "credit-rollback-1"), false); err != nil {
			t.Fatal(err)
		}
		return want
	})
	var count int64
	conn.Model(&decoration.Audit{}).Where("request_key = ?", "credit-rollback-1").Count(&count)
	if count != 0 {
		t.Fatal("receipt survived rollback")
	}
	conn.Model(&decoration.Entitlement{}).Where("user_id = ? AND code = ? AND source = ?", 1, "frame-primary", "credit").Count(&count)
	if count != 0 {
		t.Fatal("entitlement survived rollback")
	}
	// A closed account never exposes decorations, even with retained rows.
	if err := conn.Delete(&users.EntityComplete{}, 1).Error; err != nil {
		t.Fatal(err)
	}
	if d := effectiveBatch(conn, []uint64{1})[1]; d != nil {
		t.Fatalf("closed account decorations: %+v", d)
	}
}

func TestDecorationLifecycleSQLite(t *testing.T) { assertLifecycle(t, testDB(t, "")) }
func TestDecorationLifecyclePostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	assertLifecycle(t, testDB(t, dsn))
}
func TestDecorationConcurrentRevokeEquipPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn := testDB(t, dsn)
	if err := conn.Create(&[]decoration.Definition{testStyle("frame-primary", AvatarFrame)}).Error; err != nil {
		t.Fatal(err)
	}
	transact(t, conn, func(tx *gorm.DB) error {
		return AdjustTx(tx, 99, "manual", adjustment("frame-primary", "manual-concurrent-1"), false)
	})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	start := make(chan struct{})
	wg.Go(func() {
		<-start
		errs <- conn.Transaction(func(tx *gorm.DB) error { return equipTx(tx, 1, Equipped{AvatarFrame: "frame-primary"}) })
	})
	wg.Go(func() {
		<-start
		errs <- conn.Transaction(func(tx *gorm.DB) error {
			return AdjustTx(tx, 99, "manual", adjustment("frame-primary", "manual-concurrent-revoke"), true)
		})
	})
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil && !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
	var row decoration.Equipment
	err := conn.Where("user_id = ?", 1).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if row.AvatarFrame != "" {
		t.Fatalf("revoked frame remained equipped: %s", row.AvatarFrame)
	}
}

func TestDecorationFrameScaleValidation(t *testing.T) {
	for _, scale := range []float64{0, 0.5, 1.25, 2} {
		s := testStyle("frame-scale", AvatarFrame)
		payload, err := json.Marshal(map[string]float64{"assetScale": scale})
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(payload, &s); err != nil {
			t.Fatal(err)
		}
		if err := ValidateStyle(s); err != nil {
			t.Fatalf("scale %v: %v", scale, err)
		}
	}
	for _, scale := range []float64{-1, 0.49, 2.01} {
		s := testStyle("frame-scale", AvatarFrame)
		payload, err := json.Marshal(map[string]float64{"assetScale": scale})
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(payload, &s); err != nil {
			t.Fatal(err)
		}
		if ValidateStyle(s) == nil {
			t.Fatalf("accepted invalid scale %v", scale)
		}
	}
}

func TestDecorationFrameScaleLegacyCatalogAndFinite(t *testing.T) {
	conn := testDB(t, "")
	style := testStyle("frame-scale", AvatarFrame)
	if err := conn.Create(&style).Error; err != nil {
		t.Fatal(err)
	}
	// An externally populated legacy zero is read at the prior 125 percent size.
	if err := conn.Model(&style).Update("asset_scale", 0).Error; err != nil {
		t.Fatal(err)
	}
	styles, err := catalog(conn)
	if err != nil || len(styles) != 1 || styles[0].AssetScale != DefaultAssetScale {
		t.Fatalf("legacy styles=%+v err=%v", styles, err)
	}
	for _, scale := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		style.AssetScale = scale
		if ValidateStyle(style) == nil {
			t.Fatalf("accepted non-finite scale %v", scale)
		}
	}
	if err := conn.Model(&style).Update("asset_scale", 2.01).Error; err != nil {
		t.Fatal(err)
	}
	styles, err = catalog(conn)
	if err != nil || len(styles) != 0 {
		t.Fatalf("invalid scale exposed: %+v err=%v", styles, err)
	}
}
