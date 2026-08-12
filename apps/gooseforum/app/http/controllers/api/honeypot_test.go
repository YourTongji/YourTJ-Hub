package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupHoneypotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&taskQueue.Entity{},
	); err != nil {
		t.Fatalf("migrate honeypot tables: %v", err)
	}
	return conn
}

// TestRegisterHoneypotSilentlyRejects 验证注册蜜罐：隐藏字段非空时返回成功
// （MessageAuthLoginSuccess），但绝不创建用户行。
func TestRegisterHoneypotSilentlyRejects(t *testing.T) {
	setupHoneypotTestDB(t)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := `{"email":"bot@example.com","userName":"botuser","passWord":"password123","website":"http://spam.example"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Register(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (silent success)", recorder.Code)
	}
	var res component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Code != component.SUCCESS || res.MessageCode != component.MessageAuthLoginSuccess {
		t.Fatalf("response = %#v, want success + %q", res, component.MessageAuthLoginSuccess)
	}
	if users.ExistUsername("botuser") {
		t.Fatal("honeypot register must not create a user row")
	}
	if users.ExistEmail("bot@example.com") {
		t.Fatal("honeypot register must not create an email row")
	}
}

// TestWriteTopicHoneypotSilentlyRejects 验证发帖蜜罐：隐藏字段非空时返回
// SuccessResponse(true)，但绝不创建 topic/post 行。
func TestWriteTopicHoneypotSilentlyRejects(t *testing.T) {
	conn := setupHoneypotTestDB(t)
	const honeypotUserID = uint64(990105)
	conn.Unscoped().Delete(&users.EntityComplete{}, honeypotUserID)
	createTopicWriteUser(t, conn, honeypotUserID, "honeypot-author")

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: honeypotUserID,
		Params: WriteTopicReq{
			Title:       "Honeypot topic",
			Content:     "spam content",
			CategoryId:  []uint64{2001},
			TopicStatus: 1,
			Website:     "http://spam.example",
		},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("response = %#v, want silent success", res)
	}
	if result, ok := res.Data.Result.(bool); !ok || !result {
		t.Fatalf("result = %#v, want true", res.Data.Result)
	}

	var topic topics.Entity
	if err := conn.Where("title = ?", "Honeypot topic").First(&topic).Error; err == nil {
		t.Fatalf("honeypot write must not create a topic row (got id %d)", topic.Id)
	}
	var post posts.Entity
	if err := conn.Where("content = ?", "spam content").First(&post).Error; err == nil {
		t.Fatalf("honeypot write must not create a post row (got id %d)", post.Id)
	}
}

// TestForgotPasswordHoneypotSilentlyRejects 验证忘记密码蜜罐：隐藏字段非空时
// 返回成功（MessageAuthResetMailQueued），但不入队任何重置邮件。
func TestForgotPasswordHoneypotSilentlyRejects(t *testing.T) {
	conn := setupHoneypotTestDB(t)

	res := ForgotPassword(component.BetterRequest[ForgotPasswordReq]{
		Params: ForgotPasswordReq{
			Email:   "existing@example.com",
			Website: "http://spam.example",
		},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("response = %#v, want silent success", res)
	}
	if res.Data.MessageCode != component.MessageAuthResetMailQueued {
		t.Fatalf("messageCode = %q, want %q", res.Data.MessageCode, component.MessageAuthResetMailQueued)
	}

	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "reset_password").Count(&count).Error; err != nil {
		t.Fatalf("count task queue: %v", err)
	}
	if count != 0 {
		t.Fatalf("honeypot forgot-password must not enqueue a mail (count = %d)", count)
	}
}
