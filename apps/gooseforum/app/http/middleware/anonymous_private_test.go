package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/networkAccessLog"
	"github.com/gin-gonic/gin"
)

func TestAnonymousAdminLogsExcludePrivateFactsIncludingPanics(t *testing.T) {
	previousLog, previousIP := preferences.GetBool("server.accessLog", false), preferences.GetBool("log.logIp", false)
	preferences.Set("server.accessLog", true)
	preferences.Set("log.logIp", true)
	t.Cleanup(func() { preferences.Set("server.accessLog", previousLog); preferences.Set("log.logIp", previousIP) })
	conn := db.Connect()
	if err := conn.AutoMigrate(&networkAccessLog.Entity{}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	router := gin.New()
	router.Use(Recovery(), AccessLog)
	path := "/api/admin/anonymous-identities/list"
	router.POST(path, AnonymousPrivateResponse, func(c *gin.Context) {
		c.Set("userId", uint64(123456))
		if c.Query("panic") == "yes" {
			panic("fake private route failure")
		}
		c.Status(http.StatusNoContent)
	})
	if err := conn.Where("route = ?", path).Delete(&networkAccessLog.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{"", "&panic=yes"} {
		req := httptest.NewRequest(http.MethodPost, path+"?search=private-search"+extra, nil)
		req.RemoteAddr = "192.0.2.99:1234"
		router.ServeHTTP(httptest.NewRecorder(), req)
	}
	for _, secret := range []string{"private-search", "192.0.2.99"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("private context entered ordinary logs: %s", secret)
		}
	}
	var rows []networkAccessLog.Entity
	if err := conn.Where("route = ?", path).Find(&rows).Error; err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	for _, row := range rows {
		if row.UserId != 0 || row.ClientIP != "" || row.Path != path {
			t.Fatal("private context entered access table", row)
		}
	}
}
