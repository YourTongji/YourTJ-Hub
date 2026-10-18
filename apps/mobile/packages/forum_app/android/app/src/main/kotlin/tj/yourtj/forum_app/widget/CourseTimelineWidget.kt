package tj.yourtj.forum_app.widget

import android.appwidget.AppWidgetManager
import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.action.ActionParameters
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidgetManager
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.action.ActionCallback
import androidx.glance.appwidget.action.actionRunCallback
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.appwidget.state.updateAppWidgetState
import androidx.glance.background
import androidx.glance.currentState
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.semantics.contentDescription
import androidx.glance.semantics.semantics
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import es.antonborri.home_widget.HomeWidgetGlanceState
import es.antonborri.home_widget.HomeWidgetGlanceStateDefinition
import es.antonborri.home_widget.HomeWidgetGlanceWidgetReceiver
import es.antonborri.home_widget.HomeWidgetPlugin
import es.antonborri.home_widget.HomeWidgetScheduler
import tj.yourtj.forum_app.R
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Date
import java.util.Locale
import java.util.TimeZone

private fun timelineDayKey(appWidgetId: Int) = "course_timeline_show_tomorrow:$appWidgetId"

class CourseTimelineWidget : ScheduleGlanceWidget() {
    override val previewSizeMode = SizeMode.Responsive(
        setOf(
            DpSize(250.dp, 180.dp),
            DpSize(320.dp, 200.dp),
        ),
    )

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val appWidgetId = GlanceAppWidgetManager(context).getAppWidgetId(id)
        provideContent {
            val preferences = currentState<HomeWidgetGlanceState>().preferences
            val projection = ScheduleProjection.parse(preferences.getString(PROJECTION_KEY, null))
            val transparency = preferences.getInt(WIDGET_TRANSPARENCY_KEY, DEFAULT_WIDGET_TRANSPARENCY)
            val showTomorrow = preferences.getBoolean(timelineDayKey(appWidgetId), false)
            val now = Date()
            val date = if (showTomorrow) tomorrowDate(now) else schoolDate(now)
            val day = projection?.days?.firstOrNull { it.date == date }
            val emptyState = emptyStatus(preferences.getString(EMPTY_STATE_KEY, null))
            val status = when {
                day != null -> emptyStatusFor(day)
                projection == null && emptyState != "ready" -> emptyState
                else -> "needsRefresh"
            }
            val dayLabel = if (Locale.getDefault().language == "zh") {
                if (showTomorrow) "明天" else "今天"
            } else if (showTomorrow) {
                "Tomorrow"
            } else {
                "Today"
            }
            val description = listOfNotNull(
                dayLabel,
                timelineDate(date),
                day?.let { dayHeaderText(it.date, it.week, it.adjustmentLabel) },
                day?.courses?.takeIf { it.isNotEmpty() }?.joinToString("，") { course ->
                    listOfNotNull(
                        course.startSection?.let(::sectionDescription),
                        clock(course.startAt),
                        course.name,
                        course.endSection?.let(::sectionDescription),
                        clock(course.endAt),
                        course.room.trim().takeIf { it.isNotBlank() },
                        teacherDisplayName(course.teacher).takeIf { it.isNotBlank() },
                    ).joinToString("，")
                },
                if (day == null || day.courses.isEmpty()) labelFor(status) else null,
            ).joinToString("，")

            Surface(context, description, transparency) {
                Column(modifier = GlanceModifier.fillMaxSize()) {
                    TimelineHeader(date, day?.week, day?.adjustmentLabel, showTomorrow)
                    Spacer(GlanceModifier.height(7.dp))
                    if (day == null || day.courses.isEmpty()) {
                        EmptyDay(status, day, projection != null)
                    } else {
                        LazyColumn(
                            modifier = GlanceModifier.defaultWeight().fillMaxWidth(),
                        ) {
                            items(
                                items = day.courses,
                                itemId = { course -> course.id.hashCode().toLong() },
                            ) { course ->
                                TimelineCourseListItem(context, course)
                            }
                        }
                    }
                }
            }
        }
    }

    override suspend fun providePreview(context: Context, widgetCategory: Int) {
        val now = Date()
        val day = previewScheduleDay(now, 0).copy(
            courses = listOf(
                ScheduleCourse("timeline-preview-1", "计算机视觉", "张老师", "", "瑞安楼 A216", atShanghai(now, 8, 0), atShanghai(now, 9, 35), 1, 1, 4),
                ScheduleCourse("timeline-preview-2", "海洋遥感", "W. Carter", "", "北303", atShanghai(now, 10, 0), atShanghai(now, 11, 35), 2, 5, 6),
                ScheduleCourse("timeline-preview-3", "海洋流体力学", "李老师", "", "瑞安楼", atShanghai(now, 13, 30), atShanghai(now, 15, 5), 6, 7, 8),
            ),
        )
        provideContent {
            Surface(context, "课程时间线预览", DEFAULT_WIDGET_TRANSPARENCY) {
                Column(modifier = GlanceModifier.fillMaxSize()) {
                    TimelineHeader(day.date, day.week, day.adjustmentLabel, false)
                    Spacer(GlanceModifier.height(7.dp))
                    LazyColumn(modifier = GlanceModifier.defaultWeight().fillMaxWidth()) {
                        items(day.courses, itemId = { course -> course.id.hashCode().toLong() }) { course ->
                            TimelineCourseListItem(context, course)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun TimelineCourseListItem(context: Context, course: ScheduleCourse) {
    Column {
        TimelineCourseCard(context, course)
        Spacer(GlanceModifier.height(10.dp))
    }
}

@Composable
private fun TimelineHeader(date: String, week: Int?, adjustment: String?, showTomorrow: Boolean) {
    val zh = Locale.getDefault().language == "zh"
    Row(modifier = GlanceModifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Column(modifier = GlanceModifier.defaultWeight()) {
            Text(
                timelineDate(date),
                maxLines = 1,
                style = TextStyle(color = foreground, fontSize = 16.sp, fontWeight = FontWeight.Bold),
            )
            Spacer(GlanceModifier.height(2.dp))
            Text(
                timelineSubtitle(date, week, adjustment, showTomorrow),
                maxLines = 1,
                style = TextStyle(color = muted, fontSize = 11.sp, fontWeight = FontWeight.Medium),
            )
        }
        BrandMark()
        Spacer(GlanceModifier.width(7.dp))
        Box(
            modifier = GlanceModifier
                .size(36.dp)
                .background(divider)
                .cornerRadius(18.dp)
                .clickable(actionRunCallback<ToggleCourseTimelineDayAction>())
                .semantics {
                    contentDescription = if (showTomorrow) {
                        if (zh) "切换到今天" else "Switch to today"
                    } else {
                        if (zh) "切换到明天" else "Switch to tomorrow"
                    }
                },
            contentAlignment = Alignment.Center,
        ) {
            Image(
                provider = ImageProvider(
                    if (showTomorrow) R.drawable.course_timeline_arrow_back
                    else R.drawable.course_timeline_arrow_next,
                ),
                contentDescription = null,
                modifier = GlanceModifier.size(20.dp),
            )
        }
    }
}

@Composable
private fun TimelineCourseCard(context: Context, course: ScheduleCourse) {
    val room = course.room.trim()
    val teacher = teacherDisplayName(course.teacher)
    Row(
        modifier = GlanceModifier
            .fillMaxWidth()
            .background(courseTint(context, course.colorSlot))
            .cornerRadius(16.dp)
            .padding(horizontal = 8.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = GlanceModifier.width(3.dp).height(24.dp)
                .background(courseColor(course.colorSlot)).cornerRadius(2.dp),
        ) {}
        Spacer(GlanceModifier.width(7.dp))
        Column(modifier = GlanceModifier.defaultWeight()) {
            Row(modifier = GlanceModifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    course.startSection?.let(::sectionLabel) ?: "—",
                    modifier = GlanceModifier.width(29.dp),
                    style = TextStyle(color = accent, fontSize = 11.sp, fontWeight = FontWeight.Bold),
                )
                Text(
                    clock(course.startAt),
                    modifier = GlanceModifier.width(43.dp),
                    style = TextStyle(color = muted, fontSize = 11.sp, fontWeight = FontWeight.Medium),
                )
                Text(
                    course.name,
                    modifier = GlanceModifier.defaultWeight(),
                    maxLines = 2,
                    style = TextStyle(color = foreground, fontSize = 14.sp, fontWeight = FontWeight.Bold),
                )
            }
            Spacer(GlanceModifier.height(2.dp))
            Row(modifier = GlanceModifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    course.endSection?.let(::sectionLabel) ?: "—",
                    modifier = GlanceModifier.width(29.dp),
                    style = TextStyle(color = muted, fontSize = 10.sp, fontWeight = FontWeight.Medium),
                )
                Text(
                    clock(course.endAt),
                    modifier = GlanceModifier.width(43.dp),
                    style = TextStyle(color = muted, fontSize = 10.sp),
                )
                Row(
                    modifier = GlanceModifier.defaultWeight(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (room.isNotBlank()) {
                        Image(
                            provider = ImageProvider(R.drawable.course_timeline_location),
                            contentDescription = null,
                            modifier = GlanceModifier.size(12.dp),
                        )
                        Text(
                            room,
                            modifier = GlanceModifier.padding(start = 3.dp),
                            maxLines = 1,
                            style = TextStyle(color = muted, fontSize = 10.sp),
                        )
                    }
                    if (teacher.isNotBlank()) {
                        if (room.isNotBlank()) Spacer(GlanceModifier.width(4.dp))
                        Image(
                            provider = ImageProvider(R.drawable.course_timeline_teacher),
                            contentDescription = null,
                            modifier = GlanceModifier.size(12.dp),
                        )
                        Text(
                            teacher,
                            modifier = GlanceModifier.padding(start = 3.dp),
                            maxLines = 1,
                            style = TextStyle(color = muted, fontSize = 10.sp),
                        )
                    }
                }
            }
        }
    }
}

private fun timelineDate(date: String): String {
    val parsed = SimpleDateFormat("yyyy-MM-dd", Locale.US).apply {
        isLenient = false
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.parse(date) ?: return date
    return SimpleDateFormat("yyyy/M/d", Locale.getDefault()).apply {
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.format(parsed)
}

private fun sectionLabel(section: Int): String = if (Locale.getDefault().language == "zh") {
    "${section}节"
} else {
    "L$section"
}

private fun sectionDescription(section: Int): String = if (Locale.getDefault().language == "zh") {
    "第${section}节"
} else {
    "Section $section"
}

private fun timelineSubtitle(date: String, week: Int?, adjustment: String?, showTomorrow: Boolean): String {
    val zh = Locale.getDefault().language == "zh"
    val locale = if (zh) Locale.SIMPLIFIED_CHINESE else Locale.getDefault()
    val parsed = SimpleDateFormat("yyyy-MM-dd", Locale.US).apply {
        isLenient = false
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.parse(date)
    val relative = if (zh) {
        if (showTomorrow) "明天" else "今天"
    } else if (showTomorrow) {
        "Tomorrow"
    } else {
        "Today"
    }
    val weekLabel = week?.let { if (zh) "第${it}周" else "Week $it" }
    val weekday = parsed?.let {
        SimpleDateFormat("EEEE", locale).apply {
            timeZone = TimeZone.getTimeZone("Asia/Shanghai")
        }.format(it)
    }
    return listOfNotNull(relative, weekLabel, weekday, adjustment).joinToString(" · ")
}

private fun atShanghai(base: Date, hour: Int, minute: Int): Date = Calendar
    .getInstance(TimeZone.getTimeZone("Asia/Shanghai"))
    .apply {
        time = base
        set(Calendar.HOUR_OF_DAY, hour)
        set(Calendar.MINUTE, minute)
        set(Calendar.SECOND, 0)
        set(Calendar.MILLISECOND, 0)
    }
    .time

class ToggleCourseTimelineDayAction : ActionCallback {
    override suspend fun onAction(context: Context, glanceId: GlanceId, parameters: ActionParameters) {
        val appWidgetId = GlanceAppWidgetManager(context).getAppWidgetId(glanceId)
        val preferences = HomeWidgetPlugin.getData(context)
        val key = timelineDayKey(appWidgetId)
        preferences.edit().putBoolean(key, !preferences.getBoolean(key, false)).apply()
        val widget = CourseTimelineWidget()
        updateAppWidgetState<HomeWidgetGlanceState>(
            context,
            widget.stateDefinition as HomeWidgetGlanceStateDefinition,
            glanceId,
        ) { it }
        widget.update(context, glanceId)
    }
}

class CourseTimelineWidgetReceiver : HomeWidgetGlanceWidgetReceiver<CourseTimelineWidget>() {
    override val glanceAppWidget = CourseTimelineWidget()

    override fun onEnabled(context: Context) {
        super.onEnabled(context)
        scheduleNextDay(context)
    }

    override fun onUpdate(context: Context, appWidgetManager: AppWidgetManager, appWidgetIds: IntArray) {
        super.onUpdate(context, appWidgetManager, appWidgetIds)
        scheduleNextDay(context)
    }

    override fun onDeleted(context: Context, appWidgetIds: IntArray) {
        HomeWidgetPlugin.getData(context).edit().apply {
            appWidgetIds.forEach { remove(timelineDayKey(it)) }
        }.apply()
        super.onDeleted(context, appWidgetIds)
    }

    override fun onDisabled(context: Context) {
        super.onDisabled(context)
        HomeWidgetScheduler.cancel(context, javaClass.name)
    }

    private fun scheduleNextDay(context: Context) {
        HomeWidgetScheduler.schedule(context, javaClass.name, listOf(nextShanghaiMidnight(Date()).time))
    }
}
