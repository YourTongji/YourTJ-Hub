# 1Panel OpenResty 安全镜像

`Partial`：此目录维护 amd64 代理候选的构建与配置缓解，生产切换仍须完成
[代理运维验收](../../docs/operations/proxy-security.md)。不替换面板的配置、WAF 和证书管理方式。

## 构建与边界

```bash
docker build -f services/proxy/Dockerfile -t yourtj-openresty-security:1.31.1.1-openssl3.5.9 services/proxy
```

基础镜像固定 1Panel 官方 amd64 digest，保留 OpenResty 1.31.1.1 与 WAF 布局。
WAF 代码使用另一个固定官方镜像中的 v1 bundle，保持现有 1Panel v1 配置兼容；
不复制生产配置或密钥进镜像，也不通过禁用 WAF 避开兼容问题。
这不是 WAF 版本升级，不能声称修复其潜在漏洞；面板迁移到 v2 时须重新选择对应 bundle。
只替换 OpenResty 私有路径下的 OpenSSL 为 3.5.9，保留官方异步 session callback 补丁。
源码与补丁固定版本及 SHA-256；builder 新增工具不复制到运行镜像，运行阶段安装系统更新。
APT 包随上游安全源更新，产物不保证逐字节重建相同，部署必须记录本次镜像内容 digest。
构建检查 OpenSSL CLI 与 NGINX 实际加载的库，不能仅查看 NGINX 编译时的旧版本字符串。
该镜像不宣称 FIPS 认证，也不新增旧 SSL 协议或弱密码套件。

[OpenSSL 3.5.9 安全公告](https://www.openssl-library.org/news/openssl-3.5-notes/)
包含晚于基础镜像 3.5.7 的修复。此构建没有升级 NGINX 核心，不能声称所有 NGINX CVE 已修复。
部署前须根据[当前 NGINX 公告](https://nginx.org/en/security_advisories.html)核查完整配置，
只允许受影响功能未启用、或已有有效缓解的候选进入生产。

## 配置约束

`security.conf` 放入代理 `http` 上下文，显式关闭 TLS session tickets、HTTP/3、AIO 和 SSI。
须检查所有 server/location 没有重新启用它们，也没有 QUIC listener。
同时禁止在此候选启用 regex map、alias 与 rewrite 的组合、mp4/DAV 处理及未核实的受影响模块；
缺少配置检查即不满足切换条件。不能通过只检查此片段来推断完整配置安全。

使用面板现有应用定义及经过备份的完整配置，在隔离候选中验证 WAF 初始化、模块、权限、TLS/SNI、
缓存、SSE 与真实上游，之后再切换。保存旧镜像及应用定义；回滚仍有旧漏洞，须限时处理。
记录新镜像内容 digest、源码提交、实际加载库版本和配置缓解结果，禁止只记录可变标签。
后续面板升级不得覆盖这些补丁和缓解；必须先核对官方新镜像已覆盖对应安全修复，再退役本候选。
