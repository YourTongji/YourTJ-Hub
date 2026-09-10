# 图片流量成本模型：服务端内网代理读，拒绝浏览器直读 OSS

## Status
Accepted
Class: architecture

## Context and Problem Statement

S3/OSS 部署形态下图片读取有两条路径：① 服务端经 `internalEndpoint` 内网代理读取并回传浏览器（当前形态）；② `PublicUrlPrefix`/presigned GET 让浏览器直读 OSS 公网 endpoint。

OSS 公网下行流量是主要计费项，内网流量免费或极低价。校园论坛读远多于写，直读方案会把流量成本按浏览量持续放大；服务端内网拉取 + 应用回传则把 OSS 侧成本压到接近零。

现状已按此模型落地：`S3Provider` 双 client（数据面走 internal、presign 恒走公网直传），`PublicUrlPrefix` 留空，公开读取恒走 `/file/img/*` 代理路由。

## Decision Drivers

- 读流量成本必须不随浏览量线性放大。
- 写路径（直传 presigned PUT）走公网是可接受的（写入频次低）。
- 速率优化不得以转嫁流量成本为代价。

## Considered Options

- 服务端内网代理读为长期形态，速率优化限定在应用服务器内部（选定）。
- 浏览器直读 OSS（PublicUrlPrefix 公网 / 302 到 presigned GET）。
- CDN 回源（回源走内网）。

## Decision Outcome

1. 浏览器直读 OSS 类方案（PublicUrlPrefix 指向公网、302 到 presigned GET 等）从速率优化候选中整体排除——它们省应用服务器带宽，但把成本转嫁为 OSS 公网下行流量费，与本成本模型冲突。
2. 服务端内网代理读是长期形态；图片速率优化限定在应用服务器内部：引用检查查询成本（file_usage 索引）、缓存协商头（ETag/304、assets immutable）、缩略图管线、服务端字节流效率。
3. `PublicUrlPrefix` 保留为能力项（如未来接入免费/已购流量的 CDN），但当前部署恒留空。

### Consequences

- Good: OSS 侧读成本接近零；成本模型不随浏览量放大。
- Trade-off: 应用服务器承担全部图片回传带宽（校园规模可接受）。
- 优化杠杆排序（2026-09-05 分析结论）：file_usage `file_name` 索引（P0，`HasAnyReferences` 当前全表扫）→ `/assets/*` immutable 缓存头（P1）→ `/file/img` ETag/304（P1~P2）→ 缩略图管线（P2，产品级决策）→ S3 流式回传（内网近免费，次要项）。
- 缩略图管线若落地：服务端内网拉原图生成、产物写回 OSS（写内网，成本可控），读取仍回 `/file/img` 代理路由；`Provider.Get` 全量内存缓冲仍是合法优化点（`io.Copy` 流式回传）。

## Pros and Cons of the Options

### 内网代理读（选定）
- Good: OSS 成本压到近零；代理路由统一做权限/审计/缓存协商。
- Bad: 应用服务器带宽与内存占用；代理路径长于直读。

### 浏览器直读
- Good: 应用服务器卸载全部图片带宽。
- Bad: 成本按浏览量持续放大（本决策的核心否决理由）；权限旁路风险。

### CDN 回源内网
- Good: 兼顾成本与边缘加速。
- Bad: 引入 CDN 依赖与计费项；当前流量规模不足以摊薄。

## Links

- 对象存储运维：[object-storage](../operations/object-storage.md)
- 速率优化分析（2026-09-05 会话，宿主 Agent Note）
