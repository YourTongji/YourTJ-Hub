package tj.yourtj.forum_app

import android.Manifest
import android.app.Activity
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import androidx.core.app.NotificationManagerCompat
import cn.jiguang.api.utils.JCollectionAuth
import cn.jpush.android.service.JCommonService
import cn.jpush.android.api.BasicPushNotificationBuilder
import cn.jpush.android.api.JPushInterface
import cn.jpush.android.api.NotificationMessage
import cn.jpush.android.data.JPushCollectControl
import cn.jpush.android.service.JPushMessageReceiver
import io.flutter.plugin.common.MethodChannel
import org.json.JSONObject
import org.json.JSONArray
import android.net.Uri

/** Android-only JPush bridge. Initializers run only after explicit user consent. */
object PushBridge {
    const val PERMISSION_REQUEST = 8041
    private const val CHANNEL_ID = "yourtj_activity"
    @Volatile private var foreground = false
    @Volatile private var currentPeer = 0L
    private var channel: MethodChannel? = null
    private var registration: MethodChannel.Result? = null
    private var permission: MethodChannel.Result? = null
    private var pendingRoute: String? = null
    private var generation = 0
    private val main = Handler(Looper.getMainLooper())
    private fun prefs(context: Context) = context.getSharedPreferences("yourtj_push", Context.MODE_PRIVATE)
    private fun allowed(context: Context) = NotificationManagerCompat.from(context).areNotificationsEnabled()
    private fun consented(context: Context) = prefs(context).getBoolean("enabled", false)

    fun attach(activity: Activity, channel: MethodChannel) {
        this.channel = channel
        channel.setMethodCallHandler { call, result ->
            try {
                when (call.method) {
                    "configured" -> result.success(BuildConfig.JPUSH_CONFIGURED)
                    "permission" -> result.success(allowed(activity))
                    "requestPermission" -> {
                        createChannel(activity)
                        if (Build.VERSION.SDK_INT >= 33 && activity.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                            permission?.error("cancelled", "Permission request replaced", null)
                            permission = result
                            activity.requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), PERMISSION_REQUEST)
                        } else result.success(allowed(activity))
                    }
                    "register" -> {
                        check(BuildConfig.JPUSH_CONFIGURED) { "Push is not configured" }
                        prefs(activity).edit().putBoolean("enabled", true).apply()
                        createChannel(activity)
                        JCollectionAuth.enableAutoWakeup(activity, false)
                        JCollectionAuth.enableDynamicLoad(activity, false)
                        JCollectionAuth.setAuth(activity, true)
                        JPushInterface.setDebugMode(false)
                        JPushInterface.setSmartPushEnable(activity, false)
                        JPushInterface.setDataInsightsEnable(activity, false)
                        JPushInterface.setGeofenceEnable(activity, false)
                        JPushInterface.setStatisticsEnable(false)
                        JPushInterface.setCollectControl(activity, JPushCollectControl.Builder()
                            .imei(false).imsi(false).mac(false).ssid(false).bssid(false).cell(false).wifi(false).build())
                        JPushInterface.setDefaultPushNotificationBuilder(BasicPushNotificationBuilder(activity).apply {
                            statusBarDrawable = R.drawable.ic_notification
                        })
                        JPushInterface.init(activity.applicationContext)
                        JPushInterface.resumePush(activity.applicationContext)
                        registration?.error("cancelled", "Registration replaced", null)
                        registration = result
                        val current = ++generation
                        val token = JPushInterface.getRegistrationID(activity)
                        if (!token.isNullOrBlank()) registered(activity, token)
                        main.postDelayed({
                            if (current == generation) {
                                registration?.error("registration_timeout", "Push registration timed out", null)
                                registration = null
                            }
                        }, 20000)
                    }
                    "stop" -> { stop(activity); result.success(null) }
                    "presentation" -> {
                        val args = call.arguments as? Map<*, *>
                        prefs(activity).edit().putLong("userId", (args?.get("userId") as? Number)?.toLong() ?: 0L).apply()
                        currentPeer = (args?.get("peerId") as? Number)?.toLong() ?: 0L
                        foreground = args?.get("foreground") == true
                        result.success(null)
                    }
                    "initialRoute" -> { result.success(pendingRoute); pendingRoute = null }
                    else -> result.notImplemented()
                }
            } catch (_: Exception) {
                result.error("push_failed", "Push operation failed. Check configuration and connectivity.", null)
            }
        }
    }

    fun permissionResult(activity: Activity) {
        permission?.success(allowed(activity)); permission = null
    }
    fun registered(context: Context, token: String) {
        main.post {
            if (!consented(context)) return@post
            if (registration != null) {
                registration?.success(token); registration = null
            } else channel?.invokeMethod("token", token)
        }
    }
    fun opened(context: Context, route: String?) {
        if (!consented(context)) return
        main.post {
            pendingRoute = route
            channel?.invokeMethod("opened", route)
            // Retain until Dart acknowledges after session restoration.
        }
    }
    fun routeFromExtra(raw: String?): String? = try {
        if (raw == null) null else JSONObject(raw).optString("route").takeIf { it.isNotBlank() }
    } catch (_: Exception) { null }

    fun setForeground(value: Boolean) { foreground = value }

    @Synchronized fun shouldPresent(context: Context, raw: String?): Boolean {
        if (!consented(context)) return false
        val extras = try { JSONObject(raw ?: "{}") } catch (_: Exception) { return false }
        val route = Uri.parse(extras.optString("route"))
        val recipient = extras.optLong("recipientId", 0L).takeIf { it > 0 }
            ?: route.getQueryParameter("recipientId")?.toLongOrNull()
        val owner = prefs(context).getLong("userId", 0L)
        if (recipient != null && recipient != owner) return false
        if (route.path == "/chat" && (recipient == null ||
                foreground && currentPeer > 0 && route.getQueryParameter("userId")?.toLongOrNull() == currentPeer)) return false
        val kind = extras.optString("sourceKind")
        val source = extras.optString("sourceId")
        if (kind.isBlank() || source.isBlank()) return true // Legacy activity pushes.
        val key = "$owner:$kind:$source"
        val stored = prefs(context).getString("shownSources", "[]")
        val recent = try { JSONArray(stored) } catch (_: Exception) { JSONArray() }
        val keys = (0 until recent.length()).map { recent.optString(it) }.toMutableList()
        if (key in keys) return false
        keys.add(key)
        prefs(context).edit().putString("shownSources", JSONArray(keys.takeLast(256)).toString()).apply()
        return true
    }

    private fun createChannel(context: Context) {
        if (Build.VERSION.SDK_INT >= 26) {
            val manager = context.getSystemService(NotificationManager::class.java)
            // Keep the legacy channel: deleting it would discard the user's OS choices.
            val channels = listOf(
                Triple(CHANNEL_ID, "YourTJ", NotificationManager.IMPORTANCE_HIGH),
                Triple("yourtj_chat_v2", context.getString(R.string.push_chat), NotificationManager.IMPORTANCE_DEFAULT),
                Triple("yourtj_mentions_and_replies_v2", context.getString(R.string.push_mentions), NotificationManager.IMPORTANCE_DEFAULT),
                Triple("yourtj_activity_v2", context.getString(R.string.push_activity), NotificationManager.IMPORTANCE_LOW),
                Triple("yourtj_system_v2", context.getString(R.string.push_system), NotificationManager.IMPORTANCE_DEFAULT)
            )
            for ((id, name, importance) in channels) {
                manager.createNotificationChannel(NotificationChannel(id, name, importance))
            }
        }
    }
    private fun stop(context: Context) {
        val wasEnabled = consented(context)
        prefs(context).edit().putBoolean("enabled", false).apply()
        generation++
        registration?.error("cancelled", "Registration cancelled", null); registration = null
        pendingRoute = null
        currentPeer = 0L
        prefs(context).edit().remove("userId").remove("shownSources").apply()
        NotificationManagerCompat.from(context).cancelAll()
        if (wasEnabled) {
            JPushInterface.stopPush(context.applicationContext)
            JCollectionAuth.setAuth(context, false)
        }
    }
    fun detach() {
        channel?.setMethodCallHandler(null); channel = null
        permission?.error("cancelled", "Activity closed", null); permission = null
    }
}

class PushService : JCommonService()

class PushReceiver : JPushMessageReceiver() {
    override fun isNeedShowNotification(context: Context, message: NotificationMessage, developerArg0: String?): Boolean =
        PushBridge.shouldPresent(context, message.notificationExtras)
    override fun onRegister(context: Context, registrationId: String) {
        PushBridge.registered(context, registrationId)
    }
    override fun onNotifyMessageOpened(context: Context, message: NotificationMessage) {
        PushBridge.opened(context, PushBridge.routeFromExtra(message.notificationExtras))
        context.startActivity(Intent(context, ThemeRouterActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP))
    }
}

/** OEM channels launch this activity; Flutter never treats the OEM URI as a deep link. */
class PushOpenActivity : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // SDK 6.x: Huawei carries the envelope in data; other channels use JMessageExtra.
        val envelope = intent.data?.toString() ?: intent.getStringExtra("JMessageExtra")
        val extra = try {
            envelope?.let { JSONObject(it).optString("n_extras") }
                ?: intent.getStringExtra(JPushInterface.EXTRA_EXTRA)
        } catch (_: Exception) { null }
        PushBridge.opened(this, PushBridge.routeFromExtra(extra) ?: "/notifications")
        startActivity(Intent(this, ThemeRouterActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP))
        finish()
    }
}
