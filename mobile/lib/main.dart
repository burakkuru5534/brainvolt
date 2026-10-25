import 'dart:async';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app/app.dart';
import 'app/config.dart';
import 'app/providers.dart';
import 'core/ads/ad_gateway.dart';
import 'core/network/api_client.dart';
import 'core/push/push_service.dart';
import 'features/ads/presentation/ads.dart';
import 'features/auth/data/auth_providers.dart';
import 'firebase_options.dart';

/// Entry point: build the injected values, then start the widget tree.
/// Run with `flutter run --dart-define-from-file=env/dev.json`.
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  _setUpLogging();

  FlutterError.onError = (details) {
    FlutterError.presentError(details);
    debugPrint('FlutterError: ${details.exceptionAsString()}');
  };
  PlatformDispatcher.instance.onError = (error, stack) {
    debugPrint('PlatformDispatcher error: $error\n$stack');
    return true; // Keep the engine alive
  };

  final config = AppConfig.fromEnvironment();

  var pushService = const NoPushService() as PushService;
  try {
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    ).timeout(const Duration(seconds: 5));
    final emulator = config.authEmulator;
    if (emulator != null) {
      // Local development signs in against the Auth emulator (ADR-0003).
      await FirebaseAuth.instance.useAuthEmulator(emulator.host, emulator.port);
    }
    pushService = FirebasePushService();
  } catch (e, st) {
    debugPrint('Firebase initialization failed: $e\n$st');
  }

  final prefs = await SharedPreferences.getInstance();
  // Ads (D-103): consent and the SDK start after the first frame.
  final ads = GoogleAdGateway(AdUnits.forBuild(config.flavor));
  final info = await PackageInfo.fromPlatform();
  runApp(
    ProviderScope(
      // Failed requests are retried by the user (retry buttons, pull to
      // refresh), not by Riverpod in the background.
      retry: noRetry,
      overrides: [
        appConfigProvider.overrideWithValue(config),
        sharedPreferencesProvider.overrideWithValue(prefs),
        appVersionProvider.overrideWithValue(
          '${info.version}+${info.buildNumber}',
        ),
        pushServiceProvider.overrideWithValue(pushService),
        adGatewayProvider.overrideWithValue(ads),
        tokenSourceProvider.overrideWith(
          (ref) => AuthTokenSource(ref.watch(authRepositoryProvider)),
        ),
      ],
      child: const BrainVoltApp(),
    ),
  );
  WidgetsBinding.instance.addPostFrameCallback((_) => unawaited(ads.init()));
}

/// Logs go to the debug console; release builds stay silent (ADR-0003).
void _setUpLogging() {
  if (kReleaseMode) {
    Logger.root.level = Level.OFF;
    return;
  }
  Logger.root.level = Level.INFO;
  Logger.root.onRecord.listen((r) {
    debugPrint(
      '${r.level.name} ${r.loggerName}: ${r.message}'
      '${r.error == null ? '' : ' (${r.error})'}',
    );
  });
}
