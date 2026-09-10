package networkAccessLog

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
)

// Record 写入一条访问日志。调用方应吞掉错误，避免影响主请求。
func Record(entity Entity) error {
	return builder().Create(&entity).Error
}

// ExpireBefore 删除 created_at < before 的旧日志，返回删除行数。
func ExpireBefore(before time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 2000
	}
	total := 0
	for {
		var ids []uint64
		if err := builder().Select("id").Where("created_at < ?", before).
			Order("created_at ASC").Limit(limit).Pluck("id", &ids).Error; err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}
		result := builder().Where(queryopt.In("id", ids)).Delete(&Entity{})
		if result.Error != nil {
			return total, result.Error
		}
		total += int(result.RowsAffected)
		if len(ids) < limit {
			return total, nil
		}
	}
}
