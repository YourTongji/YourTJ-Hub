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

/** Android-only JPush bridge. Initializers run only after explicit user consent. */
object PushBridge {
    const val PERMISSION_REQUEST = 8041
    private const val CHANNEL_ID = "yourtj_activity"
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

    private fun createChannel(context: Context) {
        if (Build.VERSION.SDK_INT >= 26) {
            context.getSystemService(NotificationManager::class.java).createNotificationChannel(
                NotificationChannel(CHANNEL_ID, "YourTJ", NotificationManager.IMPORTANCE_HIGH).apply {
                    description = "YourTJ activity notifications"
                    enableVibration(true)
                })
        }
    }
    private fun stop(context: Context) {
        val wasEnabled = consented(context)
        prefs(context).edit().putBoolean("enabled", false).apply()
        generation++
        registration?.error("cancelled", "Registration cancelled", null); registration = null
        pendingRoute = null
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
    override fun onRegister(context: Context, registrationId: String) {
        PushBridge.registered(context, registrationId)
    }
    override fun onNotifyMessageOpened(context: Context, message: NotificationMessage) {
        PushBridge.opened(context, PushBridge.routeFromExtra(message.notificationExtras))
        context.startActivity(Intent(context, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP))
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
        startActivity(Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP))
        finish()
    }
}
