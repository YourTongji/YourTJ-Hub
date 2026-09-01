package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupAccountContractTest 在共享 harness（setupHTTPContractTest）之上补齐
// 账户域 13 条路由（baseApi 公开读 + loginApi 登录写），中间件链与
// route4api.go 的生产注册保持一致。
func setupAccountContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&userOAuth.Entity{},
		&badges.Entity{},
		&taskQueue.Entity{},
		&eventNotification.Entity{},
	); err != nil {
		t.Fatalf("migrate account contract tables: %v", err)
	}
	// filedata 走独立的 db4fileconnect 连接（测试模式同样各自 :memory:），
	// 需在文件库上单独迁移。
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate filedata contract table: %v", err)
	}

	// baseApi 组：公开只读，无 middleware。
	router.GET("/api/get-captcha", UpQueryReq(api.GetCaptcha))
	router.GET("/api/user-card", UpQueryReq(api.GetUserCard))

	// loginApi 组：JWTAuthCheck 挂在组上。
	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/set-user-info", middleware.CheckWritableAccount, UpButterReq(api.EditUserInfo))
	loginAPI.POST("/set-user-profile-cover", middleware.CheckWritableAccount, UpButterReq(api.EditUserProfileCover))
	loginAPI.POST("/set-user-email", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitEmailChange), UpButterReq(api.EditUserEmail))
	loginAPI.POST("/resend-activation-email", middleware.CheckWritableAccount, UpButterReq(api.ResendActivationEmail))
	loginAPI.POST("/set-user-name", middleware.CheckWritableAccount, UpButterReq(api.EditUsername))
	loginAPI.POST("/set-preset-avatar", middleware.CheckWritableAccount, UpButterReq(api.SetPresetAvatar))
	loginAPI.POST("/wear-badge", middleware.CheckWritableAccount, UpButterReq(api.WearBadge))
	loginAPI.POST("/upload-avatar", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.UploadAvatar)
	loginAPI.POST("/change-password", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPasswordChange), UpButterReq(api.ChangePassword))
	loginAPI.POST("/auth/:provider/unbind", middleware.CheckWritableAccount, UpButterReq(api.UnbindOAuth))
	loginAPI.GET("/oauth/bindings", UpButterReq(api.GetOAuthBindings))
	return conn, router
}

// enableContractEmailVerification 打开站点邮箱验证开关（resend-activation-email
// 与 pending 用户场景的前置），harness 清理会还原原配置。
func enableContractEmailVerification(t *testing.T, conn *gorm.DB) {
	t.Helper()
	security := defaultconfig.GetDefaultSecuritySettingsConfig()
	security.CaptchaRequired = false
	security.EnableEmailVerification = true
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, security)
	hotdataserve.ClearSecuritySettingsConfigCache()
}

// useContractTempKV 把 kvstore（激活邮件重发配额）重定向到进程级临时目录，
// 避免在仓库内写 badger 数据。kvstore 是进程级单例（sync.OnceValues 连接、
// Close 后不可重连），因此只在首次使用时设置一次路径、绝不 Close。
var contractKVPathOnce sync.Once

func useContractTempKV(t *testing.T) {
	t.Helper()
	contractKVPathOnce.Do(func() {
		preferences.Set("badger.path", filepath.Join(os.TempDir(), fmt.Sprintf("yourtj-contract-badger-%d", os.Getpid())))
	})
}

// serveMultipart 以 multipart/form-data 提交文件表单（upload-avatar 为直接
// gin handler，不经 UpButterReq）。
func serveMultipart(router http.Handler, path string, files map[string][]byte, token string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for field, content := range files {
		part, err := writer.CreateFormFile(field, field+".png")
		if err != nil {
			return httptest.NewRecorder()
		}
		_, _ = part.Write(content)
	}
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// 1x1 像素 PNG（服务端 sniff 内容类型，必须是真实图片字节）。
var contractTinyPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

func TestGetCaptchaHTTPContract(t *testing.T) {
	_, router := setupAccountContractTest(t)
	recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/get-captcha", "", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("captcha status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	if response.Code != 0 {
		t.Fatalf("captcha envelope = %#v, want success", response)
	}
	// captchaId/captchaImg 为动态值，断言非空字符串且 captchaImg 为 png data URL。
	var result struct {
		CaptchaId  string `json:"captchaId"`
		CaptchaImg string `json:"captchaImg"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode captcha result %s: %v", response.Result, err)
	}
	if result.CaptchaId == "" {
		t.Fatal("captchaId is empty")
	}
	if !strings.HasPrefix(result.CaptchaImg, "data:image/png;base64,") {
		t.Fatalf("captchaImg = %q, want png data URL", result.CaptchaImg)
	}
}

func TestUserCardHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		// 固定 id/资料/统计/时间戳，使用户卡片响应与确定性 fixture 精确一致。
		user := users.MakeUser("tongji_user", "secret123", "tongji-user@example.test")
		user.Id = 1024
		user.Nickname = "同济用户"
		user.AvatarUrl = "/static/pic/3.webp"
		user.Bio = "多喝水"
		user.Signature = "同舟共济"
		user.Prestige = 12
		user.IsActivated = users.ActivationSuccess
		user.CreatedAt = time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
		user.ExternalInformation = users.ExternalInformation{
			Github: users.ExternalInformationItem{Link: "https://github.com/tongji-user"},
		}
		if err := conn.Create(user).Error; err != nil {
			t.Fatalf("create user card user: %v", err)
		}
		if err := conn.Create(&userStatistics.Entity{
			UserId:            1024,
			TopicCount:        7,
			ReplyCount:        42,
			LikeReceivedCount: 18,
			LikeGivenCount:    23,
			FollowerCount:     5,
			FollowingCount:    9,
			CollectionCount:   3,
			LastActiveTime:    time.Date(2026, 8, 15, 12, 34, 56, 0, time.UTC),
		}).Error; err != nil {
			t.Fatalf("create user card statistics: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/user-card?userId=1024", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("user card status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "user-card-success.json"))
	})

	t.Run("unknown user returns business failure", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/user-card?userId=987654321", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown user status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-save-user-badges-user-not-found.json"))
	})

	t.Run("non-numeric userId returns strict 400", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/user-card?userId=abc", "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "parse-failed.json"))
	})
}

func TestSetUserInfoHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		body := `{"nickname":"契约昵称","bio":"契约简介","signature":"契约签名"}`
		recorder := serveJSON(router, "/api/set-user-info", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("set user info status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-user-email-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/set-user-info", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/set-user-info", `{}`, "account-frozen.json")
	})
}

func TestSetUserProfileCoverHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// 个人封面要求已分配角色（RoleId != 0）。
		if err := conn.Model(user).Update("role_id", 1).Error; err != nil {
			t.Fatalf("grant role to contract user: %v", err)
		}
		body := `{"profileCoverUrl":"/static/pic/cover.webp"}`
		recorder := serveJSON(router, "/api/set-user-profile-cover", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("set profile cover status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-user-email-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/set-user-profile-cover", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/set-user-profile-cover", `{}`, "account-frozen.json")
	})
}

func TestSetUserEmailHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		body := fmt.Sprintf(`{"email":"contract-new-%d@example.test","password":"secret123"}`, user.Id)
		recorder := serveJSON(router, "/api/set-user-email", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("set user email status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-user-email-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/set-user-email", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/set-user-email", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/set-user-email",
			`{"email":"not-an-email","password":"secret123"}`,
			"set-user-email-rate-limited.json", middleware.RateLimitEmailChange)
	})

	t.Run("invalid email stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/set-user-email", `{"email":"not-an-email","password":"secret123"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid email status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestResendActivationEmailHTTPContract(t *testing.T) {
	createPendingUser := func(t *testing.T, conn *gorm.DB) *users.EntityComplete {
		t.Helper()
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_activated", users.ActivationPending).Error; err != nil {
			t.Fatalf("mark contract user pending activation: %v", err)
		}
		user.IsActivated = users.ActivationPending
		return user
	}

	t.Run("success", func(t *testing.T) {
		useContractTempKV(t)
		conn, router := setupAccountContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createPendingUser(t, conn)
		recorder := serveJSON(router, "/api/resend-activation-email", `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("resend activation status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "resend-activation-email-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/resend-activation-email", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/resend-activation-email", `{}`, "account-frozen.json")
	})

	t.Run("consecutive resend hits cooldown", func(t *testing.T) {
		useContractTempKV(t)
		conn, router := setupAccountContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createPendingUser(t, conn)
		token := contractSessionToken(t, user)
		first := serveJSON(router, "/api/resend-activation-email", `{}`, token)
		if first.Code != http.StatusOK {
			t.Fatalf("first resend status = %d, want 200: %s", first.Code, first.Body.String())
		}
		second := serveJSON(router, "/api/resend-activation-email", `{}`, token)
		if second.Code != http.StatusOK {
			t.Fatalf("cooldown resend status = %d, want 200: %s", second.Code, second.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, second), contractFixture(t, "resend-activation-email-cooldown.json"))
	})
}

func TestSetUserNameHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// 用户名规则 6-32 位 [a-zA-Z0-9_-]：contractTestID 为 19 位纳秒时间戳，前缀需保持简短。
		body := fmt.Sprintf(`{"username":"renamed-%d"}`, user.Id)
		recorder := serveJSON(router, "/api/set-user-name", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("set username status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-user-email-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/set-user-name", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/set-user-name", `{}`, "account-frozen.json")
	})

	t.Run("too short username returns business failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/set-user-name", `{"username":"abc"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid username status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-user-name-invalid.json"))
	})
}

func TestSetPresetAvatarHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/set-preset-avatar", `{"avatarUrl":"/static/pic/3.webp"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("set preset avatar status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-preset-avatar-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/set-preset-avatar", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/set-preset-avatar", `{}`, "account-frozen.json")
	})

	t.Run("non-preset avatar stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/set-preset-avatar", `{"avatarUrl":"/static/pic/99.webp"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid preset avatar status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestWearBadgeHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// 造一枚该用户拥有的可佩戴徽章（系统定义 contributor，手动授予）。
		if err := conn.Create(&userBadges.Entity{
			UserId:    user.Id,
			BadgeCode: badgeservice.CodeContributor,
			Source:    "manual",
			GrantedAt: time.Now(),
		}).Error; err != nil {
			t.Fatalf("grant contract badge: %v", err)
		}
		body := fmt.Sprintf(`{"badgeCode":%q}`, badgeservice.CodeContributor)
		recorder := serveJSON(router, "/api/wear-badge", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("wear badge status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "set-user-email-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/wear-badge", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/wear-badge", `{}`, "account-frozen.json")
	})

	t.Run("unknown badge stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/wear-badge", `{"badgeCode":"no-such-badge"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown badge status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestUploadAvatarHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveMultipart(router, "/api/upload-avatar", map[string][]byte{
			"avatar":       contractTinyPNG,
			"avatarMedium": contractTinyPNG,
		}, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("upload avatar status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		fixture := contractFixture(t, "upload-avatar-success.json")
		if response.Code != fixture.Code || response.MessageCode != fixture.MessageCode {
			t.Fatalf("upload avatar envelope = %#v, want fixture code/messageCode %#v", response, fixture)
		}
		// 存储文件名为 uuid+时间戳（动态值），断言两个 URL 均为 /file/img/ 下的非空路径。
		var result struct {
			AvatarUrl       string `json:"avatarUrl"`
			AvatarMediumUrl string `json:"avatarMediumUrl"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Fatalf("decode upload avatar result %s: %v", response.Result, err)
		}
		for name, value := range map[string]string{"avatarUrl": result.AvatarUrl, "avatarMediumUrl": result.AvatarMediumUrl} {
			if !strings.HasPrefix(value, "/file/img/") || !strings.HasSuffix(value, ".png") {
				t.Fatalf("%s = %q, want /file/img/ png path", name, value)
			}
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		recorder := serveMultipart(router, "/api/upload-avatar", map[string][]byte{"avatar": contractTinyPNG}, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveMultipart(router, "/api/upload-avatar", map[string][]byte{"avatar": contractTinyPNG}, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		restrictContractRateLimit(t, conn, middleware.RateLimitUpload)
		for attempt := 0; attempt < 5; attempt++ {
			recorder := serveMultipart(router, "/api/upload-avatar", map[string][]byte{"avatar": contractTinyPNG}, token)
			if recorder.Code != http.StatusOK {
				t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
			}
		}
		recorder := serveMultipart(router, "/api/upload-avatar", map[string][]byte{"avatar": contractTinyPNG}, token)
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limited status = %d, want 429: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "upload-avatar-rate-limited.json"))
		assertRetryAfter(t, recorder, response, middleware.RateLimitUpload)
	})

	t.Run("missing file field returns business failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveMultipart(router, "/api/upload-avatar", map[string][]byte{"other": contractTinyPNG}, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("missing file status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-img-upload-file-missing.json"))
	})
}

func TestChangePasswordHTTPContract(t *testing.T) {
	t.Run("success invalidates the old session token", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := serveJSON(router, "/api/change-password", `{"oldPassword":"secret123","newPassword":"newsecret456"}`, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("change password status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "change-password-success.json"))
		// TokenVersion 自增语义：改密成功后旧 session token 访问任意登录接口应 401。
		stale := serveAuthSecurityJSON(router, http.MethodGet, "/api/oauth/bindings", "", token)
		if stale.Code != http.StatusUnauthorized {
			t.Fatalf("stale token status = %d, want 401: %s", stale.Code, stale.Body.String())
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/change-password", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/change-password", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/change-password",
			`{"oldPassword":"wrong-old","newPassword":"newsecret456"}`,
			"change-password-rate-limited.json", middleware.RateLimitPasswordChange)
	})

	t.Run("wrong old password returns business failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/change-password", `{"oldPassword":"wrong-old","newPassword":"newsecret456"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("old-invalid status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "change-password-old-invalid.json"))
	})
}

func TestOAuthBindingsHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		boundAt := time.Date(2025, 10, 2, 9, 30, 0, 0, time.UTC)
		if err := conn.Create(&userOAuth.Entity{
			UserId:      user.Id,
			Provider:    "github",
			ProviderUid: "github-uid-1024",
			CreatedAt:   boundAt,
			UpdatedAt:   boundAt,
		}).Error; err != nil {
			t.Fatalf("create oauth binding: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/oauth/bindings", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("oauth bindings status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "oauth-bindings-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/oauth/bindings", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})
}

func TestUnbindOAuthHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		// 有邮箱 + 一个 OAuth 绑定：解绑后仍保留邮箱登录方式。
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Create(&userOAuth.Entity{UserId: user.Id, Provider: "github", ProviderUid: "github-uid-unbind"}).Error; err != nil {
			t.Fatalf("create oauth binding: %v", err)
		}
		recorder := serveJSON(router, "/api/auth/github/unbind", `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unbind status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "oauth-unbind-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAccountContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/auth/github/unbind", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/auth/github/unbind", `{}`, "account-frozen.json")
	})

	t.Run("last login method returns business failure", func(t *testing.T) {
		conn, router := setupAccountContractTest(t)
		// 无邮箱且唯一绑定：解绑会使账号失去所有登录方式，被拒绝。
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("email", "").Error; err != nil {
			t.Fatalf("clear contract user email: %v", err)
		}
		if err := conn.Create(&userOAuth.Entity{UserId: user.Id, Provider: "github", ProviderUid: "github-uid-last"}).Error; err != nil {
			t.Fatalf("create oauth binding: %v", err)
		}
		recorder := serveJSON(router, "/api/auth/github/unbind", `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unbind failed status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "oauth-unbind-failed.json"))
	})
}
