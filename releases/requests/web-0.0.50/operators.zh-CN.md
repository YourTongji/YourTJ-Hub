数据库迁移（启动自动执行，需保持 db.migration 开启）：AutoMigrate 新增 8 张表——moderation_ai_decisions 与站内信域 7 表（inbox_message、inbox_message_version、inbox_campaign、inbox_campaign_attachment、inbox_campaign_run、inbox_delivery、inbox_claim）。SQLite 与 PostgreSQL 的全新库和存量库升级都会自动补齐；存量通知与私信表的结构和数据不受影响，幂等唯一约束与非空 CHECK 随建表生效。

AI 图文审查配置（默认关闭、默认仅记录）：管理端「设置 → AI 图文审查」配置 Jev Decisions API（OpenRouter 或 TypeSafe 预设）与 OpenAI 兼容视觉模型。两个 API key 经 securestore 加密落库，GET 仅回显是否已配置，可显式清除；保存后 5 秒内热生效。先用仅记录模式积累人工标注，用离线阈值回放校准分数线后再切换检查模式，不得直接套用示例阈值。生产环境请在 OpenRouter 开启零数据保留、关闭日志并限定 provider；视觉模型必须支持图片输入，纯文本模型会令全部带图内容转人工。发布前检查会同步等待模型（上限约 50 秒，仅这些请求放宽 10 秒写超时），反向代理读超时需大于该预算（openresty 默认 60 秒满足）；模型故障、未配置或超出护栏一律转人工，人工审核量会随之上升。观测：结构化日志 ai_moderation_decision（不含正文与密钥），失败另有 ai_moderation_vision_failed、ai_moderation_jev_failed（仅分类与状态码）。

日志脱敏扩展：campus、/api/auth/:provider/callback、内置 OIDC 回调以及登录 redirect 参数不再进入应用访问、panic 与 Gin debug 日志，Referer 一并处理；代理层仍须按 campus 运维要求过滤完整查询串。

deploy.sh 收紧：校验实例名与镜像引用格式；发布部署要求 digest 固定镜像（tag@sha256:...）并设置 EXPECTED_BINARY_SHA256，健康检查通过后再核对运行容器的镜像与二进制摘要，不符即整体回滚；配置锁提前覆盖整个部署事务。
