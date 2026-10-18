/// Localization helpers (D-001, D-002): Turkish and English, Turkish is the
/// template and the fallback.
library;

import 'package:flutter/widgets.dart';

import 'generated/app_localizations.dart';

export 'generated/app_localizations.dart';

/// Supported locales, fallback first.
const supportedLocales = [Locale('tr'), Locale('en')];

/// Picks the app locale: the user's choice, else the device language if
/// supported, else Turkish (conventions.md §4).
Locale resolveLocale(Locale? override, List<Locale>? device) {
  if (override != null) {
    return override;
  }
  for (final l in device ?? const <Locale>[]) {
    final match = supportedLocales.where(
      (s) => s.languageCode == l.languageCode,
    );
    if (match.isNotEmpty) {
      return match.first;
    }
  }
  return supportedLocales.first;
}

/// `context.l10n.appTitle`.
extension L10nContext on BuildContext {
  AppLocalizations get l10n => AppLocalizations.of(this);
}
