# Firebase, GMS ve GCM Servisleri
-keep class com.google.firebase.** { *; }
-keep class com.google.android.gms.** { *; }
-keep class com.google.android.gms.gcm.** { *; }
-keepattributes *Annotation*,Signature,EnclosingMethod

# Servislerin ve Alıcıların (BroadcastReceiver) R8 tarafından bozulmasını önle
-keep public class * extends android.app.Service { *; }
-keep public class * extends android.content.BroadcastReceiver { *; }

# Dagger / Dependency Injection koruması
-keep class dagger.** { *; }
-keep class * extends dagger.internal.Binding { *; }
-dontwarn dagger.internal.codegen.**

# Google Play Core / Split Install uyarılarını yoksay
-dontwarn com.google.android.play.core.**
-keep class com.google.android.play.core.** { *; }

# Flutter embedding sınıfları
-keep class io.flutter.app.** { *; }
-keep class io.flutter.plugin.** { *; }
-keep class io.flutter.util.** { *; }
-keep class io.flutter.view.** { *; }
-keep class io.flutter.embedding.** { *; }

# WorkManager / Room (release'te açılışta çökme düzeltmesi):
# Room, veritabanı sınıfını (ör. WorkDatabase_Impl) reflection ile oluşturur;
# R8 bu sınıfı ya da boş constructor'ını silerse uygulama açılırken çöker.
-keep class * extends androidx.room.RoomDatabase { <init>(); }
-keep class androidx.work.impl.WorkDatabase_Impl { *; }
-keep class androidx.work.** { *; }
-dontwarn androidx.work.**
