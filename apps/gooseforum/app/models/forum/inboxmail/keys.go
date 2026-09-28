package inboxmail

import "fmt"

// 幂等 key 的格式是跨 issue 的稳定契约：#774 投递去重、#775 事件触发、#778 领取
// 都写入这些字符串，格式漂移会让同一事实在数据库唯一约束下被重复插入。
//
// 各段由服务端标识符组成（数字 id、事件名、规则生成的 code），因此分段值不得
// 包含 ':' —— 否则不同分段组合可能拼出同一字符串，唯一约束的语义就不再是
// 「同一事实」。数字 id 天然安全；事件名与 event_id 由 Trigger Registry（#775）
// 在注册/触发时保证不含 ':'。
//
// key 只由服务端标识符组成，不嵌入外部输入（如 badge code）：handler 侧标识变化
// 不会让同一领取事实换一个 key。

// CampaignDedupeKey 返回 Campaign 投递的 dedupe key：
// campaign:<campaign_id>:<version_no>:user:<user_id>。
// 使用 version_no（而非版本行 id）让 key 可读且与「投递固定版本」的语义一致。
func CampaignDedupeKey(campaignID uint64, versionNo int, userID uint64) string {
	return fmt.Sprintf("campaign:%d:%d:user:%d", campaignID, versionNo, userID)
}

// TriggerDedupeKey 返回事件触发投递的 dedupe key：
// trigger:<campaign_id>:<event>:<event_id>:user:<user_id>。
//
// campaign_id 是触发规则所属的 Campaign。事件触发投递也归属某个 Campaign，必须
// 进入 key：dedupe_key 是全局唯一索引，缺少 campaign 段时两个 Campaign 匹配同一
// 事件 occurrence 会得到逐字节相同的 key，第二封信会被当作重放静默丢弃（#775
// 要求不同 occurrence 各自成信，Campaign 维度同理）。campaign 段同时让
// inbox_delivery.campaign_id 的收件人排查对 trigger 投递同样成立。
//
// event_id 是事件源自身的稳定标识（如注册事件 id），事件重放同样得到该 key。
func TriggerDedupeKey(campaignID uint64, event, eventID string, userID uint64) string {
	return fmt.Sprintf("trigger:%d:%s:%s:user:%d", campaignID, event, eventID, userID)
}

// ClaimSourceKey 返回附件领取的幂等键：inbox:<delivery_id>:<attachment_id>。
//
// 领取事实的粒度就是 (delivery, attachment)（#777 统一规则），因此 key 必须含
// attachment_id：附件唯一性只在 (campaign_id, attachment_key) 粒度上成立，同一
// delivery 的两个附件即使解析出同一个 handler key（同 badge code 两行）也必须能
// 各自领取，不能因为 key 更粗而互相顶掉。
func ClaimSourceKey(deliveryID, attachmentID uint64) string {
	return fmt.Sprintf("inbox:%d:%d", deliveryID, attachmentID)
}
