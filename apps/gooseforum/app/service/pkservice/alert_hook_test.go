package pkservice

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// markFailed 在标记 failed 后必须触发站内告警 hook（issue #855），携带受众与
// 脱敏错误文本；hook 为 best-effort，不影响 markFailed 返回原始错误。
func TestMarkFailedTriggersSyncFailureAlert(t *testing.T) {
	migratePkTables(t)

	log, err := pk.CreateFetchLogForAudience(AudienceGraduate, 501)
	if err != nil {
		t.Fatalf("create fetch log: %v", err)
	}

	type call struct {
		audience string
		message  string
	}
	var got []call
	original := notifySyncFailure
	notifySyncFailure = func(audience, message string) {
		got = append(got, call{audience: audience, message: message})
	}
	t.Cleanup(func() { notifySyncFailure = original })

	syncErr := errors.New("未登录或会话失效")
	if markErr := markFailed(log, syncErr); !errors.Is(markErr, syncErr) {
		t.Fatalf("markFailed returned %v, want original %v", markErr, syncErr)
	}

	latest, ok := pk.LatestFetchLogByAudienceCalendar(AudienceGraduate, 501)
	if !ok || latest.Status != pk.FetchStatusFailed || latest.ErrorMsg != syncErr.Error() {
		t.Fatalf("fetch log not marked failed: ok=%v status=%q error=%q", ok, latest.Status, latest.ErrorMsg)
	}
	if len(got) != 1 || got[0].audience != string(AudienceGraduate) || got[0].message != syncErr.Error() {
		t.Fatalf("alert hook calls = %+v, want one call with audience %q and message %q",
			got, AudienceGraduate, syncErr.Error())
	}
}

// cobra-less 调用方传入的日志对象若缺失受众（空串），告警受众应回落本科，
// 与 alertservice 内 pk.DefaultAudience 归一化口径一致。
func TestMarkFailedAlertAudienceFallback(t *testing.T) {
	migratePkTables(t)

	log, err := pk.CreateFetchLogForAudience(AudienceUndergraduate, 502)
	if err != nil {
		t.Fatalf("create fetch log: %v", err)
	}
	log.Audience = "" // 模拟缺受众的日志对象

	original := notifySyncFailure
	var got string
	notifySyncFailure = func(audience, _ string) { got = audience }
	t.Cleanup(func() { notifySyncFailure = original })

	if err := markFailed(log, errors.New("boom")); err == nil {
		t.Fatal("markFailed should return the original error even with empty audience")
	}
	if got != string(AudienceUndergraduate) {
		t.Fatalf("alert audience = %q, want %q", got, AudienceUndergraduate)
	}
}
