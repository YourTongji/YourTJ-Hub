package routes

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"gorm.io/gorm"
)

func TestNewTopicMentionRenderingDoesNotReenterSingleConnectionPool(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())
	mentioned := createHTTPContractUser(t, conn, contractTestID())
	config := defaultconfig.GetDefaultPostingSettingsConfig()
	config.TextControl.NewUserPostCooldownMinutes = 0
	persistHTTPContractConfig(t, conn, pageConfig.PostingSettings, config)
	hotdataserve.ClearPostingSettingsConfigCache()
	token := contractSessionToken(t, author)
	pool, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	original := pool.Stats().MaxOpenConnections
	pool.SetMaxOpenConns(1)
	defer pool.SetMaxOpenConns(original)
	body := fmt.Sprintf(`{"title":"Single connection mention publication","content":"Please read this complete post @%s","categoryId":[1],"topicStatus":1,"contentType":3}`, mentioned.Username)
	var reentered atomic.Bool
	const hook = "test:single-connection-render"
	if err := conn.Callback().Query().Before("gorm:query").Register(hook, func(tx *gorm.DB) {
		// Fail the would-be reentrant query before it blocks forever. Ordinary reads
		// have no checked-out connection; transaction reads use *sql.Tx instead.
		if tx.Statement.ConnPool == pool && pool.Stats().InUse == 1 {
			reentered.Store(true)
			_ = tx.AddError(errors.New("reentrant query during publication transaction"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Callback().Query().Remove(hook); err != nil {
			t.Errorf("remove query callback: %v", err)
		}
	}()
	response := serveJSON(router, "/api/forum/topics/write", body, token)
	if reentered.Load() {
		t.Fatal("new-topic rendering requested a second connection inside its write transaction")
	}
	envelope := decodeContractEnvelope(t, response)
	if response.Code != 200 || envelope.Code != 0 {
		t.Fatalf("write failed: %d %#v", response.Code, envelope)
	}
}
