package middleware

import (
	"net/http"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/jsonopt"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
)

// TestRateLimitConfigHotReload 验证配置热更新生效链路：
// 管理面板保存新配额（写入 page_config）→ 清除缓存 → 下一次请求按新配额执行。
func TestRateLimitConfigHotReload(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page config table: %v", err)
	}
	// 清理历史配置，回到默认。
	conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()

	// 自定义配置：topic.write 每 IP 窗口内仅 1 次。
	custom := pageConfig.RateLimitConfig{
		Enabled:   true,
		SkipAdmin: false,
		Actions: []pageConfig.RateLimitRule{
			{Action: RateLimitTopicWrite, WindowSeconds: 60, LimitPerIp: 1, LimitPerUser: 0},
		},
	}
	conn.Create(&pageConfig.Entity{PageType: pageConfig.RateLimitSettings, Config: jsonopt.Encode(custom)})
	hotdataserve.ClearRateLimitConfigCache()

	first := rateLimitRecorder(RateLimit(RateLimitTopicWrite))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200 under new limit", first.Code)
	}
	second := rateLimitRecorder(RateLimit(RateLimitTopicWrite))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429 under new limit", second.Code)
	}

	// 恢复现场：删除自定义配置并清缓存，避免影响同包其他测试。
	conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()
}
