import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/purchases/purchase_gateway.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/ads/domain/ads_repository.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/store/domain/store_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

const full = AuthUser(uid: 'u-full', isAnonymous: false, emailVerified: true);

void main() {
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  Future<TestApp> launch(
    WidgetTester tester, {
    AuthUser? user,
    void Function(TestApp)? setup,
  }) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create(user: user);
    setup?.call(t);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: t.container,
        child: const BrainVoltApp(),
      ),
    );
    await tester.pumpAndSettle();
    return t;
  }

  Future<void> finish(WidgetTester tester, TestApp t) async {
    unawaited(t.container.read(realtimeClientProvider).stop());
    await tester.pump();
    await tester.pump(const Duration(seconds: 5));
  }

  testWidgets('Plus: plans with prices, buy, then active (D-104)', (
    tester,
  ) async {
    final t = await launch(tester, user: full);
    expect(find.byKey(const Key('plus-cta')), findsOneWidget);
    await tester.tap(find.byKey(const Key('plus-cta')));
    await tester.pumpAndSettle();
    expect(find.text('₺49,99/ay'), findsOneWidget);
    expect(find.text('₺399,99/yıl'), findsOneWidget);
    expect(find.text('7 gün ücretsiz dene'), findsOneWidget);
    expect(find.text('7 gün ücretsiz başla'), findsOneWidget);

    // The annual plan has no trial.
    await tester.tap(find.byKey(const Key('plus-option-annual')));
    await tester.pumpAndSettle();
    expect(find.text('Plus\'a geç'), findsWidgets);
    await tester.tap(find.byKey(const Key('plus-option-monthly')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('plus-buy')));
    await tester.pumpAndSettle();
    expect(t.purchases.bought, ['monthly']);
    expect(find.text('Plus\'a hoş geldin!'), findsOneWidget);
    expect(find.byKey(const Key('plus-active')), findsOneWidget);
    expect(find.byKey(const Key('plus-buy')), findsNothing);
    t.container.read(routerProvider).go(Routes.home);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('plus-badge')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('Plus: a cancelled purchase changes nothing', (tester) async {
    final t = await launch(
      tester,
      user: full,
      setup: (t) {
        t.purchases.result = BuyResult.cancelled;
      },
    );
    t.container.read(routerProvider).go(Routes.plus);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('plus-buy')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('plus-active')), findsNothing);
    expect(find.byKey(const Key('plus-buy')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('Plus: guests link their account first', (tester) async {
    final t = await launch(tester);
    t.container.read(routerProvider).go(Routes.plus);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('plus-link-first')), findsOneWidget);
    expect(find.byKey(const Key('plus-buy')), findsNothing);
    await tester.tap(find.byKey(const Key('plus-link-account')));
    await tester.pumpAndSettle();
    expect(find.text('Hesap'), findsWidgets);
    await finish(tester, t);
  });

  testWidgets('Plus: restore', (tester) async {
    final t = await launch(
      tester,
      user: full,
      setup: (t) {
        t.store.value = const Entitlements();
      },
    );
    t.container.read(routerProvider).go(Routes.plus);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('plus-restore')));
    await tester.pumpAndSettle();
    expect(t.purchases.restored, 1);
    expect(
      find.text('Geri yüklenecek bir abonelik bulunamadı.'),
      findsOneWidget,
    );
    await finish(tester, t);
  });

  testWidgets('Plus earns rewarded-ad rewards without the ad', (tester) async {
    final t = await launch(
      tester,
      user: full,
      setup: (t) {
        t.ads.configValue = const AdsConfig(
          streakRepairAvailable: true,
          streakRepairDays: 3,
        );
        t.ads.grantOnReserve = true;
      },
    );
    await tester.tap(find.byKey(const Key('streak-repair-watch')));
    await tester.pumpAndSettle();
    expect(t.adGateway.rewarded, isEmpty);
    expect(find.text('Serin kurtarıldı: 3 gün!'), findsOneWidget);
    await finish(tester, t);
  });
}
