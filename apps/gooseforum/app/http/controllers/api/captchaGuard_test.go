package api

import (
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
)

func TestNewUserCaptchaRequiredThreshold(t *testing.T) {
	ratelimit.Default().ResetAll()
	created := time.Now().Add(-time.Hour) // 7 天窗口内的新用户

	if newUserCaptchaRequired(created, 9001, "topic.write", 3, 7) {
		t.Fatal("expected no captcha before threshold")
	}
	ratelimit.Default().Increment(captchaCountKey("topic.write", 9001), captchaCountWindow)
	ratelimit.Default().Increment(captchaCountKey("topic.write", 9001), captchaCountWindow)
	if newUserCaptchaRequired(created, 9001, "topic.write", 3, 7) {
		t.Fatal("expected no captcha below threshold")
	}
	ratelimit.Default().Increment(captchaCountKey("topic.write", 9001), captchaCountWindow)
	if !newUserCaptchaRequired(created, 9001, "topic.write", 3, 7) {
		t.Fatal("expected captcha at threshold")
	}

	// 老用户不受触发
	old := time.Now().Add(-30 * 24 * time.Hour)
	if newUserCaptchaRequired(old, 9002, "topic.write", 3, 7) {
		t.Fatal("old user should not require captcha")
	}

	// 阈值 0 关闭
	if newUserCaptchaRequired(created, 9001, "topic.write", 0, 7) {
		t.Fatal("threshold 0 disables captcha")
	}
}

func TestCheckCaptchaForRequestFlow(t *testing.T) {
	// 开关关闭：直接放行
	ok, need := checkCaptchaForRequest(nil, "", "", false, 1, "register")
	if !ok || need {
		t.Fatalf("required=false should pass, got ok=%v need=%v", ok, need)
	}
	// 未携带验证码：needCaptcha=true（前端应弹出验证码）
	ok, need = checkCaptchaForRequest(nil, "", "", true, 1, "register")
	if ok || !need {
		t.Fatalf("missing captcha should needCaptcha, got ok=%v need=%v", ok, need)
	}
	// 携带但答案错误：直接失败，不触发 needCaptcha
	ok, need = checkCaptchaForRequest(nil, "nonexistent-id", "123456", true, 0, "register")
	if ok || need {
		t.Fatalf("invalid captcha should fail without needCaptcha, got ok=%v need=%v", ok, need)
	}
}
