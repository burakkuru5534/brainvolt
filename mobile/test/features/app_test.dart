import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/session_controller.dart';
import 'package:brainvolt/core/l10n/locale_controller.dart';
import 'package:brainvolt/core/network/api_exception.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../support/fakes.dart';

Widget app(TestApp t) => UncontrolledProviderScope(
  container: t.container,
  child: const BrainVoltApp(),
);

void main() {
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  Future<TestApp> launch(
    WidgetTester tester, {
    bool ageConfirmed = true,
  }) async {
    // Tall enough that no screen needs scrolling.
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create(ageConfirmed: ageConfirmed);
    await tester.pumpWidget(app(t));
    await tester.pumpAndSettle();
    return t;
  }

  Future<void> tapKey(WidgetTester tester, String key) async {
    await tester.tap(find.byKey(Key(key)));
    await tester.pumpAndSettle();
  }

  /// For taps that open a dialog while a progress bar keeps animating.
  Future<void> tapToDialog(WidgetTester tester, String key) async {
    await tester.tap(find.byKey(Key(key)));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
  }

  group('age gate (D-020)', () {
    testWidgets('continue needs the confirmation, then a guest starts', (
      tester,
    ) async {
      final t = await launch(tester, ageConfirmed: false);
      expect(find.text('13 yaşından büyüğüm'), findsOneWidget);
      expect(t.auth.calls, isEmpty);

      await tapKey(tester, 'age-continue'); // disabled: nothing happens
      expect(t.auth.calls, isEmpty);

      await tapKey(tester, 'age-confirm');
      await tapKey(tester, 'age-continue');
      expect(t.auth.calls, ['signInAnonymously']);
      expect(t.prefs.getBool('age_confirmed'), isTrue);
      expect(find.text('Merhaba, Misafir 1234!'), findsOneWidget);
      expect(find.byKey(const Key('guest-banner')), findsOneWidget);
    });

    testWidgets('is skipped once confirmed', (tester) async {
      await launch(tester);
      expect(find.text('13 yaşından büyüğüm'), findsNothing);
      expect(find.text('Bugün bilgini sınamaya hazır mısın?'), findsOneWidget);
    });
  });

  testWidgets('a failed start shows the error and retries', (tester) async {
    final t = await TestApp.create();
    t.sessions.syncError = const ApiException(code: ApiException.networkCode);
    await tester.pumpWidget(app(t));
    await tester.pumpAndSettle();
    expect(find.text('Sunucuya bağlanamadık.'), findsOneWidget);

    await tapKey(tester, 'splash-retry');
    expect(find.text('Bugün bilgini sınamaya hazır mısın?'), findsOneWidget);
  });

  group('guest upgrade', () {
    const existing = AuthUser(
      uid: 'existing',
      isAnonymous: false,
      email: 'me@example.com',
      emailVerified: true,
      providers: {AuthMethod.google},
    );

    testWidgets('linking Google keeps the guest (D-013a)', (tester) async {
      final t = await launch(tester);
      await tester.tap(find.text('Hesabını bağla'));
      await tester.pumpAndSettle();
      await tapKey(tester, 'sign-in-google');

      expect(t.auth.calls, containsAllInOrder(['link:google']));
      expect(find.text('google@example.com ile giriş yapıldı'), findsOneWidget);
      expect(t.sessions.calls, isNot(contains(startsWith('abandonGuest'))));
    });

    testWidgets(
      'link conflict asks first and switches only on confirm (D-015)',
      (tester) async {
        final t = await launch(tester);
        t.auth.existing[AuthMethod.google] = existing;
        await tester.tap(find.text('Hesabını bağla'));
        await tester.pumpAndSettle();

        await tapToDialog(tester, 'sign-in-google');
        expect(
          find.text(
            'Bu hesap zaten kayıtlı. Misafir ilerlemen kaybolacak. '
            'Devam edilsin mi?',
          ),
          findsOneWidget,
        );
        await tapKey(tester, 'dialog-cancel');
        expect(t.auth.currentUser!.uid, 'guest-1');
        expect(t.auth.calls, isNot(contains('signIn:google')));

        await tapToDialog(tester, 'sign-in-google');
        await tapKey(tester, 'dialog-confirm');
        expect(t.auth.currentUser!.uid, 'existing');
        expect(t.sessions.calls, contains('abandonGuest:token-of-guest-1'));
        expect(find.text('me@example.com ile giriş yapıldı'), findsOneWidget);
      },
    );

    testWidgets('a cancelled Google sign-in shows nothing', (tester) async {
      final t = await launch(tester);
      t.auth.obtainFailure = const AuthFailure(AuthFailureKind.cancelled);
      await tester.tap(find.text('Hesabını bağla'));
      await tester.pumpAndSettle();
      await tapKey(tester, 'sign-in-google');
      expect(find.byType(SnackBar), findsNothing);
      expect(t.auth.currentUser!.isAnonymous, isTrue);
    });

    testWidgets('email sign-up leads to verification (D-017)', (tester) async {
      final t = await launch(tester);
      await tester.tap(find.text('Hesabını bağla'));
      await tester.pumpAndSettle();
      await tapKey(tester, 'sign-up-email');

      await tester.enterText(find.byKey(const Key('email-field')), 'a@b.co');
      await tester.enterText(find.byKey(const Key('password-field')), '123');
      await tapKey(tester, 'email-submit');
      expect(find.text('Şifre en az 6 karakter olmalı.'), findsOneWidget);
      expect(t.auth.calls, isNot(contains('link:email')));

      await tester.enterText(
        find.byKey(const Key('password-field')),
        'secret1',
      );
      await tapKey(tester, 'email-submit');
      expect(t.auth.calls, contains('sendEmailVerification'));
      expect(find.text('E-postanı doğrula'), findsOneWidget);

      await tapKey(tester, 'verify-check');
      expect(find.text('E-posta henüz doğrulanmadı.'), findsOneWidget);

      t.auth.verifyOnReload = true;
      await tapKey(tester, 'verify-check');
      expect(find.text('E-postan doğrulandı.'), findsOneWidget);
      final s = t.container.read(sessionControllerProvider).requireValue;
      expect((s as SignedIn).session.tier, Tier.full);
    });
  });

  testWidgets('deletion pending holds the app on the status screen', (
    tester,
  ) async {
    final t = await TestApp.create();
    t.sessions.deletionRequestedAt = DateTime.utc(2026, 10, 18);
    await tester.pumpWidget(app(t));
    await tester.pumpAndSettle();
    expect(find.text('Hesabın silinecek'), findsOneWidget);
    expect(find.textContaining('1 Kasım 2026'), findsOneWidget);

    await tapKey(tester, 'cancel-deletion');
    expect(t.sessions.calls, contains('cancelDeletion'));
    expect(find.text('Bugün bilgini sınamaya hazır mısın?'), findsOneWidget);
  });

  group('settings (1.8.6)', () {
    testWidgets('language override is applied, persisted and sent', (
      tester,
    ) async {
      final t = await launch(tester);
      await tapKey(tester, 'open-settings');
      expect(find.text('Ayarlar'), findsOneWidget);

      await tapKey(tester, 'locale-en');
      expect(find.text('Settings'), findsOneWidget);
      expect(t.prefs.getString('locale_override'), 'en');
      expect(t.profiles.calls, contains('updateProfile:{locale: en}'));

      await tapKey(tester, 'locale-system');
      expect(find.text('Ayarlar'), findsOneWidget);
      expect(t.prefs.getString('locale_override'), isNull);
    });

    testWidgets('a failed save puts the switch back', (tester) async {
      final t = await launch(tester);
      await tapKey(tester, 'open-settings');
      SwitchListTile sound() =>
          tester.widget(find.byKey(const Key('setting-sound')));
      expect(sound().value, isTrue);

      await tapKey(tester, 'setting-sound');
      expect(sound().value, isFalse);
      expect(t.profiles.userSettings.soundEnabled, isFalse);

      t.profiles.settingsError = const ApiException(
        code: ApiException.networkCode,
      );
      await tapKey(tester, 'setting-sound');
      expect(sound().value, isFalse);
      expect(
        find.text('Bağlantı kurulamadı. İnternet bağlantını kontrol et.'),
        findsOneWidget,
      );
    });

    testWidgets('game guide opens from settings (D-102)', (tester) async {
      await launch(tester);
      await tapKey(tester, 'open-settings');
      await tapKey(tester, 'open-guide');
      expect(find.text('Oyun rehberi'), findsOneWidget);
      expect(find.byKey(const Key('guide-intro')), findsOneWidget);
      expect(find.textContaining('Hızlı Düello: 7 soru'), findsNothing);
      await tapKey(tester, 'guide-types');
      expect(find.textContaining('Hızlı Düello: 7 soru'), findsOneWidget);
    });

    testWidgets('private profile switch (D-101)', (tester) async {
      final t = await launch(tester);
      await tapKey(tester, 'open-settings');
      await tester.scrollUntilVisible(
        find.byKey(const Key('setting-private')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tapKey(tester, 'setting-private');
      expect(t.profiles.userSettings.profilePrivate, isTrue);
    });

    testWidgets('profile edit sends only changes; guests keep the username', (
      tester,
    ) async {
      final t = await launch(tester);
      await tapKey(tester, 'open-settings');
      await tapKey(tester, 'open-profile');

      final username = tester.widget<TextField>(
        find.byKey(const Key('username-field')),
      );
      expect(username.enabled, isFalse);

      await tester.enterText(
        find.byKey(const Key('display-name-field')),
        'Ayşe',
      );
      await tapKey(tester, 'avatar-owl');
      await tapKey(tester, 'profile-save');
      expect(
        t.profiles.calls,
        contains('updateProfile:{display_name: Ayşe, avatar_id: owl}'),
      );
      expect(find.text('Profil kaydedildi.'), findsOneWidget);
    });
  });

  testWidgets('credits list the question sources (D-006, CC BY-SA)', (
    tester,
  ) async {
    await launch(tester);
    await tapKey(tester, 'open-settings');
    await tapKey(tester, 'open-credits');
    expect(find.text('Kaynaklar ve Lisanslar'), findsOneWidget);
    expect(find.text('Open Trivia Database'), findsOneWidget);
    expect(find.text('Lisans: CC-BY-SA-4.0'), findsOneWidget);
    expect(
      find.text(
        'Bu kaynaktan alınan sorular ve çevirileri aynı lisansla paylaşılır.',
      ),
      findsOneWidget,
    );
    expect(find.byKey(const Key('open-source-licenses')), findsOneWidget);
  });

  test('a stored language override is read on start', () async {
    final t = await TestApp.create(prefs: {'locale_override': 'en'});
    expect(t.container.read(localeOverrideProvider), const Locale('en'));
    expect(
      effectiveLocale(t.container.read(localeOverrideProvider)),
      const Locale('en'),
    );
  });
}
