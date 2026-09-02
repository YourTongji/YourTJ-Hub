package routes

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"gorm.io/gorm"
)

// persistSecurityPolicyConfig 持久化一份安全策略配置（保留/禁用名单、敏感词），
// 并在 cleanup 时恢复上一份配置（或删除测试行），避免共享内存库跨用例串扰。
// 每次写入后清空安全设置缓存，使策略检查读到最新配置。
func persistSecurityPolicyConfig(t *testing.T, conn *gorm.DB, cfg pageConfig.SecurityAndRegistration) {
	t.Helper()
	var previous *pageConfig.Entity
	var entity pageConfig.Entity
	result := conn.Where("page_type = ?", pageConfig.SecuritySettings).First(&entity)
	switch {
	case result.Error == nil:
		copy := entity
		previous = &copy
	case result.Error == gorm.ErrRecordNotFound:
		// 无既有行，cleanup 删除测试行即可
	default:
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
	// persistHTTPContractConfig 用 FirstOrCreate：securitySettings 行已存在
	// （共享 harness 已插入默认行）时原地 Update、保留主键；无行时新建。
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, cfg)
	hotdataserve.ClearSecuritySettingsConfigCache()
}

// emptySecurityConfig 返回无名单/无敏感词的基准安全配置（显式空数组，
// 不受 defaultconfig 内嵌默认词库影响——测试只关心被显式配置的条目）。
func emptySecurityConfig() pageConfig.SecurityAndRegistration {
	return pageConfig.SecurityAndRegistration{
		EnableSignup:            true,
		EnableEmailVerification: false,
		AllowedDomains:          []string{},
		ReservedUsernames:       []string{},
		BannedUsernames:         []string{},
		SensitiveWords:          []string{},
		SensitiveAction:         "block",
		CaptchaRequired:         false,
	}
}
