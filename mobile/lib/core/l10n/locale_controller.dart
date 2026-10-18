import 'package:flutter/widgets.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../app/providers.dart';
import 'l10n.dart';

part 'locale_controller.g.dart';

const _prefsKey = 'locale_override';

/// The user's language choice; `null` follows the device (roadmap 1.7.5).
/// Persisted in shared preferences.
@Riverpod(keepAlive: true)
class LocaleOverride extends _$LocaleOverride {
  @override
  Locale? build() {
    final code = ref.watch(sharedPreferencesProvider).getString(_prefsKey);
    return supportedLocales.where((l) => l.languageCode == code).firstOrNull;
  }

  /// Sets or clears (`null`) the override.
  Future<void> set(Locale? locale) async {
    final prefs = ref.read(sharedPreferencesProvider);
    if (locale == null) {
      await prefs.remove(_prefsKey);
    } else {
      await prefs.setString(_prefsKey, locale.languageCode);
    }
    state = locale;
  }
}

/// The locale the app shows and sends as Accept-Language: the override,
/// else the device languages *now* (so a change in system settings applies
/// without a restart), else Turkish.
Locale effectiveLocale(Locale? override) =>
    resolveLocale(override, WidgetsBinding.instance.platformDispatcher.locales);
