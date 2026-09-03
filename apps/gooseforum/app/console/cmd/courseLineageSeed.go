package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-lineage-seed",
		Short: "卡级课程沿革候选：装配课程目录可见课程卡，产出/写入 course_relations pending 候选",
		Long: `卡级课程沿革候选（2026 课改治理）。与 course-lineage-scan（教学班级级、
dry-run JSON 输出）互补：本命令在课程目录卡层面（course.id）配对，产出可直接
进入管理端「课程沿革」审核面板的候选。

规则（同教师工号组内配对，跨教师同名/同码卡是合法分班不产候选）：
  E1 EQUIVALENT (conf 0.9)  同师 + 归一名称一致 + 学分一致 + 共享一系统课程码
                            （course_code / new_course_code）→ 冗余卡并入规范卡，
                            可经管理端确认后合并（offering/评价/别名迁移、旧卡隐藏）。
  E2 SPLIT_FROM (conf 0.55) 同师 + 同课程家族 + 变体硬分隔（A1/A2/B、基础/进阶、
                            上/下、实验/理论…）→ 分层标注，绝不合并。
  E3 RELATED (conf 0.2)     同师 + 同家族 + 学分巨变 → 弱关联标注，供人工核查。

默认 dry-run：只装配并输出候选报告，不写库。
  --write        把 E1 EQUIVALENT 候选写入 course_relations（status=pending）
  --write-family 额外写入 E2 SPLIT_FROM / E3 RELATED 候选（仅标注，管理端 approve 展示）

写入幂等：同 (from,to,type) 已存在（含已 approved/ignored/merged）自动跳过，不复活。
建议流程：先 dry-run 看报告 → --write 落 EQUIVALENT → 管理端沿革面板人工确认合并
→ 试点家族（程序设计/高数/概率论族）复核后再 --write-family。`,
		RunE: runCourseLineageSeed,
	}
	cmd.Flags().Bool("write", false, "把 E1 EQUIVALENT 候选写入 course_relations（pending）")
	cmd.Flags().Bool("write-family", false, "额外写入 E2 SPLIT_FROM / E3 RELATED 标注候选")
	cmd.Flags().Bool("json", false, "同时把候选明细（含名称/证据）以 JSON 输出到 stdout")
	appendCommand(cmd)
}

// runCourseLineageSeed 装配卡级候选并（可选）写入 relations，输出报告。
func runCourseLineageSeed(cmd *cobra.Command, _ []string) error {
	write, _ := cmd.Flags().GetBool("write")
	writeFamily, _ := cmd.Flags().GetBool("write-family")
	asJSON, _ := cmd.Flags().GetBool("json")

	report, candidates, summaries, err := courseservice.SeedLineage(cmd.Context(), courseservice.SeedLineageOptions{
		Write:       write,
		WriteFamily: writeFamily,
	})
	if err != nil {
		return fmt.Errorf("course-lineage-seed: %w", err)
	}

	if asJSON && len(candidates) > 0 {
		label := make(map[uint64]string, len(summaries))
		for _, s := range summaries {
			label[s.ID] = s.Name + "（" + s.PrimaryCode + "）"
		}
		type seedItem struct {
			FromCardID   uint64  `json:"fromCardId"`
			FromName     string  `json:"fromName"`
			ToCardID     uint64  `json:"toCardId"`
			ToName       string  `json:"toName"`
			RelationType string  `json:"relationType"`
			Source       string  `json:"source"`
			Confidence   float64 `json:"confidence"`
			Evidence     string  `json:"evidence"`
		}
		items := make([]seedItem, 0, len(candidates))
		for _, c := range candidates {
			items = append(items, seedItem{
				FromCardID:   c.FromCardID,
				FromName:     label[c.FromCardID],
				ToCardID:     c.ToCardID,
				ToName:       label[c.ToCardID],
				RelationType: c.RelationType,
				Source:       c.Source,
				Confidence:   c.Confidence,
				Evidence:     c.Evidence,
			})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].RelationType != items[j].RelationType {
				return items[i].RelationType < items[j].RelationType
			}
			return items[i].FromCardID < items[j].FromCardID
		})
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("course-lineage-seed: 序列化候选: %w", err)
		}
		fmt.Println(string(data))
	}

	mode := "dry-run"
	if write || writeFamily {
		mode = "write"
	}
	fmt.Printf("course-lineage-seed (%s): cards=%d equiv=%d split=%d related=%d inserted(equiv=%d/family=%d) skipped(equiv=%d/family=%d)\n",
		mode, report.CardsLoaded, report.EquivCandidates, report.SplitCandidates, report.RelatedCandidates,
		report.EquivInserted, report.FamilyInserted, report.EquivSkipped, report.FamilySkipped)
	if !write && !writeFamily {
		fmt.Println("dry-run 未写库：确认候选规模后加 --write（EQUIVALENT）或 --write-family（SPLIT/RELATED 标注）落 pending")
	}
	return nil
}
