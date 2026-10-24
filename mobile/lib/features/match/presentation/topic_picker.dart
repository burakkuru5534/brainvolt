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

/// Asks which topic to play before searching (D-095): "All topics" or one
/// category. Players are matched only with players of the same topic.
Future<TopicChoice?> pickTopic(BuildContext context) =>
    showModalBottomSheet<TopicChoice>(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _TopicSheet(),
    );

class _TopicSheet extends ConsumerWidget {
  const _TopicSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final text = Theme.of(context).textTheme;
    final bv = BvColors.of(context);
    final categories = ref.watch(categoriesProvider);
    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.sizeOf(context).height * 0.8,
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
              icon: Icons.shuffle,
              title: l10n.topicAll,
              subtitle: l10n.topicAllSub,
              onTap: () => Navigator.pop(context, const TopicChoice.all()),
            ),
            const SizedBox(height: 12),
            switch (categories) {
              AsyncData(:final value) => Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final c in value)
                    ActionChip(
                      key: Key('topic-${c.id}'),
                      avatar: const Icon(Icons.local_offer_outlined, size: 18),
                      label: Text(c.name),
                      onPressed: () =>
                          Navigator.pop(context, TopicChoice(c.id)),
                    ),
                ],
              ),
              AsyncError() => Text(l10n.topicLoadFailed),
              _ => const LoadingView(),
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
            label: Text(c.name),
            selected: selected == c.id,
            onSelected: enabled ? (_) => onSelected(c.id) : null,
          ),
      ],
    );
  }
}
