package pkservice

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// 排课方案云端同步（issue #537）的服务端浅校验。
//
// 深度 sanitize（周次钳制、非法安排丢弃、activePlanId 回退等）是客户端加载
// 路径的职责（web useScheduleStore / mobile schedule_store 各自的 sanitize
// 部落）；服务端只拒绝结构性非法的快照，保证落库数据自洽且体积有界。

const (
	// MaxPlansPerSnapshot 单用户方案数上限（与 web MAX_PLANS / mobile kMaxPlans 一致）。
	MaxPlansPerSnapshot = 10
	// MaxSnapshotBytes 快照整体负载上限：10 套方案 × 数十门课 × 完整教学班详情
	// ≈ 数百 KB，1MB 余量充足（localStorage 本就承载同量级数据）。
	MaxSnapshotBytes = 1 << 20
	// MaxPlanIdLength 方案 id 长度上限：云端列 active_plan_id 为 varchar(64)，
	// 契约 schema 同步声明 maxLength——否则合法契约请求可在 PG 写入时打出 500
	// 而非 400（issue #557 review P2）。
	MaxPlanIdLength = 64
)

// PlanSnapshotPayload PUT /api/pk/plans 的请求负载（GET 响应同构，另含 updatedAt）。
type PlanSnapshotPayload struct {
	Plans         pk.PlanList              `json:"plans"`
	ActivePlanId  string                   `json:"activePlanId"`
	MajorSelected pk.MajorSelectionPayload `json:"majorSelected"`
	WeekView      pk.WeekViewPayload       `json:"weekView"`
}

// ValidatePlanSnapshot 结构浅校验：1..MaxPlansPerSnapshot 套方案、每套非空
// id/name 且 id 长度受列约束、id 不重复、activePlanId 必须命中 plans 之一。
// 错误信息为可读中文，直接作为 PK 信封 BadRequest 的 msg 下发。
func ValidatePlanSnapshot(plans pk.PlanList, activePlanId string) error {
	if len(plans) < 1 {
		return fmt.Errorf("至少需要一套排课方案")
	}
	if len(plans) > MaxPlansPerSnapshot {
		return fmt.Errorf("排课方案数量超出上限（最多 %d 套）", MaxPlansPerSnapshot)
	}
	ids := make(map[string]struct{}, len(plans))
	for _, plan := range plans {
		if strings.TrimSpace(plan.Id) == "" || strings.TrimSpace(plan.Name) == "" {
			return fmt.Errorf("方案缺少 id 或名称")
		}
		if utf8.RuneCountInString(plan.Id) > MaxPlanIdLength {
			return fmt.Errorf("方案 id 过长（最多 %d 字符）", MaxPlanIdLength)
		}
		// 两端客户端都按 id 唯一处理（active 定位/删除命中首个），重复 id
		// 会让同步后的方案歧义或不可达（issue #557 review P2）。
		if _, dup := ids[plan.Id]; dup {
			return fmt.Errorf("方案 id 重复：%s", plan.Id)
		}
		ids[plan.Id] = struct{}{}
	}
	if _, ok := ids[activePlanId]; !ok {
		return fmt.Errorf("activePlanId 未指向任何方案")
	}
	return nil
}

// NormalizePlans 归一化快照内的 nil 切片为空数组：Go JSON 编码 nil 切片输出
// null，而契约（生成 TS 同源）声明这些字段为 required 非空数组——移动端
// 合法上传（courseNature 为 null 或整体省略）会在 GET 响应里产生
// "courseNature": null，schema 校验型客户端直接失败（issue #557 review P2）。
// 落库前统一归一化，保证存储值与 GET 响应恒符合契约。
func NormalizePlans(plans pk.PlanList) pk.PlanList {
	for i := range plans {
		plan := &plans[i]
		if plan.StagedCourses == nil {
			plan.StagedCourses = []pk.StagedCoursePayload{}
		}
		if plan.SelectedCourses == nil {
			plan.SelectedCourses = []string{}
		}
		if plan.CustomEvents == nil {
			plan.CustomEvents = []pk.CustomEventPayload{}
		}
		for j := range plan.StagedCourses {
			course := &plan.StagedCourses[j]
			if course.CourseNature == nil {
				course.CourseNature = []string{}
			}
			if course.Teacher == nil {
				course.Teacher = []pk.TeacherPayload{}
			}
			if course.CourseDetail == nil {
				course.CourseDetail = []pk.CourseDetailPayload{}
			}
			for k := range course.CourseDetail {
				detail := &course.CourseDetail[k]
				if detail.ArrangementInfo == nil {
					detail.ArrangementInfo = []pk.ArrangementPayload{}
				}
				if detail.Teachers == nil {
					detail.Teachers = []pk.TeacherPayload{}
				}
				for m := range detail.ArrangementInfo {
					arrangement := &detail.ArrangementInfo[m]
					if arrangement.OccupyTime == nil {
						arrangement.OccupyTime = []int{}
					}
					if arrangement.OccupyWeek == nil {
						arrangement.OccupyWeek = []int{}
					}
				}
			}
		}
		for j := range plan.CustomEvents {
			event := &plan.CustomEvents[j]
			if event.Sections == nil {
				event.Sections = []int{}
			}
			if event.Weeks == nil {
				event.Weeks = []int{}
			}
		}
	}
	return plans
}

// ValidatePlanSnapshotSize 快照整体序列化体积校验（≤ MaxSnapshotBytes）。
func ValidatePlanSnapshotSize(payload PlanSnapshotPayload) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("方案数据序列化失败")
	}
	if len(encoded) > MaxSnapshotBytes {
		return fmt.Errorf("方案数据过大（超过 %d MB 上限）", MaxSnapshotBytes>>20)
	}
	return nil
}
