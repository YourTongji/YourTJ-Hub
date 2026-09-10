package courseservice

import (
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

type OwnReviewItem struct {
	Review        ReviewPayload `json:"review"`
	CourseId      uint64        `json:"courseId"`
	CourseName    string        `json:"courseName"`
	CourseCode    string        `json:"courseCode"`
	Hidden        bool          `json:"hidden"`
	CanOpenCourse bool          `json:"canOpenCourse"`
}

type OwnReviewPage struct {
	List       []OwnReviewItem `json:"list"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

// ListOwnReviews uses the session identity, never a caller-supplied author ID.
func ListOwnReviews(userID, beforeID uint64, pageSize int) (OwnReviewPage, error) {
	result := OwnReviewPage{List: []OwnReviewItem{}}
	if userID == 0 {
		return result, ErrReviewNotOwned
	}
	if pageSize <= 0 {
		pageSize = DefaultReviewPageSize
	}
	if pageSize > MaxReviewPageSize {
		pageSize = MaxReviewPageSize
	}
	records, err := course.ListOwnedReviews(userID, beforeID, pageSize+1)
	if err != nil {
		return result, err
	}
	if len(records) > pageSize {
		records = records[:pageSize]
		result.NextCursor = strconv.FormatUint(records[len(records)-1].Id, 10)
	}
	entities := make([]course.ReviewEntity, 0, len(records))
	for _, r := range records {
		entities = append(entities, r.ReviewEntity)
	}
	payloads, err := listReviewPayloads(entities, userID)
	if err != nil {
		return result, err
	}
	for i, r := range records {
		hidden := r.Status != course.ReviewStatusVisible
		payloads[i].Viewer.CanEdit = !hidden
		result.List = append(result.List, OwnReviewItem{Review: payloads[i], CourseId: r.CourseId, CourseName: r.CourseName, CourseCode: r.CourseCode, Hidden: hidden, CanOpenCourse: r.CanOpenCourse && !hidden})
	}
	return result, nil
}
