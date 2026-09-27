package pkservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// CredentialValidation 一系统凭证校验结果（管理端保存前探测，issue #856）。
type CredentialValidation struct {
	Valid   bool
	Message string
}

// validationProbePageSize 校验时向一系统发起最小探测的页大小：只要证明凭证可用，
// 不需要完整数据（一页一条即可）。
const validationProbePageSize = 1

// validateClientBuilder 可注入的客户端构造器（测试替换为指向本地 httptest 服务的客户端，
// 避免真实抓取一系统；与 syncWith 注入 onesystemClient 同一风格）。
var validateClientBuilder = newOnesystemClientForAudience

// ValidateCredential 预测一系统凭证是否可用（保存前探测，issue #856）。
//
//   - credential 为空时按 ResolveCredentialForAudience 的优先级解析：显式参数 →
//     环境变量 → 管理端已保存设置（securestore 解密）。
//   - 以目标受众最新已同步学期为真实探测目标，最小抓取一页（pageSize=1）：
//     不写库、不写 fetchlog、不触发配置变更或后台任务，仅验证凭证能否通过一系统鉴权。
//   - 一系统侧失败（HTTP 401/403/5xx、业务 code!=0、网络错误等）不是硬错误：
//     返回 Valid=false 与脱敏后的失败说明（客户端 redactCredentials 已剥除凭证片段）。
//   - 硬错误（audience 非法、缺少凭证来源、数据库读取出错）返回 error，由调用方映射为失败信封。
//   - 目标受众尚无已同步学期时无法选择真实探测目标：返回 Valid=false（非 error），
//     提示先同步一个学期。
func ValidateCredential(ctx context.Context, audience Audience, credential string) (CredentialValidation, error) {
	if !audience.Valid() {
		return CredentialValidation{}, fmt.Errorf("无效的一系统数据来源 %q", audience)
	}

	effective := strings.TrimSpace(credential)
	if effective == "" {
		resolved, err := ResolveCredentialForAudience("", audience)
		if err != nil {
			return CredentialValidation{}, err
		}
		effective = resolved
	}

	calendars, err := pk.ListCalendarsForAudience(audience, 1)
	if err != nil {
		return CredentialValidation{}, err
	}
	if len(calendars) == 0 {
		return CredentialValidation{Valid: false, Message: "尚无已同步学期，无法校验一系统凭证：请先对该数据范围同步一个学期"}, nil
	}

	calendarID := pk.ExternalID(audience, calendars[0].CalendarId)
	client := validateClientBuilder(audience)
	if _, err := client.fetchPage(ctx, effective, int(calendarID), 1, validationProbePageSize); err != nil {
		// 客户端已对错误提示执行 AC2 脱敏（redactCredentials），可直接展示。
		return CredentialValidation{Valid: false, Message: err.Error()}, nil //nolint:nilerr // Probe failure is a business result; only setup failures use the error envelope.
	}
	return CredentialValidation{Valid: true}, nil
}
