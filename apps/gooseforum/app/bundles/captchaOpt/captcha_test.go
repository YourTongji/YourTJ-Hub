package captchaOpt

import (
	"testing"
	"time"
)

// TestSubmittedTooFast 验证提交耗时检测：
// 刚签发的验证码立即提交（minSeconds=1）判为过快；minSeconds<=0 关闭检测；
// 未知验证码 ID 不判为过快。
func TestSubmittedTooFast(t *testing.T) {
	id, b64 := GenerateCaptcha()
	if id == "" || b64 == "" {
		t.Fatal("GenerateCaptcha returned empty id/image")
	}
	if !SubmittedTooFast(id, 1) {
		t.Fatal("freshly issued captcha must be reported too fast for minSeconds=1")
	}
	if SubmittedTooFast(id, 0) {
		t.Fatal("minSeconds=0 must disable the timing check")
	}
	if SubmittedTooFast("no-such-captcha-id", 1) {
		t.Fatal("unknown captcha id must not be reported too fast")
	}
}

// TestTimingCheckRecoversAfterWindow 验证耗时检测窗口过后恢复正常：
// 睡眠超过 minSeconds 后，SubmittedTooFast 返回 false，且正常 VerifyCaptcha 路径可校验通过。
func TestTimingCheckRecoversAfterWindow(t *testing.T) {
	id, _ := GenerateCaptcha()
	store.RLock()
	info, exists := store.data[id]
	store.RUnlock()
	if !exists || info.code == "" {
		t.Fatal("issued captcha missing from store")
	}
	code := info.code

	time.Sleep(1100 * time.Millisecond) // 超过 minSubmitSeconds=1

	if SubmittedTooFast(id, 1) {
		t.Fatal("after sleeping past the threshold, must not be reported too fast")
	}
	if !VerifyCaptcha(id, code) {
		t.Fatal("VerifyCaptcha must accept the correct answer after the timing window")
	}
}
