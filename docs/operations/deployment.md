# 部署与发布

> 文档类型：运维
>
> 状态：Active（部署形态已定，具体 runbook `Planned`）
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

## 部署形态

- **单二进制**：`make build` 产出 `bin/yourtj-hub`（web 产物 go:embed）。
- 依赖服务：Casdoor、Meilisearch、PostgreSQL（选型待定）以 compose 编排。
- 生产不引入 nginx/CDN 分离部署；如需反代统一入口，由学校基础设施提供（不改变单二进制）。

## 构建

```bash
make build     # web → webdist → go build → bin/yourtj-hub
```

## 运行

```bash
YOURTJ_ENV=production YOURTJ_PORT=8080 ./bin/yourtj-hub
```

环境变量见 `deploy/env.example`（后续扩展 DATABASE_URL / MEILI_* / CASDOOR_*）。

## 发布流程（Planned）

- CI 构建产物 + 版本 tag（semver）；
- 部署到测试服务器（沿用 YourTJ-Platform 的 PR preview + staging 模式，落地时写 runbook）；
- 数据库迁移：golang-migrate 在启动前执行（落地时定策略）；
- 回滚：保留上一版本二进制 + 迁移向前兼容（append-only migrations）。

## 待补 runbook

- 数据库迁移执行与回滚
- Casdoor 生产配置（域名、证书、client 注册）
- Meilisearch 索引重建、备份
- 日志与监控（server 日志、健康检查探针）
