package pk

import "strings"

// Audience 是一系统数据的登录受众，也是 PK 表之间统一的隔离维度。
// undergraduate 保持现有数据和接口的默认语义，graduate 用于研究生课程。
type Audience string

const (
	AudienceUndergraduate Audience = "undergraduate"
	AudienceGraduate      Audience = "graduate"
)

// ParseAudience 将 API/CLI 的受众参数规范化；空值保持历史本科默认行为。
func ParseAudience(value string) (Audience, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "undergraduate", "ug", "本科":
		return AudienceUndergraduate, true
	case "graduate", "grad", "研究生":
		return AudienceGraduate, true
	default:
		return "", false
	}
}

func (a Audience) Valid() bool {
	return a == AudienceUndergraduate || a == AudienceGraduate
}

// 研究生 ID 使用现有 uint64 列的独立命名空间；本科 ID 保持原值，兼容已存链接。
// Bit 52 keeps both namespaces within the exact integer range of browser JSON numbers.
// Upstream numeric IDs must be below this boundary; ingestion rejects larger IDs.
const graduateIDMask uint64 = 1 << 52

func ScopeID(audience Audience, externalID uint64) uint64 {
	if audience == AudienceGraduate {
		return externalID | graduateIDMask
	}
	return externalID
}

func ExternalID(audience Audience, scopedID uint64) uint64 {
	if audience == AudienceGraduate {
		return scopedID &^ graduateIDMask
	}
	return scopedID
}

func DefaultAudience(value string) Audience {
	audience, ok := ParseAudience(value)
	if !ok {
		return AudienceUndergraduate
	}
	return audience
}

func ValidExternalID(id uint64) bool { return id > 0 && id < graduateIDMask }
