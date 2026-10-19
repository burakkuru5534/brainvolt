import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../match/presentation/match_type_name.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../data/lobby_providers.dart';
import '../domain/lobby_repository.dart';

/// Pending challenges: received first, then sent ones.
final challengesProvider = FutureProvider.autoDispose<List<(bool, Challenge)>>((
  ref,
) async {
  final repo = ref.watch(lobbyRepositoryProvider);
  return [
    for (final c in await repo.challenges(incoming: true)) (true, c),
    for (final c in await repo.challenges(incoming: false)) (false, c),
  ];
});

/// Received and sent challenges. Accepting opens the match; the list
/// follows challenge.received / challenge.updated.
class ChallengesScreen extends ConsumerStatefulWidget {
  const ChallengesScreen({super.key});

  @override
  ConsumerState<ChallengesScreen> createState() => _ChallengesScreenState();
}

class _ChallengesScreenState extends ConsumerState<ChallengesScreen> {
  StreamSubscription<Object>? _sub;

  @override
  void initState() {
    super.initState();
    final rt = ref.read(realtimeClientProvider)..start();
    _sub = rt.frames.listen((f) {
      if (f.type == 'challenge.received' || f.type == 'challenge.updated') {
        ref.invalidate(challengesProvider);
      }
    });
  }

  @override
  void dispose() {
    unawaited(_sub?.cancel());
    super.dispose();
  }

  Future<void> _respond(Challenge c, ChallengeAction action) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final updated = await ref
          .read(lobbyRepositoryProvider)
          .respondChallenge(c.id, action);
      if (updated.matchId case final id? when mounted) {
        context.go(Routes.match(id));
        return;
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
    ref.invalidate(challengesProvider);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.challengesTitle)),
      body: switch (ref.watch(challengesProvider)) {
        AsyncData(:final value) when value.isEmpty => Center(
          child: Text(l10n.challengesEmpty),
        ),
        AsyncData(:final value) => RefreshIndicator(
          onRefresh: () => ref.refresh(challengesProvider.future),
          child: ListView(
            children: [
              for (final (incoming, c) in value)
                ListTile(
                  key: Key('challenge-${c.id}'),
                  leading: Text(
                    avatarFor(
                      incoming ? c.challenger.avatarId : c.target.avatarId,
                    ),
                    style: const TextStyle(fontSize: 28),
                  ),
                  title: Text(
                    incoming
                        ? l10n.challengeFrom(c.challenger.displayName)
                        : l10n.challengeTo(c.target.displayName),
                  ),
                  subtitle: Text(matchTypeName(l10n, c.matchType)),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: incoming
                        ? [
                            IconButton(
                              key: Key('accept-challenge-${c.id}'),
                              tooltip: l10n.accept,
                              icon: const Icon(Icons.check),
                              onPressed: () =>
                                  _respond(c, ChallengeAction.accept),
                            ),
                            IconButton(
                              tooltip: l10n.decline,
                              icon: const Icon(Icons.close),
                              onPressed: () =>
                                  _respond(c, ChallengeAction.decline),
                            ),
                          ]
                        : [
                            TextButton(
                              onPressed: () =>
                                  _respond(c, ChallengeAction.cancel),
                              child: Text(l10n.withdraw),
                            ),
                          ],
                  ),
                ),
            ],
          ),
        ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}
