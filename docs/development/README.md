# 开发入口

> 文档类型：开发指南
>
> 状态：Active
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

任何代码、契约、migration、CI 或文档变更都从这里开始。`AGENTS.md` 保存仓库硬约束，本目录
保存可执行流程；不要从历史 PR 或聊天记录复制开发步骤。

## 开始前

1. 阅读根目录 `AGENTS.md`、[文档索引](../README.md)和与需求直接相关的产品/架构/运维规范。
2. 确认请求是只读分析、实现变更，还是还明确授权 commit/push/开 PR。
3. 检查 branch、worktree 和未提交内容；不得覆盖或提交他人改动。
4. 从 `origin/main` 创建 feature/fix/docs branch。
5. 写出 change impact：backend、web、contract、migration、auth/PII、search、deploy、docs。

仓库级 `$yourtj-development` skill 位于 `.agents/skills/yourtj-development`，用于统一上述流程、
验证和交付。

## 标准工作流

```text
需求与产品语义
  -> 影响与风险边界
  -> contract/migration（如需要）
  -> domain/repository/service/http 实现
  -> focused tests
  -> scope-wide CI-parity checks
  -> 文档影响与 diff review
  -> commit/push/PR（仅在明确授权后）
  -> CI + preview 验证
```

## 详细指南

- [本地环境](local-development.md)
- [测试策略与命令](testing.md)
- [分支、提交与 Pull Request](pull-requests.md)
- [文档治理](documentation.md)
- [契约、数据与派生投影](../architecture/contracts-and-data.md)

## 完成定义

- 产品语义、权限、失败/恢复、隐私与保留没有未说明的空白。
- 代码在正确层（repository/service/http），OpenAPI/生成类型/migration 与实现一致。
- 数字 ID 约束（uint64 sub）未被绕过；认证仍以 Casdoor 为唯一来源。
- 文档状态词已更新；契约变更已同步生成物与 fixture。
- 实际运行的命令与结果已报告；本地子集不等于 CI 通过。
