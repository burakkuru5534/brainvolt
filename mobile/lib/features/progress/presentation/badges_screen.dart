import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/progress_providers.dart';
import '../domain/progress_repository.dart';

final badgesProvider = FutureProvider.autoDispose<List<BadgeInfo>>(
  (ref) => ref.watch(progressRepositoryProvider).badges(),
);

/// The badge catalogue: earned badges in colour, the rest locked.
class BadgesScreen extends ConsumerWidget {
  const BadgesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.badgesTitle)),
      body: switch (ref.watch(badgesProvider)) {
        AsyncData(:final value) => ListView(
          children: [
            for (final b in value)
              ListTile(
                key: Key('badge-${b.code}'),
                leading: Icon(
                  b.earnedAt != null
                      ? Icons.workspace_premium
                      : Icons.lock_outline,
                  color: b.earnedAt != null ? Colors.amber : null,
                ),
                title: Text(b.name),
                subtitle: Text(
                  b.earnedAt != null
                      ? b.description
                      : '${b.description}\n${l10n.badgeLocked}',
                ),
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
