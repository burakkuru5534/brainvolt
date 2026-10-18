import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';

/// Home screen. Game entry points (quick duel, ranked, rooms…) are added by
/// their phases; the screen exists now so routing and localization run end
/// to end.
class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.appTitle),
        actions: [
          IconButton(
            key: const Key('open-settings'),
            tooltip: l10n.settingsTitle,
            icon: const Icon(Icons.settings),
            onPressed: () => context.push(Routes.settings),
          ),
        ],
      ),
      body: Center(
        child: Text(
          l10n.homeWelcome,
          style: Theme.of(context).textTheme.headlineSmall,
        ),
      ),
    );
  }
}
