# 测试策略与命令

> 文档类型：开发指南
>
> 状态：Active
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

## 原则

- 测试与验证的强度与变更的风险成正比（auth/PII/治理/积分/搜索必须含负例、重放、隐私、
  失败与对账用例）。
- 本地子集不等于 CI 通过；报告实际运行的命令与结果。
- 契约测试（fixture 反序列化）是防契约漂移的第一道防线，契约管线接入后强制。

## 命令

```bash
# 后端
cd apps/server && go vet ./... && go test ./...

# Web
cd apps/web && pnpm typecheck && pnpm build

# 全量
make test

# 契约（管线接入后）
cd packages/api-contract && dart test test/contracts   # 或脚本封装

# 构建冒烟
make build && ./bin/yourtj-hub   # 然后 curl /healthz
```

## 分层

| 层 | 测试类型 | 工具 |
|---|---|---|
| domain | 单元测试（纯逻辑） | go test |
| service | 单测 + 事务用例 | go test + sqlmock 或 testcontainers（落地时定） |
| repository | 集成测试（真实 DB） | testcontainers / 本地 postgres |
| http | handler 测试 | httptest |
| web | typecheck + 组件测试 | vue-tsc + Vitest（页面层接入后） |
| contract | fixture 反序列化 | dart test / jest |
| mobile | widget/unit | flutter test（移动端搭建后） |

## CI 对应

- server.yml：go vet + go test + go build（apps/server/**）
- web.yml：pnpm typecheck + build（apps/web/**）
- contract.yml：openapi 校验 + 生成无 diff + fixture（apps/server/**、packages/**）

## 冒烟验证清单（骨架）

```bash
curl localhost:8080/healthz          # {"status":"ok"}
curl localhost:8080/api/ping         # {"message":"pong"}
curl localhost:8080/                 # SPA index.html（embed）
curl localhost:8080/p/post/123       # 200 SPA fallback
```
