import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/core/widgets/bv.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
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

  /// The socket, welcomed. These tests are about rooms and matches, not the
  /// socket's lifecycle (D-097), so they start with it open, as if a
  /// screen held a lease.
  Future<FakeSocket> welcome(WidgetTester tester, TestApp t) async {
    t.container.read(realtimeClientProvider).start();
    await tester.pump();
    return t.sockets.single..serverSends({
      't': 'sys.welcome',
      'd': {'session_id': 's1', 'resume_token': 'r1'},
    });
  }

  testWidgets('mode picker: Category Master needs a category', (tester) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('play-modes')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('type-sudden_death')));
    await tester.tap(find.byKey(const Key('mode-category_master')));
    await tester.pumpAndSettle();
    final search = find.byKey(const Key('mode-search'));
    expect(tester.widget<GradientButton>(search).onPressed, isNull);
    await tester.tap(find.byKey(const Key('category')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Tarih').last);
    await tester.pumpAndSettle();
    await tester.tap(search);
    await step(tester);
    expect(t.matches.calls, [
      'enqueue:sudden_death:1v1:category_master:cat-history::null',
    ]);
    await tester.tap(find.byKey(const Key('queue-cancel')));
    await step(tester);
    await finish(tester, t);
  });

  testWidgets('mode picker: a topic queues Category Master on it (D-095)', (
    tester,
  ) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('play-modes')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('topic-chip-cat-science')));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.byKey(const Key('mode-search')));
    await tester.tap(find.byKey(const Key('mode-search')));
    await step(tester);
    expect(t.matches.calls, [
      'enqueue:quick_duel:1v1:category_master:cat-science::null',
    ]);
    await tester.tap(find.byKey(const Key('queue-cancel')));
    await step(tester);
    await finish(tester, t);
  });

  testWidgets('mode picker: a 2v2 queue with a friend in the party', (
    tester,
  ) async {
    final t = await launch(
      tester,
      setup: (t) => t.social.friendList = [
        const Friend(UserCard(userId: 'u-ali', displayName: 'Ali')),
      ],
    );
    await tester.tap(find.byKey(const Key('play-modes')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('format-2v2')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Arkadaşlarını seç'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('party-u-ali')));
    await tester.tap(find.byKey(const Key('party-ok')));
    await tester.pumpAndSettle();
    expect(find.text('Ali'), findsOneWidget);
    await tester.ensureVisible(find.byKey(const Key('mode-search')));
    await tester.tap(find.byKey(const Key('mode-search')));
    await step(tester);
    expect(t.matches.calls, ['enqueue:quick_duel:2v2:null:null:u-ali:null']);
    await tester.tap(find.byKey(const Key('queue-cancel')));
    await step(tester);
    await finish(tester, t);
  });

  testWidgets('Mixed Battle: the picker chooses the category', (tester) async {
    final t = await launch(tester);
    final s = await welcome(tester, t);
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
    final s = await welcome(tester, t);
    await step(tester);
    await tester.tap(find.byKey(const Key('private-room')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-room-quick_duel')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('format-ok')));
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
    // D-093: the host changes the room to Marathon 2v2, Shared Answer.
    await tester.tap(find.byKey(const Key('room-settings')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('match-type-marathon')));
    await tester.tap(find.byKey(const Key('format-2v2')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('team-mode-shared_answer')));
    // D-095: the host picks the room's topic.
    await tester.tap(find.byKey(const Key('topic-chip-cat-history')));
    await tester.tap(find.byKey(const Key('format-ok')));
    await tester.pumpAndSettle();
    expect(
      t.lobby.calls.last,
      'updateRoom:r1:marathon:2v2:shared_answer:cat-history:v2',
    );
    expect(
      find.text('Oda ayarları güncellendi, herkes tekrar hazır olmalı'),
      findsOneWidget,
    );
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

    // D-092: the match ends and everyone goes back to the room lobby.
    t.matches.matchResult = MatchResult(
      id: 'm5',
      status: 'completed',
      origin: 'private_room',
      endedAt: DateTime.now(),
      scores: const [900, 300],
      winnerSide: 1,
      players: const [],
      rounds: const [],
    );
    s
      ..ack('match.join')
      ..serverSends({
        't': 'match.snapshot',
        's': 'match:m5',
        'seq': 1,
        'd': {
          'status': 'active',
          'phase': 'lobby',
          'question_count': 7,
          'participants': [
            {'user_id': me, 'side_no': 1, 'display_name': 'Ben'},
            {'user_id': 'opp', 'side_no': 2, 'display_name': 'Ayşe'},
          ],
          'sides': [
            {'side_no': 1, 'score': 0},
            {'side_no': 2, 'score': 0},
          ],
        },
      })
      ..serverSends({
        't': 'match.completed',
        's': 'match:m5',
        'seq': 2,
        'd': {
          'end_reason': 'all_rounds_played',
          'winner_side_no': 1,
          'is_draw': false,
          'sides': [
            {'side_no': 1, 'score': 900},
            {'side_no': 2, 'score': 300},
          ],
        },
      });
    await step(tester);
    await step(tester);
    expect(find.byKey(const Key('room-returning')), findsOneWidget);
    expect(find.byKey(const Key('play-again')), findsNothing);
    for (var i = 0; i < 6; i++) {
      await tester.pump(const Duration(seconds: 1));
    }
    await step(tester);
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.room('r1'),
    );
    // The room is open again (seq 4) with the ready flags cleared.
    final reopened = state(ready: false)..['seq'] = 4;
    ((reopened['d'] as Map)['room'] as Map)['version'] = 4;
    s
      ..ack('room.join')
      ..serverSends(reopened);
    await step(tester);
    // Ready flags were cleared: the host waits for the others again.
    expect(
      tester
          .widget<ButtonStyleButton>(find.byKey(const Key('room-start')))
          .onPressed,
      isNull,
    );
    await finish(tester, t);
  });

  testWidgets('private room: joining by code shows the room', (tester) async {
    final t = await launch(tester);
    final s = await welcome(tester, t);
    await step(tester);
    final joined = {
      't': 'room.state',
      's': 'room:r1',
      'seq': 2,
      'd': {
        'room': {
          'id': 'r1',
          'status': 'open',
          'version': 2,
          'host_user_id': 'host',
          'code': 'ROOM23',
          'capacity': 2,
          'match_config': {'match_type': 'quick_duel', 'format': '1v1'},
          'members': [
            {
              'user_id': 'host',
              'display_name': 'Kurucu Ali',
              'side_no': 1,
              'seat_no': 1,
              'is_host': true,
            },
            {'user_id': me, 'display_name': 'Ben', 'side_no': 2, 'seat_no': 1},
          ],
        },
      },
    };
    await tester.tap(find.byKey(const Key('private-room')));
    await tester.pumpAndSettle();
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('room-code-input')), 'room23');
    // The join broadcast arrives while the home screen is still open.
    s.serverSends(joined);
    await tester.tap(find.byKey(const Key('room-code-join')));
    await step(tester);
    await step(tester);
    expect(t.lobby.calls, ['joinRoom:room23']);
    // The room screen asks again and gets the same version.
    s
      ..ack('room.join')
      ..serverSends(joined);
    await step(tester);
    expect(find.text('Kurucu Ali'), findsOneWidget);
    expect(find.byKey(const Key('room-ready')), findsOneWidget);
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

  testWidgets('team queue: waiting lobby fills, everyone readies, the leader '
      'starts (D-094)', (tester) async {
    final t = await launch(tester);
    final s = await welcome(tester, t);
    await step(tester);
    await tester.tap(find.byKey(const Key('play-quick-duel')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.byKey(const Key('topic-all')));
    await step(tester);
    s.serverSends({
      't': 'mm.lobby',
      'd': {'ticket_id': 't1', 'room_id': 'r7'},
    });
    await step(tester);
    await step(tester);
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.room('r7'),
    );
    expect(s.lastSent('room.join')['d'], {'room_id': 'r7'});
    Map<String, dynamic> lobby(int seq, {String? leader, bool ready = false}) {
      Map<String, dynamic> m(String id, String name, int side, int seat) => {
        'user_id': id,
        'display_name': name,
        'side_no': side,
        'seat_no': seat,
        'ready': id == me ? ready : true,
      };
      return {
        't': 'room.state',
        's': 'room:r7',
        'seq': seq,
        'd': {
          'room': {
            'id': 'r7',
            'kind': 'matchmaking',
            'status': 'open',
            'version': seq,
            'host_user_id': '00000000-0000-0000-0000-000000000000',
            'code': '',
            'capacity': 4,
            'leader_user_id': ?leader,
            'match_config': {
              'match_type': 'quick_duel',
              'format': '2v2',
              'team_answer_mode': 'fast_team_round',
            },
            'members': [
              m(me, 'Ben', 1, 1),
              m('u2', 'Ali', 1, 2),
              m('u3', 'Ece', 2, 1),
              if (leader != null) m('u4', 'Can', 2, 2),
            ],
          },
        },
      };
    }

    s
      ..ack('room.join')
      ..serverSends(lobby(1));
    await step(tester);
    expect(find.text('Eşleşme Lobisi'), findsOneWidget);
    expect(find.text('Oyuncular aranıyor… (3/4)'), findsOneWidget);
    expect(find.byKey(const Key('room-code')), findsNothing);
    expect(find.byKey(const Key('room-start')), findsNothing);

    // Full: the drawn leader is this player; the others are ready.
    s.serverSends(lobby(2, leader: me));
    await step(tester);
    expect(find.byKey(const Key('mm-lobby-you-lead')), findsOneWidget);
    final start = find.byKey(const Key('room-start'));
    expect(tester.widget<ButtonStyleButton>(start).onPressed, isNull);
    await tester.tap(find.byKey(const Key('room-ready')));
    await step(tester);
    expect(s.lastSent('room.ready')['d'], {'room_id': 'r7'});
    s
      ..ack('room.ready')
      ..serverSends(lobby(3, leader: me, ready: true));
    await step(tester);
    await tester.tap(start);
    await step(tester);
    expect(s.lastSent('room.start')['d'], {'room_id': 'r7'});
    s
      ..ack('room.start')
      ..serverSends({
        't': 'room.match_starting',
        's': 'room:r7',
        'd': {'room_id': 'r7', 'match_id': 'm7'},
      })
      ..serverSends({
        't': 'room.closed',
        's': 'room:r7',
        'seq': 4,
        'd': {'room_id': 'r7', 'reason': 'match_started'},
      });
    await step(tester);
    await step(tester);
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.match('m7'),
    );
    await finish(tester, t);
  });
}
