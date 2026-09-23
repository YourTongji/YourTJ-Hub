package tj.yourtj.forum_app.widget

import org.json.JSONArray
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

internal const val PROJECTION_KEY = "schedule_widget_projection"
internal const val EMPTY_STATE_KEY = "schedule_widget_empty_state"
internal const val WIDGET_TRANSPARENCY_KEY = "schedule_widget_transparency_percent"
internal const val DEFAULT_WIDGET_TRANSPARENCY = 9

internal data class ScheduleCourse(
    val id: String,
    val name: String,
    val teacher: String,
    val campus: String,
    val room: String,
    val startAt: Date,
    val endAt: Date,
    val colorSlot: Int,
)

internal data class ScheduleDay(
    val date: String,
    val week: Int?,
    val source: String,
    val kind: String,
    val adjustmentLabel: String?,
    val courses: List<ScheduleCourse>,
)

internal data class NextClassState(
    val status: String,
    val course: ScheduleCourse? = null,
    val day: ScheduleDay? = null,
)

internal data class ScheduleProjection(
    val generatedAt: Date,
    val days: List<ScheduleDay>,
) {
    val schoolDate: String get() = days.first().date
    val week: Int? get() = days.first().week
    val kind: String get() = days.first().kind
    val adjustmentLabel: String? get() = days.first().adjustmentLabel
    val courses: List<ScheduleCourse> get() = days.first().courses

    fun dayFor(value: Date = Date()): ScheduleDay? = days.firstOrNull { it.date == schoolDate(value) }

    fun status(now: Date = Date()): Pair<String, ScheduleCourse?> {
        if (now.time - generatedAt.time > 7L * 24 * 60 * 60 * 1000) return "stale" to null
        val day = dayFor(now) ?: return "needsRefresh" to null
        if (day.kind == "unknown") return "needsRefresh" to null
        if (day.kind == "holiday") return "holiday" to null
        if (day.courses.isEmpty()) return "noClasses" to null
        day.courses.forEachIndexed { index, course ->
            if (!now.before(course.startAt) && now.before(course.endAt)) return "inClass" to course
            if (now.before(course.startAt)) return (if (index == 0) "upcoming" else "break") to course
        }
        return "finished" to null
    }

    fun nextClass(now: Date = Date()): NextClassState {
        if (now.time - generatedAt.time > 7L * 24 * 60 * 60 * 1000) return NextClassState("stale")
        val date = schoolDate(now)
        val today = dayFor(now) ?: return NextClassState("needsRefresh")
        if (today.kind == "unknown") return NextClassState("needsRefresh")
        today.courses.firstOrNull { !now.before(it.startAt) && now.before(it.endAt) }?.let {
            return NextClassState("inClass", it, today)
        }
        val next = days.asSequence()
            .filter { it.date >= date }
            .flatMap { day -> day.courses.asSequence().map { day to it } }
            .filter { (_, course) -> now.before(course.startAt) }
            .minByOrNull { (_, course) -> course.startAt }
        if (next != null) return NextClassState("upcoming", next.second, next.first)
        if (days.size < 8 || days.any { it.date >= date && it.kind == "unknown" }) {
            return NextClassState("needsRefresh")
        }
        return NextClassState("noneUpcoming")
    }

    companion object {
        fun parse(raw: String?): ScheduleProjection? = try {
            val root = JSONObject(raw ?: return null)
            val version = root.getInt("schemaVersion")
            if (version !in 1..2 || root.getString("timezone") != "Asia/Shanghai") return null
            val identity = root.getJSONObject("identity")
            if (optionalText(identity.opt("siteKey")) == null ||
                optionalText(identity.opt("accountScope")) == null ||
                optionalText(identity.opt("bindingRevision")) == null
            ) return null
            val semester = root.optJSONObject("semester")
            val days = if (version == 1) {
                listOf(
                    parseDay(
                        root.getJSONObject("today"),
                        optionalText(root.opt("schoolDate")) ?: return null,
                        semester?.optInt("week")?.takeIf { it > 0 },
                        "server-adjusted",
                    ) ?: return null,
                )
            } else {
                root.getJSONArray("days").mapObjects(::parseDay).ifEmpty { return null }
            }
            ScheduleProjection(
                generatedAt = parseDate(root.getString("generatedAt")),
                days = days,
            )
        } catch (_: Exception) {
            null
        }
    }
}

private fun parseDay(
    value: JSONObject,
    fallbackDate: String? = null,
    fallbackWeek: Int? = null,
    fallbackSource: String = "unknown",
): ScheduleDay? {
    val date = optionalText(value.opt("date")) ?: fallbackDate ?: return null
    val kind = optionalText(value.opt("kind"))
        ?.takeIf { it in setOf("none", "normal", "holiday", "moved", "makeup", "unknown") }
        ?: "unknown"
    val courses = value.optJSONArray("courses")?.mapObjects(::parseCourse).orEmpty().sortedBy { it.startAt }
    return ScheduleDay(
        date = date,
        week = value.optInt("week").takeIf { it > 0 } ?: fallbackWeek,
        source = optionalText(value.opt("source")) ?: fallbackSource,
        kind = kind,
        adjustmentLabel = optionalText(value.opt("adjustmentLabel")),
        courses = courses,
    )
}

private fun parseCourse(value: JSONObject): ScheduleCourse? {
    val id = optionalText(value.opt("stableId")) ?: return null
    val name = optionalText(value.opt("name")) ?: return null
    val startAt = parseDate(optionalText(value.opt("startAt")) ?: return null)
    val endAt = parseDate(optionalText(value.opt("endAt")) ?: return null)
    val slot = value.optInt("colorSlot")
    if (!endAt.after(startAt) || slot !in 1..8) return null
    return ScheduleCourse(
        id = id,
        name = name,
        teacher = optionalText(value.opt("teacher")).orEmpty(),
        campus = optionalText(value.opt("campus")).orEmpty(),
        room = optionalText(value.opt("room")).orEmpty(),
        startAt = startAt,
        endAt = endAt,
        colorSlot = slot,
    )
}

internal fun ScheduleCourse.locationText(): String =
    listOf(campus, room).filter { it.isNotBlank() }.joinToString(" · ")

private fun <T : Any> JSONArray.mapObjects(transform: (JSONObject) -> T?): List<T> =
    List(length()) { index -> optJSONObject(index) }.mapNotNull { it?.let(transform) }

internal fun optionalText(value: Any?): String? {
    if (value == null || value == JSONObject.NULL) return null
    val text = (value as? String)?.trim().orEmpty()
    return text.takeIf { it.isNotEmpty() && it.lowercase(Locale.ROOT) !in setOf("null", "undefined") }
}

internal fun labelFor(status: String): String = if (Locale.getDefault().language == "zh") {
    when (status) {
        "inClass" -> "正在上课"
        "upcoming", "break" -> "下一节"
        "finished" -> "今日课程已结束"
        "noClasses" -> "今天暂无课程安排"
        "holiday" -> "今天放假"
        "noneUpcoming" -> "近期暂无课程安排"
        "stale" -> "课表可能已更新"
        "unbound" -> "绑定同济账号后显示课表"
        "authorizationRequired" -> "校园授权已失效，请打开 YourTJ 重新授权"
        "signedOut" -> "请登录并绑定同济账号"
        else -> "需要更新课表"
    }
} else {
    when (status) {
        "inClass" -> "In class"
        "upcoming", "break" -> "Up next"
        "finished" -> "Classes finished for today"
        "noClasses" -> "No classes today"
        "holiday" -> "No classes today"
        "noneUpcoming" -> "No classes in the next 8 days"
        "stale" -> "Schedule may have changed"
        "unbound" -> "Bind your Tongji account to show classes"
        "authorizationRequired" -> "Open YourTJ to restore campus access"
        "signedOut" -> "Sign in and bind your Tongji account"
        else -> "Schedule needs an update"
    }
}

internal fun supportFor(status: String, hasProjection: Boolean): String? =
    if (Locale.getDefault().language == "zh") {
        when (status) {
            "stale" -> "打开 YourTJ 刷新课表"
            "needsRefresh" -> if (hasProjection) "打开 YourTJ 刷新课表" else "打开 YourTJ 获取课表"
            "noneUpcoming" -> "打开 YourTJ 查看完整课表"
            else -> null
        }
    } else {
        when (status) {
            "stale" -> "Open YourTJ to refresh"
            "needsRefresh" -> if (hasProjection) "Open YourTJ to refresh" else "Open YourTJ to get your schedule"
            "noneUpcoming" -> "Open YourTJ to view your full schedule"
            else -> null
        }
    }

internal fun clock(value: Date): String = SimpleDateFormat("HH:mm", Locale.getDefault()).apply {
    timeZone = SHANGHAI
}.format(value)

private val SHANGHAI = TimeZone.getTimeZone("Asia/Shanghai")

private fun parseDate(value: String): Date = SimpleDateFormat(
    "yyyy-MM-dd'T'HH:mm:ssXXX",
    Locale.US,
).apply { isLenient = false }.parse(value) ?: error("Invalid date")

internal fun schoolDate(value: Date): String = SimpleDateFormat("yyyy-MM-dd", Locale.US).apply {
    timeZone = SHANGHAI
}.format(value)
