import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../domain/match_repository.dart';
import 'play_controllers.dart';
import 'result_screen.dart';

/// How long after the end "Play again" may be offered (D-091; the
/// server's lobby RematchWindow).
const playAgainWindow = Duration(seconds: 60);

/// Whether [r] can be played again from its end screen: a completed match
/// that is not a private room match (those go back to the room) or a
/// tournament match, still inside [playAgainWindow].
bool canPlayAgain(MatchResult r, {DateTime? now}) =>
    r.status == 'completed' &&
    r.origin != 'private_room' &&
    r.origin != 'tournament' &&
    (r.endedAt?.add(playAgainWindow).isAfter(now ?? DateTime.now()) ?? false);

/// "Play again" on the match end and result screens (D-091, replaces
/// revenge): every player — winner or loser, every format — can offer
/// it; the others see the offer and accept or decline; once everyone
/// accepted, the server starts the same match again and all players go
/// there (websocket.md rematch.*).
class PlayAgainPanel extends ConsumerStatefulWidget {
  const PlayAgainPanel({required this.matchId, super.key});

  final String matchId;

  @override
  ConsumerState<PlayAgainPanel> createState() => _PlayAgainPanelState();
}

class _PlayAgainPanelState extends ConsumerState<PlayAgainPanel> {
  // Live offers only matter inside the play-again window (D-091), so the
  // socket is held that long and no longer (D-097).
  RealtimeLease? _lease;
  Timer? _window;

  @override
  void initState() {
    super.initState();
    _lease = ref.read(realtimeClientProvider).acquire('play-again');
    _window = Timer(rematchLeaseFor, _release);
  }

  void _release() {
    _window?.cancel();
    _lease?.release();
  }

  @override
  void dispose() {
    _release();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final matchId = widget.matchId;
    final l10n = context.l10n;
    // Listen from the first frame so an early offer is not missed.
    ref.listen(rematchControllerProvider(matchId), (_, next) {
      if (next.newMatchId case final id?) {
        context.go(Routes.match(id));
      } else if (next.declined) {
        _release();
      }
    });
    final s = ref.watch(rematchControllerProvider(matchId));
    final result = ref.watch(matchResultProvider(matchId)).value;
    if (result == null) {
      return const SizedBox.shrink();
    }
    final active = s.asked || s.othersAccepted || s.declined;
    if (!active && !canPlayAgain(result)) {
      return const SizedBox.shrink();
    }
    final ctrl = ref.read(rematchControllerProvider(matchId).notifier);
    final text = Theme.of(context).textTheme;
    return Padding(
      key: const Key('play-again'),
      padding: const EdgeInsets.only(top: 12),
      child: BvCard(
        padding: const EdgeInsets.all(14),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (s.declined)
              Text(
                l10n.rematchDeclined,
                key: const Key('rematch-declined'),
                textAlign: TextAlign.center,
              )
            else if (s.asked)
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                  const SizedBox(width: 10),
                  Flexible(
                    child: Text(
                      l10n.rematchWaiting,
                      key: const Key('rematch-waiting'),
                    ),
                  ),
                ],
              )
            else ...[
              if (s.othersAccepted)
                Padding(
                  padding: const EdgeInsets.only(bottom: 10),
                  child: Text(
                    l10n.rematchOffered,
                    key: const Key('rematch-offered'),
                    textAlign: TextAlign.center,
                    style: text.titleMedium,
                  ),
                ),
              Wrap(
                alignment: WrapAlignment.center,
                spacing: 8,
                runSpacing: 8,
                children: [
                  FilledButton.icon(
                    key: const Key('rematch'),
                    icon: const Icon(Icons.replay),
                    onPressed: () =>
                        ctrl.respond(accept: true).catchError((Object _) {}),
                    label: Text(
                      s.othersAccepted ? l10n.accept : l10n.rematchButton,
                    ),
                  ),
                  if (s.othersAccepted)
                    TextButton(
                      key: const Key('rematch-decline'),
                      onPressed: () =>
                          ctrl.respond(accept: false).catchError((Object _) {}),
                      child: Text(l10n.decline),
                    ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}
