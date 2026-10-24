import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';

/// Challenge and tournament-match events for the signed-in caller
/// (websocket.md §5.2).
final lobbyEventsProvider = StreamProvider.autoDispose<ServerFrame>((ref) {
  if (ref.watch(sessionControllerProvider).value is! SignedIn) {
    return const Stream.empty();
  }
  final rt = ref.watch(realtimeClientProvider)..start();
  return rt.frames.where(
    (f) =>
        f.type == 'challenge.received' ||
        f.type == 'challenge.updated' ||
        f.type == 'tournament.match_ready',
  );
});

/// Shows challenges in a snack bar on every screen (D-089). The match
/// screens are left alone: the end screen answers "Play again" itself.
class ChallengeListener extends ConsumerWidget {
  const ChallengeListener({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.listen(lobbyEventsProvider, (_, next) {
      final f = next.value;
      final router = ref.read(routerProvider);
      if (f == null ||
          router.routerDelegate.currentConfiguration.uri.path.startsWith(
            '/match/',
          )) {
        return;
      }
      final l10n = context.l10n;
      final messenger = ScaffoldMessenger.of(context);
      if (f.type == 'tournament.match_ready') {
        // The tournament screen goes there itself.
        if (f.data['match_id'] case final String id
            when !router.routerDelegate.currentConfiguration.uri.path
                .startsWith('/tournaments/')) {
          messenger.showSnackBar(
            SnackBar(
              content: Text(l10n.tournamentMatchStarted),
              action: SnackBarAction(
                label: l10n.playNow,
                onPressed: () => router.go(Routes.match(id)),
              ),
            ),
          );
        }
      } else if (f.type == 'challenge.received') {
        final c = (f.data['challenge'] as Map?)?.cast<String, dynamic>();
        final from = (c?['challenger'] as Map?)?['display_name'] as String?;
        messenger.showSnackBar(
          SnackBar(
            content: Text(l10n.challengeReceived(from ?? '')),
            action: SnackBarAction(
              label: l10n.view,
              onPressed: () => router.push(Routes.challenges),
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
              onPressed: () => router.go(Routes.match(id)),
            ),
          ),
        );
      }
    });
    return child;
  }
}
