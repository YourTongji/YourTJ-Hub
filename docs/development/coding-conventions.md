# Coding Conventions

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-18

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
单遍语境扫描 `resource/src/**` 的 `.vue`/`.ts` 引号内 class 串，报错并给出文件与
**冲突 token 所在行号**（多行 class 串不指偏）。收集范围按语境区分：

- `.vue` 模板：只收集 class 类属性（`class` / `*-class`，如 `img-class`、`enter-active-class`；
  `data-*`/`aria-*` 前缀排除）；`:class` 绑定（双引号或单引号外层）下钻内部字面量，
  三元/条件分支各自成串；`title`/`aria-label`/`data-*`/`:style` 等非 class 属性值不判定；
- `.vue` 的 script/style 块与 `.ts` 文件：纯代码语境，全部字符串/模板字面量参与判定；
  正则字面量整体跳过（其中引号不是定界符），模板 `${ }` 插值区按代码语境递归扫描。

判定规则（同一 class 串内，clamp 与裸 display 类共存时）：

1. **同前缀**：完整变体前缀逐字相同 → 源序竞争，报错。变体前缀含命名变体
   （`group-hover/item`）、容器查询（`@md`、`@lg/name`）与 arbitrary 段 `/` 修饰
   （`data-[state=open]/open:`）。
2. **超集**：display 的变体链段集合 ⊇ clamp 的变体链段集合（含「无变体 clamp +
   断点/状态变体 display」，如 `line-clamp-2 sm:block`）→ display 在 clamp 生效的
   全部语境额外激活，`@media`/状态规则源序靠后覆盖 clamp，报错。
3. **反方向合法**：clamp 变体链更长（`sm:line-clamp-2` + 裸 `block`）→ clamp 在更窄
   语境源序靠后获胜，不报；不相关变体链（`sm:line-clamp-2` + `md:block`）语境不相交，不报。

important 修饰符（v3 前缀 `!block` / v4 后缀 `block!`）参与判定：带 important 的声明
确定性覆盖不带的，与源序无关——clamp 带 important 而 display 不带 → clamp 必胜，合法不报
（同前缀与超集两种形态均豁免）；display 带而 clamp 不带 → 确定缺陷，报错；双方同带 →
回到源序竞争，报错。

注释（HTML/块/行）、文本节点（含英文撇号 don't/can't）、`{{ }}` 文本插值不参与判定；
注释屏蔽与字符串提取共用同一语境状态机，撇号奇偶不影响注释识别。未闭合的字符串/模板
字面量是**结构错误，fail loud**（exit 1 并报位置），不静默跳过文件剩余部分。
文件遍历跳过符号链接、单文件上限 2 MB（CI/pre-push 可信入口加固）。
确需豁免时走脚本内 `lineClampAllowlist` 文件级清单（沿 `audit-site-colors.mjs` 惯例），
每项在 PR 中说明理由并同步更新测试中的空清单断言，不放宽规则。

门禁边界（v1 不覆盖，靠 review 兜底）：

- **跨属性拆分**：静态 `class="line-clamp-2"` 与 `:class="'block'"` 分写同元素，逐串扫描不合并多属性；
- **`:class` 对象/数组语法**：`{'line-clamp-2': true, 'block': true}` 或 `['line-clamp-2', 'block']`
  各键/项独立成串，不合并判定同元素多键同时生效的冲突；
- **跨组件传导**：父组件 `*:…:flex` 变体打子组件根、子组件根自带 clamp，单文件视角不可见；
- 扫描范围不含 `resource/templates` gohtml 模板与 `resource/packages/client` 工作区
  （当前均无 line-clamp 用法）。

粗扫描不做完整词法分析：极端形态可能漏报，方向保守；发现误报走豁免清单并修正扫描器，
不放宽判定规则。

**通用浏览器断言（完备性兜底）**：静态门禁只覆盖源码可见形态；真实级联结果由
`apps/gooseforum/resource/test/helpers/line-clamp-assert.mjs` 在浏览器里以 computed
style 最终裁决——元素携带 `line-clamp-N` 类时 computed display 必须是截断形态
（`-webkit-box`，或新版 Chromium 序列化的 `flow-root`），否则截断已失效。important、
变体、源序、跨组件传导、运行时拼接类名全部已被浏览器算完，按构造覆盖静态门禁的
全部边界（跨属性拆分、对象/数组语法、可叠加变体组合等）。浏览器套件在页面断言完成后
调用 `assertNoLineClampViolations(page, { assert })` 即整页生效；覆盖范围 = 套件渲染到
的页面与状态，未渲染分支不在范围内，与静态门禁互补。已接入：`line-clamp.browser.mjs`
（含自证用例：跨属性拆分盲区必须被抓、合法 clamp 与 line-clamp-none 零误报）、
`link-preview-card.browser.mjs`、`bookmark-excerpts.browser.mjs`（两处历史缺陷现场）；
其余含 line-clamp 组件的套件按同一模式逐步接入。

回归测试见 `apps/gooseforum/resource/test/check-line-clamp.test.ts`（#1688 修复前两条
真实旧类串先红后绿 + 合法写法负例 + 当前 `src` 全量零误报断言 + PR #1693 两轮审查
发现的复现用例：正则字面量、注释撇号、文本裸 `<`、超集判定、data-class 排除、token 行号）。

## TODO 标记

代码中遗留的已知问题使用三个标记，按紧急程度选择：

| 标记 | 语义 | 发布约束 |
|---|---|---|
| `FIXME` | 应阻塞下一次发布的问题 | 除非评审明确同意，否则发布不应携带未解决的 `FIXME` |
| `TODO` | 应当尽快修复的问题 | 有资源时尽快处理 |
| `XXX` | 也许某天会修复的问题 | 最低优先级，无承诺 |

扫描代码时即可从标记区分「发布阻塞项」与「以后再说项」。新增或关闭标记时随改动一并更新。
