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
逐串扫描 `src/**` 的 `.vue`/`.ts` 引号内 class 串（含 `:class` 三元/条件分支的字面量，
双引号与单引号外层绑定均支持），同一串内 clamp 与裸 display 类共存、且**完整变体前缀
逐字相同**才报错并给出文件/行号。变体前缀含命名变体（`group-hover/item`）与容器查询
（`@md`、`@lg/name`）。前缀不同不报（如 `sm:line-clamp-2` + 裸 `block`，断点下 clamp
源序靠后获胜，属合法写法）；条件分支互斥的类串逐串隔离，不产生误报。

important 修饰符（v3 前缀 `!block` / v4 后缀 `block!`）参与判定：带 important 的声明
确定性覆盖不带的，与源序无关——clamp 带 important 而 display 不带 → clamp 必胜，合法不报；
display 带而 clamp 不带 → 确定缺陷，报错；双方同带 → 回到源序竞争，报错。

注释、文本节点（含英文撇号 don't/can't）、`{{ }}` 文本插值与模板 `${ }` 插值不参与判定。
确需豁免时走脚本内 `lineClampAllowlist` 文件级清单（沿 `audit-site-colors.mjs` 惯例），
每项在 PR 中说明理由并同步更新测试中的空清单断言，不放宽规则。

门禁边界（v1 不覆盖，靠 review 兜底）：

- **跨属性拆分**：静态 `class="line-clamp-2"` 与 `:class="'block'"` 分写同元素，逐串扫描不合并多属性；
- **`:class` 对象/数组语法**：`{'line-clamp-2': true, 'block': true}` 或 `['line-clamp-2', 'block']`
  各键/项独立成串，不合并判定同元素多键同时生效的冲突；
- **跨组件传导**：父组件 `*:…:flex` 变体打子组件根、子组件根自带 clamp，单文件视角不可见；
- 扫描范围不含 `resource/templates` gohtml 模板（当前无 line-clamp 用法）。

粗扫描不做完整词法分析：极端形态（未配平引号、文本节点裸露 `<` 等）可能漏报，
方向保守；发现误报走豁免清单并修正扫描器，不放宽判定规则。

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
