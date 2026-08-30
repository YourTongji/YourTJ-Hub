package courseservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// HotInstructor 热门教师（目录页右侧栏，issue #331）。
type HotInstructor struct {
	Id          uint64   `json:"id"`
	Name        string   `json:"name"`
	Department  string   `json:"department,omitempty"`
	RatingAvg   *float64 `json:"ratingAvg,omitempty"`
	ReviewCount int      `json:"reviewCount,omitempty"`
}

// ListHotInstructors 按评分提取热门教师（前 limit 名）。
func ListHotInstructors(limit int) ([]HotInstructor, error) {
	rows, err := course.ListHotInstructors(limit)
	if err != nil {
		return nil, err
	}
	out := make([]HotInstructor, 0, len(rows))
	for _, r := range rows {
		var avg *float64
		if r.RatingCount > 0 {
			v := float64(r.RatingSum) / float64(r.RatingCount)
			avg = &v
		}
		out = append(out, HotInstructor{
			Id:          r.Id,
			Name:        r.Name,
			Department:  r.Department,
			RatingAvg:   avg,
			ReviewCount: int(r.ReviewCount),
		})
	}
	return out, nil
}
