import org.jetbrains.kotlin.gradle.dsl.JvmTarget

// Standalone build that compiles the dnatsgen-generated Kotlin (Messages.kt,
// Api.kt, runtime/DnatsProto.kt) against the same stack the mods use:
// Kotlin 2.0.0 + kotlinx-serialization + jnats. This proves the generated
// protobuf-binary client/event code compiles. The mods pull these via Kotlin
// for Forge; here we declare them directly.
plugins {
    kotlin("jvm") version "2.0.0"
    kotlin("plugin.serialization") version "2.0.0"
    application
}

repositories {
    mavenCentral()
}

dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-protobuf:1.6.3")
    implementation("io.nats:jnats:2.20.4")
}

// `./gradlew run` executes the Go<->Kotlin cross-language interop check.
application {
    mainClass.set("ru.lnik801l.example.CrossLangCheckKt")
}

kotlin {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_21)
    }
}
