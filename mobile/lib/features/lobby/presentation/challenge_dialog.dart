import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/match_type_name.dart';
import '../../match/presentation/play_error_text.dart';
import '../../social/domain/social_repository.dart';
import '../data/lobby_providers.dart';
import '../domain/lobby_repository.dart';

/// Picks a match type and sends [target] a direct challenge (1v1).
Future<void> showChallengeDialog(
  BuildContext context,
  WidgetRef ref,
  UserCard target,
) async {
  final l10n = context.l10n;
  final type = await showDialog<MatchType>(
    context: context,
    builder: (context) => SimpleDialog(
      title: Text(l10n.challengeTitle(target.displayName)),
      children: [
        for (final t in MatchType.values)
          SimpleDialogOption(
            key: Key('challenge-${t.wire}'),
            onPressed: () => Navigator.pop(context, t),
            child: Row(
              children: [
                IconBadge(switch (t) {
                  MatchType.quickDuel => Icons.bolt,
                  MatchType.suddenDeath => Icons.whatshot_rounded,
                  MatchType.marathon => Icons.directions_run_rounded,
                }, size: 40),
                const SizedBox(width: 12),
                Text(
                  matchTypeName(l10n, t.wire),
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ],
            ),
          ),
      ],
    ),
  );
  if (type == null || !context.mounted) {
    return;
  }
  final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
  try {
    await ref.read(lobbyRepositoryProvider).challenge(target.userId, type);
    messenger.showSnackBar(SnackBar(content: Text(l10n.challengeSent)));
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
}
