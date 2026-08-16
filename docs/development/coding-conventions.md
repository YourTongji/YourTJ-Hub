# Coding Conventions

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-15

本文档收录代码级约定；仓库级硬约束与边界规则见根 `AGENTS.md`。

## TODO 标记

代码中遗留的已知问题使用三个标记，按紧急程度选择：

| 标记 | 语义 | 发布约束 |
|---|---|---|
| `FIXME` | 应阻塞下一次发布的问题 | 除非评审明确同意，否则发布不应携带未解决的 `FIXME` |
| `TODO` | 应当尽快修复的问题 | 有资源时尽快处理 |
| `XXX` | 也许某天会修复的问题 | 最低优先级，无承诺 |

扫描代码时即可从标记区分「发布阻塞项」与「以后再说项」。新增或关闭标记时随改动一并更新。
