import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/play_error_text.dart';
import '../../social/domain/social_repository.dart';
import '../data/lobby_providers.dart';
import '../domain/lobby_repository.dart';

/// The localized name of a match type.
String matchTypeName(AppLocalizations l, String wire) =>
    switch (MatchType.parse(wire)) {
      MatchType.suddenDeath => l.matchTypeSuddenDeath,
      MatchType.marathon => l.matchTypeMarathon,
      _ => l.matchTypeQuickDuel,
    };

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
            child: Text(matchTypeName(l10n, t.wire)),
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
