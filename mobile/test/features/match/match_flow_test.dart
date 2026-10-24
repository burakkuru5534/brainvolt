import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
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

  /// Screens with countdowns never settle; advance a little instead.
  Future<void> step(WidgetTester tester) async {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
  }

  Future<TestApp> launch(WidgetTester tester) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: t.container,
        child: const BrainVoltApp(),
      ),
    );
    await tester.pumpAndSettle();
    return t;
  }

  /// Closes the socket so no reconnect or reauth timer outlives the test.
  Future<void> finish(WidgetTester tester, TestApp t) async {
    // stop() cancels its timers before its first await.
    unawaited(t.container.read(realtimeClientProvider).stop());
    await tester.pump();
  }

  /// Opens the queue and lets the server find a match.
  Future<FakeSocket> queueAndMatch(WidgetTester tester, TestApp t) async {
    await tester.tap(find.byKey(const Key('play-quick-duel')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.byKey(const Key('topic-all')));
    await step(tester);
    expect(find.byKey(const Key('queue-searching')), findsOneWidget);
    expect(t.matches.calls, ['enqueue']);
    final s = t.sockets.single;
    expect(s.sent.first['t'], 'sys.auth');
    s.serverSends({
      't': 'sys.welcome',
      'd': {'session_id': 's1', 'resume_token': 'r1'},
    });
    await step(tester);
    s.serverSends({
      't': 'mm.match_found',
      'd': {'match_id': 'm1'},
    });
    await step(tester);
    await step(tester);
    expect(s.lastSent('match.join')['d'], {'match_id': 'm1'});
    s
      ..ack('match.join')
      ..serverSends({
        't': 'match.snapshot',
        's': 'match:m1',
        'seq': 1,
        'd': {
          'status': 'waiting',
          'phase': 'lobby',
          'question_count': 7,
          'ready_deadline_at': now() + 20000,
          'participants': [
            {'user_id': me, 'side_no': 1, 'display_name': 'Ben'},
            {'user_id': 'opp', 'side_no': 2, 'display_name': 'Rakip'},
          ],
          'sides': [
            {'side_no': 1, 'score': 0},
            {'side_no': 2, 'score': 0},
          ],
        },
      });
    await step(tester);
    return s;
  }

  testWidgets('quick duel: queue, ready, question, answer, reveal, result', (
    tester,
  ) async {
    final t = await launch(tester);
    final s = await queueAndMatch(tester, t);
    expect(find.text('Maç bulundu — Hazır mısın?'), findsOneWidget);
    expect(find.text('Rakip'), findsOneWidget);

    await tester.tap(find.byKey(const Key('ready')));
    await step(tester);
    s.ack('match.ready');
    expect(find.text('Rakip bekleniyor…'), findsOneWidget);

    s.serverSends({
      't': 'match.starting',
      's': 'match:m1',
      'seq': 2,
      'd': {'starts_at': now() + 3000},
    });
    await step(tester);
    expect(find.byKey(const Key('countdown')), findsOneWidget);

    s
      ..serverSends({
        't': 'match.started',
        's': 'match:m1',
        'seq': 3,
        'd': <String, dynamic>{},
      })
      ..serverSends({
        't': 'round.started',
        's': 'match:m1',
        'seq': 4,
        'd': {
          'round_no': 1,
          'question': {
            'id': 'q1',
            'prompt': 'Türkiye’nin başkenti neresidir?',
            'category': {'id': 'c1', 'name': 'Coğrafya'},
          },
          'options': [
            {'id': 'o1', 'text': 'Ankara'},
            {'id': 'o2', 'text': 'İzmir'},
          ],
          'show_at': now() - 100,
          'deadline_at': now() + 15000,
        },
      });
    await step(tester);
    expect(find.text('Türkiye’nin başkenti neresidir?'), findsOneWidget);
    expect(find.text('Soru 1/7'), findsOneWidget);
    expect(find.text('Coğrafya'), findsOneWidget);

    // Quick chat: the opponent's phrase shows; ours goes out.
    s.serverSends({
      't': 'match.quick_chat',
      's': 'match:m1',
      'd': {'match_id': 'm1', 'user_id': 'opp', 'phrase_id': 'gg'},
    });
    await step(tester);
    expect(find.text('Rakip: GG'), findsOneWidget);
    await tester.tap(find.byKey(const Key('match-quick')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('match-quick-good_luck')));
    await step(tester);
    await step(tester);
    expect(s.lastSent('match.quick_chat')['d'], {
      'match_id': 'm1',
      'phrase_id': 'good_luck',
    });
    s.ack('match.quick_chat');
    await step(tester);

    await tester.tap(find.byKey(const Key('option-o1')));
    await step(tester);
    expect(s.lastSent('match.answer')['d'], {
      'match_id': 'm1',
      'round_no': 1,
      'option_id': 'o1',
    });
    s
      ..ack('match.answer')
      ..serverSends({
        't': 'round.answer_received',
        's': 'match:m1',
        'seq': 5,
        'd': {'round_no': 1, 'user_id': me, 'option_id': 'o1'},
      })
      ..serverSends({
        't': 'round.answer_received',
        's': 'match:m1',
        'seq': 6,
        'd': {'round_no': 1, 'user_id': 'opp'},
      });
    await step(tester);
    expect(find.byKey(const Key('opponent-answered')), findsOneWidget);

    s.serverSends({
      't': 'round.revealed',
      's': 'match:m1',
      'seq': 7,
      'd': {
        'round_no': 1,
        'correct_option_id': 'o1',
        'category': {'id': 'c1', 'name': 'Coğrafya'},
        'answers': [
          {
            'user_id': me,
            'option_id': 'o1',
            'correct': true,
            'points': {'total': 180},
          },
          {
            'user_id': 'opp',
            'option_id': 'o2',
            'correct': false,
            'points': {'total': 0},
          },
        ],
        'sides': [
          {'side_no': 1, 'score': 180},
          {'side_no': 2, 'score': 0},
        ],
      },
    });
    await step(tester);
    expect(find.text('Doğru!'), findsOneWidget);
    expect(find.text('+180 puan'), findsOneWidget);
    expect(tester.widget<Text>(find.byKey(const Key('my-score'))).data, '180');

    s.serverSends({
      't': 'match.completed',
      's': 'match:m1',
      'seq': 8,
      'd': {
        'end_reason': 'all_rounds_played',
        'winner_side_no': 1,
        'is_draw': false,
        'sides': [
          {'side_no': 1, 'score': 1200},
          {'side_no': 2, 'score': 640},
        ],
      },
    });
    await step(tester);
    expect(find.text('Kazandın!'), findsOneWidget);
    expect(find.text('1200 — 640'), findsOneWidget);

    await tester.tap(find.byKey(const Key('back-home')));
    await step(tester);
    expect(find.byKey(const Key('play-quick-duel')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('Quick Duel asks for the topic and queues in it (D-095)', (
    tester,
  ) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('play-quick-duel')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.text('Hangi konuda yarışmak istersin?'), findsOneWidget);
    await tester.tap(find.byKey(const Key('topic-cat-history')));
    await step(tester);
    await step(tester);
    expect(t.matches.calls, [
      'enqueue:quick_duel:1v1:category_master:cat-history::null',
    ]);
    expect(find.text('Konu: Tarih'), findsOneWidget);
    await tester.tap(find.byKey(const Key('queue-cancel')));
    await step(tester);
    await finish(tester, t);
  });

  testWidgets('cancelling the search leaves the queue', (tester) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('play-quick-duel')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.byKey(const Key('topic-all')));
    await step(tester);
    await tester.tap(find.byKey(const Key('queue-cancel')));
    await step(tester);
    expect(t.matches.calls, ['enqueue', 'leaveQueue']);
    expect(find.byKey(const Key('play-quick-duel')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('leaving a match asks first, then sends match.leave', (
    tester,
  ) async {
    final t = await launch(tester);
    final s = await queueAndMatch(tester, t);
    await tester.tap(find.byKey(const Key('match-leave')));
    await step(tester);
    expect(find.text('Maçtan ayrılmak istiyor musun?'), findsOneWidget);
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await step(tester);
    s.ack('match.leave');
    await step(tester);
    expect(s.lastSent('match.leave')['d'], {'match_id': 'm1'});
    expect(find.byKey(const Key('play-quick-duel')), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('a live match on start offers to go back to it', (tester) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final t = await TestApp.create();
    t.matches.current = 'm9';
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: t.container,
        child: const BrainVoltApp(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('resume-match')), findsOneWidget);
    expect(find.byKey(const Key('play-practice')), findsOneWidget);
  });

  /// Ends the matched game with [winner] ahead.
  Future<void> complete(WidgetTester tester, FakeSocket s, int winner) async {
    s.serverSends({
      't': 'match.completed',
      's': 'match:m1',
      'seq': 2,
      'd': {
        'end_reason': 'all_rounds_played',
        'winner_side_no': winner,
        'is_draw': false,
        'sides': [
          {'side_no': 1, 'score': winner == 1 ? 900 : 300},
          {'side_no': 2, 'score': winner == 2 ? 900 : 300},
        ],
      },
    });
    await step(tester);
    await step(tester);
  }

  MatchResult result() => MatchResult(
    id: 'm1',
    status: 'completed',
    origin: 'matchmaking',
    endedAt: DateTime.now(),
    scores: const [300, 900],
    winnerSide: 2,
    players: const [],
    rounds: const [],
  );

  testWidgets('the loser offers "Play again" on the end screen and is taken '
      'to the new match once everyone accepted (D-091)', (tester) async {
    final t = await launch(tester);
    t.matches.matchResult = result();
    final s = await queueAndMatch(tester, t);
    await tester.tap(find.byKey(const Key('ready')));
    await step(tester);
    await complete(tester, s, 2);
    expect(find.text('Kaybettin'), findsOneWidget);
    expect(find.byKey(const Key('revenge')), findsNothing);
    expect(find.text('Tekrar Oyna'), findsOneWidget);
    await tester.tap(find.byKey(const Key('rematch')));
    await step(tester);
    expect(s.lastSent('rematch.respond')['d'], {
      'match_id': 'm1',
      'accept': true,
    });
    s.ack('rematch.respond');
    await step(tester);
    expect(find.byKey(const Key('rematch-waiting')), findsOneWidget);

    s.serverSends({
      't': 'rematch.result',
      'd': {'match_id': 'm1', 'accepted': true, 'new_match_id': 'm9'},
    });
    await step(tester);
    await step(tester);
    expect(s.lastSent('match.join')['d'], {'match_id': 'm9'});
    // A fresh screen: the new match asks for ready again.
    s
      ..ack('match.join')
      ..serverSends({
        't': 'match.snapshot',
        's': 'match:m9',
        'seq': 1,
        'd': {
          'status': 'waiting',
          'phase': 'lobby',
          'question_count': 7,
          'ready_deadline_at': now() + 20000,
          'participants': [
            {'user_id': me, 'side_no': 1, 'display_name': 'Ben'},
            {'user_id': 'opp', 'side_no': 2, 'display_name': 'Rakip'},
          ],
          'sides': [
            {'side_no': 1, 'score': 0},
            {'side_no': 2, 'score': 0},
          ],
        },
      });
    await step(tester);
    await tester.tap(find.byKey(const Key('ready')));
    await step(tester);
    expect(s.lastSent('match.ready')['d'], {'match_id': 'm9'});
    await finish(tester, t);
  });

  testWidgets('the winner sees the "Play again" offer arrive and accepts it '
      '(D-091)', (tester) async {
    final t = await launch(tester);
    t.matches.matchResult = result();
    final s = await queueAndMatch(tester, t);
    await complete(tester, s, 1);
    expect(find.byKey(const Key('rematch-offered')), findsNothing);
    expect(find.byKey(const Key('rematch')), findsOneWidget);

    s.serverSends({
      't': 'rematch.offer',
      'd': {
        'match_id': 'm1',
        'accepted_user_ids': ['opp'],
        'expires_at': now() + 60000,
      },
    });
    await step(tester);
    expect(find.text('Tekrar oynama teklifi geldi!'), findsOneWidget);
    expect(find.byKey(const Key('rematch-decline')), findsOneWidget);
    await tester.tap(find.byKey(const Key('rematch')));
    await step(tester);
    expect(s.lastSent('rematch.respond')['d'], {
      'match_id': 'm1',
      'accept': true,
    });
    s
      ..ack('rematch.respond')
      ..serverSends({
        't': 'rematch.result',
        'd': {'match_id': 'm1', 'accepted': true, 'new_match_id': 'm9'},
      });
    await step(tester);
    await step(tester);
    expect(s.lastSent('match.join')['d'], {'match_id': 'm9'});
    await finish(tester, t);
  });
}
