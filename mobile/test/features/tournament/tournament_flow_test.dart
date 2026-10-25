import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/core/widgets/bv.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/store/domain/store_repository.dart';
import 'package:brainvolt/features/tournament/domain/tournament_repository.dart';
import 'package:brainvolt/features/tournament/presentation/tournament_screens.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

const me = 'user-guest-1';
const fullUser = AuthUser(
  uid: 'u-full',
  isAnonymous: false,
  emailVerified: true,
);

Map<String, dynamic> tournamentJson({
  String status = 'registration',
  Map<String, dynamic>? myEntry,
  DateTime? startsAt,
}) {
  final now = DateTime.now();
  final start = startsAt ?? now.add(const Duration(minutes: 30));
  return {
    'id': 't1',
    'name': 'Bilgi Kupası',
    'status': status,
    'entity_kind': 'user',
    'match_config': {'match_type': 'quick_duel', 'format': '1v1'},
    'min_entrants': 2,
    'max_entrants': 8,
    'entrants': 3,
    'registration_opens_at': now
        .subtract(const Duration(hours: 1))
        .toUtc()
        .toIso8601String(),
    'registration_closes_at': start.toUtc().toIso8601String(),
    'check_in_opens_at': start
        .subtract(const Duration(hours: 1))
        .toUtc()
        .toIso8601String(),
    'starts_at': start.toUtc().toIso8601String(),
    'round_ready_window_s': 300,
    'rewards': {
      'places': [
        {'place': 1, 'xp': 300},
      ],
      'participation_xp': 20,
    },
    'my_entry': ?myEntry,
  };
}

Map<String, dynamic> entry(String id, String user, String name, int seed) => {
  'id': id,
  'entity_kind': 'user',
  'entity_id': user,
  'name': name,
  'image': 'fox',
  'lineup': [user],
  'seed': seed,
  'status': 'active',
};

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
    AuthUser? user,
  ]) async {
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

  testWidgets('home tile, list, register, check-in opens in the last hour', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.tournaments
        ..tournaments = [Tournament.fromJson(tournamentJson())]
        ..details['t1'] = TournamentDetail.fromJson({
          ...tournamentJson(),
          'rounds': 0,
          'entries': const <Object>[],
          'bracket': const <Object>[],
        });
    });
    await tester.tap(find.byKey(const Key('open-tournaments')));
    await tester.pumpAndSettle();
    expect(find.text('Bilgi Kupası'), findsOneWidget);
    expect(find.text('Kayıt açık'), findsOneWidget);
    expect(find.byKey(const Key('create-tournament')), findsNothing);
    await tester.tap(find.byKey(const Key('tournament-t1')));
    await step(tester);
    expect(find.text('1.: 300 XP'), findsOneWidget);
    await tester.tap(find.byKey(const Key('tournament-register')));
    await step(tester);
    expect(t.tournaments.calls, ['register:t1']);

    // Registered, 30 minutes before the start: check-in is open.
    t.tournaments.details['t1'] = TournamentDetail.fromJson({
      ...tournamentJson(
        myEntry: entry('e1', me, 'Ben', 0)..['status'] = 'registered',
      ),
      'rounds': 0,
      'entries': const <Object>[],
      'bracket': const <Object>[],
    });
    t.container.invalidate(tournamentProvider('t1'));
    await step(tester);
    expect(
      find.text('Check-in açık! Oynamak için check-in yap.'),
      findsOneWidget,
    );
    await tester.tap(find.byKey(const Key('tournament-check-in')));
    await step(tester);
    expect(t.tournaments.calls, ['register:t1', 'check-in:t1']);
    await finish(tester, t);
  });

  testWidgets('running: the bracket, Play, then the match', (tester) async {
    final t = await launch(tester, (t) {
      t.tournaments.details['t1'] = TournamentDetail.fromJson({
        ...tournamentJson(
          status: 'running',
          myEntry: entry('e1', me, 'Ben', 1),
          startsAt: DateTime.now().subtract(const Duration(minutes: 1)),
        ),
        'rounds': 2,
        'entries': [
          entry('e1', me, 'Ben', 1),
          entry('e2', 'u2', 'Ali', 2),
          entry('e3', 'u3', 'Ayşe', 3),
        ],
        'bracket': [
          {
            'round_no': 1,
            'slot_no': 1,
            'entry_a_id': 'e1',
            'status': 'walkover',
            'winner_entry_id': 'e1',
          },
          {
            'round_no': 1,
            'slot_no': 2,
            'entry_a_id': 'e2',
            'entry_b_id': 'e3',
            'status': 'done',
            'winner_entry_id': 'e3',
          },
          {
            'round_no': 2,
            'slot_no': 1,
            'entry_a_id': 'e1',
            'entry_b_id': 'e3',
            'status': 'ready',
            'ready_until': DateTime.now()
                .add(const Duration(minutes: 4))
                .toUtc()
                .toIso8601String(),
          },
        ],
        'my_match': {
          'round_no': 2,
          'slot_no': 1,
          'entry_a_id': 'e1',
          'entry_b_id': 'e3',
          'status': 'ready',
          'ready_until': DateTime.now()
              .add(const Duration(minutes: 4))
              .toUtc()
              .toIso8601String(),
        },
      });
      t.tournaments.played = 'm7';
    });
    unawaited(t.container.read(routerProvider).push(Routes.tournament('t1')));
    await step(tester);
    expect(find.text('Final'), findsOneWidget);
    expect(find.text('Bay'), findsOneWidget);
    expect(find.text('2. tur: rakibin Ayşe'), findsOneWidget);
    expect(
      find.textContaining('Katılmak için kalan süre: 3:5'),
      findsOneWidget,
    );
    await tester.tap(find.byKey(const Key('tournament-play')));
    await step(tester);
    await step(tester);
    expect(t.tournaments.calls, ['play:t1']);
    expect(
      t.container.read(routerProvider).state.matchedLocation,
      Routes.match('m7'),
    );
    await finish(tester, t);
  });

  testWidgets('admins create a tournament', (tester) async {
    final t = await launch(tester, (t) {
      t.sessions.roles = ['admin'];
      t.tournaments.details['new-t'] = TournamentDetail.fromJson({
        ...tournamentJson(),
        'rounds': 0,
        'entries': const <Object>[],
        'bracket': const <Object>[],
      });
    });
    unawaited(t.container.read(routerProvider).push(Routes.tournaments));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-tournament')));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('tournament-name')),
      'Hafta Sonu Kupası',
    );
    await tester.pump();
    await tester.tap(find.byKey(const Key('tournament-submit')));
    await tester.pumpAndSettle();
    expect(t.tournaments.calls, ['create:Hafta Sonu Kupası:user:1v1:8']);
    expect(find.text('Bilgi Kupası'), findsWidgets);
    await finish(tester, t);
  });

  testWidgets('friend tournament: Plus sets up for free (D-109)', (
    tester,
  ) async {
    final t = await launch(tester, (t) {
      t.store.value = const Entitlements(plus: true);
      t.tournaments.details['new-ft'] = TournamentDetail.fromJson({
        ...tournamentJson(),
        'id': 'new-ft',
        'name': 'Kuzenler',
        'kind': 'friends',
        'is_owner': true,
        'rewards': {'places': const <Object>[]},
        'rounds': 0,
        'entries': const <Object>[],
        'bracket': const <Object>[],
      });
    }, fullUser);
    unawaited(t.container.read(routerProvider).push(Routes.tournaments));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-friend-tournament')));
    await tester.pumpAndSettle();
    expect(find.text('Plus ile ücretsiz kurarsın'), findsOneWidget);
    await tester.enterText(find.byKey(const Key('ft-name')), 'Kuzenler');
    await tester.tap(find.byKey(const Key('ft-16')));
    await tester.tap(find.byKey(const Key('ft-marathon')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('ft-submit')));
    await tester.pumpAndSettle();
    expect(t.tournaments.calls, ['create-friends:Kuzenler:marathon:16']);
    expect(find.byKey(const Key('tournament-friends-rules')), findsOneWidget);

    // The owner cancels before the start.
    await tester.tap(find.byKey(const Key('tournament-cancel-friends')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('dialog-confirm')));
    await tester.pumpAndSettle();
    expect(t.tournaments.calls.last, 'cancel:new-ft');
    expect(find.text('Turnuva iptal edildi'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('friend tournament: 300 Volt without Plus', (tester) async {
    final t = await launch(tester, null, fullUser);
    unawaited(
      t.container.read(routerProvider).push(Routes.friendTournamentNew),
    );
    await tester.pumpAndSettle();
    expect(find.text('Kurma ücreti: 300 Volt'), findsOneWidget);
    await tester.enterText(find.byKey(const Key('ft-name')), 'Kuzenler');
    await tester.pump();
    // The fake wallet has 240 Volt: not enough.
    final button = tester.widget<GradientButton>(
      find.byKey(const Key('ft-submit')),
    );
    expect(button.onPressed, isNull);
    await finish(tester, t);
  });
}
