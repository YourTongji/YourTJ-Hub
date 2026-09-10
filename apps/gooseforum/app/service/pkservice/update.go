package pkservice

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// GetLatestUpdate P11：最近一次同步日期（YYYY-MM-DD），无记录返回空字符串。
func GetLatestUpdate() (string, error) {
	ts, err := pk.GetLatestFetchTime()
	if err != nil {
		return "", err
	}
	if ts <= 0 {
		return "", nil
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02"), nil
}
