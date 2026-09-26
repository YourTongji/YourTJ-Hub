package tj.yourtj.forum_app

import android.app.UiModeManager
import android.content.Context
import android.content.Intent
import android.content.res.Configuration
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.core.content.FileProvider
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import tj.yourtj.forum_app.widget.publishScheduleWidgetPreviews
import java.io.File
import java.security.MessageDigest

internal object NativeThemeMode {
    fun isDark(context: Context): Boolean {
        val saved = context.getSharedPreferences("FlutterSharedPreferences", Context.MODE_PRIVATE)
            .getString("flutter.theme_mode", "system")
        return when (saved) {
            "dark" -> true
            "light" -> false
            else -> systemIsDark(context)
        }
    }

    private fun systemIsDark(context: Context): Boolean {
        val mode = context.getSystemService(UiModeManager::class.java).nightMode
        return when (mode) {
            UiModeManager.MODE_NIGHT_YES -> true
            UiModeManager.MODE_NIGHT_NO -> false
            else -> context.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK == Configuration.UI_MODE_NIGHT_YES
        }
    }

    fun apply(context: Context, mode: String) {
        val nativeMode = when (mode) {
            "dark" -> UiModeManager.MODE_NIGHT_YES
            "light" -> UiModeManager.MODE_NIGHT_NO
            // AUTO clears the app override so Android follows the device theme.
            "system" -> UiModeManager.MODE_NIGHT_AUTO
            else -> return
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            context.getSystemService(UiModeManager::class.java).setApplicationNightMode(nativeMode)
        }
    }
}

class MainActivity : FlutterActivity() {
    private var oidcEventSink: EventChannel.EventSink? = null
    private var pendingOidcCallback: String? = null
    private var hasResumedOnce = false
    private var fullyDrawnReported = false
    private var startupLaunchKind = "cold"

    companion object {
        private var hasStartedActivityInProcess = false
    }

    override fun onCreate(savedInstanceState: android.os.Bundle?) {
        startupLaunchKind = if (hasStartedActivityInProcess) "warm" else "cold"
        hasStartedActivityInProcess = true
        setTheme(if (NativeThemeMode.isDark(this)) R.style.LaunchThemeDark else R.style.LaunchThemeLight)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) installSplashScreen()
        super.onCreate(savedInstanceState)
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        publishScheduleWidgetPreviews(applicationContext)
        PushBridge.attach(this, MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/push"))
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/startup")
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "setThemeMode" -> {
                        NativeThemeMode.apply(this, call.argument<String>("mode") ?: "")
                        result.success(true)
                    }
                    "getDeviceProfile" -> result.success(
                        mapOf(
                            "device" to "${Build.MANUFACTURER} ${Build.MODEL}".trim(),
                            "osVersion" to "Android ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})",
                            "refreshRateHz" to currentDisplayRefreshRate(),
                            "launchKind" to startupLaunchKind,
                        ),
                    )
                    "reportFullyDrawn" -> {
                        if (!fullyDrawnReported) {
                            fullyDrawnReported = true
                            reportFullyDrawn()
                        }
                        result.success(true)
                    }
                    else -> result.notImplemented()
                }
            }
        pendingOidcCallback = exactOidcCallback(intent)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/oidc")
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "getInitialCallback" -> {
                        val callback = pendingOidcCallback
                        pendingOidcCallback = null
                        result.success(callback)
                    }
                    else -> result.notImplemented()
                }
            }
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/oidc/callbacks")
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    oidcEventSink = events
                }

                override fun onCancel(arguments: Any?) {
                    oidcEventSink = null
                }
            })
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/app_updates")
            .setMethodCallHandler { call, result ->
                try {
                    when (call.method) {
                        "getInfo" -> {
                            val installed = packageManager.getPackageInfo(packageName, 0)
                            result.success(mapOf("version" to installed.versionName,
                                "buildNumber" to versionCode(installed),
                                "abis" to Build.SUPPORTED_ABIS.toList()))
                        }
                        "install" -> result.success(installUpdate(call.argument<String>("path") ?: ""))
                        else -> result.notImplemented()
                    }
                } catch (error: Exception) {
                    result.error("update_failed", "The update could not be verified or opened.", null)
                }
            }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        startupLaunchKind = "hot"
        val callback = exactOidcCallback(intent) ?: return
        val sink = oidcEventSink
        if (sink != null) {
            sink.success(callback)
        } else {
            pendingOidcCallback = callback
        }
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == PushBridge.PERMISSION_REQUEST) PushBridge.permissionResult(this)
    }

    override fun onResume() {
        super.onResume()
        if (hasResumedOnce) startupLaunchKind = "hot"
        hasResumedOnce = true
    }

    @Suppress("DEPRECATION")
    private fun currentDisplayRefreshRate(): Float {
        val currentDisplay = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            display
        } else {
            windowManager.defaultDisplay
        }
        return currentDisplay?.refreshRate ?: 0f
    }

    override fun cleanUpFlutterEngine(flutterEngine: FlutterEngine) {
        oidcEventSink = null
        pendingOidcCallback = null
        PushBridge.detach()
        super.cleanUpFlutterEngine(flutterEngine)
    }

    private fun exactOidcCallback(intent: Intent?): String? {
        val data = intent?.data ?: return null
        if (data.scheme != "yourtj" || data.host != "callback") return null
        if (!data.userInfo.isNullOrEmpty() || data.fragment != null) return null
        if (data.port != -1 || !data.path.isNullOrEmpty()) return null
        return data.toString()
    }

    @Suppress("DEPRECATION")
    private fun versionCode(info: PackageInfo): Long =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) info.longVersionCode else info.versionCode.toLong()

    @Suppress("DEPRECATION")
    private fun certificates(info: PackageInfo): Set<String> {
        val signatures = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P)
            info.signingInfo?.apkContentsSigners else info.signatures
        return signatures.orEmpty().map { signature ->
            MessageDigest.getInstance("SHA-256").digest(signature.toByteArray())
                .joinToString("") { "%02x".format(it) }
        }.toSet()
    }

    @Suppress("DEPRECATION")
    private fun installUpdate(path: String): Boolean {
        val file = File(path).canonicalFile
        val directory = File(cacheDir, "updates").canonicalFile
        require(file.parentFile == directory && file.extension == "apk" && file.isFile)
        val flags = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P)
            PackageManager.GET_SIGNING_CERTIFICATES else PackageManager.GET_SIGNATURES
        val installed = packageManager.getPackageInfo(packageName, flags)
        val archive = packageManager.getPackageArchiveInfo(file.path, flags)
            ?: throw IllegalArgumentException("Invalid APK")
        require(archive.packageName == packageName && versionCode(archive) > versionCode(installed))
        val signingCertificates = certificates(installed)
        require(signingCertificates.isNotEmpty() && certificates(archive) == signingCertificates)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O && !packageManager.canRequestPackageInstalls()) {
            startActivity(Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:$packageName")))
            return false
        }
        val uri = FileProvider.getUriForFile(this, "$packageName.updates", file)
        startActivity(Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uri, "application/vnd.android.package-archive")
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        })
        // Android's installer performs APK signature verification and requests
        // user confirmation. Opening it never means installation succeeded.
        return true
    }
}
