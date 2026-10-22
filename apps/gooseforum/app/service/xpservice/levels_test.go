package xpservice

import "testing"

func TestLevelForXPBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		xp       int64
		eligible bool
		want     Level
	}{
		{name: "ineligible despite high XP", xp: 30_000, eligible: false, want: LevelSeedling},
		{name: "negative XP floors at first eligible level", xp: -1, eligible: true, want: LevelGreenTea},
		{name: "zero XP", xp: 0, eligible: true, want: LevelGreenTea},
		{name: "below white tea", xp: 249, eligible: true, want: LevelGreenTea},
		{name: "white tea threshold", xp: 250, eligible: true, want: LevelWhiteTea},
		{name: "below yellow tea", xp: 1_749, eligible: true, want: LevelWhiteTea},
		{name: "yellow tea threshold", xp: 1_750, eligible: true, want: LevelYellowTea},
		{name: "below blue tea", xp: 6_249, eligible: true, want: LevelYellowTea},
		{name: "blue tea threshold", xp: 6_250, eligible: true, want: LevelBlueTea},
		{name: "below red tea", xp: 14_999, eligible: true, want: LevelBlueTea},
		{name: "red tea threshold", xp: 15_000, eligible: true, want: LevelRedTea},
		{name: "below black tea", xp: 29_999, eligible: true, want: LevelRedTea},
		{name: "black tea threshold", xp: 30_000, eligible: true, want: LevelBlackTea},
		{name: "above highest threshold", xp: 90_000, eligible: true, want: LevelBlackTea},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LevelForXP(tt.xp, tt.eligible); got != tt.want {
				t.Fatalf("LevelForXP(%d, %t) = %d, want %d", tt.xp, tt.eligible, got, tt.want)
			}
		})
	}
}

func TestLevelForXPDemotesWhenCurrentXPFalls(t *testing.T) {
	if got := LevelForXP(15_000, true); got != LevelRedTea {
		t.Fatalf("level at 15,000 XP = %d, want %d", got, LevelRedTea)
	}
	if got := LevelForXP(14_999, true); got != LevelBlueTea {
		t.Fatalf("level after XP reversal = %d, want %d", got, LevelBlueTea)
	}
}
