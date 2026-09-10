# 部署配置治理：config as artifact + Environments secrets + 漂移守卫

## Status
Accepted
Class: process

## Context and Problem Statement

实例运行时配置（`main/config.toml`、`dev/config.toml`）此前为服务器本地人工维护：`init-server.sh` 装机生成一次，之后改动全靠 SSH 手改；CI 部署从不触碰 config，配置变更无 review、无审计、无回滚。

实测漂移：生产 `[server].url` 指向已死的 forum.yourtj.de，真实入口 f.yourtj.de 只在 DB 站点设置里；main/dev 模板注释与 example 脱节 60+ 行；GitHub OAuth 凭据等敏感值散落服务器文件。

## Decision Drivers

- 配置变更必须有 review、审计与回滚路径。
- 敏感值绝不进仓库、不进日志。
- 渲染必须 fail-closed：配置错误宁可拒绝部署，不可静默落错值。

## Considered Options

- 模板 + 实例差异 JSON + CI Environments secrets 渲染下发 + 漂移守卫（选定）。
- 配置继续服务器本地维护，仅加人工检查清单。
- 引入外部配置中心（Consul/Vault 类）。

## Decision Outcome

1. 单一配置权威：`deploy/config.toml.tmpl`（全量模板，含注释）+ `deploy/instances/{main,dev}.json`（实例差异非敏感）；敏感值一律由 GitHub Environments secrets（`production`/`dev` 分存）提供，任何值不落仓库、不进日志。
2. 渲染 fail-closed：`deploy/render_config.py`（python3 标准库）——残留占位符/必需 secret 空/TOML 解析失败即拒绝输出；PG_DSN 的 dbname 必须匹配实例 `pg_dbname`（防 dev 误指生产库）；摘要只输出键名/来源/长度，stdout 模式诊断走 stderr。
3. CI 下发 + 同事务回滚：镜像部署（deploy-dev/deploy-main）以 `CONFIG_FILE` 随镜像原子替换；config-only 走 `apply-config.yml`（apply-config.sh：原子锁 `.config.lock` → 备份 .prev → 原子 mv → force-recreate 重挂（单文件 bind-mount 须重建容器才生效）→ 健康检查 → 失败自动恢复）；成功写 `.config.sha256` marker。
4. 域名收口：生产 `[server].url` = `https://f.yourtj.de`（forum.yourtj.de 已死）；init-server 默认域名同步。
5. 漂移守卫：`verify-instance.sh` + `config-drift-check.yml`（每日 + manual）核对 marker sha / server.url / dbname / 必需键。
6. 服务器 config 不再人工 SSH 修改；任何配置变更 = PR（模板/instances）+ Environments secret 更新。

### Consequences

- Good: 服务器 config 由 CI 渲染产物原子写入；人工改动会被 drift-check 告警、下次部署纠正。
- Trade-off: dev 每次从 main 快照 DB → dev 的 DB 站点设置跟随 main，无环境隔离（已知边界，见 [0007](0007-dev-instance-no-env-isolation.md)）。
- 孤儿脚本不收入仓库：服务器 `/opt/yourtj/scripts/merge-wiki-services.sh`（VitePress wiki 时代遗留，无引用）随服务器留存待清理。
- 脱敏键值对照表留存宿主项目 note（Deployment 配置治理 P0 脱敏对照表），不入 git。

## Pros and Cons of the Options

### 模板 + secrets 渲染 + 漂移守卫（选定）
- Good: 配置获得与代码同级的 review/审计/回滚；零新增基础设施依赖。
- Bad: 模板/实例/secret 三处协作，新键需要走 PR + secret 两步。

### 服务器本地维护 + 检查清单
- Good: 零改造。
- Bad: 漂移已实测发生，人工清单不可靠。

### 外部配置中心
- Good: 动态下发、多环境成熟。
- Bad: 引入常驻新组件，违背单二进制 + 少服务的部署形态。

## Links

- PR #385
- 部署文档：docs/operations/deployment.md
- 关联边界：[0007](0007-dev-instance-no-env-isolation.md)
