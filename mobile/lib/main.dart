import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app/app.dart';
import 'app/config.dart';
import 'app/providers.dart';

/// Entry point: build the injected values, then start the widget tree.
/// Run with `flutter run --dart-define-from-file=env/dev.json`.
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final config = AppConfig.fromEnvironment();
  final prefs = await SharedPreferences.getInstance();
  final info = await PackageInfo.fromPlatform();
  runApp(
    ProviderScope(
      overrides: [
        appConfigProvider.overrideWithValue(config),
        sharedPreferencesProvider.overrideWithValue(prefs),
        appVersionProvider.overrideWithValue(
          '${info.version}+${info.buildNumber}',
        ),
      ],
      child: const BrainVoltApp(),
    ),
  );
}
