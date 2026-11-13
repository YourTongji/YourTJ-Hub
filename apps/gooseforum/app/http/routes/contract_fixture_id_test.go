package routes

import (
	"testing"
	"testing/synctest"
)

func TestContractFixtureIDsAdvanceWithFrozenClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first := contractTestID()
		second := contractTestID()
		if second <= first+30 {
			t.Fatalf("fixture IDs %d and %d leave no space for derived IDs", first, second)
		}
	})
}
