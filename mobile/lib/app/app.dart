import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/l10n/l10n.dart';
import '../core/l10n/locale_controller.dart';
import '../core/realtime/realtime_lifecycle.dart';
import '../core/widgets/wc_background.dart';
import '../features/lobby/presentation/challenge_listener.dart';
import 'router.dart';
import 'theme.dart';

final _theme = buildTheme(Brightness.dark, (page) => WcBackground(child: page));

/// Root widget.
class BrainVoltApp extends ConsumerWidget {
  const BrainVoltApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) => MaterialApp.router(
    debugShowCheckedModeBanner: false,
    onGenerateTitle: (context) => context.l10n.appTitle,
    // BrainVolt is dark only (D-099); every page gets the star field.
    theme: _theme,
    darkTheme: _theme,
    themeMode: ThemeMode.dark,
    routerConfig: ref.watch(routerProvider),
    // The star field also sits under the navigator, for pages without a
    // Material transition (go_router may build plain pages).
    builder: (context, child) => WcBackground(
      child: RealtimeLifecycle(
        child: ChallengeListener(child: child ?? const SizedBox.shrink()),
      ),
    ),
    // null = follow the device; Flutter re-resolves when the device language changes.
    locale: ref.watch(localeOverrideProvider),
    localeListResolutionCallback: (device, _) => resolveLocale(null, device),
    supportedLocales: supportedLocales,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
  );
}
