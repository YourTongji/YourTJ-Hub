package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupWritableAccountGateTest 迁移中间件测试表并持久化 SecuritySettings
// （EnableEmailVerification 按用例开关），cleanup 恢复原配置并清配置缓存。
func setupWritableAccountGateTest(t *testing.T, enableEmailVerification bool) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate activation gate tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&users.EntityComplete{})

	var previous *pageConfig.Entity
	var entity pageConfig.Entity
	result := conn.Where("page_type = ?", pageConfig.SecuritySettings).First(&entity)
	if result.Error == nil {
		copy := entity
		previous = &copy
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		t.Fatalf("read existing security config: %v", result.Error)
	}
	t.Cleanup(func() {
		if previous != nil {
			if err := conn.Save(previous).Error; err != nil {
				t.Errorf("restore security config: %v", err)
			}
		} else if err := conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("delete test security config: %v", err)
		}
		hotdataserve.ClearSecuritySettingsConfigCache()
	})

	security := defaultconfig.GetDefaultSecuritySettingsConfig()
	security.EnableEmailVerification = enableEmailVerification
	security.CaptchaRequired = false
	if err := conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{}).Error; err != nil {
		t.Fatalf("clear security config: %v", err)
	}
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.SecuritySettings, Config: jsonopt.Encode(security)}).Error; err != nil {
		t.Fatalf("persist security config: %v", err)
	}
	hotdataserve.ClearSecuritySettingsConfigCache()
}

// createWritableGateUser 直接落库一个中间件测试用户；activated=false 等价于
// 邮箱验证开启时"注册即发会话"的待激活账号（含 OAuth 待激活存量账号的同一状态）。
func createWritableGateUser(t *testing.T, id uint64, username string, activated bool) {
	t.Helper()
	user := users.MakeUser(username, "secret123", username+"@example.test")
	user.Id = id
	if activated {
		user.IsActivated = users.ActivationSuccess
	}
	if err := users.Create(user); err != nil {
		t.Fatalf("create activation gate user: %v", err)
	}
}

// writableAccountRequest 以给定 userId 直接执行写权限中间件（不经 JWT 链路，
// 聚焦中间件自身判定），中间件放行后由空 handler 返回 200。
func writableAccountRequest(middleware gin.HandlerFunc, userId uint64) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userId)
		c.Next()
	})
	router.Use(middleware)
	router.POST("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeMiddlewareResult(t *testing.T, recorder *httptest.ResponseRecorder) component.ResultStruct {
	t.Helper()
	var body component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return body
}

// TestCheckWritableAccountBlocksPendingActivationWrite 复现 issue #404：
// 邮箱验证开启时，待激活账号（密码注册/OAuth 注册即发会话的存量 pending
// 会话）的写请求必须被稳定 403 拦截，返回 permission.emailRequired。
func TestCheckWritableAccountBlocksPendingActivationWrite(t *testing.T) {
	setupWritableAccountGateTest(t, true)
	createWritableGateUser(t, 4101, "pendingwriter", false)

	recorder := writableAccountRequest(CheckWritableAccount, 4101)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("pending write status = %d, want 403: %s", recorder.Code, recorder.Body.String())
	}
	body := decodeMiddlewareResult(t, recorder)
	if body.MessageCode != component.MessagePermissionEmailRequired {
		t.Fatalf("messageCode = %q, want %q", body.MessageCode, component.MessagePermissionEmailRequired)
	}
	if body.Result != nil {
		t.Fatalf("result = %#v, want nil (no account info leak)", body.Result)
	}
	if body.Params["actionCode"] != "write" {
		t.Fatalf("params.actionCode = %#v, want %q", body.Params["actionCode"], "write")
	}
	if body.Params["action"] != "写入" {
		t.Fatalf("params.action = %#v, want 写入", body.Params["action"])
	}
}

// TestCheckWritableAccountAllowsActivatedUserWrite 邮箱验证开启时已激活账号正常放行。
func TestCheckWritableAccountAllowsActivatedUserWrite(t *testing.T) {
	setupWritableAccountGateTest(t, true)
	createWritableGateUser(t, 4102, "activatedwriter", true)

	recorder := writableAccountRequest(CheckWritableAccount, 4102)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activated write status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
}

// TestCheckWritableAccountIgnoresPendingWhenEmailVerificationDisabled
// 关闭邮箱验证时待激活账号写入零回归（判定只在开关开启时生效）。
func TestCheckWritableAccountIgnoresPendingWhenEmailVerificationDisabled(t *testing.T) {
	setupWritableAccountGateTest(t, false)
	createWritableGateUser(t, 4103, "pendinglegacy", false)

	recorder := writableAccountRequest(CheckWritableAccount, 4103)
	if recorder.Code != http.StatusOK {
		t.Fatalf("pending write with verification disabled status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
}
