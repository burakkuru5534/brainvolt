import 'package:brainvolt/app/app.dart';
import 'package:brainvolt/app/config.dart';
import 'package:brainvolt/app/providers.dart';
import 'package:brainvolt/core/l10n/locale_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

Future<SharedPreferences> prefs([Map<String, Object> values = const {}]) async {
  SharedPreferences.setMockInitialValues(values);
  return SharedPreferences.getInstance();
}

/// A root container with the values main() injects.
ProviderContainer container(SharedPreferences p) {
  final c = ProviderContainer(
    overrides: [
      appConfigProvider.overrideWithValue(
        AppConfig.parse(
          flavor: 'dev',
          apiBaseUrl: 'http://10.0.2.2:8080',
          wsUrl: 'ws://10.0.2.2:8080/v1/ws',
        ),
      ),
      sharedPreferencesProvider.overrideWithValue(p),
      appVersionProvider.overrideWithValue('1.0.0+1'),
    ],
  );
  addTearDown(c.dispose);
  return c;
}

Widget app(ProviderContainer c) =>
    UncontrolledProviderScope(container: c, child: const BrainVoltApp());

void main() {
  testWidgets('starts in Turkish when the device language is unsupported', (
    tester,
  ) async {
    tester.platformDispatcher.localesTestValue = const [Locale('de')];
    addTearDown(tester.platformDispatcher.clearLocalesTestValue);
    await tester.pumpWidget(app(container(await prefs())));
    await tester.pumpAndSettle();
    expect(find.text('Bilgi yarışmasına hoş geldin!'), findsOneWidget);
  });

  testWidgets('language override is applied and persisted', (tester) async {
    tester.platformDispatcher.localesTestValue = const [Locale('tr', 'TR')];
    addTearDown(tester.platformDispatcher.clearLocalesTestValue);
    final p = await prefs();
    await tester.pumpWidget(app(container(p)));
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('open-settings')));
    await tester.pumpAndSettle();
    expect(find.text('Ayarlar'), findsOneWidget);

    await tester.tap(find.byKey(const Key('locale-en')));
    await tester.pumpAndSettle();
    expect(find.text('Settings'), findsOneWidget);
    expect(p.getString('locale_override'), 'en');

    await tester.tap(find.byKey(const Key('locale-system')));
    await tester.pumpAndSettle();
    expect(find.text('Ayarlar'), findsOneWidget);
    expect(p.getString('locale_override'), isNull);
  });

  testWidgets('a stored override is read on start', (tester) async {
    final c = container(await prefs({'locale_override': 'en'}));
    expect(c.read(localeOverrideProvider), const Locale('en'));
    expect(effectiveLocale(c.read(localeOverrideProvider)), const Locale('en'));
  });
}
