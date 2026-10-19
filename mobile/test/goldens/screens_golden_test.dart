@Tags(['golden'])
library;

import 'dart:async';
import 'dart:io';

import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/app/theme.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../support/fakes.dart';

/// Screenshots of the main screens in light and dark (D-085): the
/// reference for the visual redesign. Regenerate with
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

void main() {
  setUpAll(loadFonts);
  setUp(() {
    final binding = TestWidgetsFlutterBinding.ensureInitialized();
    binding.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(binding.platformDispatcher.clearLocalesTestValue);
  });

  for (final mode in [ThemeMode.light, ThemeMode.dark]) {
    Future<TestApp> launch(WidgetTester tester, String route) async {
      tester.view.physicalSize = const Size(1170, 2532);
      tester.view.devicePixelRatio = 3;
      addTearDown(tester.view.reset);
      final t = await TestApp.create(prefs: {'theme_mode': mode.name});
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
      return t;
    }

    Future<void> finish(WidgetTester tester, TestApp t) async {
      unawaited(t.container.read(realtimeClientProvider).stop());
      await tester.pump();
      await tester.pump(const Duration(seconds: 5));
    }

    testWidgets('home ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.home);
      await expectLater(
        find.byType(BrainVoltApp),
        matchesGoldenFile('home_${mode.name}.png'),
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

    testWidgets('room sheet ${mode.name}', (tester) async {
      final t = await launch(tester, Routes.home);
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
