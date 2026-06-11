# Add project specific ProGuard rules here.
# By default, the flags in this file are appended to flags specified
# in /sdk/tools/proguard/proguard-android.txt

# Keep Gomobile bindings
-keep class com.steadyphoto.mobile.** { *; }
-dontwarn com.steadyphoto.mobile.**

# Keep Koin classes
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}

# Keep Room entities
-keepclassmembers class ** {
    @androidx.room.Entity <fields>;
}

# Preserve generic type signatures needed for reflection
-keepattributes Signature, InnerClasses, EnclosingMethod, *Annotation*

# 1. Keep Kotlin reflection metadata and attributes
-keepattributes Signature, InnerClasses, EnclosingMethod, *Annotation*
-keep class kotlin.reflect.jvm.internal.** { *; }

# 2. Prevent Ktor's TypeInfo and serialization tokens from being erased'
-keep class io.ktor.util.reflect.TypeInfo { *; }
-keep class io.ktor.client.statement.HttpResponse { *; }

# 3. If you use Ktor's ContentNegotiation plugin with Kotlinx Serialization'
-keepattributes *RuntimeVisibleAnnotations*, *AnnotationDefault*
-keep class kotlinx.serialization.json.** { *; }

# 
-keep class com.steadyphoto.sync.data.** { *; }

