import Foundation
import XCTest

final class ScheduleProjectionTests: XCTestCase {
    private func time(_ clock: String, day: String = "2026-09-30") -> Date {
        ISO8601DateFormatter().date(from: "\(day)T\(clock):00+08:00")!
    }

    private func fixture() throws -> Projection {
        func course(_ name: String, _ start: String, _ end: String, day: String = "2026-09-30") -> [String: Any] {
            ["stableId": name, "name": name, "teacher": "Teacher", "room": "Room",
             "campus": "Campus", "colorSlot": 1,
             "startAt": "\(day)T\(start):00+08:00", "endAt": "\(day)T\(end):00+08:00"]
        }
        let data = try JSONSerialization.data(withJSONObject: [
            "schemaVersion": 2,
            "identity": ["siteKey": "site", "accountScope": "account", "bindingRevision": "binding"],
            "generatedAt": "2026-09-30T07:00:00+08:00", "schoolDate": "2026-09-30",
            "timezone": "Asia/Shanghai", "semester": ["id": "term", "week": 5],
            "days": [
                ["date": "2026-09-30", "week": 5, "source": "server-adjusted", "kind": "normal",
                 "courses": [course("A", "08:00", "09:00"), course("B", "10:00", "11:00"),
                             course("C", "14:00", "15:00"), course("D", "16:00", "17:00")]],
                ["date": "2026-10-01", "week": 5, "source": "derived", "kind": "normal",
                 "courses": [course("Tomorrow", "08:00", "09:00", day: "2026-10-01")]],
                ["date": "2026-10-02", "week": 5, "source": "derived", "kind": "holiday", "courses": []],
                ["date": "2026-10-03", "week": 5, "source": "derived", "kind": "normal", "courses": []],
            ],
        ])
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(Projection.self, from: data)
    }

    func testVisibleDayRemovesFinishedCoursesAtTheirExactEnd() throws {
        let projection = try fixture()
        XCTAssertEqual(projection.day(at: time("07:00"))?.courses.map(\.name), ["A", "B", "C", "D"])
        XCTAssertEqual(projection.day(at: time("09:00"))?.courses.map(\.name), ["B", "C", "D"])
        XCTAssertEqual(projection.day(at: time("11:00"))?.courses.map(\.name), ["C", "D"])
        XCTAssertEqual(projection.day(at: time("17:00"))?.courses.map(\.name), [])
        // Filtering the view must not mutate the stored timetable or turn a completed day into a free day.
        XCTAssertEqual(projection.days[0].courses.count, 4)
        XCTAssertEqual(projection.state(at: time("17:00")).0, "finished")
    }

    func testCurrentCourseStaysVisibleUntilItEnds() throws {
        let projection = try fixture()
        for clock in ["10:00", "10:30", "10:59"] {
            XCTAssertEqual(projection.state(at: time(clock)).0, "inClass")
            XCTAssertEqual(projection.day(at: time(clock))?.courses.map(\.name), ["B", "C", "D"])
        }
        XCTAssertEqual(projection.state(at: time("11:00")).0, "break")
    }

    func testSmallScheduleNeverBackfillsEarlierCourses() throws {
        let projection = try fixture()
        for clock in ["15:30", "16:00", "16:30"] {
            let next = projection.nextClass(at: time(clock))
            XCTAssertEqual(next.1?.name, "D")
            XCTAssertEqual(next.2?.courses.map(\.name), ["D"])
        }
        let tomorrow = projection.nextClass(at: time("17:00"))
        XCTAssertEqual(tomorrow.1?.name, "Tomorrow")
        XCTAssertEqual(tomorrow.2?.date, "2026-10-01")
    }

    func testTimelineAdvancesThroughCourseEndsAndShanghaiMidnight() throws {
        let projection = try fixture()
        let dates = projection.timelineDates(after: time("08:30"))
        XCTAssertEqual(dates, Array(Set(dates)).sorted())
        XCTAssertTrue(dates.allSatisfy { $0 > time("08:30") })
        for clock in ["09:00", "10:00", "11:00", "14:00", "15:00", "16:00", "17:00"] {
            XCTAssertTrue(dates.contains(time(clock)))
        }
        let midnight = time("00:00", day: "2026-10-01")
        XCTAssertTrue(dates.contains(midnight))
        XCTAssertEqual(projection.day(at: midnight)?.date, "2026-10-01")
        XCTAssertEqual(projection.day(at: midnight)?.courses.map(\.name), ["Tomorrow"])
        XCTAssertEqual(projection.day(at: time("23:59"))?.courses.count, 0)
    }

    func testEmptyHolidayAndUnknownDaysKeepDistinctStates() throws {
        let projection = try fixture()
        XCTAssertEqual(projection.state(at: time("12:00", day: "2026-10-02")).0, "holiday")
        XCTAssertEqual(projection.state(at: time("12:00", day: "2026-10-03")).0, "noClasses")
        XCTAssertEqual(projection.state(at: time("12:00", day: "2026-10-04")).0, "needsRefresh")
    }
}
