package inboxmail

import "fmt"

// 幂等 key 的格式是跨 issue 的稳定契约：#774 投递去重、#775 事件触发、#778 领取
// 都写入这些字符串，格式漂移会让同一事实在数据库唯一约束下被重复插入。
//
// 各段由服务端标识符组成（数字 id、registry key、事件名、规则生成的 code），
// 因此分段值不得包含 ':' —— 否则不同分段组合可能拼出同一字符串，唯一约束的
// 语义就不再是「同一事实」。外部输入（如 badge code）需在创建附件/触发规则时
// 校验后再传入。

// CampaignDedupeKey 返回 Campaign 投递的 dedupe key：
// campaign:<campaign_id>:<version_no>:user:<user_id>。
// 使用 version_no（而非版本行 id）让 key 可读且与「投递固定版本」的语义一致。
func CampaignDedupeKey(campaignID uint64, versionNo int, userID uint64) string {
	return fmt.Sprintf("campaign:%d:%d:user:%d", campaignID, versionNo, userID)
}

// TriggerDedupeKey 返回事件触发投递的 dedupe key：
// trigger:<event>:<event_id>:user:<user_id>。
// event_id 是事件源自身的稳定标识（如注册事件 id），事件重放同样得到该 key。
func TriggerDedupeKey(event, eventID string, userID uint64) string {
	return fmt.Sprintf("trigger:%s:%s:user:%d", event, eventID, userID)
}

// ClaimSourceKey 返回 Reward Handler 层的领取幂等键：
// inbox_claim:<handler>:<handler_key>:delivery:<delivery_id>。
// handler_key 是 handler 自己的稳定标识（徽章为 badge code）。
func ClaimSourceKey(handler, handlerKey string, deliveryID uint64) string {
	return fmt.Sprintf("inbox_claim:%s:%s:delivery:%d", handler, handlerKey, deliveryID)
}
