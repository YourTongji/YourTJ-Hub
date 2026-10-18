package tj.yourtj.forum_app.widget

import android.appwidget.AppWidgetManager
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
import androidx.glance.appwidget.provideContent
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
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import es.antonborri.home_widget.HomeWidgetGlanceState
import es.antonborri.home_widget.HomeWidgetGlanceWidgetReceiver
import es.antonborri.home_widget.HomeWidgetPlugin
import es.antonborri.home_widget.HomeWidgetScheduler
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

class NextClassWidget : ScheduleGlanceWidget() {
    override val previewSizeMode = SizeMode.Responsive(
        setOf(
            DpSize(110.dp, 50.dp),
            DpSize(250.dp, 110.dp),
            DpSize(320.dp, 180.dp),
        ),
    )

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        provideContent {
            val preferences = currentState<HomeWidgetGlanceState>().preferences
            val projection = ScheduleProjection.parse(preferences.getString(PROJECTION_KEY, null))
            val transparency = preferences.getInt(WIDGET_TRANSPARENCY_KEY, DEFAULT_WIDGET_TRANSPARENCY)
            val now = Date()
            val state = projection?.nextClass(now)
                ?: NextClassState(emptyStatus(preferences.getString(EMPTY_STATE_KEY, null)))
            NextClassContent(context, state, now, projection?.generatedAt, projection != null, transparency)
        }
    }

    override suspend fun providePreview(context: Context, widgetCategory: Int) {
        val now = Date()
        val day = previewScheduleDay(now, 0)
        val course = ScheduleCourse(
            id = "widget-preview-next",
            name = "课程名称",
            teacher = "",
            campus = "",
            room = "",
            startAt = Date(now.time + 60 * 60 * 1000L),
            endAt = Date(now.time + 105 * 60 * 1000L),
            colorSlot = 5,
        )
        provideContent {
            NextClassContent(
                context,
                NextClassState("upcoming", course, day),
                now,
                null,
                true,
                DEFAULT_WIDGET_TRANSPARENCY,
            )
        }
    }

    @Composable
    private fun NextClassContent(
        context: Context,
        state: NextClassState,
        now: Date,
        generatedAt: Date?,
        hasProjection: Boolean,
        transparency: Int,
    ) {
        val course = state.course
        val title = labelFor(state.status)
        val support = supportFor(state.status, hasProjection)
        val date = state.day?.let { dateContext(it.date, now) }
        val time = course?.let { courseTime(state.status, it) }
        val distance = course?.let { distanceText(state.status, it, now) }
        val location = course?.locationText()?.takeIf { it.isNotBlank() }
            ?.let { detailText("地点", "Location", it) }
        val teacher = course?.teacher?.let(::teacherDisplayName)
            ?.takeIf { it.isNotBlank() }
            ?.let { detailText("教师", "Teacher", it) }
        val updated = generatedAt?.let(::updatedText)
        val description = listOfNotNull(
            title,
            date,
            course?.name,
            time,
            location,
            teacher,
            distance,
            support,
            updated,
        ).joinToString("，")
        val widgetSize = LocalSize.current
        Surface(
            context,
            description,
            transparency,
            horizontalPadding = if (nextClassSize(widgetSize) == NextClassSize.WideCompact) 16.dp else null,
        ) {
            when (nextClassSize(widgetSize)) {
                NextClassSize.WideCompact -> WideCompactNextClass(
                    widgetSize,
                    title,
                    support,
                    date,
                    course,
                    time,
                    distance,
                    location,
                    teacher,
                    updated,
                )
                NextClassSize.Compact, NextClassSize.Expanded -> Column(
                    modifier = GlanceModifier.fillMaxSize(),
                ) {
                    val expanded = nextClassSize(widgetSize) == NextClassSize.Expanded
                    if (course == null) {
                        Row(modifier = GlanceModifier.fillMaxWidth()) {
                            Text(
                                title,
                                maxLines = 2,
                                modifier = GlanceModifier.defaultWeight(),
                                style = TextStyle(
                                    color = foreground,
                                    fontSize = 17.sp,
                                    fontWeight = FontWeight.Bold,
                                ),
                            )
                            BrandMark()
                        }
                        support?.let {
                            Spacer(GlanceModifier.height(4.dp))
                            Text(it, style = TextStyle(color = muted, fontSize = 11.sp))
                        }
                        if (expanded) {
                            updated?.let {
                                Spacer(GlanceModifier.height(6.dp))
                                Text(it, style = TextStyle(color = muted, fontSize = 10.sp))
                            }
                        }
                        return@Column
                    }
                    Row(modifier = GlanceModifier.fillMaxWidth()) {
                        Text(
                            title,
                            modifier = GlanceModifier.defaultWeight(),
                            style = TextStyle(color = accent, fontSize = 12.sp, fontWeight = FontWeight.Bold),
                        )
                        BrandMark()
                    }
                    date?.let {
                        Spacer(GlanceModifier.height(4.dp))
                        Text(it, style = TextStyle(color = muted, fontSize = 11.sp))
                    }
                    Spacer(GlanceModifier.height(5.dp))
                    Text(
                        course.name,
                        maxLines = 2,
                        style = TextStyle(color = foreground, fontSize = 17.sp, fontWeight = FontWeight.Bold),
                    )
                    Spacer(GlanceModifier.height(4.dp))
                    Text(time.orEmpty(), style = TextStyle(color = muted, fontSize = 11.sp))
                    Text(distance.orEmpty(), style = TextStyle(color = accent, fontSize = 11.sp))
                    if (expanded) {
                        location?.let { Text(it, style = TextStyle(color = muted, fontSize = 11.sp)) }
                        teacher?.let { Text(it, style = TextStyle(color = muted, fontSize = 11.sp)) }
                        updated?.let {
                            Spacer(GlanceModifier.height(6.dp))
                            Text(it, style = TextStyle(color = muted, fontSize = 10.sp))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun WideCompactNextClass(
    size: DpSize,
    title: String,
    support: String?,
    date: String?,
    course: ScheduleCourse?,
    time: String?,
    distance: String?,
    location: String?,
    teacher: String?,
    updated: String?,
) {
    Column(
        modifier = GlanceModifier.fillMaxSize(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (course == null) {
            Row(modifier = GlanceModifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    title,
                    maxLines = 1,
                    modifier = GlanceModifier.defaultWeight().padding(end = 8.dp),
                    style = TextStyle(color = foreground, fontSize = 16.sp, fontWeight = FontWeight.Bold),
                )
                BrandMark()
            }
            support?.let {
                Spacer(GlanceModifier.height(3.dp))
                Text(it, maxLines = 1, style = TextStyle(color = muted, fontSize = 12.sp))
            }
        } else {
            Row(modifier = GlanceModifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    title,
                    maxLines = 1,
                    style = TextStyle(color = accent, fontSize = 12.sp, fontWeight = FontWeight.Bold),
                )
                date?.let {
                    Text(
                        it,
                        maxLines = 1,
                        modifier = GlanceModifier.defaultWeight().padding(start = 6.dp, end = 6.dp),
                        style = TextStyle(color = muted, fontSize = 11.sp),
                    )
                } ?: Spacer(GlanceModifier.defaultWeight())
                BrandMark()
            }
            Spacer(GlanceModifier.height(4.dp))
            Row(
                modifier = GlanceModifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    GlanceModifier
                        .width(3.dp)
                        .height(30.dp)
                        .background(courseColor(course.colorSlot))
                        .cornerRadius(2.dp),
                ) {}
                Spacer(GlanceModifier.width(8.dp))
                Column(modifier = GlanceModifier.defaultWeight().padding(end = 8.dp)) {
                    Text(
                        course.name,
                        maxLines = 2,
                        style = TextStyle(color = foreground, fontSize = 16.sp, fontWeight = FontWeight.Bold),
                    )
                    if (size.height >= 82.dp) {
                        location?.let { Text(it, maxLines = 1, style = TextStyle(color = muted, fontSize = 11.sp)) }
                    }
                    if (size.height >= 110.dp) {
                        teacher?.let { Text(it, maxLines = 1, style = TextStyle(color = muted, fontSize = 11.sp)) }
                    }
                }
                Column(horizontalAlignment = Alignment.End) {
                    Text(
                        time.orEmpty(),
                        style = TextStyle(color = foreground, fontSize = 14.sp, fontWeight = FontWeight.Bold),
                    )
                    Spacer(GlanceModifier.height(2.dp))
                    Text(distance.orEmpty(), style = TextStyle(color = accent, fontSize = 12.sp))
                    if (size.height >= 130.dp) {
                        updated?.let { Text(it, maxLines = 1, style = TextStyle(color = muted, fontSize = 10.sp)) }
                    }
                }
            }
        }
    }
}

private enum class NextClassSize { Compact, WideCompact, Expanded }

private fun nextClassSize(size: DpSize): NextClassSize = when {
    size.height >= 160.dp -> NextClassSize.Expanded
    size.width >= 250.dp -> NextClassSize.WideCompact
    else -> NextClassSize.Compact
}

private fun courseTime(status: String, course: ScheduleCourse): String =
    if (status == "inClass") {
        if (Locale.getDefault().language == "zh") "至 ${clock(course.endAt)}" else "Until ${clock(course.endAt)}"
    } else {
        "${clock(course.startAt)}–${clock(course.endAt)}"
    }

private fun distanceText(status: String, course: ScheduleCourse, now: Date): String {
    val target = if (status == "inClass") course.endAt else course.startAt
    val remainingMinutes = maxOf(1L, (target.time - now.time + 59_999L) / 60_000L)
    val minutes = if (remainingMinutes >= 24L * 60) {
        remainingMinutes
    } else {
        (((remainingMinutes + 4) / 5) * 5).coerceAtMost(24L * 60 - 5)
    }
    val zh = Locale.getDefault().language == "zh"
    val duration = when {
        minutes >= 24 * 60 -> {
            val days = minutes / (24 * 60)
            val hours = minutes % (24 * 60) / 60
            if (zh) {
                if (hours == 0L) "$days 天" else "$days 天 $hours 小时"
            } else {
                if (hours == 0L) "${days}d" else "${days}d ${hours}h"
            }
        }
        minutes >= 60 -> {
            val hours = minutes / 60
            val rest = minutes % 60
            if (zh) {
                if (rest == 0L) "$hours 小时" else "$hours 小时 $rest 分钟"
            } else {
                if (rest == 0L) "${hours}h" else "${hours}h ${rest}m"
            }
        }
        zh -> "$minutes 分钟"
        else -> "${minutes}m"
    }
    if (zh) {
        return "距${if (status == "inClass") "下课" else "上课"} $duration"
    }
    return if (status == "inClass") "Ends in $duration" else "Starts in $duration"
}

private fun dateContext(date: String, now: Date): String {
    val zh = Locale.getDefault().language == "zh"
    val relative = when (date) {
        schoolDate(now) -> if (zh) "今天" else "Today"
        tomorrowDate(now) -> if (zh) "明天" else "Tomorrow"
        else -> null
    }
    val parsed = SimpleDateFormat("yyyy-MM-dd", Locale.US).apply {
        isLenient = false
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.parse(date) ?: return listOfNotNull(relative, date).joinToString(" · ")
    val locale = if (zh) Locale.SIMPLIFIED_CHINESE else Locale.getDefault()
    val weekday = SimpleDateFormat("EEE", locale).apply {
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.format(parsed)
    val shortDate = SimpleDateFormat(if (zh) "M月d日" else "MMM d", locale).apply {
        timeZone = TimeZone.getTimeZone("Asia/Shanghai")
    }.format(parsed)
    return "${listOfNotNull(relative, weekday).joinToString(" ")} · $shortDate"
}

private fun detailText(zhLabel: String, enLabel: String, value: String): String =
    "${if (Locale.getDefault().language == "zh") zhLabel else enLabel}：$value"

class NextClassWidgetReceiver : HomeWidgetGlanceWidgetReceiver<NextClassWidget>() {
    override val glanceAppWidget = NextClassWidget()

    override fun onEnabled(context: Context) {
        super.onEnabled(context)
        scheduleNextUpdate(context)
    }

    override fun onUpdate(
        context: Context,
        appWidgetManager: AppWidgetManager,
        appWidgetIds: IntArray,
    ) {
        super.onUpdate(context, appWidgetManager, appWidgetIds)
        scheduleNextUpdate(context)
    }

    private fun scheduleNextUpdate(context: Context) {
        val projection = ScheduleProjection.parse(
            HomeWidgetPlugin.getData(context).getString(PROJECTION_KEY, null),
        ) ?: return
        val nextUpdate = projection.nextUpdateAt(Date()) ?: return
        HomeWidgetScheduler.schedule(context, javaClass.name, listOf(nextUpdate.time))
    }
}
