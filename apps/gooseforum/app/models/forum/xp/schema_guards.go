package xp

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsureAppendOnlyGuards installs database-level protections for the ledger
// and published rule definitions. ORM hooks provide earlier errors; triggers
// also protect direct SQL maintenance paths from rewriting audit history.
func EnsureAppendOnlyGuards(db *gorm.DB) error {
	for _, model := range []any{&Entry{}, &RuleSet{}, &RewardRule{}} {
		if !db.Migrator().HasTable(model) {
			return fmt.Errorf("XP append-only table missing: %T", model)
		}
	}
	switch db.Name() {
	case "sqlite":
		for _, table := range []string{"xp_entries", "xp_rule_sets", "xp_reward_rules"} {
			for _, operation := range []string{"UPDATE", "DELETE"} {
				name := table + "_no_" + operation
				query := fmt.Sprintf(
					"CREATE TRIGGER IF NOT EXISTS %s BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, 'xp record is append-only'); END",
					name, operation, table,
				)
				if err := db.Exec(query).Error; err != nil {
					return fmt.Errorf("install SQLite XP %s guard: %w", name, err)
				}
			}
		}
	case "postgres":
		if err := db.Exec(`CREATE OR REPLACE FUNCTION xp_reject_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'xp record is append-only' USING ERRCODE = '55000'; END; $$`).Error; err != nil {
			return fmt.Errorf("install PostgreSQL XP append-only function: %w", err)
		}
		for _, table := range []string{"xp_entries", "xp_rule_sets", "xp_reward_rules"} {
			name := table + "_no_mutation"
			if err := db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", name, table)).Error; err != nil {
				return fmt.Errorf("replace PostgreSQL XP trigger %s: %w", name, err)
			}
			if err := db.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION xp_reject_mutation()", name, table)).Error; err != nil {
				return fmt.Errorf("install PostgreSQL XP trigger %s: %w", name, err)
			}
		}
	default:
		return fmt.Errorf("unsupported database dialect for XP append-only guards: %s", db.Name())
	}
	return nil
}
