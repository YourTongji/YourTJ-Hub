# Wiki git 同步部署（issue #265）

> Doc type: operations
>
> Status: Active（部署侧配置已落地；同步引擎 issue #260 与 Webhook API issue #261 未落地，相关能力标为 `Planned`）
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-15

## 目标形态

wiki 内容真源为 git 仓库（`YourTongji/YourTJ-Wiki`），论坛内嵌 wiki 页面是**只读投影**：同步引擎
（issue #260）克隆/拉取仓库后写入 wiki 页面/修订，webhook（issue #261）接收推送触发增量同步，
`schedule` 轮询兜底。**同步引擎/Webhook 落地前**，wiki 仍按当前"写即发布"模型运行（见
`docs/architecture/system-overview.md`）。

本文件只记录**已落地**的部署侧配置（`[wiki.git]` 段、镜像 git 能力、init-server 注入）；消费这些
配置的引擎/Webhook 落地后在此补充运行细节。

## 配置项（三处同步）

`[wiki.git]` 段（置于 `[github]` 之后）同时维护在：

- `apps/gooseforum/app/bundles/preferences/config.templ.toml` —— 本地/测试自动生成 `config.toml`
  的来源；`GenerateConfig` 注入随机 `webhook_secret`（`{{.WikiWebhookSecret}}`）。
- `deploy/config.toml.example` —— 服务器模板；`init-server.sh` 用 `.env` 的 `WIKI_WEBHOOK_SECRET`
  替换 `REPLACE_WIKI_WEBHOOK_SECRET` 占位符。
- 本地 `apps/gooseforum/config.toml`（gitignored）—— 首次启动由 `GenerateConfig` 自动生成。

```toml
[wiki.git]
repo = "https://github.com/YourTongji/YourTJ-Wiki.git"
branch = "main"
clone_dir = "./storage/wiki-git"
webhook_secret = "<随机值>"          # 本地由 GenerateConfig 生成；部署由 init-server.sh 注入
schedule = "0 3 * * *"               # 每日 03:00，避开现有 03:03-03:10 定时任务窗口
```

字段说明：

- `repo` / `branch`：真源仓库与同步分支。
- `clone_dir`：仓库克隆目录（相对进程工作目录 `/app`）。
- `webhook_secret`：webhook 验签密钥（issue #261 消费）；本地自动生成，服务器由 init-server.sh
  生成并注入。
- `schedule`：轮询兜底 cron；默认 03:00（与 `db.spec` 备份同刻，PG 部署下 SQLite 备份为空操作），
  与 03:03–03:10 的统计/清理任务错峰。

## clone_dir 与卷布局

容器内 `config.toml` 以 `:ro` 挂载（`deploy/docker-compose.yaml`），**clone_dir 必须落在 storage
卷内**：

- 宿主机：`main/storage/wiki-git`、`dev/storage/wiki-git`（init-server.sh 预创建，uid 1000:1000）
- 容器内：`./storage/wiki-git`（即 `/app/storage/wiki-git`）

## init-server.sh 行为（已落地）

- `.env` 新增 `WIKI_WEBHOOK_SECRET`（`openssl rand -hex 32`；存量服务器缺失时追加）。
- 生成 `main/config.toml` / `dev/config.toml` 时替换 `REPLACE_WIKI_WEBHOOK_SECRET`。
- 存量 config：无 `[wiki.git]` 段时追加整段（含注入 secret）；已有但 `webhook_secret` 为空/
  占位符时仅替换该值（幂等）。
- 预创建 `main/storage/wiki-git`、`dev/storage/wiki-git`，最终 `chown -R 1000:1000 "$ROOT"`。

## 同步账号与权限（`Planned`，待 issue #260）

同步引擎落地后，写入 wiki 页面需要一个同步账号：

- 复用 `wikiImport.go` 的 `resolveUserID` 模式将 git 提交者/配置账号解析为论坛用户（该文件随
  #260 落地）。
- 该账号需对目标 namespace 有**贡献者权限**：管理员在管理端 `/admin/wiki` 的编辑者管理里授权。

## Webhook 与轮询（Webhook `Planned`，待 issue #261）

- webhook 端点落地后，在 GitHub 仓库 Settings → Webhooks 配置推送地址与 secret（=
  `webhook_secret`），实现增量同步。
- 未配置 webhook 时 `schedule` 轮询兜底（默认每日 03:00）。

## 私有仓库凭据注入（如仓库转私有）

`repo` 支持带凭据 URL；凭据与 config.toml 中其他 secret 同等对待（文件权限 600，main/dev 各一份）：

- 推荐：只读 PAT 内联 URL：`https://<user>:<token>@github.com/YourTongji/YourTJ-Wiki.git`
- 备选：把含凭据的 clone 预置到 `storage/wiki-git`（容器内以 app uid 直接 `git fetch`），避免
  凭据进 config.toml。

## 出网 443 实测项

新服务器首次启用同步前，确认容器可出网访问 GitHub：

```bash
docker exec yourtj-main sh -c 'curl -fsSI https://github.com >/dev/null && echo OK'
docker exec yourtj-main sh -c 'git ls-remote https://github.com/YourTongji/YourTJ-Wiki.git HEAD'
```

## 验收自查清单

- [ ] 三处模板含 `[wiki.git]`；本地启动生成的 config.toml 可读该段
- [ ] 镜像含 git：`docker run --rm <image> git --version`
- [ ] init-server.sh 生成 `.env` 的 `WIKI_WEBHOOK_SECRET` 并注入 `main|dev/config.toml`
- [ ] `main|dev/storage/wiki-git` 存在且属主 1000:1000
- [ ] 出网 443 实测通过
- [ ] CI 构建（含 Docker 镜像）通过
