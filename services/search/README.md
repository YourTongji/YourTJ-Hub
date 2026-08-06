# Meilisearch 搜索服务（M6 接入）
# dev 已在根 docker-compose.yml（:7700，master key yourtj-dev-master-key）
#
# 计划：
# - server 写帖/回帖时异步同步索引（topic/posts 两个索引）
# - 搜索 API 走 server 代理（统一鉴权），web/mobile 共用
# - 中文分词：Meilisearch 自带，无需额外配置
