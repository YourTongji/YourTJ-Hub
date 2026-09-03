package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice/lineage"
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
                            （course_code / new_course_code，含跨字段匹配）→ 冗余卡
                            并入规范卡，可经管理端确认后合并。
  E2 SPLIT_FROM (conf 0.5)  同师 + 同课程家族 + 变体不同（A1/A2/B、基础/进阶、
                            上/下、实验/理论、generic→A1 层次重组）→ 分层标注，绝不合并。
  E3 RELATED (conf 0.2)     同师 + 同变体 + 学分巨变 → 弱关联标注，供人工核查。

方向：优先按开课学期（旧学期 → 新学期）；学期缺失/相等时回退创建时间
（created_at 是摄入序，历史学期补录可能晚于新学期，不可单独作方向依据）。

默认 dry-run：只装配并输出候选报告，不写库。
  --write        把 E1 EQUIVALENT 候选写入 course_relations（status=pending）
  --write-family 额外写入 E2 SPLIT_FROM / E3 RELATED 候选（仅标注，管理端 approve 展示）
  --json         候选明细（含名称/证据）以 JSON 数组输出到 stdout（空候选输出 []，
                 人类可读汇总写入 stderr，stdout 始终是纯 JSON，可被 jq 直接消费）

写入幂等：同 (from,to,type) 已存在（含已 approved/ignored/merged）自动跳过，不复活。
建议流程：先 dry-run 看报告 → --write 落 EQUIVALENT → 管理端沿革面板人工确认合并
→ 试点家族（程序设计/高数/概率论族）复核后再 --write-family。`,
		RunE: runCourseLineageSeed,
	}
	cmd.Flags().Bool("write", false, "把 E1 EQUIVALENT 候选写入 course_relations（pending）")
	cmd.Flags().Bool("write-family", false, "额外写入 E2 SPLIT_FROM / E3 RELATED 标注候选")
	cmd.Flags().Bool("json", false, "候选明细以 JSON 数组输出到 stdout（汇总走 stderr）")
	appendCommand(cmd)
}

// seedItem --json 单条候选明细。
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

// runCourseLineageSeed 装配卡级候选并（可选）写入 relations，输出报告。
//
// stdout 契约：--json 时 stdout 只含候选明细 JSON 数组（空候选输出 []），人类可读
// 汇总写入 stderr，保证 `course-lineage-seed --json | jq` 可直接消费（review C5）；
// 非 --json 时 stdout 为人读汇总。
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

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if asJSON {
		if err := writeSeedJSON(out, candidates, summaries); err != nil {
			return err
		}
		// 人类可读汇总进 stderr：stdout 保持纯 JSON。
		if err := writeLine(errOut, seedSummary(mode(write, writeFamily), report)); err != nil {
			return err
		}
		if !write && !writeFamily {
			if err := writeLine(errOut, "dry-run 未写库：确认候选规模后加 --write（EQUIVALENT）或 --write-family（SPLIT/RELATED 标注）落 pending"); err != nil {
				return err
			}
		}
		return nil
	}

	if err := writeLine(out, seedSummary(mode(write, writeFamily), report)); err != nil {
		return err
	}
	if !write && !writeFamily {
		if err := writeLine(out, "dry-run 未写库：确认候选规模后加 --write（EQUIVALENT）或 --write-family（SPLIT/RELATED 标注）落 pending"); err != nil {
			return err
		}
	}
	return nil
}

// writeLine 向 w 写一行并传播错误（errcheck 要求检查 Fprintln 返回值）。
func writeLine(w io.Writer, s string) error {
	if _, err := fmt.Fprintln(w, s); err != nil {
		return fmt.Errorf("course-lineage-seed: 写输出: %w", err)
	}
	return nil
}

// writeSeedJSON 输出候选明细 JSON 数组（空候选输出 []；含课程名标签）。
func writeSeedJSON(w io.Writer, candidates []lineage.CardCandidate, summaries []lineage.CardSummary) error {
	label := make(map[uint64]string, len(summaries))
	for _, s := range summaries {
		label[s.ID] = s.Name + "（" + s.PrimaryCode + "）"
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
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// seedSummary 人类可读汇总行。
func seedSummary(mode string, report *courseservice.SeedLineageReport) string {
	return fmt.Sprintf("course-lineage-seed (%s): cards=%d equiv=%d split=%d related=%d inserted(equiv=%d/family=%d) skipped(equiv=%d/family=%d)",
		mode, report.CardsLoaded, report.EquivCandidates, report.SplitCandidates, report.RelatedCandidates,
		report.EquivInserted, report.FamilyInserted, report.EquivSkipped, report.FamilySkipped)
}

// mode 汇总写库模式名。
func mode(write, writeFamily bool) string {
	if write || writeFamily {
		return "write"
	}
	return "dry-run"
}
