import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
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
    void Function(TestApp)? setup,
  }) async {
    tester.view.physicalSize = const Size(800, 2000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create(user: full);
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

  Future<void> openStore(WidgetTester tester) async {
    await tester.tap(find.byKey(const Key('volt-chip')));
    await tester.pumpAndSettle();
  }

  testWidgets('Volt: balance on Home, the store, buying a shield (D-105)', (
    tester,
  ) async {
    final t = await launch(tester);
    expect(
      find.descendant(
        of: find.byKey(const Key('volt-chip')),
        matching: find.text('240'),
      ),
      findsOneWidget,
    );
    await openStore(tester);
    expect(find.byKey(const Key('volt-balance')), findsOneWidget);
    expect(find.text('Seviye 3'), findsOneWidget);
    expect(find.text('Görev ödülü'), findsOneWidget);
    expect(find.text('Mağaza: Seri Kalkanı'), findsOneWidget);
    expect(find.text('Bugün 3 hakkın kaldı'), findsOneWidget);

    await tester.tap(find.byKey(const Key('volt-buy-streak_shield')));
    await tester.pumpAndSettle();
    expect(find.text('Seri Kalkanı alınsın mı?'), findsOneWidget);
    expect(find.text('100 Volt harcanacak.'), findsOneWidget);
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await tester.pumpAndSettle();
    expect(t.store.bought, [StoreItemCode.streakShield]);
    expect(find.text('Seri Kalkanı alındı'), findsOneWidget);
    expect(
      tester.widget<Text>(find.byKey(const Key('volt-balance'))).data,
      '140',
    );
    await finish(tester, t);
  });

  testWidgets('Volt: full shields, refused purchases, the store ad', (
    tester,
  ) async {
    final t = await launch(
      tester,
      setup: (t) {
        t.progress.current = const Progress(
          xpTotal: 120,
          level: 2,
          levelXp: 20,
          levelSize: 125,
          streak: 3,
          longestStreak: 3,
          playedToday: true,
          badges: 2,
          shields: 2,
        );
        t.store.buyError = 'ITEM_UNAVAILABLE';
      },
    );
    await openStore(tester);
    expect(find.textContaining('Elinde: 2/2'), findsOneWidget);
    final shield = tester.widget<ButtonStyleButton>(
      find.byKey(const Key('volt-buy-streak_shield')),
    );
    expect(shield.onPressed, isNull);

    await tester.tap(find.byKey(const Key('volt-buy-quest_reroll')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await tester.pumpAndSettle();
    expect(t.store.bought, isEmpty);
    expect(find.text('Bu ürünü şu an alamazsın.'), findsOneWidget);

    // The store's rewarded ad gives Volt.
    await tester.tap(find.byKey(const Key('volt-ad-watch')));
    await tester.pump();
    await tester.pump(const Duration(seconds: 2));
    await tester.pumpAndSettle();
    expect(t.ads.calls.first, 'reserve:volt:');
    expect(t.adGateway.rewarded, hasLength(1));
    expect(find.text('+5 Volt kazandın'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('Quests show their Volt and offer a Volt reroll', (tester) async {
    final t = await launch(
      tester,
      setup: (t) => t.progress.questList = QuestList(
        daily: [
          Quest(
            id: 'q1',
            cadence: 'daily',
            title: 'Play 3 matches',
            progress: 1,
            target: 3,
            rewardXp: 50,
            rewardVolt: 10,
            endsAt: DateTime(2030),
          ),
        ],
        weekly: const [],
        rerollsLeft: 0,
      ),
    );
    final router = t.container.read(routerProvider);
    unawaited(router.push('/quests'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('quest-volt-q1')), findsOneWidget);
    expect(find.text('+10'), findsOneWidget);
    await tester.tap(find.byKey(const Key('quest-volt-reroll-q1')));
    await tester.pumpAndSettle();
    expect(find.text('Ek görev değiştirme alınsın mı?'), findsOneWidget);
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await tester.pumpAndSettle();
    expect(t.store.bought, [StoreItemCode.questReroll]);
    expect(t.progress.rerolled, ['q1']);
    await finish(tester, t);
  });
}
