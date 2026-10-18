/// App-wide values created before the widget tree in `main.dart` and
/// injected with `ProviderScope.overrides` (Riverpod doubles as DI, ADR-0003).
library;

import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'config.dart';

part 'providers.g.dart';

const _injected =
    'is created in main() and injected via ProviderScope.overrides';

/// Build-time configuration.
@Riverpod(keepAlive: true)
AppConfig appConfig(Ref ref) =>
    throw UnimplementedError('appConfigProvider $_injected');

/// Shared preferences (locale override, onboarding flags).
@Riverpod(keepAlive: true)
SharedPreferences sharedPreferences(Ref ref) =>
    throw UnimplementedError('sharedPreferencesProvider $_injected');

/// "version+build", sent as X-App-Version (conventions.md §2).
@Riverpod(keepAlive: true)
String appVersion(Ref ref) =>
    throw UnimplementedError('appVersionProvider $_injected');
