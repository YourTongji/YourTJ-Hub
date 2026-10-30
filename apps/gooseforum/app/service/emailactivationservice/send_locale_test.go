package emailactivationservice

import (
	"encoding/json"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
)

func TestActivationQueuePreservesSupportedLocales(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	oldKey := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", "locale-activation-test-key")
	t.Cleanup(func() { preferences.Set("app.signingKey", oldKey) })
	for i, locale := range []string{"zh", "en", "ja", "de"} {
		t.Run(locale, func(t *testing.T) {
			email := "locale-check-" + locale + "@example.test"
			where := conn.Where("type = ? AND task_json LIKE ?", "email.activation", "%"+email+"%")
			if err := where.Delete(&taskQueue.Entity{}).Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				conn.Where("type = ? AND task_json LIKE ?", "email.activation", "%"+email+"%").Delete(&taskQueue.Entity{})
			})
			if err := SendActivationEmail(&users.EntityComplete{Id: uint64(7000 + i), Username: "locale-check", Email: email, Locale: locale}); err != nil {
				t.Fatal(err)
			}
			var stored taskQueue.Entity
			if err := conn.Where("type = ? AND task_json LIKE ?", "email.activation", "%"+email+"%").First(&stored).Error; err != nil {
				t.Fatal(err)
			}
			var task mailservice.EmailTask
			if err := json.Unmarshal([]byte(stored.TaskJson), &task); err != nil {
				t.Fatal(err)
			}
			if task.Locale != locale || task.To != email || task.Token == "" || task.Type != "activation" {
				t.Fatal("activation task lost recipient, language, token or type")
			}
		})
	}
}
