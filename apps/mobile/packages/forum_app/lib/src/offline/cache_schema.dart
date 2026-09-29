/// Versioned, disposable schema. Kept independent of the workspace's Freezed
/// build-runner toolchain. Every migration is exercised against a real SQLite DB.
const cacheSchema = [
  'CREATE TABLE IF NOT EXISTS cache_entries ('
      'scope TEXT NOT NULL, domain TEXT NOT NULL, entry_key TEXT NOT NULL, '
      'payload TEXT NOT NULL, saved_at INTEGER NOT NULL, accessed_at INTEGER NOT NULL, '
      'byte_size INTEGER NOT NULL CHECK(byte_size >= 0), document_version INTEGER NOT NULL DEFAULT 1, '
      'PRIMARY KEY (scope, domain, entry_key))',
  'CREATE INDEX IF NOT EXISTS cache_eviction ON cache_entries(domain, accessed_at)',
  'CREATE TABLE IF NOT EXISTS campus_snapshots ('
      'site TEXT NOT NULL, account_id INTEGER NOT NULL, binding_revision TEXT NOT NULL, '
      'schema_version INTEGER NOT NULL, payload TEXT NOT NULL, committed_at TEXT NOT NULL, '
      'PRIMARY KEY (site, account_id))',
  'CREATE TABLE IF NOT EXISTS storage_operations ('
      'operation TEXT PRIMARY KEY, value TEXT NOT NULL)',
];
