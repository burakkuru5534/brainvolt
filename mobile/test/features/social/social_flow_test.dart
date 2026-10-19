import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/network/api_exception.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/lobby/domain/lobby_repository.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/social/domain/social_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

const ali = UserCard(
  userId: 'u-ali',
  username: 'ali',
  displayName: 'Ali',
  avatarId: 'fox',
);
const me = UserCard(userId: 'user-guest-1', displayName: 'Ben');

void main() {
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  Future<void> step(WidgetTester tester) async {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
  }

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

  /// Stops the socket and lets snack bars time out.
  Future<void> finish(WidgetTester tester, TestApp t) async {
    unawaited(t.container.read(realtimeClientProvider).stop());
    await tester.pump();
    await tester.pump(const Duration(seconds: 5));
  }

  void go(TestApp t, String location) =>
      t.container.read(routerProvider).go(location);

  testWidgets('friends: list with presence, add by code, challenge', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.social.friendList = [const Friend(ali, online: true)];
      t.lobby.resolved = const ResolvedCode(kind: 'friend', user: ali);
    });
    await tester.tap(find.byKey(const Key('open-friends')));
    await tester.pumpAndSettle();
    expect(find.text('Ali'), findsOneWidget);
    expect(find.text('@ali · Çevrimiçi'), findsOneWidget);
    expect(find.text('Arkadaş kodun: ABC234'), findsOneWidget);

    // Add by friend code: resolve, then a request via invite_code.
    await tester.tap(find.byKey(const Key('add-friend')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Arkadaş kodu'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('add-friend-input')), 'xyz789');
    await tester.tap(find.byKey(const Key('add-friend-send')));
    await tester.pumpAndSettle();
    expect(t.lobby.calls, ['resolve:xyz789']);
    expect(t.social.calls, ['send:u-ali:invite_code']);
    expect(find.text('Arkadaşlık isteği gönderildi.'), findsOneWidget);

    // Challenge from the friend menu.
    await tester.tap(find.byKey(const Key('friend-menu-u-ali')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Meydan oku'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('challenge-sudden_death')));
    await tester.pumpAndSettle();
    expect(t.lobby.calls.last, 'challenge:u-ali:sudden_death');
    expect(find.text('Meydan okuma gönderildi.'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('friends: guests are asked to link their account', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.social.failure = const ApiException(
        code: 'ACCOUNT_UPGRADE_REQUIRED',
        status: 403,
      );
    });
    go(t, Routes.friends);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('friends-upgrade')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('challenges: a received one is announced and accepted', (
    tester,
  ) async {
    final c = Challenge(
      id: 'c1',
      status: 'pending',
      challenger: ali,
      target: me,
      matchType: 'quick_duel',
      expiresAt: DateTime(2030),
    );
    final t = await launch(tester, (t) {
      t.lobby
        ..incoming = [c]
        ..responded = Challenge(
          id: 'c1',
          status: 'started',
          challenger: ali,
          target: me,
          matchType: 'quick_duel',
          expiresAt: DateTime(2030),
          matchId: 'm9',
        );
    });
    final s = t.sockets.single;
    s.serverSends({
      't': 'sys.welcome',
      'd': {'session_id': 's1', 'resume_token': 'r1'},
    });
    await step(tester);
    s.serverSends({
      't': 'challenge.received',
      'd': {
        'challenge': {
          'id': 'c1',
          'challenger': {'user_id': 'u-ali', 'display_name': 'Ali'},
        },
      },
    });
    await step(tester);
    expect(find.text('Ali sana meydan okudu!'), findsOneWidget);
    await tester.pump(const Duration(seconds: 1));
    await tester.tap(find.text('Gör'));
    await tester.pumpAndSettle();
    expect(find.text('Ali sana meydan okudu'), findsOneWidget);
    expect(find.text('Hızlı Düello'), findsOneWidget);
    await tester.tap(find.byKey(const Key('accept-challenge-c1')));
    await step(tester);
    await step(tester);
    expect(t.lobby.calls, ['accept:c1']);
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.match('m9'),
    );
    await finish(tester, t);
  });

  testWidgets('result: rematch asked, accepted, opens the new match', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.matches.matchResult = MatchResult(
        id: 'm1',
        status: 'completed',
        origin: 'matchmaking',
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
    final s = t.sockets.single;
    s.serverSends({
      't': 'sys.welcome',
      'd': {'session_id': 's1', 'resume_token': 'r1'},
    });
    go(t, Routes.matchResult('m1'));
    await step(tester);
    await step(tester);
    s.serverSends({
      't': 'rematch.offer',
      'd': {
        'match_id': 'm1',
        'accepted_user_ids': ['opp'],
      },
    });
    await step(tester);
    expect(find.text('Rakibin rövanş istiyor!'), findsOneWidget);
    await tester.tap(find.byKey(const Key('rematch')));
    await step(tester);
    expect(s.lastSent('rematch.respond')['d'], {
      'match_id': 'm1',
      'accept': true,
    });
    s.ack('rematch.respond');
    expect(find.byKey(const Key('rematch-waiting')), findsOneWidget);
    s.serverSends({
      't': 'rematch.result',
      'd': {'match_id': 'm1', 'accepted': true, 'new_match_id': 'm2'},
    });
    await step(tester);
    await step(tester);
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.match('m2'),
    );
    await finish(tester, t);
  });
}
