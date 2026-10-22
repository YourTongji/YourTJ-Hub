# Meilisearch search service

论坛通过服务端访问 Meilisearch；浏览器和移动端不直接取得引擎管理凭据。
PostgreSQL 是业务真相源，搜索索引是可重建投影。服务边界和现有重建入口见
[部署文档](../../docs/operations/deployment.md#admin-search-index-maintenance)。

## 版本与安全边界

本地和部署 Compose 使用 Meilisearch `v1.47.1`。该版本包含
[配置写权限下的 SSRF 修复](https://www.meilisearch.com/blog/CVE-update-Jan-2026)，
以及 [CVE-2026-57823 / CVE-2026-57824 修复](https://github.com/meilisearch/meilisearch/releases/tag/v1.47.1)。
新版本默认阻止引擎请求非公网地址；不要为了恢复旧配置而直接放开私网请求。
实际风险仍取决于 API key 权限、tenant token、embedder 和网络配置。
生产引擎只接受受控服务端访问，管理密钥通过现有安全配置注入。

## 升级已有数据目录

更新镜像声明不会迁移已有索引。生产切换需要维护者单独批准，并安排搜索维护窗口。
按[官方升级文档](https://www.meilisearch.com/docs/learn/update_and_migration/updating)执行：

1. 核实当前运行版本、数据目录和调用方，保留旧镜像及完整旧卷备份。
   main 与 dev 共用的引擎和索引必须一起安排，暂停索引写入并记录待补写数据。
2. 使用旧版本引擎调用 `POST /dumps`，等待对应任务成功；安全保存 dump 并检查可读性。
   dump 可能含文档和 API key，按敏感备份管理，不提交 Git，也不贴到日志或 Issue。
3. 在隔离的新空卷启动 `v1.47.1`，通过 `--import-dump` 导入旧 dump。
   `v1.11` 数据目录不支持 `--upgrade-db`；不要让新引擎直接打开旧卷，
   不要用 snapshot 代替跨版本 dump，也不要删除旧卷。
4. 等待导入和重新索引完成，核对文档数、设置、密钥权限、搜索结果和任务状态。
   检查公开性变更与删除后的搜索移除，并确认两个论坛实例均可读取新引擎。
5. 验收后才切换引擎连接并恢复写入。部署 Compose 的逻辑卷仍名为 `meilidata`；
   将服务器 `.env` 的 `MEILI_DATA_VOLUME` 设置为已验证新卷的实际名称，
   防止后续发布覆盖 Compose 时重新连接旧卷。未设置时保留原项目默认卷名。
   用 `docker compose config` 核对解析后的卷名，再只重建搜索服务；禁止执行 `down -v`。
   通过现有索引维护入口补齐停写期间的投影。
   若选择从 PostgreSQL 重建，也必须在新卷完成并核实，禁止先清空共享旧索引。

切换前失败时继续使用旧镜像与旧卷。切换后若需回滚，先暂停写入并核对切换后的
投影变化，再恢复保留的旧镜像和旧卷、补写差异；不要将新数据目录交给旧引擎，
也不要假定新版 dump 可以导回旧版本。业务数据库保持不动。
