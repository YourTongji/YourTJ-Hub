# Object storage

> Doc type: operations
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-02

## 概述

论坛文件默认存于 SQLite BLOB（`[db.file]`，无外部依赖）。在管理端（设置 → 存储设置）配置
S3 兼容对象存储（MinIO / Tencent COS / Alibaba OSS / Cloudflare R2）后，**帖子图片**改为
浏览器直传对象存储：浏览器向服务端请求一份短时效的 POST 策略，直接把文件上传到 bucket，
再由服务端校验并发布。本地存储提供方（`provider = "local"`）保持服务端代理的 multipart
上传（`POST /file/img-upload`），行为不变。

配置形状（`pageConfig.StorageSettings`，管理端存储设置页维护，securestore 加密落库）：

| 字段 | 说明 |
|---|---|
| `provider` | `local` / `s3` |
| `endpoint` | 对象存储端点，可带 `https://`；服务端剥离 scheme 并据此推导 TLS（或由 secure 开关决定） |
| `bucket` | bucket 名称 |
| `region` | 区域（Alibaba OSS / Tencent COS 必须显式设置） |
| `bucketLookup` | `auto` / `dns` / `path`；OSS/COS（2024-01-01 后创建的 bucket）仅支持 `dns`（virtual-hosted），MinIO/R2 可用 `auto`/`path` |
| `secure` | 是否使用 TLS |
| `publicUrlPrefix` | 可选公开 URL 前缀（CDN 直读）；未配置时论坛通过 `/file/img/*` 代理读取 |

## 帖子图片直传流程

1. 浏览器 `POST /file/img-upload/init`（JWT + 可写账号校验 + 上传限流），携带
   `{ filename, contentType, size }`；服务端先按发布设置做扩展名白名单与大小上限校验。
2. 响应 `mode`：
   - `proxy`（本地提供方）——浏览器继续走既有 multipart `POST /file/img-upload`；
   - `direct`（S3 提供方）——响应携带短时效的预签名 POST 策略（`upload.url` +
     `upload.fields`）与待发布对象名 `name`。
3. `direct` 模式下浏览器把 `upload.fields` 与文件组装成 FormData，`POST` 到 `upload.url`
   （bucket 端点），随后 `POST /file/img-upload/complete`（`{ name }`）请求发布。
4. 服务端在发布前重新校验：对象归属（必须属于当前调用者）、对象大小、MIME 类型与
   **解码后的图片头**；伪造或无效对象以 `upload.invalidImage` 拒绝，绝不直接信任浏览器声明。
5. 发布成功后返回最终公开 `url`、原始 `filename` 与字节 `size`；失败或取消时浏览器
   `POST /file/img-upload/abort`（`{ name }`）删除待发布对象。

**未完成上传的清理**：超过 2 小时仍未 complete 的待发布对象由服务端清理任务移除
（abort 失败可忽略，清理任务兜底）。

## Bucket CORS

浏览器直传要求 bucket 允许来自论坛源站的 `POST` 请求。`AllowedOrigins` 必须填写**论坛的
精确公开源**（含 scheme 与端口），不要使用通配符：

```json
[
  {
    "AllowedOrigins": ["https://forum.example.com"],
    "AllowedMethods": ["POST"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```

## 最小权限

- bucket 保持**私有**；未配置 `publicUrlPrefix` 时读取也走论坛代理（`/file/img/*`）。
- 应用凭据（accessKey/secretKey）只授予配置 bucket 所需的对象操作（如 `s3:PutObject`、
  `s3:GetObject`、`s3:DeleteObject`、`s3:ListBucket`），不授予跨 bucket 或管理权限。
- **任何密钥都不会下发到浏览器**：浏览器只拿到短时效、限定对象与大小的预签名 POST 策略，
  服务端凭据仅存于服务端（securestore 加密落库，管理端不回显）。