package forum

import (
	"encoding/json"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestLoginPagePublishesOnlyRegistrationAvailability(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	old := pageConfig.GetByPageType(pageConfig.SecuritySettings)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{})
		if old.Id != 0 {
			conn.Create(&old)
		}
		hotdataserve.ClearSecuritySettingsConfigCache()
	})
	for _, tc := range []struct {
		name    string
		enabled bool
		limit   int
		want    string
	}{
		{"disabled", false, -1, "disabled"}, {"zero quota", true, 0, "daily_limit"}, {"unlimited", true, -1, "open"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := pageConfig.SecurityAndRegistration{EnableSignup: tc.enabled, MaxDailySignups: tc.limit}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			current := pageConfig.GetByPageType(pageConfig.SecuritySettings)
			current.PageType = pageConfig.SecuritySettings
			current.Config = string(data)
			if pageConfig.CreateOrSave(&current) != 1 {
				t.Fatal("save security config")
			}
			hotdataserve.ClearSecuritySettingsConfigCache()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/login?register=true", nil)
			encoded, err := json.Marshal(buildLoginPageProps(c))
			if err != nil {
				t.Fatal(err)
			}
			var props map[string]any
			if err := json.Unmarshal(encoded, &props); err != nil {
				t.Fatal(err)
			}
			if props["passwordSignupState"] != tc.want {
				t.Fatalf("availability=%v want %s", props["passwordSignupState"], tc.want)
			}
			for _, key := range []string{"maxDailySignups", "createdToday", "signupCount", "remainingSignups"} {
				if _, ok := props[key]; ok {
					t.Fatalf("private aggregate leaked: %s", encoded)
				}
			}
		})
	}
}
