import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/daily/domain/daily_repository.dart';
import 'package:brainvolt/features/social/domain/social_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

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

  testWidgets('daily: start, answer over the socket, result', (tester) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('open-daily')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('daily-start')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(t.daily.started, ['official']);

    final s = t.sockets.last
      ..serverSends({
        't': 'sys.welcome',
        'd': {'session_id': 's1', 'resume_token': 'r1'},
      });
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    expect(s.lastSent('daily.join')['d'], {'run_id': 'run-official'});
    s.ack('daily.join');
    final now = DateTime.now().millisecondsSinceEpoch;
    s.serverSends({
      't': 'daily.question',
      'd': {
        'run_id': 'run-official',
        'kind': 'official',
        'position': 1,
        'total': 10,
        'show_at': now - 100,
        'deadline': now + 15000,
        'prompt': 'Türkiye’nin başkenti?',
        'category': 'Coğrafya',
        'options': [
          {'id': 'o1', 'text': 'Ankara'},
          {'id': 'o2', 'text': 'İstanbul'},
        ],
      },
    });
    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text('Türkiye’nin başkenti?'), findsOneWidget);
    expect(find.text('Soru 1/10'), findsOneWidget);
    await tester.tap(find.byKey(const Key('daily-option-o1')));
    await tester.pump();
    expect(s.lastSent('daily.answer')['d'], {
      'run_id': 'run-official',
      'position': 1,
      'option_id': 'o1',
    });
    s
      ..ack('daily.answer')
      ..serverSends({
        't': 'daily.revealed',
        'd': {
          'run_id': 'run-official',
          'position': 1,
          'correct_option_id': 'o1',
          'option_id': 'o1',
          'correct': true,
        },
      });
    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text('Doğru!'), findsOneWidget);
    s.serverSends({
      't': 'daily.completed',
      'd': {
        'run_id': 'run-official',
        'kind': 'official',
        'correct_count': 9,
        'total_time_ms': 12345,
      },
    });
    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text('9/10 doğru · 12.3 sn'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('daily: finished today shows result, place and practice', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.daily
        ..todayValue = const DailyToday(
          day: '2026-10-20',
          available: true,
          practiceAllowed: true,
          official: DailyRun(
            id: 'r1',
            kind: 'official',
            status: 'completed',
            correct: 8,
            totalTimeMs: 40200,
          ),
        )
        ..boardValue = const DailyBoard(
          entries: [
            DailyEntry(
              rank: 1,
              userId: 'u-ali',
              correct: 10,
              totalTimeMs: 30000,
              user: UserCard(userId: 'u-ali', displayName: 'Ali'),
            ),
          ],
          me: DailyEntry(
            rank: 4,
            userId: 'user-guest-1',
            correct: 8,
            totalTimeMs: 40200,
          ),
        );
    });
    await tester.tap(find.byKey(const Key('open-daily')));
    await tester.pumpAndSettle();
    expect(find.text('8/10 doğru · 40.2 sn'), findsOneWidget);
    expect(find.text('Sıralaman: #4'), findsOneWidget);
    expect(find.textContaining('Ali'), findsOneWidget);
    await tester.tap(find.byKey(const Key('daily-practice')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(t.daily.started, ['practice']);
    await finish(tester, t);
  });
}
