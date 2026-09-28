package api

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/captchaOpt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"gorm.io/gorm"
)

func restoreRateLimitSettings(t *testing.T, conn *gorm.DB) {
	t.Helper()
	var previous pageConfig.Entity
	result := conn.Where("page_type = ?", pageConfig.RateLimitSettings).First(&previous)
	hadPrevious := result.Error == nil
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		t.Fatalf("read existing rate limit settings: %v", result.Error)
	}
	t.Cleanup(func() {
		if hadPrevious {
			if err := conn.Save(&previous).Error; err != nil {
				t.Errorf("restore rate limit settings: %v", err)
			}
		} else if err := conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("delete rate limit settings fixture: %v", err)
		}
		hotdataserve.ClearRateLimitConfigCache()
	})
}

// issue #895 回归：空标题的非瞬间请求必须在验证码之前被拒绝。
// 原 bind-time validate:"required" 在 executeValidated 就失败、不触碰验证码；
// 移到 handler 后若排在验证码之后，会先消费验证码再返回 invalidParams，
// 调用方用同一个 captchaId 纠正标题重试时会得到 auth.captcha.invalid。
func TestWriteTopicEmptyTitleRejectsBeforeCaptcha(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page config: %v", err)
	}
	ratelimit.Default().ResetAll()
	t.Cleanup(ratelimit.Default().ResetAll)
	restoreRateLimitSettings(t, conn)

	createTopicWriteUser(t, conn, 9301, "author")
	if err := conn.Create(&category.Entity{Id: 9301, Name: "General", Slug: "general-9301"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	// 所有用户窗口内成功发帖 1 次即要求验证码；MinSubmitSeconds=0 关闭提交耗时检测。
	config := defaultconfig.GetDefaultRateLimitConfig()
	config.NewUserCaptchaAfterPosts = 1
	config.NewUserCaptchaDays = 0
	config.MinSubmitSeconds = 0
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode rate limit settings: %v", err)
	}
	entity := pageConfig.Entity{PageType: pageConfig.RateLimitSettings, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageConfig.RateLimitSettings).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save rate limit settings: %v", err)
	}
	hotdataserve.ClearRateLimitConfigCache()

	recordSuccessfulWrite(9301, "topic.write")

	captchaID, _ := captchaOpt.GenerateCaptcha()
	if captchaID == "" {
		t.Fatal("GenerateCaptcha returned empty id")
	}
	if !captchaOpt.SubmittedTooFast(captchaID, 1) {
		t.Fatal("fresh captcha entry missing from store")
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 9301,
		Params: WriteTopicReq{
			Title:       "",
			Content:     "Article body long enough for the posting rules.",
			CategoryId:  []uint64{9301},
			TopicStatus: 1,
			ContentType: posts.ContentTypeArticle,
			CaptchaId:   captchaID,
			CaptchaCode: "wrong-answer",
		},
	})
	if res.Data.Code == component.SUCCESS || res.Data.MessageCode != component.MessageRequestInvalidParams {
		t.Fatalf("empty-title article response = %#v, want invalidParams before the captcha gate", res.Data)
	}
	// 验证码条目仍在：SubmittedTooFast 依赖 store 中的签发记录，条目被 Verify 消费后会消失。
	if !captchaOpt.SubmittedTooFast(captchaID, 1) {
		t.Fatal("empty-title request consumed the captcha entry")
	}
}
