import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../auth/domain/session.dart';
import '../../profile/domain/avatars.dart';

/// Home screen. Game entry points (quick duel, ranked, rooms…) are added by
/// their phases; for now it greets the player and nudges guests and
/// unverified accounts to secure their progress.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final session = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session,
      _ => null,
    };
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
      body: Column(
        children: [
          if (session?.tier == Tier.guest)
            _Banner(
              key: const Key('guest-banner'),
              text: l10n.homeGuestBanner,
              action: l10n.linkAccountAction,
              onPressed: () => context.push(Routes.account),
            )
          else if (session?.flags.emailVerificationRequired ?? false)
            _Banner(
              key: const Key('verify-banner'),
              text: l10n.homeVerifyBanner,
              action: l10n.verifyAction,
              onPressed: () => context.push(Routes.verifyEmail),
            ),
          Expanded(
            child: Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (session != null) ...[
                    Text(
                      avatarFor(session.profile.avatarId),
                      style: const TextStyle(fontSize: 64),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      l10n.homeGreeting(session.profile.displayName),
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 8),
                  ],
                  Text(
                    l10n.homeWelcome,
                    style: Theme.of(context).textTheme.headlineSmall,
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Banner extends StatelessWidget {
  const _Banner({
    required this.text,
    required this.action,
    required this.onPressed,
    super.key,
  });

  final String text;
  final String action;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => MaterialBanner(
    content: Text(text),
    actions: [TextButton(onPressed: onPressed, child: Text(action))],
  );
}
