package tj.yourtj.forum_app.widget

import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceModifier
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.layout.Column
import androidx.glance.layout.Spacer
import androidx.glance.layout.height
import androidx.glance.layout.size
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import tj.yourtj.forum_app.R
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Date
import java.util.Locale
import java.util.TimeZone

internal fun dayHeaderText(date: String, week: Int?, adjustment: String?): String {
    val zh = Locale.getDefault().language == "zh"
    val locale = if (zh) Locale.SIMPLIFIED_CHINESE else Locale.getDefault()
    val parsed = SimpleDateFormat("yyyy-MM-dd", Locale.US).apply {
        isLenient = false
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.parse(date)
    val shortDate = parsed?.let {
        SimpleDateFormat(if (zh) "M月d日" else "MMM d", locale).apply {
            timeZone = TimeZone.getTimeZone("Asia/Shanghai")
        }.format(it)
    } ?: date.takeIf { it.isNotBlank() }
    val weekday = parsed?.let {
        SimpleDateFormat("EEEE", locale).apply {
            timeZone = TimeZone.getTimeZone("Asia/Shanghai")
        }.format(it)
    }
    return listOfNotNull(
        shortDate,
        week?.let { if (zh) "第${it}周" else "Week $it" },
        weekday,
        adjustment,
    ).joinToString(" · ")
}

internal fun teacherNames(value: String): List<String> = value
    .split(Regex("[,，、;；]"))
    .map { it.replace(Regex("\\s*[（(][^（）()]*[）)]\\s*$"), "").trim() }
    .filter { it.isNotBlank() }

internal fun teacherDisplayName(value: String): String {
    val names = teacherNames(value)
    val visible = names.take(2).map { name ->
        val words = name.split(Regex("\\s+")).filter { it.isNotBlank() }
        if (words.size < 2 || name.any { it in '\u4e00'..'\u9fff' }) {
            name
        } else {
            val initials = words.dropLast(1).joinToString("") { word ->
                if (word.endsWith(".")) word else "${word.first().uppercaseChar()}."
            }
            "$initials ${words.last()}"
        }
    }
    return visible.joinToString("、") + if (names.size > visible.size) " 等" else ""
}

internal fun updatedText(value: Date): String =
    if (Locale.getDefault().language == "zh") "更新于 ${clock(value)}" else "Updated at ${clock(value)}"

internal fun emptyStatusFor(day: ScheduleDay?): String = when (day?.kind) {
    "holiday" -> "holiday"
    "unknown", null -> "needsRefresh"
    else -> "noClasses"
}

@Composable
internal fun BrandMark() {
    Image(
        provider = ImageProvider(R.drawable.schedule_widget_brand),
        contentDescription = null,
        modifier = GlanceModifier.size(16.dp),
    )
}

@Composable
internal fun EmptyDay(status: String, day: ScheduleDay?, hasProjection: Boolean) {
    Text(
        if (status == "holiday") day?.adjustmentLabel ?: labelFor(status) else labelFor(status),
        style = TextStyle(color = foreground, fontSize = 14.sp, fontWeight = FontWeight.Bold),
    )
    supportFor(status, hasProjection)?.let {
        Spacer(GlanceModifier.height(4.dp))
        Text(it, style = TextStyle(color = muted, fontSize = 11.sp))
    }
}

internal fun tomorrowDate(now: Date = Date()): String {
    val calendar = Calendar.getInstance(TimeZone.getTimeZone("Asia/Shanghai"))
    calendar.time = now
    calendar.add(Calendar.DAY_OF_MONTH, 1)
    return schoolDate(calendar.time)
}

internal fun previewScheduleDay(now: Date, offset: Int): ScheduleDay {
    val day = Calendar.getInstance(TimeZone.getTimeZone("Asia/Shanghai"))
    day.time = now
    day.add(Calendar.DAY_OF_MONTH, offset)
    val date = schoolDate(day.time)
    fun timeAt(hour: Int, minute: Int): Date {
        val value = Calendar.getInstance(TimeZone.getTimeZone("Asia/Shanghai"))
        value.time = day.time
        value.set(Calendar.HOUR_OF_DAY, hour)
        value.set(Calendar.MINUTE, minute)
        value.set(Calendar.SECOND, 0)
        value.set(Calendar.MILLISECOND, 0)
        return value.time
    }
    return ScheduleDay(
        date = date,
        week = 4,
        source = "widget-picker-preview",
        kind = "normal",
        adjustmentLabel = null,
        courses = listOf(
            ScheduleCourse("widget-preview-$offset-1", "高等数学", "", "瑞安楼", "李老师", timeAt(8, 0), timeAt(9, 35), 5),
            ScheduleCourse("widget-preview-$offset-2", "大学物理", "", "嘉定楼", "王老师", timeAt(10, 0), timeAt(11, 35), 2),
        ),
    )
}

internal fun emptyStatus(value: String?): String = if (value == "needsData") "needsRefresh" else value ?: "needsRefresh"
