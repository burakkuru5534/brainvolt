import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
import 'package:brainvolt/features/social/domain/social_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

Quest quest(String id, {int progress = 0, int target = 3, bool done = false}) =>
    Quest(
      id: id,
      cadence: 'daily',
      title: 'Görev $id',
      progress: progress,
      target: target,
      rewardXp: 50,
      endsAt: DateTime(2030),
      completedAt: done ? DateTime(2026) : null,
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

  testWidgets('home card: level, XP, streak, league', (tester) async {
    final t = await launch(tester);
    expect(find.text('Seviye 2'), findsOneWidget);
    expect(find.text('20 / 125 XP'), findsOneWidget);
    expect(find.textContaining('3 gün seri'), findsOneWidget);
    expect(find.text('Bronz II'), findsOneWidget);
    expect(find.text('130 SP'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('leaderboards: filters change the board, own row pinned', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.progress.boards['weekly/user/global'] = const Board(
        entries: [
          BoardEntry(rank: 1, entityId: 'a', score: 90, name: 'Ali'),
          BoardEntry(rank: 2, entityId: 'me', score: 40, name: 'Ben'),
        ],
        me: BoardEntry(rank: 2, entityId: 'me', score: 40, name: 'Ben'),
      );
      t.progress.boards['season/team/global'] = const Board(
        entries: [
          BoardEntry(
            rank: 1,
            entityId: 't1',
            score: 300,
            name: 'Kartallar [KRT]',
            emblemId: 'bolt',
          ),
        ],
      );
    });
    await tester.tap(find.byKey(const Key('open-leaderboards')));
    await tester.pumpAndSettle();
    expect(find.text('Ali'), findsOneWidget);
    expect(find.text('Sen · Ben'), findsOneWidget);
    expect(find.text('90 puan'), findsOneWidget);

    // Friends only on individual boards: picking teams drops the chip.
    await tester.tap(find.byKey(const Key('scope-friends')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('board-empty')), findsOneWidget);
    await tester.tap(find.byKey(const Key('entity-team')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('scope-friends')), findsNothing);
    await tester.tap(find.byKey(const Key('period-season')));
    await tester.pumpAndSettle();
    expect(find.text('Kartallar [KRT]'), findsOneWidget);
    expect(t.progress.boardCalls, contains('weekly/user/friends'));
    expect(t.progress.boardCalls.last, 'season/team/global');
    await finish(tester, t);
  });

  testWidgets('leaderboards: a player card adds a friend and opens the '
      'profile (D-101)', (tester) async {
    final t = await launch(tester, (t) {
      t.progress.boards['weekly/user/global'] = const Board(
        entries: [
          BoardEntry(
            rank: 1,
            entityId: 'u-ali',
            score: 90,
            name: 'Ali',
            user: UserCard(
              userId: 'u-ali',
              displayName: 'Ali',
              username: 'ali',
            ),
          ),
          BoardEntry(
            rank: 2,
            entityId: 'u-gizli',
            score: 70,
            name: 'Gizli',
            user: UserCard(userId: 'u-gizli', displayName: 'Gizli'),
          ),
        ],
      );
      t.social.profiles['u-ali'] = const PlayerProfile(
        card: UserCard(userId: 'u-ali', displayName: 'Ali', username: 'ali'),
        countryCode: 'TR',
        stats: PlayerStats(level: 7, matches: 40, wins: 30, badges: 5),
      );
      t.social.profiles['u-gizli'] = const PlayerProfile(
        card: UserCard(userId: 'u-gizli', displayName: 'Gizli'),
        private: true,
        relation: Relation.requestSent,
      );
    });
    await tester.tap(find.byKey(const Key('open-leaderboards')));
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('board-row-u-ali')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('player-sheet')), findsOneWidget);
    expect(find.text('@ali'), findsOneWidget);
    expect(find.text('%75'), findsOneWidget);
    await tester.tap(find.byKey(const Key('player-add-friend')));
    await tester.pumpAndSettle();
    expect(t.social.calls, contains('send:u-ali:profile'));

    await tester.tap(find.byKey(const Key('player-view-profile')));
    await tester.pumpAndSettle();
    expect(find.text('Oyuncu profili'), findsOneWidget);
    expect(
      tester.widget<Text>(find.byKey(const Key('player-level'))).data,
      '7',
    );
    t.container.read(routerProvider).pop();
    await tester.pumpAndSettle();

    // A private profile shows no stats and the request state.
    await tester.tap(find.byKey(const Key('board-row-u-gizli')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('player-private')), findsOneWidget);
    expect(find.text('İstek gönderildi'), findsOneWidget);
    expect(find.byKey(const Key('player-add-friend')), findsNothing);
    await finish(tester, t);
  });

  testWidgets('quests: claim a completed one, reroll an open one', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.progress.questList = QuestList(
        daily: [quest('q1', progress: 3, done: true), quest('q2')],
        weekly: [quest('w1', target: 15)],
        rerollsLeft: 1,
      );
    });
    await tester.tap(find.byKey(const Key('open-quests')));
    await tester.pumpAndSettle();
    expect(find.text('Görev q1'), findsOneWidget);
    expect(find.text('Bugün kalan değiştirme: 1'), findsOneWidget);
    await tester.tap(find.byKey(const Key('quest-claim-q1')));
    await tester.pumpAndSettle();
    expect(t.progress.claimed, ['q1']);
    expect(find.text('+50 XP kazandın'), findsOneWidget);
    await tester.tap(find.byKey(const Key('quest-reroll-q2')));
    await tester.pumpAndSettle();
    expect(t.progress.rerolled, ['q2']);
    await finish(tester, t);
  });

  testWidgets('badges: earned and locked', (tester) async {
    final t = await launch(tester, (t) {
      t.progress.catalogue = [
        BadgeInfo(
          code: 'first_win',
          family: 'wins',
          tier: 1,
          name: 'İlk Galibiyet',
          description: 'İlk maçını kazan.',
          earnedAt: DateTime(2026),
        ),
        const BadgeInfo(
          code: 'wins_10',
          family: 'wins',
          tier: 2,
          name: '10 Galibiyet',
          description: '10 maç kazan.',
        ),
      ];
    });
    await tester.tap(find.byKey(const Key('open-badges')));
    await tester.pumpAndSettle();
    expect(find.text('İlk Galibiyet'), findsOneWidget);
    expect(find.textContaining('Henüz kazanılmadı'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('result screen merges both match.awards messages', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.matches.matchResult = MatchResult(
        id: 'm1',
        status: 'completed',
        origin: 'private_room',
        endedAt: DateTime.now(),
        scores: const [300, 200],
        winnerSide: 1,
        players: const [
          ResultPlayer(
            userId: 'user-guest-1',
            side: 1,
            displayName: 'Ben',
            avatarId: 'fox',
            correct: 3,
            status: 'finished',
          ),
          ResultPlayer(
            userId: 'opp',
            side: 2,
            displayName: 'Rakip',
            avatarId: 'owl',
            correct: 2,
            status: 'finished',
          ),
        ],
        rounds: const [],
      );
    });
    t.container.read(routerProvider).go(Routes.matchResult('m1'));
    await tester.pumpAndSettle();
    expect(find.text('Ödüller hesaplanıyor…'), findsOneWidget);
    // The feature under test, not the socket's lifecycle (D-097): start
    // with the socket open, as if a screen held a lease.
    t.container.read(realtimeClientProvider).start();
    await tester.pump();
    final s = t.sockets.last
      ..serverSends({
        't': 'sys.welcome',
        'd': {'session_id': 's1', 'resume_token': 'r1'},
      });
    await tester.pump();
    s
      ..serverSends({
        't': 'match.awards',
        'd': {
          'match_id': 'm1',
          'xp': 95,
          'level': 3,
          'level_up': true,
          'streak': 2,
          'badges': [
            {'code': 'first_win', 'name': 'İlk Galibiyet', 'tier': 1},
          ],
          'quests': [
            {'id': 'q', 'completed': true},
          ],
        },
      })
      ..serverSends({
        't': 'match.awards',
        'd': {'match_id': 'other', 'xp': 1},
      })
      ..serverSends({
        't': 'match.awards',
        'd': {
          'match_id': 'm1',
          'sp_delta': 35,
          'season_points': 335,
          'league': 'silver',
          'division': 3,
          'league_up': true,
        },
      });
    await tester.pump();
    await tester.pump();
    expect(find.text('+95 XP'), findsOneWidget);
    expect(find.text('Seviye atladın: 3!'), findsOneWidget);
    expect(find.text('+35 SP'), findsOneWidget);
    expect(find.text('Yeni lig: Gümüş III!'), findsOneWidget);
    expect(find.text('Yeni rozet: İlk Galibiyet'), findsOneWidget);
    expect(find.text('1 görev tamamlandı'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('result screen: "Play again" for both sides, never revenge '
      '(D-091)', (tester) async {
    final t = await launch(tester, (t) {
      t.matches.matchResult = MatchResult(
        id: 'm2',
        status: 'completed',
        origin: 'rematch',
        endedAt: DateTime.now(),
        scores: const [300, 100],
        winnerSide: 1,
        players: const [
          ResultPlayer(
            userId: 'user-guest-1',
            side: 1,
            displayName: 'Ben',
            avatarId: 'fox',
            correct: 3,
            status: 'finished',
          ),
          ResultPlayer(
            userId: 'opp',
            side: 2,
            displayName: 'Rakip',
            avatarId: 'owl',
            correct: 1,
            status: 'finished',
          ),
        ],
        rounds: const [],
      );
    });
    t.container.read(routerProvider).go(Routes.matchResult('m2'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('revenge')), findsNothing);
    expect(find.byKey(const Key('play-again')), findsOneWidget);
    expect(find.text('Tekrar Oyna'), findsOneWidget);
    await finish(tester, t);
  });
}
