# 课评界面六域审计基线

> Doc type: audit baseline
>
> Status: Active（Phase 1 基线；F1-F3 已改造，2026-08-12 复测，HIGH = 0）
>
> Owner: Platform maintainers / Web frontend
>
> Last verified: 2026-08-12
>
> Epic: #172（PRD §7.2 + A6） · Issue: #180

## 范围

better-interface `full` 模式六域审计（accessibility / layout / writing / typography / colors / ui），对象为课评 3 个页面：

| 页面 | 文件 | 行数 |
|---|---|---|
| 课程目录 | `apps/gooseforum/resource/src/site/pages/CourseCatalogPage.vue` | 136 |
| 课程详情 | `apps/gooseforum/resource/src/site/pages/CourseDetailPage.vue` | 544 |
| 课评审核 | `apps/gooseforum/resource/src/site/pages/CourseReviewModerationPage.vue` | 404 |

基线方法：逐页逐域源码审计（对照 PRD §7.2 已知发现逐项核实），findings 按严重度 HIGH / MEDIUM / LOW 分级，每条含 `file:line`、Before/After、Why。改造完成后（F1-F3）按本文档复测，验收目标为 HIGH = 0。

## 审计总览

| 域 | HIGH | MEDIUM | LOW | 合计 |
|---|---|---|---|---|
| accessibility | 2 | 3 | 2 | 7 |
| layout | 0 | 0 | 0 | 0 |
| writing | 0 | 0 | 0 | 0 |
| typography | 0 | 1 | 1 | 2 |
| colors | 0 | 0 | 0 | 0 |
| ui | 0 | 1 | 1 | 2 |
| **合计** | **2** | **5** | **4** | **11** |

六域全部完成检查，无 "Not reviewed"。layout / writing / colors 三域未发现阻断性或改进性 finding（达标项见下文「已达标项核查」）。

## Findings

> 状态列：✅ 已修复（F1-F3，2026-08-12）· ⬜ 待办（记录在案，留待后续 slice）

### HIGH

| # | 严重度 | 域 | 位置 | 问题 | Before → After | Why |
|---|---|---|---|---|---|---|
| A1 ✅ | HIGH | accessibility | `CourseDetailPage.vue:530-591`（举报弹窗） | 举报弹窗（`role="dialog" aria-modal="true"`）无 focus trap：打开后焦点停留在触发按钮，Tab 可穿出弹窗操作背景页面；无初始聚焦、无 `aria-labelledby` 关联标题、无 Escape 关闭、背景无滚动锁定 | Before：打开弹窗 → 焦点留在触发按钮，键盘可 Tab 到背景元素；After（F1，已实现）：打开时焦点移入弹窗首个表单控件，Tab/Shift+Tab 在弹窗内循环，Escape 关闭并归还焦点，标题用 `aria-labelledby` 关联，`#goose-app` 置 `inert` 使背景不可达 | 模态对话框必须闭环键盘导航（WCAG 2.1.2 / 2.4.3）；焦点穿出会导致键盘与屏幕阅读器用户误操作背景内容，属无障碍硬伤 |
| A2 ✅ | HIGH | accessibility | `CourseReviewModerationPage.vue:353-415`（身份揭示弹窗） | 身份揭示弹窗（`role="dialog" aria-modal="true"`）与 A1 相同：无 focus trap、无初始聚焦、无 `aria-labelledby`、无 Escape 处理、无滚动锁定 | Before：同上；After（F1，已实现）：与 A1 相同的模态闭环（焦点圈定、Escape 归还、标题关联、背景 inert），由共享 `site/composables/useModalDialog.ts` 提供 | 同 A1；该弹窗含敏感操作（揭示匿名作者），焦点失控风险更高 |

### MEDIUM

| # | 严重度 | 域 | 位置 | 问题 | Before → After | Why |
|---|---|---|---|---|---|---|
| M1 ⬜ | MEDIUM | accessibility | `CourseDetailPage.vue:179-187,202-207` | 删除确认与错误提示使用浏览器原生 `window.confirm` / `window.alert` | Before：删除评价弹原生 confirm，helpful/删除失败弹原生 alert；After：替换为站内语义化 dialog（复用模态组件），成功/失败用 `role="status"`/`role="alert"` 内联提示 | 原生对话框不可样式化、打断流程、屏幕阅读器语境割裂，且无法聚焦管理；PRD §7.2 已知发现「alert/confirm 替换」核实属实。待办 |
| M2 ⬜ | MEDIUM | accessibility | `CourseReviewModerationPage.vue:199-224` | 举报队列三态切换（open/resolved/rejected）用裸 `<button>` 实现，无 tab 语义：无 `role="tab"` / `aria-selected` / `aria-controls`，无方向键切换 | Before：三个 tab 外观按钮，仅 `@click` 切换；After：补 `role="tablist"`/`role="tab"` + `aria-selected` + 方向键导航 + `role="tabpanel"` | PRD §7.2 已知发现「三态齐全」——状态数据齐全但键盘/读屏语义缺失；tab 模式要求方向键导航（WAI-ARIA Tabs Pattern）。部分缓解：已补 `aria-pressed` 表达选中态（2026-08-12），完整 tablist 模式留待后续 |
| M3 ⬜ | MEDIUM | accessibility | `CourseDetailPage.vue:335-337,424` · `CourseReviewModerationPage.vue:189,389` | 异步状态无 live region：评价加载/空态（EmptyState `loading`）、表单错误 `<p>`、举报错误 `<p>` 均为静态文本，屏幕阅读器不播报状态变化 | Before：加载中/失败静默切换；After：加载容器加 `aria-live="polite"`（或 `role="status"`），错误提示加 `role="alert"`，提交按钮 `aria-busy` | 异步内容变化需通过 live region 告知辅助技术（WCAG 4.1.3）；否则读屏用户无法感知加载失败与成功。部分缓解：错误提示已加 `role="alert"`、揭示结果加 `role="status"`（2026-08-12）；加载/空态 live region 留待后续 |
| M4 ⬜ | MEDIUM | ui | `CourseDetailPage.vue:374-400` | 星级评分用 5 个独立 `<button>` 实现，无 `radiogroup` 语义、无键盘方向键调节、无 `aria-valuenow` | Before：只能点击/聚焦后按 Enter 逐星；After：改用 `role="radiogroup"` + 5 个 `role="radio"`（`aria-checked`），支持 ←/→ 增减、Home/End 到头尾；或 `input type="range"` 包装 | 评分是单选语义，radiogroup 模式提供标准键盘操作（WAI-ARIA Radio Group Pattern）；当前实现键盘操作低效且读屏播报割裂。部分缓解（F2）：星星补 `aria-pressed`、评分容器补 `role="group"` + `aria-label`、未选中 sr-only 播报（2026-08-12）；保留按钮模式（与全站控件一致），radiogroup/range 改造留待后续 |
| M5 ⬜ | MEDIUM | typography | `CourseDetailPage.vue:222,228,252,268-272,283,328,334-337` 等 | 评价列表/表单大量使用任意值字号与颜色透明度（`text-[13px] text-base-content/55` 等），未走设计 token，字号层级（11/12/13/15px）与页头（`text-xl sm:text-2xl`）无系统化 scale | Before：分散任意值，正文/辅助文本层级靠手写；After：收敛为 token 化文本样式（如 `gf-text-sm/gf-text-xs` 或设计 token scale），同域页面字号一致 | 任意值字体与透明度破坏主题一致性，改主题时难以全局调优；PRD §7.2「评分数字排版」相关排版基准需 token 化（评分数字 `tabular-nums` 已达标，见 T1）。待办 |

### LOW

| # | 严重度 | 域 | 位置 | 问题 | Before → After | Why |
|---|---|---|---|---|---|---|
| L1 ✅ | LOW | accessibility | `CourseDetailPage.vue:544-551` · `CourseReviewModerationPage.vue:367-374` | 弹窗关闭按钮只有 `<X>` 图标，无文本与 `aria-label`，读屏无法识别按钮用途 | Before：`<button><X/></button>`；After（已实现）：加 `:aria-label="t('common.close')"` | 图标按钮必须有可访问名称（WCAG 1.1.1 / 4.1.2） |
| L2 ⬜ | LOW | accessibility | `CourseDetailPage.vue:554-558` | 举报理由 radio 组无 `fieldset`/`legend` 分组（offering 选择已用 fieldset/legend，见下文「已达标项核查」radio aria 行） | Before：5 个 radio 无组语义；After：`<fieldset><legend>` 包裹，或至少 `role="radiogroup"` + `aria-label` | 同组单选控件需分组命名，读屏才能播报完整语境（WCAG 1.3.1）。待办 |
| L3 ⬜ | LOW | ui | `CourseDetailPage.vue:465-472` | 评价列表星级只显示星形图标，无数字评分文本；读屏仅能听到星形（无 aria-label），视觉用户也需数星 | Before：5 个 `<Star>` 图标，`review.rating` 仅驱动高亮；After：旁置 `aria-label="4/5 星"`，或在评分徽标显示 `4.0`（与表单 `tabular-nums` 数字一致） | 评分是核心数据，需同时以文本形式暴露（可感知性）；PRD §7.2「评分数字排版」在列表侧未落实。待办 |
| L4 ⬜ | LOW | typography | `CourseCatalogPage.vue:101-110` · `CourseReviewModerationPage.vue:229,231,235-236,239,251,258,267` | 卡片信息层级全部依赖任意值字号（`text-[15px]/[12px]/[11px]`）表达，无语义标记（如 `<p>`+token） | Before：标题/元信息/徽标字号各异但均手写；After：与 M5 一并 token 化，列表项标题可用语义元素 | 与 M5 同源，归并为排版 token 化工作项。待办 |

## 已达标项核查（PRD §7.2 已知发现逐项）

| PRD 已知发现 | 核查结果 | 证据 |
|---|---|---|
| radio aria（单选可访问性） | ✅ 部分达标 | offering 选择：`<fieldset><legend>` + `<label>` 包裹原生 radio（`CourseDetailPage.vue:350-372`）；评分星已用 `<button>` + `aria-label` + `aria-pressed`（374-400，见 M4）；举报理由 radio 缺分组（见 L2） |
| 弹窗 focus trap | ✅ 已达标（2026-08-12，F1） | 两个弹窗均接入 `useModalDialog`：焦点移入/陷阱/Escape/归还 + `#goose-app` inert（见 A1、A2） |
| 移动端堆叠 | ✅ 达标 | 三页面均使用响应式 grid：目录 `md:grid-cols-2 xl:grid-cols-3`（`CourseCatalogPage.vue:94`）；审核列表 `lg:grid-cols-[...]` 移动端单列 + `lg:hidden` 紧凑元信息（`CourseReviewModerationPage.vue:222,239,251,258`）；详情卡片 `flex-wrap`；无固定宽度溢出 |
| i18n 覆盖 | ✅ 达标 | 4 locale（zh/en/ja/it）三 section 键数完全一致：`coursesPage` 15、`courseDetailPage` 52、`courseReviewModeration` 36（`locales/{zh,en,ja,it}.ts`，2026-08-12 新增 `pageLabel`/`ratingUnselected`/`reportNoteLabel`/`revealReasonLabel` 后仍四语一致）；页面文案全部走 `t()`，服务端硬编码标签经 `authorLabel` 本地化（`CourseDetailPage.vue:41-47`） |
| messageCode 不泄露 | ✅ 达标 | 前端统一 `ApiResponseError`：`message` 为翻译/fallback 文本，`messageCode` 存独立字段不直接渲染（`resource/src/runtime/api.ts:17-20,48-54`；`api-message.ts:40-46`）；页面 catch 仅展示 `error.message` |
| 评分数字排版 | ✅ 部分达标 | 表单评分 `tabular-nums` + `{{ formRating }}.0`（`CourseDetailPage.vue:397`）达标；评价列表无数字评分（见 L3） |
| alert/confirm 替换 | ❌ 未达标 | `window.confirm`/`window.alert` 仍用于删除确认与错误提示（见 M1） |
| 三态齐全 | ⚠️ 部分达标 | open/resolved/rejected 三态数据与切换逻辑齐全（`CourseReviewModerationPage.vue:27,58-66`）；2026-08-12 补 `aria-pressed` 表达选中态，完整 tab 语义（tablist/方向键）仍缺（见 M2） |

## 复测记录（F1-F3 改造后，2026-08-12）

验收门禁：**HIGH（A1、A2）= 0 ✅**。

本次改造关闭/部分关闭项：

- A1、A2（弹窗 focus trap）— 关闭，新增共享 `site/composables/useModalDialog.ts` 接入两弹窗。
- L1（关闭按钮 aria-label）— 关闭。
- F2 附带（对应 M4 部分）：星级 `aria-pressed` + 评分组 `role="group"` + `aria-label` + 未选中 sr-only 播报。
- F3 附带（对应 M3 部分）：表单/页/弹窗错误 `role="alert"`、揭示结果 `role="status"`、offering/rating/content 三字段 `aria-invalid` + `aria-describedby`、校验失败聚焦首个无效字段。
- 附加修复（基线外）：审核页移除嵌套 `<main>` 地标（读屏地标导航）；状态 tab 补 `aria-pressed`；resolved 标签改「已处理/Handled/処理済み/Gestite」（原「已隐藏」与语义不符，该 tab 含隐藏与恢复两类已处理举报）；目录卡片截断字段补 `title`；页头页码 `sr-only` 标签（新增 `pageLabel` 键）。

仍待办（⬜）：M1（alert/confirm 替换）、M2（完整 tab 语义）、M3（加载/空态 live region）、M4（radiogroup/range）、M5+L4（排版 token 化）、L2（举报理由 fieldset）、L3（列表数字评分）；colors 域弱化文本对比度（light `/45`=3.34:1、dark `/45`=3.09:1、dark `/55`=4.18:1 < 4.5:1 AA）与未填充星对比度（`/20`=1.60:1 < 3:1）记录在案，token 变更需同步 mobile `tokens.json`（AGENTS.md 硬约束），属独立 PR。

验证：`pnpm install` 通过；`pnpm run typecheck` exit 0；`pnpm test` 14 files / 101 tests 通过；`pnpm build` exit 0（仅 @vueuse/core rolldown `INVALID_ANNOTATION` 与 chunk-size 既有第三方告警）。真实浏览器键盘/读屏人工走查未在本环境执行，`useModalDialog` 按 APG dialog pattern 实现，需人工复核。

## 复测指引（后续 slice）

1. 复测范围：同一 3 页面，按本表逐条核对 Before → After。
2. 验收门禁：HIGH 必须为 0；MEDIUM 逐条关闭或给出例外理由；LOW 可排入 backlog。
3. 复测工具：键盘走查（Tab/Shift+Tab/Escape/方向键）+ 读屏抽查（评分、tab、弹窗、live region）+ `git diff --check`。
4. 复测结论回填至本文档「Last verified」与 findings 状态列。
