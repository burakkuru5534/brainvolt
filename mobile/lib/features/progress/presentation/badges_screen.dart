import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/progress_providers.dart';
import '../domain/progress_repository.dart';

final badgesProvider = FutureProvider.autoDispose<List<BadgeInfo>>(
  (ref) => ref.watch(progressRepositoryProvider).badges(),
);

/// The badge catalogue: earned badges as gold medals, the rest locked.
class BadgesScreen extends ConsumerWidget {
  const BadgesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.badgesTitle)),
      body: switch (ref.watch(badgesProvider)) {
        AsyncData(:final value) when value.isEmpty => EmptyState(
          icon: Icons.workspace_premium_rounded,
          text: l10n.badgesEmpty,
        ),
        AsyncData(:final value) => GridView.builder(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
            maxCrossAxisExtent: 220,
            mainAxisSpacing: 12,
            crossAxisSpacing: 12,
            mainAxisExtent: 196,
          ),
          itemCount: value.length,
          itemBuilder: (_, i) => BadgeTile(badge: value[i]),
        ),
        AsyncError(:final error) => ErrorView(
          message: playErrorText(l10n, error),
          retryLabel: l10n.retry,
          onRetry: () => ref.invalidate(badgesProvider),
        ),
        _ => const LoadingView(),
      },
    );
  }
}

/// One badge: a medal (gold when earned, a lock otherwise), its name and
/// what earns it.
class BadgeTile extends StatelessWidget {
  const BadgeTile({required this.badge, this.compact = false, super.key});

  final BadgeInfo badge;

  /// Medal and name only (profile strip).
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final earned = badge.earnedAt != null;
    final medal = Container(
      width: compact ? 54 : 64,
      height: compact ? 54 : 64,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        gradient: earned
            ? LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [bv.gold, bv.gradientEnd],
              )
            : null,
        color: earned ? null : Colors.white.withAlpha(16),
        border: Border.all(
          color: earned ? Colors.white.withAlpha(90) : bv.cardBorder,
          width: 2,
        ),
        boxShadow: [
          if (earned) BoxShadow(color: bv.gold.withAlpha(90), blurRadius: 16),
        ],
      ),
      child: Icon(
        earned ? Icons.workspace_premium_rounded : Icons.lock_outline_rounded,
        color: earned ? Colors.white : bv.muted,
        size: compact ? 28 : 32,
      ),
    );
    return BvCard(
      key: Key('badge-${badge.code}'),
      radius: 18,
      padding: EdgeInsets.all(compact ? 10 : 12),
      child: Column(
        children: [
          medal,
          const SizedBox(height: 8),
          Text(
            badge.name,
            textAlign: TextAlign.center,
            maxLines: compact ? 1 : 2,
            overflow: TextOverflow.ellipsis,
            style: (compact ? text.labelMedium : text.titleSmall)?.copyWith(
              color: earned ? bv.ink : bv.muted,
            ),
          ),
          if (!compact) ...[
            const SizedBox(height: 4),
            Expanded(
              child: Text(
                earned
                    ? badge.description
                    : '${badge.description}\n${l10n.badgeLocked}',
                textAlign: TextAlign.center,
                overflow: TextOverflow.fade,
                style: text.bodySmall,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
