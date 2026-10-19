import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/config.dart';
import '../../../app/providers.dart';
import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../auth/domain/session.dart';
import '../../lobby/presentation/room_screen.dart';
import '../../match/data/match_providers.dart';
import '../../profile/domain/avatars.dart';

/// The caller's live match, if any (back from a crash or another screen).
final currentMatchProvider = FutureProvider.autoDispose<String?>(
  (ref) => ref.watch(matchRepositoryProvider).currentMatch(),
);

/// Challenge events for the caller (websocket.md §5.2): home shows them in
/// a snack bar wherever the player is in the app's main screen.
final lobbyEventsProvider = StreamProvider.autoDispose<ServerFrame>((ref) {
  final rt = ref.watch(realtimeClientProvider)..start();
  return rt.frames.where(
    (f) => f.type == 'challenge.received' || f.type == 'challenge.updated',
  );
});

/// Home screen: greets the player, nudges guests and unverified accounts to
/// secure their progress, and starts a Quick Duel (more modes join with
/// their phases). Development builds also offer a practice match against
/// the server's bot (D-066).
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final session = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session,
      _ => null,
    };
    ref.listen(lobbyEventsProvider, (_, next) {
      final f = next.value;
      if (f == null) {
        return;
      }
      final messenger = ScaffoldMessenger.of(context);
      if (f.type == 'challenge.received') {
        final c = (f.data['challenge'] as Map?)?.cast<String, dynamic>();
        final from = (c?['challenger'] as Map?)?['display_name'] as String?;
        messenger.showSnackBar(
          SnackBar(
            content: Text(l10n.challengeReceived(from ?? '')),
            action: SnackBarAction(
              label: l10n.view,
              onPressed: () => context.push(Routes.challenges),
            ),
          ),
        );
      } else if (f.data['status'] == 'started' &&
          f.data['match_id'] is String) {
        final id = f.data['match_id'] as String;
        messenger.showSnackBar(
          SnackBar(
            content: Text(l10n.challengeAccepted),
            action: SnackBarAction(
              label: l10n.playNow,
              onPressed: () => context.go(Routes.match(id)),
            ),
          ),
        );
      }
    });
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.appTitle),
        actions: [
          IconButton(
            key: const Key('open-challenges'),
            tooltip: l10n.homeChallenges,
            icon: const Icon(Icons.sports_kabaddi),
            onPressed: () => context.push(Routes.challenges),
          ),
          IconButton(
            key: const Key('open-friends'),
            tooltip: l10n.homeFriends,
            icon: const Icon(Icons.people),
            onPressed: () => context.push(Routes.friends),
          ),
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
                  const SizedBox(height: 32),
                  if (ref.watch(currentMatchProvider).value case final id?)
                    FilledButton.icon(
                      key: const Key('resume-match'),
                      icon: const Icon(Icons.play_arrow),
                      onPressed: () => context.go(Routes.match(id)),
                      label: Text(l10n.resumeMatch),
                    )
                  else
                    FilledButton.icon(
                      key: const Key('play-quick-duel'),
                      icon: const Icon(Icons.bolt),
                      onPressed: () => context.push(Routes.play),
                      label: Text(l10n.playQuickDuel),
                    ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    key: const Key('play-modes'),
                    icon: const Icon(Icons.tune),
                    onPressed: () => context.push(Routes.modes),
                    label: Text(l10n.playModes),
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    key: const Key('private-room'),
                    icon: const Icon(Icons.meeting_room),
                    onPressed: () => showRoomSheet(context, ref),
                    label: Text(l10n.privateRoom),
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    key: const Key('open-teams'),
                    icon: const Icon(Icons.groups),
                    onPressed: () => context.push(Routes.teams),
                    label: Text(l10n.homeTeams),
                  ),
                  if (ref.watch(appConfigProvider).flavor == Flavor.dev) ...[
                    const SizedBox(height: 12),
                    OutlinedButton.icon(
                      key: const Key('play-practice'),
                      icon: const Icon(Icons.smart_toy_outlined),
                      onPressed: () => context.push(Routes.practice),
                      label: Text(l10n.playPractice),
                    ),
                  ],
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
