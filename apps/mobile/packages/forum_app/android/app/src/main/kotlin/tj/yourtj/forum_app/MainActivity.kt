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
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.security.MessageDigest

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        PushBridge.attach(this, MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "yourtj/push"))
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

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == PushBridge.PERMISSION_REQUEST) PushBridge.permissionResult(this)
    }
    override fun cleanUpFlutterEngine(flutterEngine: FlutterEngine) {
        PushBridge.detach()
        super.cleanUpFlutterEngine(flutterEngine)
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
