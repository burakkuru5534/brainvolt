import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/core/widgets/bv.dart';
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

  Future<(TestApp, FakeSocket)> openMatch(
    WidgetTester tester,
    String teamMode,
    List<String> answering,
  ) async {
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
    final s = t.sockets.single
      ..serverSends({
        't': 'sys.welcome',
        'd': {'session_id': 's1', 'resume_token': 'r1'},
      });
    await step(tester);
    t.container.read(routerProvider).go(Routes.match('m1'));
    await step(tester);
    s
      ..ack('match.join')
      ..serverSends({
        't': 'match.snapshot',
        's': 'match:m1',
        'seq': 10,
        'd': {
          'status': 'in_progress',
          'phase': 'question',
          'match_type': 'quick_duel',
          'format': '2v2',
          'team_answer_mode': teamMode,
          'question_count': 7,
          'participants': [
            {'user_id': me, 'side_no': 1, 'display_name': 'Ben'},
            {'user_id': 'mate', 'side_no': 1, 'display_name': 'Ayşe'},
            {'user_id': 'o1', 'side_no': 2, 'display_name': 'Rakip 1'},
            {'user_id': 'o2', 'side_no': 2, 'display_name': 'Rakip 2'},
          ],
          'sides': [
            {'side_no': 1, 'score': 0},
            {'side_no': 2, 'score': 0},
          ],
          'round': {
            'round_no': 1,
            'question': {'id': 'q1', 'prompt': 'Soru?'},
            'options': [
              {'id': 'a', 'text': 'A'},
              {'id': 'b', 'text': 'B'},
            ],
            'show_at': now() - 1000,
            'deadline_at': now() + 15000,
            'answering_user_ids': answering,
            'proposals': <Object>[],
          },
        },
      });
    await step(tester);
    return (t, s);
  }

  Future<void> finish(WidgetTester tester, TestApp t) async {
    unawaited(t.container.read(realtimeClientProvider).stop());
    await tester.pump();
    await tester.pump(const Duration(seconds: 5));
  }

  testWidgets('Shared Answer: propose, see a teammate, lock, side reveal', (
    tester,
  ) async {
    final (t, s) = await openMatch(tester, 'shared_answer', [
      me,
      'mate',
      'o1',
      'o2',
    ]);
    expect(find.text('Takımın'), findsOneWidget);
    final lock = find.byKey(const Key('lock'));
    expect(tester.widget<ButtonStyleButton>(lock).onPressed, isNull);

    await tester.tap(find.byKey(const Key('option-a')));
    await step(tester);
    expect(s.lastSent('match.propose')['d'], {
      'match_id': 'm1',
      'round_no': 1,
      'option_id': 'a',
    });
    s
      ..ack('match.propose')
      ..serverSends({
        't': 'round.proposals',
        's': 'match:m1',
        'seq': 11,
        'd': {
          'round_no': 1,
          'proposals': [
            {'user_id': me, 'option_id': 'a'},
            {'user_id': 'mate', 'option_id': 'b'},
          ],
        },
      });
    await step(tester);
    expect(find.text('Ayşe'), findsOneWidget); // under option B
    await tester.tap(lock);
    await step(tester);
    expect(s.lastSent('match.lock')['d'], {
      'match_id': 'm1',
      'round_no': 1,
      'option_id': 'a',
    });
    s
      ..ack('match.lock')
      ..serverSends({
        't': 'round.answer_received',
        's': 'match:m1',
        'seq': 12,
        'd': {
          'round_no': 1,
          'user_id': me,
          'side_no': 1,
          'option_id': 'a',
          'locked': true,
        },
      })
      ..serverSends({
        't': 'round.revealed',
        's': 'match:m1',
        'seq': 13,
        'd': {
          'round_no': 1,
          'correct_option_id': 'a',
          'answers': [
            {
              'user_id': me,
              'option_id': 'a',
              'correct': true,
              'points': {'total': 150},
            },
          ],
          'sides': [
            {'side_no': 1, 'correct': true, 'score': 150},
            {'side_no': 2, 'correct': false, 'score': 0},
          ],
        },
      });
    await step(tester);
    expect(find.text('Doğru!'), findsOneWidget);
    expect(find.text('150'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('Individual Turns: only the turn holder answers', (tester) async {
    final (t, _) = await openMatch(tester, 'individual_turns', ['mate', 'o1']);
    expect(find.text('Sıra: Ayşe'), findsOneWidget);
    expect(
      tester.widget<OptionTile>(find.byKey(const Key('option-a'))).onPressed,
      isNull,
    );
    await finish(tester, t);
  });
}
