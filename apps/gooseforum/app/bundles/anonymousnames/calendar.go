package anonymousnames

import "time"

// Shanghai has no seasonal clock changes in the supported contemporary dates.
var Shanghai = time.FixedZone("Asia/Shanghai", 8*60*60)

// Day returns the server-owned calendar date and the next midnight boundary.
func Day(now time.Time) (string, time.Time) {
	local := now.In(Shanghai)
	midnight := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, Shanghai)
	return local.Format("2006-01-02"), midnight
}

// Anniversary clamps February 29 to February 28 rather than Go's March 1
// normalization, preserving the selection's local wall-clock time.
func Anniversary(selected time.Time) time.Time {
	local := selected.In(Shanghai)
	day := local.Day()
	last := time.Date(local.Year()+1, local.Month()+1, 0, 0, 0, 0, 0, Shanghai).Day()
	if day > last {
		day = last
	}
	return time.Date(local.Year()+1, local.Month(), day, local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), Shanghai)
}
