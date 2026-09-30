import Foundation

let appGroup = "group.tj.yourtj.forumApp.widgets"
private let projectionKey = "schedule_widget_projection"

private func safeText(_ value: String?) -> String? {
    guard let text = value?.trimmingCharacters(in: .whitespacesAndNewlines),
          !text.isEmpty, !["null", "undefined"].contains(text.lowercased()) else { return nil }
    return text
}

struct Projection: Decodable {
    struct Identity: Decodable {
        let siteKey: String
        let accountScope: String
        let bindingRevision: String

        init(from decoder: Decoder) throws {
            let values = try decoder.container(keyedBy: CodingKeys.self)
            guard let siteKey = safeText(try values.decodeIfPresent(String.self, forKey: .siteKey)),
                  let accountScope = safeText(try values.decodeIfPresent(String.self, forKey: .accountScope)),
                  let bindingRevision = safeText(try values.decodeIfPresent(String.self, forKey: .bindingRevision)) else {
                throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Invalid identity"))
            }
            self.siteKey = siteKey
            self.accountScope = accountScope
            self.bindingRevision = bindingRevision
        }

        private enum CodingKeys: String, CodingKey { case siteKey, accountScope, bindingRevision }
    }

    struct Semester: Decodable {
        let id: String
        let week: Int?

        init(from decoder: Decoder) throws {
            let values = try decoder.container(keyedBy: CodingKeys.self)
            id = safeText(try values.decodeIfPresent(String.self, forKey: .id)) ?? ""
            week = try values.decodeIfPresent(Int.self, forKey: .week).flatMap { $0 > 0 ? $0 : nil }
        }

        private enum CodingKeys: String, CodingKey { case id, week }
    }

    struct Day: Decodable {
        let date: String
        let week: Int?
        let source: String
        let kind: String
        let adjustmentLabel: String?
        let courses: [Course]

        init(from decoder: Decoder) throws {
            let values = try decoder.container(keyedBy: CodingKeys.self)
            guard let date = safeText(try values.decodeIfPresent(String.self, forKey: .date)) else {
                throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Invalid day"))
            }
            self.date = date
            week = try values.decodeIfPresent(Int.self, forKey: .week).flatMap { $0 > 0 ? $0 : nil }
            source = safeText(try values.decodeIfPresent(String.self, forKey: .source)) ?? "unknown"
            let decodedKind = safeText(try values.decodeIfPresent(String.self, forKey: .kind)) ?? "unknown"
            kind = ["none", "normal", "holiday", "moved", "makeup", "unknown"].contains(decodedKind)
                ? decodedKind : "unknown"
            adjustmentLabel = safeText(try values.decodeIfPresent(String.self, forKey: .adjustmentLabel))
            courses = try values.decode([Course].self, forKey: .courses).sorted { $0.startAt < $1.startAt }
        }

        init(date: String, week: Int?, source: String, kind: String, adjustmentLabel: String?, courses: [Course]) {
            self.date = date
            self.week = week
            self.source = source
            self.kind = kind
            self.adjustmentLabel = safeText(adjustmentLabel)
            self.courses = courses.sorted { $0.startAt < $1.startAt }
        }

        private enum CodingKeys: String, CodingKey { case date, week, source, kind, adjustmentLabel, courses }

        /// A display-only copy: ongoing courses remain until their end boundary.
        func remaining(at now: Date) -> Day {
            Day(date: date, week: week, source: source, kind: kind,
                adjustmentLabel: adjustmentLabel, courses: courses.filter { $0.endAt > now })
        }
    }

    struct LegacyToday: Decodable {
        let kind: String
        let adjustmentLabel: String?
        let courses: [Course]
    }

    struct Course: Decodable, Identifiable {
        var id: String { stableId }
        let stableId: String
        let name: String
        let teacher: String
        let room: String
        let campus: String
        let startAt: Date
        let endAt: Date
        let colorSlot: Int

        init(from decoder: Decoder) throws {
            let values = try decoder.container(keyedBy: CodingKeys.self)
            guard let stableId = safeText(try values.decodeIfPresent(String.self, forKey: .stableId)),
                  let name = safeText(try values.decodeIfPresent(String.self, forKey: .name)) else {
                throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Invalid course"))
            }
            self.stableId = stableId
            self.name = name
            teacher = safeText(try values.decodeIfPresent(String.self, forKey: .teacher)) ?? ""
            room = safeText(try values.decodeIfPresent(String.self, forKey: .room)) ?? ""
            campus = safeText(try values.decodeIfPresent(String.self, forKey: .campus)) ?? ""
            startAt = try values.decode(Date.self, forKey: .startAt)
            endAt = try values.decode(Date.self, forKey: .endAt)
            colorSlot = try values.decode(Int.self, forKey: .colorSlot)
        }

        private enum CodingKeys: String, CodingKey {
            case stableId, name, teacher, room, campus, startAt, endAt, colorSlot
        }
    }

    let schemaVersion: Int
    let identity: Identity
    let generatedAt: Date
    let schoolDate: String
    let timezone: String
    let semester: Semester
    let days: [Day]

    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        schemaVersion = try values.decode(Int.self, forKey: .schemaVersion)
        identity = try values.decode(Identity.self, forKey: .identity)
        generatedAt = try values.decode(Date.self, forKey: .generatedAt)
        schoolDate = safeText(try values.decodeIfPresent(String.self, forKey: .schoolDate)) ?? ""
        timezone = try values.decode(String.self, forKey: .timezone)
        semester = try values.decode(Semester.self, forKey: .semester)
        if schemaVersion == 1 {
            let today = try values.decode(LegacyToday.self, forKey: .today)
            days = [Day(
                date: schoolDate,
                week: semester.week,
                source: "server-adjusted",
                kind: safeText(today.kind) ?? "unknown",
                adjustmentLabel: today.adjustmentLabel,
                courses: today.courses
            )]
        } else {
            days = try values.decode([Day].self, forKey: .days)
        }
    }

    private enum CodingKeys: String, CodingKey {
        case schemaVersion, identity, generatedAt, schoolDate, timezone, semester, days, today
    }

    static func load() -> Projection? {
        guard let raw = UserDefaults(suiteName: appGroup)?.string(forKey: projectionKey),
              let data = raw.data(using: .utf8) else { return nil }
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        guard let value = try? decoder.decode(Projection.self, from: data),
              (1...2).contains(value.schemaVersion), value.timezone == "Asia/Shanghai",
              !value.days.isEmpty,
              value.days.allSatisfy({ Projection.schoolDate.date(from: $0.date) != nil }),
              value.days.flatMap(\.courses).allSatisfy({
                  (1...8).contains($0.colorSlot) && $0.endAt > $0.startAt
              }) else { return nil }
        return value
    }

    func day(at date: Date) -> Day? {
        storedDay(at: date)?.remaining(at: date)
    }

    private func storedDay(at date: Date) -> Day? {
        let key = Self.schoolDate.string(from: date)
        return days.first { $0.date == key }
    }

    func state(at date: Date) -> (String, Course?) {
        if date.timeIntervalSince(generatedAt) > 7 * 86_400 { return ("stale", nil) }
        // Keep the full day for status so finishing the last course is distinct
        // from having no scheduled classes at all.
        guard let day = storedDay(at: date) else { return ("needsRefresh", nil) }
        if day.kind == "unknown" { return ("needsRefresh", nil) }
        if day.kind == "holiday" { return ("holiday", nil) }
        if day.courses.isEmpty { return ("noClasses", nil) }
        for (index, course) in day.courses.enumerated() {
            if date >= course.startAt && date < course.endAt { return ("inClass", course) }
            if date < course.startAt { return (index == 0 ? "upcoming" : "break", course) }
        }
        return ("finished", nil)
    }

    func nextClass(at date: Date) -> (String, Course?, Day?) {
        if date.timeIntervalSince(generatedAt) > 7 * 86_400 { return ("stale", nil, nil) }
        guard let today = day(at: date), today.kind != "unknown" else {
            return ("needsRefresh", nil, nil)
        }
        if let current = today.courses.first(where: { date >= $0.startAt && date < $0.endAt }) {
            return ("inClass", current, today)
        }
        let dateKey = Self.schoolDate.string(from: date)
        let future = days
            .filter { $0.date >= dateKey }
            .flatMap { day in day.courses.map { (day, $0) } }
            .filter { $0.1.startAt > date }
            .min { $0.1.startAt < $1.1.startAt }
        if let future { return ("upcoming", future.1, future.0.remaining(at: date)) }
        if days.count < 8 || days.contains(where: { $0.date >= dateKey && $0.kind == "unknown" }) {
            return ("needsRefresh", nil, nil)
        }
        return ("noneUpcoming", nil, nil)
    }

    func timelineDates(after now: Date) -> [Date] {
        let courseDates = days.flatMap(\.courses).flatMap { [$0.startAt, $0.endAt] }
        let midnights = days.compactMap { Self.schoolDate.date(from: $0.date) }
        return Set(courseDates + midnights + [generatedAt.addingTimeInterval(7 * 86_400)])
            .filter { $0 > now }.sorted()
    }

    static let schoolDate: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    static let clock: DateFormatter = {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
        formatter.dateFormat = "HH:mm"
        return formatter
    }()
}
