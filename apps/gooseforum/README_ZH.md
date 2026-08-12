<div align="center">
  <img src="resource/static/pic/icon_300.webp" width="140"/>
  <h1>GooseForum</h1>
  <p>🚀 现代化的 Go + Vue 3 论坛系统</p>

  <p>
    <a href="https://github.com/YourTongji/YourTJ-Hub/releases"><img src="https://img.shields.io/github/release/YourTongji/YourTJ-Hub.svg" alt="GitHub release"></a>
    <a href="https://github.com/avelino/awesome-go"><img src="https://awesome.re/mentioned-badge-flat.svg" alt="Mentioned in Awesome Go"></a>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26+-blue.svg" alt="Go version"></a>
    <a href="https://tailwindcss.com"><img src="https://img.shields.io/badge/TailwindCSS-4-blue.svg" alt="TailwindCSS"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/YourTongji/YourTJ-Hub.svg" alt="License"></a>
    <a href="https://github.com/YourTongji/YourTJ-Hub/stargazers"><img src="https://img.shields.io/github/stars/YourTongji/YourTJ-Hub.svg?style=social" alt="GitHub stars"></a>
  </p>

  <p><a href="README_ZH.md">中文</a> | <a href="README.md">English</a></p>
</div>

> **说明**：本目录是 [YourTJ Hub](https://github.com/YourTongji/YourTJ-Hub) monorepo 内的 GooseForum fork。它保留了“Go + Vue 单二进制”的部署形态，同时产品、认证、搜索、数据、移动端与运维能力都在本仓库中持续演进，超越上游。

![GooseForum 界面预览](../../screenshots/web/home.png)

## 快速开始

### 下载并启动

从 [GitHub Releases](https://github.com/YourTongji/YourTJ-Hub/releases) 下载最新预编译版本，然后启动：

```bash
tar -zxvf yourtj-hub_*.tar.gz
chmod +x ./yourtj-hub
./yourtj-hub serve
```

访问 `http://localhost:5234`。第一个注册用户会自动成为管理员。

### 从源码构建

环境要求：

- Go 1.26+
- Node.js 18+
- pnpm

```bash
git clone https://github.com/YourTongji/YourTJ-Hub.git
cd YourTJ-Hub/apps/gooseforum

cd resource && pnpm install && pnpm build && cd ..
go mod tidy
go build -o yourtj-hub -ldflags="-w -s" .

./yourtj-hub serve
```

或在 monorepo 根目录执行 `make build`，生成单二进制 `bin/yourtj-hub`。

### 配置

GooseForum 首次启动会自动创建 `config.toml`，默认使用 SQLite。

```toml
[app]
env = "production"

[server]
port = 5234
url = "http://localhost"

[db.default]
connection = "sqlite"
path = "./storage/database/sqlite.db"
```

MySQL、PostgreSQL、邮件、备份、安全和站点配置见 [配置文档](docs/user/configuration.md)。

### 管理命令

```bash
./yourtj-hub set-user-admin <用户ID>
./yourtj-hub set-user-email <用户ID> <邮箱>
./yourtj-hub set-user-password <用户ID> <密码>
```

## GooseForum 是什么？

GooseForum 是一个技术社区平台，使用 Go、Gin、GORM、Vue 3、TypeScript、Vite 和 TailwindCSS 构建。它以单个可执行文件发布，支持 SQLite/MySQL/PostgreSQL，并通过服务端 payload 驱动 SPA 体验，同时保留 no-js/SEO 友好的 GoHTML 页面。

## 核心特性

- Markdown 主题、回复、分类、通知、私信、草稿和用户主页。
- 角色和权限管理，内置完整管理后台。
- 桌面端与移动端响应式主站界面。
- 主题工作台，支持浅色/深色主题预览和发布。
- 默认 SQLite，可选 MySQL/PostgreSQL，支持定时备份。
- Payload 驱动站内导航，并保留 no-js GoHTML 降级模板。
- 支持站点 Logo、品牌文案、Footer 和资源自定义。

## 开发

```bash
# 后端热重载
air

# 主站和管理后台前端
cd resource && pnpm dev
```

管理后台由同一个 Vue 应用提供，访问路径为 `/admin`，不需要单独启动管理端前端服务。

## 项目结构

```text
GooseForum/
├── app/                    # 后端代码
│   ├── console/            # CLI 命令
│   ├── http/               # 控制器、中间件、路由
│   ├── models/             # GORM 模型
│   └── service/            # 业务服务
├── resource/               # Vue 3 前端、模板和静态资源
│   ├── src/site/           # 主站
│   ├── src/admin/          # 管理后台
│   ├── src/runtime/        # Payload 运行时和共享浏览器工具
│   └── templates/          # GoHTML 降级模板
├── docs/                   # 文档
├── main.go
└── config.toml
```

## 部署建议

生产环境建议放在 Nginx 或 Caddy 等反向代理后，开启 HTTPS，并配置数据库备份。

最小容器镜像示例：

```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY yourtj-hub .
CMD ["./yourtj-hub", "serve"]
```

## 文档

- [Monorepo 文档中心](../../docs/README.md)
- [配置文档](docs/user/configuration.md)
- [前端架构](docs/architecture/resource-frontend.md)
- [UI 规范](docs/frontend/ui-spec.md)
- [English README](README.md)

## 许可证

MIT License，见 [LICENSE](LICENSE)。
