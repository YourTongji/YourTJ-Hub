# 宿主机安全补丁维护

> Doc type: reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-22

本页维护 Debian 系统包与 Docker 的变更边界。[部署文档](deployment.md)负责实例、数据库恢复和
业务部署；论坛镜像更新不会替换宿主机 Docker、SSH 或面板代理。

## 核查与停止条件

维护者先在 Issue 记录公告来源、目标包版本、适用条件、业务影响、恢复方式和可观察验收结果，
以 PR 评审运维规则。原始日志、配置、主机标识及密钥只留在主机受限目录，公开记录仅保留脱敏结论。
发行版可能回补补丁，须用完整包版本对照 [Debian 安全跟踪](https://security-tracker.debian.org/tracker/)，
不能仅比较上游版本。另行核查容器内的库；升级宿主机 OpenSSL 不会替换代理镜像内的 OpenSSL。

索引刷新和模拟升级可在不中断服务的准备阶段执行：

```bash
apt-get update
apt-cache policy openssl libssl3 openssh-server docker-ce docker-ce-cli containerd.io
apt-get --simulate install openssl libssl3 openssh-server docker-ce docker-ce-cli containerd.io
```

检查所有源是否刷新成功、签名是否有效、包的来源、精确候选版本、依赖变化及磁盘空间。
出现源错误、非预期来源、删除包、保留关键包或异常依赖时停止。模拟结果不是安装脚本演练：
实际安装可能通过 maintainer scripts 或 needrestart 重启服务，即使没有手动执行 restart。
实际执行命令须在私有维护记录中固定已评审版本；不能把持续变化的 candidate 当作批准对象。

## 执行前提

- 选择维护窗口，暂停自动部署及其他运维变更，指定执行人与回滚负责人。用户要求完全不中断时，
  必须先有经过验证的冗余方案；单机升级不能承诺零中断。
- 验证带外控制台可用；保留当前 SSH，再建立第二个密钥认证连接。SSH 更新前检查 `sshd -t`，
  检查实际连接对应的 Match 条件，避免只看全局默认值。
- 修改配置前保存带时间戳的备份，保留原权限/所有者；新增备份使用 root 专属目录和 `umask 077`。
  备份包版本、软件源、systemd overrides、Docker daemon 配置和面板管理的服务定义。
- 复用[部署备份与恢复流程](deployment.md)，分别确认 PostgreSQL 与文件数据库的可恢复备份，
  在隔离环境验证恢复；不要用运行中数据库目录的普通复制代替一致性备份。
  同时保留持久卷、旧镜像和可获得的旧包；任何清理均等恢复窗口结束。

## Docker 与系统库

根据 [Docker 安全发布说明](https://docs.docker.com/engine/release-notes/29/)选择目标，
同时核查 containerd、runc 和构建工具；仅更新 CLI 不会修复 daemon。

`live-restore` 可以降低 daemon 停止的影响，但[官方只保证支持的补丁升级条件](https://docs.docker.com/engine/daemon/live-restore/)。
跨 minor、跳版本、containerd 更新及 daemon 网络/存储参数变化不能据此承诺容器不中断。
若单独启用它，先备份并合并现有 JSON，不覆盖其他配置；检查启动参数冲突，使用
`dockerd --validate --config-file=/etc/docker/daemon.json` 验证，再通过 `systemctl reload docker`
生效并核对 `docker info` 和容器启动时间。该准备动作不等于授权执行包升级。

安装分批进行，每批先检查包脚本及重启影响，并预设停止条件。不要通过禁止所有服务重启后
直接宣布修复：运行中的进程可能仍映射旧库。系统库安装后确认受影响进程已在批准窗口内重启；
内核修复需要验证实际运行版本，不能只检查磁盘上的新包。

## 验收与恢复

每批后核对包与实际 daemon/进程版本、容器健康及启动时间、主站与 dev `/health`、外部 HTTPS、
登录、附件、数据库连接及搜索。SSH 更新后从新连接验证登录，成功前保留旧连接。
观察错误率、延迟和后台任务；缺少业务或外部 TLS 验证时标为未验收。

失败时停止后续升级，通过带外控制台恢复配置和经验证兼容的包/镜像，重新检查服务。
包降级不能保证兼容新的运行时状态或数据格式；有状态服务按其专属恢复流程处理，
不能删除卷或把旧镜像直接接到已升级数据上。仍有新业务写入时，不得以恢复旧数据库掩盖故障。
健康恢复后保留故障证据并评审下一步；只有实际生产升级与验收均完成，才关闭安全 Issue。
