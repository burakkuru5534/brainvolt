@Tags(['golden'])
library;

import 'dart:async';
import 'dart:io';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/app/theme.dart';
import 'package:brainvolt/core/network/api_exception.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/core/share/result_card.dart';
import 'package:brainvolt/core/widgets/bv.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../support/fakes.dart';

/// Screenshots of the screens in the BrainVolt design (D-099): the
/// reference for visual changes. Regenerate with
/// `flutter test --run-skipped --tags golden --update-goldens`.
Future<void> loadFonts() async {
  final bytes = File('assets/fonts/Nunito-Variable.ttf').readAsBytesSync();
  final loader = FontLoader(brandFont)
    ..addFont(Future.value(ByteData.view(bytes.buffer)));
  await loader.load();
  final icons = File(
    '${Platform.environment['FLUTTER_ROOT'] ?? '/opt/sdk/flutter'}/bin/cache/artifacts/material_fonts/MaterialIcons-Regular.otf',
  ).readAsBytesSync();
  await (FontLoader(
    'MaterialIcons',
  )..addFont(Future.value(ByteData.view(icons.buffer)))).load();
}

/// Decodes the bundled images for real (test time is fake), then repaints.
Future<void> loadImages(WidgetTester tester) async {
  await tester.runAsync(() async {
    final context = tester.element(find.byType(Navigator).first);
    await precacheImage(const AssetImage(BrandMark.asset), context);
  });
  await tester.pump();
}

void main() {
  setUpAll(loadFonts);
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  // BrainVolt has one (dark) theme (D-099); the files keep the `_dark` name.
  for (final mode in [ThemeMode.dark]) {
    Future<TestApp> launch(
      WidgetTester tester,
      String route, {
      AuthUser? user,
    }) async {
      tester.view.physicalSize = const Size(1170, 2532);
      tester.view.devicePixelRatio = 3;
      addTearDown(tester.view.reset);
      final t = await TestApp.create(
        prefs: {'theme_mode': mode.name},
        user: user,
      );
      _seedProgress(t);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: t.container,
          child: const BrainVoltApp(),
        ),
      );
      await tester.pumpAndSettle();
      t.container.read(routerProvider).go(route);
      // Some screens animate forever (the queue radar).
      for (var i = 0; i < 20; i++) {
        await tester.pump(const Duration(milliseconds: 50));
      }
      await loadImages(tester);
      return t;
    }

    Future<void> finish(WidgetTester tester, TestApp t) async {
      unawaited(t.container.read(realtimeClientProvider).stop());
      await tester.pump();
      await tester.pump(const Duration(seconds: 5));
    }

    Future<void> shoot(WidgetTester tester, TestApp t, String name) async {
      tester.view.physicalSize = const Size(1170, 2532);
      tester.view.devicePixelRatio = 3;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: t.container,
          child: const BrainVoltApp(),
        ),
      );
      await tester.pumpAndSettle();
      await loadImages(tester);
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('${name}_${mode.name}.png'),
      );
      await finish(tester, t);
    }

    testWidgets('welcome ${mode.name}', (tester) async {
      final t = await TestApp.create(ageConfirmed: false);
      await shoot(tester, t, 'welcome');
    });

    testWidgets('splash error ${mode.name}', (tester) async {
      final t = await TestApp.create();
      t.sessions.syncError = const ApiException(code: ApiException.networkCode);
      await shoot(tester, t, 'splash_error');
    });

    testWidgets('home ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.home);
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('home_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    for (final (name, route) in [
      ('daily', Routes.daily),
      ('daily_answers', Routes.dailyAnswers),
      ('player', Routes.player('u-deniz')),
      ('guide', Routes.guide),
      ('leaderboards', Routes.leaderboards),
      ('quests', Routes.quests),
      ('badges', Routes.badges),
      ('friends', Routes.friends),
      ('challenges', Routes.challenges),
      ('teams', Routes.teams),
      ('quizzes', Routes.quizzes),
      ('settings', Routes.settings),
      ('profile', Routes.profile),
      ('account', Routes.account),
      ('email_sign_up', Routes.emailSignUp),
      ('verify_email', Routes.verifyEmail),
      ('credits', Routes.credits),
      ('sanctions', Routes.sanctions),
      ('inbox', Routes.inbox),
      ('notification_settings', Routes.notificationSettings),
      ('tournaments', Routes.tournaments),
      ('tournament_new', Routes.tournamentNew),
      ('quiz_new', Routes.quizNew),
      ('store', Routes.store),
      ('store_cosmetics', '${Routes.store}?tab=cosmetics'),
    ]) {
      testWidgets('$name ${mode.name}', (tester) async {
        final t = await launch(tester, route);
        await expectLater(
          find.byType(BrainVoltApp),
          matchesGoldenFile('${name}_${mode.name}.png'),
        );
        await finish(tester, t);
      });
    }

    testWidgets('plus ${mode.name}', (tester) async {
      final t = await launch(
        tester,
        Routes.plus,
        user: const AuthUser(
          uid: 'u-full',
          isAnonymous: false,
          emailVerified: true,
        ),
      );
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('plus_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    testWidgets('guide open ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.guide);
      await tester.tap(find.byKey(const Key('guide-scoring')));
      await tester.pumpAndSettle();
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('guide_open_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    testWidgets('modes ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.modes);
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('modes_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    testWidgets('queue ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.play);
      await tester.pump(const Duration(milliseconds: 700));
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('queue_${mode.name}.png'),
      );
      await tester.tap(find.byKey(const Key('queue-cancel')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      await finish(tester, t);
    });

    testWidgets('match ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.play);
      final s = t.sockets.single;
      int now() => DateTime.now().millisecondsSinceEpoch;
      Future<void> step() async {
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 150));
      }

      s.serverSends({
        't': 'sys.welcome',
        'd': {'session_id': 's1', 'resume_token': 'r1'},
      });
      await step();
      s.serverSends({
        't': 'mm.match_found',
        'd': {'match_id': 'm1'},
      });
      await step();
      await step();
      s
        ..ack('match.join')
        ..serverSends({
          't': 'match.snapshot',
          's': 'match:m1',
          'seq': 1,
          'd': {
            'status': 'in_progress',
            'phase': 'lobby',
            'question_count': 7,
            'participants': [
              {
                'user_id': 'user-guest-1',
                'side_no': 1,
                'display_name': 'Ben',
                'avatar_id': 'penguin',
              },
              {
                'user_id': 'opp',
                'side_no': 2,
                'display_name': 'Pratik Botu',
                'avatar_id': 'tiger',
              },
            ],
            'sides': [
              {'side_no': 1, 'score': 0},
              {'side_no': 2, 'score': 159},
            ],
          },
        })
        ..serverSends({
          't': 'round.started',
          's': 'match:m1',
          'seq': 2,
          'd': {
            'round_no': 1,
            'question': {
              'id': 'q1',
              'prompt':
                  'Bugün İstanbul olarak bilinen şehrin tarihteki adı neydi?',
              'category': {'id': 'c1', 'name': 'Coğrafya'},
            },
            'options': [
              {'id': 'o1', 'text': 'Konstantinopolis'},
              {'id': 'o2', 'text': 'İskenderiye'},
              {'id': 'o3', 'text': 'Antakya'},
              {'id': 'o4', 'text': 'Efes'},
            ],
            'show_at': now() - 100,
            'deadline_at': now() + 15000,
          },
        });
      await step();
      await tester.tap(find.byKey(const Key('option-o1')));
      await step();
      s
        ..ack('match.answer')
        ..serverSends({
          't': 'round.answer_received',
          's': 'match:m1',
          'seq': 3,
          'd': {'round_no': 1, 'user_id': 'user-guest-1', 'option_id': 'o1'},
        });
      await step();
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('match_question_${mode.name}.png'),
      );
      s.serverSends({
        't': 'round.revealed',
        's': 'match:m1',
        'seq': 4,
        'd': {
          'round_no': 1,
          'correct_option_id': 'o1',
          'answers': [
            {
              'user_id': 'user-guest-1',
              'option_id': 'o1',
              'correct': true,
              'points': {'total': 108},
            },
            {
              'user_id': 'opp',
              'option_id': 'o1',
              'correct': true,
              'points': {'total': 0},
            },
          ],
          'sides': [
            {'side_no': 1, 'score': 108},
            {'side_no': 2, 'score': 159},
          ],
        },
      });
      await step();
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('match_reveal_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    testWidgets('result ${mode.name}', (tester) async {
      final t = await TestApp.create(prefs: {'theme_mode': mode.name});
      _seedProgress(t);
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
          ResultPlayer(
            userId: 'opp',
            side: 2,
            displayName: 'Rakip',
            avatarId: 'tiger',
            correct: 4,
            status: 'finished',
          ),
        ],
        rounds: [
          ResultRound(
            roundNo: 1,
            questionId: 'q1',
            prompt: 'Türkiye’nin başkenti neresidir?',
            correctText: 'Ankara',
            answers: [
              ResultAnswer(
                userId: 'user-guest-1',
                optionId: 'o1',
                correct: true,
                points: 108,
              ),
            ],
            options: {},
          ),
          ResultRound(
            roundNo: 2,
            questionId: 'q2',
            prompt: 'En uzun nehir hangisidir?',
            correctText: 'Nil',
            answers: [
              ResultAnswer(
                userId: 'user-guest-1',
                optionId: 'o2',
                correct: false,
                points: 0,
              ),
            ],
            options: {},
          ),
        ],
      );
      tester.view.physicalSize = const Size(1170, 2532);
      tester.view.devicePixelRatio = 3;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: t.container,
          child: const BrainVoltApp(),
        ),
      );
      await tester.pumpAndSettle();
      t.container.read(routerProvider).go(Routes.matchResult('m1'));
      for (var i = 0; i < 20; i++) {
        await tester.pump(const Duration(milliseconds: 50));
      }
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('result_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    testWidgets('share card ${mode.name}', (tester) async {
      tester.view.physicalSize = const Size(1170, 1500);
      tester.view.devicePixelRatio = 3;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        MaterialApp(
          debugShowCheckedModeBanner: false,
          theme: buildTheme(
            mode == ThemeMode.dark ? Brightness.dark : Brightness.light,
          ),
          home: const Scaffold(
            body: Center(
              child: ResultCard(
                headline: 'Kazandın!',
                big: '612 — 431',
                detail: '5/7 doğru',
              ),
            ),
          ),
        ),
      );
      await loadImages(tester);
      await expectLater(
        find.byType(ResultCard),
        matchesGoldenFile('share_card_${mode.name}.png'),
      );
    });

    testWidgets('topic sheet ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.home);
      await tester.scrollUntilVisible(
        find.byKey(const Key('play-quick-duel')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.byKey(const Key('play-quick-duel')));
      await tester.pumpAndSettle();
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('topic_sheet_${mode.name}.png'),
      );
      await finish(tester, t);
    });

    testWidgets('room sheet ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.home);
      await tester.scrollUntilVisible(
        find.byKey(const Key('private-room')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.byKey(const Key('private-room')));
      await tester.pumpAndSettle();
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('room_sheet_${mode.name}.png'),
      );
      await finish(tester, t);
    });
  }
}

/// Sample boards, quests and badges, so the screens show their rows.
void _seedProgress(TestApp t) {
  const names = ['Ali', 'Ayşe', 'Deniz', 'Ece', 'Mert', 'Ben'];
  const avatars = ['fox', 'owl', 'cat', 'panda', 'lion', 'penguin'];
  t.progress.boards['weekly/user/global'] = Board(
    entries: [
      for (var i = 0; i < names.length; i++)
        BoardEntry(
          rank: i + 1,
          entityId: names[i] == 'Ben' ? 'me' : 'u$i',
          score: 980 - i * 130,
          name: names[i],
          avatarId: avatars[i],
        ),
    ],
    me: const BoardEntry(
      rank: 6,
      entityId: 'me',
      score: 330,
      name: 'Ben',
      avatarId: 'penguin',
    ),
  );
  Quest quest(
    String id,
    String title,
    int progress,
    int target, [
    bool done = false,
  ]) => Quest(
    id: id,
    cadence: 'daily',
    title: title,
    progress: progress,
    target: target,
    rewardXp: 50,
    endsAt: DateTime(2030),
    completedAt: done ? DateTime(2026) : null,
  );
  t.progress.questList = QuestList(
    daily: [
      quest('q1', '3 maç kazan', 3, 3, true),
      quest('q2', '10 doğru cevap ver', 4, 10),
    ],
    weekly: [quest('w1', '15 maç oyna', 6, 15)],
    rerollsLeft: 1,
  );
  t.progress.catalogue = [
    BadgeInfo(
      code: 'first_win',
      family: 'wins',
      tier: 1,
      name: 'İlk Galibiyet',
      description: 'İlk maçını kazan.',
      earnedAt: DateTime(2026),
    ),
    BadgeInfo(
      code: 'streak_3',
      family: 'streak',
      tier: 1,
      name: '3 Gün Seri',
      description: '3 gün üst üste oyna.',
      earnedAt: DateTime(2026),
    ),
    const BadgeInfo(
      code: 'wins_10',
      family: 'wins',
      tier: 2,
      name: '10 Galibiyet',
      description: '10 maç kazan.',
    ),
    const BadgeInfo(
      code: 'quiz_master',
      family: 'quiz',
      tier: 1,
      name: 'Quiz Ustası',
      description: 'Bir quiz yayınla.',
    ),
  ];
}
