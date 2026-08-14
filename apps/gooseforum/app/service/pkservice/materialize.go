package pkservice

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
)

// materializeToCatalog 将已同步学期的 PK 课程物化到课程目录（course 域），默认 off。
// 跨域写入遵循边界规则：由课程域 owner（courseservice）的公开 API 完成。
func materializeToCatalog(ctx context.Context, calendarIds []uint64) (int, error) {
	report, err := courseservice.MaterializeFromPk(ctx, calendarIds)
	if err != nil {
		return 0, err
	}
	return report.CoursesInserted + report.CoursesUpdated, nil
}
