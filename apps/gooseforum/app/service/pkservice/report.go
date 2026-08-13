package pkservice

// SyncReport 一次 course-pk-sync 的运行报表（终端打印用）。
type SyncReport struct {
	CalendarIDs           []uint64 // 实际同步的学期
	TeachingClassInserted int      // 处理（upsert）的教学班行数
	BatchesCommitted      int      // 已提交的批量事务数
	FetchedPages          int      // 抓取页数
	TimeslotsRebuilt      int      // 重建的 teacher_timeslots 行数
	MaterializedCourses   int      // 物化到课程目录的课程数（--materialize 时）
	ResumedFromPage       int      // 断点续跑起始页（0 表示全量）
}
