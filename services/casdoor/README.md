# Casdoor 统一认证部署配置（本地 dev 已在根 docker-compose.yml 中）
# 初始化清单（沿用已验证配置）：
# 1. 创建组织 forum（owner=admin）
# 2. 创建应用 forum-app：
#    - organization=forum（纯组织名）
#    - signupItems: ID rule=Incremental（用户自助注册得数字 ID）
#    - enablePassword=1, enableSignUp=1
#    - grantTypes: authorization_code, password, refresh_token, token, id_token
# 3. 管理员建号显式传数字 id（API 创建的用户 passwordType=bcrypt 需修正）
# 4. 记录 client_id / client_secret 到 deploy/.env（CASDOOR_CLIENT_ID/SECRET）
#
# 注意：用户 id 默认是 UUID，credit 的 GetID() 需要数字 uint64 ——
#       必须用上述数字 ID 配置，否则所有用户解析为 0 互相覆盖。
