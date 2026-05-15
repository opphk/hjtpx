buildscript {
    repositories {
        google()
        mavenCentral()
    }
}

plugins {
    id("com.android.library") version "8.2.0" apply false
    id("org.jetbrains.kotlin.android") version "1.9.21" apply false
    id("org.jetbrains.kotlin.kapt") version "1.9.21" apply false
}

tasks.register("clean", Delete::class) {
    delete(rootProject.buildDir)
}
