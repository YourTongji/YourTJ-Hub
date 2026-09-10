package pk

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

func TestAudienceScopedIDsKeepExternalIDsAndNamespacesSeparate(t *testing.T) {
	const externalID uint64 = 121

	undergraduateID := ScopeID(AudienceUndergraduate, externalID)
	graduateID := ScopeID(AudienceGraduate, externalID)
	if undergraduateID == graduateID {
		t.Fatalf("audience scoped ids collide: undergraduate=%d graduate=%d", undergraduateID, graduateID)
	}
	if ExternalID(AudienceUndergraduate, undergraduateID) != externalID {
		t.Fatalf("undergraduate external id = %d, want %d", ExternalID(AudienceUndergraduate, undergraduateID), externalID)
	}
	if ExternalID(AudienceGraduate, graduateID) != externalID {
		t.Fatalf("graduate external id = %d, want %d", ExternalID(AudienceGraduate, graduateID), externalID)
	}
}

func TestParseAudienceDefaultsOnlyEmptyInput(t *testing.T) {
	for _, value := range []string{"", "undergraduate", "ug", "本科"} {
		got, ok := ParseAudience(value)
		if !ok || got != AudienceUndergraduate {
			t.Errorf("ParseAudience(%q) = %q, %v; want undergraduate, true", value, got, ok)
		}
	}
	for _, value := range []string{"graduate", "grad", "研究生"} {
		got, ok := ParseAudience(value)
		if !ok || got != AudienceGraduate {
			t.Errorf("ParseAudience(%q) = %q, %v; want graduate, true", value, got, ok)
		}
	}
}

func TestAudienceScopedIDPersistsInExistingUint64PKTables(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&CalendarEntity{}); err != nil {
		t.Fatalf("migrate calendar: %v", err)
	}
	const externalID uint64 = 121
	rows := []CalendarEntity{
		{CalendarId: ScopeID(AudienceUndergraduate, externalID), Audience: string(AudienceUndergraduate), ExternalId: externalID, CalendarIdI18n: "ug"},
		{CalendarId: ScopeID(AudienceGraduate, externalID), Audience: string(AudienceGraduate), ExternalId: externalID, CalendarIdI18n: "grad"},
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatalf("persist audience-scoped calendar ids: %v", err)
	}
	var graduate CalendarEntity
	if err := conn.Where("audience = ? AND calendar_id = ?", AudienceGraduate, ScopeID(AudienceGraduate, externalID)).First(&graduate).Error; err != nil {
		t.Fatalf("read graduate calendar: %v", err)
	}
	if graduate.ExternalId != externalID {
		t.Fatalf("graduate external id = %d, want %d", graduate.ExternalId, externalID)
	}
}

func TestAudienceScopedIDsSurviveJSONNumberRoundTrip(t *testing.T) {
	for _, id := range []uint64{1, 121, 10000001, graduateIDMask - 1} {
		scoped := ScopeID(AudienceGraduate, id)
		if uint64(float64(scoped)) != scoped {
			t.Fatalf("scoped id %d loses precision in a browser JSON number", scoped)
		}
	}
}

func TestGraduateTeacherLookupAcceptsExternalAndScopedClassIDs(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&TeacherEntity{}); err != nil {
		t.Fatal(err)
	}
	const classID uint64 = 998144
	teacher := TeacherEntity{Id: ScopeID(AudienceGraduate, 998145), Audience: string(AudienceGraduate), TeachingClassId: ScopeID(AudienceGraduate, classID), TeacherName: "graduate lookup"}
	if err := conn.Create(&teacher).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{classID, ScopeID(AudienceGraduate, classID)} {
		rows, err := ListTeachersByAudienceClassIdsTx(conn, AudienceGraduate, []uint64{id})
		if err != nil || len(rows) != 1 || rows[0].Id != teacher.Id {
			t.Fatalf("lookup class %d = %v, %v; want graduate teacher", id, rows, err)
		}
	}
}
