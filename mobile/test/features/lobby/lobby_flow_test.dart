import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/social/domain/social_repository.dart';
import 'package:brainvolt/features/team/domain/team_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

const me = 'user-guest-1';

void main() {
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  int now() => DateTime.now().millisecondsSinceEpoch;

  Future<void> step(WidgetTester tester) async {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
  }

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

  FakeSocket welcome(TestApp t) => t.sockets.single
    ..serverSends({
      't': 'sys.welcome',
      'd': {'session_id': 's1', 'resume_token': 'r1'},
    });

  testWidgets('mode picker: Category Master needs a category', (tester) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('play-modes')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('type-sudden_death')));
    await tester.tap(find.byKey(const Key('mode-category_master')));
    await tester.pumpAndSettle();
    final search = find.byKey(const Key('mode-search'));
    expect(tester.widget<ButtonStyleButton>(search).onPressed, isNull);
    await tester.tap(find.byKey(const Key('category')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Tarih').last);
    await tester.pumpAndSettle();
    await tester.tap(search);
    await step(tester);
    expect(t.matches.calls, [
      'enqueue:sudden_death:category_master:cat-history',
    ]);
    await tester.tap(find.byKey(const Key('queue-cancel')));
    await step(tester);
    await finish(tester, t);
  });

  testWidgets('Mixed Battle: the picker chooses the category', (tester) async {
    final t = await launch(tester);
    final s = welcome(t);
    await step(tester);
    t.container.read(routerProvider).go(Routes.match('m1'));
    await step(tester);
    s
      ..ack('match.join')
      ..serverSends({
        't': 'match.snapshot',
        's': 'match:m1',
        'seq': 5,
        'd': {
          'status': 'in_progress',
          'phase': 'pre_round',
          'question_count': 7,
          'participants': [
            {'user_id': me, 'side_no': 1, 'display_name': 'Ben'},
            {'user_id': 'opp', 'side_no': 2, 'display_name': 'Rakip'},
          ],
          'sides': [
            {'side_no': 1, 'score': 0},
            {'side_no': 2, 'score': 0},
          ],
          'pre_round': {
            'round_no': 1,
            'picker_user_id': me,
            'categories': [
              {'id': 'c1', 'name': 'Tarih'},
              {'id': 'c2', 'name': 'Bilim'},
              {'id': 'c3', 'name': 'Spor'},
            ],
            'allowed_category_ids': ['c1', 'c2'],
            'pick_deadline_at': now() + 8000,
          },
        },
      });
    await step(tester);
    expect(find.text('Bir kategori seç'), findsOneWidget);
    final spor = find.byKey(const Key('pick-c3'));
    expect(tester.widget<ButtonStyleButton>(spor).onPressed, isNull);
    await tester.tap(find.byKey(const Key('pick-c2')));
    await step(tester);
    expect(s.lastSent('match.pick_category')['d'], {
      'match_id': 'm1',
      'round_no': 1,
      'category_id': 'c2',
    });
    s
      ..ack('match.pick_category')
      ..serverSends({
        't': 'round.category_picked',
        's': 'match:m1',
        'seq': 6,
        'd': {'round_no': 1, 'category_id': 'c2'},
      });
    await step(tester);
    expect(find.byKey(const Key('pre-round')), findsNothing);
    await finish(tester, t);
  });

  testWidgets('private room: create, the guest readies, host starts', (
    tester,
  ) async {
    final t = await launch(tester);
    final s = welcome(t);
    await step(tester);
    await tester.tap(find.byKey(const Key('private-room')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-room-quick_duel')));
    await step(tester);
    await step(tester);
    expect(t.lobby.calls, ['createRoom:quick_duel']);
    expect(s.lastSent('room.join')['d'], {'room_id': 'r1'});
    Map<String, dynamic> state({required bool ready}) => {
      't': 'room.state',
      's': 'room:r1',
      'seq': ready ? 3 : 2,
      'd': {
        'room': {
          'id': 'r1',
          'status': 'open',
          'version': ready ? 3 : 2,
          'host_user_id': me,
          'code': 'ROOM23',
          'capacity': 2,
          'match_config': {'match_type': 'quick_duel', 'format': '1v1'},
          'members': [
            {
              'user_id': me,
              'display_name': 'Ben',
              'side_no': 1,
              'seat_no': 1,
              'is_host': true,
            },
            {
              'user_id': 'opp',
              'display_name': 'Ayşe',
              'side_no': 2,
              'seat_no': 1,
              'ready': ready,
            },
          ],
        },
      },
    };
    s
      ..ack('room.join')
      ..serverSends(state(ready: false));
    await step(tester);
    expect(find.text('Oda kodu: ROOM23'), findsOneWidget);
    expect(find.text('Ayşe'), findsOneWidget);
    final start = find.byKey(const Key('room-start'));
    expect(tester.widget<ButtonStyleButton>(start).onPressed, isNull);
    s.serverSends(state(ready: true));
    await step(tester);
    await tester.tap(start);
    await step(tester);
    s
      ..ack('room.start')
      ..serverSends({
        't': 'room.match_starting',
        's': 'room:r1',
        'd': {'room_id': 'r1', 'match_id': 'm5'},
      });
    await step(tester);
    await step(tester);
    expect(s.lastSent('match.ready')['d'], {'match_id': 'm5'});
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.match('m5'),
    );
    await finish(tester, t);
  });

  testWidgets('teams: guests are asked to link their account', (tester) async {
    final t = await launch(tester);
    t.container.read(routerProvider).go(Routes.teams);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('teams-upgrade')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('teams: accept an invite, create a team, ask for a couple', (
    tester,
  ) async {
    final t = await launch(
      tester,
      user: const AuthUser(
        uid: 'full-1',
        isAnonymous: false,
        email: 'a@b.c',
        emailVerified: true,
      ),
      setup: (t) {
        t.social.friendList = [
          const Friend(UserCard(userId: 'u-ali', displayName: 'Ali')),
        ];
        t.teams.pendingInvites = [
          const TeamInvite(
            id: 'i1',
            teamName: 'Volt',
            teamTag: 'BVX',
            emblemId: 'bolt',
            invitedBy: UserCard(userId: 'u-ali', displayName: 'Ali'),
          ),
        ];
      },
    );
    await tester.tap(find.byKey(const Key('open-teams')));
    await tester.pumpAndSettle();
    expect(find.text('[BVX] Volt'), findsOneWidget);
    await tester.tap(find.byKey(const Key('accept-invite-i1')));
    await tester.pumpAndSettle();
    expect(t.teams.calls, ['accept:i1']);

    await tester.tap(find.byKey(const Key('create-team')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('team-name')), 'Beyin Takımı');
    await tester.enterText(find.byKey(const Key('team-tag')), 'BTK');
    await tester.tap(find.byKey(const Key('team-create')));
    await tester.pumpAndSettle();
    expect(t.teams.calls.last, 'create:Beyin Takımı:BTK:bolt');
    expect(find.text('[BTK] Beyin Takımı'), findsOneWidget);

    await tester.tap(find.byKey(const Key('request-couple')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('pick-friend-u-ali')));
    await tester.pumpAndSettle();
    expect(t.teams.calls.last, 'couple:u-ali');
    expect(find.text('Onay bekleniyor'), findsOneWidget);
    await finish(tester, t);
  });
}
