import SwiftUI
import WidgetKit

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
            .widgetAccent()
            .accessibilityHidden(true)
    }
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

private func title(_ state: String, isTomorrow: Bool = false) -> String {
    let zh = Locale.current.languageCode == "zh"
    if zh {
        switch state {
        case "inClass": return "正在上课"
        case "upcoming", "break": return "下一节"
        case "finished": return "今日课程已结束"
        case "noClasses": return isTomorrow ? "明天暂无课程安排" : "今天暂无课程安排"
        case "holiday": return isTomorrow ? "明天放假" : "今天放假"
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
    case "noClasses", "holiday": return isTomorrow ? "No classes tomorrow" : "No classes today"
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

private func courseTime(_ course: Projection.Course) -> String {
    return "\(Projection.clock.string(from: course.startAt))–\(Projection.clock.string(from: course.endAt))"
}

private func compactRoom(_ room: String) -> String {
    let aliases = [
        "教学北楼": "北", "教学南楼": "南", "教学东楼": "东",
        "教学西楼": "西", "教学中楼": "中", "教学楼": "教",
    ]
    for (prefix, short) in aliases where room.hasPrefix(prefix) {
        return short + room.dropFirst(prefix.count).replacingOccurrences(of: " ", with: "")
    }
    return room.replacingOccurrences(of: " ", with: "")
}

private func scheduleAccessibilityText(
    state: String,
    today: Projection.Day?,
    tomorrow: Projection.Day?
) -> String {
    let zh = Locale.current.languageCode == "zh"
    return [
        title(state),
        today.map { dayAccessibilityText(zh ? "今天" : "Today", day: $0, emptyState: state) },
        tomorrow.map { dayAccessibilityText(zh ? "明天" : "Tomorrow", day: $0, isTomorrow: true) },
    ].compactMap { $0 }.joined(separator: "，")
}

private func dayAccessibilityText(_ heading: String, day: Projection.Day, emptyState: String? = nil, isTomorrow: Bool = false) -> String {
    let courses = day.courses.map {
        [$0.name, courseTime($0), $0.campus, $0.room, $0.teacher]
            .filter { !$0.isEmpty }.joined(separator: "，")
    }.joined(separator: "，")
    let empty: String?
    if courses.isEmpty {
        switch day.kind {
        case "holiday": empty = day.adjustmentLabel ?? title("holiday", isTomorrow: isTomorrow)
        case "unknown": empty = title("needsRefresh")
        default: empty = title(emptyState ?? "noClasses", isTomorrow: isTomorrow)
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

private func dayDate(_ day: Projection.Day?, fallbackDate: Date) -> Date {
    day.flatMap { Projection.schoolDate.date(from: $0.date) } ?? fallbackDate
}

private func shortDateText(_ day: Projection.Day?, fallbackDate: Date) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
    formatter.dateFormat = "M/d"
    return formatter.string(from: dayDate(day, fallbackDate: fallbackDate))
}

private func weekdayText(_ day: Projection.Day?, fallbackDate: Date) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale.current.languageCode == "zh" ? Locale(identifier: "zh_CN") : Locale.current
    formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
    formatter.dateFormat = "EEE"
    return formatter.string(from: dayDate(day, fallbackDate: fallbackDate))
}

private func weekText(_ day: Projection.Day?) -> String {
    let zh = Locale.current.languageCode == "zh"
    return day?.week.map { zh ? "第\($0)周" : "Week \($0)" }
        ?? (zh ? "课表待更新" : "Update needed")
}

private func shortWeekText(_ day: Projection.Day?) -> String {
    guard let week = day?.week else { return Locale.current.languageCode == "zh" ? "待更新" : "Update" }
    return Locale.current.languageCode == "zh" ? "\(week)周" : "W\(week)"
}

private func lastUpdatedText(_ date: Date) -> String {
    let zh = Locale.current.languageCode == "zh"
    let formatter = DateFormatter()
    formatter.locale = zh ? Locale(identifier: "zh_CN") : Locale.current
    formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
    formatter.dateFormat = zh ? "M/d HH:mm" : "MMM d HH:mm"
    return "\(zh ? "最后更新于" : "Updated") \(formatter.string(from: date))"
}

private func compactUpdatedText(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
    formatter.dateFormat = "M/d HH:mm"
    return "\(Locale.current.languageCode == "zh" ? "更新于" : "Updated")\n\(formatter.string(from: date))"
}

// sRGB mirrors resource/src/styles/tokens.css --gf-color-course-1...8.
// SwiftUI cannot read the web CSS variables inside a WidgetKit extension.
private let lightCourseStripeRGB: [UInt32] = [
    0x2A44BC, 0x005F67, 0x006630, 0x923100,
    0xAC0024, 0x3C3990, 0x005A99, 0x7F3E7A,
]
private let darkCourseStripeRGB: [UInt32] = [
    0x6FA2FF, 0x48B7BD, 0x59B47D, 0xD5A13C,
    0xF2716A, 0x9793E6, 0x32B3E6, 0xCB7FC5,
]

private func courseStripeColor(_ slot: Int, scheme: ColorScheme) -> Color {
    let colors = scheme == .dark ? darkCourseStripeRGB : lightCourseStripeRGB
    let rgb = colors[slot - 1]
    return Color(
        red: Double((rgb >> 16) & 0xFF) / 255,
        green: Double((rgb >> 8) & 0xFF) / 255,
        blue: Double(rgb & 0xFF) / 255
    )
}

private struct WidgetFooter: View {
    let projection: Projection?
    var compact = false

    var body: some View {
        if let projection {
            Text(compact ? compactUpdatedText(projection.generatedAt) : lastUpdatedText(projection.generatedAt))
                .font(.system(size: 11))
                .foregroundColor(.secondary)
                .lineLimit(2)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct NextClassView: View {
    let entry: ScheduleEntry

    var body: some View {
        let state = entry.projection?.nextClass(at: entry.date) ?? (entry.emptyState, nil, nil)
        let day = state.2 ?? entry.projection?.day(at: entry.date)
        let courses = state.1 == nil ? [] : Array((day?.courses ?? []).prefix(3))
        let accessibility = [
            title(state.0), day.map(dayHeaderText),
            courses.map { [$0.name, compactRoom($0.room), $0.teacher, courseTime($0)]
                .filter { !$0.isEmpty }.joined(separator: "，") }.joined(separator: "，"),
            entry.projection.map { lastUpdatedText($0.generatedAt) },
        ].compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: "，")
        VStack(alignment: .leading, spacing: 2) {
            HStack(alignment: .top, spacing: 4) {
                Text("\(shortWeekText(day)) \(weekdayText(day, fallbackDate: entry.date)) \(shortDateText(day, fallbackDate: entry.date))")
                    .font(.system(size: 11, weight: .medium))
                    .foregroundColor(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                Spacer(minLength: 0)
                WidgetBrandMark()
            }
            if !courses.isEmpty {
                ForEach(courses.indices, id: \.self) { index in
                    if index == 2 {
                        CompactCoursePreview(course: courses[index])
                    } else {
                        CourseRow(course: courses[index],
                                  current: courses[index].id == state.1?.id,
                                  compact: true, small: true)
                    }
                }
            } else {
                Text(title(state.0)).font(.subheadline.weight(.semibold)).lineLimit(3)
                if let support = support(state.0, hasProjection: entry.projection != nil) {
                    Text(support).font(.caption2).foregroundColor(.secondary).lineLimit(3)
                }
            }
            Spacer(minLength: 0)
            WidgetFooter(projection: entry.projection)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text(accessibility))
        .widgetURL(URL(string: "yourtj://campus/today?focus=\(state.1?.stableId ?? "")&homeWidget=true"))
        .widgetContainerBackground()
    }
}

private struct CompactCoursePreview: View {
    @Environment(\.colorScheme) private var colorScheme
    let course: Projection.Course

    var body: some View {
        let stripe = courseStripeColor(course.colorSlot, scheme: colorScheme)
        return HStack(alignment: .firstTextBaseline, spacing: 4) {
            Text(course.name).font(.system(size: 11, weight: .semibold)).foregroundColor(stripe)
            Spacer(minLength: 0)
            Text(Projection.clock.string(from: course.startAt))
                .font(.system(size: 11).monospacedDigit())
                .foregroundColor(.secondary)
        }
        .padding(.leading, 12)
        .padding(.trailing, 4)
        .padding(.vertical, 1)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(RoundedRectangle(cornerRadius: 7).fill(stripe.opacity(colorScheme == .dark ? 0.18 : 0.07)))
        .overlay(
            CourseStripe(color: stripe, verticalInset: 3),
            alignment: .leading
        )
    }
}

private struct TodayScheduleView: View {
    @Environment(\.widgetFamily) private var family
    let entry: ScheduleEntry

    var body: some View {
        let projection = entry.projection
        let state = projection?.state(at: entry.date) ?? (entry.emptyState, nil)
        let today = projection?.day(at: entry.date)
        let next = tomorrow(in: projection)
        VStack(alignment: .leading, spacing: 7) {
            if family == .systemLarge {
                HStack(alignment: .top, spacing: 12) {
                    LargeDayColumn(title: "今天", day: today, fallbackDate: entry.date,
                                   currentId: state.1?.id, hasProjection: projection != nil,
                                   showsBrandMark: false, currentState: state.0)
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                    LargeDayColumn(title: "明天", day: next,
                                   fallbackDate: nextDay(after: entry.date),
                                   currentId: nil, hasProjection: projection != nil,
                                   showsBrandMark: true)
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                }
            } else {
                MediumSchedule(entry: entry, today: today, tomorrow: next,
                               state: state, showsTomorrow: mediumShowsTomorrow(today))
            }
            if family == .systemLarge {
                Spacer(minLength: 0)
                WidgetFooter(projection: projection)
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text(scheduleAccessibilityText(
            state: state.0,
            today: today,
            tomorrow: family == .systemLarge || mediumShowsTomorrow(today) ? next : nil
        )))
        .widgetURL(URL(string: "yourtj://campus/today?homeWidget=true"))
        .widgetContainerBackground()
    }

    private func tomorrow(in projection: Projection?) -> Projection.Day? {
        guard let projection else { return nil }
        let date = Projection.schoolDate.string(from: nextDay(after: entry.date))
        return projection.days.first { $0.date == date }
    }
}

private func mediumShowsTomorrow(_ day: Projection.Day?) -> Bool {
    guard let day, day.kind != "unknown" else { return false }
    return day.courses.count <= 2
}

private func nextDay(after date: Date) -> Date {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(identifier: "Asia/Shanghai")!
    return calendar.date(byAdding: .day, value: 1, to: date) ?? date.addingTimeInterval(86_400)
}

private struct MediumSchedule: View {
    let entry: ScheduleEntry
    let today: Projection.Day?
    let tomorrow: Projection.Day?
    let state: (String, Projection.Course?)
    let showsTomorrow: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            ZStack(alignment: .topTrailing) {
                HStack(alignment: .top, spacing: 10) {
                    MediumDateHeading(day: today, fallbackDate: entry.date)
                        .frame(maxWidth: .infinity, alignment: .leading)
                    if showsTomorrow {
                        MediumDateHeading(day: tomorrow, fallbackDate: nextDay(after: entry.date))
                            .padding(.trailing, 22)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                WidgetBrandMark()
            }
            GeometryReader { geometry in
                let columnWidth = max(0, (geometry.size.width - 10) / 2)
                HStack(alignment: .top, spacing: 10) {
                    if showsTomorrow {
                        MediumDayColumn(day: today, state: state.0, currentId: state.1?.id,
                                        hasProjection: entry.projection != nil)
                            .frame(width: columnWidth, height: geometry.size.height, alignment: .topLeading)
                        MediumDayColumn(day: tomorrow, state: tomorrowState,
                                        currentId: nil, hasProjection: entry.projection != nil,
                                        isTomorrow: true)
                            .frame(width: columnWidth, height: geometry.size.height, alignment: .topLeading)
                    } else if let today, !today.courses.isEmpty {
                        MediumSplitCourses(courses: today.courses, currentId: state.1?.id)
                            .frame(width: geometry.size.width, height: geometry.size.height, alignment: .topLeading)
                    } else {
                        EmptyDay(state: state.0, day: today, hasProjection: entry.projection != nil)
                    }
                }
            }
            WidgetFooter(projection: entry.projection)
        }
    }

    private var tomorrowState: String {
        switch tomorrow?.kind {
        case "holiday": return "holiday"
        case "unknown", nil: return "needsRefresh"
        default: return "noClasses"
        }
    }
}

private struct MediumDateHeading: View {
    let day: Projection.Day?
    let fallbackDate: Date

    var body: some View {
        Text("\(shortWeekText(day))  \(weekdayText(day, fallbackDate: fallbackDate))  \(shortDateText(day, fallbackDate: fallbackDate))")
            .font(.system(size: 11, weight: .semibold))
            .foregroundColor(.secondary)
            .fixedSize(horizontal: false, vertical: true)
    }
}

private struct MediumDayColumn: View {
    let day: Projection.Day?
    let state: String
    let currentId: String?
    let hasProjection: Bool
    var isTomorrow = false

    var body: some View {
        Group {
            if let day, !day.courses.isEmpty {
                if #available(iOSApplicationExtension 16.0, *) {
                    ViewThatFits(in: .vertical) {
                        CourseRows(courses: day.courses, currentId: currentId,
                                   count: 2, compact: true, small: true)
                        CourseRows(courses: day.courses, currentId: currentId,
                                   count: 1, compact: true, small: true)
                    }
                } else {
                    CourseRows(courses: day.courses, currentId: currentId,
                               count: 1, compact: true, small: true)
                }
            } else if isTomorrow && state == "noClasses" {
                Text(Locale.current.languageCode == "zh" ? "明天暂无课程安排" : "No classes tomorrow")
                    .font(.system(size: 11, weight: .medium))
            } else {
                EmptyDay(state: state, day: day, hasProjection: hasProjection, isTomorrow: isTomorrow)
            }
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
    }
}

private struct MediumSplitCourses: View {
    let courses: [Projection.Course]
    let currentId: String?

    var body: some View {
        if #available(iOSApplicationExtension 16.0, *) {
            ViewThatFits(in: .vertical) {
                columns(visibleCount: 4)
                columns(visibleCount: 3)
                columns(visibleCount: 2)
                columns(visibleCount: 1)
            }
        } else {
            columns(visibleCount: 2)
        }
    }

    private func columns(visibleCount: Int) -> some View {
        let firstCount = visibleCount >= 3 ? 2 : 1
        return HStack(alignment: .top, spacing: 10) {
            CourseRows(courses: Array(courses.prefix(firstCount)), currentId: currentId,
                       count: firstCount, compact: true, small: true)
                .frame(maxWidth: .infinity, alignment: .topLeading)
            CourseRows(courses: Array(courses.dropFirst(firstCount)), currentId: currentId,
                       count: visibleCount - firstCount, compact: true, small: true)
                .frame(maxWidth: .infinity, alignment: .topLeading)
        }
    }
}

private struct DayHeading: View {
    let day: Projection.Day?
    let fallbackDate: Date

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(weekdayText(day, fallbackDate: fallbackDate))
                .font(.system(size: 11, weight: .semibold))
                .foregroundColor(.secondary)
            Text(shortDateText(day, fallbackDate: fallbackDate))
                .font(.system(size: 21, weight: .bold, design: .rounded))
            Text(weekText(day))
                .font(.system(size: 11, weight: .medium))
                .foregroundColor(.secondary)
                .lineLimit(2)
        }
    }
}

private struct LargeDayColumn: View {
    let title: String
    let day: Projection.Day?
    let fallbackDate: Date
    let currentId: String?
    let hasProjection: Bool
    let showsBrandMark: Bool
    var currentState: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .top, spacing: 6) {
                DayHeading(day: day, fallbackDate: fallbackDate)
                Spacer(minLength: 0)
                VStack(alignment: .trailing, spacing: 4) {
                    if showsBrandMark { WidgetBrandMark() }
                    Text(Locale.current.languageCode == "zh" ? title : (title == "今天" ? "Today" : "Tomorrow"))
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundColor(.secondary)
                }
            }
            if let day, !day.courses.isEmpty {
                FittingCourses(courses: day.courses, currentId: currentId, preferredCount: 3)
            } else {
                EmptyDay(state: emptyState, day: day, hasProjection: hasProjection, isTomorrow: title == "明天")
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text(accessibilityText))
    }

    private var accessibilityText: String {
        let zh = Locale.current.languageCode == "zh"
        let heading = zh ? "\(title)课表" : (title == "今天" ? "Today schedule" : "Tomorrow schedule")
        guard let day else {
            return "\(heading)，\(zh ? "需要更新课表" : "Schedule needs an update")"
        }
        return dayAccessibilityText(heading, day: day, emptyState: emptyState, isTomorrow: title == "明天")
    }

    private var emptyState: String {
        if let currentState { return currentState }
        switch day?.kind {
        case "holiday": return "holiday"
        case "unknown", nil: return "needsRefresh"
        default: return "noClasses"
        }
    }
}

private struct FittingCourses: View {
    let courses: [Projection.Course]
    let currentId: String?
    let preferredCount: Int

    var body: some View {
        if #available(iOSApplicationExtension 16.0, *) {
            ViewThatFits(in: .vertical) {
                if preferredCount >= 3 { CourseRows(courses: courses, currentId: currentId, count: 3) }
                CourseRows(courses: courses, currentId: currentId, count: 2)
                CourseRows(courses: courses, currentId: currentId, count: 1)
            }
        } else {
            CourseRows(courses: courses, currentId: currentId, count: 1)
        }
    }
}

private struct CourseRows: View {
    let courses: [Projection.Course]
    let currentId: String?
    let count: Int
    var compact = false
    var small = false

    var body: some View {
        let visible = Array(courses.prefix(count))
        VStack(alignment: .leading, spacing: compact ? (small ? 2 : 3) : 6) {
            ForEach(visible) { course in
                CourseRow(course: course, current: currentId == course.id,
                          compact: compact, small: small)
            }
            Remaining(count: courses.count - visible.count)
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
    }
}

private struct EmptyDay: View {
    let state: String
    let day: Projection.Day?
    let hasProjection: Bool
    var isTomorrow = false

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(state == "holiday" ? day?.adjustmentLabel ?? title(state, isTomorrow: isTomorrow) : title(state, isTomorrow: isTomorrow))
                .font(.subheadline.weight(.semibold))
            if let support = support(state, hasProjection: hasProjection) {
                Text(support).font(.caption2).foregroundColor(.secondary)
            }
        }
    }
}

private struct CourseStripe: View {
    let color: Color
    let verticalInset: CGFloat

    var body: some View {
        Capsule()
            .fill(color)
            .frame(width: 3)
            .padding(.vertical, verticalInset)
            .padding(.leading, 4)
            .widgetAccent()
    }
}

private struct CourseRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let course: Projection.Course
    let current: Bool
    var compact = false
    var small = false

    var body: some View {
        let stripe = courseStripeColor(course.colorSlot, scheme: colorScheme)
        return VStack(alignment: .leading, spacing: compact ? 0 : 2) {
            Text(course.name)
                .font(.system(size: small ? 11 : 13, weight: current ? .semibold : .medium))
                .foregroundColor(stripe)
                .lineLimit(2)
            if compact {
                let detail = [compactRoom(course.room), course.teacher, courseTime(course)]
                    .filter { !$0.isEmpty }.joined(separator: " · ")
                Text(detail).font(.system(size: 11).monospacedDigit())
                    .foregroundColor(.secondary).lineLimit(2)
            } else {
                let room = compactRoom(course.room)
                if !room.isEmpty {
                    Text(room).font(.system(size: 11)).foregroundColor(.secondary)
                        .lineLimit(2)
                }
                let time = courseTime(course)
                let detail = [course.teacher, time].filter { !$0.isEmpty }.joined(separator: " · ")
                Text(detail).font(.system(size: 11).monospacedDigit()).foregroundColor(.secondary)
                    .lineLimit(2)
            }
        }
        .fixedSize(horizontal: false, vertical: true)
        .padding(.leading, 12)
        .padding(.trailing, 6)
        .padding(.vertical, compact ? (small ? 1 : 2) : 5)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(RoundedRectangle(cornerRadius: 9).fill(stripe.opacity(colorScheme == .dark ? 0.18 : 0.07)))
        .overlay(
            CourseStripe(color: stripe, verticalInset: compact ? 4 : 5),
            alignment: .leading
        )
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
            .configurationDisplayName("近期课程")
            .description("离线显示正在进行和即将开始的课程，已结束课程自动移除。")
            .supportedFamilies([.systemSmall])
    }
}

struct TodayScheduleWidget: Widget {
    let kind = "TodayScheduleWidget"
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: ScheduleProvider()) { TodayScheduleView(entry: $0) }
            .configurationDisplayName("今日课表")
            .description("离线显示今天剩余课程与明天课表，随上下课时间更新。")
            .supportedFamilies([.systemMedium, .systemLarge])
    }
}

@main
struct ScheduleWidgetsBundle: WidgetBundle {
    var body: some Widget { NextClassWidget(); TodayScheduleWidget() }
}
