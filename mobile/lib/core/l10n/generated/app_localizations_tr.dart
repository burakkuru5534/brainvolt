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
  String get homeWelcome => 'Bugün bilgini sınamaya hazır mısın?';

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
  String get creditsTitle => 'Kaynaklar ve Lisanslar';

  @override
  String get creditsQuestions => 'Soruların kaynakları';

  @override
  String creditsLicense(String license) {
    return 'Lisans: $license';
  }

  @override
  String get creditsShareAlike =>
      'Bu kaynaktan alınan sorular ve çevirileri aynı lisansla paylaşılır.';

  @override
  String get creditsLinkCopied => 'Bağlantı kopyalandı.';

  @override
  String get creditsOpenSource => 'Açık kaynak lisansları';

  @override
  String get errorNetwork =>
      'Bağlantı kurulamadı. İnternet bağlantını kontrol et.';

  @override
  String get errorTimeout => 'Sunucu zamanında yanıt vermedi. Tekrar dene.';

  @override
  String get errorUnknown => 'Beklenmeyen bir hata oluştu. Lütfen tekrar dene.';

  @override
  String get playQuickDuel => 'Hızlı Düello';

  @override
  String get playPractice => 'Bota karşı pratik';

  @override
  String get searchingTitle => 'Rakip aranıyor…';

  @override
  String searchingElapsed(int seconds) {
    return '$seconds sn';
  }

  @override
  String get searchingCancel => 'Aramayı iptal et';

  @override
  String get matchFoundTitle => 'Maç bulundu — Hazır mısın?';

  @override
  String get readyButton => 'Hazırım';

  @override
  String readyDeadline(int seconds) {
    return '$seconds sn içinde hazır ol';
  }

  @override
  String get waitingForOpponent => 'Rakip bekleniyor…';

  @override
  String matchStartsIn(int seconds) {
    return '$seconds';
  }

  @override
  String roundOf(int round, int total) {
    return 'Soru $round/$total';
  }

  @override
  String get you => 'Sen';

  @override
  String get opponentAnswered => 'Rakip cevapladı';

  @override
  String get answerCorrect => 'Doğru!';

  @override
  String get answerWrong => 'Yanlış';

  @override
  String get answerTimeout => 'Süre doldu';

  @override
  String pointsGained(int points) {
    return '+$points puan';
  }

  @override
  String get matchWon => 'Kazandın!';

  @override
  String get matchLost => 'Kaybettin';

  @override
  String get matchDraw => 'Berabere';

  @override
  String get matchCancelled => 'Maç iptal edildi';

  @override
  String get opponentLeft => 'Rakibin maçtan ayrıldı';

  @override
  String get backHome => 'Ana sayfaya dön';

  @override
  String get showDetails => 'Soruları gör';

  @override
  String get leaveMatchTitle => 'Maçtan ayrılmak istiyor musun?';

  @override
  String get leaveMatchBody => 'Ayrılırsan maçı kaybetmiş sayılırsın.';

  @override
  String get leaveButton => 'Ayrıl';

  @override
  String get reconnecting => 'Bağlantı yeniden kuruluyor…';

  @override
  String get opponentDisconnected => 'Rakibin bağlantısı koptu';

  @override
  String get seatMoved => 'Maç başka bir cihazda açıldı';

  @override
  String get matchResumed => 'Sunucu değişti, soru yenilendi';

  @override
  String get resultTitle => 'Maç sonucu';

  @override
  String resultCorrectAnswer(String answer) {
    return 'Doğru cevap: $answer';
  }

  @override
  String get resumeMatch => 'Maça dön';

  @override
  String get homeFriends => 'Arkadaşlar';

  @override
  String get homeChallenges => 'Meydan okumalar';

  @override
  String get friendsTab => 'Arkadaşlar';

  @override
  String get requestsTab => 'İstekler';

  @override
  String get friendsEmpty =>
      'Henüz arkadaşın yok. Kullanıcı adı ya da arkadaş koduyla ekleyebilirsin.';

  @override
  String get requestsEmpty => 'Bekleyen istek yok.';

  @override
  String get friendsUpgrade => 'Arkadaş eklemek için hesabını bağla.';

  @override
  String friendCode(String code) {
    return 'Arkadaş kodun: $code';
  }

  @override
  String get copied => 'Kopyalandı';

  @override
  String get addFriend => 'Arkadaş ekle';

  @override
  String get addByUsername => 'Kullanıcı adı';

  @override
  String get addByCode => 'Arkadaş kodu';

  @override
  String get send => 'Gönder';

  @override
  String get friendRequestSent => 'Arkadaşlık isteği gönderildi.';

  @override
  String get friendAdded => 'Artık arkadaşsınız!';

  @override
  String get online => 'Çevrimiçi';

  @override
  String get incomingLabel => 'Gelen';

  @override
  String get outgoingLabel => 'Giden';

  @override
  String get accept => 'Kabul et';

  @override
  String get decline => 'Reddet';

  @override
  String get withdraw => 'Geri çek';

  @override
  String get unfriend => 'Arkadaşlıktan çıkar';

  @override
  String get blockUser => 'Engelle';

  @override
  String blockConfirm(String name) {
    return '$name engellensin mi? Arkadaşlığınız ve bekleyen istekler sona erer.';
  }

  @override
  String get challengeAction => 'Meydan oku';

  @override
  String challengeTitle(String name) {
    return '$name adlı oyuncuya meydan oku';
  }

  @override
  String get challengeSent => 'Meydan okuma gönderildi.';

  @override
  String get matchTypeQuickDuel => 'Hızlı Düello';

  @override
  String get matchTypeSuddenDeath => 'Ani Ölüm';

  @override
  String get matchTypeMarathon => 'Maraton';

  @override
  String get challengesTitle => 'Meydan okumalar';

  @override
  String get challengesEmpty => 'Bekleyen meydan okuma yok.';

  @override
  String challengeFrom(String name) {
    return '$name sana meydan okudu';
  }

  @override
  String challengeTo(String name) {
    return '$name adlı oyuncuya';
  }

  @override
  String challengeReceived(String name) {
    return '$name sana meydan okudu!';
  }

  @override
  String get challengeAccepted => 'Meydan okuman kabul edildi!';

  @override
  String get view => 'Gör';

  @override
  String get playNow => 'Oyna';

  @override
  String get rematchButton => 'Tekrar Oyna';

  @override
  String get rematchWaiting => 'Diğer oyuncular bekleniyor…';

  @override
  String get rematchOffered => 'Tekrar oynama teklifi geldi!';

  @override
  String get rematchDeclined => 'Bu sefer tekrar oynanmayacak.';

  @override
  String get playModes => 'Diğer modlar';

  @override
  String get modesTitle => 'Mod seç';

  @override
  String get matchTypeRanked => 'Dereceli';

  @override
  String get categoryModeLabel => 'Kategori modu';

  @override
  String get categoryModeNone => 'Standart';

  @override
  String get categoryModeBlind => 'Kör Meydan Okuma';

  @override
  String get categoryModeMaster => 'Kategori Ustası';

  @override
  String get categoryModeMixed => 'Karışık Savaş';

  @override
  String get categoryLabel => 'Kategori';

  @override
  String get startSearch => 'Rakip ara';

  @override
  String get pickCategory => 'Bir kategori seç';

  @override
  String opponentPicking(String name) {
    return '$name kategori seçiyor…';
  }

  @override
  String get privateRoom => 'Özel oda';

  @override
  String get createRoom => 'Oda kur';

  @override
  String get joinRoom => 'Kodla katıl';

  @override
  String get roomCodeLabel => 'Oda kodu';

  @override
  String roomCode(String code) {
    return 'Oda kodu: $code';
  }

  @override
  String sideLabel(int no) {
    return 'Taraf $no';
  }

  @override
  String get emptySeat => 'Boş koltuk — buraya geç';

  @override
  String get host => 'Kurucu';

  @override
  String get readyLabel => 'Hazır';

  @override
  String get notReady => 'Hazır değil';

  @override
  String get startMatch => 'Maçı başlat';

  @override
  String get roomWaitingPlayers => 'Oyuncu bekleniyor…';

  @override
  String get leaveRoom => 'Odadan çık';

  @override
  String get roomClosed => 'Oda kapandı.';

  @override
  String get backToRoom => 'Odaya dön';

  @override
  String get homeTeams => 'Takımlar';

  @override
  String get teamsUpgrade => 'Takım kurmak ve çift olmak için hesabını bağla.';

  @override
  String get coupleSection => 'Çift profili';

  @override
  String get noCouple => 'Bir çift profilin yok.';

  @override
  String get requestCouple => 'Çift ol';

  @override
  String coupleWith(String name) {
    return '$name ile çift';
  }

  @override
  String get couplePending => 'Onay bekleniyor';

  @override
  String get coupleActive => 'Aktif';

  @override
  String get dissolveCouple => 'Çifti boz';

  @override
  String get teamInvites => 'Takım davetleri';

  @override
  String get myTeams => 'Takımlarım';

  @override
  String get teamsEmpty => 'Henüz bir takımın yok.';

  @override
  String get createTeam => 'Takım kur';

  @override
  String get teamName => 'Takım adı';

  @override
  String get teamTag => 'Kısaltma (2–4)';

  @override
  String get captain => 'Kaptan';

  @override
  String get makeCaptain => 'Kaptan yap';

  @override
  String get kick => 'Takımdan çıkar';

  @override
  String get inviteToTeam => 'Arkadaş davet et';

  @override
  String get teamInviteSent => 'Davet gönderildi.';

  @override
  String get leaveTeam => 'Takımdan ayrıl';

  @override
  String get disbandTeam => 'Takımı dağıt';

  @override
  String get disbandConfirm => 'Takım dağıtılsın mı? Bu geri alınamaz.';

  @override
  String get pickFriend => 'Arkadaş seç';

  @override
  String get yourTeam => 'Takımın';

  @override
  String get opponentsTeam => 'Rakipler';

  @override
  String get lockAnswer => 'Cevabı kilitle';

  @override
  String get yourTurn => 'Sıra sende';

  @override
  String turnOf(String name) {
    return 'Sıra: $name';
  }

  @override
  String get formatLabel => 'Format';

  @override
  String get formatCouple => 'Çift';

  @override
  String get teamModeLabel => 'Takım modu';

  @override
  String get teamModeShared => 'Ortak Cevap';

  @override
  String get teamModeTurns => 'Sırayla';

  @override
  String get teamModeFast => 'Hızlı Takım Turu';

  @override
  String get partyLabel => 'Grubun';

  @override
  String get pickParty => 'Arkadaşlarını seç';

  @override
  String get playAsTeam => 'Takımımla oyna';

  @override
  String get noTeam => 'Takımsız';

  @override
  String get leaderboardsTitle => 'Liderlik Tablosu';

  @override
  String get questsTitle => 'Görevler';

  @override
  String get badgesTitle => 'Rozetler';

  @override
  String get periodDaily => 'Günlük';

  @override
  String get periodWeekly => 'Haftalık';

  @override
  String get periodMonthly => 'Aylık';

  @override
  String get periodAllTime => 'Tüm zamanlar';

  @override
  String get periodSeason => 'Sezon';

  @override
  String get entityUser => 'Bireysel';

  @override
  String get entityTeam => 'Takımlar';

  @override
  String get entityCouple => 'Çiftler';

  @override
  String get scopeGlobal => 'Dünya';

  @override
  String get scopeCountry => 'Ülke';

  @override
  String get scopeFriends => 'Arkadaşlar';

  @override
  String get leaderboardEmpty => 'Burada henüz kimse yok.';

  @override
  String get leaderboardMe => 'Sen';

  @override
  String boardPoints(int count) {
    return '$count puan';
  }

  @override
  String levelLabel(int level) {
    return 'Seviye $level';
  }

  @override
  String xpProgress(int current, int total) {
    return '$current / $total XP';
  }

  @override
  String streakDays(int days) {
    return '$days gün seri';
  }

  @override
  String seasonPoints(int sp) {
    return '$sp SP';
  }

  @override
  String get leagueBronze => 'Bronz';

  @override
  String get leagueSilver => 'Gümüş';

  @override
  String get leagueGold => 'Altın';

  @override
  String get leaguePlatinum => 'Platin';

  @override
  String get leagueDiamond => 'Elmas';

  @override
  String nextDivisionAt(int sp) {
    return 'Sonraki kademe: $sp SP';
  }

  @override
  String get questsDaily => 'Günlük görevler';

  @override
  String get questsWeekly => 'Haftalık görevler';

  @override
  String get questClaim => 'Ödülü al';

  @override
  String get questClaimed => 'Alındı';

  @override
  String get questReroll => 'Değiştir';

  @override
  String questRerollsLeft(int count) {
    return 'Bugün kalan değiştirme: $count';
  }

  @override
  String questXpGained(int xp) {
    return '+$xp XP kazandın';
  }

  @override
  String questReward(int xp) {
    return '$xp XP';
  }

  @override
  String get badgeLocked => 'Henüz kazanılmadı';

  @override
  String get awardsTitle => 'Ödüller';

  @override
  String get awardsPending => 'Ödüller hesaplanıyor…';

  @override
  String awardsXp(int xp) {
    return '+$xp XP';
  }

  @override
  String awardsLevelUp(int level) {
    return 'Seviye atladın: $level!';
  }

  @override
  String awardsSp(String sp) {
    return '$sp SP';
  }

  @override
  String awardsLeagueUp(String league) {
    return 'Yeni lig: $league!';
  }

  @override
  String awardsBadge(String name) {
    return 'Yeni rozet: $name';
  }

  @override
  String awardsQuests(int count) {
    return '$count görev tamamlandı';
  }

  @override
  String get revengeLabel => 'Rövanş';

  @override
  String get dailyTitle => 'Günün Meydan Okuması';

  @override
  String get dailyIntro =>
      'Herkes aynı 10 soruyu cevaplar. Günde bir resmi deneme hakkın var; sonrası pratik.';

  @override
  String get dailyStart => 'Başla';

  @override
  String get dailyResume => 'Devam et';

  @override
  String get dailyPractice => 'Pratik yap';

  @override
  String get dailyNotAvailable => 'Bugünün soruları henüz hazır değil.';

  @override
  String dailyResult(int correct, String seconds) {
    return '$correct/10 doğru · $seconds sn';
  }

  @override
  String dailyRank(int rank) {
    return 'Sıralaman: #$rank';
  }

  @override
  String get dailyBoardTitle => 'Günün tablosu';

  @override
  String dailyQuestionOf(int position, int total) {
    return 'Soru $position/$total';
  }

  @override
  String get dailyPracticeLabel => 'Pratik';

  @override
  String get dailyDone => 'Tamamlandı!';

  @override
  String get dailyBack => 'Günün meydan okumasına dön';

  @override
  String get dailyVoided => 'Bu deneme sayılmadı.';

  @override
  String get dailyCorrect => 'Doğru!';

  @override
  String get dailyWrong => 'Yanlış';

  @override
  String get dailyTimeout => 'Süre doldu';

  @override
  String get reportQuestion => 'Soruyu bildir';

  @override
  String get reportReasonTitle => 'Sorun ne?';

  @override
  String get reportReasonWrongAnswer => 'Cevap yanlış';

  @override
  String get reportReasonAmbiguous => 'Belirsiz / birden fazla doğru';

  @override
  String get reportReasonOutdated => 'Bilgi güncel değil';

  @override
  String get reportReasonTypo => 'Yazım veya çeviri hatası';

  @override
  String get reportReasonOffensive => 'Uygunsuz içerik';

  @override
  String get reportReasonOther => 'Diğer';

  @override
  String get reportComment => 'Not (isteğe bağlı)';

  @override
  String get reportSend => 'Gönder';

  @override
  String get reportSent => 'Bildirimin için teşekkürler.';

  @override
  String get sanctionsTitle => 'Yaptırımlarım';

  @override
  String get sanctionsEmpty => 'Hesabında yaptırım yok.';

  @override
  String get sanctionWarning => 'Uyarı';

  @override
  String get sanctionChatMute => 'Sohbet engeli';

  @override
  String get sanctionRankedBan => 'Dereceli yasağı';

  @override
  String get sanctionMatchmakingBan => 'Eşleştirme yasağı';

  @override
  String get sanctionSocialBan => 'Sosyal yasak';

  @override
  String get sanctionSuspension => 'Hesap askıya alma';

  @override
  String sanctionUntil(String date) {
    return '$date tarihine kadar';
  }

  @override
  String get sanctionPermanent => 'Süresiz';

  @override
  String get sanctionEnded => 'Sona erdi';

  @override
  String get appealButton => 'İtiraz et';

  @override
  String get appealHint => 'Neden haksız olduğunu yaz (en az 10 karakter)';

  @override
  String get appealSent => 'İtirazın alındı.';

  @override
  String get appealPending => 'İtiraz inceleniyor';

  @override
  String get appealAccepted => 'İtiraz kabul edildi';

  @override
  String get appealRejected => 'İtiraz reddedildi';

  @override
  String get quizzesTitle => 'Soru Atölyesi';

  @override
  String get quizzesMine => 'Quizlerim';

  @override
  String get quizzesDiscover => 'Keşfet';

  @override
  String get quizzesEmptyMine => 'Henüz bir quiz yazmadın.';

  @override
  String get quizzesEmptyDiscover => 'Yayında quiz yok.';

  @override
  String get quizNew => 'Yeni quiz';

  @override
  String get quizEdit => 'Quizi düzenle';

  @override
  String get quizTitleField => 'Başlık';

  @override
  String get quizDescriptionField => 'Açıklama (isteğe bağlı)';

  @override
  String get quizLanguage => 'Dil';

  @override
  String get quizCategory => 'Kategori';

  @override
  String get quizWindow => 'Cevap süresi';

  @override
  String quizSeconds(int seconds) {
    return '$seconds sn';
  }

  @override
  String quizQuestionN(int n) {
    return 'Soru $n';
  }

  @override
  String get quizPrompt => 'Soru metni';

  @override
  String quizOptionN(int n) {
    return 'Seçenek $n';
  }

  @override
  String get quizTrueFalse => 'Doğru / Yanlış';

  @override
  String get quizTrue => 'Doğru';

  @override
  String get quizFalse => 'Yanlış';

  @override
  String get quizCorrectHint => 'Doğru cevabı daire ile işaretle.';

  @override
  String get quizAddQuestion => 'Soru ekle';

  @override
  String get quizRemoveQuestion => 'Soruyu sil';

  @override
  String get quizSave => 'Kaydet';

  @override
  String get quizSaved => 'Quiz kaydedildi.';

  @override
  String get quizInvalid =>
      'Başlık 3–60 karakter olmalı ve her soru ile seçenek dolu olmalı.';

  @override
  String quizCountRule(int min, int max) {
    return '$min–$max soru ekleyebilirsin.';
  }

  @override
  String get quizStatusDraft => 'Taslak';

  @override
  String get quizStatusSubmitted => 'İncelemede';

  @override
  String get quizStatusPublished => 'Yayında';

  @override
  String get quizStatusRejected => 'Reddedildi';

  @override
  String get quizStatusUnpublished => 'Yayından kaldırıldı';

  @override
  String get quizSubmit => 'İncelemeye gönder';

  @override
  String get quizSubmitted => 'Quiz incelemeye gönderildi.';

  @override
  String get quizDelete => 'Sil';

  @override
  String get quizDeleteConfirm => 'Bu quiz silinsin mi?';

  @override
  String get quizPlayInRoom => 'Odada oyna';

  @override
  String quizMeta(int count, String author) {
    return '$count soru · $author';
  }

  @override
  String quizLikes(int count, int plays) {
    return '$count beğeni · $plays oyun';
  }

  @override
  String get quizLike => 'Beğen';

  @override
  String quizRejectedReason(String reason) {
    return 'Red sebebi: $reason';
  }

  @override
  String get quizUnpublishedNote =>
      'Yayında olmayan quiz yalnızca arkadaşlarınla oynanır ve ödül vermez.';

  @override
  String roomQuiz(int count) {
    return 'Quiz · $count soru';
  }

  @override
  String get roomFriendsOnly =>
      'Yalnızca ev sahibinin arkadaşları katılabilir.';

  @override
  String get homeQuizzes => 'Soru Atölyesi';

  @override
  String get homeQuickDuelSub => 'Sıradaki rakibini bul';

  @override
  String get homeModesSub => 'Tüm modları keşfet';

  @override
  String get homeRoomSub => 'Oda kur ya da katıl';

  @override
  String get homeTeamsSub => 'Birlikte oyna, kazan';

  @override
  String get homeDailySub => 'Günün 10 sorusu';

  @override
  String get homeQuizzesSub => 'Soru öner, quiz hazırla';

  @override
  String get homePracticeSub => 'Tek başına pratik yap';

  @override
  String get homeTotalWins => 'Toplam galibiyet';

  @override
  String get homeCurrentRank => 'Mevcut lig';

  @override
  String get homeNoRank => 'Sıralamasız';

  @override
  String get roomSheetTitle => 'Özel Oda';

  @override
  String get roomSheetSub => 'Nasıl oynamak istediğini seç';

  @override
  String get roomQuickDuelSub => 'Arkadaşlarınla klasik maç';

  @override
  String get roomSuddenDeathSub => 'Tek hata her şeyi değiştirir';

  @override
  String get roomMarathonSub => 'Uzun soluklu yarış';

  @override
  String get roomJoinWithCode => 'Kodla katıl';

  @override
  String get roomCodeHint => 'Oda kodunu gir';

  @override
  String get roomJoinAction => 'Katıl';

  @override
  String get gameModeLabel => 'Oyun modu';

  @override
  String get queueFinding => 'Sıradaki meydan okuman bulunuyor';

  @override
  String get inboxTitle => 'Bildirimler';

  @override
  String get inboxEmpty => 'Henüz bildirimin yok.';

  @override
  String get inboxMarkAll => 'Tümünü okundu say';

  @override
  String get notificationSettingsTitle => 'Bildirim ayarları';

  @override
  String get notifPush => 'Anlık';

  @override
  String get notifInApp => 'Uygulama içi';

  @override
  String get notifQuietHours => 'Sessiz saatler';

  @override
  String get notifQuietHoursHint =>
      'Bu saatlerde anlık bildirim gelmez; bildirimler kutuda bekler.';

  @override
  String get notifQuietStart => 'Başlangıç';

  @override
  String get notifQuietEnd => 'Bitiş';

  @override
  String get notifLocked => 'Hesap bildirimleri kapatılamaz.';

  @override
  String get notifCatChallenge => 'Meydan okumalar';

  @override
  String get notifCatFriend => 'Arkadaşlar';

  @override
  String get notifCatMatchFound => 'Maç bulundu';

  @override
  String get notifCatTeam => 'Takım ve çift';

  @override
  String get notifCatChat => 'Sohbet';

  @override
  String get notifCatSeason => 'Sezon';

  @override
  String get notifCatQuest => 'Görevler';

  @override
  String get notifCatTournament => 'Turnuvalar';

  @override
  String get notifCatModeration => 'Hesap bildirimleri';

  @override
  String get notifCatProductNews => 'Haberler ve kampanyalar';

  @override
  String get settingsNotifications => 'Bildirimler';

  @override
  String get shareAction => 'Paylaş';

  @override
  String get shareResult => 'Sonucu paylaş';

  @override
  String shareMatchText(String score, String result) {
    return 'BrainVolt\'ta $score: $result Sıra sende!';
  }

  @override
  String shareDailyText(int correct) {
    return 'BrainVolt Günün Meydan Okuması: $correct/10 doğru. Sen kaç yaparsın?';
  }

  @override
  String shareCorrectOf(int correct, int total) {
    return '$correct/$total doğru';
  }

  @override
  String get chatTitle => 'Sohbet';

  @override
  String get chatHint => 'Mesaj yaz…';

  @override
  String get chatSend => 'Gönder';

  @override
  String get chatEmpty => 'Henüz mesaj yok. İlk mesajı sen yaz!';

  @override
  String get chatRemoved => 'Bu mesaj silindi.';

  @override
  String chatTyping(String name) {
    return '$name yazıyor…';
  }

  @override
  String get chatDelete => 'Mesajı sil';

  @override
  String get chatOpen => 'Sohbeti aç';

  @override
  String get chatQuickGoodLuck => 'Bol şans!';

  @override
  String get chatQuickWellPlayed => 'İyi oyundu!';

  @override
  String get chatQuickNice => 'Harika!';

  @override
  String get chatQuickThanks => 'Teşekkürler!';

  @override
  String get chatQuickOops => 'Hay aksi!';

  @override
  String get chatQuickLetsGo => 'Hadi başlayalım!';

  @override
  String get chatQuickGg => 'GG';

  @override
  String get chatQuickRematch => 'Tekrar oynayalım mı?';

  @override
  String get tournamentsTitle => 'Turnuvalar';

  @override
  String get homeTournamentsSub => 'Kupayı kazan';

  @override
  String get tournamentStatusRegistration => 'Kayıt açık';

  @override
  String get tournamentStatusRunning => 'Devam ediyor';

  @override
  String get tournamentStatusFinished => 'Bitti';

  @override
  String get tournamentStatusCancelled => 'İptal edildi';

  @override
  String get tournamentStatusDraft => 'Taslak';

  @override
  String get tournamentCheckedIn => 'Check-in yapıldı';

  @override
  String get tournamentPlaying => 'Turnuvada';

  @override
  String tournamentPlace(int place) {
    return '$place. oldun';
  }

  @override
  String get tournamentChampion => 'Şampiyon! 🏆';

  @override
  String get tournamentRegistered => 'Kayıtlı';

  @override
  String get tournamentKindCouple => 'Çiftler';

  @override
  String tournamentKindTeam(String format) {
    return 'Takımlar $format';
  }

  @override
  String get tournamentKindSolo => 'Bireysel';

  @override
  String get tournamentCreate => 'Turnuva oluştur';

  @override
  String get tournamentsEmpty => 'Şu an turnuva yok. Yenileri yakında!';

  @override
  String tournamentEntrants(int count, int max) {
    return '$count/$max katılımcı';
  }

  @override
  String get tournamentBracket => 'Eşleşme tablosu';

  @override
  String get tournamentRegister => 'Kayıt ol';

  @override
  String get tournamentCheckedInInfo =>
      'Check-in tamam. Turnuva başlayınca maçın burada görünecek.';

  @override
  String get tournamentCheckInNow =>
      'Check-in açık! Oynamak için check-in yap.';

  @override
  String tournamentCheckInAt(String time) {
    return 'Check-in $time itibarıyla açılır. Check-in yapmayanlar turnuvaya alınmaz.';
  }

  @override
  String get tournamentCheckIn => 'Check-in yap';

  @override
  String get tournamentWithdraw => 'Kaydı geri çek';

  @override
  String tournamentYourMatch(int round, String name) {
    return '$round. tur: rakibin $name';
  }

  @override
  String get tournamentWaitingOpponent => 'Buradasın. Rakibin bekleniyor…';

  @override
  String tournamentReadyLeft(String time) {
    return 'Katılmak için kalan süre: $time';
  }

  @override
  String get tournamentGoToMatch => 'Maça git';

  @override
  String get tournamentPlay => 'Oyna';

  @override
  String get tournamentWaitingRound => 'Sıradaki turu bekliyorsun.';

  @override
  String tournamentStarts(String time) {
    return 'Başlangıç: $time';
  }

  @override
  String tournamentRewardLine(int place, int xp) {
    return '$place.: $xp XP';
  }

  @override
  String get tournamentFinal => 'Final';

  @override
  String get tournamentSemiFinal => 'Yarı final';

  @override
  String tournamentRound(int round) {
    return '$round. tur';
  }

  @override
  String get tournamentBye => 'Bay';

  @override
  String get tournamentWalkover => 'Hükmen';

  @override
  String tournamentPickLineup(int size) {
    return 'Takımını ve $size oyuncuyu seç';
  }

  @override
  String get tournamentNoCaptainTeam => 'Kaptanı olduğun bir takım yok.';

  @override
  String get tournamentNameTr => 'Ad (Türkçe)';

  @override
  String get tournamentNameEn => 'Ad (İngilizce, isteğe bağlı)';

  @override
  String get tournamentKind => 'Katılım';

  @override
  String get tournamentFormat => 'Takım boyutu';

  @override
  String get tournamentMatchType => 'Maç türü';

  @override
  String get tournamentMax => 'En fazla katılımcı';

  @override
  String get tournamentStartIn => 'Başlamasına';

  @override
  String tournamentMinutes(int minutes) {
    return '$minutes dk';
  }

  @override
  String get tournamentWindow => 'Maça katılma süresi';

  @override
  String get tournamentCreateHint =>
      'Kayıt hemen açılır, başlangıçta kapanır. Check-in son 1 saatte. Ödüller: 1. 300 XP, 2. 150 XP, 3. 75 XP, diğerleri 20 XP.';

  @override
  String get tournamentMatchStarted => 'Turnuva maçın başladı!';

  @override
  String roomReturning(int seconds) {
    return '$seconds sn içinde odaya dönülüyor…';
  }

  @override
  String get roomSettingsTitle => 'Oda ayarları';

  @override
  String get matchTypeLabel => 'Maç tipi';

  @override
  String get roomSettingsSaved =>
      'Oda ayarları güncellendi, herkes tekrar hazır olmalı';

  @override
  String get mmLobbyTitle => 'Eşleşme Lobisi';

  @override
  String mmLobbyWaiting(int count, int total) {
    return 'Oyuncular aranıyor… ($count/$total)';
  }

  @override
  String get mmLobbyLeaderSoon =>
      'Lobi dolunca rastgele bir lider seçilecek; maçı o başlatır.';

  @override
  String get mmLobbyYouLead =>
      'Lobi lideri sensin! Herkes hazır olunca maçı başlat.';

  @override
  String mmLobbyLeaderIs(String name) {
    return 'Lobi lideri: $name. Hazır ol, maçı lider başlatacak.';
  }

  @override
  String get mmLobbyWaitReady => 'Herkesin hazır olması bekleniyor';

  @override
  String get mmLobbySearchingSeat => 'Oyuncu aranıyor…';

  @override
  String get mmLobbyDropped =>
      'Zamanında hazır olmadığın için lobiden çıkarıldın.';

  @override
  String get mmLobbyExpired => 'Yeterli oyuncu bulunamadı, lobi kapandı.';

  @override
  String get topicLabel => 'Konu';

  @override
  String get topicAll => 'Tüm konular';

  @override
  String get topicAllSub => 'Her konudan karışık sorular';

  @override
  String get topicPickTitle => 'Hangi konuda yarışmak istersin?';

  @override
  String get topicPickSub => 'Sadece aynı konuyu seçen oyuncularla eşleşirsin.';

  @override
  String topicLine(String name) {
    return 'Konu: $name';
  }

  @override
  String get topicLoadFailed =>
      'Konular yüklenemedi; \"Tüm konular\" ile oynayabilirsin.';

  @override
  String get topicRankedNote =>
      'Dereceli maçlar her zaman tüm konulardan oynanır.';

  @override
  String get brandTagline => 'BİLGİNİ YARIŞTIR, ZİRVEYE ÇIK';

  @override
  String get homeModesTitle => 'Oyun Modları';

  @override
  String get homeSeeAll => 'Tümü';

  @override
  String get homeMoreTitle => 'Keşfet';

  @override
  String get homeAllModes => 'Tüm modlar';

  @override
  String get homeRankedSub => 'Ligini yükselt';

  @override
  String get homeDailyTag => 'BUGÜN';

  @override
  String get homeDailyPlay => 'Oyna';

  @override
  String get navHome => 'Ana Sayfa';

  @override
  String get navLeaderboards => 'Liderlik';

  @override
  String get profileEdit => 'Profili düzenle';

  @override
  String get profileMatches => 'Maç';

  @override
  String get profileWins => 'Galibiyet';

  @override
  String get profileWinRate => 'Kazanma';

  @override
  String get profileBestStreak => 'En iyi seri';

  @override
  String get profileAchievements => 'Başarımlar';

  @override
  String get profileNoBadges =>
      'Henüz rozet kazanmadın. Maç oyna, görevleri tamamla!';

  @override
  String get badgesEmpty => 'Henüz rozet yok.';

  @override
  String get questsEmpty => 'Şu an görev yok, yakında yenileri gelecek.';

  @override
  String get resultRounds => 'Sorular';

  @override
  String get queueTip1 =>
      'Doğru cevabı ne kadar hızlı verirsen o kadar çok puan kazanırsın.';

  @override
  String get queueTip2 =>
      'Ani Ölüm\'de ilk yanlış cevap maçı bitirir: acele etme, emin ol.';

  @override
  String get queueTip3 =>
      'Maraton uzun soluklu bir maç: dikkatini sona kadar koru.';

  @override
  String get queueTip4 =>
      'Dereceli maçlar lig puanını değiştirir; Bronz\'dan Elmas\'a tırman.';

  @override
  String get queueTip5 =>
      'Konu seçersen yalnızca aynı konuyu seçen oyuncularla eşleşirsin.';

  @override
  String get queueTip6 =>
      'Günün Meydan Okuması\'nda herkes aynı 10 soruyu çözer; sıralamada yerini al.';

  @override
  String get queueTip7 =>
      'Görevleri tamamlayıp XP kazan; beğenmediğin bir görevi günde bir kez değiştirebilirsin.';

  @override
  String get queueTip8 => 'Her gün en az bir maç oynayarak serini koru.';

  @override
  String get queueTip9 =>
      'Arkadaş kodunu paylaş, Özel Oda kurup arkadaşlarınla oyna.';

  @override
  String get queueTip10 =>
      'Takım kur, 2v2, 3v3 ve 4v4 maçlarda birlikte yarış.';

  @override
  String get queueTip11 =>
      'Soru Atölyesi\'nde kendi sorularını öner; onaylananları herkes oynar.';

  @override
  String get queueTipLabel => 'İPUCU';

  @override
  String get answerHiddenNote =>
      'Doğru cevabı maç bitince sonuç ekranında görebilirsin.';

  @override
  String get dailyAnswerNextDay =>
      'Doğru cevaplar yarın \"Dünün cevapları\"nda açılır.';

  @override
  String get dailyYesterdayAnswers => 'Dünün cevapları';

  @override
  String get dailyYesterdaySub => 'Dünkü 10 sorunun doğru cevapları';

  @override
  String get dailyAnswersTitle => 'Dünün cevapları';

  @override
  String get dailyAnswersYours => 'Senin cevabın';

  @override
  String get dailyAnswersNotPlayed => 'Bu soruyu cevaplamadın.';

  @override
  String get dailyAnswersEmpty => 'Dün için cevap yok.';

  @override
  String get playerProfileTitle => 'Oyuncu profili';

  @override
  String get playerViewProfile => 'Profili gör';

  @override
  String get playerAddFriend => 'Arkadaş ekle';

  @override
  String get playerAccept => 'İsteği kabul et';

  @override
  String get playerFriend => 'Arkadaşsınız';

  @override
  String get playerRequestSent => 'İstek gönderildi';

  @override
  String get playerPrivate =>
      'Bu profil gizli. İstatistikleri yalnızca arkadaşları görebilir.';

  @override
  String get playerPrivateTag => 'Gizli profil';

  @override
  String get playerBlocked => 'Bu oyuncunun profilini göremezsin.';

  @override
  String get playerStats => 'İstatistikler';

  @override
  String playerLevel(int level) {
    return 'Seviye $level';
  }

  @override
  String get playerLevelLabel => 'Seviye';

  @override
  String get playerMatches => 'Maç';

  @override
  String get playerWins => 'Galibiyet';

  @override
  String get playerWinRate => 'Kazanma';

  @override
  String get playerBadges => 'Rozet';

  @override
  String get playerStreak => 'Seri';

  @override
  String get settingsPrivateProfile => 'Profilimi gizle';

  @override
  String get settingsPrivateProfileSub =>
      'İstatistiklerini yalnızca arkadaşların görür. Adın sıralamalarda görünmeye devam eder.';

  @override
  String get guideTitle => 'Oyun rehberi';

  @override
  String get guideSub => 'Modlar, puanlama, ligler ve daha fazlası';

  @override
  String get guideIntro =>
      'BrainVolt gerçek zamanlı bir bilgi yarışması. Rakibinle aynı anda aynı soruyu görürsün; hem doğru hem hızlı olan kazanır. Aşağıdaki başlıklara dokunarak ayrıntıları okuyabilirsin.';

  @override
  String get guideBasicsTitle => 'Nasıl oynanır?';

  @override
  String get guideBasicsBody =>
      'Her soruda dört seçenek (bazen Doğru / Yanlış) ve bir süre sayacı var. Süre bitmeden bir seçeneğe dokun.\n• İki taraf da aynı soruyu aynı anda görür; bağlantı gecikmesi sunucuda telafi edilir.\n• Yanlış cevap ya da süre dolması puan kaybettirmez, sadece 0 puan verir.\n• Yanlış cevap verdiysen doğru cevap maç sırasında gösterilmez; maç bitince sonuç ekranında tüm sorulara ve cevaplara bakabilirsin.\n• Son 30 gün içinde gördüğün sorular sana tekrar sorulmaz.\n• Bağlantın koparsa uygulama otomatik olarak maça geri döner. Çok uzun kopukluk ya da sık kopmalar maçı kaybetmene yol açar.';

  @override
  String get guideScoringTitle => 'Puanlama';

  @override
  String get guideScoringBody =>
      'Doğru cevap üç parçadan puan toplar:\n• Taban puan: sorunun zorluğuna göre 80–120.\n• Hız bonusu: ne kadar erken cevaplarsan o kadar çok, en fazla +100. Son saniyede cevap verirsen bonus 0 olur.\n• Seri bonusu: üst üste 3–4 doğruda +20, 5 ve sonrasında +40.\nBir soruda en fazla 260 puan alınabilir. Doğru / Yanlış soruları yarı puan değerindedir (en fazla 150).\nMaçın sonunda toplam puanı yüksek olan kazanır; eşitlikte doğru sayısı ve hız belirleyicidir.';

  @override
  String get guideTypesTitle => 'Maç türleri';

  @override
  String get guideTypesBody =>
      '• Hızlı Düello: 7 soru, soru başına 15 sn. Kısa ve rahat bir maç.\n• Dereceli: 10 soru, 15 sn, sadece çoktan seçmeli. Yalnızca rakip eşleştirmeyle oynanır; lig puanını (SP) değiştiren tek mod budur.\n• Ani Ölüm: soru başına 10 sn. Bir tur sende doğru, rakipte yanlışsa maç biter. Zorluk her 3 turda artar; 30 turda karar çıkmazsa puanı yüksek olan kazanır.\n• Maraton: 25 soru, soru başına 12 sn. 12. sorudan sonra 15 sn mola verilir.';

  @override
  String get guideCategoryTitle => 'Konu modları';

  @override
  String get guideCategoryBody =>
      'Dereceli dışındaki maçlarda soruların nasıl seçileceğini belirleyebilirsin:\n• Standart: sorular farklı konulardan karışık gelir.\n• Kör Meydan Okuma: sorunun konusu cevaplanana kadar gizli kalır.\n• Kategori Ustası: tüm sorular seçtiğin tek konudan gelir. Eşleştirmede yalnızca aynı konuyu arayan oyuncularla eşleşirsin.\n• Karışık Savaş: maçta 3 konu belirlenir; her tur sırayla bir taraf bu üç konudan birini seçer (10 sn).\n• Rövanş: kaybeden taraf maç sonunda 24 saat içinde rövanş isteyebilir; soruların yarısı onun seçtiği konudan gelir. Bir seride en fazla 3 rövanş oynanır.';

  @override
  String get guideTeamsTitle => 'Takımlar ve çiftler';

  @override
  String get guideTeamsBody =>
      'Bire bir dışında 2v2, 3v3, 4v4 ve Çift formatlarında oynayabilirsin. Kalıcı bir takım kurup arkadaşlarını davet edebilir ya da bir arkadaşınla çift olabilirsin. Takım maçlarında üç cevap modu var:\n• Ortak Cevap: takım birlikte bir seçenek önerir ve biri cevabı kilitler. Konuşmak için 5 sn ek süre verilir.\n• Sırayla: her soruyu takımdan sıradaki oyuncu cevaplar.\n• Hızlı Takım Turu: herkes kendi başına cevaplar; takımın puanı üyelerin ortalamasından hesaplanır.\nTakımın puanı tek oyuncunun puanıyla aynı ölçektedir; kalabalık takım fazladan puan toplamaz.';

  @override
  String get guideDailyTitle => 'Günün Meydan Okuması';

  @override
  String get guideDailyBody =>
      'Her gün herkes aynı 10 soruyu aynı sırayla cevaplar. Gün İstanbul saatiyle gece yarısı yenilenir.\n• Günde bir resmi deneme hakkın var; sonraki denemeler pratiktir ve sıralamayı etkilemez.\n• Sıralamada önce doğru sayısına, eşitlikte toplam cevap süresine bakılır.\n• Gün içinde doğru cevaplar gösterilmez. Ertesi gün \"Dünün cevapları\" bölümünden bakabilirsin.\n• Bağlantın koparsa 2 dakika içinde kaldığın yerden devam edebilirsin; süre bu sırada işlemeye devam eder.\n• Resmi deneme XP kazandırır (30 + doğru başına 5) ve günlük serini sürdürür.';

  @override
  String get guideRoomsTitle => 'Özel oda ve meydan okumalar';

  @override
  String get guideRoomsBody =>
      '• Özel oda: bir oda kurup kodu arkadaşlarınla paylaş. Ev sahibi maç türünü, formatı ve konu modunu seçer; herkes hazır olunca maç başlar. Maç bitince herkes odaya döner.\n• Meydan okuma: arkadaş listesinden bir arkadaşına doğrudan maç teklifi gönderebilirsin.\n• Bu maçlar XP kazandırır ama SP ve sıralama puanı vermez; böylece arkadaşlar arasında puan kasılamaz.';

  @override
  String get guideTournamentsTitle => 'Turnuvalar';

  @override
  String get guideTournamentsBody =>
      'Turnuvalar eleme usulü oynanır. Kayıt ol, başlangıçtan önceki son 1 saatte check-in yap; check-in yapmayanlar turnuvaya alınmaz.\n• Her turda maçın hazır olunca bildirim alırsın; belirlenen süre içinde maça katılmazsan hükmen kaybedersin.\n• Bireysel, takım ve çift turnuvaları var.\n• Ödüller: 1. 300 XP, 2. 150 XP, 3. 75 XP, diğer katılımcılar 20 XP.';

  @override
  String get guideLabTitle => 'Soru Atölyesi';

  @override
  String get guideLabBody =>
      'Kendi quizini hazırla: tek bir soruyla başla, \"Soru ekle\" ile en fazla 30 soruya çıkar. Cevap süresini (10, 15 ya da 20 sn) sen belirlersin.\n• Taslak quizler yalnızca arkadaşlarınla özel odada oynanır ve ödül vermez.\n• İncelemeye gönderdiğin quiz onaylanırsa yayına girer; herkes keşfedip oynayabilir ve oyuncular XP kazanır.\n• Kendi quizinden sen XP kazanmazsın.';

  @override
  String get guideLeaguesTitle => 'Ligler, sezonlar ve SP';

  @override
  String get guideLeaguesBody =>
      'Sezon Puanı (SP) yalnızca Dereceli maçlardan gelir: galibiyet +30, beraberlik +10, mağlubiyet −15. Üst üste 3. dereceli galibiyetten itibaren +5, +10 ve en fazla +15 seri bonusu eklenir.\n• Ligler: Bronz (0), Gümüş (300), Altın (800), Platin (1600) ve Elmas (2800 SP). Elmas dışındaki her ligin üç kademesi vardır (III, II, I).\n• Sezon içinde bir lig düşmezsin; ligin içindeki kademeler inip çıkabilir.\n• Bir sezon 8 hafta sürer. Sezon sonunda ulaştığın en yüksek lige göre rozet kazanırsın, ardından herkes Bronz III\'ten başlar.';

  @override
  String get guideXpTitle => 'XP, seviye, seri ve görevler';

  @override
  String get guideXpBody =>
      '• Her tamamlanan maç XP kazandırır: 20 + doğru cevap başına 5 + sonuç bonusu (galibiyet 15, beraberlik 8).\n• Seviye atlamak için gereken XP her seviyede biraz artar; en yüksek seviye 100. Seviye sadece ilerlemeni gösterir, eşleştirmeyi ya da puanı etkilemez.\n• Seri: her gün en az bir maç ya da resmi Günün Meydan Okuması tamamlarsan serin büyür; bir gün atlarsan sıfırlanır.\n• Görevler: her gün 3 günlük, her hafta 3 haftalık görev gelir. Tamamlayınca ödülünü al; günde bir görevi ücretsiz değiştirebilirsin.\n• Rozetler: galibiyetler, isabet, hız, seriler, konular ve sezonlar için rozet kazanırsın.';

  @override
  String get guideBoardsTitle => 'Sıralamalar';

  @override
  String get guideBoardsBody =>
      'Günlük, haftalık, aylık ve tüm zamanlar sıralamaları var; dünya, ülke ve arkadaşlar arasında ayrı ayrı bakabilirsin. Takımların ve çiftlerin de kendi sıralamaları var.\n• Sıralama puanı yalnızca eşleştirmeyle oynanan maçlardan gelir: galibiyet 20, beraberlik 10, her doğru cevap +2.\n• Sezon sıralaması SP\'ye göre, Günün Meydan Okuması sıralaması ise ayrı tutulur.\n• Bir oyuncuya dokunarak arkadaş ekleyebilir ya da profiline bakabilirsin. Profilini gizlersen istatistiklerini yalnızca arkadaşların görür.';

  @override
  String get guideFairTitle => 'Adil oyun';

  @override
  String get guideFairBody =>
      '• Puanları ve sonuçları yalnızca sunucu hesaplar; uygulamadaki bir değişiklik sonucu etkilemez.\n• Doğru cevaplar maç sırasında yanlış cevaplayanlara, Günün Meydan Okumasında ise gün bitene kadar kimseye gösterilmez.\n• Arkadaşlar arası maçlar SP ve sıralama puanı vermez.\n• Hatalı bir soru görürsen maç sonunda \"Soruyu bildir\" ile bize ilet. Kurallara aykırı davranışlar yaptırımla sonuçlanabilir.';

  @override
  String get adBonusXpButton => 'Reklam izle: +%50 XP';

  @override
  String adBonusXpDone(int xp) {
    return '+$xp XP eklendi!';
  }

  @override
  String get adNoFill =>
      'Şu an gösterilecek reklam yok. Biraz sonra tekrar dene.';

  @override
  String get adNotEarned => 'Ödül için reklamı sonuna kadar izlemelisin.';

  @override
  String get adPending => 'Ödülün birazdan hesabına geçecek.';

  @override
  String get adQuestReroll => 'Reklam izle, değiştir';

  @override
  String adStreakTitle(int days) {
    return '$days günlük serin bozuldu!';
  }

  @override
  String get adStreakSub => 'Bir reklam izle, serini kurtar.';

  @override
  String adStreakDone(int days) {
    return 'Serin kurtarıldı: $days gün!';
  }

  @override
  String get adWatch => 'İzle';

  @override
  String get plusTitle => 'BrainVolt Plus';

  @override
  String get plusTagline => 'Reklamsız oyna, her gün daha fazla ödül kazan.';

  @override
  String get plusPerkNoAds => 'Reklamsız oyun: ara reklam yok';

  @override
  String get plusPerkRewards => 'Ödüllü reklam ödülleri reklam izlemeden';

  @override
  String get plusPerkQuests => 'Her gün +1 görev ve +1 ücretsiz değiştirme';

  @override
  String get plusPerkBadge => 'Ana sayfada Plus rozeti';

  @override
  String get plusMonthly => 'Aylık';

  @override
  String get plusAnnual => 'Yıllık';

  @override
  String plusPerMonth(String price) {
    return '$price/ay';
  }

  @override
  String plusPerYear(String price) {
    return '$price/yıl';
  }

  @override
  String plusTrial(int days) {
    return '$days gün ücretsiz dene';
  }

  @override
  String plusStartTrial(int days) {
    return '$days gün ücretsiz başla';
  }

  @override
  String get plusBuy => 'Plus\'a geç';

  @override
  String get plusRestore => 'Satın alımları geri yükle';

  @override
  String get plusLegal =>
      'Abonelik iptal edilmedikçe her dönemin sonunda Google Play üzerinden otomatik yenilenir. İstediğin zaman Google Play\'den iptal edebilirsin.';

  @override
  String get plusActive => 'Plus aktif.';

  @override
  String plusActiveUntil(String date) {
    return 'Plus aktif. Yenileme tarihi: $date';
  }

  @override
  String get plusManage => 'Aboneliği yönet';

  @override
  String get plusLinkFirst =>
      'Satın almak için önce hesabını Google, Apple ya da e-postaya bağla; böylece aboneliğin kaybolmaz.';

  @override
  String get plusUnavailable => 'Satın alma bu cihazda henüz kullanılamıyor.';

  @override
  String get plusNoPlans =>
      'Planlar şu an yüklenemedi. Biraz sonra tekrar dene.';

  @override
  String get plusWelcome => 'Plus\'a hoş geldin!';

  @override
  String get plusActivating =>
      'Satın alma tamam; Plus birkaç saniye içinde açılacak.';

  @override
  String get plusPending => 'Ödemen onay bekliyor; onaylanınca Plus açılacak.';

  @override
  String get plusFailed => 'Satın alma tamamlanamadı. Tekrar dene.';

  @override
  String get plusNothingToRestore => 'Geri yüklenecek bir abonelik bulunamadı.';

  @override
  String get plusCta => 'Plus\'a geç';

  @override
  String get voltTitle => 'Volt Mağazası';

  @override
  String get voltBalanceLabel => 'Volt bakiyen';

  @override
  String get voltTagline => 'Oynayarak kazan, mağazada harca.';

  @override
  String voltAdTitle(int amount) {
    return 'Reklam izle, $amount Volt kazan';
  }

  @override
  String voltAdLeft(int count) {
    return 'Bugün $count hakkın kaldı';
  }

  @override
  String get voltAdNone => 'Bugünkü reklam hakların bitti';

  @override
  String voltAdDone(int amount) {
    return '+$amount Volt kazandın';
  }

  @override
  String get voltShop => 'Mağaza';

  @override
  String get voltItemShield => 'Seri Kalkanı';

  @override
  String voltItemShieldSub(int count, int max) {
    return 'Kaçırdığın bir günde serini korur. Elinde: $count/$max';
  }

  @override
  String get voltItemReroll => 'Ek görev değiştirme';

  @override
  String get voltItemRerollSub =>
      'Ücretsiz hakkın bitince bir görevi daha değiştir.';

  @override
  String voltBuyTitle(String item) {
    return '$item alınsın mı?';
  }

  @override
  String voltBuyBody(int price) {
    return '$price Volt harcanacak.';
  }

  @override
  String get voltBuy => 'Satın al';

  @override
  String voltBought(String item) {
    return '$item alındı';
  }

  @override
  String get voltEarnTitle => 'Nasıl kazanılır';

  @override
  String get voltEarnQuests => 'Günlük görev +10 · Haftalık görev +50';

  @override
  String get voltEarnDaily => 'Günün Meydan Okuması +10';

  @override
  String get voltEarnLevel => 'Her yeni seviye +25';

  @override
  String get voltEarnTournament => 'Turnuvada ilk üç: +300 / +150 / +75';

  @override
  String get voltEarnSeason => 'Sezon sonu lig ödülü: +50 ile +500 arası';

  @override
  String get voltEarnPlus => 'Plus aboneleri her ay +400';

  @override
  String get voltHistory => 'Son hareketler';

  @override
  String get voltHistoryEmpty => 'Henüz bir hareket yok.';

  @override
  String get voltReasonQuest => 'Görev ödülü';

  @override
  String get voltReasonDaily => 'Günün Meydan Okuması';

  @override
  String voltReasonLevel(String level) {
    return 'Seviye $level';
  }

  @override
  String get voltReasonTournament => 'Turnuva derecesi';

  @override
  String get voltReasonSeason => 'Sezon ödülü';

  @override
  String get voltReasonAd => 'Reklam ödülü';

  @override
  String get voltReasonPlus => 'Plus aylık Volt';

  @override
  String get voltReasonPlusRefund => 'Plus iadesi';

  @override
  String voltReasonItem(String item) {
    return 'Mağaza: $item';
  }

  @override
  String get voltReasonOther => 'Düzeltme';

  @override
  String voltRerollWith(int price) {
    return '$price Volt ile değiştir';
  }

  @override
  String questVoltReward(int volt) {
    return '+$volt';
  }

  @override
  String get error_ACCOUNT_DELETION_PENDING =>
      'Hesabın silinme sürecinde. Yalnızca silme işlemini iptal edebilirsin.';

  @override
  String get error_ACCOUNT_SUSPENDED => 'Hesabın askıya alındı.';

  @override
  String get error_ACCOUNT_UPGRADE_REQUIRED =>
      'Bu işlem için hesabını yükseltmelisin.';

  @override
  String get error_AD_LIMIT_REACHED => 'Bu ödül için bugünkü sınıra ulaştın.';

  @override
  String get error_AD_REWARD_LIMIT =>
      'Bugün için reklam ödülü sınırına ulaştın.';

  @override
  String get error_AD_REWARD_UNAVAILABLE => 'Bu ödül şu an alınamıyor.';

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
  String get error_DAILY_ANSWERS_NOT_OPEN =>
      'Cevaplar o gün bittikten sonra açılır.';

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
  String get error_ITEM_UNAVAILABLE => 'Bu ürünü şu an alamazsın.';

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
  String get error_QUESTION_DUPLICATE => 'Bu soru zaten var.';

  @override
  String get error_QUESTION_IN_USE =>
      'Bu soru maçlarda kullanıldığı için bu şekilde değiştirilemez.';

  @override
  String get error_QUESTION_SOURCE_REQUIRED =>
      'Sorunun kaynağı ve lisansı belirtilmeli.';

  @override
  String get error_QUESTION_STATE_INVALID =>
      'Sorunun şu anki durumunda bu işlem yapılamaz.';

  @override
  String get error_QUEST_ALREADY_CLAIMED => 'Bu ödülü zaten aldın.';

  @override
  String get error_QUEST_NOT_COMPLETED => 'Bu görev henüz tamamlanmadı.';

  @override
  String get error_QUEST_NOT_REROLLABLE => 'Bu görev değiştirilemez.';

  @override
  String get error_QUEST_REROLL_LIMIT =>
      'Bugünkü görev değiştirme hakkın bitti.';

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

  @override
  String get error_VOLT_INSUFFICIENT => 'Yeterli Volt\'un yok.';
}
