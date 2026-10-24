import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/creator/domain/creator_repository.dart';
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
    // Tall enough for the editor to build every question.
    tester.view.physicalSize = const Size(800, 8000);
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

  Future<void> step(WidgetTester tester) async {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
  }

  Future<void> finish(WidgetTester tester, TestApp t) async {
    unawaited(t.container.read(realtimeClientProvider).stop());
    await tester.pump();
    await tester.pump(const Duration(seconds: 5));
  }

  const published = Quiz(
    id: 'pub-1',
    title: 'Dünya başkentleri',
    locale: 'tr',
    answerWindowMs: 15000,
    questionCount: 8,
    status: 'published',
    version: 3,
    authorName: 'Ayşe',
    likeCount: 4,
    playCount: 12,
  );

  testWidgets('write a quiz: details, a true/false question, save', (
    tester,
  ) async {
    final t = await launch(tester);
    await tester.tap(find.byKey(const Key('open-quizzes')));
    await tester.pumpAndSettle();
    expect(find.text('Henüz bir quiz yazmadın.'), findsOneWidget);
    await tester.tap(find.byKey(const Key('quiz-new')));
    await tester.pumpAndSettle();

    // Empty questions are refused before anything is sent.
    await tester.enterText(find.byKey(const Key('quiz-title')), 'Başkentler');
    await tester.tap(find.byKey(const Key('quiz-save')));
    await tester.pump();
    expect(find.textContaining('Başlık 3–60 karakter'), findsOneWidget);
    expect(t.creator.calls, isEmpty);

    await tester.tap(find.byKey(const Key('quiz-category')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Tarih').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('20 sn'));
    await tester.tap(find.byKey(const Key('quiz-q-0-tf')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('quiz-q-0-correct-1')));
    // A new quiz starts with one question; "Add question" adds the next.
    expect(find.byKey(const Key('quiz-q-1')), findsNothing);
    // One question is the minimum: it cannot be removed.
    expect(
      tester
          .widget<IconButton>(find.byKey(const Key('quiz-q-0-remove')))
          .onPressed,
      isNull,
    );
    await tester.ensureVisible(find.byKey(const Key('quiz-add-question')));
    await tester.tap(find.byKey(const Key('quiz-add-question')));
    await tester.pumpAndSettle();
    for (var i = 0; i < 2; i++) {
      await tester.enterText(find.byKey(Key('quiz-q-$i-prompt')), 'Soru $i?');
      if (i > 0) {
        for (var j = 0; j < 4; j++) {
          await tester.enterText(
            find.byKey(Key('quiz-q-$i-option-$j')),
            'Seçenek $j',
          );
        }
      }
    }
    await tester.tap(find.byKey(const Key('quiz-save')));
    await tester.pumpAndSettle();
    expect(t.creator.calls, ['create:Başkentler:2']);
    final d = t.creator.saved!;
    expect(d.categoryId, 'cat-history');
    expect(d.locale, 'tr');
    expect(d.answerWindowMs, 20000);
    expect(d.questions.first.options, ['Doğru', 'Yanlış']);
    expect(d.questions.first.correct, 2);
    expect(d.questions.first.type, 'true_false');
    expect(d.questions[1].type, 'multiple_choice');
    // Back on the list with the new draft.
    expect(find.byKey(const Key('my-quiz-quiz-1')), findsOneWidget);
    expect(find.textContaining('Taslak'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('my quiz: submit and play it in a room', (tester) async {
    final t = await launch(tester, (t) {
      t.creator.quizzes = [
        const Quiz(
          id: 'q1',
          title: 'Benim quizim',
          locale: 'tr',
          answerWindowMs: 15000,
          questionCount: 5,
          status: 'draft',
          version: 1,
        ),
      ];
    });
    t.container.read(routerProvider).go(Routes.quizzes);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('quiz-menu-q1')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('quiz-submit')));
    await tester.pumpAndSettle();
    expect(t.creator.calls, ['submit:q1']);
    expect(find.text('Quiz incelemeye gönderildi.'), findsOneWidget);

    await tester.tap(find.byKey(const Key('quiz-menu-q1')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('quiz-play')));
    await step(tester);
    await step(tester);
    expect(t.lobby.calls, contains('createRoom:quick_duel:quiz:q1'));
    expect(
      t.container.read(routerProvider).state.uri.path,
      startsWith('/rooms/'),
    );
    await finish(tester, t);
  });

  testWidgets('discover: like a published quiz and open a room', (
    tester,
  ) async {
    final t = await launch(tester, (t) => t.creator.published = [published]);
    t.container.read(routerProvider).go(Routes.quizzes);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('tab-discover')));
    await tester.pumpAndSettle();
    expect(find.text('Dünya başkentleri'), findsOneWidget);
    expect(find.textContaining('8 soru · Ayşe'), findsOneWidget);
    await tester.tap(find.byKey(const Key('quiz-like-pub-1')));
    await tester.pumpAndSettle();
    expect(t.creator.calls, ['like:pub-1:true']);
    await tester.tap(find.byKey(const Key('quiz-room-pub-1')));
    await step(tester);
    await step(tester);
    expect(t.lobby.calls, contains('createRoom:quick_duel:quiz:pub-1'));
    await finish(tester, t);
  });
}
