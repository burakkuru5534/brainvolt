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
  /// **'Bugün bilgini sınamaya hazır mısın?'**
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

  /// Retry button.
  ///
  /// In tr, this message translates to:
  /// **'Tekrar dene'**
  String get retry;

  /// Dialog cancel button.
  ///
  /// In tr, this message translates to:
  /// **'Vazgeç'**
  String get cancel;

  /// Generic continue button.
  ///
  /// In tr, this message translates to:
  /// **'Devam et'**
  String get continueButton;

  /// Save button.
  ///
  /// In tr, this message translates to:
  /// **'Kaydet'**
  String get save;

  /// Splash screen: the session could not be started.
  ///
  /// In tr, this message translates to:
  /// **'Sunucuya bağlanamadık.'**
  String get splashError;

  /// Age gate (D-020) title.
  ///
  /// In tr, this message translates to:
  /// **'Hoş geldin!'**
  String get ageGateTitle;

  /// Age gate explanation.
  ///
  /// In tr, this message translates to:
  /// **'BrainVolt\'u kullanmak için en az 13 yaşında olmalısın.'**
  String get ageGateBody;

  /// Age gate checkbox.
  ///
  /// In tr, this message translates to:
  /// **'13 yaşından büyüğüm'**
  String get ageGateConfirm;

  /// Home greeting with the display name.
  ///
  /// In tr, this message translates to:
  /// **'Merhaba, {name}!'**
  String homeGreeting(String name);

  /// Home banner for guests.
  ///
  /// In tr, this message translates to:
  /// **'Misafir olarak oynuyorsun. İlerlemeni korumak için hesabını bağla.'**
  String get homeGuestBanner;

  /// Home banner for unverified email accounts (D-017).
  ///
  /// In tr, this message translates to:
  /// **'E-posta adresini doğrula; doğrulanana kadar misafir sınırları geçerli.'**
  String get homeVerifyBanner;

  /// Opens the account screen from the guest banner.
  ///
  /// In tr, this message translates to:
  /// **'Hesabını bağla'**
  String get linkAccountAction;

  /// Opens the verify-email screen.
  ///
  /// In tr, this message translates to:
  /// **'Doğrula'**
  String get verifyAction;

  /// Account screen title.
  ///
  /// In tr, this message translates to:
  /// **'Hesap'**
  String get accountTitle;

  /// Account screen text for guests.
  ///
  /// In tr, this message translates to:
  /// **'Misafir hesabı. Bir giriş yöntemi bağlarsan ilerlemen korunur ve başka cihazlarda da oynayabilirsin.'**
  String get accountGuestInfo;

  /// Account screen: signed-in email.
  ///
  /// In tr, this message translates to:
  /// **'{email} ile giriş yapıldı'**
  String accountSignedInAs(String email);

  /// Account screen when the account has no email.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın bağlı.'**
  String get accountSignedIn;

  /// Snackbar after a successful guest upgrade.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın bağlandı. İlerlemen korunuyor.'**
  String get accountLinked;

  /// Google sign-in button.
  ///
  /// In tr, this message translates to:
  /// **'Google ile devam et'**
  String get signInWithGoogle;

  /// Apple sign-in button (iOS).
  ///
  /// In tr, this message translates to:
  /// **'Apple ile devam et'**
  String get signInWithApple;

  /// Email sign-up button and screen title.
  ///
  /// In tr, this message translates to:
  /// **'E-posta ile kaydol'**
  String get signUpWithEmail;

  /// Email sign-in button.
  ///
  /// In tr, this message translates to:
  /// **'Hesabım var: e-posta ile giriş yap'**
  String get signInWithEmail;

  /// Email sign-in screen title.
  ///
  /// In tr, this message translates to:
  /// **'E-posta ile giriş'**
  String get emailSignInTitle;

  /// Email field label.
  ///
  /// In tr, this message translates to:
  /// **'E-posta'**
  String get emailLabel;

  /// Password field label.
  ///
  /// In tr, this message translates to:
  /// **'Şifre'**
  String get passwordLabel;

  /// Password field helper text.
  ///
  /// In tr, this message translates to:
  /// **'En az 6 karakter'**
  String get passwordHint;

  /// Email sign-up submit button.
  ///
  /// In tr, this message translates to:
  /// **'Kaydol'**
  String get signUpButton;

  /// Email sign-in submit button.
  ///
  /// In tr, this message translates to:
  /// **'Giriş yap'**
  String get signInButton;

  /// Sign-out button.
  ///
  /// In tr, this message translates to:
  /// **'Çıkış yap'**
  String get signOut;

  /// Sign-out confirmation.
  ///
  /// In tr, this message translates to:
  /// **'Çıkış yaparsan bu cihazda yeni bir misafir olarak devam edersin.'**
  String get signOutConfirm;

  /// Account deletion button.
  ///
  /// In tr, this message translates to:
  /// **'Hesabımı sil'**
  String get deleteAccount;

  /// Account deletion confirmation (D-022).
  ///
  /// In tr, this message translates to:
  /// **'Hesabın 14 gün sonra kalıcı olarak silinir. Bu süre içinde giriş yapıp iptal edebilirsin.'**
  String get deleteAccountConfirm;

  /// Link conflict dialog title (D-015).
  ///
  /// In tr, this message translates to:
  /// **'Hesap zaten kayıtlı'**
  String get linkConflictTitle;

  /// Link conflict dialog text (D-015).
  ///
  /// In tr, this message translates to:
  /// **'Bu hesap zaten kayıtlı. Misafir ilerlemen kaybolacak. Devam edilsin mi?'**
  String get linkConflictBody;

  /// Verify-email screen title.
  ///
  /// In tr, this message translates to:
  /// **'E-postanı doğrula'**
  String get verifyEmailTitle;

  /// Verify-email screen text.
  ///
  /// In tr, this message translates to:
  /// **'{email} adresine bir doğrulama bağlantısı gönderdik. Bağlantıyı açtıktan sonra aşağıya dokun.'**
  String verifyEmailBody(String email);

  /// Verify-email check button.
  ///
  /// In tr, this message translates to:
  /// **'Doğruladım'**
  String get verifyEmailDone;

  /// Resend verification email.
  ///
  /// In tr, this message translates to:
  /// **'Tekrar gönder'**
  String get verifyEmailResend;

  /// Snackbar after resending.
  ///
  /// In tr, this message translates to:
  /// **'Doğrulama e-postası gönderildi.'**
  String get verifyEmailSent;

  /// Snackbar when the check fails.
  ///
  /// In tr, this message translates to:
  /// **'E-posta henüz doğrulanmadı.'**
  String get verifyEmailNotYet;

  /// Snackbar after verification.
  ///
  /// In tr, this message translates to:
  /// **'E-postan doğrulandı.'**
  String get verifyEmailVerified;

  /// Sign-in error.
  ///
  /// In tr, this message translates to:
  /// **'E-posta veya şifre hatalı.'**
  String get authErrorInvalidCredential;

  /// Sign-up error.
  ///
  /// In tr, this message translates to:
  /// **'Şifre en az 6 karakter olmalı.'**
  String get authErrorWeakPassword;

  /// Sign-in error.
  ///
  /// In tr, this message translates to:
  /// **'Geçerli bir e-posta adresi gir.'**
  String get authErrorInvalidEmail;

  /// Sign-in error.
  ///
  /// In tr, this message translates to:
  /// **'Bu e-postayla kayıtlı bir hesap yok.'**
  String get authErrorUserNotFound;

  /// Sign-in error.
  ///
  /// In tr, this message translates to:
  /// **'Çok fazla deneme yaptın. Biraz sonra tekrar dene.'**
  String get authErrorTooManyRequests;

  /// Sign-in error.
  ///
  /// In tr, this message translates to:
  /// **'Bu giriş yöntemi bu cihazda kullanılamıyor.'**
  String get authErrorUnsupported;

  /// Account status screen title.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın askıya alındı'**
  String get accountSuspendedTitle;

  /// Account status screen text.
  ///
  /// In tr, this message translates to:
  /// **'Ayrıntılar için destek ekibiyle iletişime geç.'**
  String get accountSuspendedBody;

  /// Deletion pending screen title.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın silinecek'**
  String get deletionPendingTitle;

  /// Deletion pending text.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın {date} tarihinde kalıcı olarak silinecek.'**
  String deletionPendingBody(DateTime date);

  /// Cancels a pending deletion.
  ///
  /// In tr, this message translates to:
  /// **'Silmeyi iptal et'**
  String get cancelDeletion;

  /// Profile screen title.
  ///
  /// In tr, this message translates to:
  /// **'Profil'**
  String get profileTitle;

  /// Display name field.
  ///
  /// In tr, this message translates to:
  /// **'Görünen ad'**
  String get displayNameLabel;

  /// Display name helper text.
  ///
  /// In tr, this message translates to:
  /// **'2–24 karakter'**
  String get displayNameHint;

  /// Username field.
  ///
  /// In tr, this message translates to:
  /// **'Kullanıcı adı'**
  String get usernameLabel;

  /// Username helper text.
  ///
  /// In tr, this message translates to:
  /// **'3–20 karakter: küçük harf, rakam, _ ve .'**
  String get usernameHint;

  /// Username locked for restricted accounts.
  ///
  /// In tr, this message translates to:
  /// **'Kullanıcı adını değiştirmek için hesabını bağla ve doğrula.'**
  String get usernameGuestHint;

  /// Username change cooldown.
  ///
  /// In tr, this message translates to:
  /// **'Kullanıcı adını {date} tarihinden sonra değiştirebilirsin.'**
  String usernameAvailableAt(DateTime date);

  /// Avatar picker label.
  ///
  /// In tr, this message translates to:
  /// **'Avatar'**
  String get avatarLabel;

  /// Snackbar after saving the profile.
  ///
  /// In tr, this message translates to:
  /// **'Profil kaydedildi.'**
  String get profileSaved;

  /// Settings section.
  ///
  /// In tr, this message translates to:
  /// **'Oyun'**
  String get settingsGame;

  /// Setting.
  ///
  /// In tr, this message translates to:
  /// **'Ses efektleri'**
  String get settingsSound;

  /// Setting.
  ///
  /// In tr, this message translates to:
  /// **'Müzik'**
  String get settingsMusic;

  /// Setting.
  ///
  /// In tr, this message translates to:
  /// **'Titreşim'**
  String get settingsHaptics;

  /// Settings section.
  ///
  /// In tr, this message translates to:
  /// **'Gizlilik'**
  String get settingsPrivacy;

  /// Setting.
  ///
  /// In tr, this message translates to:
  /// **'Çevrimiçi durumumu göster'**
  String get settingsShowOnline;

  /// Setting.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlık isteklerine izin ver'**
  String get settingsAllowFriendRequests;

  /// Setting.
  ///
  /// In tr, this message translates to:
  /// **'Meydan okumalar'**
  String get settingsChallengesFrom;

  /// challenges_from value.
  ///
  /// In tr, this message translates to:
  /// **'Herkesten'**
  String get challengesEveryone;

  /// challenges_from value.
  ///
  /// In tr, this message translates to:
  /// **'Yalnızca arkadaşlardan'**
  String get challengesFriends;

  /// challenges_from value.
  ///
  /// In tr, this message translates to:
  /// **'Kimseden'**
  String get challengesNobody;

  /// Credits screen title.
  ///
  /// In tr, this message translates to:
  /// **'Kaynaklar ve Lisanslar'**
  String get creditsTitle;

  /// Credits: section of question sources (D-006).
  ///
  /// In tr, this message translates to:
  /// **'Soruların kaynakları'**
  String get creditsQuestions;

  /// Credits: license code of a source.
  ///
  /// In tr, this message translates to:
  /// **'Lisans: {license}'**
  String creditsLicense(String license);

  /// Credits: CC BY-SA share-alike note.
  ///
  /// In tr, this message translates to:
  /// **'Bu kaynaktan alınan sorular ve çevirileri aynı lisansla paylaşılır.'**
  String get creditsShareAlike;

  /// Snackbar after copying a link.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı kopyalandı.'**
  String get creditsLinkCopied;

  /// Credits: opens the licenses of the app's packages.
  ///
  /// In tr, this message translates to:
  /// **'Açık kaynak lisansları'**
  String get creditsOpenSource;

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

  /// Home: start a Quick Duel 1v1.
  ///
  /// In tr, this message translates to:
  /// **'Hızlı Düello'**
  String get playQuickDuel;

  /// Home: development-only practice match against the server bot.
  ///
  /// In tr, this message translates to:
  /// **'Bota karşı pratik'**
  String get playPractice;

  /// Queue screen title.
  ///
  /// In tr, this message translates to:
  /// **'Rakip aranıyor…'**
  String get searchingTitle;

  /// Queue screen: seconds waited.
  ///
  /// In tr, this message translates to:
  /// **'{seconds} sn'**
  String searchingElapsed(int seconds);

  /// Queue screen: leave the queue.
  ///
  /// In tr, this message translates to:
  /// **'Aramayı iptal et'**
  String get searchingCancel;

  /// Ready prompt (match-lifecycle §4).
  ///
  /// In tr, this message translates to:
  /// **'Maç bulundu — Hazır mısın?'**
  String get matchFoundTitle;

  /// Ready button.
  ///
  /// In tr, this message translates to:
  /// **'Hazırım'**
  String get readyButton;

  /// Ready countdown.
  ///
  /// In tr, this message translates to:
  /// **'{seconds} sn içinde hazır ol'**
  String readyDeadline(int seconds);

  /// Shown after tapping Ready.
  ///
  /// In tr, this message translates to:
  /// **'Rakip bekleniyor…'**
  String get waitingForOpponent;

  /// Big start countdown number.
  ///
  /// In tr, this message translates to:
  /// **'{seconds}'**
  String matchStartsIn(int seconds);

  /// Round counter.
  ///
  /// In tr, this message translates to:
  /// **'Soru {round}/{total}'**
  String roundOf(int round, int total);

  /// Label for the player.
  ///
  /// In tr, this message translates to:
  /// **'Sen'**
  String get you;

  /// Shown when the opponent answered.
  ///
  /// In tr, this message translates to:
  /// **'Rakip cevapladı'**
  String get opponentAnswered;

  /// Reveal: own answer correct.
  ///
  /// In tr, this message translates to:
  /// **'Doğru!'**
  String get answerCorrect;

  /// Reveal: own answer wrong.
  ///
  /// In tr, this message translates to:
  /// **'Yanlış'**
  String get answerWrong;

  /// Reveal: no answer in time.
  ///
  /// In tr, this message translates to:
  /// **'Süre doldu'**
  String get answerTimeout;

  /// Points of a round.
  ///
  /// In tr, this message translates to:
  /// **'+{points} puan'**
  String pointsGained(int points);

  /// Result: win.
  ///
  /// In tr, this message translates to:
  /// **'Kazandın!'**
  String get matchWon;

  /// Result: loss.
  ///
  /// In tr, this message translates to:
  /// **'Kaybettin'**
  String get matchLost;

  /// Result: draw.
  ///
  /// In tr, this message translates to:
  /// **'Berabere'**
  String get matchDraw;

  /// Result: cancelled.
  ///
  /// In tr, this message translates to:
  /// **'Maç iptal edildi'**
  String get matchCancelled;

  /// Result: opponent left / forfeited.
  ///
  /// In tr, this message translates to:
  /// **'Rakibin maçtan ayrıldı'**
  String get opponentLeft;

  /// Return to the home screen.
  ///
  /// In tr, this message translates to:
  /// **'Ana sayfaya dön'**
  String get backHome;

  /// Open the result screen with rounds.
  ///
  /// In tr, this message translates to:
  /// **'Soruları gör'**
  String get showDetails;

  /// Leave confirmation title.
  ///
  /// In tr, this message translates to:
  /// **'Maçtan ayrılmak istiyor musun?'**
  String get leaveMatchTitle;

  /// Leave confirmation body (edge-cases §3).
  ///
  /// In tr, this message translates to:
  /// **'Ayrılırsan maçı kaybetmiş sayılırsın.'**
  String get leaveMatchBody;

  /// Leave button.
  ///
  /// In tr, this message translates to:
  /// **'Ayrıl'**
  String get leaveButton;

  /// Banner while the WebSocket reconnects.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı yeniden kuruluyor…'**
  String get reconnecting;

  /// Banner when the opponent is disconnected.
  ///
  /// In tr, this message translates to:
  /// **'Rakibin bağlantısı koptu'**
  String get opponentDisconnected;

  /// Shown on match.seat_moved.
  ///
  /// In tr, this message translates to:
  /// **'Maç başka bir cihazda açıldı'**
  String get seatMoved;

  /// Shown on match.resumed.
  ///
  /// In tr, this message translates to:
  /// **'Sunucu değişti, soru yenilendi'**
  String get matchResumed;

  /// Result screen title.
  ///
  /// In tr, this message translates to:
  /// **'Maç sonucu'**
  String get resultTitle;

  /// Result screen: correct answer of a round.
  ///
  /// In tr, this message translates to:
  /// **'Doğru cevap: {answer}'**
  String resultCorrectAnswer(String answer);

  /// Home: a live match is running; go back to it.
  ///
  /// In tr, this message translates to:
  /// **'Maça dön'**
  String get resumeMatch;

  /// Home: open the friends screen.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlar'**
  String get homeFriends;

  /// Home: open the challenges screen.
  ///
  /// In tr, this message translates to:
  /// **'Meydan okumalar'**
  String get homeChallenges;

  /// Friends screen tab: friends.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlar'**
  String get friendsTab;

  /// Friends screen tab: friend requests.
  ///
  /// In tr, this message translates to:
  /// **'İstekler'**
  String get requestsTab;

  /// Friends list is empty.
  ///
  /// In tr, this message translates to:
  /// **'Henüz arkadaşın yok. Kullanıcı adı ya da arkadaş koduyla ekleyebilirsin.'**
  String get friendsEmpty;

  /// No friend requests.
  ///
  /// In tr, this message translates to:
  /// **'Bekleyen istek yok.'**
  String get requestsEmpty;

  /// Guests cannot have friends (D-016).
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş eklemek için hesabını bağla.'**
  String get friendsUpgrade;

  /// The caller's friend invite code.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş kodun: {code}'**
  String friendCode(String code);

  /// Snack bar after copying a code.
  ///
  /// In tr, this message translates to:
  /// **'Kopyalandı'**
  String get copied;

  /// Add friend dialog title / button.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş ekle'**
  String get addFriend;

  /// Add friend: by username.
  ///
  /// In tr, this message translates to:
  /// **'Kullanıcı adı'**
  String get addByUsername;

  /// Add friend: by friend code.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş kodu'**
  String get addByCode;

  /// Generic send button.
  ///
  /// In tr, this message translates to:
  /// **'Gönder'**
  String get send;

  /// After sending a friend request.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlık isteği gönderildi.'**
  String get friendRequestSent;

  /// A friend request made a friendship at once.
  ///
  /// In tr, this message translates to:
  /// **'Artık arkadaşsınız!'**
  String get friendAdded;

  /// Friend is online.
  ///
  /// In tr, this message translates to:
  /// **'Çevrimiçi'**
  String get online;

  /// Incoming requests / challenges.
  ///
  /// In tr, this message translates to:
  /// **'Gelen'**
  String get incomingLabel;

  /// Sent requests / challenges.
  ///
  /// In tr, this message translates to:
  /// **'Giden'**
  String get outgoingLabel;

  /// Accept button.
  ///
  /// In tr, this message translates to:
  /// **'Kabul et'**
  String get accept;

  /// Decline button.
  ///
  /// In tr, this message translates to:
  /// **'Reddet'**
  String get decline;

  /// Withdraw an own request or challenge.
  ///
  /// In tr, this message translates to:
  /// **'Geri çek'**
  String get withdraw;

  /// Friend menu: unfriend.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlıktan çıkar'**
  String get unfriend;

  /// Friend menu: block.
  ///
  /// In tr, this message translates to:
  /// **'Engelle'**
  String get blockUser;

  /// Block confirmation.
  ///
  /// In tr, this message translates to:
  /// **'{name} engellensin mi? Arkadaşlığınız ve bekleyen istekler sona erer.'**
  String blockConfirm(String name);

  /// Friend menu: send a challenge.
  ///
  /// In tr, this message translates to:
  /// **'Meydan oku'**
  String get challengeAction;

  /// Challenge dialog title.
  ///
  /// In tr, this message translates to:
  /// **'{name} adlı oyuncuya meydan oku'**
  String challengeTitle(String name);

  /// After sending a challenge.
  ///
  /// In tr, this message translates to:
  /// **'Meydan okuma gönderildi.'**
  String get challengeSent;

  /// Match type name.
  ///
  /// In tr, this message translates to:
  /// **'Hızlı Düello'**
  String get matchTypeQuickDuel;

  /// Match type name.
  ///
  /// In tr, this message translates to:
  /// **'Ani Ölüm'**
  String get matchTypeSuddenDeath;

  /// Match type name.
  ///
  /// In tr, this message translates to:
  /// **'Maraton'**
  String get matchTypeMarathon;

  /// Challenges screen title.
  ///
  /// In tr, this message translates to:
  /// **'Meydan okumalar'**
  String get challengesTitle;

  /// No challenges.
  ///
  /// In tr, this message translates to:
  /// **'Bekleyen meydan okuma yok.'**
  String get challengesEmpty;

  /// Incoming challenge line.
  ///
  /// In tr, this message translates to:
  /// **'{name} sana meydan okudu'**
  String challengeFrom(String name);

  /// Outgoing challenge line.
  ///
  /// In tr, this message translates to:
  /// **'{name} adlı oyuncuya'**
  String challengeTo(String name);

  /// Snack bar for challenge.received.
  ///
  /// In tr, this message translates to:
  /// **'{name} sana meydan okudu!'**
  String challengeReceived(String name);

  /// Snack bar when the target accepted.
  ///
  /// In tr, this message translates to:
  /// **'Meydan okuman kabul edildi!'**
  String get challengeAccepted;

  /// Snack bar action: open the screen.
  ///
  /// In tr, this message translates to:
  /// **'Gör'**
  String get view;

  /// Snack bar action: go to the match.
  ///
  /// In tr, this message translates to:
  /// **'Oyna'**
  String get playNow;

  /// Match end / result screen: offer or accept "Play again" (D-091).
  ///
  /// In tr, this message translates to:
  /// **'Tekrar Oyna'**
  String get rematchButton;

  /// Play again offered, waiting for the others.
  ///
  /// In tr, this message translates to:
  /// **'Diğer oyuncular bekleniyor…'**
  String get rematchWaiting;

  /// Another player offered to play again.
  ///
  /// In tr, this message translates to:
  /// **'Tekrar oynama teklifi geldi!'**
  String get rematchOffered;

  /// Play again declined or failed.
  ///
  /// In tr, this message translates to:
  /// **'Bu sefer tekrar oynanmayacak.'**
  String get rematchDeclined;

  /// Home: open the mode picker.
  ///
  /// In tr, this message translates to:
  /// **'Diğer modlar'**
  String get playModes;

  /// Mode picker title.
  ///
  /// In tr, this message translates to:
  /// **'Mod seç'**
  String get modesTitle;

  /// Match type name.
  ///
  /// In tr, this message translates to:
  /// **'Dereceli'**
  String get matchTypeRanked;

  /// Mode picker: category mode.
  ///
  /// In tr, this message translates to:
  /// **'Kategori modu'**
  String get categoryModeLabel;

  /// No category mode (mixed questions).
  ///
  /// In tr, this message translates to:
  /// **'Standart'**
  String get categoryModeNone;

  /// Category mode name.
  ///
  /// In tr, this message translates to:
  /// **'Kör Meydan Okuma'**
  String get categoryModeBlind;

  /// Category mode name.
  ///
  /// In tr, this message translates to:
  /// **'Kategori Ustası'**
  String get categoryModeMaster;

  /// Category mode name.
  ///
  /// In tr, this message translates to:
  /// **'Karışık Savaş'**
  String get categoryModeMixed;

  /// Category Master: the category.
  ///
  /// In tr, this message translates to:
  /// **'Kategori'**
  String get categoryLabel;

  /// Mode picker: join the queue.
  ///
  /// In tr, this message translates to:
  /// **'Rakip ara'**
  String get startSearch;

  /// Mixed Battle: the player picks.
  ///
  /// In tr, this message translates to:
  /// **'Bir kategori seç'**
  String get pickCategory;

  /// Mixed Battle: the other player picks.
  ///
  /// In tr, this message translates to:
  /// **'{name} kategori seçiyor…'**
  String opponentPicking(String name);

  /// Private room button / title.
  ///
  /// In tr, this message translates to:
  /// **'Özel oda'**
  String get privateRoom;

  /// Create a private room.
  ///
  /// In tr, this message translates to:
  /// **'Oda kur'**
  String get createRoom;

  /// Join a room by code.
  ///
  /// In tr, this message translates to:
  /// **'Kodla katıl'**
  String get joinRoom;

  /// Room code field.
  ///
  /// In tr, this message translates to:
  /// **'Oda kodu'**
  String get roomCodeLabel;

  /// The room's invite code.
  ///
  /// In tr, this message translates to:
  /// **'Oda kodu: {code}'**
  String roomCode(String code);

  /// A side of the room.
  ///
  /// In tr, this message translates to:
  /// **'Taraf {no}'**
  String sideLabel(int no);

  /// Empty seat in a room.
  ///
  /// In tr, this message translates to:
  /// **'Boş koltuk — buraya geç'**
  String get emptySeat;

  /// Room host.
  ///
  /// In tr, this message translates to:
  /// **'Kurucu'**
  String get host;

  /// Room: ready.
  ///
  /// In tr, this message translates to:
  /// **'Hazır'**
  String get readyLabel;

  /// Room: not ready.
  ///
  /// In tr, this message translates to:
  /// **'Hazır değil'**
  String get notReady;

  /// Room host: start.
  ///
  /// In tr, this message translates to:
  /// **'Maçı başlat'**
  String get startMatch;

  /// Room is not full yet.
  ///
  /// In tr, this message translates to:
  /// **'Oyuncu bekleniyor…'**
  String get roomWaitingPlayers;

  /// Leave a room.
  ///
  /// In tr, this message translates to:
  /// **'Odadan çık'**
  String get leaveRoom;

  /// The room is gone.
  ///
  /// In tr, this message translates to:
  /// **'Oda kapandı.'**
  String get roomClosed;

  /// Result screen: back to the private room.
  ///
  /// In tr, this message translates to:
  /// **'Odaya dön'**
  String get backToRoom;

  /// Home: teams and couple.
  ///
  /// In tr, this message translates to:
  /// **'Takımlar'**
  String get homeTeams;

  /// Guests cannot have teams (D-016a).
  ///
  /// In tr, this message translates to:
  /// **'Takım kurmak ve çift olmak için hesabını bağla.'**
  String get teamsUpgrade;

  /// Teams screen: couple section.
  ///
  /// In tr, this message translates to:
  /// **'Çift profili'**
  String get coupleSection;

  /// No couple.
  ///
  /// In tr, this message translates to:
  /// **'Bir çift profilin yok.'**
  String get noCouple;

  /// Ask a friend to form a couple.
  ///
  /// In tr, this message translates to:
  /// **'Çift ol'**
  String get requestCouple;

  /// Couple without a name.
  ///
  /// In tr, this message translates to:
  /// **'{name} ile çift'**
  String coupleWith(String name);

  /// Couple request pending.
  ///
  /// In tr, this message translates to:
  /// **'Onay bekleniyor'**
  String get couplePending;

  /// Couple is active.
  ///
  /// In tr, this message translates to:
  /// **'Aktif'**
  String get coupleActive;

  /// Dissolve the couple.
  ///
  /// In tr, this message translates to:
  /// **'Çifti boz'**
  String get dissolveCouple;

  /// Teams screen: invites.
  ///
  /// In tr, this message translates to:
  /// **'Takım davetleri'**
  String get teamInvites;

  /// Teams screen: own teams.
  ///
  /// In tr, this message translates to:
  /// **'Takımlarım'**
  String get myTeams;

  /// No teams.
  ///
  /// In tr, this message translates to:
  /// **'Henüz bir takımın yok.'**
  String get teamsEmpty;

  /// Create a team.
  ///
  /// In tr, this message translates to:
  /// **'Takım kur'**
  String get createTeam;

  /// Team name field.
  ///
  /// In tr, this message translates to:
  /// **'Takım adı'**
  String get teamName;

  /// Team tag field.
  ///
  /// In tr, this message translates to:
  /// **'Kısaltma (2–4)'**
  String get teamTag;

  /// Team captain.
  ///
  /// In tr, this message translates to:
  /// **'Kaptan'**
  String get captain;

  /// Hand over the captaincy.
  ///
  /// In tr, this message translates to:
  /// **'Kaptan yap'**
  String get makeCaptain;

  /// Kick a member.
  ///
  /// In tr, this message translates to:
  /// **'Takımdan çıkar'**
  String get kick;

  /// Invite a friend to the team.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş davet et'**
  String get inviteToTeam;

  /// Team invite sent.
  ///
  /// In tr, this message translates to:
  /// **'Davet gönderildi.'**
  String get teamInviteSent;

  /// Leave a team.
  ///
  /// In tr, this message translates to:
  /// **'Takımdan ayrıl'**
  String get leaveTeam;

  /// Disband a team.
  ///
  /// In tr, this message translates to:
  /// **'Takımı dağıt'**
  String get disbandTeam;

  /// Disband confirmation.
  ///
  /// In tr, this message translates to:
  /// **'Takım dağıtılsın mı? Bu geri alınamaz.'**
  String get disbandConfirm;

  /// Pick a friend dialog.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaş seç'**
  String get pickFriend;

  /// Team scoreboard: own side.
  ///
  /// In tr, this message translates to:
  /// **'Takımın'**
  String get yourTeam;

  /// Team scoreboard: other side.
  ///
  /// In tr, this message translates to:
  /// **'Rakipler'**
  String get opponentsTeam;

  /// Shared Answer: lock the side's answer.
  ///
  /// In tr, this message translates to:
  /// **'Cevabı kilitle'**
  String get lockAnswer;

  /// Individual Turns: the player answers.
  ///
  /// In tr, this message translates to:
  /// **'Sıra sende'**
  String get yourTurn;

  /// Individual Turns: a teammate answers.
  ///
  /// In tr, this message translates to:
  /// **'Sıra: {name}'**
  String turnOf(String name);

  /// Picker: format.
  ///
  /// In tr, this message translates to:
  /// **'Format'**
  String get formatLabel;

  /// Couple format name.
  ///
  /// In tr, this message translates to:
  /// **'Çift'**
  String get formatCouple;

  /// Picker: team answer mode.
  ///
  /// In tr, this message translates to:
  /// **'Takım modu'**
  String get teamModeLabel;

  /// Team answer mode name.
  ///
  /// In tr, this message translates to:
  /// **'Ortak Cevap'**
  String get teamModeShared;

  /// Team answer mode name.
  ///
  /// In tr, this message translates to:
  /// **'Sırayla'**
  String get teamModeTurns;

  /// Team answer mode name.
  ///
  /// In tr, this message translates to:
  /// **'Hızlı Takım Turu'**
  String get teamModeFast;

  /// Queue: friends playing on the player's side.
  ///
  /// In tr, this message translates to:
  /// **'Grubun'**
  String get partyLabel;

  /// Queue: choose party members.
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlarını seç'**
  String get pickParty;

  /// Queue: play as a persistent team.
  ///
  /// In tr, this message translates to:
  /// **'Takımımla oyna'**
  String get playAsTeam;

  /// Queue: no persistent team.
  ///
  /// In tr, this message translates to:
  /// **'Takımsız'**
  String get noTeam;

  /// Leaderboards screen title
  ///
  /// In tr, this message translates to:
  /// **'Liderlik Tablosu'**
  String get leaderboardsTitle;

  /// Quests screen title
  ///
  /// In tr, this message translates to:
  /// **'Görevler'**
  String get questsTitle;

  /// Badges screen title
  ///
  /// In tr, this message translates to:
  /// **'Rozetler'**
  String get badgesTitle;

  /// Board period
  ///
  /// In tr, this message translates to:
  /// **'Günlük'**
  String get periodDaily;

  /// Board period
  ///
  /// In tr, this message translates to:
  /// **'Haftalık'**
  String get periodWeekly;

  /// Board period
  ///
  /// In tr, this message translates to:
  /// **'Aylık'**
  String get periodMonthly;

  /// Board period
  ///
  /// In tr, this message translates to:
  /// **'Tüm zamanlar'**
  String get periodAllTime;

  /// Board period (Season Points)
  ///
  /// In tr, this message translates to:
  /// **'Sezon'**
  String get periodSeason;

  /// Board entity
  ///
  /// In tr, this message translates to:
  /// **'Bireysel'**
  String get entityUser;

  /// Board entity
  ///
  /// In tr, this message translates to:
  /// **'Takımlar'**
  String get entityTeam;

  /// Board entity
  ///
  /// In tr, this message translates to:
  /// **'Çiftler'**
  String get entityCouple;

  /// Board scope
  ///
  /// In tr, this message translates to:
  /// **'Dünya'**
  String get scopeGlobal;

  /// Board scope
  ///
  /// In tr, this message translates to:
  /// **'Ülke'**
  String get scopeCountry;

  /// Board scope
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlar'**
  String get scopeFriends;

  /// Empty leaderboard
  ///
  /// In tr, this message translates to:
  /// **'Burada henüz kimse yok.'**
  String get leaderboardEmpty;

  /// The caller's own row
  ///
  /// In tr, this message translates to:
  /// **'Sen'**
  String get leaderboardMe;

  /// Leaderboard score
  ///
  /// In tr, this message translates to:
  /// **'{count} puan'**
  String boardPoints(int count);

  /// Player level
  ///
  /// In tr, this message translates to:
  /// **'Seviye {level}'**
  String levelLabel(int level);

  /// XP inside the level
  ///
  /// In tr, this message translates to:
  /// **'{current} / {total} XP'**
  String xpProgress(int current, int total);

  /// Play streak
  ///
  /// In tr, this message translates to:
  /// **'{days} gün seri'**
  String streakDays(int days);

  /// Season Points
  ///
  /// In tr, this message translates to:
  /// **'{sp} SP'**
  String seasonPoints(int sp);

  /// League
  ///
  /// In tr, this message translates to:
  /// **'Bronz'**
  String get leagueBronze;

  /// League
  ///
  /// In tr, this message translates to:
  /// **'Gümüş'**
  String get leagueSilver;

  /// League
  ///
  /// In tr, this message translates to:
  /// **'Altın'**
  String get leagueGold;

  /// League
  ///
  /// In tr, this message translates to:
  /// **'Platin'**
  String get leaguePlatinum;

  /// League
  ///
  /// In tr, this message translates to:
  /// **'Elmas'**
  String get leagueDiamond;

  /// SP of the next division
  ///
  /// In tr, this message translates to:
  /// **'Sonraki kademe: {sp} SP'**
  String nextDivisionAt(int sp);

  /// Quest section
  ///
  /// In tr, this message translates to:
  /// **'Günlük görevler'**
  String get questsDaily;

  /// Quest section
  ///
  /// In tr, this message translates to:
  /// **'Haftalık görevler'**
  String get questsWeekly;

  /// Claim a quest reward
  ///
  /// In tr, this message translates to:
  /// **'Ödülü al'**
  String get questClaim;

  /// Claimed quest
  ///
  /// In tr, this message translates to:
  /// **'Alındı'**
  String get questClaimed;

  /// Swap a quest
  ///
  /// In tr, this message translates to:
  /// **'Değiştir'**
  String get questReroll;

  /// Rerolls
  ///
  /// In tr, this message translates to:
  /// **'Bugün kalan değiştirme: {count}'**
  String questRerollsLeft(int count);

  /// After claiming
  ///
  /// In tr, this message translates to:
  /// **'+{xp} XP kazandın'**
  String questXpGained(int xp);

  /// Quest reward
  ///
  /// In tr, this message translates to:
  /// **'{xp} XP'**
  String questReward(int xp);

  /// Locked badge
  ///
  /// In tr, this message translates to:
  /// **'Henüz kazanılmadı'**
  String get badgeLocked;

  /// Match rewards section
  ///
  /// In tr, this message translates to:
  /// **'Ödüller'**
  String get awardsTitle;

  /// Waiting for match.awards
  ///
  /// In tr, this message translates to:
  /// **'Ödüller hesaplanıyor…'**
  String get awardsPending;

  /// XP from the match
  ///
  /// In tr, this message translates to:
  /// **'+{xp} XP'**
  String awardsXp(int xp);

  /// Level up
  ///
  /// In tr, this message translates to:
  /// **'Seviye atladın: {level}!'**
  String awardsLevelUp(int level);

  /// Signed SP change
  ///
  /// In tr, this message translates to:
  /// **'{sp} SP'**
  String awardsSp(String sp);

  /// League promotion
  ///
  /// In tr, this message translates to:
  /// **'Yeni lig: {league}!'**
  String awardsLeagueUp(String league);

  /// Badge earned
  ///
  /// In tr, this message translates to:
  /// **'Yeni rozet: {name}'**
  String awardsBadge(String name);

  /// Quests completed by the match
  ///
  /// In tr, this message translates to:
  /// **'{count} görev tamamlandı'**
  String awardsQuests(int count);

  /// Revenge challenge label
  ///
  /// In tr, this message translates to:
  /// **'Rövanş'**
  String get revengeLabel;

  /// Daily Challenge title
  ///
  /// In tr, this message translates to:
  /// **'Günün Meydan Okuması'**
  String get dailyTitle;

  /// Daily Challenge explanation
  ///
  /// In tr, this message translates to:
  /// **'Herkes aynı 10 soruyu cevaplar. Günde bir resmi deneme hakkın var; sonrası pratik.'**
  String get dailyIntro;

  /// Start the official run
  ///
  /// In tr, this message translates to:
  /// **'Başla'**
  String get dailyStart;

  /// Resume a running daily run
  ///
  /// In tr, this message translates to:
  /// **'Devam et'**
  String get dailyResume;

  /// Start a practice run
  ///
  /// In tr, this message translates to:
  /// **'Pratik yap'**
  String get dailyPractice;

  /// No daily set
  ///
  /// In tr, this message translates to:
  /// **'Bugünün soruları henüz hazır değil.'**
  String get dailyNotAvailable;

  /// Daily result
  ///
  /// In tr, this message translates to:
  /// **'{correct}/10 doğru · {seconds} sn'**
  String dailyResult(int correct, String seconds);

  /// Place on the daily board
  ///
  /// In tr, this message translates to:
  /// **'Sıralaman: #{rank}'**
  String dailyRank(int rank);

  /// Daily board section
  ///
  /// In tr, this message translates to:
  /// **'Günün tablosu'**
  String get dailyBoardTitle;

  /// Daily question counter
  ///
  /// In tr, this message translates to:
  /// **'Soru {position}/{total}'**
  String dailyQuestionOf(int position, int total);

  /// Practice run marker
  ///
  /// In tr, this message translates to:
  /// **'Pratik'**
  String get dailyPracticeLabel;

  /// Daily run finished
  ///
  /// In tr, this message translates to:
  /// **'Tamamlandı!'**
  String get dailyDone;

  /// Back button
  ///
  /// In tr, this message translates to:
  /// **'Günün meydan okumasına dön'**
  String get dailyBack;

  /// Voided run
  ///
  /// In tr, this message translates to:
  /// **'Bu deneme sayılmadı.'**
  String get dailyVoided;

  /// Daily reveal
  ///
  /// In tr, this message translates to:
  /// **'Doğru!'**
  String get dailyCorrect;

  /// Daily reveal
  ///
  /// In tr, this message translates to:
  /// **'Yanlış'**
  String get dailyWrong;

  /// Daily reveal
  ///
  /// In tr, this message translates to:
  /// **'Süre doldu'**
  String get dailyTimeout;

  /// Report a question
  ///
  /// In tr, this message translates to:
  /// **'Soruyu bildir'**
  String get reportQuestion;

  /// Report reason dialog
  ///
  /// In tr, this message translates to:
  /// **'Sorun ne?'**
  String get reportReasonTitle;

  /// Report reason
  ///
  /// In tr, this message translates to:
  /// **'Cevap yanlış'**
  String get reportReasonWrongAnswer;

  /// Report reason
  ///
  /// In tr, this message translates to:
  /// **'Belirsiz / birden fazla doğru'**
  String get reportReasonAmbiguous;

  /// Report reason
  ///
  /// In tr, this message translates to:
  /// **'Bilgi güncel değil'**
  String get reportReasonOutdated;

  /// Report reason
  ///
  /// In tr, this message translates to:
  /// **'Yazım veya çeviri hatası'**
  String get reportReasonTypo;

  /// Report reason
  ///
  /// In tr, this message translates to:
  /// **'Uygunsuz içerik'**
  String get reportReasonOffensive;

  /// Report reason
  ///
  /// In tr, this message translates to:
  /// **'Diğer'**
  String get reportReasonOther;

  /// Report comment
  ///
  /// In tr, this message translates to:
  /// **'Not (isteğe bağlı)'**
  String get reportComment;

  /// Send report
  ///
  /// In tr, this message translates to:
  /// **'Gönder'**
  String get reportSend;

  /// Report sent
  ///
  /// In tr, this message translates to:
  /// **'Bildirimin için teşekkürler.'**
  String get reportSent;

  /// Sanctions screen
  ///
  /// In tr, this message translates to:
  /// **'Yaptırımlarım'**
  String get sanctionsTitle;

  /// No sanctions
  ///
  /// In tr, this message translates to:
  /// **'Hesabında yaptırım yok.'**
  String get sanctionsEmpty;

  /// Sanction kind
  ///
  /// In tr, this message translates to:
  /// **'Uyarı'**
  String get sanctionWarning;

  /// Sanction kind
  ///
  /// In tr, this message translates to:
  /// **'Sohbet engeli'**
  String get sanctionChatMute;

  /// Sanction kind
  ///
  /// In tr, this message translates to:
  /// **'Dereceli yasağı'**
  String get sanctionRankedBan;

  /// Sanction kind
  ///
  /// In tr, this message translates to:
  /// **'Eşleştirme yasağı'**
  String get sanctionMatchmakingBan;

  /// Sanction kind
  ///
  /// In tr, this message translates to:
  /// **'Sosyal yasak'**
  String get sanctionSocialBan;

  /// Sanction kind
  ///
  /// In tr, this message translates to:
  /// **'Hesap askıya alma'**
  String get sanctionSuspension;

  /// Sanction end
  ///
  /// In tr, this message translates to:
  /// **'{date} tarihine kadar'**
  String sanctionUntil(String date);

  /// Permanent sanction
  ///
  /// In tr, this message translates to:
  /// **'Süresiz'**
  String get sanctionPermanent;

  /// Inactive sanction
  ///
  /// In tr, this message translates to:
  /// **'Sona erdi'**
  String get sanctionEnded;

  /// Appeal button
  ///
  /// In tr, this message translates to:
  /// **'İtiraz et'**
  String get appealButton;

  /// Appeal text hint
  ///
  /// In tr, this message translates to:
  /// **'Neden haksız olduğunu yaz (en az 10 karakter)'**
  String get appealHint;

  /// Appeal sent
  ///
  /// In tr, this message translates to:
  /// **'İtirazın alındı.'**
  String get appealSent;

  /// Appeal status
  ///
  /// In tr, this message translates to:
  /// **'İtiraz inceleniyor'**
  String get appealPending;

  /// Appeal status
  ///
  /// In tr, this message translates to:
  /// **'İtiraz kabul edildi'**
  String get appealAccepted;

  /// Appeal status
  ///
  /// In tr, this message translates to:
  /// **'İtiraz reddedildi'**
  String get appealRejected;

  /// Creator Quiz screen title
  ///
  /// In tr, this message translates to:
  /// **'Quizler'**
  String get quizzesTitle;

  /// Tab with the caller's quizzes
  ///
  /// In tr, this message translates to:
  /// **'Quizlerim'**
  String get quizzesMine;

  /// Tab with published quizzes
  ///
  /// In tr, this message translates to:
  /// **'Keşfet'**
  String get quizzesDiscover;

  /// Empty my quizzes
  ///
  /// In tr, this message translates to:
  /// **'Henüz bir quiz yazmadın.'**
  String get quizzesEmptyMine;

  /// Empty discovery
  ///
  /// In tr, this message translates to:
  /// **'Yayında quiz yok.'**
  String get quizzesEmptyDiscover;

  /// Create quiz button
  ///
  /// In tr, this message translates to:
  /// **'Yeni quiz'**
  String get quizNew;

  /// Editor title for an existing quiz
  ///
  /// In tr, this message translates to:
  /// **'Quizi düzenle'**
  String get quizEdit;

  /// Quiz title field
  ///
  /// In tr, this message translates to:
  /// **'Başlık'**
  String get quizTitleField;

  /// Quiz description field
  ///
  /// In tr, this message translates to:
  /// **'Açıklama (isteğe bağlı)'**
  String get quizDescriptionField;

  /// Quiz language label
  ///
  /// In tr, this message translates to:
  /// **'Dil'**
  String get quizLanguage;

  /// Quiz category label
  ///
  /// In tr, this message translates to:
  /// **'Kategori'**
  String get quizCategory;

  /// Answer window label
  ///
  /// In tr, this message translates to:
  /// **'Cevap süresi'**
  String get quizWindow;

  /// Seconds
  ///
  /// In tr, this message translates to:
  /// **'{seconds} sn'**
  String quizSeconds(int seconds);

  /// Question heading
  ///
  /// In tr, this message translates to:
  /// **'Soru {n}'**
  String quizQuestionN(int n);

  /// Prompt field
  ///
  /// In tr, this message translates to:
  /// **'Soru metni'**
  String get quizPrompt;

  /// Option field
  ///
  /// In tr, this message translates to:
  /// **'Seçenek {n}'**
  String quizOptionN(int n);

  /// True/false toggle
  ///
  /// In tr, this message translates to:
  /// **'Doğru / Yanlış'**
  String get quizTrueFalse;

  /// True option
  ///
  /// In tr, this message translates to:
  /// **'Doğru'**
  String get quizTrue;

  /// False option
  ///
  /// In tr, this message translates to:
  /// **'Yanlış'**
  String get quizFalse;

  /// Correct option hint
  ///
  /// In tr, this message translates to:
  /// **'Doğru cevabı daire ile işaretle.'**
  String get quizCorrectHint;

  /// Add question button
  ///
  /// In tr, this message translates to:
  /// **'Soru ekle'**
  String get quizAddQuestion;

  /// Remove question tooltip
  ///
  /// In tr, this message translates to:
  /// **'Soruyu sil'**
  String get quizRemoveQuestion;

  /// Save quiz
  ///
  /// In tr, this message translates to:
  /// **'Kaydet'**
  String get quizSave;

  /// Saved snack
  ///
  /// In tr, this message translates to:
  /// **'Quiz kaydedildi.'**
  String get quizSaved;

  /// Client validation
  ///
  /// In tr, this message translates to:
  /// **'Başlık 3–60 karakter olmalı ve her soru ile seçenek dolu olmalı.'**
  String get quizInvalid;

  /// Question count rule
  ///
  /// In tr, this message translates to:
  /// **'{min}–{max} soru gerekli.'**
  String quizCountRule(int min, int max);

  /// Status
  ///
  /// In tr, this message translates to:
  /// **'Taslak'**
  String get quizStatusDraft;

  /// Status
  ///
  /// In tr, this message translates to:
  /// **'İncelemede'**
  String get quizStatusSubmitted;

  /// Status
  ///
  /// In tr, this message translates to:
  /// **'Yayında'**
  String get quizStatusPublished;

  /// Status
  ///
  /// In tr, this message translates to:
  /// **'Reddedildi'**
  String get quizStatusRejected;

  /// Status
  ///
  /// In tr, this message translates to:
  /// **'Yayından kaldırıldı'**
  String get quizStatusUnpublished;

  /// Submit action
  ///
  /// In tr, this message translates to:
  /// **'İncelemeye gönder'**
  String get quizSubmit;

  /// Submitted snack
  ///
  /// In tr, this message translates to:
  /// **'Quiz incelemeye gönderildi.'**
  String get quizSubmitted;

  /// Delete action
  ///
  /// In tr, this message translates to:
  /// **'Sil'**
  String get quizDelete;

  /// Delete confirmation
  ///
  /// In tr, this message translates to:
  /// **'Bu quiz silinsin mi?'**
  String get quizDeleteConfirm;

  /// Create a room with the quiz
  ///
  /// In tr, this message translates to:
  /// **'Odada oyna'**
  String get quizPlayInRoom;

  /// Quiz list subtitle
  ///
  /// In tr, this message translates to:
  /// **'{count} soru · {author}'**
  String quizMeta(int count, String author);

  /// Quiz stats
  ///
  /// In tr, this message translates to:
  /// **'{count} beğeni · {plays} oyun'**
  String quizLikes(int count, int plays);

  /// Like tooltip
  ///
  /// In tr, this message translates to:
  /// **'Beğen'**
  String get quizLike;

  /// Rejection reason
  ///
  /// In tr, this message translates to:
  /// **'Red sebebi: {reason}'**
  String quizRejectedReason(String reason);

  /// Unpublished play note
  ///
  /// In tr, this message translates to:
  /// **'Yayında olmayan quiz yalnızca arkadaşlarınla oynanır ve ödül vermez.'**
  String get quizUnpublishedNote;

  /// Room subtitle for a quiz room
  ///
  /// In tr, this message translates to:
  /// **'Quiz · {count} soru'**
  String roomQuiz(int count);

  /// Friends-only room note
  ///
  /// In tr, this message translates to:
  /// **'Yalnızca ev sahibinin arkadaşları katılabilir.'**
  String get roomFriendsOnly;

  /// Home button to quizzes
  ///
  /// In tr, this message translates to:
  /// **'Quizler'**
  String get homeQuizzes;

  /// Theme section title
  ///
  /// In tr, this message translates to:
  /// **'Görünüm'**
  String get settingsTheme;

  /// Follow the device theme
  ///
  /// In tr, this message translates to:
  /// **'Sistem'**
  String get themeSystem;

  /// Light theme
  ///
  /// In tr, this message translates to:
  /// **'Açık'**
  String get themeLight;

  /// Dark theme
  ///
  /// In tr, this message translates to:
  /// **'Koyu'**
  String get themeDark;

  /// Quick Duel hero subtitle
  ///
  /// In tr, this message translates to:
  /// **'Sıradaki rakibini bul'**
  String get homeQuickDuelSub;

  /// More modes tile subtitle
  ///
  /// In tr, this message translates to:
  /// **'Tüm modları keşfet'**
  String get homeModesSub;

  /// Private room tile subtitle
  ///
  /// In tr, this message translates to:
  /// **'Oda kur ya da katıl'**
  String get homeRoomSub;

  /// Teams tile subtitle
  ///
  /// In tr, this message translates to:
  /// **'Birlikte oyna, kazan'**
  String get homeTeamsSub;

  /// Daily tile subtitle
  ///
  /// In tr, this message translates to:
  /// **'Günün 10 sorusu'**
  String get homeDailySub;

  /// Quizzes tile subtitle
  ///
  /// In tr, this message translates to:
  /// **'Oyuncuların quizleri'**
  String get homeQuizzesSub;

  /// Practice row subtitle
  ///
  /// In tr, this message translates to:
  /// **'Tek başına pratik yap'**
  String get homePracticeSub;

  /// Stats label
  ///
  /// In tr, this message translates to:
  /// **'Toplam galibiyet'**
  String get homeTotalWins;

  /// Stats label
  ///
  /// In tr, this message translates to:
  /// **'Mevcut lig'**
  String get homeCurrentRank;

  /// No league yet
  ///
  /// In tr, this message translates to:
  /// **'Sıralamasız'**
  String get homeNoRank;

  /// Room sheet title
  ///
  /// In tr, this message translates to:
  /// **'Özel Oda'**
  String get roomSheetTitle;

  /// Room sheet subtitle
  ///
  /// In tr, this message translates to:
  /// **'Nasıl oynamak istediğini seç'**
  String get roomSheetSub;

  /// Room option subtitle
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlarınla klasik maç'**
  String get roomQuickDuelSub;

  /// Room option subtitle
  ///
  /// In tr, this message translates to:
  /// **'Tek hata her şeyi değiştirir'**
  String get roomSuddenDeathSub;

  /// Room option subtitle
  ///
  /// In tr, this message translates to:
  /// **'Uzun soluklu yarış'**
  String get roomMarathonSub;

  /// Join section title
  ///
  /// In tr, this message translates to:
  /// **'Kodla katıl'**
  String get roomJoinWithCode;

  /// Code field hint
  ///
  /// In tr, this message translates to:
  /// **'Oda kodunu gir'**
  String get roomCodeHint;

  /// Join button
  ///
  /// In tr, this message translates to:
  /// **'Katıl'**
  String get roomJoinAction;

  /// Mode picker section
  ///
  /// In tr, this message translates to:
  /// **'Oyun modu'**
  String get gameModeLabel;

  /// Queue subtitle
  ///
  /// In tr, this message translates to:
  /// **'Sıradaki meydan okuman bulunuyor'**
  String get queueFinding;

  /// Inbox title
  ///
  /// In tr, this message translates to:
  /// **'Bildirimler'**
  String get inboxTitle;

  /// Empty inbox
  ///
  /// In tr, this message translates to:
  /// **'Henüz bildirimin yok.'**
  String get inboxEmpty;

  /// Mark all read action
  ///
  /// In tr, this message translates to:
  /// **'Tümünü okundu say'**
  String get inboxMarkAll;

  /// Settings screen title
  ///
  /// In tr, this message translates to:
  /// **'Bildirim ayarları'**
  String get notificationSettingsTitle;

  /// Push channel label
  ///
  /// In tr, this message translates to:
  /// **'Anlık'**
  String get notifPush;

  /// In-app channel label
  ///
  /// In tr, this message translates to:
  /// **'Uygulama içi'**
  String get notifInApp;

  /// Quiet hours title
  ///
  /// In tr, this message translates to:
  /// **'Sessiz saatler'**
  String get notifQuietHours;

  /// Quiet hours explanation
  ///
  /// In tr, this message translates to:
  /// **'Bu saatlerde anlık bildirim gelmez; bildirimler kutuda bekler.'**
  String get notifQuietHoursHint;

  /// Quiet start
  ///
  /// In tr, this message translates to:
  /// **'Başlangıç'**
  String get notifQuietStart;

  /// Quiet end
  ///
  /// In tr, this message translates to:
  /// **'Bitiş'**
  String get notifQuietEnd;

  /// Locked category note
  ///
  /// In tr, this message translates to:
  /// **'Hesap bildirimleri kapatılamaz.'**
  String get notifLocked;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Meydan okumalar'**
  String get notifCatChallenge;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Arkadaşlar'**
  String get notifCatFriend;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Maç bulundu'**
  String get notifCatMatchFound;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Takım ve çift'**
  String get notifCatTeam;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Sohbet'**
  String get notifCatChat;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Sezon'**
  String get notifCatSeason;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Görevler'**
  String get notifCatQuest;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Turnuvalar'**
  String get notifCatTournament;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Hesap bildirimleri'**
  String get notifCatModeration;

  /// Category
  ///
  /// In tr, this message translates to:
  /// **'Haberler ve kampanyalar'**
  String get notifCatProductNews;

  /// Settings tile
  ///
  /// In tr, this message translates to:
  /// **'Bildirimler'**
  String get settingsNotifications;

  /// Share button
  ///
  /// In tr, this message translates to:
  /// **'Paylaş'**
  String get shareAction;

  /// Share tooltip
  ///
  /// In tr, this message translates to:
  /// **'Sonucu paylaş'**
  String get shareResult;

  /// Share text for a match
  ///
  /// In tr, this message translates to:
  /// **'BrainVolt\'ta {score}: {result} Sıra sende!'**
  String shareMatchText(String score, String result);

  /// Share text for a daily run
  ///
  /// In tr, this message translates to:
  /// **'BrainVolt Günün Meydan Okuması: {correct}/10 doğru. Sen kaç yaparsın?'**
  String shareDailyText(int correct);

  /// Correct answers line
  ///
  /// In tr, this message translates to:
  /// **'{correct}/{total} doğru'**
  String shareCorrectOf(int correct, int total);

  /// Chat screen title
  ///
  /// In tr, this message translates to:
  /// **'Sohbet'**
  String get chatTitle;

  /// Composer hint
  ///
  /// In tr, this message translates to:
  /// **'Mesaj yaz…'**
  String get chatHint;

  /// Send tooltip
  ///
  /// In tr, this message translates to:
  /// **'Gönder'**
  String get chatSend;

  /// Empty chat
  ///
  /// In tr, this message translates to:
  /// **'Henüz mesaj yok. İlk mesajı sen yaz!'**
  String get chatEmpty;

  /// Removed message
  ///
  /// In tr, this message translates to:
  /// **'Bu mesaj silindi.'**
  String get chatRemoved;

  /// Typing indicator
  ///
  /// In tr, this message translates to:
  /// **'{name} yazıyor…'**
  String chatTyping(String name);

  /// Delete action
  ///
  /// In tr, this message translates to:
  /// **'Mesajı sil'**
  String get chatDelete;

  /// Open chat tooltip
  ///
  /// In tr, this message translates to:
  /// **'Sohbeti aç'**
  String get chatOpen;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'Bol şans!'**
  String get chatQuickGoodLuck;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'İyi oyundu!'**
  String get chatQuickWellPlayed;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'Harika!'**
  String get chatQuickNice;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'Teşekkürler!'**
  String get chatQuickThanks;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'Hay aksi!'**
  String get chatQuickOops;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'Hadi başlayalım!'**
  String get chatQuickLetsGo;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'GG'**
  String get chatQuickGg;

  /// Quick phrase
  ///
  /// In tr, this message translates to:
  /// **'Tekrar oynayalım mı?'**
  String get chatQuickRematch;

  /// Tournaments screen title
  ///
  /// In tr, this message translates to:
  /// **'Turnuvalar'**
  String get tournamentsTitle;

  /// Home tile subtitle
  ///
  /// In tr, this message translates to:
  /// **'Kupayı kazan'**
  String get homeTournamentsSub;

  /// Tournament status
  ///
  /// In tr, this message translates to:
  /// **'Kayıt açık'**
  String get tournamentStatusRegistration;

  /// Tournament status
  ///
  /// In tr, this message translates to:
  /// **'Devam ediyor'**
  String get tournamentStatusRunning;

  /// Tournament status
  ///
  /// In tr, this message translates to:
  /// **'Bitti'**
  String get tournamentStatusFinished;

  /// Tournament status
  ///
  /// In tr, this message translates to:
  /// **'İptal edildi'**
  String get tournamentStatusCancelled;

  /// Tournament status
  ///
  /// In tr, this message translates to:
  /// **'Taslak'**
  String get tournamentStatusDraft;

  /// Entry status
  ///
  /// In tr, this message translates to:
  /// **'Check-in yapıldı'**
  String get tournamentCheckedIn;

  /// Entry status
  ///
  /// In tr, this message translates to:
  /// **'Turnuvada'**
  String get tournamentPlaying;

  /// Final place
  ///
  /// In tr, this message translates to:
  /// **'{place}. oldun'**
  String tournamentPlace(int place);

  /// Entry status: won
  ///
  /// In tr, this message translates to:
  /// **'Şampiyon! 🏆'**
  String get tournamentChampion;

  /// Entry status
  ///
  /// In tr, this message translates to:
  /// **'Kayıtlı'**
  String get tournamentRegistered;

  /// Entry kind
  ///
  /// In tr, this message translates to:
  /// **'Çiftler'**
  String get tournamentKindCouple;

  /// Entry kind with format
  ///
  /// In tr, this message translates to:
  /// **'Takımlar {format}'**
  String tournamentKindTeam(String format);

  /// Entry kind
  ///
  /// In tr, this message translates to:
  /// **'Bireysel'**
  String get tournamentKindSolo;

  /// Admin action
  ///
  /// In tr, this message translates to:
  /// **'Turnuva oluştur'**
  String get tournamentCreate;

  /// Empty list
  ///
  /// In tr, this message translates to:
  /// **'Şu an turnuva yok. Yenileri yakında!'**
  String get tournamentsEmpty;

  /// Entrant count
  ///
  /// In tr, this message translates to:
  /// **'{count}/{max} katılımcı'**
  String tournamentEntrants(int count, int max);

  /// Section title
  ///
  /// In tr, this message translates to:
  /// **'Eşleşme tablosu'**
  String get tournamentBracket;

  /// Button
  ///
  /// In tr, this message translates to:
  /// **'Kayıt ol'**
  String get tournamentRegister;

  /// Info
  ///
  /// In tr, this message translates to:
  /// **'Check-in tamam. Turnuva başlayınca maçın burada görünecek.'**
  String get tournamentCheckedInInfo;

  /// Info
  ///
  /// In tr, this message translates to:
  /// **'Check-in açık! Oynamak için check-in yap.'**
  String get tournamentCheckInNow;

  /// Info
  ///
  /// In tr, this message translates to:
  /// **'Check-in {time} itibarıyla açılır. Check-in yapmayanlar turnuvaya alınmaz.'**
  String tournamentCheckInAt(String time);

  /// Button
  ///
  /// In tr, this message translates to:
  /// **'Check-in yap'**
  String get tournamentCheckIn;

  /// Button
  ///
  /// In tr, this message translates to:
  /// **'Kaydı geri çek'**
  String get tournamentWithdraw;

  /// Current match
  ///
  /// In tr, this message translates to:
  /// **'{round}. tur: rakibin {name}'**
  String tournamentYourMatch(int round, String name);

  /// Info
  ///
  /// In tr, this message translates to:
  /// **'Buradasın. Rakibin bekleniyor…'**
  String get tournamentWaitingOpponent;

  /// Countdown
  ///
  /// In tr, this message translates to:
  /// **'Katılmak için kalan süre: {time}'**
  String tournamentReadyLeft(String time);

  /// Button
  ///
  /// In tr, this message translates to:
  /// **'Maça git'**
  String get tournamentGoToMatch;

  /// Button
  ///
  /// In tr, this message translates to:
  /// **'Oyna'**
  String get tournamentPlay;

  /// Info
  ///
  /// In tr, this message translates to:
  /// **'Sıradaki turu bekliyorsun.'**
  String get tournamentWaitingRound;

  /// Header
  ///
  /// In tr, this message translates to:
  /// **'Başlangıç: {time}'**
  String tournamentStarts(String time);

  /// Reward
  ///
  /// In tr, this message translates to:
  /// **'{place}.: {xp} XP'**
  String tournamentRewardLine(int place, int xp);

  /// Round name
  ///
  /// In tr, this message translates to:
  /// **'Final'**
  String get tournamentFinal;

  /// Round name
  ///
  /// In tr, this message translates to:
  /// **'Yarı final'**
  String get tournamentSemiFinal;

  /// Round name
  ///
  /// In tr, this message translates to:
  /// **'{round}. tur'**
  String tournamentRound(int round);

  /// No opponent
  ///
  /// In tr, this message translates to:
  /// **'Bay'**
  String get tournamentBye;

  /// Slot decided without a match
  ///
  /// In tr, this message translates to:
  /// **'Hükmen'**
  String get tournamentWalkover;

  /// Lineup sheet
  ///
  /// In tr, this message translates to:
  /// **'Takımını ve {size} oyuncuyu seç'**
  String tournamentPickLineup(int size);

  /// Lineup sheet
  ///
  /// In tr, this message translates to:
  /// **'Kaptanı olduğun bir takım yok.'**
  String get tournamentNoCaptainTeam;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Ad (Türkçe)'**
  String get tournamentNameTr;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Ad (İngilizce, isteğe bağlı)'**
  String get tournamentNameEn;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Katılım'**
  String get tournamentKind;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Takım boyutu'**
  String get tournamentFormat;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Maç türü'**
  String get tournamentMatchType;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'En fazla katılımcı'**
  String get tournamentMax;

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Başlamasına'**
  String get tournamentStartIn;

  /// Minutes
  ///
  /// In tr, this message translates to:
  /// **'{minutes} dk'**
  String tournamentMinutes(int minutes);

  /// Form
  ///
  /// In tr, this message translates to:
  /// **'Maça katılma süresi'**
  String get tournamentWindow;

  /// Form hint
  ///
  /// In tr, this message translates to:
  /// **'Kayıt hemen açılır, başlangıçta kapanır. Check-in son 1 saatte. Ödüller: 1. 300 XP, 2. 150 XP, 3. 75 XP, diğerleri 20 XP.'**
  String get tournamentCreateHint;

  /// Snack bar
  ///
  /// In tr, this message translates to:
  /// **'Turnuva maçın başladı!'**
  String get tournamentMatchStarted;

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

  /// Server error code QUESTION_DUPLICATE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu soru zaten var.'**
  String get error_QUESTION_DUPLICATE;

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

  /// Server error code QUESTION_STATE_INVALID (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Sorunun şu anki durumunda bu işlem yapılamaz.'**
  String get error_QUESTION_STATE_INVALID;

  /// Server error code QUEST_ALREADY_CLAIMED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu ödülü zaten aldın.'**
  String get error_QUEST_ALREADY_CLAIMED;

  /// Server error code QUEST_NOT_COMPLETED (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu görev henüz tamamlanmadı.'**
  String get error_QUEST_NOT_COMPLETED;

  /// Server error code QUEST_NOT_REROLLABLE (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bu görev değiştirilemez.'**
  String get error_QUEST_NOT_REROLLABLE;

  /// Server error code QUEST_REROLL_LIMIT (docs/api/errors.md).
  ///
  /// In tr, this message translates to:
  /// **'Bugünkü görev değiştirme hakkın bitti.'**
  String get error_QUEST_REROLL_LIMIT;

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
