# Cloudflare 状态站部署

> Doc type: operations tutorial
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-04

`Partial`：状态站的 Cloudflare 实现使用 Workers Static Assets、只读 Worker API 和私有 R2 快照。较重的采集
在公开仓库的 GitHub Actions 标准 Linux runner 执行，独立于论坛进程、数据库和静态资源。
页面行为由[状态站规范](../product/server-status.md)维护。

正式读取器、域名和手动公开／设备采集已验证；独立 Cron 调度器仍需凭据配置和连续定时验收。

## 1. Cloudflare 资源

使用 Walker_killer 账户的 Workers Free 和 R2 Standard，不自动升级付费套餐。账户 ID、
公开来源和 bucket 名称由 [`wrangler.jsonc`](../../apps/status/wrangler.jsonc)维护。
使用 Node 24、pnpm 11，进入 `apps/status`：

```sh
pnpm install --frozen-lockfile
pnpm exec wrangler login
pnpm exec wrangler whoami
pnpm exec wrangler r2 bucket list
```

仅当 bucket 不存在时创建 `yourtj-status`、`yourtj-status-preview`，均不启用公共读取。
公开 Worker 使用 R2 binding，不保存 S3 密钥、Umami 密码或调度令牌，也不配置 Cron。
独立的 `yourtj-status-scheduler` Worker 仅定时向 GitHub 发起采集工作流，没有公开 URL、
域名路由或 R2 binding；预览配置不含定时任务。它使用同一 Workers Free 账户。
默认预览的来源关闭并绑定独立 bucket；生产明确使用 `--env production`。

## 2. 采集与凭据

[Collect / status](../../.github/workflows/collect-status.yml) 每十五分钟采集资源、可用性、
历史和流量，每小时采集设备统计。两个任务串行执行，最长五分钟，不上传快照为 Actions
artifact。Cloudflare Cron 按 UTC 触发独立调度器，向固定仓库的固定采集工作流发送 main
`workflow_dispatch`；它不抓取、转换或保存业务统计。响应不是 204、超时或缺少令牌会使
Cron 执行失败，日志仅记录结果类型和 HTTP 状态码。不会自动重发结果不明的 POST。

GitHub 自带定时任务只作为备用，在默认分支 dev 触发；该调度 job 不挂载生产 environment，
只用短期 `GITHUB_TOKEN` 的 `actions: write` 权限向 main 发起 `workflow_dispatch`。
生产采集的 workflow 定义和源码都来自该 main 提交；仅 checkout main 不能隔离 dev 的 workflow。
手动采集也仅允许从 main 运行。仓库变量 `STATUS_COLLECTION_ENABLED=true` 才启用任务；
资源、main 源码和凭据就绪后再开启。此开关不影响公开站点读取已保存快照。
`STATUS_SCHEDULER_ENABLED=true` 允许部署独立调度器，并停用 GitHub 备用调度 job；
它不是已部署 Cron 的停止开关。暂停采集用 `STATUS_COLLECTION_ENABLED=false`，若还要停止
dispatch 请求，则清空调度器的 Cron。不要同时运行两套定时调度。

### 独立调度器配置

1. 为调度器创建专用 GitHub fine-grained token：仅 `YourTongji/YourTJ-Hub` 仓库，
   `Actions: Read and write`，不授予 Contents write、Secrets 或 Administration 权限。
   Actions 权限可触发该仓库其他工作流，GitHub 不支持将此权限收窄到单个工作流；代码固定目标
   不能代替令牌权限边界。使用有限有效期，并在到期前轮换；组织审批完成前不能启用。
2. 将令牌私密写入独立 Worker 的 `GITHUB_DISPATCH_TOKEN` secret，不复制本机 gh 登录令牌，
   不写入公开读取器、仓库文件或前端变量。先从已审核 main 手动运行两种采集并验证全部来源。
3. 首次创建先用无 Cron 的临时配置建立 Worker，再私密写入 secret；确认 secret 名称存在
   后设置 `STATUS_SCHEDULER_ENABLED=true` 停止备用调度，再从已提交、已验证 main 部署
   `pnpm deploy:scheduler`。该命令只部署独立调度器并保留既有 secret；失败则按下述回退
   步骤恢复备用调度，不在未配置凭据时启用 Cron。
4. Cron 配置全球传播可能需要十五分钟。确认至少连续两个公共触发和一次设备触发均实际
   创建 main 工作流、采集成功，且正式 API 原始时间戳推进。部署成功或单次手动采集
   不算定时验收；必须另查 Cron CPU 是否在 Free 额度内。
5. 启用后 `Deploy / status` 同步部署调度器。排障先看 Cron 执行记录、GitHub main 的采集
   记录和 `/api/status` 时间戳。令牌失效时修复权限／轮换，手动补采集可暂时恢复；
   回退前先移除 Cloudflare Cron，再将 `STATUS_SCHEDULER_ENABLED=false` 恢复 GitHub 备用调度。

GitHub `status-collector` environment 配置四个 secrets：

- `R2_ACCESS_KEY_ID`、`R2_SECRET_ACCESS_KEY`：仅目标私有 bucket 的 Object Read & Write 凭据，
  用于 S3 条件写；不授予域名或 Worker 部署权限。
- `UMAMI_USERNAME`、`UMAMI_PASSWORD`：有目标网站只读权限的 Umami 账号。设备采集复用一次
  登录的短期 token，通过 Breakdown 报表只保留类别与人数，不保存 token 或原始报表。

`status-collector` 和独立的 `status-production` environment 均将 Deployment branches and tags
限定为分支 main，不允许 dev 或任何 tag。环境策略独立于 workflow 条件，阻止 dev 修改
workflow 后绕过限制读取生产凭据。不要将秘密放入聊天、日志、仓库或
`VITE_*`。Netlify 标为 secret 的值不可从其 API 导出；必须从凭据原始来源私密重新配置，
不能将 API 的遮蔽值当作密码。需要人工二次验证的账号不适合该采集器。

`UMAMI_DEVICE_REVISION` 是公开缓存版本，不是密码。Worker 和采集器使用相同的值读取设备
快照。撤销设备数据访问时清空该值并重新部署；更换账号／权限时递增该值，避免沿用旧账号
快照，然后重新采集。单纯修改 GitHub 密码不会通知已部署 Worker。设备凭据缺失时设备任务
明确失败，其他公开来源仍独立运行。

## 3. 验证与发布

```sh
pnpm contract:check
pnpm test
pnpm build
pnpm test:browser
pnpm worker:build
pnpm deploy:preview
```

本地 `pnpm dev` 使用 Wrangler 模拟 R2；`pnpm dev:local` 使用本机采集模拟。云端预览应显式
启用公开来源，写入独立预览 bucket，不能使用生产 bucket 当试验数据。采集 CLI 强制显式目标：
`pnpm exec tsx scripts/collect-cloud.ts public preview`（生产为 `public production`，设备任务将
`public` 换为 `devices`）。CLI 的公开来源和设备版本从同一 Wrangler production 配置读取。

[Deploy / status](../../.github/workflows/deploy-status.yml) 仅从 main 发布：状态站或相关
workflow 改动才触发，也可手动重发。仓库变量 `STATUS_DEPLOYMENT_ENABLED=true` 后才运行，
避免合入代码立即接管尚未验收的域名。先运行状态站检查，再构建部署，最后核对正式
`/status-build.json` 的 Git tree 和 API。生产发布串行执行，不取消正在上传的版本。

GitHub `status-production` environment 设置 `CLOUDFLARE_API_TOKEN`，限定目标账户的 Worker
部署、R2 binding 读取，以及 yourtj.de 的路由权限。不要使用 Global API Key，不要把本机
Wrangler OAuth token 复制给 CI，也不要额外开启 Cloudflare Git 自动构建。

首次发布可从干净、已提交且已验证的源执行：

```sh
CONTEXT=production pnpm build
pnpm deploy:production
node scripts/verify-deployment.mjs https://status.yourtj.de
```

构建指纹来自提交的 `apps/status` tree，未提交预览不能作为正式发布证明。

## 4. 域名切换与停用旧平台

删除 Netlify 配置／Functions 的提交进入 main **之前**，必须在旧 Netlify 项目暂停自动构建
（API `build_settings.stop_builds=true`），确认没有排队或正在执行的部署，且已发布部署 ID
保持不变。保留旧部署及其 Functions，直到 Cloudflare 验收与 DNS 切换完成；额度暂停不能
替代这个构建开关。迁移期间不要用不含 Netlify Functions 的源码手动覆盖旧站点。

先验证临时 Worker 域名的页面、安全响应头、`/status` 重定向、API、真实 R2 快照和 CPU。
production Wrangler 配置声明 `status.yourtj.de` Custom Domain；首次部署前必须完成切换准备。
同名 CNAME 不能并存：保存旧目标、TTL、代理状态后替换 status 记录；不修改其他子域或 NS。
原始历史从上游重新读取，R2 只保存快照，不需要迁移业务数据库。

首次上线保持两个仓库变量关闭，从已审核 main 构建，并用临时 Wrangler 配置省略生产
`routes`，部署到 `yourtj-status.walker-killer.workers.dev` 先验收生产 bucket。快照及临时域名
通过后替换旧 DNS 记录，用正式配置部署绑定 Custom Domain；核对域名与指纹，再开启
`STATUS_DEPLOYMENT_ENABLED`、`STATUS_COLLECTION_ENABLED`。临时配置不提交，也不承载秘密。

验收后保持 Netlify 自动构建关闭，并停止旧站点运行，确认定时采集不会在下个账期恢复。保留项目用于
恢复，不删除历史部署。额度耗尽时回指 Netlify 不保证可用，应优先回滚 Cloudflare 已验收版本。

## 5. 故障与成本

- Worker 没有采集路由，只读取快照；未知／重复查询参数返回 400，存储异常返回不可缓存
  的 503。36 种合法范围组合通过来源作用域指纹选择数据；动态响应 `no-store`，不写入
  Cache API，避免边缘缓存复制的 CPU 峰值。读取不会推进采样时间。
- 公开采集另写十二种范围组合的只读视图；每次 API 请求最多读取一个公开视图和
  一个设备对象。视图保留各来源的时间、失败标记和作用域指纹，不会延长数据有效期。
- R2 和 S3 适配器都使用 ETag 条件写，慢任务不能覆盖新快照。单源失败保留原始成功时间并
  标记过期；采集任务报告失败，其他来源已成功写入的结果仍可用。请求拒绝重定向和过大响应。
- GitHub 备用 schedule 可能排队、延迟或丢弃，不能保证页面数据持续可用。Cloudflare Cron
  避开此定时事件链路，但 GitHub runner 排队和上游故障仍会影响采集。状态站不是告警
  系统；监控由独立 Uptime Kuma 提供。当前／历史／流量二十分钟后过期，一小时后隐藏；
  设备七十分钟后过期，三小时后隐藏。页面仍每六十秒读取，并独立检查时间。
- 此公开仓库的标准 GitHub runner 免分钟费；不使用收费 larger runner，不累积快照 artifacts。
  若仓库转私有，必须重新评估 Actions 额度。公开采集每天 96 次、设备 24 次，正常共约
  2,088 次 R2 写入/日、62,640 次/30日，远低于 Standard 的每月 100 万次 Class A 免费额度。
- Workers Free 每日 10 万请求、每次 10ms CPU 与其他应用共享；静态资源优先返回，不经过
  Worker。采集不占 Worker CPU，读取器使用原生 Web Crypto，不启用 Node 兼容层。
  API 和冷启动仍须实测；每次 API 读取都计 Worker 请求。
  R2 每月含 10GB-month、1,000 万次 Class B，超量另计。按 Worker 每日 10 万请求上限和
  每次两次读取估算，31 天最多约 620 万次公开 API 对象读取，另加采集读取；这些免费额度
  与账户其他应用共享，零月费不是无限用量承诺。

## 官方参考

- [Static Assets](https://developers.cloudflare.com/workers/static-assets/)
- [R2 条件写](https://developers.cloudflare.com/r2/api/s3/api/)
- [Workers 限制](https://developers.cloudflare.com/workers/platform/limits/)
- [R2 定价](https://developers.cloudflare.com/r2/pricing/)
- [GitHub Actions 免费用量](https://docs.github.com/en/billing/concepts/product-billing/github-actions)
- [GitHub schedule 限制](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule)
- [Cloudflare Cron](https://developers.cloudflare.com/workers/configuration/cron-triggers/)
- [GitHub dispatch 权限](https://docs.github.com/en/rest/actions/workflows#create-a-workflow-dispatch-event)
- [GitHub token 触发 workflow_dispatch](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow)
