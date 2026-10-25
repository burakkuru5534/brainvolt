import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/core/widgets/bv.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
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
    tester.view.physicalSize = const Size(800, 2400);
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

  Future<void> openCosmetics(WidgetTester tester) async {
    await tester.tap(find.byKey(const Key('volt-chip')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('tab-cosmetics')));
    await tester.pumpAndSettle();
  }

  testWidgets('Cosmetics: wear, take off, buy an avatar and use it (D-106)', (
    tester,
  ) async {
    final t = await launch(tester);
    await openCosmetics(tester);

    // Owned: wear it, then take it off.
    await tester.tap(find.byKey(const Key('cos-use-frame_bronze')));
    await tester.pumpAndSettle();
    expect(t.store.equipCalls, ['frame:frame_bronze']);
    expect(find.text('Bronz Halka takıldı'), findsOneWidget);
    await tester.tap(find.byKey(const Key('cos-off-frame_bronze')));
    await tester.pumpAndSettle();
    expect(t.store.equipCalls.last, 'frame:-');

    // Too expensive (240 Volt) and Plus-only items.
    final crown = tester.widget<ButtonStyleButton>(
      find.byKey(const Key('cos-buy-frame_crown')),
    );
    expect(crown.onPressed, isNull);
    expect(find.byKey(const Key('cos-plus-frame_plus')), findsOneWidget);

    // A special avatar: buy, then wear it through the profile.
    await tester.ensureVisible(find.byKey(const Key('cos-buy-avatar_octopus')));
    await tester.tap(find.byKey(const Key('cos-buy-avatar_octopus')));
    await tester.pumpAndSettle();
    expect(find.text('Ahtapot alınsın mı?'), findsOneWidget);
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await tester.pumpAndSettle();
    expect(t.store.bought, ['avatar_octopus']);
    expect(find.text('Ahtapot senin!'), findsOneWidget);
    await tester.tap(find.byKey(const Key('cos-use-avatar_octopus')));
    await tester.pumpAndSettle();
    expect(t.profiles.calls, contains('updateProfile:{avatar_id: octopus}'));
    await finish(tester, t);
  });

  testWidgets('Worn frame on Home; victory animation after a win', (
    tester,
  ) async {
    final t = await launch(
      tester,
      setup: (t) {
        t.store.cosmeticsValue = const MyCosmetics(
          equipped: {'frame': 'frame_crown', 'victory': 'victory_confetti'},
        );
        t.matches.matchResult = MatchResult(
          id: 'm1',
          status: 'completed',
          origin: 'matchmaking',
          matchType: 'quick_duel',
          endedAt: DateTime.now(),
          scores: const [300, 200],
          winnerSide: 1,
          players: const [
            ResultPlayer(
              userId: 'user-u-full',
              side: 1,
              displayName: 'Ben',
              avatarId: 'fox',
              correct: 7,
              status: 'finished',
            ),
          ],
          rounds: const [],
        );
      },
    );
    expect(
      find.byWidgetPredicate(
        (w) => w is AvatarBubble && w.frame == 'frame_crown',
      ),
      findsOneWidget,
    );
    t.container.read(routerProvider).go(Routes.matchResult('m1'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.byKey(const Key('victory-celebration')), findsOneWidget);
    await tester.pumpAndSettle();
    await finish(tester, t);
  });
}
