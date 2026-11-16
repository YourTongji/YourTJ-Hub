# Coding Conventions

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-16

本文档收录代码级约定；仓库级硬约束与边界规则见根 `AGENTS.md`。

## line-clamp 与 display 工具类

带 `line-clamp-*`（含 `line-clamp-none`）的元素**不得在同一 class 串内携带其他写
`display` 的工具类**（`block`/`flex`/`inline`/`inline-block`/`inline-flex`/`grid`/`hidden`/
`flow-root`/`table`/`contents` 等）。两者都写 `display` 属性，Tailwind utilities 层内同
specificity 按源序覆盖，行数截断会静默失效（缺陷链 #1639/#1640 → #1677/#1688）。
对齐或可见性需求放父/子层元素承担（先例：`LinkPreviewCard.vue` 把 `hidden`/`block`
放外层 span，clamp 留在内层）。

该约定由静态门禁 `apps/gooseforum/resource/scripts/check-line-clamp.mjs` 机械执行
（`pnpm check:line-clamp`，已并入 `pnpm check`，随 lefthook pre-push 与 CI 前端门禁运行）：
逐串扫描 `src/**` 的 `.vue`/`.ts` 引号内 class 串（含 `:class` 三元/条件分支的字面量），
同一串内 clamp 与裸 display 类共存、且**完整变体前缀逐字相同**才报错并给出文件/行号。
前缀不同不报（如 `sm:line-clamp-2` + 裸 `block`，断点下 clamp 源序靠后获胜，属合法写法）；
条件分支互斥的类串逐串隔离，天然零误报。确需豁免时走脚本内 `lineClampAllowlist`
文件级清单（沿 `audit-site-colors.mjs` 惯例），每项在 PR 中说明理由，不放宽规则。

门禁边界（v1 不覆盖，靠 review 兜底）：

- **跨属性拆分**：静态 `class="line-clamp-2"` 与 `:class="'block'"` 分写同元素，逐串扫描不合并多属性；
- **跨组件传导**：父组件 `*:…:flex` 变体打子组件根、子组件根自带 clamp，单文件视角不可见；
- 扫描范围不含 `resource/templates` gohtml 模板（当前无 line-clamp 用法）。

回归测试见 `apps/gooseforum/resource/test/check-line-clamp.test.ts`（#1688 修复前两条
真实旧类串先红后绿 + 合法写法负例 + 当前 `src` 全量零误报断言）。

## TODO 标记

代码中遗留的已知问题使用三个标记，按紧急程度选择：

| 标记 | 语义 | 发布约束 |
|---|---|---|
| `FIXME` | 应阻塞下一次发布的问题 | 除非评审明确同意，否则发布不应携带未解决的 `FIXME` |
| `TODO` | 应当尽快修复的问题 | 有资源时尽快处理 |
| `XXX` | 也许某天会修复的问题 | 最低优先级，无承诺 |

扫描代码时即可从标记区分「发布阻塞项」与「以后再说项」。新增或关闭标记时随改动一并更新。
