import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/l10n/locale_controller.dart';
import '../../../core/widgets/bv.dart';
import '../../content/data/content_providers.dart';
import '../../content/domain/content_repository.dart';
import '../../social/data/social_providers.dart';
import '../../social/domain/social_repository.dart';
import '../../team/data/team_providers.dart';
import '../../team/domain/team_repository.dart';
import '../domain/match_repository.dart';
import 'format_picker.dart';
import 'topic_picker.dart';

/// The player's teams (to queue as a team).
final myTeamsProvider = FutureProvider.autoDispose<List<Team>>(
  (ref) => ref
      .watch(teamRepositoryProvider)
      .myTeams()
      .catchError((Object _) => <Team>[]),
);

/// The active categories (Category Master).
/// The active categories, kept for the session (D-095): Home loads them
/// in the background so the topic sheet opens with them at once. A new
/// app language loads them again (the names come translated).
final categoriesProvider = FutureProvider<List<CategoryInfo>>((ref) {
  ref.watch(localeOverrideProvider);
  return ref.watch(contentRepositoryProvider).categories();
});

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
  String _format = '1v1';
  String? _team;
  List<UserCard> _party = const [];

  int get _sideSize => switch (_format) {
    '3v3' => 3,
    '4v4' => 4,
    '1v1' => 1,
    _ => 2,
  };

  /// Party members to choose from: teammates when playing as a team,
  /// friends otherwise.
  Future<void> _pickParty(List<Team> teams) async {
    final l10n = context.l10n;
    final List<UserCard> candidates;
    if (_team != null) {
      final me = switch (ref.read(sessionControllerProvider).value) {
        SignedIn(:final session) => session.user.id,
        _ => '',
      };
      candidates = [
        for (final m in teams.firstWhere((t) => t.id == _team).members)
          if (m.card.userId != me) m.card,
      ];
    } else {
      final friends = await ref
          .read(socialRepositoryProvider)
          .friends()
          .catchError((Object _) => <Friend>[]);
      candidates = [for (final f in friends) f.card];
    }
    if (!mounted) {
      return;
    }
    final picked = await showDialog<List<UserCard>>(
      context: context,
      builder: (_) => _PartyDialog(
        title: l10n.pickParty,
        candidates: candidates,
        max: _sideSize - 1,
        initial: _party,
      ),
    );
    if (picked != null) {
      setState(() => _party = picked);
    }
  }

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
    final teams = ref.watch(myTeamsProvider).value ?? const <Team>[];
    final teamFormat = _format != '1v1' && _format != 'couple';
    final lineupOk = _team == null || _party.length == _sideSize - 1;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.modesTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          SectionTitle(l10n.gameModeLabel),
          _Grid(
            children: [
              for (final e in types.entries)
                ChoiceTile(
                  key: Key('type-${e.key}'),
                  icon: _typeIcons[e.key]!,
                  label: e.value,
                  selected: _type == e.key,
                  onTap: () => setState(() {
                    _type = e.key;
                    if (e.key == 'ranked') {
                      _mode = null;
                    }
                  }),
                ),
            ],
          ),
          SectionTitle(l10n.topicLabel),
          Padding(
            padding: const EdgeInsets.only(bottom: 4),
            child: TopicChips(
              key: const Key('mode-topics'),
              categories: categories,
              enabled: !ranked,
              // A topic is Category Master on that category (D-095).
              selected: _mode == 'category_master' ? _category : null,
              onSelected: (id) => setState(() {
                if (id == null) {
                  if (_mode == 'category_master') {
                    _mode = null;
                  }
                  _category = null;
                } else {
                  _mode = 'category_master';
                  _category = id;
                }
              }),
            ),
          ),
          if (ranked)
            Text(
              l10n.topicRankedNote,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          SectionTitle(l10n.categoryModeLabel),
          _Grid(
            children: [
              for (final e in modes.entries)
                Opacity(
                  opacity: ranked && e.key != null ? 0.45 : 1,
                  child: ChoiceTile(
                    key: Key('mode-${e.key ?? 'none'}'),
                    icon: _modeIcons[e.key]!,
                    label: e.value,
                    selected: _mode == e.key,
                    onTap: ranked && e.key != null
                        ? null
                        : () => setState(() => _mode = e.key),
                  ),
                ),
            ],
          ),
          SectionTitle(l10n.formatLabel),
          SizedBox(
            height: 84,
            child: Row(
              children: [
                for (final (i, f) in formats.indexed) ...[
                  if (i > 0) const SizedBox(width: 8),
                  Expanded(
                    child: ChoiceTile(
                      key: Key('format-$f'),
                      icon: f == 'couple'
                          ? Icons.favorite
                          : f == '1v1'
                          ? Icons.people_alt
                          : Icons.groups,
                      label: formatName(l10n, f),
                      vertical: true,
                      selected: _format == f,
                      onTap: () => setState(() {
                        _format = f;
                        _party = const [];
                        _team = null;
                      }),
                    ),
                  ),
                ],
              ],
            ),
          ),
          if (teamFormat) ...[
            const SizedBox(height: 16),
            if (teams.isNotEmpty)
              DropdownButtonFormField<String?>(
                key: const Key('queue-team'),
                initialValue: _team,
                decoration: InputDecoration(labelText: l10n.playAsTeam),
                items: [
                  DropdownMenuItem(child: Text(l10n.noTeam)),
                  for (final t in teams)
                    DropdownMenuItem(
                      value: t.id,
                      child: Text('[${t.tag}] ${t.name}'),
                    ),
                ],
                onChanged: (v) => setState(() {
                  _team = v;
                  _party = const [];
                }),
              ),
            ListTile(
              key: const Key('queue-party'),
              contentPadding: EdgeInsets.zero,
              title: Text(l10n.partyLabel),
              subtitle: Text(
                _party.isEmpty
                    ? '—'
                    : _party.map((c) => c.displayName).join(', '),
              ),
              trailing: TextButton(
                onPressed: () => _pickParty(teams),
                child: Text(l10n.pickParty),
              ),
            ),
          ],
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
          GradientButton(
            key: const Key('mode-search'),
            icon: Icons.search,
            label: l10n.startSearch,
            onPressed: (needsCategory && _category == null) || !lineupOk
                ? null
                : () => context.pushReplacement(
                    Routes.playMode(
                      QueueMode(
                        matchType: _type,
                        categoryMode: _mode,
                        categoryId: needsCategory ? _category : null,
                        format: _format,
                        partyUserIds: [for (final c in _party) c.userId],
                        teamId: _team,
                      ),
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}

const _typeIcons = {
  'quick_duel': Icons.bolt,
  'ranked': Icons.emoji_events,
  'sudden_death': Icons.whatshot,
  'marathon': Icons.directions_run,
};

const _modeIcons = <String?, IconData>{
  null: Icons.layers,
  'blind_challenge': Icons.visibility_off,
  'mixed_battle': Icons.shuffle,
  'category_master': Icons.workspace_premium,
};

/// Two choice tiles per row.
class _Grid extends StatelessWidget {
  const _Grid({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) => GridView.count(
    crossAxisCount: 2,
    shrinkWrap: true,
    physics: const NeverScrollableScrollPhysics(),
    mainAxisSpacing: 10,
    crossAxisSpacing: 10,
    childAspectRatio: 2.6,
    children: children,
  );
}

/// Picks up to [max] players for the party.
class _PartyDialog extends StatefulWidget {
  const _PartyDialog({
    required this.title,
    required this.candidates,
    required this.max,
    required this.initial,
  });

  final String title;
  final List<UserCard> candidates;
  final int max;
  final List<UserCard> initial;

  @override
  State<_PartyDialog> createState() => _PartyDialogState();
}

class _PartyDialogState extends State<_PartyDialog> {
  late final _picked = {for (final c in widget.initial) c.userId};

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AlertDialog(
      title: Text(widget.title),
      content: SizedBox(
        width: double.maxFinite,
        child: ListView(
          shrinkWrap: true,
          children: [
            if (widget.candidates.isEmpty) Text(l10n.friendsEmpty),
            for (final c in widget.candidates)
              CheckboxListTile(
                key: Key('party-${c.userId}'),
                value: _picked.contains(c.userId),
                title: Text(c.displayName),
                onChanged: (on) => setState(() {
                  if (on ?? false) {
                    if (_picked.length < widget.max) {
                      _picked.add(c.userId);
                    }
                  } else {
                    _picked.remove(c.userId);
                  }
                }),
              ),
          ],
        ),
      ),
      actions: [
        FilledButton(
          key: const Key('party-ok'),
          onPressed: () => Navigator.pop(context, [
            for (final c in widget.candidates)
              if (_picked.contains(c.userId)) c,
          ]),
          child: Text(l10n.continueButton),
        ),
      ],
    );
  }
}
