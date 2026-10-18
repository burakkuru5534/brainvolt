// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Turkish (`tr`).
class AppLocalizationsTr extends AppLocalizations {
  AppLocalizationsTr([String locale = 'tr']) : super(locale);

  @override
  String get appTitle => 'BrainVolt';

  @override
  String get homeWelcome => 'Bilgi yarışmasına hoş geldin!';

  @override
  String get settingsTitle => 'Ayarlar';

  @override
  String get settingsLanguage => 'Dil';

  @override
  String get languageSystem => 'Cihaz dili';

  @override
  String get languageTurkish => 'Türkçe';

  @override
  String get languageEnglish => 'English';

  @override
  String get retry => 'Tekrar dene';

  @override
  String get cancel => 'Vazgeç';

  @override
  String get continueButton => 'Devam et';

  @override
  String get save => 'Kaydet';

  @override
  String get splashError => 'Sunucuya bağlanamadık.';

  @override
  String get ageGateTitle => 'Hoş geldin!';

  @override
  String get ageGateBody =>
      'BrainVolt\'u kullanmak için en az 13 yaşında olmalısın.';

  @override
  String get ageGateConfirm => '13 yaşından büyüğüm';

  @override
  String homeGreeting(String name) {
    return 'Merhaba, $name!';
  }

  @override
  String get homeGuestBanner =>
      'Misafir olarak oynuyorsun. İlerlemeni korumak için hesabını bağla.';

  @override
  String get homeVerifyBanner =>
      'E-posta adresini doğrula; doğrulanana kadar misafir sınırları geçerli.';

  @override
  String get linkAccountAction => 'Hesabını bağla';

  @override
  String get verifyAction => 'Doğrula';

  @override
  String get accountTitle => 'Hesap';

  @override
  String get accountGuestInfo =>
      'Misafir hesabı. Bir giriş yöntemi bağlarsan ilerlemen korunur ve başka cihazlarda da oynayabilirsin.';

  @override
  String accountSignedInAs(String email) {
    return '$email ile giriş yapıldı';
  }

  @override
  String get accountSignedIn => 'Hesabın bağlı.';

  @override
  String get accountLinked => 'Hesabın bağlandı. İlerlemen korunuyor.';

  @override
  String get signInWithGoogle => 'Google ile devam et';

  @override
  String get signInWithApple => 'Apple ile devam et';

  @override
  String get signUpWithEmail => 'E-posta ile kaydol';

  @override
  String get signInWithEmail => 'Hesabım var: e-posta ile giriş yap';

  @override
  String get emailSignInTitle => 'E-posta ile giriş';

  @override
  String get emailLabel => 'E-posta';

  @override
  String get passwordLabel => 'Şifre';

  @override
  String get passwordHint => 'En az 6 karakter';

  @override
  String get signUpButton => 'Kaydol';

  @override
  String get signInButton => 'Giriş yap';

  @override
  String get signOut => 'Çıkış yap';

  @override
  String get signOutConfirm =>
      'Çıkış yaparsan bu cihazda yeni bir misafir olarak devam edersin.';

  @override
  String get deleteAccount => 'Hesabımı sil';

  @override
  String get deleteAccountConfirm =>
      'Hesabın 14 gün sonra kalıcı olarak silinir. Bu süre içinde giriş yapıp iptal edebilirsin.';

  @override
  String get linkConflictTitle => 'Hesap zaten kayıtlı';

  @override
  String get linkConflictBody =>
      'Bu hesap zaten kayıtlı. Misafir ilerlemen kaybolacak. Devam edilsin mi?';

  @override
  String get verifyEmailTitle => 'E-postanı doğrula';

  @override
  String verifyEmailBody(String email) {
    return '$email adresine bir doğrulama bağlantısı gönderdik. Bağlantıyı açtıktan sonra aşağıya dokun.';
  }

  @override
  String get verifyEmailDone => 'Doğruladım';

  @override
  String get verifyEmailResend => 'Tekrar gönder';

  @override
  String get verifyEmailSent => 'Doğrulama e-postası gönderildi.';

  @override
  String get verifyEmailNotYet => 'E-posta henüz doğrulanmadı.';

  @override
  String get verifyEmailVerified => 'E-postan doğrulandı.';

  @override
  String get authErrorInvalidCredential => 'E-posta veya şifre hatalı.';

  @override
  String get authErrorWeakPassword => 'Şifre en az 6 karakter olmalı.';

  @override
  String get authErrorInvalidEmail => 'Geçerli bir e-posta adresi gir.';

  @override
  String get authErrorUserNotFound => 'Bu e-postayla kayıtlı bir hesap yok.';

  @override
  String get authErrorTooManyRequests =>
      'Çok fazla deneme yaptın. Biraz sonra tekrar dene.';

  @override
  String get authErrorUnsupported =>
      'Bu giriş yöntemi bu cihazda kullanılamıyor.';

  @override
  String get accountSuspendedTitle => 'Hesabın askıya alındı';

  @override
  String get accountSuspendedBody =>
      'Ayrıntılar için destek ekibiyle iletişime geç.';

  @override
  String get deletionPendingTitle => 'Hesabın silinecek';

  @override
  String deletionPendingBody(DateTime date) {
    final intl.DateFormat dateDateFormat = intl.DateFormat.yMMMMd(localeName);
    final String dateString = dateDateFormat.format(date);

    return 'Hesabın $dateString tarihinde kalıcı olarak silinecek.';
  }

  @override
  String get cancelDeletion => 'Silmeyi iptal et';

  @override
  String get profileTitle => 'Profil';

  @override
  String get displayNameLabel => 'Görünen ad';

  @override
  String get displayNameHint => '2–24 karakter';

  @override
  String get usernameLabel => 'Kullanıcı adı';

  @override
  String get usernameHint => '3–20 karakter: küçük harf, rakam, _ ve .';

  @override
  String get usernameGuestHint =>
      'Kullanıcı adını değiştirmek için hesabını bağla ve doğrula.';

  @override
  String usernameAvailableAt(DateTime date) {
    final intl.DateFormat dateDateFormat = intl.DateFormat.yMMMMd(localeName);
    final String dateString = dateDateFormat.format(date);

    return 'Kullanıcı adını $dateString tarihinden sonra değiştirebilirsin.';
  }

  @override
  String get avatarLabel => 'Avatar';

  @override
  String get profileSaved => 'Profil kaydedildi.';

  @override
  String get settingsGame => 'Oyun';

  @override
  String get settingsSound => 'Ses efektleri';

  @override
  String get settingsMusic => 'Müzik';

  @override
  String get settingsHaptics => 'Titreşim';

  @override
  String get settingsPrivacy => 'Gizlilik';

  @override
  String get settingsShowOnline => 'Çevrimiçi durumumu göster';

  @override
  String get settingsAllowFriendRequests => 'Arkadaşlık isteklerine izin ver';

  @override
  String get settingsChallengesFrom => 'Meydan okumalar';

  @override
  String get challengesEveryone => 'Herkesten';

  @override
  String get challengesFriends => 'Yalnızca arkadaşlardan';

  @override
  String get challengesNobody => 'Kimseden';

  @override
  String get errorNetwork =>
      'Bağlantı kurulamadı. İnternet bağlantını kontrol et.';

  @override
  String get errorTimeout => 'Sunucu zamanında yanıt vermedi. Tekrar dene.';

  @override
  String get errorUnknown => 'Beklenmeyen bir hata oluştu. Lütfen tekrar dene.';

  @override
  String get error_ACCOUNT_DELETION_PENDING =>
      'Hesabın silinme sürecinde. Yalnızca silme işlemini iptal edebilirsin.';

  @override
  String get error_ACCOUNT_SUSPENDED => 'Hesabın askıya alındı.';

  @override
  String get error_ACCOUNT_UPGRADE_REQUIRED =>
      'Bu işlem için hesabını yükseltmelisin.';

  @override
  String get error_AD_REWARD_LIMIT =>
      'Bugün için reklam ödülü sınırına ulaştın.';

  @override
  String get error_AGE_CONFIRMATION_REQUIRED =>
      'Devam etmek için yaşını onaylamalısın.';

  @override
  String get error_ALREADY_FRIENDS => 'Zaten arkadaşsınız.';

  @override
  String get error_ALREADY_IN_MATCH => 'Zaten devam eden bir maçın var.';

  @override
  String get error_ALREADY_QUEUED => 'Zaten eşleşme sırasındasın.';

  @override
  String get error_ALREADY_REPORTED => 'Bunu zaten bildirdin.';

  @override
  String get error_ANSWER_DUPLICATE => 'Bu soruyu zaten cevapladın.';

  @override
  String get error_ANSWER_INVALID_OPTION => 'Geçersiz cevap seçeneği.';

  @override
  String get error_ANSWER_LATE => 'Cevap süresi doldu.';

  @override
  String get error_APPEAL_EXISTS => 'Bu yaptırım için zaten bir itirazın var.';

  @override
  String get error_APP_VERSION_UNSUPPORTED =>
      'Devam etmek için uygulamayı güncellemelisin.';

  @override
  String get error_BLOCKED => 'Bu kullanıcıyla etkileşim kuramazsın.';

  @override
  String get error_CHALLENGES_DISABLED =>
      'Bu kullanıcı şu anda meydan okuma kabul etmiyor.';

  @override
  String get error_CHALLENGE_EXPIRED => 'Meydan okumanın süresi doldu.';

  @override
  String get error_CHANGE_TOO_SOON =>
      'Bunu henüz değiştiremezsin. Daha sonra tekrar dene.';

  @override
  String get error_COUPLE_COOLDOWN =>
      'Yeni bir çift oluşturmak için biraz beklemelisin.';

  @override
  String get error_COUPLE_EXISTS => 'Zaten bir çiftin var.';

  @override
  String get error_DAILY_ATTEMPT_IN_PROGRESS =>
      'Devam eden bir günlük denemen var.';

  @override
  String get error_DAILY_ATTEMPT_USED =>
      'Bugünkü resmi denemeni zaten kullandın.';

  @override
  String get error_DAILY_NOT_AVAILABLE =>
      'Günün meydan okuması henüz hazır değil.';

  @override
  String get error_FORBIDDEN => 'Bu işlem için yetkin yok.';

  @override
  String get error_FRIEND_LIMIT_REACHED => 'Arkadaş sınırına ulaştın.';

  @override
  String get error_IDEMPOTENCY_KEY_REUSED =>
      'Bu istek anahtarı farklı bir istek için zaten kullanıldı.';

  @override
  String get error_IDEMPOTENT_REQUEST_IN_PROGRESS =>
      'Bu istek hâlâ işleniyor. Birazdan tekrar dene.';

  @override
  String get error_INTERNAL =>
      'Beklenmeyen bir hata oluştu. Lütfen tekrar dene.';

  @override
  String get error_INVITE_CODE_EXPIRED => 'Davet kodunun süresi doldu.';

  @override
  String get error_INVITE_CODE_INVALID => 'Davet kodu geçersiz.';

  @override
  String get error_MALFORMED_REQUEST => 'İstek okunamadı.';

  @override
  String get error_MATCH_NOT_FOUND => 'Maç bulunamadı.';

  @override
  String get error_MATCH_STATE_INVALID =>
      'Bu işlem maçın şu anki aşamasında yapılamaz.';

  @override
  String get error_METHOD_NOT_ALLOWED => 'Bu işlem bu adreste desteklenmiyor.';

  @override
  String get error_MODE_NOT_ALLOWED =>
      'Bu oyun modu kombinasyonu desteklenmiyor.';

  @override
  String get error_NOT_A_PARTICIPANT => 'Bu maçın oyuncusu değilsin.';

  @override
  String get error_NOT_CAPTAIN => 'Bunu yalnızca takım kaptanı yapabilir.';

  @override
  String get error_NOT_ENOUGH_QUESTIONS => 'Bu ayarlar için yeterli soru yok.';

  @override
  String get error_NOT_FOUND => 'Aradığın şey bulunamadı.';

  @override
  String get error_NOT_YOUR_TURN => 'Sıra sende değil.';

  @override
  String get error_NO_ACTIVE_SEASON => 'Şu anda aktif bir sezon yok.';

  @override
  String get error_PAYLOAD_TOO_LARGE => 'İstek çok büyük.';

  @override
  String get error_PICK_NOT_ALLOWED => 'Bu kategori seçilemez.';

  @override
  String get error_PURCHASE_ALREADY_PROCESSED => 'Bu satın alma zaten işlendi.';

  @override
  String get error_PURCHASE_INVALID => 'Satın alma doğrulanamadı.';

  @override
  String get error_QUESTION_IN_USE =>
      'Bu soru maçlarda kullanıldığı için bu şekilde değiştirilemez.';

  @override
  String get error_QUESTION_SOURCE_REQUIRED =>
      'Sorunun kaynağı ve lisansı belirtilmeli.';

  @override
  String get error_QUEUE_COOLDOWN => 'Bir süre eşleşme sırasına giremezsin.';

  @override
  String get error_QUIZ_LIMIT_REACHED => 'Quiz sınırına ulaştın.';

  @override
  String get error_QUIZ_SIZE_INVALID =>
      'Quiz soru sayısı izin verilen aralıkta değil.';

  @override
  String get error_RATE_LIMITED =>
      'Çok fazla istek gönderdin. Biraz bekleyip tekrar dene.';

  @override
  String get error_REPORT_LIMIT_REACHED =>
      'Bugün için bildirim sınırına ulaştın.';

  @override
  String get error_REQUEST_PENDING =>
      'Bekleyen bir arkadaşlık isteği zaten var.';

  @override
  String get error_REVENGE_NOT_ALLOWED => 'Rövanş maçı şu anda mümkün değil.';

  @override
  String get error_ROOM_CLOSED => 'Oda kapandı.';

  @override
  String get error_ROOM_FRIENDS_ONLY =>
      'Bu odaya yalnızca kurucunun arkadaşları katılabilir.';

  @override
  String get error_ROOM_FULL => 'Oda dolu.';

  @override
  String get error_SANCTIONED =>
      'Bu özellik hesabın için geçici olarak kısıtlandı.';

  @override
  String get error_SEAT_TAKEN_ELSEWHERE =>
      'Bu maça başka bir cihazdan bağlandın.';

  @override
  String get error_SELF_REVIEW_FORBIDDEN => 'Kendi dosyanı inceleyemezsin.';

  @override
  String get error_TEAM_FULL => 'Takım dolu.';

  @override
  String get error_TEAM_LIMIT_REACHED => 'Takım sınırına ulaştın.';

  @override
  String get error_TEXT_REJECTED => 'Bu metin kurallarımıza uymuyor.';

  @override
  String get error_TOKEN_EXPIRED =>
      'Oturumunun süresi doldu. Lütfen tekrar dene.';

  @override
  String get error_TOURNAMENT_FULL => 'Turnuva dolu.';

  @override
  String get error_TOURNAMENT_REGISTRATION_CLOSED =>
      'Turnuva kayıtları kapandı.';

  @override
  String get error_TOURNAMENT_REQUIREMENTS_NOT_MET =>
      'Bu turnuvanın katılım şartlarını karşılamıyorsun.';

  @override
  String get error_UNAUTHENTICATED => 'Oturum açman gerekiyor.';

  @override
  String get error_UNAVAILABLE =>
      'Hizmet şu anda kullanılamıyor. Biraz sonra tekrar dene.';

  @override
  String get error_USERNAME_TAKEN => 'Bu kullanıcı adı zaten alınmış.';

  @override
  String get error_VALIDATION_FAILED => 'Gönderilen bilgiler geçersiz.';

  @override
  String get error_VERSION_CONFLICT =>
      'Bu kayıt başka biri tarafından değiştirildi. Yenileyip tekrar dene.';
}
