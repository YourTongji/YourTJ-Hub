# Meilisearch search service (to be wired)
# dev is already in the root docker-compose.yml (:7700, master key yourtj-dev-master-key)
#
# Plan:
# - server syncs indexes asynchronously on topic/reply writes (topic/posts indexes)
# - search API goes through the server proxy (unified auth), shared by web/mobile
# - Chinese tokenization: built into Meilisearch, no extra config
