package httpnotifyservice

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func TestFeishuRetryRefreshesTimestampAndCurrentSecret(t *testing.T) {
	oldKey := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", "httpnotify-retry-test-key-0123456789")
	t.Cleanup(func() { preferences.Set("app.signingKey", oldKey) })
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	t.Cleanup(server.Close)
	if err := dbconnect.Connect().AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
	original := entity.Config
	t.Cleanup(func() {
		entity.Config = original
		pageConfig.CreateOrSave(&entity)
		hotdataserve.ClearHttpNotifyConfigCache()
	})
	sealed, err := securestore.EncryptPurpose("current-secret", securestore.HttpNotifySecretPurpose)
	if err != nil {
		t.Fatal(err)
	}
	entity.PageType = pageConfig.HttpNotify
	entity.Config = jsonopt.Encode(pageConfig.HttpNotifyStorageConfig{Enabled: true, Endpoints: []pageConfig.HttpNotifyStorageEndpoint{{Id: "retry-signing", URL: server.URL, Enabled: true, ChannelType: pageConfig.HttpNotifyChannelFeishu, SecretEncrypted: sealed, Events: []string{EventReportPostCreated}}}})
	pageConfig.CreateOrSave(&entity)
	hotdataserve.ClearHttpNotifyConfigCache()
	oldTime := time.Now().Add(-2 * time.Hour)
	body, err := (feishuChannel{}).encode(pageConfig.HttpNotifyEndpoint{Secret: "old-secret"}, Alternative{}, testApproval(), oldTime)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(deliveryRetryTask{EndpointKey: "id:retry-signing", ChannelType: pageConfig.HttpNotifyChannelFeishu, Event: EventReportPostCreated, Timestamp: oldTime.Unix(), Body: body, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := RunDeliveryRetryTask(context.Background(), &taskQueue.Entity{TaskJson: string(payload)}); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Timestamp string          `json:"timestamp"`
		Sign      string          `json:"sign"`
		Card      json.RawMessage `json:"card"`
	}
	if err := json.Unmarshal(received, &decoded); err != nil {
		t.Fatal(err)
	}
	timestamp, err := strconv.ParseInt(decoded.Timestamp, 10, 64)
	if err != nil || time.Now().Unix()-timestamp > 5 || decoded.Sign != feishuSign(decoded.Timestamp, "current-secret") {
		t.Fatalf("retry used stale signing credentials: timestamp=%s sign=%s", decoded.Timestamp, decoded.Sign)
	}
	if len(decoded.Card) == 0 {
		t.Fatal("retry lost approval card")
	}
}
