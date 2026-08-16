---
name: yourtj-simplifications
description: Use when asked to find non-obvious simplification candidates in the yourtj-hub repository — auditing dead, duplicated, speculative, over-built, or hand-rolled-where-a-dependency-exists code — or to fold worthwhile simplification ideas into Agent Notes or TODO/FIXME/XXX markers. Not for implementing changes (route those to $yourtj-development).
---

# Finding YourTJ Hub Simplifications

把「找点能简化的东西」变成有证据支撑的清理候选。这是指导，不是检查表：跟随代码、保持判断力，
宁要几个经过验证的强候选，不要一堆薄猜测。

## Start with repo context

- 读根 `AGENTS.md`（尤其 §3 硬约束与 §4 验证哲学）、`docs/development/testing.md`、
  `docs/development/coding-conventions.md`（TODO 三档语义）。
- 用 `docs/README.md` 事实源表判断领域归属；简化候选若与架构边界（bundles → models → service → http）冲突，需额外证据。
- 本仓库的 Agent Note 记录在 Synergy 原生 note（yourtj-hub ADR note），不在仓库内——简化候选的输出落点是
  note 或代码内 TODO/XXX 标记，不在仓库建 `.agents/notes` 目录。

## What counts as a strong candidate

强候选 = 移除/折叠/降级某样真实存在的东西，且有明确证据表明当前设计成本大于收益：

- 公开方法、事件、配置开关、helper、包、测试产物没有生产消费者。
- 测试或文档是唯一消费者，且其钉住的行为不承重。
- 两份表示镜像同一事实（尤其是持久化与瞬态事件各存一份）。
- 某个 seam 的方法所有实现都必须支持，但没有消费者使用。
- 独立包仅为测试/演示/支持代码存在，增加发布或依赖开销。
- 投机性产品泛化：多会话/会话加载、后台任务名册、实时注册表失效、mid-turn 控制等无产品所有者的设计。
- 手写代码重新实现了维护良好的外部包或语言内置能力，且替换能删掉实现加其专属测试。
- 简化后行为可能略有不同，但新行为依然合理且更容易解释。

薄候选通常不够格：删一个 typo、跑一次静态检查、删一个有意文档化的后端、「这看起来很复杂」而无调用点证据。

## Survey broadly

用户要广度时用并行 subagent 分域调查，要求证据而非猜测。可用分域：

- backend service（`app/service/**`）：重复生命周期/防御机制、双表示、死分支
- models / migration（`app/models`、`app/migration`）：冗余列/索引、无消费者的模型方法
- bundles（`app/bundles/**`）：手写 vs 依赖、无消费者工具
- frontend（`resource/src/**`、`resource/test/**`）：死组件、重复状态、投机抽象
- contract / scripts / tests：包拆分、静态清单、冗余 fixture

不要被第一个好候选挡住；从最大的生产代码 delta 开始。

## Audit trust and lifecycle boundaries

对每个防御性拷贝、freeze、validator、回调捕获，说明值从哪里来、下一手归谁。同进程的类型化调用通常借用
只读值；解析器、配置加载、队列、模型/工具 JSON、持久化文件、worker、进程、wire 解码器拥有或校验自己的
数据。围绕 hostile getter、伪造类型对象、回调替换或同进程交接后变更的测试，是投机契约的证据，不是保留理由。

复杂异步代码画所有权图：每个 sentinel、readiness promise、取消路径、disposer、状态旗标映射到唯一所有者或
转移。多个机制镜像同一 liveness/settlement 事实时，提议一个事务或生命周期控制器；保留保护同步发布与回滚、
回调包含、first-terminal-outcome 仲裁、worker/进程所有权、dispose-to-quiescence 的独立机制。

## Hand-rolled code vs a dependency

引入依赖是合法的简化动作：问协议解析器、framing、重试/退避循环、glob 匹配、diff 引擎等，维护良好的
外部包或语言内置是否已提供。证明依赖交换候选时：

- 读手写实现，说出包覆盖的确切面；包不覆盖的残余语义计入反对并保留在候选里。
- 诚实检查包健康度（维护、采用、传递依赖体积）；Go 标准库与 Flutter 生态有大量内置，先查内置再引依赖。

## Output

- 每个候选给出：位置、证据（调用点/消费者 grep）、建议动作（删/折叠/降级/换依赖）、影响面（契约/迁移/文档）。
- 按强弱排序；强候选写 Agent Note（Synergy 原生 note）或代码内 TODO/XXX 标记，弱候选列表给用户。
- 不实现：发现与建议到此为止，实施交给 `$yourtj-development`。
