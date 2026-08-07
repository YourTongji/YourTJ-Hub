package api

import (
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/captchaOpt"
)

// TestCheckCaptchaForRequestSubmitTooFast 验证 controller 级提交耗时检测：
// 刚签发的验证码 + minSubmitSeconds=1 立即提交 → 拒绝且不触发 needCaptcha
// （走 captcha_submit_too_fast 路径，而非前端弹验证码）。
func TestCheckCaptchaForRequestSubmitTooFast(t *testing.T) {
	id, _ := captchaOpt.GenerateCaptcha()
	if id == "" {
		t.Fatal("GenerateCaptcha returned empty id")
	}
	ok, need := checkCaptchaForRequest(nil, id, "wrong-answer", true, 1, "register")
	if ok || need {
		t.Fatalf("fresh captcha: ok=%v needCaptcha=%v, want false/false (submit too fast)", ok, need)
	}
}

// TestCheckCaptchaForRequestAfterTimingWindow 验证耗时窗口过后走正常
// VerifyCaptcha 路径：答案错误被拒（ok=false），但仍不是 needCaptcha。
func TestCheckCaptchaForRequestAfterTimingWindow(t *testing.T) {
	id, _ := captchaOpt.GenerateCaptcha()
	if id == "" {
		t.Fatal("GenerateCaptcha returned empty id")
	}
	time.Sleep(1100 * time.Millisecond) // 超过 minSubmitSeconds=1

	ok, need := checkCaptchaForRequest(nil, id, "wrong-answer", true, 1, "register")
	if ok || need {
		t.Fatalf("after timing window with wrong answer: ok=%v needCaptcha=%v, want false/false", ok, need)
	}
}

// TestCheckCaptchaForRequestUnknownId 验证未知验证码 ID 不触发耗时检测，
// 直接走验证码校验失败（ok=false, needCaptcha=false）。
func TestCheckCaptchaForRequestUnknownId(t *testing.T) {
	ok, need := checkCaptchaForRequest(nil, "no-such-id", "123456", true, 1, "register")
	if ok || need {
		t.Fatalf("unknown captcha id: ok=%v needCaptcha=%v, want false/false", ok, need)
	}
}
