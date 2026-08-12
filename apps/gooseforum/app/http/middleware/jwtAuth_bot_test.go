package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	jwt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpChallenges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
)

func TestJWTAuthRejectsBotToken(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&userSessions.Entity{},
		&userTotpChallenges.Entity{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userTotpChallenges.Entity{})
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})

	bot := users.EntityComplete{
		Username:    "bot-jwt-test",
		ActorType:   users.ActorTypeBot,
		IsActivated: users.ActivationSuccess,
	}
	if err := conn.Create(&bot).Error; err != nil {
		t.Fatalf("create bot: %v", err)
	}
	token, jti, err := jwt.CreateSessionToken(bot.Id, bot.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken() error = %v", err)
	}
	if err := sessionservice.Create(bot.Id, jti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("sessionservice.Create() error = %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthCheck)
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("bot token status = %d, want 401", recorder.Code)
	}
}
