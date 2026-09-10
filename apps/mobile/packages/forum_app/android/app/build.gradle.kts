import java.util.Properties

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

// Client identifiers only. Master Secret must never enter an APK.
val pushConfig = Properties()
val pushFile = rootProject.file("push.properties")
if (pushFile.exists()) pushFile.inputStream().use { pushConfig.load(it) }
val pushKey = pushConfig.getProperty("JPUSH_APPKEY", "")
val vendorKeys = mapOf(
    "xiaomi" to listOf("XIAOMI_APPID", "XIAOMI_APPKEY"),
    "oppo" to listOf("OPPO_APPID", "OPPO_APPKEY", "OPPO_APPSECRET"),
    "vivo" to listOf("VIVO_APPID", "VIVO_APPKEY"),
    "honor" to listOf("HONOR_APPID"),
    "meizu" to listOf("MEIZU_APPID", "MEIZU_APPKEY"),
)
val vendors = pushConfig.getProperty("VENDORS", "").split(",").filter { it.isNotBlank() }
require(vendors.all { it in vendorKeys || it == "huawei" }) { "Unknown push vendor" }

val releaseKeys = Properties()
val releaseKeysFile = rootProject.file("key.properties")
if (releaseKeysFile.exists()) {
    releaseKeysFile.inputStream().use { releaseKeys.load(it) }
}

android {
    namespace = "tj.yourtj.forum_app"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    buildFeatures { buildConfig = true }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        applicationId = "tj.yourtj.forum_app"
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName

        buildConfigField("boolean", "JPUSH_CONFIGURED", pushKey.isNotBlank().toString())
        manifestPlaceholders["JPUSH_APPKEY"] = pushKey.ifBlank { "000000000000000000000000" }
        manifestPlaceholders["JPUSH_CHANNEL"] = "github"
        for (vendor in vendors) for (key in vendorKeys[vendor].orEmpty()) {
            val value = pushConfig.getProperty(key, "")
            require(value.isNotBlank()) { "Missing push parameter: $key" }
            manifestPlaceholders[key] = value
        }
        // flutter_appauth:OIDC custom scheme 回跳(yourtj://callback)。
        manifestPlaceholders["appAuthRedirectScheme"] = "yourtj"
    }

    signingConfigs {
        create("release") {
            if (releaseKeysFile.exists()) {
                keyAlias = releaseKeys.getProperty("keyAlias")
                keyPassword = releaseKeys.getProperty("keyPassword")
                storeFile = releaseKeys.getProperty("storeFile")?.let { file(it) }
                storePassword = releaseKeys.getProperty("storePassword")
            }
        }
    }

    buildTypes {
        release {
            // A release must never silently fall back to the development key.
            signingConfig = signingConfigs.getByName("release")
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}

tasks.configureEach {
    if (name in listOf("validateSigningRelease", "packageRelease", "signReleaseBundle")) {
        doFirst {
            require(releaseKeysFile.exists()) { "Release signing requires android/key.properties" }
            for (key in listOf("keyAlias", "keyPassword", "storeFile", "storePassword")) {
                require(!releaseKeys.getProperty(key).isNullOrBlank()) { "Missing release signing property: $key" }
            }
        }
    }
}

// Pin both SDK and OEM adapters, including jcore's otherwise dynamic transitive dependency.
configurations.all { resolutionStrategy.force("cn.jiguang.sdk:jcore:5.5.2") }
dependencies {
    implementation("cn.jiguang.sdk:jpush:6.2.1")
    implementation("cn.jiguang.sdk:jcore:5.5.2")
    for (vendor in vendors) implementation("cn.jiguang.sdk.plugin:$vendor:6.2.1")
}
if ("huawei" in vendors) {
    require(file("agconnect-services.json").exists()) { "Huawei push requires app/agconnect-services.json" }
    apply(plugin = "com.huawei.agconnect")
}
