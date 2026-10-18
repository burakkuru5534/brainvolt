import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_tr.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'generated/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('tr'),
  ];

  /// App name shown in the task switcher and app bar.
  ///
  /// In tr, this message translates to:
  /// **'BrainVolt'**
  String get appTitle;

  /// Greeting on the home screen.
  ///
  /// In tr, this message translates to:
  /// **'Bilgi yarışmasına hoş geldin!'**
  String get homeWelcome;

  /// Title of the settings screen.
  ///
  /// In tr, this message translates to:
  /// **'Ayarlar'**
  String get settingsTitle;

  /// Label of the language selector.
  ///
  /// In tr, this message translates to:
  /// **'Dil'**
  String get settingsLanguage;

  /// Follow the device language.
  ///
  /// In tr, this message translates to:
  /// **'Cihaz dili'**
  String get languageSystem;

  /// Turkish, always written in Turkish.
  ///
  /// In tr, this message translates to:
  /// **'Türkçe'**
  String get languageTurkish;

  /// English, always written in English.
  ///
  /// In tr, this message translates to:
  /// **'English'**
  String get languageEnglish;

  /// Client-side: the server could not be reached.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı kurulamadı. İnternet bağlantını kontrol et.'**
  String get errorNetwork;

  /// Client-side: the request timed out.
  ///
  /// In tr, this message translates to:
  /// **'Sunucu zamanında yanıt vermedi. Tekrar dene.'**
  String get errorTimeout;

  /// Client-side: unknown error without a server message.
  ///
  /// In tr, this message translates to:
  /// **'Beklenmeyen bir hata oluştu. Lütfen tekrar dene.'**
  String get errorUnknown;

  /// Server error code ACCOUNT_DELETION_PENDING (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Hesabın silinme sürecinde. Yalnızca silme işlemini iptal edebilirsin.'**
  String get error_ACCOUNT_DELETION_PENDING;

  /// Server error code ACCOUNT_SUSPENDED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Hesabın askıya alındı.'**
  String get error_ACCOUNT_SUSPENDED;

  /// Server error code ACCOUNT_UPGRADE_REQUIRED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu işlem için hesabını yükseltmelisin.'**
  String get error_ACCOUNT_UPGRADE_REQUIRED;

  /// Server error code AD_REWARD_LIMIT (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bugün için reklam ödülü sınırına ulaştın.'**
  String get error_AD_REWARD_LIMIT;

  /// Server error code AGE_CONFIRMATION_REQUIRED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Devam etmek için yaşını onaylamalısın.'**
  String get error_AGE_CONFIRMATION_REQUIRED;

  /// Server error code ALREADY_FRIENDS (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Zaten arkadaşsınız.'**
  String get error_ALREADY_FRIENDS;

  /// Server error code ALREADY_IN_MATCH (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Zaten devam eden bir maçın var.'**
  String get error_ALREADY_IN_MATCH;

  /// Server error code ALREADY_QUEUED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Zaten eşleşme sırasındasın.'**
  String get error_ALREADY_QUEUED;

  /// Server error code ALREADY_REPORTED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bunu zaten bildirdin.'**
  String get error_ALREADY_REPORTED;

  /// Server error code ANSWER_DUPLICATE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu soruyu zaten cevapladın.'**
  String get error_ANSWER_DUPLICATE;

  /// Server error code ANSWER_INVALID_OPTION (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Geçersiz cevap seçeneği.'**
  String get error_ANSWER_INVALID_OPTION;

  /// Server error code ANSWER_LATE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Cevap süresi doldu.'**
  String get error_ANSWER_LATE;

  /// Server error code APPEAL_EXISTS (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu yaptırım için zaten bir itirazın var.'**
  String get error_APPEAL_EXISTS;

  /// Server error code APP_VERSION_UNSUPPORTED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Devam etmek için uygulamayı güncellemelisin.'**
  String get error_APP_VERSION_UNSUPPORTED;

  /// Server error code BLOCKED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu kullanıcıyla etkileşim kuramazsın.'**
  String get error_BLOCKED;

  /// Server error code CHALLENGES_DISABLED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu kullanıcı şu anda meydan okuma kabul etmiyor.'**
  String get error_CHALLENGES_DISABLED;

  /// Server error code CHALLENGE_EXPIRED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Meydan okumanın süresi doldu.'**
  String get error_CHALLENGE_EXPIRED;

  /// Server error code CHANGE_TOO_SOON (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bunu henüz değiştiremezsin. Daha sonra tekrar dene.'**
  String get error_CHANGE_TOO_SOON;

  /// Server error code COUPLE_COOLDOWN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Yeni bir çift oluşturmak için biraz beklemelisin.'**
  String get error_COUPLE_COOLDOWN;

  /// Server error code COUPLE_EXISTS (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Zaten bir çiftin var.'**
  String get error_COUPLE_EXISTS;

  /// Server error code DAILY_ATTEMPT_IN_PROGRESS (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Devam eden bir günlük denemen var.'**
  String get error_DAILY_ATTEMPT_IN_PROGRESS;

  /// Server error code DAILY_ATTEMPT_USED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bugünkü resmi denemeni zaten kullandın.'**
  String get error_DAILY_ATTEMPT_USED;

  /// Server error code DAILY_NOT_AVAILABLE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Günün meydan okuması henüz hazır değil.'**
  String get error_DAILY_NOT_AVAILABLE;

  /// Server error code FORBIDDEN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu işlem için yetkin yok.'**
  String get error_FORBIDDEN;

  /// Server error code FRIEND_LIMIT_REACHED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş sınırına ulaştın.'**
  String get error_FRIEND_LIMIT_REACHED;

  /// Server error code IDEMPOTENCY_KEY_REUSED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu istek anahtarı farklı bir istek için zaten kullanıldı.'**
  String get error_IDEMPOTENCY_KEY_REUSED;

  /// Server error code IDEMPOTENT_REQUEST_IN_PROGRESS (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu istek hâlâ işleniyor. Birazdan tekrar dene.'**
  String get error_IDEMPOTENT_REQUEST_IN_PROGRESS;

  /// Server error code INTERNAL (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Beklenmeyen bir hata oluştu. Lütfen tekrar dene.'**
  String get error_INTERNAL;

  /// Server error code INVITE_CODE_EXPIRED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Davet kodunun süresi doldu.'**
  String get error_INVITE_CODE_EXPIRED;

  /// Server error code INVITE_CODE_INVALID (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Davet kodu geçersiz.'**
  String get error_INVITE_CODE_INVALID;

  /// Server error code MALFORMED_REQUEST (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'İstek okunamadı.'**
  String get error_MALFORMED_REQUEST;

  /// Server error code MATCH_NOT_FOUND (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Maç bulunamadı.'**
  String get error_MATCH_NOT_FOUND;

  /// Server error code MATCH_STATE_INVALID (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu işlem maçın şu anki aşamasında yapılamaz.'**
  String get error_MATCH_STATE_INVALID;

  /// Server error code METHOD_NOT_ALLOWED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu işlem bu adreste desteklenmiyor.'**
  String get error_METHOD_NOT_ALLOWED;

  /// Server error code MODE_NOT_ALLOWED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu oyun modu kombinasyonu desteklenmiyor.'**
  String get error_MODE_NOT_ALLOWED;

  /// Server error code NOT_A_PARTICIPANT (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu maçın oyuncusu değilsin.'**
  String get error_NOT_A_PARTICIPANT;

  /// Server error code NOT_CAPTAIN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bunu yalnızca takım kaptanı yapabilir.'**
  String get error_NOT_CAPTAIN;

  /// Server error code NOT_ENOUGH_QUESTIONS (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu ayarlar için yeterli soru yok.'**
  String get error_NOT_ENOUGH_QUESTIONS;

  /// Server error code NOT_FOUND (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Aradığın şey bulunamadı.'**
  String get error_NOT_FOUND;

  /// Server error code NOT_YOUR_TURN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Sıra sende değil.'**
  String get error_NOT_YOUR_TURN;

  /// Server error code NO_ACTIVE_SEASON (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Şu anda aktif bir sezon yok.'**
  String get error_NO_ACTIVE_SEASON;

  /// Server error code PAYLOAD_TOO_LARGE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'İstek çok büyük.'**
  String get error_PAYLOAD_TOO_LARGE;

  /// Server error code PICK_NOT_ALLOWED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu kategori seçilemez.'**
  String get error_PICK_NOT_ALLOWED;

  /// Server error code PURCHASE_ALREADY_PROCESSED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu satın alma zaten işlendi.'**
  String get error_PURCHASE_ALREADY_PROCESSED;

  /// Server error code PURCHASE_INVALID (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Satın alma doğrulanamadı.'**
  String get error_PURCHASE_INVALID;

  /// Server error code QUESTION_IN_USE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu soru maçlarda kullanıldığı için bu şekilde değiştirilemez.'**
  String get error_QUESTION_IN_USE;

  /// Server error code QUESTION_SOURCE_REQUIRED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Sorunun kaynağı ve lisansı belirtilmeli.'**
  String get error_QUESTION_SOURCE_REQUIRED;

  /// Server error code QUEUE_COOLDOWN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bir süre eşleşme sırasına giremezsin.'**
  String get error_QUEUE_COOLDOWN;

  /// Server error code QUIZ_LIMIT_REACHED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Quiz sınırına ulaştın.'**
  String get error_QUIZ_LIMIT_REACHED;

  /// Server error code QUIZ_SIZE_INVALID (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Quiz soru sayısı izin verilen aralıkta değil.'**
  String get error_QUIZ_SIZE_INVALID;

  /// Server error code RATE_LIMITED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Çok fazla istek gönderdin. Biraz bekleyip tekrar dene.'**
  String get error_RATE_LIMITED;

  /// Server error code REPORT_LIMIT_REACHED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bugün için bildirim sınırına ulaştın.'**
  String get error_REPORT_LIMIT_REACHED;

  /// Server error code REQUEST_PENDING (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bekleyen bir arkadaşlık isteği zaten var.'**
  String get error_REQUEST_PENDING;

  /// Server error code REVENGE_NOT_ALLOWED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Rövanş maçı şu anda mümkün değil.'**
  String get error_REVENGE_NOT_ALLOWED;

  /// Server error code ROOM_CLOSED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Oda kapandı.'**
  String get error_ROOM_CLOSED;

  /// Server error code ROOM_FRIENDS_ONLY (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu odaya yalnızca kurucunun arkadaşları katılabilir.'**
  String get error_ROOM_FRIENDS_ONLY;

  /// Server error code ROOM_FULL (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Oda dolu.'**
  String get error_ROOM_FULL;

  /// Server error code SANCTIONED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu özellik hesabın için geçici olarak kısıtlandı.'**
  String get error_SANCTIONED;

  /// Server error code SEAT_TAKEN_ELSEWHERE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu maça başka bir cihazdan bağlandın.'**
  String get error_SEAT_TAKEN_ELSEWHERE;

  /// Server error code SELF_REVIEW_FORBIDDEN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Kendi dosyanı inceleyemezsin.'**
  String get error_SELF_REVIEW_FORBIDDEN;

  /// Server error code TEAM_FULL (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Takım dolu.'**
  String get error_TEAM_FULL;

  /// Server error code TEAM_LIMIT_REACHED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Takım sınırına ulaştın.'**
  String get error_TEAM_LIMIT_REACHED;

  /// Server error code TEXT_REJECTED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu metin kurallarımıza uymuyor.'**
  String get error_TEXT_REJECTED;

  /// Server error code TOKEN_EXPIRED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Oturumunun süresi doldu. Lütfen tekrar dene.'**
  String get error_TOKEN_EXPIRED;

  /// Server error code TOURNAMENT_FULL (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Turnuva dolu.'**
  String get error_TOURNAMENT_FULL;

  /// Server error code TOURNAMENT_REGISTRATION_CLOSED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Turnuva kayıtları kapandı.'**
  String get error_TOURNAMENT_REGISTRATION_CLOSED;

  /// Server error code TOURNAMENT_REQUIREMENTS_NOT_MET (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu turnuvanın katılım şartlarını karşılamıyorsun.'**
  String get error_TOURNAMENT_REQUIREMENTS_NOT_MET;

  /// Server error code UNAUTHENTICATED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Oturum açman gerekiyor.'**
  String get error_UNAUTHENTICATED;

  /// Server error code UNAVAILABLE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Hizmet şu anda kullanılamıyor. Biraz sonra tekrar dene.'**
  String get error_UNAVAILABLE;

  /// Server error code USERNAME_TAKEN (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu kullanıcı adı zaten alınmış.'**
  String get error_USERNAME_TAKEN;

  /// Server error code VALIDATION_FAILED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Gönderilen bilgiler geçersiz.'**
  String get error_VALIDATION_FAILED;

  /// Server error code VERSION_CONFLICT (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu kayıt başka biri tarafından değiştirildi. Yenileyip tekrar dene.'**
  String get error_VERSION_CONFLICT;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'tr'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'tr':
      return AppLocalizationsTr();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
