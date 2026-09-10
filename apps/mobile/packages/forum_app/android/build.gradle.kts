buildscript {
    repositories {
        google()
        mavenCentral()
        maven { url = uri("https://developer.huawei.com/repo/") }
    }
    dependencies {
        // AGConnect inspects this explicit dependency; match settings.gradle.kts.
        classpath("com.android.tools.build:gradle:9.0.1")
        classpath("com.huawei.agconnect:agcp:1.9.1.301") {
            exclude(group = "com.android.tools.build", module = "gradle")
        }
    }
}
allprojects {
    repositories {
        google()
        mavenCentral()
        maven { url = uri("https://developer.huawei.com/repo/") }
        maven { url = uri("https://developer.hihonor.com/repo/") }
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    project.layout.buildDirectory.value(newSubprojectBuildDir)
}
subprojects {
    project.evaluationDependsOn(":app")
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
