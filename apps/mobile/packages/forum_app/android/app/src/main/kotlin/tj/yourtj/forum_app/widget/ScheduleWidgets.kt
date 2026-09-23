package tj.yourtj.forum_app.widget

import android.appwidget.AppWidgetProviderInfo
import android.content.Context
import android.net.Uri
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.collection.intSetOf
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.LocalSize
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetManager
import androidx.glance.appwidget.GlanceAppWidgetManager.Companion.SET_WIDGET_PREVIEWS_RESULT_SUCCESS
import androidx.glance.appwidget.GlanceAppWidgetReceiver
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
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxHeight
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
import androidx.glance.unit.ColorProvider
import androidx.core.content.ContextCompat
import es.antonborri.home_widget.HomeWidgetGlanceState
import es.antonborri.home_widget.HomeWidgetGlanceStateDefinition
import es.antonborri.home_widget.HomeWidgetGlanceWidgetReceiver
import es.antonborri.home_widget.actionStartActivity
import tj.yourtj.forum_app.MainActivity
import tj.yourtj.forum_app.R
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Date
import java.util.Locale
import java.util.TimeZone
import kotlin.reflect.KClass

private val foreground = ColorProvider(R.color.widget_foreground)
private val muted = ColorProvider(R.color.widget_muted)
private val accent = ColorProvider(R.color.widget_accent)
private val divider = ColorProvider(R.color.widget_divider)

private fun courseColor(slot: Int) = ColorProvider(
    when (slot) {
        1 -> R.color.course_slot_1
        2 -> R.color.course_slot_2
        3 -> R.color.course_slot_3
        4 -> R.color.course_slot_4
        5 -> R.color.course_slot_5
        6 -> R.color.course_slot_6
        7 -> R.color.course_slot_7
        else -> R.color.course_slot_8
    },
)

abstract class ScheduleGlanceWidget : GlanceAppWidget() {
    override val stateDefinition = HomeWidgetGlanceStateDefinition()
    override val sizeMode: SizeMode = SizeMode.Exact

    @Composable
    protected fun Surface(
        context: Context,
        description: String,
        transparency: Int,
        focusId: String? = null,
        horizontalPadding: Dp? = null,
        content: @Composable () -> Unit,
    ) {
        val uri = Uri.parse("yourtj://campus/today${focusId?.let { "?focus=$it" } ?: ""}")
        val verticalPadding = if (LocalSize.current.height < 80.dp) 8.dp else 12.dp
        val backgroundColor = Color(ContextCompat.getColor(context, R.color.widget_background))
            .copy(alpha = 1f - transparency.coerceIn(0, 15) / 100f)
        Box(
            modifier = GlanceModifier
                .fillMaxSize()
                .background(ColorProvider(backgroundColor))
                .cornerRadius(24.dp)
                .semantics { contentDescription = description }
                .clickable(actionStartActivity<MainActivity>(context, uri))
        ) {
            Box(
                modifier = GlanceModifier
                    .fillMaxSize()
                    .background(ImageProvider(R.drawable.schedule_widget_surface_overlay))
                    .padding(horizontal = horizontalPadding ?: verticalPadding, vertical = verticalPadding),
            ) { content() }
        }
    }
}

internal fun publishScheduleWidgetPreviews(context: Context) {
    if (android.os.Build.VERSION.SDK_INT < android.os.Build.VERSION_CODES.VANILLA_ICE_CREAM) return
    CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
        try {
            val installedAt = context.packageManager.getPackageInfo(context.packageName, 0).lastUpdateTime
            val preferences = context.getSharedPreferences("schedule_widget_previews", Context.MODE_PRIVATE)
            val manager = GlanceAppWidgetManager(context)
            suspend fun publish(receiver: KClass<out GlanceAppWidgetReceiver>) {
                val key = receiver.java.name
                if (preferences.getLong(key, Long.MIN_VALUE) == installedAt) return
                val result = manager.setWidgetPreviews(
                    receiver,
                    intSetOf(AppWidgetProviderInfo.WIDGET_CATEGORY_HOME_SCREEN),
                )
                if (result == SET_WIDGET_PREVIEWS_RESULT_SUCCESS) {
                    preferences.edit().putLong(key, installedAt).apply()
                }
            }
            publish(NextClassWidgetReceiver::class)
            publish(TodayScheduleWidgetReceiver::class)
        } catch (_: Exception) {
            // The static picker preview remains available if generation is unavailable.
        }
    }
}

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
        val teacher = course?.teacher?.takeIf { it.isNotBlank() }?.let { detailText("教师", "Teacher", it) }
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
            course?.id,
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

class TodayScheduleWidget : ScheduleGlanceWidget() {
    override val previewSizeMode = SizeMode.Responsive(
        setOf(
            DpSize(250.dp, 110.dp),
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
            val large = LocalSize.current.height >= 180.dp
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
    val minutes = maxOf(1L, (target.time - now.time + 59_999L) / 60_000L)
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

private fun dayHeaderText(date: String, week: Int?, adjustment: String?): String {
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

private fun detailText(zhLabel: String, enLabel: String, value: String): String =
    "${if (Locale.getDefault().language == "zh") zhLabel else enLabel}：$value"

private fun updatedText(value: Date): String =
    if (Locale.getDefault().language == "zh") "更新于 ${clock(value)}" else "Updated at ${clock(value)}"

private fun emptyStatusFor(day: ScheduleDay?): String = when (day?.kind) {
    "holiday" -> "holiday"
    "unknown", null -> "needsRefresh"
    else -> "noClasses"
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
        listOf(course.name, "${clock(course.startAt)}–${clock(course.endAt)}", course.locationText(), course.teacher)
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
private fun BrandMark() {
    Image(
        provider = ImageProvider(R.drawable.schedule_widget_brand),
        contentDescription = null,
        modifier = GlanceModifier.size(16.dp),
    )
}

@Composable
private fun EmptyDay(status: String, day: ScheduleDay?, hasProjection: Boolean) {
    Text(
        if (status == "holiday") day?.adjustmentLabel ?: labelFor(status) else labelFor(status),
        style = TextStyle(color = foreground, fontSize = 14.sp, fontWeight = FontWeight.Bold),
    )
    supportFor(status, hasProjection)?.let {
        Spacer(GlanceModifier.height(4.dp))
        Text(it, style = TextStyle(color = muted, fontSize = 11.sp))
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
        Column {
            Text(
                course.name,
                maxLines = 2,
                style = TextStyle(
                    color = if (current) accent else foreground,
                    fontSize = if (large) 14.sp else 13.sp,
                    fontWeight = if (current) FontWeight.Bold else FontWeight.Normal,
                ),
            )
            val location = course.locationText()
            if (course.teacher.isNotBlank() || location.isNotBlank()) {
                Text(
                    listOf(location, course.teacher).filter { it.isNotBlank() }.joinToString(" · "),
                    style = TextStyle(color = muted, fontSize = 11.sp),
                )
            }
            Text(
                "${clock(course.startAt)}–${clock(course.endAt)}",
                style = TextStyle(color = muted, fontSize = 11.sp),
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

private fun tomorrowDate(now: Date = Date()): String {
    val calendar = Calendar.getInstance(TimeZone.getTimeZone("Asia/Shanghai"))
    calendar.time = now
    calendar.add(Calendar.DAY_OF_MONTH, 1)
    return schoolDate(calendar.time)
}

private fun previewScheduleDay(now: Date, offset: Int): ScheduleDay {
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
        week = null,
        source = "widget-picker-preview",
        kind = "normal",
        adjustmentLabel = null,
        courses = listOf(
            ScheduleCourse("widget-preview-$offset-1", "示例课程", "", "", "", timeAt(9, 0), timeAt(10, 35), 5),
            ScheduleCourse("widget-preview-$offset-2", "下一节课程", "", "", "", timeAt(11, 0), timeAt(12, 35), 2),
        ),
    )
}

private fun emptyStatus(value: String?): String = if (value == "needsData") "needsRefresh" else value ?: "needsRefresh"

class NextClassWidgetReceiver : HomeWidgetGlanceWidgetReceiver<NextClassWidget>() {
    override val glanceAppWidget = NextClassWidget()
}

class TodayScheduleWidgetReceiver : HomeWidgetGlanceWidgetReceiver<TodayScheduleWidget>() {
    override val glanceAppWidget = TodayScheduleWidget()
}
