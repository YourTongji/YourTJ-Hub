package tj.yourtj.forum_app.widget

import android.content.Context
import android.content.Intent
import android.content.res.Configuration
import android.net.Uri
import androidx.collection.intSetOf
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.glance.GlanceModifier
import androidx.glance.ImageProvider
import androidx.glance.LocalSize
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetManager
import androidx.glance.appwidget.GlanceAppWidgetManager.Companion.SET_WIDGET_PREVIEWS_RESULT_SUCCESS
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.cornerRadius
import androidx.glance.background
import androidx.glance.color.ColorProvider as DayNightColorProvider
import androidx.glance.layout.Box
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.padding
import androidx.glance.semantics.contentDescription
import androidx.glance.semantics.semantics
import androidx.glance.unit.ColorProvider
import androidx.core.content.ContextCompat
import android.appwidget.AppWidgetProviderInfo
import es.antonborri.home_widget.HomeWidgetGlanceStateDefinition
import es.antonborri.home_widget.HomeWidgetLaunchIntent
import tj.yourtj.forum_app.ThemeRouterActivity
import tj.yourtj.forum_app.R
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlin.reflect.KClass

internal val foreground = ColorProvider(R.color.widget_foreground)
internal val muted = ColorProvider(R.color.widget_muted)
internal val accent = ColorProvider(R.color.widget_accent)
internal val divider = ColorProvider(R.color.widget_divider)

internal fun courseColor(slot: Int) = ColorProvider(
    courseColorResource(slot),
)

private fun courseColorResource(slot: Int) = when (slot) {
    1 -> R.color.course_slot_1
    2 -> R.color.course_slot_2
    3 -> R.color.course_slot_3
    4 -> R.color.course_slot_4
    5 -> R.color.course_slot_5
    6 -> R.color.course_slot_6
    7 -> R.color.course_slot_7
    else -> R.color.course_slot_8
}

internal fun courseTint(context: Context, slot: Int) = DayNightColorProvider(
    day = Color(ContextCompat.getColor(
        context.createConfigurationContext(Configuration(context.resources.configuration).apply {
            uiMode = (uiMode and Configuration.UI_MODE_TYPE_MASK) or Configuration.UI_MODE_NIGHT_NO
        }),
        courseColorResource(slot),
    )).copy(alpha = 0.12f),
    night = Color(ContextCompat.getColor(
        context.createConfigurationContext(Configuration(context.resources.configuration).apply {
            uiMode = (uiMode and Configuration.UI_MODE_TYPE_MASK) or Configuration.UI_MODE_NIGHT_YES
        }),
        courseColorResource(slot),
    )).copy(alpha = 0.18f),
)

private fun widgetBackground(context: Context, transparency: Int): ColorProvider {
    val configuration = context.resources.configuration
    val alpha = 1f - transparency.coerceIn(0, 15) / 100f
    fun colorFor(nightMode: Int) = Color(
        ContextCompat.getColor(
            context.createConfigurationContext(
                Configuration(configuration).apply {
                    uiMode = (uiMode and Configuration.UI_MODE_TYPE_MASK) or nightMode
                },
            ),
            R.color.widget_background,
        ),
    ).copy(alpha = alpha)

    return DayNightColorProvider(
        day = colorFor(Configuration.UI_MODE_NIGHT_NO),
        night = colorFor(Configuration.UI_MODE_NIGHT_YES),
    )
}

abstract class ScheduleGlanceWidget : GlanceAppWidget() {
    override val stateDefinition = HomeWidgetGlanceStateDefinition()
    override val sizeMode: SizeMode = SizeMode.Exact

    @Composable
    protected fun Surface(
        context: Context,
        description: String,
        transparency: Int,
        horizontalPadding: Dp? = null,
        content: @Composable () -> Unit,
    ) {
        val uri = Uri.parse("yourtj://campus/today")
        val verticalPadding = if (LocalSize.current.height < 80.dp) 8.dp else 12.dp
        Box(
            modifier = GlanceModifier
                .fillMaxSize()
                .background(widgetBackground(context, transparency))
                .cornerRadius(24.dp)
                .semantics { contentDescription = description }
                .clickable(actionStartActivity(Intent(context, ThemeRouterActivity::class.java).apply {
                    data = uri
                    action = HomeWidgetLaunchIntent.HOME_WIDGET_LAUNCH_ACTION
                }))
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
            publish(CourseTimelineWidgetReceiver::class)
        } catch (_: Exception) {
            // The static picker preview remains available if generation is unavailable.
        }
    }
}
