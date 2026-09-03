# Go Dependency Vulnerability Scanning

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-03

后端 Go 模块（`apps/gooseforum`）的依赖漏洞扫描基线（issue #410）。本文回答三个问题：怎么扫、扫描失败怎么诊断、`x/crypto` 间接依赖的结论是什么。

## 扫描方式

| 场景 | 命令 / 位置 |
|---|---|
| CI（每次 push/PR） | `.github/workflows/ci-govulncheck.yml` 的 `ci-backend-govulncheck` job（独立 workflow，push 无路径过滤），用官方 `golang/govulncheck-action@v1`（`work-dir: apps/gooseforum`） |
| 本地 | `cd apps/gooseforum && go run golang.org/x/vuln/cmd/govulncheck@latest ./...`，或仓库根 `make govulncheck` |

行为要点：

- govulncheck 只报告**可达**漏洞：标准库 + 实际调用链能到达的依赖漏洞；`ssh`/`openpgp` 之类只存在于依赖树、代码未调用（甚至未 import）的漏洞会归入 Module/Package 桶，不影响退出码。无业务代码、无 `go.mod`/`go.sum` 版本变更也能干净复跑本扫描。
- CI 用默认 `output-format=text`，job 直接透传 govulncheck 退出码：发现可达漏洞 exit 3（job 红，日志含漏洞 ID 与修复版本），无漏洞 exit 0。
- vuln DB 每次运行都从 `vuln.go.dev` 拉最新；不做本地缓存快照，保证新披露漏洞在下次 CI 即被发现。
- job 位于**独立 workflow**（无路径门）：push 到 dev/main 不带 `paths` 过滤，每次 push 全量运行；除了 `go.mod`/`go.sum`，Go patch 版本与 vuln DB 更新同样会让结论过期，按 backend 路径触发会漏扫。

## vuln DB 更新策略

- DB 上游：Go 官方 `vuln.go.dev`（`GOVULNDB` 环境变量可覆盖，见下）。
- govulncheck 二进制本身用 `go install ...@latest`（CI action 内部）或 `go run ...@latest`（本地），每次取最新版。
- Go 工具链：CI 用 `go-version-input: "1.26"`，action 内部 `check-latest: true` 会取最新 1.26.x patch。标准库漏洞随工具链升级消解：例如 go1.26.5 报告的 7 个 stdlib CVE（`net/url`、`html/template`、`crypto/tls`、`net/http`、`encoding/xml`、`encoding/asn1` 等）全部修复于 go1.26.6，升级 Go patch 即可清零，无需动任何依赖。
- 依赖版本升级由 Dependabot（`.github/dependabot.yml`，`go_modules` ecosystem，每周）提出 PR；升级 PR 会触发本扫描与 `ci-backend` 全量测试。

## 代理失败诊断

govulncheck 需要网络拉取两样东西：govulncheck 工具本身（走 `GOPROXY`）与漏洞 DB（走 `GOVULNDB`）。任何一步失败都会**非零退出，CI job 变红**——不允许"网络错误静默变绿"。

本地失败排查顺序：

1. 确认报错是网络层还是漏洞层：网络失败是 `go: ... Get "...": dial tcp ...: i/o timeout` / `proxyconnect` 类错误；漏洞层是 `Vulnerability #N` + `exit status 3`。前者是环境问题，后者是真发现。
2. 国内网络直连 `proxy.golang.org` / `vuln.go.dev` 常超时，改用镜像：

   ```bash
   cd apps/gooseforum
   GOPROXY=https://goproxy.cn,direct GOVULNDB=https://goproxy.cn \
     go run golang.org/x/vuln/cmd/govulncheck@latest ./...
   ```

   > 注意：goproxy.cn 镜像的 vuln DB 有滞后，结论以官方 `vuln.go.dev` 为准；本地复核建议直接 `GOVULNDB=https://vuln.go.dev`。

3. 若镜像也失败，检查代理环境变量（`HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`）与 DNS；CI 运行在 GitHub 官方网络，不经过本地代理。
4. 复现/记录问题时贴完整原始输出（含退出码），并区分三类结论：应用可达漏洞、依赖存在但不可达提示、扫描基础设施失败。

## x/crypto 传递依赖链结论（2026-09-03 复核）

现状：`apps/gooseforum/go.mod:137` 声明 `golang.org/x/crypto v0.54.0 // indirect`，全模块（含 `main.go` 与测试）**没有任何第一方文件直接 import 该模块**（`grep -rn 'golang.org/x/crypto' --include='*.go'` 为空）。

引入链（`go mod why golang.org/x/crypto/<leaf>` 实测；首行为直接依赖方）：

| x/crypto 叶子 | 引入者 | 用途 |
|---|---|---|
| `bcrypt` / `blowfish` | `Masterminds/sprig/v3`（controller 层模板渲染） | 口令哈希 |
| `scrypt` / `pbkdf2` | `Masterminds/sprig/v3` | 模板密码学函数 |
| `chacha20poly1305` / `chacha20` | `quic-go/quic-go`（gin v1.12 自带 HTTP/3 支持的传递依赖，经 gin-contrib/gzip 进入构建图；仓库自身无 quic/HTTP3 使用） | QUIC 传输加密 |
| `argon2` / `blake2b` | `minio-go/v7`（S3 存储服务） | 对象加密 |
| `sha3` | `go-playground/validator/v10` | 校验散列 |
| `md4` | `wneessen/go-mail`（SMTP NTLM） | 邮件认证 |

`go list -deps` 全图内的 x/crypto 包仅上述密码学原语 + 内部辅助包（`internal/alias`、`internal/poly1305`）。**没有任何 `ssh`/`openpgp`/`cryptobyte` 叶子**——go.sum 里这些有漏洞记录的包只存在于 module graph，不在构建图里。

govulncheck 实测（2026-09-03，v0.54.0，text 格式）将 4 条 x/crypto 条目全部归入 Module Results（不可达）：`GO-2026-6355` / `GO-2026-6354` / `GO-2026-6303`（`ssh` 通道死锁/认证回调）+ `GO-2026-5932`（`openpgp` 不再维护），退出码不受影响。

**结论：x/crypto 为间接依赖、不可达（无 ssh/openpgp 叶子）、接受风险**，无需为扫描条目升级或引入任何代码；x/crypto 版本跟随引入方（sprig/quic-go/minio-go/go-mail）的升级而升级，Dependabot 每周自动提出。升级 x/crypto 的修复版本不改变本结论时，不需要额外动作。

> 守则：**新增对 `golang.org/x/crypto/ssh` 或 `golang.org/x/crypto/openpgp` 的直接依赖前，必须重新评估**——ssh/openpgp 是 govulncheck 报告的高危面（且 openpgp 官方已弃用），任何引入都要在本文更新可达性结论、版本约束与复核日期，并确保 CI 扫描通过。

## 扫描结论口径

报告与 issue 中区分三类，不要混为一谈：

1. **应用可达漏洞**（Symbol Results）：当前实际基线为 **0 个**——CI run 33781751646（2026-09-03，action `check-latest` 取到 Go 1.26.8）显示 `No vulnerabilities found`。历史注记：go1.26.5 曾报告 7 个标准库漏洞（均修复于 go1.26.6，随工具链升级消解）；`x/image` VP8L `GO-2026-6222`（v0.44.0 → 修复于 v0.45.0）在本分支仅出现在 Package 层（import 但未调用、不可达，不影响退出码），随 issue #405 合入后条目整体消失。
2. **依赖存在但不可达提示**（Package/Module Results）：含本文 x/crypto 4 条。
3. **扫描基础设施失败**：网络/代理错误，必须让 CI 红并按下节诊断。
