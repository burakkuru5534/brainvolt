import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/ads/ad_gateway.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/ads/domain/ads_repository.dart';
import 'package:brainvolt/features/ads/presentation/ads.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

MatchResult result(
  String id, {
  String type = 'quick_duel',
  String origin = 'matchmaking',
}) => MatchResult(
  id: id,
  status: 'completed',
  origin: origin,
  matchType: type,
  endedAt: DateTime.now(),
  scores: const [300, 200],
  winnerSide: 1,
  players: const [],
  rounds: const [],
);

void main() {
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  Future<TestApp> launch(
    WidgetTester tester, [
    void Function(TestApp)? setup,
  ]) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create();
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

  testWidgets('result screen: watch an ad for +50% XP (D-103)', (tester) async {
    final t = await launch(tester, (t) {
      t.ads.configValue = const AdsConfig(bonusXpLeft: 5);
      t.matches.matchResult = result('m1');
    });
    t.container.read(routerProvider).go(Routes.matchResult('m1'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('ad-bonus-xp')));
    await tester.pumpAndSettle();
    expect(t.ads.calls, ['reserve:bonus_xp:m1', 'grant:ad-1']);
    expect(t.adGateway.rewarded, ['ad-1']);
    expect(find.text('+35 XP eklendi!'), findsOneWidget);
    expect(find.byKey(const Key('ad-bonus-xp')), findsNothing);
    await finish(tester, t);
  });

  testWidgets('result screen: an ad closed early gives nothing', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.ads.configValue = const AdsConfig(bonusXpLeft: 5);
      t.adGateway.rewardedResult = AdShowResult.dismissed;
      t.matches.matchResult = result('m1');
    });
    t.container.read(routerProvider).go(Routes.matchResult('m1'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('ad-bonus-xp')));
    await tester.pumpAndSettle();
    expect(t.ads.calls, ['reserve:bonus_xp:m1']);
    expect(
      find.text('Ödül için reklamı sonuna kadar izlemelisin.'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('ad-bonus-xp')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('interstitial every 2nd match, never after Ranked (D-103)', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.ads.configValue = const AdsConfig(interstitialEnabled: true);
    });
    final policy = t.container.read(interstitialPolicyProvider);
    expect(await policy.afterMatch(result('a')), isFalse);
    expect(await policy.afterMatch(result('a')), isFalse); // counted once
    expect(await policy.afterMatch(result('r', type: 'ranked')), isFalse);
    expect(await policy.afterMatch(result('t', origin: 'tournament')), isFalse);
    expect(await policy.afterMatch(result('b')), isTrue);
    expect(await policy.afterMatch(result('c')), isFalse);
    expect(t.adGateway.interstitials, 1);
    await finish(tester, t);
  });

  testWidgets('no interstitials when the server turns them off', (
    tester,
  ) async {
    final t = await launch(tester);
    final policy = t.container.read(interstitialPolicyProvider);
    for (final id in ['a', 'b', 'c', 'd']) {
      expect(await policy.afterMatch(result(id)), isFalse);
    }
    expect(t.adGateway.interstitials, 0);
    await finish(tester, t);
  });

  testWidgets('quests: an ad buys a swap after the free one', (tester) async {
    final t = await launch(tester, (t) {
      t.ads.configValue = const AdsConfig(questRerollsLeft: 2);
      t.progress.questList = QuestList(
        daily: [
          Quest(
            id: 'q2',
            cadence: 'daily',
            title: 'Görev q2',
            progress: 0,
            target: 3,
            rewardXp: 50,
            endsAt: DateTime(2030),
          ),
        ],
        weekly: const [],
        rerollsLeft: 0,
      );
    });
    await tester.tap(find.byKey(const Key('open-quests')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('quest-reroll-q2')), findsNothing);
    await tester.tap(find.byKey(const Key('quest-ad-reroll-q2')));
    await tester.pumpAndSettle();
    expect(t.ads.calls, ['reserve:quest_reroll:', 'grant:ad-1']);
    expect(t.progress.rerolled, ['q2']);
    await finish(tester, t);
  });

  testWidgets('home: save a broken streak with an ad', (tester) async {
    final t = await launch(tester, (t) {
      t.ads.configValue = const AdsConfig(
        streakRepairAvailable: true,
        streakRepairDays: 4,
      );
    });
    expect(find.text('4 günlük serin bozuldu!'), findsOneWidget);
    await tester.tap(find.byKey(const Key('streak-repair-watch')));
    await tester.pumpAndSettle();
    expect(t.ads.calls, ['reserve:streak_repair:', 'grant:ad-1']);
    expect(find.text('Serin kurtarıldı: 4 gün!'), findsOneWidget);
    await finish(tester, t);
  });
}
