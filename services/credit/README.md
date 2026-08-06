# credit（linux-do）积分结算 —— 二期
# 参考：https://github.com/linux-do/credit
#
# 接入要点（已源码确认）：
# - credit 是 OAuth2/OIDC 客户端，需 IdP 提供数字 uint64 用户 ID → 用 Casdoor 数字 ID 配置
# - 部署：PostgreSQL 18+ / Redis 6+ / Go 1.26，api+scheduler+worker 三进程 + Next.js 前端
# - 积分跨平台：商户模型（API Key + 签名）分发/订单/转账，论坛作为商户接入
# 本期只预留目录，M6 后规划。
