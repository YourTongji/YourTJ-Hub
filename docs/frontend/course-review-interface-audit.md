# 课评界面审计基线（Course Review Interface Audit）

> Doc type: interface audit baseline
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-12

对应 Epic #172（PRD §7.2 + A6）与 issue #180：对课评 3 个页面（Catalog / Detail / Moderation）做
better-interface `full` 模式六域审计，产出 findings 表；F1–F3 改造后复测；`docs/product/current-state.md`
同步更新。

## Scope and Coverage

- Mode: `full`（六域全检，含空态/加载/错误/窄屏状态；finding 上限 15）
- Scope: `CourseCatalogPage.vue`（/courses）、`CourseDetailPage.vue`（/courses/:courseId）、
  `CourseReviewModerationPage.vue`（/moderation/course-reviews），以及它们共享的
  `PageHeader.vue` / `EmptyState.vue` / `UserAvatar.vue` 与全局 tokens（`styles/tokens.css`）。
- Stack: Vue 3.5 + Tailwind v4（OKLCH 语义 token）+ vue-i18n（zh/en/ja/it）+ Lucide icons；
  设计系统为 `gf-*` 组件类（components.css）。

| Domain | Evidence inspected | Result |
|---|---|---|
| Accessibility | 3 页面全部控件（键盘路径、aria 属性、表单、弹窗、焦点）、`AppShell.vue:568` main 地标、`app.gohtml:43` `#goose-app` 根节点 | 3 HIGH + 2 MEDIUM/LOW（HIGH 已修复） |
| Layout | 三页响应式网格、弹窗、窄屏断点、内容列宽（`gf-shell-content`）、按钮间距 | 1 MEDIUM |
| Writing | zh/en/ja/it 四语种文案（`locales/*.ts` 课评段）、按钮/空态/错误文案 | 1 MEDIUM（已修复） |
| Typography | 字号阶梯（11–15px）、truncate/line-clamp、tabular-nums、行高 | 2 LOW |
| Colors | OKLCH token 对比度实测（脚本换算 WCAG 比值）、星星填充/未填充状态 | 1 MEDIUM + 1 LOW |
| UI | 按钮/徽章/圆角/过渡、motion-reduce、图标状态 | 1 LOW（已修复） |

## Findings

| # | Severity | Domain | Location | Before | After | Why |
|---|---|---|---|---|---|---|
| 1 | HIGH | Accessibility | `CourseDetailPage.vue:530`–591（举报弹窗）、`CourseReviewModerationPage.vue:353`–415（揭示弹窗） | `role="dialog"` + 遮罩，但无焦点移入/陷阱、无 Escape、关闭后焦点不还原、背景仍可 Tab 到达 | 新增 `site/composables/useModalDialog.ts`：打开时焦点移入首个表单控件、Tab/Shift+Tab 循环、Escape 关闭、关闭后还原焦点、打开时 `#goose-app` 置 `inert` | F1。弹窗缺焦点管理：键盘用户会 Tab 进背景、读屏可读到弹窗外内容、关闭后焦点丢失（WCAG 2.1.2 / APG dialog pattern） |
| 2 | HIGH | Accessibility | `CourseDetailPage.vue:424`（表单错误）、`CourseReviewModerationPage.vue:192`、`:389`（页/弹窗错误） | 错误仅以 `text-error` 颜色渲染，无 `role="alert"`、无 `aria-invalid`/`aria-describedby`，校验失败不聚焦首个无效字段 | 错误加 `role="alert"`；offering/rating/content 三字段加 `aria-invalid` + `aria-describedby="course-review-form-error"`；`submitForm` 校验失败聚焦对应控件（`CourseDetailPage.vue:129`–177） | F3。表单校验错误未被读屏播报，用户不知道为何提交失败（WCAG 4.1.3 / better-accessibility「Errors that announce」） |
| 3 | HIGH | Accessibility | `CourseDetailPage.vue:383`–398（星级评分） | 星星是 `<button>` + `aria-label="N 星"`，但无选中状态；评分组无可访问名称；未选中有无提示 | 星星加 `aria-pressed`；评分容器加 `role="group"` + `aria-label`；未选中时 `sr-only` 播报「未选择评分」 | F2。读屏用户听不出当前选了几星，表单状态不可感知（ARIA pressed state / 表单状态播报） |
| 4 | MEDIUM | Accessibility | `CourseReviewModerationPage.vue:177`（改造前） | 页面自渲 `<main>`，嵌套在 `AppShell.vue:568` 的 `<main>` 内 | 改为 `<div class="min-w-0 pb-8">` | 重复 main 地标破坏读屏地标导航（better-accessibility「Structure is navigation」）。已修复 |
| 5 | MEDIUM | Colors | `tokens.css:7`–8 派生：`text-base-content/45`、`/55`（如 `CourseCatalogPage.vue:106,110,117`、`CourseDetailPage.vue:277,300–301,315,473–475`、`CourseReviewModerationPage.vue:239,244–245,276,348,376`） | 小字号弱化文本：light `/45` = 3.34:1、dark `/45` = 3.09:1、dark `/55` = 4.18:1，均低于 4.5:1 | 信息性文本至少用 `/65`（light 6.94:1 / dark 5.55:1 达标）；`/45` 仅保留给纯装饰 | WCAG 1.4.3 AA 小字号 4.5:1。属全局 token 系统性调整（需同步 mobile `tokens.json`，AGENTS.md 硬约束），本基线只记录，不在此 PR 动 token |
| 6 | MEDIUM | Layout | `CourseReviewModerationPage.vue:230`–249（队列行）、`app/http/controllers/forum/moderation.go:624` | 审核队列行无「查看被举报评价」入口；审核日志 snapshot 的 `TargetURL` 指向 `/courses/reviews/:id`，而该路由不存在（`route4api.go:103`–104 仅 `/courses`、`/courses/:courseId`） | 队列行加评价链接（指向 `/courses/:courseId`），或注册 `/courses/reviews/:id` 路由 | 审核员无法从队列跳转核对原文，且日志存在死链。需产品确认路由形态，留待后续 |
| 7 | MEDIUM | Writing | `locales/zh.ts:413`（及 en/ja/it 同键，改造前） | resolved 标签为「已隐藏 / Hidden / 非表示済み / Nascoste」 | 「已处理 / Handled / 処理済み / Gestite」 | 该 tab 含「隐藏」与「恢复显示」两类已处理举报（`moderation.go:352`–360），原词义不准确、误导审核员。已修复 |
| 8 | LOW | Accessibility | `CourseCatalogPage.vue:17`–24（改造前） | 页头徽章只显示裸页码数字 | 加 `sr-only` 「第 N 页 / Page N」标签 | 读屏把裸数字读成无意义内容（better-accessibility「Accessible names」）。已修复 |
| 9 | LOW | Typography | `CourseCatalogPage.vue:104,110,117`、`CourseReviewModerationPage.vue:230,275` | `text-[11px]` 徽章/元信息低于 12px 地板 | 提到 `12px` | 小字号可读性（better-typography「Size and Contrast Floors」）。留待后续 |
| 10 | LOW | Colors | `CourseDetailPage.vue:394,470` | 未填充星 `text-base-content/20` = 1.60:1（图形对象） | 提到 `/35`（约 2.6:1）或明确依赖冗余线索（`aria-pressed` + 数字读值） | WCAG 1.4.11 非文本 3:1。已有冗余线索（选中态还有数字与 pressed 状态），定为 LOW。留待后续 |
| 11 | LOW | Typography | `CourseReviewModerationPage.vue:244` | 评价摘录 `line-clamp-2` 截断，无 title/展开 | 加 `:title` 或随 #6 提供全文链接 | 截断内容在队列内不可达（better-typography「Truncate Without Losing Content」）。与 #6 关联 |

## Considered but Rejected

| Location | Candidate | Rejected because |
|---|---|---|
| `CourseReviewModerationPage.vue:200`–224 | 把状态 tab 改为 `role="tablist"/tab` + roving tabindex | 项目全局 `gf-tab` 是按钮式 tab（components.css:133）；`aria-pressed` 已表达选中态，引入新 widget 模式反而增加不一致 |
| `CourseDetailPage.vue:405`（删除确认） | 把 `window.confirm` 换成自定义弹窗 | 原生 confirm 可访问且全站一致；换自定义弹窗超出本基线范围 |
| `tokens.css:7` | 直接提高弱化文本 alpha 修 #5 | 跨全站系统改动，且按 AGENTS.md 设计 token 变更必须同步 mobile `tokens.json`；本 PR 只建基线并记录 |
| `CourseDetailPage.vue:383` | 星级改为原生 `<select>`/`range` | 自定义按钮 + `aria-pressed` 保留视觉设计，Tab/Space 键盘路径完整 |
| 课评列表分页 | 给详情页评价列表加分页 | `ReviewListMaxItems = 200`（`review.go:340`）是防无界读取的有意上限；分页属后续 slice 增强，不在 F1–F3 |

## Verification

- `cd apps/gooseforum/resource && pnpm install` — 通过（6.3s，pnpm v11.18.0）。
- `pnpm run typecheck`（client tsc + `vue-tsc --noEmit`）— **exit 0**。
- `pnpm test`（vitest run --dir test）— **14 files / 101 tests passed**。
- `pnpm build`（client build + vue-tsc + vite build）— **exit 0**；仅 @vueuse/core 的 rolldown
  `INVALID_ANNOTATION` 与 chunk-size 既有告警，与本次改动无关。
- 对比度：按 `tokens.css` OKLCH 值做 oklch→sRGB→WCAG 换算（light/dark 两套），数值见 #5/#10。
- **Not verified**：真实浏览器键盘走查与读屏（NVDA/VoiceOver）未在本环境运行；`useModalDialog`
  的焦点行为已按 APG dialog pattern 实现并通过类型检查，仍需人工走查确认。

## Verdict

`Needs changes` — F1–F3 已在本 PR 落地，**HIGH = 0**；剩余 2 MEDIUM（#5 token 对比度、#6 审核跳转/死链）
与 3 LOW 记录在案，留待后续 slice。
