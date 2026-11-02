package xpservice

// Level is derived from an account's current XP and participation eligibility.
// It is not persisted as authoritative account state.
type Level uint8

const (
	LevelSeedling  Level = iota // Lv0 茶苗：当前不具备参与资格
	LevelGreenTea               // Lv1 绿茶：0 XP
	LevelWhiteTea               // Lv2 白茶：250 XP
	LevelYellowTea              // Lv3 黄茶：1,750 XP
	LevelBlueTea                // Lv4 青茶：6,250 XP
	LevelRedTea                 // Lv5 红茶：15,000 XP
	LevelBlackTea               // Lv6 黑茶：30,000 XP
)

// LevelForXP computes the current level without consulting account age or
// storing a level snapshot. Ineligible accounts are always Lv0; eligible
// accounts at or below zero XP are Lv1.
func LevelForXP(currentXP int64, eligible bool) Level {
	if !eligible {
		return LevelSeedling
	}
	switch {
	case currentXP >= 30_000:
		return LevelBlackTea
	case currentXP >= 15_000:
		return LevelRedTea
	case currentXP >= 6_250:
		return LevelBlueTea
	case currentXP >= 1_750:
		return LevelYellowTea
	case currentXP >= 250:
		return LevelWhiteTea
	default:
		return LevelGreenTea
	}
}
