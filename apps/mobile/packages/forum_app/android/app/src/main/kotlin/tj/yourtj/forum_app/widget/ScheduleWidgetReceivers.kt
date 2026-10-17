package tj.yourtj.forum_app.widget

import android.appwidget.AppWidgetManager
import android.appwidget.AppWidgetProviderInfo
import android.content.Context
import androidx.collection.intSetOf
import androidx.glance.GlanceId
import androidx.glance.action.ActionParameters
import androidx.glance.appwidget.GlanceAppWidgetManager
import androidx.glance.appwidget.GlanceAppWidgetManager.Companion.SET_WIDGET_PREVIEWS_RESULT_SUCCESS
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.action.ActionCallback
import androidx.glance.appwidget.state.updateAppWidgetState
import es.antonborri.home_widget.HomeWidgetGlanceState
import es.antonborri.home_widget.HomeWidgetGlanceStateDefinition
import es.antonborri.home_widget.HomeWidgetGlanceWidgetReceiver
import es.antonborri.home_widget.HomeWidgetPlugin
import es.antonborri.home_widget.HomeWidgetScheduler
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.util.Date
import kotlin.reflect.KClass

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

class TodayScheduleWidgetReceiver : HomeWidgetGlanceWidgetReceiver<TodayScheduleWidget>() {
    override val glanceAppWidget = TodayScheduleWidget()
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
