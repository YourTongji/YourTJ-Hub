import SwiftUI
import WidgetKit

private let appGroup = "group.tj.yourtj.forumApp.widgets"
private let projectionKey = "schedule_widget_projection"
private let emptyStateKey = "schedule_widget_empty_state"
private let transparencyKey = "schedule_widget_transparency_percent"

private func widgetSurfaceBackgroundOpacity() -> Double {
    let defaults = UserDefaults(suiteName: appGroup)
    guard let value = defaults?.object(forKey: transparencyKey) as? Int else { return 0.91 }
    return Double(100 - min(max(value, 0), 15)) / 100
}

private struct WidgetBrandMark: View {
    var body: some View {
        Image("WidgetBrand")
            .resizable()
            .scaledToFit()
            .frame(width: 16, height: 16)
            .accessibilityHidden(true)
    }
}

private func safeText(_ value: String?) -> String? {
    guard let text = value?.trimmingCharacters(in: .whitespacesAndNewlines),
          !text.isEmpty, !["null", "undefined"].contains(text.lowercased()) else { return nil }
    return text
}

private struct Projection: Decodable {
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
        let key = Self.schoolDate.string(from: date)
        return days.first { $0.date == key }
    }

    func state(at date: Date) -> (String, Course?) {
        if date.timeIntervalSince(generatedAt) > 7 * 86_400 { return ("stale", nil) }
        guard let day = day(at: date) else { return ("needsRefresh", nil) }
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
        if let future { return ("upcoming", future.1, future.0) }
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

private struct ScheduleEntry: TimelineEntry {
    let date: Date
    let projection: Projection?
    let emptyState: String
}

private struct ScheduleProvider: TimelineProvider {
    private func entry(at date: Date) -> ScheduleEntry {
        let defaults = UserDefaults(suiteName: appGroup)
        return ScheduleEntry(
            date: date,
            projection: Projection.load(),
            emptyState: normalizedEmptyState(defaults?.string(forKey: emptyStateKey))
        )
    }

    func placeholder(in context: Context) -> ScheduleEntry {
        ScheduleEntry(date: Date(), projection: nil, emptyState: "needsRefresh")
    }

    func getSnapshot(in context: Context, completion: @escaping (ScheduleEntry) -> Void) {
        completion(entry(at: Date()))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<ScheduleEntry>) -> Void) {
        let now = Date()
        let projection = Projection.load()
        let dates = [now] + (projection?.timelineDates(after: now) ?? [])
        let emptyState = normalizedEmptyState(UserDefaults(suiteName: appGroup)?.string(forKey: emptyStateKey))
        completion(Timeline(entries: dates.map {
            ScheduleEntry(date: $0, projection: projection, emptyState: emptyState)
        }, policy: .never))
    }
}

private func normalizedEmptyState(_ value: String?) -> String {
    value == "needsData" ? "needsRefresh" : value ?? "needsRefresh"
}

private func title(_ state: String) -> String {
    let zh = Locale.current.languageCode == "zh"
    if zh {
        switch state {
        case "inClass": return "正在上课"
        case "upcoming", "break": return "下一节"
        case "finished": return "今日课程已结束"
        case "noClasses": return "今天暂无课程安排"
        case "holiday": return "今天放假"
        case "noneUpcoming": return "近期暂无课程安排"
        case "stale": return "课表可能已更新"
        case "unbound": return "绑定同济账号后显示课表"
        case "authorizationRequired": return "校园授权已失效，请打开 YourTJ 重新授权"
        case "signedOut": return "请登录并绑定同济账号"
        default: return "需要更新课表"
        }
    }
    switch state {
    case "inClass": return "In class"
    case "upcoming", "break": return "Up next"
    case "finished": return "Classes finished for today"
    case "noClasses": return "No classes today"
    case "holiday": return "No classes today"
    case "noneUpcoming": return "No classes in the next 8 days"
    case "stale": return "Schedule may have changed"
    case "unbound": return "Bind your Tongji account to show classes"
    case "authorizationRequired": return "Open YourTJ to restore campus access"
    case "signedOut": return "Sign in and bind your Tongji account"
    default: return "Schedule needs an update"
    }
}

private func support(_ state: String, hasProjection: Bool) -> String? {
    let zh = Locale.current.languageCode == "zh"
    switch state {
    case "stale": return zh ? "打开 YourTJ 刷新课表" : "Open YourTJ to refresh"
    case "needsRefresh":
        return zh
            ? (hasProjection ? "打开 YourTJ 刷新课表" : "打开 YourTJ 获取课表")
            : (hasProjection ? "Open YourTJ to refresh" : "Open YourTJ to get your schedule")
    case "noneUpcoming":
        return zh ? "打开 YourTJ 查看完整课表" : "Open YourTJ to view your full schedule"
    default: return nil
    }
}

private func courseTime(_ state: String, course: Projection.Course) -> String {
    if state == "inClass" {
        return Locale.current.languageCode == "zh"
            ? "至 \(Projection.clock.string(from: course.endAt))"
            : "Until \(Projection.clock.string(from: course.endAt))"
    }
    return "\(Projection.clock.string(from: course.startAt))–\(Projection.clock.string(from: course.endAt))"
}

private func distanceText(_ state: String, course: Projection.Course, now: Date) -> String {
    let target = state == "inClass" ? course.endAt : course.startAt
    let minutes = max(1, Int((target.timeIntervalSince(now) + 59) / 60))
    let zh = Locale.current.languageCode == "zh"
    let duration: String
    if minutes >= 24 * 60 {
        let days = minutes / (24 * 60)
        let hours = minutes % (24 * 60) / 60
        duration = zh
            ? (hours == 0 ? "\(days) 天" : "\(days) 天 \(hours) 小时")
            : (hours == 0 ? "\(days)d" : "\(days)d \(hours)h")
    } else if minutes >= 60 {
        let hours = minutes / 60
        let rest = minutes % 60
        duration = zh
            ? (rest == 0 ? "\(hours) 小时" : "\(hours) 小时 \(rest) 分钟")
            : (rest == 0 ? "\(hours)h" : "\(hours)h \(rest)m")
    } else {
        duration = zh ? "\(minutes) 分钟" : "\(minutes)m"
    }
    if zh { return "距\(state == "inClass" ? "下课" : "上课") \(duration)" }
    return state == "inClass" ? "Ends in \(duration)" : "Starts in \(duration)"
}

private func dateContext(_ date: String, now: Date) -> String {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(identifier: "Asia/Shanghai")!
    let today = Projection.schoolDate.string(from: now)
    let tomorrow = calendar.date(byAdding: .day, value: 1, to: now).map {
        Projection.schoolDate.string(from: $0)
    }
    let relative: String?
    if date == today {
        relative = Locale.current.languageCode == "zh" ? "今天" : "Today"
    } else if date == tomorrow {
        relative = Locale.current.languageCode == "zh" ? "明天" : "Tomorrow"
    } else {
        relative = nil
    }
    guard let parsed = Projection.schoolDate.date(from: date) else {
        return [relative, date].compactMap { $0 }.joined(separator: " · ")
    }
    let zh = Locale.current.languageCode == "zh"
    let formatter = DateFormatter()
    formatter.locale = zh ? Locale(identifier: "zh_CN") : Locale.current
    formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
    formatter.dateFormat = "EEE"
    let weekday = formatter.string(from: parsed)
    formatter.dateFormat = zh ? "M月d日" : "MMM d"
    let shortDate = formatter.string(from: parsed)
    return "\([relative, weekday].compactMap { $0 }.joined(separator: " ")) · \(shortDate)"
}

private func scheduleAccessibilityText(
    state: String,
    today: Projection.Day?,
    tomorrow: Projection.Day?
) -> String {
    let zh = Locale.current.languageCode == "zh"
    return [
        title(state),
        today.map { dayAccessibilityText(zh ? "今天" : "Today", day: $0) },
        tomorrow.map { dayAccessibilityText(zh ? "明天" : "Tomorrow", day: $0) },
    ].compactMap { $0 }.joined(separator: "，")
}

private func dayAccessibilityText(_ heading: String, day: Projection.Day) -> String {
    let courses = day.courses.map {
        [$0.name, courseTime("upcoming", course: $0), $0.campus, $0.room, $0.teacher]
            .filter { !$0.isEmpty }.joined(separator: "，")
    }.joined(separator: "，")
    let empty: String?
    if courses.isEmpty {
        switch day.kind {
        case "holiday": empty = day.adjustmentLabel ?? title("holiday")
        case "unknown": empty = title("needsRefresh")
        default: empty = title("noClasses")
        }
    } else {
        empty = nil
    }
    return [heading, dayHeaderText(day), courses.isEmpty ? nil : courses, empty]
        .compactMap { $0 }.joined(separator: "，")
}

private func dayHeaderText(_ day: Projection.Day) -> String {
    let zh = Locale.current.languageCode == "zh"
    let locale = zh ? Locale(identifier: "zh_CN") : Locale.current
    let timeZone = TimeZone(identifier: "Asia/Shanghai")
    let parsed = Projection.schoolDate.date(from: day.date)
    let dateFormatter = DateFormatter()
    dateFormatter.locale = locale
    dateFormatter.timeZone = timeZone
    dateFormatter.dateFormat = zh ? "M月d日" : "MMM d"
    let weekdayFormatter = DateFormatter()
    weekdayFormatter.locale = locale
    weekdayFormatter.timeZone = timeZone
    weekdayFormatter.dateFormat = "EEEE"
    return [
        parsed.map { dateFormatter.string(from: $0) } ?? day.date,
        day.week.map { zh ? "第\($0)周" : "Week \($0)" },
        parsed.map { weekdayFormatter.string(from: $0) },
        day.adjustmentLabel,
    ].compactMap { $0 }.joined(separator: " · ")
}

private let courseColors: [Color] = [
    .red, .orange, .yellow, .green,
    Color(red: 0.149, green: 0.486, blue: 0.471), .blue,
    Color(red: 0.447, green: 0.337, blue: 0.710), .pink,
]

private struct NextClassView: View {
    let entry: ScheduleEntry

    var body: some View {
        let state = entry.projection?.nextClass(at: entry.date) ?? (entry.emptyState, nil, nil)
        let body = state.1?.name ?? title(state.0)
        let context = state.2.map { dateContext($0.date, now: entry.date) }
        let time = state.1.map { courseTime(state.0, course: $0) }
        let distance = state.1.map { distanceText(state.0, course: $0, now: entry.date) }
        let place = state.1.map {
            [$0.campus, $0.room, $0.teacher].filter { !$0.isEmpty }.joined(separator: " · ")
        }
        let updated = entry.projection.map { "更新于 \(Projection.clock.string(from: $0.generatedAt))" }
        let accessibility = [
            title(state.0), context, state.1?.name, time, state.1?.campus, state.1?.room,
            state.1?.teacher, distance, updated,
        ].compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: "，")
        VStack(alignment: .leading, spacing: 4) {
            if state.1 != nil {
                HStack(spacing: 4) {
                    Text(title(state.0)).font(.caption.weight(.semibold)).foregroundColor(.accentColor)
                    Spacer(minLength: 4)
                    WidgetBrandMark()
                }
                if let context {
                    Text(context).font(.caption2).foregroundColor(.secondary)
                }
            }
            HStack(spacing: 4) {
                Text(body).font(.headline).lineLimit(2).layoutPriority(1)
                if state.1 == nil {
                    Spacer(minLength: 4)
                    WidgetBrandMark()
                }
            }
            if let place, !place.isEmpty {
                Text(place).font(.caption2).foregroundColor(.secondary).lineLimit(2)
            }
            if state.1 != nil {
                HStack(spacing: 6) {
                    if let time { Text(time).font(.caption2.monospacedDigit()).foregroundColor(.secondary) }
                    Spacer(minLength: 4)
                    if let distance { Text(distance).font(.caption2.weight(.medium)).foregroundColor(.accentColor) }
                }
            }
            if let support = support(state.0, hasProjection: entry.projection != nil) {
                Text(support).font(.caption2).foregroundColor(.secondary)
            }
            if let updated {
                Text(updated).font(.caption2).foregroundColor(.secondary)
            }
            if let course = state.1 {
                Capsule().fill(courseColors[course.colorSlot - 1]).frame(height: 4).widgetAccent()
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text(accessibility))
        .widgetURL(URL(string: "yourtj://campus/today?focus=\(state.1?.stableId ?? "")&homeWidget=true"))
        .widgetContainerBackground()
    }
}

private struct TodayScheduleView: View {
    @Environment(\.widgetFamily) private var family
    let entry: ScheduleEntry

    var body: some View {
        let projection = entry.projection
        let state = projection?.state(at: entry.date) ?? (entry.emptyState, nil)
        let today = projection?.day(at: entry.date)
        VStack(alignment: .leading, spacing: 6) {
            if family == .systemLarge {
                HStack(alignment: .top, spacing: 0) {
                    LargeDayColumn(title: "今天", day: today, currentId: state.1?.id, hasProjection: projection != nil)
                        .frame(maxWidth: .infinity, alignment: .topLeading)
                        .padding(.trailing, 6)
                    Divider().overlay(Color.secondary.opacity(0.22))
                    LargeDayColumn(title: "明天", day: tomorrow(in: projection), currentId: nil, hasProjection: projection != nil)
                        .frame(maxWidth: .infinity, alignment: .topLeading)
                        .padding(.leading, 6)
                }
                Spacer(minLength: 0)
                if let generatedAt = projection?.generatedAt {
                    Text("更新于 \(Projection.clock.string(from: generatedAt))")
                        .font(.caption2).foregroundColor(.secondary)
                }
            } else {
                MediumDayColumn(day: today, state: state, hasProjection: projection != nil)
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text(scheduleAccessibilityText(
            state: state.0,
            today: today,
            tomorrow: family == .systemLarge ? tomorrow(in: projection) : nil
        )))
        .widgetURL(URL(string: "yourtj://campus/today?homeWidget=true"))
        .widgetContainerBackground()
    }

    private func tomorrow(in projection: Projection?) -> Projection.Day? {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "Asia/Shanghai")!
        guard let projection = projection,
              let tomorrow = calendar.date(byAdding: .day, value: 1, to: entry.date) else {
            return nil
        }
        let date = Projection.schoolDate.string(from: tomorrow)
        return projection.days.first { $0.date == date }
    }
}

private struct MediumDayColumn: View {
    let day: Projection.Day?
    let state: (String, Projection.Course?)
    let hasProjection: Bool

    var body: some View {
        let visible = Array((day?.courses ?? []).prefix(3))
        VStack(alignment: .leading, spacing: 4) {
            DayHeader(day: day)
            if visible.isEmpty {
                EmptyDay(state: state.0, day: day, hasProjection: hasProjection)
            } else {
                ForEach(visible) { course in CourseRow(course: course, current: state.1?.id == course.id) }
                Remaining(count: (day?.courses.count ?? 0) - visible.count)
            }
        }
    }
}

private struct LargeDayColumn: View {
    let title: String
    let day: Projection.Day?
    let currentId: String?
    let hasProjection: Bool

    var body: some View {
        let visible = Array((day?.courses ?? []).prefix(2))
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 4) {
                Text(title).font(.headline)
                Spacer(minLength: 0)
                if title == "明天" { WidgetBrandMark() }
            }
            DayHeader(day: day, showBrand: false)
            if visible.isEmpty {
                EmptyDay(state: emptyState, day: day, hasProjection: hasProjection)
            } else {
                ForEach(visible) { course in CourseRow(course: course, current: currentId == course.id) }
                Remaining(count: (day?.courses.count ?? 0) - visible.count)
            }
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text(accessibilityText))
    }

    private var accessibilityText: String {
        let zh = Locale.current.languageCode == "zh"
        let heading = zh ? "\(title)课表" : (title == "今天" ? "Today schedule" : "Tomorrow schedule")
        guard let day else {
            return "\(heading)，\(zh ? "需要更新课表" : "Schedule needs an update")"
        }
        return dayAccessibilityText(heading, day: day)
    }

    private var emptyState: String {
        switch day?.kind {
        case "holiday": return "holiday"
        case "unknown", nil: return "needsRefresh"
        default: return "noClasses"
        }
    }
}

private struct DayHeader: View {
    let day: Projection.Day?
    var showBrand = true

    var body: some View {
        HStack(spacing: 4) {
            Text(day.map(dayHeaderText) ?? "")
                .font(.caption2.weight(.medium))
                .foregroundColor(.secondary)
                .lineLimit(2)
                .fixedSize(horizontal: false, vertical: true)
            if showBrand {
                Spacer(minLength: 0)
                WidgetBrandMark()
            }
        }
    }
}

private struct EmptyDay: View {
    let state: String
    let day: Projection.Day?
    let hasProjection: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(state == "holiday" ? day?.adjustmentLabel ?? title(state) : title(state))
                .font(.subheadline.weight(.semibold))
            if let support = support(state, hasProjection: hasProjection) {
                Text(support).font(.caption2).foregroundColor(.secondary)
            }
        }
    }
}

private struct CourseRow: View {
    let course: Projection.Course
    let current: Bool

    var body: some View {
        HStack(alignment: .top, spacing: 6) {
            Capsule().fill(courseColors[course.colorSlot - 1])
                .frame(width: 4, height: 48).widgetAccent()
            VStack(alignment: .leading, spacing: 2) {
                Text(course.name)
                    .font(.footnote.weight(current ? .semibold : .medium))
                    .foregroundColor(current ? .accentColor : .primary)
                    .lineLimit(2)
                let place = [course.campus, course.room, course.teacher]
                    .filter { !$0.isEmpty }.joined(separator: " · ")
                if !place.isEmpty { Text(place).font(.caption2).foregroundColor(.secondary) }
                Text("\(Projection.clock.string(from: course.startAt))–\(Projection.clock.string(from: course.endAt))")
                    .font(.caption2.monospacedDigit()).foregroundColor(.secondary)
            }
        }
    }
}

private struct Remaining: View {
    let count: Int

    var body: some View {
        if count > 0 {
            Text(Locale.current.languageCode == "zh" ? "还有 \(count) 门课程" : "\(count) more classes")
                .font(.caption2).foregroundColor(.secondary)
        }
    }
}

private extension View {
    @ViewBuilder func widgetAccent() -> some View {
        if #available(iOSApplicationExtension 16.0, *) {
            widgetAccentable()
        } else {
            self
        }
    }

    @ViewBuilder func widgetContainerBackground() -> some View {
        let opacity = widgetSurfaceBackgroundOpacity()
        if #available(iOSApplicationExtension 17.0, *) {
            containerBackground(for: .widget) {
                ZStack {
                    ContainerRelativeShape().fill(.ultraThinMaterial)
                    ContainerRelativeShape().fill(Color(.systemBackground).opacity(opacity))
                }
            }
        } else if #available(iOSApplicationExtension 15.0, *) {
            padding().background {
                ZStack {
                    ContainerRelativeShape().fill(.ultraThinMaterial)
                    ContainerRelativeShape().fill(Color(.systemBackground).opacity(opacity))
                }
            }
        } else {
            padding()
                .background(Color(.secondarySystemBackground).opacity(opacity))
                .clipShape(ContainerRelativeShape())
        }
    }
}

struct NextClassWidget: Widget {
    let kind = "NextClassWidget"
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: ScheduleProvider()) { NextClassView(entry: $0) }
            .configurationDisplayName("下一节课")
            .description("离线显示当前或下一节校园课程。")
            .supportedFamilies([.systemSmall])
    }
}

struct TodayScheduleWidget: Widget {
    let kind = "TodayScheduleWidget"
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: ScheduleProvider()) { TodayScheduleView(entry: $0) }
            .configurationDisplayName("今日课表")
            .description("离线显示今天与明天的课程及调休状态。")
            .supportedFamilies([.systemMedium, .systemLarge])
    }
}

@main
struct ScheduleWidgetsBundle: WidgetBundle {
    var body: some Widget { NextClassWidget(); TodayScheduleWidget() }
}
