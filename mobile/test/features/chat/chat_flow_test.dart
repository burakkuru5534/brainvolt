import 'dart:async';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/chat/domain/chat_repository.dart';
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

  Future<void> step(WidgetTester tester) async {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
  }

  Map<String, dynamic> wire(
    String id,
    String sender,
    String name, {
    String? body,
    String? quick,
  }) => {
    'id': id,
    'channel_id': 'ch1',
    'sender': {'user_id': sender, 'display_name': name},
    'body': ?body,
    'quick_id': ?quick,
    'created_at': '2026-10-20T09:00:00Z',
  };

  testWidgets(
    'team chat: history, send, live message, typing, quick, removal',
    (tester) async {
      tester.view.physicalSize = const Size(800, 1600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final t = await TestApp.create();
      t.chat.messages = [
        ChatMessage(
          id: 'm1',
          channelId: 'ch1',
          sender: const UserCard(userId: 'u-ali', displayName: 'Ali'),
          body: 'Selam takım!',
          createdAt: DateTime(2026, 10, 20, 9),
        ),
      ];
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: t.container,
          child: const BrainVoltApp(),
        ),
      );
      await tester.pumpAndSettle();
      t.container.read(routerProvider).go(Routes.teamChat('t1', 'Volt'));
      await step(tester);
      await step(tester);
      expect(t.chat.calls, ['open:t1', 'read:m1']);
      expect(find.text('Selam takım!'), findsOneWidget);
      final s = t.sockets.single;
      s.serverSends({
        't': 'sys.welcome',
        'd': {'session_id': 's1', 'resume_token': 'r1'},
      });
      await step(tester);

      await tester.enterText(find.byKey(const Key('chat-input')), 'Merhaba');
      await tester.tap(find.byKey(const Key('chat-send')));
      await step(tester);
      expect(s.lastSent('chat.send')['d'], {
        'channel_id': 'ch1',
        'body': 'Merhaba',
      });
      s
        ..ack('chat.send')
        ..serverSends({
          't': 'chat.message',
          's': 'chat:ch1',
          'd': {'message': wire('m2', 'user-guest-1', 'Ben', body: 'Merhaba')},
        });
      await step(tester);
      expect(find.text('Merhaba'), findsOneWidget);
      expect(t.chat.calls.last, 'read:m2');

      s.serverSends({
        't': 'chat.typing',
        's': 'chat:ch1',
        'd': {'channel_id': 'ch1', 'user_id': 'u-ali'},
      });
      await step(tester);
      expect(find.text('Ali yazıyor…'), findsOneWidget);

      await tester.tap(find.byKey(const Key('quick-good_luck')));
      await step(tester);
      expect(s.lastSent('chat.send')['d'], {
        'channel_id': 'ch1',
        'quick_id': 'good_luck',
      });
      s.ack('chat.send');

      s.serverSends({
        't': 'chat.message_removed',
        's': 'chat:ch1',
        'd': {'channel_id': 'ch1', 'id': 'm1'},
      });
      await step(tester);
      expect(find.text('Bu mesaj silindi.'), findsOneWidget);

      // Long press on one's own message removes it.
      await tester.longPress(find.byKey(const Key('chat-m2')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('chat-delete-confirm')));
      await step(tester);
      expect(t.chat.calls, contains('remove:m2'));

      unawaited(t.container.read(realtimeClientProvider).stop());
      await tester.pump();
      await tester.pump(const Duration(seconds: 5));
    },
  );
}
