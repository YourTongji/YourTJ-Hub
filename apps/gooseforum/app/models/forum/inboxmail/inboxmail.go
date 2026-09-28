// Package inboxmail owns the official 站内信 (inbox mail) domain: an independent
// data model for long-lived, rule-triggered, rich-text official letters with
// attachments and reward claims. It deliberately does not extend
// event_notification (lightweight interaction events) or the private chat
// tables.
//
// The fixed main model is
//
//	Message → Message Version → Campaign → Campaign Run → Delivery → Claim
//
// with these invariants (epic #769):
//
//   - A published message version is immutable. Content, block JSON, content
//     hash, schema version and version number can never be updated in place;
//     a revision creates a new version row. Draft rows are editable through
//     UpdateDraftContentTx.
//   - Every materialized per-user delivery has a NOT NULL, globally unique
//     dedupe_key; worker retries and event replays therefore cannot duplicate
//     a letter. Build keys with CampaignDedupeKey / TriggerDedupeKey.
//   - Every claim has a NOT NULL, globally unique source_key (namespaced by
//     reward handler) and is additionally protected by a unique
//     (delivery_id, attachment_id) constraint. Build keys with ClaimSourceKey.
//   - Read, popup and claim state are separate columns on the delivery, so
//     mailbox unread, popup annoyance and claim conversion each have their own
//     accurate denominator.
//
// User lifecycle: DeleteUserDataTx / AnonymizeUserDataTx are the reserved
// boundary for account close and retention cleanup; wiring them into
// userservice.CloseAccount and the bounded-retention policy belongs to #787.
//
// Cross-domain note: this package does no SQL against other domains' tables.
// Campaign audience compilation (#772), campaign lifecycle (#773), delivery
// materialization (#774), trigger registry (#775), mailbox API (#776) and
// reward handlers (#777/#778) build on these rows.
package inboxmail

const (
	messageTableName            = "inbox_message"
	messageVersionTableName     = "inbox_message_version"
	campaignTableName           = "inbox_campaign"
	campaignAttachmentTableName = "inbox_campaign_attachment"
	campaignRunTableName        = "inbox_campaign_run"
	deliveryTableName           = "inbox_delivery"
	claimTableName              = "inbox_claim"
)

// AnonymizedUserID is the tombstone user id written by AnonymizeUserDataTx.
// Real user ids start at 1, so 0 never matches a live account.
const AnonymizedUserID uint64 = 0

// AllModels returns every inboxmail model for AutoMigrate in tests and future
// tooling. The production entry point is migration.SchemaModels(), which lists
// the same seven models explicitly (a slice literal cannot spread a call);
// TestSchemaModelsRegistersInboxModels guards the two lists against drift.
func AllModels() []any {
	return []any{
		&MessageEntity{},
		&MessageVersionEntity{},
		&CampaignEntity{},
		&CampaignAttachmentEntity{},
		&CampaignRunEntity{},
		&DeliveryEntity{},
		&ClaimEntity{},
	}
}
