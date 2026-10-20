import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/notifications/domain/notification_repository.dart';
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

  final request = InboxItem(
    id: 'n1',
    category: 'friend',
    kind: 'friend.request',
    title: 'Arkadaşlık isteği',
    body: 'Ayşe sana arkadaşlık isteği gönderdi.',
    deepLink: '/friends',
    createdAt: DateTime(2026, 10, 20, 9, 30),
    read: false,
  );

  testWidgets('bell badge, inbox, tapping opens the route and reads it', (
    tester,
  ) async {
    final t = await launch(tester, (t) => t.notifications.items = [request]);
    expect(find.text('1'), findsOneWidget);
    await tester.tap(find.byKey(const Key('open-inbox')));
    await tester.pumpAndSettle();
    expect(find.text('Ayşe sana arkadaşlık isteği gönderdi.'), findsOneWidget);
    expect(find.byKey(const Key('inbox-unread-n1')), findsOneWidget);
    await tester.tap(find.byKey(const Key('inbox-n1')));
    await tester.pumpAndSettle();
    expect(t.notifications.calls, ['read:n1']);
    expect(t.container.read(routerProvider).state.uri.path, Routes.friends);
    await finish(tester, t);
  });

  testWidgets('mark all read, empty inbox', (tester) async {
    final t = await launch(tester, (t) => t.notifications.items = [request]);
    t.container.read(routerProvider).go(Routes.inbox);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('inbox-mark-all')));
    await tester.pumpAndSettle();
    expect(t.notifications.calls, ['read:all']);
    expect(find.byKey(const Key('inbox-unread-n1')), findsNothing);
    t.notifications.items = [];
    t.container.read(routerProvider).go(Routes.home);
    await tester.pumpAndSettle();
    t.container.read(routerProvider).go(Routes.inbox);
    await tester.pumpAndSettle();
    expect(find.text('Henüz bildirimin yok.'), findsOneWidget);
    await finish(tester, t);
  });

  testWidgets('settings: mute a category, quiet hours, locked moderation', (
    tester,
  ) async {
    final t = await launch(tester);
    t.container.read(routerProvider).go(Routes.notificationSettings);
    await tester.pumpAndSettle();
    expect(find.text('Hesap bildirimleri kapatılamaz.'), findsOneWidget);
    expect(find.byKey(const Key('pref-moderation-push')), findsNothing);
    await tester.tap(find.byKey(const Key('pref-friend-push')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('quiet-enabled')));
    await tester.pumpAndSettle();
    expect(t.notifications.calls, [
      'update:friend/false/true:null',
      'update:null:true',
    ]);
    expect(
      tester.widget<Switch>(find.byKey(const Key('pref-friend-push'))).value,
      isFalse,
    );
    await finish(tester, t);
  });
}
