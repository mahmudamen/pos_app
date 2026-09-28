pluginManagement {
    val flutterSdkPath =
        run {
            val properties = java.util.Properties()
            file("local.properties").inputStream().use { properties.load(it) }
            val flutterSdkPath = properties.getProperty("flutter.sdk")
            require(flutterSdkPath != null) { "flutter.sdk not set in local.properties" }
            flutterSdkPath
        }

    includeBuild("$flutterSdkPath/packages/flutter_tools/gradle")

    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

plugins {
    id("dev.flutter.flutter-plugin-loader") version "1.0.0"
    id("com.android.application") version "9.1.0" apply false
    id("org.jetbrains.kotlin.android") version "2.4.0" apply false
    // Added for Firebase. The google-services Gradle plugin fails the build when
    // android/app/google-services.json is missing, which is deliberate: a
    // release build with no Firebase config is a build pointing at the wrong
    // project, and it should fail loudly here rather than ship silently.
    id("com.google.gms.google-services") version "4.4.2" apply false
}

include(":app")
