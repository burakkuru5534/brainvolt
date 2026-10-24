import 'dart:async';
import 'dart:io';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/app/theme.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../support/fakes.dart';

/// Every routed screen at the smallest supported phone width (360 dp) with
/// large text (130 %) must lay out without overflow (D-099). Overflow is
/// reported as a test failure by the framework.
void main() {
  setUpAll(() async {
    final bytes = File('assets/fonts/Nunito-Variable.ttf').readAsBytesSync();
    await (FontLoader(
      brandFont,
    )..addFont(Future.value(ByteData.view(bytes.buffer)))).load();
  });
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  for (final (name, route) in [
    ('home', Routes.home),
    ('modes', Routes.modes),
    ('queue', Routes.play),
    ('daily', Routes.daily),
    ('leaderboards', Routes.leaderboards),
    ('quests', Routes.quests),
    ('badges', Routes.badges),
    ('friends', Routes.friends),
    ('challenges', Routes.challenges),
    ('teams', Routes.teams),
    ('quizzes', Routes.quizzes),
    ('quiz new', Routes.quizNew),
    ('settings', Routes.settings),
    ('profile', Routes.profile),
    ('account', Routes.account),
    ('email sign-up', Routes.emailSignUp),
    ('verify email', Routes.verifyEmail),
    ('credits', Routes.credits),
    ('sanctions', Routes.sanctions),
    ('inbox', Routes.inbox),
    ('notification settings', Routes.notificationSettings),
    ('tournaments', Routes.tournaments),
    ('tournament new', Routes.tournamentNew),
  ]) {
    testWidgets('$name fits 360 dp at 130 % text', (tester) async {
      tester.view.physicalSize = const Size(360, 740);
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = 1.3;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      final t = await TestApp.create();
      _seed(t);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: t.container,
          child: const BrainVoltApp(),
        ),
      );
      await tester.pumpAndSettle();
      t.container.read(routerProvider).go(route);
      for (var i = 0; i < 20; i++) {
        await tester.pump(const Duration(milliseconds: 50));
      }
      expect(tester.takeException(), isNull);
      if (route == Routes.play) {
        // Leave the queue, as a player would.
        await tester.ensureVisible(find.byKey(const Key('queue-cancel')));
        await tester.tap(find.byKey(const Key('queue-cancel')));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 500));
      }
      unawaited(t.container.read(realtimeClientProvider).stop());
      await tester.pump();
      await tester.pump(const Duration(seconds: 5));
    });
  }

  testWidgets('home sheets and dialogs fit 360 dp at 130 % text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 740);
    tester.view.devicePixelRatio = 1;
    tester.platformDispatcher.textScaleFactorTestValue = 1.3;
    addTearDown(tester.view.reset);
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
    final t = await TestApp.create();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: t.container,
        child: const BrainVoltApp(),
      ),
    );
    await tester.pumpAndSettle();
    Future<void> scrollTo(String key) => tester.scrollUntilVisible(
      find.byKey(Key(key)),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    // The topic sheet of Quick Duel.
    await scrollTo('play-quick-duel');
    await tester.tap(find.byKey(const Key('play-quick-duel')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.tapAt(const Offset(180, 20));
    await tester.pumpAndSettle();
    // The private room sheet, then the room settings dialog.
    await scrollTo('private-room');
    await tester.tap(find.byKey(const Key('private-room')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.tap(find.byKey(const Key('create-room-quick_duel')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('format-ok')), findsOneWidget);
    expect(tester.takeException(), isNull);
    unawaited(t.container.read(realtimeClientProvider).stop());
    await tester.pump();
    await tester.pump(const Duration(seconds: 5));
  });
}

void _seed(TestApp t) {
  t.progress.boards['weekly/user/global'] = Board(
    entries: [
      for (var i = 0; i < 6; i++)
        BoardEntry(
          rank: i + 1,
          entityId: 'u$i',
          score: 98000 - i * 130,
          name: 'Uzun İsimli Oyuncu $i',
          avatarId: 'fox',
        ),
    ],
    me: const BoardEntry(
      rank: 6,
      entityId: 'u5',
      score: 97350,
      name: 'Uzun İsimli Oyuncu 5',
    ),
  );
  t.progress.questList = QuestList(
    daily: [
      Quest(
        id: 'q1',
        cadence: 'daily',
        title: 'Bugün üç farklı kategoride en az beş doğru cevap ver',
        progress: 3,
        target: 3,
        rewardXp: 150,
        endsAt: DateTime(2030),
        completedAt: DateTime(2026),
      ),
    ],
    weekly: const [],
    rerollsLeft: 1,
  );
  t.progress.catalogue = [
    BadgeInfo(
      code: 'b1',
      family: 'wins',
      tier: 1,
      name: 'Çok Uzun Bir Rozet Adı Burada',
      description: 'Arka arkaya on maç kazanarak bu rozeti elde et.',
      earnedAt: DateTime(2026),
    ),
  ];
}
