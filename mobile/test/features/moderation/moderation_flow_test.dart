import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/moderation/domain/moderation_repository.dart';
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

  testWidgets('result: report a question with a reason and a note', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.matches.matchResult = const MatchResult(
        id: 'm1',
        status: 'cancelled',
        scores: [0, 0],
        players: [],
        rounds: [
          ResultRound(
            roundNo: 1,
            questionId: 'q-1',
            prompt: 'Başkent?',
            correctText: 'Ankara',
            answers: [],
            options: {},
          ),
        ],
      );
    });
    t.container.read(routerProvider).go(Routes.matchResult('m1'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('report-q-1')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('report-reason-wrong_answer')));
    await tester.pump();
    await tester.enterText(
      find.byKey(const Key('report-comment')),
      'Cevap İzmir',
    );
    await tester.tap(find.byKey(const Key('report-send')));
    await tester.pumpAndSettle();
    expect(t.moderation.reports, ['q-1:wrong_answer:m1:Cevap İzmir']);
    expect(find.text('Bildirimin için teşekkürler.'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('sanctions: list and appeal an active one', (tester) async {
    final t = await launch(tester, (t) {
      t.moderation.sanctions = [
        Sanction(
          id: 's1',
          kind: 'matchmaking_ban',
          reasonCode: 'cheating_confirmed',
          active: true,
          endsAt: DateTime(2030),
        ),
        const Sanction(
          id: 's0',
          kind: 'warning',
          reasonCode: 'spam',
          active: false,
        ),
      ];
    });
    t.container.read(routerProvider).go(Routes.sanctions);
    await tester.pumpAndSettle();
    expect(find.text('Eşleştirme yasağı'), findsOneWidget);
    expect(find.text('Uyarı'), findsOneWidget);
    expect(find.byKey(const Key('appeal-s0')), findsNothing);
    await tester.tap(find.byKey(const Key('appeal-s1')));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('appeal-text')),
      'Hile yapmadım, lütfen kontrol edin.',
    );
    await tester.pump();
    await tester.tap(find.byKey(const Key('appeal-send')));
    await tester.pumpAndSettle();
    expect(t.moderation.appeals, ['s1:Hile yapmadım, lütfen kontrol edin.']);
    await finish(tester, t);
  });

  testWidgets('result: share the match as a card', (tester) async {
    final t = await launch(tester, (t) {
      t.matches.matchResult = const MatchResult(
        id: 'm1',
        status: 'completed',
        scores: [612, 431],
        winnerSide: 1,
        players: [
          ResultPlayer(
            userId: 'user-guest-1',
            side: 1,
            displayName: 'Ben',
            avatarId: 'penguin',
            correct: 5,
            status: 'finished',
          ),
        ],
        rounds: [],
      );
    });
    t.container.read(routerProvider).go(Routes.matchResult('m1'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('share-result')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('share-card')), findsOneWidget);
    expect(find.text('612 — 431'), findsNWidgets(2));
    await tester.tap(find.byKey(const Key('share-send')));
    await tester.pumpAndSettle();
    expect(t.share.shared, ["WitClash'ta 612—431: Kazandın! Sıra sende!"]);
    expect(find.byKey(const Key('share-card')), findsNothing);
    await finish(tester, t);
  });
}
