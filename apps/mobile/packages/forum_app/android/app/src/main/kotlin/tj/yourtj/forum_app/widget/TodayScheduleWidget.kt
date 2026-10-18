package tj.yourtj.forum_app.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.LocalSize
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.background
import androidx.glance.currentState
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxHeight
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.width
import androidx.glance.semantics.contentDescription
import androidx.glance.semantics.semantics
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import es.antonborri.home_widget.HomeWidgetGlanceState
import es.antonborri.home_widget.HomeWidgetGlanceWidgetReceiver
import java.util.Date
import java.util.Locale

class TodayScheduleWidget : ScheduleGlanceWidget() {
    override val previewSizeMode = SizeMode.Responsive(
        setOf(
            DpSize(250.dp, 180.dp),
            DpSize(320.dp, 200.dp),
        ),
    )

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        provideContent {
            val preferences = currentState<HomeWidgetGlanceState>().preferences
            val projection = ScheduleProjection.parse(preferences.getString(PROJECTION_KEY, null))
            val transparency = preferences.getInt(WIDGET_TRANSPARENCY_KEY, DEFAULT_WIDGET_TRANSPARENCY)
            val status = projection?.status()
                ?: (emptyStatus(preferences.getString(EMPTY_STATE_KEY, null)) to null)
            val large = LocalSize.current.height >= 150.dp
            val today = projection?.dayFor()
            val tomorrow = projection?.days?.firstOrNull { it.date == tomorrowDate() }
            val description = scheduleDescription(status.first, today, if (large) tomorrow else null, projection != null)
            Surface(context, description, transparency) {
                if (large) {
                    LargeSchedule(today, tomorrow, status.second?.id, projection != null, projection?.generatedAt)
                } else {
                    MediumDayColumn(today, status, projection != null)
                }
            }
        }
    }

    override suspend fun providePreview(context: Context, widgetCategory: Int) {
        val now = Date()
        val today = previewScheduleDay(now, 0)
        val tomorrow = previewScheduleDay(now, 1)
        val currentId = today.courses.firstOrNull()?.id
        provideContent {
            val large = LocalSize.current.height >= 180.dp
            Surface(context, "今日与明日课表预览", DEFAULT_WIDGET_TRANSPARENCY) {
                if (large) {
                    LargeSchedule(today, tomorrow, currentId, true, null)
                } else {
                    MediumDayColumn(today, "upcoming" to today.courses.firstOrNull(), true)
                }
            }
        }
    }
}

@Composable
private fun MediumDayColumn(
    day: ScheduleDay?,
    status: Pair<String, ScheduleCourse?>,
    hasProjection: Boolean,
) {
    val visible = day?.courses?.take(3).orEmpty()
    Column {
        DayHeader(day?.date ?: "今日课表", day?.week, day?.adjustmentLabel)
        Spacer(GlanceModifier.height(6.dp))
        if (visible.isEmpty()) {
            EmptyDay(status.first, day, hasProjection)
        } else {
            visible.forEach { CourseRow(it, status.second?.id == it.id, false) }
            Remaining(day!!.courses.size - visible.size)
        }
    }
}

@Composable
private fun LargeSchedule(
    today: ScheduleDay?,
    tomorrow: ScheduleDay?,
    currentId: String?,
    hasProjection: Boolean,
    generatedAt: Date?,
) {
    Column(modifier = GlanceModifier.fillMaxSize()) {
        Row(modifier = GlanceModifier.fillMaxWidth().defaultWeight()) {
            LargeDayColumn(
                "今天",
                today,
                currentId,
                hasProjection,
                GlanceModifier.defaultWeight().fillMaxHeight().padding(end = 6.dp),
            )
            Divider()
            LargeDayColumn(
                "明天",
                tomorrow,
                null,
                hasProjection,
                GlanceModifier.defaultWeight().fillMaxHeight().padding(start = 6.dp),
            )
        }
        generatedAt?.let { Text(updatedText(it), style = TextStyle(color = muted, fontSize = 10.sp)) }
    }
}

@Composable
private fun LargeDayColumn(
    title: String,
    day: ScheduleDay?,
    currentId: String?,
    hasProjection: Boolean,
    modifier: GlanceModifier,
) {
    val accessibilityTitle = when {
        Locale.getDefault().language == "zh" -> "${title}课表"
        title == "今天" -> "Today schedule"
        else -> "Tomorrow schedule"
    }
    val description = day?.let { dayDescription(accessibilityTitle, it) }
        ?: listOfNotNull(
            accessibilityTitle,
            labelFor("needsRefresh"),
            supportFor("needsRefresh", hasProjection),
        ).joinToString("，")
    Column(modifier = modifier.semantics { contentDescription = description }) {
        Row(modifier = GlanceModifier.fillMaxWidth()) {
            Text(
                title,
                modifier = GlanceModifier.defaultWeight(),
                style = TextStyle(color = foreground, fontSize = 15.sp, fontWeight = FontWeight.Bold),
            )
            if (title == "明天") BrandMark()
        }
        DayHeader(day?.date.orEmpty(), day?.week, day?.adjustmentLabel, showBrand = false)
        Spacer(GlanceModifier.height(4.dp))
        if (day == null || day.courses.isEmpty()) {
            EmptyDay(emptyStatusFor(day), day, hasProjection)
        } else {
            LazyColumn(
                modifier = GlanceModifier.defaultWeight().fillMaxWidth(),
            ) {
                items(
                    items = day.courses,
                    itemId = { course -> course.id.hashCode().toLong() },
                ) { course ->
                    CourseRow(course, currentId == course.id, true)
                }
            }
        }
    }
}

@Composable
private fun Divider() {
    Box(GlanceModifier.width(1.dp).fillMaxHeight().background(divider)) {}
}

private fun scheduleDescription(
    status: String,
    today: ScheduleDay?,
    tomorrow: ScheduleDay?,
    hasProjection: Boolean,
): String = listOfNotNull(
    labelFor(status),
    today?.let { dayDescription(if (Locale.getDefault().language == "zh") "今天" else "Today", it) },
    tomorrow?.let { dayDescription(if (Locale.getDefault().language == "zh") "明天" else "Tomorrow", it) },
    supportFor(status, hasProjection),
).joinToString("，")

private fun dayDescription(title: String, day: ScheduleDay): String {
    val courses = day.courses.joinToString("，") { course ->
        listOf(
            course.name,
            "${clock(course.startAt)}–${clock(course.endAt)}",
            course.room.trim(),
            teacherNames(course.teacher).joinToString("、"),
        )
            .filter { it.isNotBlank() }
            .joinToString("，")
    }
    val empty = if (courses.isEmpty()) labelFor(emptyStatusFor(day)) else null
    return listOfNotNull(
        title,
        dayHeaderText(day.date, day.week, day.adjustmentLabel),
        courses.ifEmpty { null },
        empty,
    ).joinToString("，")
}

@Composable
private fun DayHeader(date: String, week: Int?, adjustment: String?, showBrand: Boolean = true) {
    Row(modifier = GlanceModifier.fillMaxWidth()) {
        Text(
            dayHeaderText(date, week, adjustment),
            maxLines = 2,
            modifier = if (showBrand) GlanceModifier.defaultWeight() else GlanceModifier.fillMaxWidth(),
            style = TextStyle(color = muted, fontSize = 10.sp),
        )
        if (showBrand) BrandMark()
    }
}

@Composable
private fun CourseRow(course: ScheduleCourse, current: Boolean, large: Boolean) {
    Row(modifier = GlanceModifier.fillMaxWidth().padding(end = 8.dp).padding(vertical = 3.dp)) {
        Box(
            GlanceModifier
                .width(4.dp)
                .height(if (large) 54.dp else 48.dp)
                .background(courseColor(course.colorSlot))
                .cornerRadius(2.dp),
        ) {}
        Spacer(GlanceModifier.width(8.dp))
        Column(modifier = GlanceModifier.defaultWeight()) {
            Text(
                course.name,
                maxLines = 2,
                style = TextStyle(
                    color = if (current) accent else foreground,
                    fontSize = if (large) 15.sp else 14.sp,
                    fontWeight = FontWeight.Bold,
                ),
            )
            val location = course.room.trim()
            val teacher = teacherDisplayName(course.teacher)
            if (teacher.isNotBlank() || location.isNotBlank()) {
                Text(
                    listOf(location, teacher).filter { it.isNotBlank() }.joinToString(" · "),
                    maxLines = 1,
                    style = TextStyle(color = muted, fontSize = 11.sp),
                )
            }
            Text(
                "${clock(course.startAt)}–${clock(course.endAt)}",
                maxLines = 1,
                style = TextStyle(
                    color = foreground,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Medium,
                ),
            )
        }
    }
}

@Composable
private fun Remaining(count: Int) {
    if (count <= 0) return
    Text(
        if (Locale.getDefault().language == "zh") "还有 $count 门课程" else "$count more classes",
        style = TextStyle(color = muted, fontSize = 11.sp),
    )
}

class TodayScheduleWidgetReceiver : HomeWidgetGlanceWidgetReceiver<TodayScheduleWidget>() {
    override val glanceAppWidget = TodayScheduleWidget()
}
