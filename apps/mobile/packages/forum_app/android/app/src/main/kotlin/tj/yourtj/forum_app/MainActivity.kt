package tj.yourtj.forum_app

import android.content.Intent
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.core.content.FileProvider
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.security.MessageDigest

class MainActivity : FlutterActivity() {
    private var oidcEventSink: EventChannel.EventSink? = null
    private var pendingOidcCallback: String? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        PushBridge.attach(this, MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/push"))
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
