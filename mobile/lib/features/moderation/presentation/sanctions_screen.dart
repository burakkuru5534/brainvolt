import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/moderation_providers.dart';
import '../domain/moderation_repository.dart';

final mySanctionsProvider = FutureProvider.autoDispose<List<Sanction>>(
  (ref) => ref.watch(moderationRepositoryProvider).mySanctions(),
);

String sanctionName(AppLocalizations l10n, String kind) => switch (kind) {
  'chat_mute' => l10n.sanctionChatMute,
  'ranked_ban' => l10n.sanctionRankedBan,
  'matchmaking_ban' => l10n.sanctionMatchmakingBan,
  'social_ban' => l10n.sanctionSocialBan,
  'suspension' => l10n.sanctionSuspension,
  _ => l10n.sanctionWarning,
};

/// The player's sanctions and warnings with their appeal state; one
/// appeal per sanction (schema-moderation §2).
class SanctionsScreen extends ConsumerWidget {
  const SanctionsScreen({super.key});

  Future<void> _appeal(BuildContext context, WidgetRef ref, Sanction s) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    final text = await showDialog<String>(
      context: context,
      builder: (_) => const _AppealDialog(),
    );
    if (text == null) {
      return;
    }
    try {
      await ref.read(moderationRepositoryProvider).appeal(s.id, text);
      messenger.showSnackBar(SnackBar(content: Text(l10n.appealSent)));
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
    ref.invalidate(mySanctionsProvider);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final locale = Localizations.localeOf(context).toLanguageTag();
    return Scaffold(
      appBar: AppBar(title: Text(l10n.sanctionsTitle)),
      body: switch (ref.watch(mySanctionsProvider)) {
        AsyncData(:final value) when value.isEmpty => Center(
          child: Text(l10n.sanctionsEmpty, key: const Key('sanctions-empty')),
        ),
        AsyncData(:final value) => ListView(
          children: [
            for (final s in value)
              ListTile(
                key: Key('sanction-${s.id}'),
                leading: Icon(s.active ? Icons.gavel : Icons.history),
                title: Text(sanctionName(l10n, s.kind)),
                subtitle: Text(
                  [
                    if (!s.active && s.kind != 'warning')
                      l10n.sanctionEnded
                    else if (s.endsAt case final end?)
                      l10n.sanctionUntil(
                        MaterialLocalizations.of(context).formatShortDate(end),
                      )
                    else if (s.kind != 'warning')
                      l10n.sanctionPermanent,
                    switch (s.appealStatus) {
                      'pending' => l10n.appealPending,
                      'accepted' => l10n.appealAccepted,
                      'rejected' => l10n.appealRejected,
                      _ => '',
                    },
                  ].where((t) => t.isNotEmpty).join(' · '),
                  key: Key('sanction-info-${s.id}-$locale'),
                ),
                trailing: s.appealStatus == null && s.active
                    ? TextButton(
                        key: Key('appeal-${s.id}'),
                        onPressed: () => _appeal(context, ref, s),
                        child: Text(l10n.appealButton),
                      )
                    : null,
              ),
          ],
        ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _AppealDialog extends StatefulWidget {
  const _AppealDialog();

  @override
  State<_AppealDialog> createState() => _AppealDialogState();
}

class _AppealDialogState extends State<_AppealDialog> {
  final _text = TextEditingController();

  @override
  void initState() {
    super.initState();
    _text.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final text = _text.text.trim();
    return AlertDialog(
      title: Text(l10n.appealButton),
      content: TextField(
        key: const Key('appeal-text'),
        controller: _text,
        maxLines: 4,
        maxLength: 1000,
        decoration: InputDecoration(hintText: l10n.appealHint),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('appeal-send'),
          onPressed: text.length < 10
              ? null
              : () => Navigator.pop(context, text),
          child: Text(l10n.reportSend),
        ),
      ],
    );
  }
}
