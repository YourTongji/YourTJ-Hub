# Netlify 状态站部署

> Doc type: deployment tutorial
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-15

## 应用边界

**Current**：`apps/status` 是独立 Vue/Vite 应用，Netlify Functions 负责采集与快照读取，
Blobs 保存公开汇总。它不依赖论坛进程、数据库、登录态或论坛静态文件；论坛侧栏只提供外链。
Uptime Kuma 在独立于论坛的主机部署。产品口径见[运行状态](../product/server-status.md)。

**Partial**：仓库具备构建、函数、契约和测试；实际 Netlify 项目、域名绑定及云端定时
执行需要按以下步骤配置并验收。代码验证不代表域名或云端任务已上线。

## 1. 导入 GitHub 项目

在 Netlify 选择 Add new project → Import an existing project，连接
`YourTongji/YourTJ-Hub`，按下表设置：

| 项目 | 值 |
|---|---|
| Production branch | `main` |
| Base directory | `apps/status` |
| Package directory | 留空；本应用的 base 就是独立 pnpm workspace |
| Build command | `pnpm build` |
| Publish directory | `dist`（相对于 base，界面也可能显示完整路径 `apps/status/dist`） |
| Functions directory | `netlify/functions`（相对于 base，由配置文件声明） |
| Node.js | `24` |
| pnpm | `11.11.0` |

配置文件是 [`apps/status/netlify.toml`](../../apps/status/netlify.toml)。设置 base 后 Netlify
应显示加载了此文件，而不是论坛的构建目录。不要选择整个论坛的 Vite 入口。

沿用仓库的发布约定：功能 PR 合入 `dev`，通过既有发布流程进入 `main` 后，由 Netlify
独立构建状态站。PR 使用 Deploy Preview。首次导入时，生产分支必须已经包含
`apps/status`；未发布的功能可通过预览验收，避免将缺少应用目录的分支作为首次生产构建输入。

## 2. 配置 Functions 环境变量

打开 Project configuration → Environment variables。下面是 YourTJ 的公开来源配置；
环境变量作用域选择 **Functions**，上下文选择 **Production**。无需 Umami 管理员账号、
管理员密码或额外 Blobs token；Blobs 使用 Netlify 函数运行时身份。

| 变量 | YourTJ 配置 |
|---|---|
| `STATUS_ENABLED` | `true` |
| `UMAMI_URL` | `https://umi.yourtj.de` |
| `UMAMI_SHARE_ID` | `ppYkIEzslggfAk81` |
| `KOMARI_URL` | `https://km.ryusel.com` |
| `KOMARI_NODE_ID` | `e643a364-0372-43b2-a341-b05d858866ad` |
| `UPTIME_URL` | `https://uptime.mortis.de5.net` |
| `UPTIME_SLUG` | `a` |

来源 URL 只能是 HTTPS origin，不带路径、查询串或用户凭据。设置环境变量后重新部署使
函数获得新配置。通用 `.env.example` 默认关闭且来源为空；不要把这些变量改成 `VITE_*`。

部署预览如需真实数据，可给 Deploy Previews 上下文设置同样的公开来源变量，然后手动
运行两项采集函数。它使用按部署 ID 隔离的 Blobs store，不读取或覆盖正式快照。
只给 Production 配置变量时，预览显示「尚未连接」属于正常行为。

## 3. 首次采集与检查

发布完成后，在 Functions 页面确认三项函数：

- `collect-current`：每分钟采集 Uptime 可用性与 Komari 当前资源。
- `collect-history`：每五分钟采集四个资源范围和三个访问统计范围。
- `status`：`GET /api/status`，只读取快照，不触发上游采集。

对两个采集函数分别点击 **Run now**，然后打开已分配的 Netlify 站点域名查看页面。
首次采集之前显示「暂时不可用」；history 采集之前可以显示当前资源但缺少曲线。
定时函数仅对 published deploy 自动运行；Deploy Preview 和本地环境要手动触发。

每个来源请求总预算八秒，来源独立失败。Blobs 成功快照跨正式发布保留，强一致读取与
条件写避免旧采集覆盖新数据；来源配置变更会切换快照键，旧来源不会继续展示。
正式采集不由外部 URL 触发。数据更新无需重新构建或发布页面。

## 4. 绑定 status.yourtj.de

1. 在 Netlify 项目的 Domain management 添加 `status.yourtj.de`，设置为 primary domain。
2. 在现有 `yourtj.de` DNS 提供商添加 **CNAME**：主机记录 `status`，目标填写 Netlify
   界面为该项目实际给出的 `.netlify.app` 域名。无需迁移整个域名的 DNS。
3. 等待域名验证完成，确认 Netlify 的 HTTPS 证书已签发；Netlify 管理证书续期。
4. 打开 `https://status.yourtj.de`，确认资源、`/api/status` 和范围切换使用同一域名。

不要猜测 Netlify 站点名称或 CNAME 目标；以创建项目后 Domain management 提供的记录为准。

## 5. 故障与发布验收

- 确认两个函数按计划继续产生新快照，而非仅 Run now 成功。
- 模拟论坛连接不可用时，状态站静态页面及 API 仍正常提供服务。
- 单个来源超时，其余面板仍可读取。刷新旧快照不能把原始采样时间更新为当前时间。
- 当前资源／可用性获取时间超过 150 秒、统计／历史超过十分钟即过期；全部来源最长
  展示十五分钟前的成功数据，超过后显示不可用。最新检测与主机采样时间也独立判断。
- 切换 `1h/6h/24h/7d` 与 `24h/7d/30d`，确认曲线所属范围和时间正确。
- 正式部署后从目标用户网络检查域名、API 延迟、移动布局与明暗主题。

Functions 日志只用于诊断执行失败，不应记录上游原始响应、共享 token 或访客数据。
采集器失败应结合页面的 `fetchedAt` 与 Functions 日志排查。关闭 `STATUS_ENABLED` 并
重新部署可停止外部采集与公开旧快照读取；保留的旧 Blobs 数据可在控制台清除。

Netlify 的 compute、带宽、请求和正式部署使用套餐额度。按当前额度规则，团队额度耗尽
会暂停其站点；正式运行应核对套餐并配置额度通知，按需选择付费额度或自动充值。
本应用不更改账户计费设置。

## 官方参考

- [Monorepo 构建目录](https://docs.netlify.com/build/configure-builds/monorepos/)
- [Functions 环境变量](https://docs.netlify.com/build/functions/environment-variables/)
- [定时函数限制与手动执行](https://docs.netlify.com/build/functions/scheduled-functions/)
- [Blobs 持久化与一致性](https://docs.netlify.com/build/data-and-storage/netlify-blobs/)
- [CDN 缓存](https://docs.netlify.com/build/caching/caching-overview/)
- [外部 DNS 子域名配置](https://docs.netlify.com/manage/domains/configure-domains/configure-external-dns/)
- [HTTPS 证书](https://docs.netlify.com/manage/domains/secure-domains-with-https/https-ssl/)
- [额度耗尽与恢复](https://docs.netlify.com/manage/accounts-and-billing/billing/resume-paused-projects/)
