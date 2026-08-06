# 积分与跨平台结算

> 文档类型：产品规范
>
> 状态：Active（实现 `Planned`，明确二期）
>
> 负责人：Product owner、Platform maintainers
>
> 最近核验：2026-08-06

## 定位

积分（credit，linux-do）是**跨平台结算中心**，不是论坛的附属功能。论坛、Web、移动端以及
未来的 campus 服务（选课、评课等）都是积分系统的**商户/消费者**，共享 Casdoor 身份。

- 账本唯一来源 = credit（PostgreSQL），余额、流水、转账、红包、订单、商户结算都在 credit。
- 论坛只产生积分**事件**（发帖、回帖、互动），通过 credit 的商户分发 API 结算。
- credit 自带商户模型：API Key + 签名（MD5/Ed25519）、`/pay/distribute` 发分、
  `/pay/submit.php` 收款、转账、排行榜、gamification 任务。
- 积分是贡献产生的闭环虚拟权益，**不是可充值、提现或自由转账的货币**。

## 关键约束（已源码确认）

- credit 是 OAuth2/OIDC **客户端**：登录时校验 id_token 的 `sub` 必须是数字 uint64。
  → 数字 ID 约束由此而来（见 identity-and-access.md）。
- credit 部署：PostgreSQL 18+ / Redis 6+ / Go 1.26，`api`+`scheduler`+`worker` 三进程 +
  Next.js 前端；它会在自己库中 upsert 一份用户（按 IdP ID），不是只读代理。
- 用户封禁语义：credit 登录时校验 `active`；GooseForum/自研 server 同步 Casdoor 状态。

## 对接形态（二期设计草案）

```
Casdoor (OIDC, 数字 sub)
   ├── apps/server（论坛）─────┐
   ├── apps/mobile ───────────┼──→ credit（积分账本）
   └── 未来 campus 服务 ───────┘        ↑
                                 商户分发 API（API Key + 签名）
```

- 论坛发帖/回帖奖励 → server 以商户身份调 `POST /pay/distribute` → 用户 `BalanceAdd`。
- 用户间转账 → credit 自带 `POST /api/v1/payment/transfer`。
- 服务内消费（徽章/置顶）→ 商户创建订单 → 用户支付 → 商户收。
- 对账：credit 持久化只读 reconcile + 逐钱包漂移指标（参照 YourTJ 的积分运营经验）。

## 当前边界

- **当前不实现积分**：services/credit 只留部署配置与 README，不部署、不接入。
- 不做充值/提现/法币兑换/自由转账。
- 积分事件在论坛业务稳定后再设计桥接，避免早期耦合。
