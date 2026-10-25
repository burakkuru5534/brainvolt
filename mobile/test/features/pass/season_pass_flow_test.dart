import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
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

  Future<TestApp> launch(WidgetTester tester) async {
    tester.view.physicalSize = const Size(800, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create(user: full);
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

  testWidgets('Season Pass: Home card, claim, unlock premium (D-108)', (
    tester,
  ) async {
    final t = await launch(tester);
    expect(find.text('Kademe 6/40'), findsOneWidget);
    expect(find.byKey(const Key('season-pass-claimable')), findsOneWidget);
    await tester.tap(find.byKey(const Key('season-pass-card')));
    await tester.pumpAndSettle();
    expect(find.text('Sezon 1: Fırtına Sezonu'), findsOneWidget);
    expect(find.text('30 gün kaldı'), findsOneWidget);
    expect(find.text('Sonraki kademeye 300 XP'), findsOneWidget);

    // The free track's tier 5.
    await tester.tap(find.byKey(const Key('pass-claim')));
    await tester.pumpAndSettle();
    expect(t.seasonPass.calls, ['claim']);
    expect(find.text('+50 Volt'), findsWidgets);
    expect(find.byKey(const Key('pass-claim')), findsNothing);

    // Premium: confirm, then tiers 1–6 become claimable.
    await tester.tap(find.byKey(const Key('pass-unlock')));
    await tester.pumpAndSettle();
    expect(find.text('Ücretli yol açılsın mı?'), findsOneWidget);
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await tester.pumpAndSettle();
    expect(find.text('Ücretli yol açıldı!'), findsOneWidget);
    expect(find.byKey(const Key('pass-unlock-card')), findsNothing);
    expect(find.text('Ödülleri al (5)'), findsOneWidget);
    await tester.tap(find.byKey(const Key('pass-claim')));
    await tester.pumpAndSettle();
    expect(find.text('+180 Volt · 1 kalkan · 1 kozmetik'), findsOneWidget);
    await finish(tester, t);
  });
}
