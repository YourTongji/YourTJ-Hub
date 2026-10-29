# Issues, Development & Pull Requests

> Doc type: reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-29

本页是需求准备、Issue、PR 与评审的流程事实源。GitHub 模板负责收集信息，repository skills
负责执行本页标准；不在各处维护另一套要求。Issue 说明问题与预期结果，PR 说明交付行为与验证证据，
合并后的长期行为归属 [docs 中心](../README.md) 中对应的产品、架构或运维文档。

## Scope and proportionality

所有变更都应让维护者理解：为什么做、谁受影响、成功是什么、凭什么认为完成。内容深度按影响选择：

| 变更类型 | 需求与调研 | 验收证据 |
|---|---|---|
| 新功能、用户流程或交互调整 | 现状与痛点、相关外部产品调研、目标用户、多视角故事、范围与取舍 | 逐项验收；涉及界面时提供对应端的操作与视觉证据 |
| 缺陷修复 | 受影响角色和任务、环境、复现、实际/预期结果；行为无歧义时无需竞品调研 | 最小失败用例的先红后绿证据、相关边界回归 |
| 重构、依赖、CI、文档等维护 | 当前维护成本/风险、受影响的开发者或运维等、保持不变的契约 | 与变更匹配的检查及行为等价证据；纯文档可人工核对 |

这些类别不降低安全要求；小 diff 也可能改变权限或数据生命周期。功能变更若确无相关外部产品，
写明检索范围与原因，使用适用标准或本仓库证据。维护变更若涉及工具/架构选型，则调研对应方案。
不适用项说明原因即可，不为凑栏目虚构用户、数据或调研，也不要求每个 typo 单独创建 Issue。

收到的 Issue 可以是未完善的反馈。缺陷报告者不必先设计测试，功能提出者也不必完成专业调研；
开发者在实施前按下列标准补齐影响决策的信息。已经充分的用户指令可以直接作为输入，不重复索要确认。

## Issue requirements

### Background and evidence

说明当前行为、使用场景、实际痛点及影响，引用相关代码、文档、复现步骤、反馈或测量结果。
区分已观察事实、推断与待验证假设；没有数据就说明未知，不编造频率、规模或收益。
先解释为什么需要改变，再描述可能的方案。

功能/体验变更应查阅与问题直接相关的外部产品或公开标准。优先官方文档、可操作的产品流程、
正式设计指南及源码，记录来源链接、查阅日期、产品版本/平台（可获得时）。对比具体机制及取舍，
说明校园社区场景中采纳什么、不采纳什么、为什么。产品首页和截图本身不构成调研结论，
其他产品的做法也不自动成为本项目需求。可复用仍适用的已有调研，并核对其时效性。

建议使用紧凑对照表：

| 来源、日期与版本/平台 | 观察到的行为/证据 | 适用性与限制 | 本次结论 |
|---|---|---|---|
| 官方说明或实测链接 | 精确到相关流程或机制 | 与本仓库用户、权限、部署约束的差异 | 采纳/调整/不采纳及理由 |

无法访问来源时写明限制及待验证问题，不声称已完成实测。原始调研材料放在不入 Git 的 `research/`；
可复核的结论和公开来源链接进入 Issue/PR，影响持久决策的依据进入 MADR。

### Target users and user stories

列出主用户及实际受影响的其他角色，说明各自的任务、使用端、权限和约束。论坛常见视角包括
访客、内容作者、读者/被互动者、版主、管理员；基础设施的用户可以是开发者、运维或 API 消费者。
按本次影响选取，检查被动受影响者，避免只写笼统的“用户”。仅一个角色受影响时说明依据。

每个相关角色的故事用 `US-1`、`US-2` 等稳定编号，描述“作为什么角色，在什么情境下，
希望完成什么任务，从而获得什么价值”。故事表达目标，不把“增加按钮/接口/表”当作需求本身。
说明角色之间的冲突、权限差异与失败后恢复；无须为无关角色制造故事。

### Goals and scope

写出目标、包含/不包含的行为、涉及的平台，以及依赖和待决事项。方案与需求分开：目标清晰时
选择满足需求的最小完整方案，必要时比较维持现状、复用已有能力与新实现。
成功指标只有在有测量方式或基线时才量化，不为了正式感任意设定指标。

### Acceptance criteria

每条标准使用 `AC-1`、`AC-2` 等编号，并关联故事或缺陷复现。写成“前置条件 → 操作/事件 →
可观察结果”（Given/When/Then 也可），标明适用角色与平台。覆盖本次改变的成功、权限/拒绝、
边界、失败恢复及兼容行为；性能或可访问性目标要有可检验的条件。
“体验良好”“代码已完成”“测试通过”不能单独作为行为验收标准。

例如，下列是**假设的举报功能**的表达示例，不代表本仓库已实现或本次要实现：

- `US-1`：作为举报者，我希望提交后知道举报已被受理，从而无需反复提交确认。
- `US-2`：作为版主，我希望查看自己负责板块的待处理举报，以便履行管理职责并遵守权限边界。
- `AC-1 → US-1`：已登录用户提交有效举报且服务端保存成功时，界面显示受理结果；保存失败则保留输入并提示重试。
- `AC-2 → US-2`：版主打开负责板块的待处理举报时能看到处理所需信息；请求职责范围外的举报时被拒绝且不返回举报正文。

### Supplementary materials

链接相关 Issue/PR、产品文档、MADR、设计稿、原型、截图/录屏、已脱敏复现材料与依赖。
材料应能被后续 reviewer 访问；不可访问的附件不能作为唯一证据。不要提交凭据、个人数据或原始生产日志。

## Ready for implementation

开始实施前，开发者确认问题、用户、范围和可验证结果足够明确，调研已形成适用性结论，
并识别契约、数据、安全和运维风险。未解决且会改变目标、访问权限或数据语义的问题标为
`Decision needed`，只暂停依赖该决策的工作，提出具体问题；可以继续独立调查和可逆准备。

涉及权限、PII/保留删除、破坏性兼容或迁移、生产发布边界的变更，需要可追溯的风险约定：
在关联 Issue 或 PR 中记录选定行为、边界、失败恢复方式和验收标准，并引用维护者的明确决定。
本仓库 [spec 入口](../specs/README.md) 指向本流程；`Approved spec` 表示这些约定得到确认，
不要求另建一份 spec 文件或使用特定状态标签。已有明确授权覆盖该范围时即可沿用，
不能由代理编造批准，也不因模板存在就增设一次审批。准备风险约定不授权提交或发布。

## Pull Requests

PR 开头说明具体问题及改动后的行为，关联 Issue（部分交付用 `Refs #…`，完整解决才使用关闭关键词）。
无独立 Issue 时在 PR 内补足适用需求信息。沿用 Issue 的故事/验收编号，概括背景、调研结论、
目标用户与故事并链接详情；PR 必须能独立读懂，不能只有一个 Issue 链接。
范围改变时先说明调整的理由并同步需求与验收，不能为了让实现“通过”悄悄降低标准。

使用 [PR 模板](../../.github/pull_request_template.md)，保留原有五个工程栏目，并补充产品视角：

| 栏目 | 要求 |
|---|---|
| **Summary / 摘要**（保留） | 用简短段落说明具体问题、最终改动与价值，关联 Issue |
| **Background & research / 背景与调研**（补充） | 现状、影响、证据、外部方案适用性结论；可概括并链接 Issue 详情 |
| **Target users & user stories / 目标用户与用户故事**（补充） | 受影响角色及任务，关联 `US-*` 或缺陷场景 |
| **Behavior change / 行为变更**（保留） | 改动前后行为、范围、关键取舍、兼容性；持久取舍链接 MADR |
| **Visual change declaration / 视觉变更声明表**（UI 类必填） | 按[声明与对账规则](#visual-change-declaration)逐呈现面列出旧 → 新、移除/弱化元素及范围来源；非 UI 类说明不适用原因 |
| **Acceptance criteria & results / 验收标准与结果**（补充） | 将 `AC-*` 逐项映射到证据，写出通过/失败/未验证/不适用及原因 |
| **Verification / 验证**（保留） | 实际命令、执行环境与结果，区分本地与 CI；UI 变更附对应端操作与视觉证据 |
| **Docs & contract impact / 文档与契约影响**（保留） | 文档/状态词、契约/生成物/fixtures、跨端镜像；不适用说明原因 |
| **Known gaps / 已知缺口**（保留） | 未覆盖项、剩余风险、迁移/恢复限制、待决事项及 reviewer 关注点；确无则写无 |
| **Supplementary materials / 补充材料**（补充） | 设计、截图/录屏、依赖、引用等可访问链接 |

简小变更可以使用简短段落，保留五个工程栏目；产品栏目可概括、链接或说明不适用原因。
验收不适用须有范围依据，失败或未验证不能勾选完成；关键行为缺证据时保留为 draft/未就绪。
PR 本身已有充分信息时
不为了流程追建 Issue。历史 Issue 不批量重写，在继续相关工作时补齐必要信息。

契约变更须包含生成输出和 fixture 更新，其他跨端镜像按 [AGENTS.md](../../AGENTS.md) 同步。
本地检查按 [testing.md](testing.md) 选取，CI 全量结论绑定被评审版本；旧提交的通过结果不能证明新代码。
功能完成同时满足具体 AC 与 [通用完成标准](README.md#definition-of-done)。
不要自行合并自己的 PR（solo repo 且明确允许的例外除外）；至少一次 review，发布还需遵守专门的人工审核。

## Visual change declaration

凡 diff 触及用户可见呈现，PR 必须填写“视觉变更声明表”。判断标准是：**用户能截图圈出的每一个
差别，都能对应声明中的一项**；动效及断点变化按同一原则核对相关状态。纯逻辑/后端、不可见重构、
仅测试文件或无呈现变化的依赖升级不触发；组件换皮、布局、配色、图标/图片、文案排版、动效、
断点及空态/加载态/错误态均触发。按实际影响而非文件后缀或 PR 标签判断；判断不了时算触发。

### Declaration rows

一行对应一个“呈现面”：用户能独立感知的页面、弹层/菜单/sheet，或有独立视觉身份的 widget，
不是一个文件。共享组件改变多个呈现面时列出受影响入口，不能只写组件名。

| 字段 | 要求 |
|---|---|
| 呈现面 | 界面名 + 用户入口；有平台、主题、断点或状态差异时注明适用范围 |
| 变更类别 | 换皮 / 布局 / 动效 / 断点 / 其他，可多选 |
| 特性 delta | 逐项写增/删/改的“旧 → 新”，最小粒度到图标、副标题、选中态样式、选项顺序、禁用态、取消入口和文案格式；同时标注范围来源 |
| 移除/弱化元素 | 列出移除或弱化的元素；确无必须显式写“无”，留空或“样式微调”不是声明 |

已进入实施范围的 Issue 中预先写明的视觉变化，作为维护者默认同意的范围：PR 表可以引用具体
Issue 条目，reviewer 核对实际实现，不重复挑战该取舍。执行中新增或超出 Issue 的视觉变化须
另列一行，标“新增范围”，说明原因，按普通变更评审；无独立 Issue 时也在 PR 中显式声明。
不能把 Issue 未提及的元素丢失解释成默认同意，也不能把本条当作绕过
[风险约定与明确决定](#ready-for-implementation) 的授权。

### Mechanical reskin exception

纯 1:1 换皮（同一特性集换组件）允许批量合并一行，但必须同时满足：

1. 特性栏明确写“特性无 delta”，移除/弱化栏写“无”；
2. 给出完整的呈现面、调用点及文件清单（可链接 PR 附表），而非笼统写“全部调用点”；
3. reviewer 或审查 AI 独立抽查至少三个调用点，不足三个则全部检查，并覆盖不同平台/呈现分支；
   记录抽查位置、旧/新特性及结果。发现任一 delta，整批退回，按呈现面逐行重填，不能只补抽中的一行。

这里的“无 delta”指特性集合不变；换皮本身的外观变化仍由批量行的类别与范围说明，不代表无需视觉验证。

### Reconciliation before code review

代码细读前，reviewer 先遍历完整 diff 清单，独立于作者声明产出“实际呈现变化清单”，再打开声明
逐项对账，避免用作者的表决定只检查哪些变化。清单记录呈现面、旧 → 新及文件/行或视觉证据。
对共享组件追踪受影响调用点；按相关平台、主题、断点及状态核对，无法确认的部分明确标为未验证。

对账输出至少包含：实际清单与声明行的映射、实际变化减去已覆盖声明的差集、与实现不符的声明，
以及批量抽查结果。差集为空只说明声明覆盖，不能替代行为测试或对应端的实机/模拟器视觉验收。

**diff 中用户可感知的变化对不上声明条目 = process blocker**：包括漏报、含糊到无法核对，
或“特性无 delta”与实际不符。指出具体差异并要求补充/纠正声明，或恢复未声明的旧特性；对账后
可解除。它不因新设计本身的取舍而阻塞，也不用于羞辱作者。正确性、安全和验收证据仍按 review contract 独立判断。

规则依据见 [MADR 0116](../decisions/0116-visual-change-reconciliation.md)。首个案例是
[PR #1170](https://github.com/YourTongji/YourTJ-Hub/pull/1170) 中 Home“显示方式”的
`ListTile.leading` 图标在迁移为文本菜单项时丢失：即使菜单动作值保持不变，图标“有 → 无”仍须
声明。共享菜单项的独立 leading 能力由 [Issue #1241](https://github.com/YourTongji/YourTJ-Hub/issues/1241)
单独跟踪；此案例不表示当前 Home 仍有相同缺陷，也不要求恢复已被后续决策替换的呈现方式。

## Review contract

评审先检查需求与证据；UI 类变更在代码细读前按[视觉变更规则](#reconciliation-before-code-review)
独立列出实际呈现变化并与声明对账，再按实际影响检查实现。使用 [repo-review](../../.agents/skills/repo-review/SKILL.md)
和 [yourtj-code-review](../../.agents/skills/yourtj-code-review/SKILL.md)。需要回答：

- 现状有证据吗？调研是否支持所选行为，且适用于本仓库？
- 所有实际受影响角色的故事与权限是否被处理？实现是否满足 AC，有无无关扩张？
- 每项关键验收是否有可信证据？故障、恢复及兼容性是否覆盖实际风险？
- 文档、契约和运维是否匹配最终行为？是否遗留影响交付的决策？

报告分开列出可复现缺陷、缺失的需求/验证证据与非阻塞建议。严重程度以影响为依据：
`blocker` 表示已证实的正确性/安全/硬约束问题，或关键语义/证据缺失导致无法确认安全交付；
视觉变化无法对应声明条目另标 `process blocker`，按上述对账规则补齐即可解除；
`should` 是有依据的非阻塞改进；`nit` 是可选表述或风格建议。不以栏目数量、主观偏好或
未达到 100% 分支覆盖率判定阻塞。未发现问题可明确报告，无需凑数；没有执行过的验证如实标记。

模板和 skills 是人工/代理工作约定；当前确定性 CI 门禁校验文件、链接和既有工程规则，
不校验 Issue/PR 正文是否满足这些语义标准。不得把模板字段已填满等同于需求已成立或验收已通过。

## Branches

- `dev` is the main development line: create `feat/<topic>` / `fix/<topic>` / `docs/<topic>` from
  `origin/dev`, open PRs against `dev`. CI builds and auto-deploys `dev` to the test instance.
- `main` contains reviewed production source. Promote dev with an ordinary PR; release preparation
  never merges it implicitly. [Release requests](../operations/releases.md) are the explicit branch
  exception: `codex/release/<id>` starts from main and targets main with only its release metadata.
  Final-head human review and merge trigger approved platform publication. Never develop directly
  on main or dev.
- The dev instance syncs a consistent snapshot of the main database on each deploy (see
  `docs/operations/deployment.md`), so DB migrations are rehearsed on dev before reaching main.
- Prefer worktrees (`git worktree add`) for parallel tasks; do not mix branches in one checkout.

## Commits

- Conventional Commits: `feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`.
- Stage only files this task owns; leave unrelated dirty/untracked files alone.
- Never push to protected branches; releases go through PR + CI.

## Forbidden

- No `push --force` to shared branches; never change git config.
- No machine-local absolute paths, secrets, raw production logs, personal data or internal addresses in
  commits/PRs/comments. Use repository-relative paths and sanitized minimal evidence.
- No merging production deployments or external changes (unless the user explicitly asks).

## References

以下一手资料用于表单机制、故事表达与评审原则的参考；本页的具体流程是仓库约定，
不声称属于某项认证标准。查阅日期：2026-10-03。

- [GitHub Issue Forms syntax](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)
  — Issue 表单的字段、校验与 Markdown 输出。
- [GitHub pull request templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/creating-a-pull-request-template-for-your-repository)
  — PR 正文模板的机制。
- [Atlassian user stories](https://www.atlassian.com/agile/project-management/user-stories)
  — 用角色、目标、价值表达故事，并配合可检验结果。
- [Google engineering review guidance](https://google.github.io/eng-practices/review/reviewer/looking-for.html)
  — 评审设计、用户行为、有效测试与文档，而非只核查代码风格。
