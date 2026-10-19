import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../content/data/content_providers.dart';
import '../../content/domain/content_repository.dart';
import '../domain/match_repository.dart';

/// The active categories (Category Master).
final categoriesProvider = FutureProvider.autoDispose<List<CategoryInfo>>(
  (ref) => ref.watch(contentRepositoryProvider).categories(),
);

/// Picks the queue: match type and category mode (mode-matrix §2; Ranked
/// has no category mode, Category Master needs a category).
class ModePickerScreen extends ConsumerStatefulWidget {
  const ModePickerScreen({super.key});

  @override
  ConsumerState<ModePickerScreen> createState() => _ModePickerScreenState();
}

class _ModePickerScreenState extends ConsumerState<ModePickerScreen> {
  String _type = 'quick_duel';
  String? _mode;
  String? _category;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final types = {
      'quick_duel': l10n.matchTypeQuickDuel,
      'ranked': l10n.matchTypeRanked,
      'sudden_death': l10n.matchTypeSuddenDeath,
      'marathon': l10n.matchTypeMarathon,
    };
    final modes = <String?, String>{
      null: l10n.categoryModeNone,
      'blind_challenge': l10n.categoryModeBlind,
      'mixed_battle': l10n.categoryModeMixed,
      'category_master': l10n.categoryModeMaster,
    };
    final ranked = _type == 'ranked';
    final needsCategory = _mode == 'category_master';
    final categories = ref.watch(categoriesProvider).value ?? const [];
    return Scaffold(
      appBar: AppBar(title: Text(l10n.modesTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Wrap(
            spacing: 8,
            children: [
              for (final e in types.entries)
                ChoiceChip(
                  key: Key('type-${e.key}'),
                  label: Text(e.value),
                  selected: _type == e.key,
                  onSelected: (_) => setState(() {
                    _type = e.key;
                    if (e.key == 'ranked') {
                      _mode = null;
                    }
                  }),
                ),
            ],
          ),
          const SizedBox(height: 24),
          Text(
            l10n.categoryModeLabel,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            children: [
              for (final e in modes.entries)
                ChoiceChip(
                  key: Key('mode-${e.key ?? 'none'}'),
                  label: Text(e.value),
                  selected: _mode == e.key,
                  onSelected: ranked && e.key != null
                      ? null
                      : (_) => setState(() => _mode = e.key),
                ),
            ],
          ),
          if (needsCategory) ...[
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              key: const Key('category'),
              initialValue: _category,
              decoration: InputDecoration(labelText: l10n.categoryLabel),
              items: [
                for (final c in categories)
                  DropdownMenuItem(value: c.id, child: Text(c.name)),
              ],
              onChanged: (v) => setState(() => _category = v),
            ),
          ],
          const SizedBox(height: 32),
          FilledButton.icon(
            key: const Key('mode-search'),
            icon: const Icon(Icons.search),
            onPressed: needsCategory && _category == null
                ? null
                : () => context.pushReplacement(
                    Routes.playMode(
                      QueueMode(
                        matchType: _type,
                        categoryMode: _mode,
                        categoryId: needsCategory ? _category : null,
                      ),
                    ),
                  ),
            label: Text(l10n.startSearch),
          ),
        ],
      ),
    );
  }
}
