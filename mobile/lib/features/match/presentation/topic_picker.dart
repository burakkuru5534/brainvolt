import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../content/domain/content_repository.dart';
import 'mode_picker_screen.dart';

/// The topic a player competes in (D-095): one category, or every topic.
class TopicChoice {
  const TopicChoice.all() : categoryId = null;
  const TopicChoice(String this.categoryId);

  /// null: every topic (the server mixes them).
  final String? categoryId;

  bool get all => categoryId == null;
}

/// The localized name of a topic, given the active categories.
String topicName(
  AppLocalizations l10n,
  String? categoryId,
  List<CategoryInfo> categories,
) {
  if (categoryId == null) {
    return l10n.topicAll;
  }
  return categories.where((c) => c.id == categoryId).firstOrNull?.name ??
      l10n.topicAll;
}

/// The icon of a category, by its `icon_key`.
IconData topicIcon(String iconKey) => switch (iconKey) {
  'history' => Icons.account_balance_rounded,
  'geography' => Icons.public_rounded,
  'science' => Icons.science_rounded,
  'arts' => Icons.palette_rounded,
  'sports' => Icons.sports_soccer_rounded,
  'entertainment' => Icons.movie_rounded,
  'technology' => Icons.memory_rounded,
  'general' => Icons.lightbulb_rounded,
  _ => Icons.local_offer_rounded,
};

/// The tint of a category's icon tile, by its `icon_key`.
Color topicColor(String iconKey) => switch (iconKey) {
  'history' => const Color(0xFFF59E0B),
  'geography' => const Color(0xFF22C55E),
  'science' => const Color(0xFF06B6D4),
  'arts' => const Color(0xFFEC4899),
  'sports' => const Color(0xFFF97316),
  'entertainment' => const Color(0xFFA855F7),
  'technology' => const Color(0xFF3B82F6),
  'general' => const Color(0xFFFBBF24),
  _ => const Color(0xFF8B5CF6),
};

/// A category's icon on a tinted gradient tile.
class TopicIcon extends StatelessWidget {
  const TopicIcon(this.iconKey, {this.size = 44, super.key});

  final String iconKey;
  final double size;

  @override
  Widget build(BuildContext context) {
    final c = topicColor(iconKey);
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(size * 0.32),
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [c, Color.lerp(c, Colors.black, 0.35)!],
        ),
        boxShadow: [BoxShadow(color: c.withAlpha(80), blurRadius: 10)],
      ),
      child: Icon(topicIcon(iconKey), color: Colors.white, size: size * 0.56),
    );
  }
}

/// Asks which topic to play before searching (D-095): "All topics" or one
/// category. Players are matched only with players of the same topic.
/// The categories are usually loaded already (Home warms them up).
Future<TopicChoice?> pickTopic(BuildContext context) {
  final container = ProviderScope.containerOf(context);
  if (container.read(categoriesProvider).hasError) {
    container.invalidate(categoriesProvider);
  }
  return showModalBottomSheet<TopicChoice>(
    context: context,
    isScrollControlled: true,
    builder: (_) => const _TopicSheet(),
  );
}

class _TopicSheet extends ConsumerWidget {
  const _TopicSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final text = Theme.of(context).textTheme;
    final bv = BvColors.of(context);
    final categories = ref.watch(categoriesProvider);
    Widget tile(Key key, Widget icon, String name, VoidCallback onTap) =>
        BvCard(
          key: key,
          radius: 18,
          onTap: onTap,
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: [
              icon,
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  name,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: text.titleSmall,
                ),
              ),
            ],
          ),
        );
    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.sizeOf(context).height * 0.85,
        ),
        child: ListView(
          shrinkWrap: true,
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          children: [
            Text(
              l10n.topicPickTitle,
              textAlign: TextAlign.center,
              style: text.headlineSmall,
            ),
            Text(
              l10n.topicPickSub,
              textAlign: TextAlign.center,
              style: text.bodyMedium?.copyWith(color: bv.muted),
            ),
            const SizedBox(height: 14),
            ActionCard(
              key: const Key('topic-all'),
              icon: Icons.shuffle_rounded,
              title: l10n.topicAll,
              subtitle: l10n.topicAllSub,
              onTap: () => Navigator.pop(context, const TopicChoice.all()),
            ),
            const SizedBox(height: 12),
            switch (categories) {
              AsyncValue(value: final value?) => GridView.count(
                crossAxisCount: 2,
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                mainAxisSpacing: 10,
                crossAxisSpacing: 10,
                childAspectRatio: 2.4,
                children: [
                  for (final c in value)
                    tile(
                      Key('topic-${c.id}'),
                      TopicIcon(c.iconKey, size: 40),
                      c.name,
                      () => Navigator.pop(context, TopicChoice(c.id)),
                    ),
                ],
              ),
              AsyncValue(hasError: true) => ErrorView(
                message: l10n.topicLoadFailed,
                retryLabel: l10n.retry,
                onRetry: () => ref.invalidate(categoriesProvider),
              ),
              _ => const Padding(
                padding: EdgeInsets.all(24),
                child: LoadingView(),
              ),
            },
          ],
        ),
      ),
    );
  }
}

/// Topic chips inside a form (mode picker, room settings): "All topics"
/// first, then every category; [enabled] false greys them out (Ranked).
class TopicChips extends StatelessWidget {
  const TopicChips({
    required this.categories,
    required this.selected,
    required this.onSelected,
    this.enabled = true,
    super.key,
  });

  final List<CategoryInfo> categories;
  final String? selected;
  final ValueChanged<String?> onSelected;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        ChoiceChip(
          key: const Key('topic-chip-all'),
          label: Text(l10n.topicAll),
          selected: selected == null,
          onSelected: enabled ? (_) => onSelected(null) : null,
        ),
        for (final c in categories)
          ChoiceChip(
            key: Key('topic-chip-${c.id}'),
            avatar: Icon(topicIcon(c.iconKey), color: topicColor(c.iconKey)),
            label: Text(c.name),
            selected: selected == c.id,
            onSelected: enabled ? (_) => onSelected(c.id) : null,
          ),
      ],
    );
  }
}
