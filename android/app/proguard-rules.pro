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
