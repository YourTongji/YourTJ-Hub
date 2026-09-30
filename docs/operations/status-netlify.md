# Netlify 状态站部署

> Doc type: deployment tutorial
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-30

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

**Current**：配置通过 `scripts/ignore-build.mjs` 显式决定是否构建。Production 使用 Netlify
内置 `URL` 指向的正式域名读取 `/status-build.json`，仅在记录为 production 且
`apps/status` 的 Git tree 与当前提交完全一致时跳过。该记录随站点产物发布，失败构建和
Deploy Preview 不能推进正式版本。请求超时、记录缺失/无效、无法读取 Git tree 时继续构建。
记录不含配置或凭据，并设置 `Cache-Control: no-store`。

Deploy Preview 与 branch deploy 仍仅在两个不同且可读取的缓存提交之间确认
`apps/status` 无变化时跳过。未知上下文、无缓存和缓存指向当前提交时继续构建。
**仅修改环境变量或需要强制重发**时，使用 Retry with cleared cache，或为构建临时设置
`STATUS_FORCE_BUILD=true`；重发完成后移除该覆盖。生产比较不以预览构建缓存作为上线证明。
脚本只依赖 Node 内置模块，记录比较提交及决定原因。

## 2. 配置 Functions 环境变量

打开 Project configuration → Environment variables。下面是 YourTJ 的公开来源配置；
环境变量作用域选择 **Functions**，上下文选择 **Production**。基础统计无需 Umami
账号或额外 Blobs token；Blobs 使用 Netlify 函数运行时身份。

| 变量 | YourTJ 配置 |
|---|---|
| `STATUS_ENABLED` | `true` |
| `UMAMI_URL` | `https://umi.yourtj.de` |
| `UMAMI_SHARE_ID` | `ppYkIEzslggfAk81` |
| `KOMARI_URL` | `https://km.ryusel.com` |
| `KOMARI_NODE_ID` | `e643a364-0372-43b2-a341-b05d858866ad` |
| `UPTIME_URL` | `https://uptime.mortis.de5.net` |
| `UPTIME_SLUG` | `a` |

**Current**：访客设备分布另需 `UMAMI_USERNAME`、`UMAMI_PASSWORD` 两项服务端秘密变量。
使用有该网站读取权限的专用 Umami 账号；不需要管理员角色。采集器通过登录取得
短期 token，只查询共享 ID 指向的网站的 Breakdown（device/os/browser）和访客总数。
当前适配 `POST /api/reports/breakdown` 报表协议；升级 Umami 时应验证该接口兼容性。
需要交互式二次验证的账号不能用于无人值守采集，采集器不会绕过该验证。
没有这两项变量时面板显示尚未连接，其他统计仍使用公开共享页。
不要扩大共享链接的报表权限；账号密码、登录 token 和原始报表不写入 Blobs 或日志。

来源 URL 只能是 HTTPS origin，不带路径、查询串或用户凭据。设置环境变量后重新部署使
函数获得新配置。通用 `.env.example` 默认关闭且来源为空；不要把这些变量改成 `VITE_*`。

部署预览如需真实数据，可给 Deploy Previews 上下文设置同样的公开来源变量，然后手动
运行三项采集函数。它使用按部署 ID 隔离的 Blobs store，不读取或覆盖正式快照。
只给 Production 配置变量时，预览显示「尚未连接」属于正常行为。

## 3. 首次采集与检查

发布完成后，在 Functions 页面确认四项函数：

- `collect-current`：每分钟采集 Uptime 可用性与 Komari 当前资源。
- `collect-history`：每十五分钟采集四个资源范围和三个访问统计范围。
- `collect-devices`：每小时第七分钟采集三个设备分布范围，与整点历史任务错开。
- `status`：`GET /api/status`，只读取快照，不触发上游采集。

对三个采集函数分别点击 **Run now**，然后打开已分配的 Netlify 站点域名查看页面。
首次采集之前显示「暂时不可用」；history 采集之前可以显示当前资源但缺少曲线。
定时函数仅对 published deploy 自动运行；Deploy Preview 和本地环境要手动触发。

基础来源请求总预算八秒；设备分布的登录、联合报表和人数核对共用十五秒预算，三个
范围共用一次登录并并行读取。来源独立失败。Blobs 成功快照跨正式发布保留，强一致读取与
条件写避免旧采集覆盖新数据；来源配置变更会切换快照键，旧来源不会继续展示。
存储按部署上下文选择：`production` 统一读写 `status-v1`，预览和分支部署使用
`status-preview-v1-<deploy ID>`。不以单次调用的 `published` 标记选择存储，因为定时
调用与 HTTP 调用的标记可能不同；生产部署的独立链接也读取同一份正式快照。
正式采集不由外部 URL 触发。数据更新无需重新构建或发布页面。

## 4. 绑定 status.yourtj.de

1. 在 Netlify 项目的 Domain management 添加 `status.yourtj.de`，设置为 primary domain。
2. 在现有 `yourtj.de` DNS 提供商添加 **CNAME**：主机记录 `status`，目标填写 Netlify
   界面为该项目实际给出的 `.netlify.app` 域名。无需迁移整个域名的 DNS。
3. 等待域名验证完成，确认 Netlify 的 HTTPS 证书已签发；Netlify 管理证书续期。
4. 打开 `https://status.yourtj.de`，确认资源、`/api/status` 和范围切换使用同一域名。

不要猜测 Netlify 站点名称或 CNAME 目标；以创建项目后 Domain management 提供的记录为准。

## 5. 故障与发布验收

- GitHub Release 成功后，确认正式 `/status-build.json` 的 tree 与当前 `main` 的
  `git rev-parse main:apps/status` 一致；状态站无变化时允许跳过新的生产部署。
  有变化时确认新 Production 部署为 **Published**；Deploy Preview 的 Completed 不代表上线。
- 若构建在 `checking build content for changes` 阶段取消，确认加载的是
  `apps/status/netlify.toml`，日志执行了 `node ./scripts/ignore-build.mjs`。出现
  `Skip: status source tree already published in production` 表示正常跳过；其他取消原因
  需排查。仅更新环境变量或需要强制部署时，使用前述强制重建方式。
- 确认三个采集函数按计划继续产生新快照，而非仅 Run now 成功。
- 若采集日志正常但 API 全部返回 `unavailable`，检查 Blobs 的 `status-v1` 下是否有
  新快照；生产采集不应写入当前部署 ID 对应的 preview store。修复部署后运行三项采集
  函数即可重新获取当前数据及上游历史，无需搬移旧 preview store 的短期快照。
- 模拟论坛连接不可用时，状态站静态页面及 API 仍正常提供服务。
- 单个来源超时，其余面板仍可读取。刷新旧快照不能把原始采样时间更新为当前时间。
- 当前资源／可用性获取时间超过 150 秒、统计／历史超过二十分钟、设备超过七十分钟
  即过期；保留上限分别为十五分钟、一小时、三小时。超过后显示不可用，失败采集不能
  更新成功时间。最新检测与主机采样时间也独立判断，不能用低频报表判断服务健康。
- 实时采集失败超过十五分钟时，当前资源卡显示不可用，仍在一小时保留期内的历史
  曲线继续展示。历史没有可用实时采样时也可独立返回；不能因此把当前服务标为正常。
- 切换 `1h/6h/24h/7d` 与 `24h/7d/30d`，确认曲线所属范围和时间正确。
- 设备分布默认 `7d`，切换范围不改变访问趋势或服务器范围；人数应与相同时间窗口的
  联合报表对齐。报表超过 500 行或人数不全时显示覆盖提示。停用／更换账号会隔离旧
  设备快照，报表鉴权失败仅使设备面板过期或不可用，不影响基础访问统计。
- 正式部署后从目标用户网络检查域名、API 延迟、移动布局与明暗主题。

Functions 日志只用于诊断执行失败，不应记录上游原始响应、共享 token 或访客数据。
采集器失败应结合页面的 `fetchedAt` 与 Functions 日志排查。关闭 `STATUS_ENABLED` 并
重新部署可停止外部采集与公开旧快照读取；保留的旧 Blobs 数据可在控制台清除。

Netlify 的 compute、带宽、请求和正式部署使用套餐额度。额度不足可能先暂停生产部署，
剩余 operational credits 仅支持站点继续运行，以账户账单页为准。本应用不改变计费设置。
所有来源启用且不含重试时，每天历史采集 96 轮、设备采集 24 轮、实时采集 1,440 轮，
共写入约 3,624 个快照；实际 credits 取决于函数内存、耗时和访问量，不能按调用次数
直接换算。浏览器轮询间隔为一分钟，API 的 CDN 缓存上限为三十秒。

## 官方参考

- [Monorepo 构建目录](https://docs.netlify.com/build/configure-builds/monorepos/)
- [构建跳过规则](https://docs.netlify.com/build/configure-builds/ignore-builds/)
- [Functions 环境变量](https://docs.netlify.com/build/functions/environment-variables/)
- [定时函数限制与手动执行](https://docs.netlify.com/build/functions/scheduled-functions/)
- [Blobs 持久化与一致性](https://docs.netlify.com/build/data-and-storage/netlify-blobs/)
- [CDN 缓存](https://docs.netlify.com/build/caching/caching-overview/)
- [外部 DNS 子域名配置](https://docs.netlify.com/manage/domains/configure-domains/configure-external-dns/)
- [HTTPS 证书](https://docs.netlify.com/manage/domains/secure-domains-with-https/https-ssl/)
- [额度耗尽与恢复](https://docs.netlify.com/manage/accounts-and-billing/billing/resume-paused-projects/)
