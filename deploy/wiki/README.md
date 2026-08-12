# Waline 评论服务（wiki 评论后端）

见 [wiki 部署文档](../../../wiki/docs/deployment/waline.md)。

```bash
cd deploy/wiki/waline
cp .env.example .env
docker compose up -d
```

- 服务端口：`8360`
- 管理后台：`/ui`
- 登录：经 `OAUTH_URL`（OAuth Center）走 YourTJ-Hub OIDC
