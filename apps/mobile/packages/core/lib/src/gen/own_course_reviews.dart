import 'course_review.dart';

/// Mirrors the authenticated OwnCourseReviewItem OpenAPI schema.
class OwnCourseReviewItem {
  const OwnCourseReviewItem({
    required this.review,
    required this.courseId,
    required this.courseName,
    required this.courseCode,
    required this.hidden,
    required this.canOpenCourse,
  });
  final ReviewPayload review;
  final int courseId;
  final String courseName;
  final String courseCode;
  final bool hidden;
  final bool canOpenCourse;
  factory OwnCourseReviewItem.fromJson(Map<String, dynamic> json) =>
      OwnCourseReviewItem(
        review: ReviewPayload.fromJson(
          Map<String, dynamic>.from(json['review'] as Map),
        ),
        courseId: (json['courseId'] as num).toInt(),
        courseName: json['courseName'] as String,
        courseCode: json['courseCode'] as String,
        hidden: json['hidden'] as bool,
        canOpenCourse: json['canOpenCourse'] as bool,
      );
  OwnCourseReviewItem withReview(ReviewPayload value) => OwnCourseReviewItem(
    review: value,
    courseId: courseId,
    courseName: courseName,
    courseCode: courseCode,
    hidden: hidden,
    canOpenCourse: canOpenCourse,
  );
}

class OwnCourseReviewPage {
  const OwnCourseReviewPage({required this.list, this.nextCursor = ''});
  final List<OwnCourseReviewItem> list;
  final String nextCursor;
  factory OwnCourseReviewPage.fromJson(Map<String, dynamic> json) =>
      OwnCourseReviewPage(
        list: (json['list'] as List)
            .map(
              (item) => OwnCourseReviewItem.fromJson(
                Map<String, dynamic>.from(item as Map),
              ),
            )
            .toList(),
        nextCursor: json['nextCursor'] as String? ?? '',
      );
}
