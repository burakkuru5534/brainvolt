import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../../team/domain/team_repository.dart';
import '../data/progress_providers.dart';
import '../domain/progress_repository.dart';

typedef BoardFilter = ({
  BoardPeriod period,
  BoardEntity entity,
  BoardScope scope,
});

/// The first page of a board.
final boardProvider = FutureProvider.autoDispose.family<Board, BoardFilter>(
  (ref, f) => ref
      .watch(progressRepositoryProvider)
      .leaderboard(period: f.period, entity: f.entity, scope: f.scope),
);

String periodName(AppLocalizations l10n, BoardPeriod p) => switch (p) {
  BoardPeriod.daily => l10n.periodDaily,
  BoardPeriod.weekly => l10n.periodWeekly,
  BoardPeriod.monthly => l10n.periodMonthly,
  BoardPeriod.allTime => l10n.periodAllTime,
  BoardPeriod.season => l10n.periodSeason,
};

String entityName(AppLocalizations l10n, BoardEntity e) => switch (e) {
  BoardEntity.user => l10n.entityUser,
  BoardEntity.team => l10n.entityTeam,
  BoardEntity.couple => l10n.entityCouple,
};

String scopeName(AppLocalizations l10n, BoardScope s) => switch (s) {
  BoardScope.global => l10n.scopeGlobal,
  BoardScope.country => l10n.scopeCountry,
  BoardScope.friends => l10n.scopeFriends,
};

/// Leaderboards with period, entity and scope filters; the caller's own
/// row stays at the bottom on individual boards.
class LeaderboardsScreen extends ConsumerStatefulWidget {
  const LeaderboardsScreen({super.key});

  @override
  ConsumerState<LeaderboardsScreen> createState() => _LeaderboardsScreenState();
}

class _LeaderboardsScreenState extends ConsumerState<LeaderboardsScreen> {
  var _period = BoardPeriod.weekly;
  var _entity = BoardEntity.user;
  var _scope = BoardScope.global;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final filter = (period: _period, entity: _entity, scope: _scope);
    final board = ref.watch(boardProvider(filter));
    Widget chips<T>(
      List<T> values,
      T selected,
      String Function(T) name,
      void Function(T) pick,
      String key,
    ) => SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Row(
        children: [
          for (final v in values)
            Padding(
              padding: const EdgeInsets.only(right: 6),
              child: ChoiceChip(
                key: Key('$key-${(v as Enum).name}'),
                label: Text(name(v)),
                selected: v == selected,
                onSelected: (_) => setState(() => pick(v)),
              ),
            ),
        ],
      ),
    );
    return Scaffold(
      appBar: AppBar(title: Text(l10n.leaderboardsTitle)),
      body: Column(
        children: [
          const SizedBox(height: 8),
          chips(
            BoardPeriod.values,
            _period,
            (p) => periodName(l10n, p),
            (p) => _period = p,
            'period',
          ),
          chips(BoardEntity.values, _entity, (e) => entityName(l10n, e), (e) {
            _entity = e;
            if (e != BoardEntity.user && _scope == BoardScope.friends) {
              _scope = BoardScope.global;
            }
          }, 'entity'),
          chips(
            [
              BoardScope.global,
              BoardScope.country,
              if (_entity == BoardEntity.user) BoardScope.friends,
            ],
            _scope,
            (s) => scopeName(l10n, s),
            (s) => _scope = s,
            'scope',
          ),
          const Divider(),
          Expanded(
            child: switch (board) {
              AsyncData(:final value) when value.entries.isEmpty => Center(
                child: Text(
                  l10n.leaderboardEmpty,
                  key: const Key('board-empty'),
                ),
              ),
              AsyncData(:final value) => ListView(
                children: [
                  for (final e in value.entries)
                    _Row(entry: e, mine: e.entityId == value.me?.entityId),
                ],
              ),
              AsyncError(:final error) => Center(
                child: Text(playErrorText(l10n, error)),
              ),
              _ => const Center(child: CircularProgressIndicator()),
            },
          ),
          if (board.value?.me case final me?) ...[
            const Divider(height: 1),
            _Row(entry: me, mine: true, label: l10n.leaderboardMe),
          ],
        ],
      ),
    );
  }
}

class _Row extends StatelessWidget {
  const _Row({required this.entry, required this.mine, this.label});

  final BoardEntry entry;
  final bool mine;
  final String? label;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final leading = entry.emblemId.isNotEmpty
        ? emblemEmoji[entry.emblemId] ?? '🛡️'
        : entry.avatarId.isNotEmpty
        ? avatarFor(entry.avatarId)
        : '👥';
    return ListTile(
      key: Key('board-row-${entry.entityId}'),
      selected: mine,
      leading: SizedBox(
        width: 72,
        child: Row(
          children: [
            SizedBox(width: 32, child: Text('#${entry.rank}')),
            Text(leading, style: const TextStyle(fontSize: 24)),
          ],
        ),
      ),
      title: Text(label != null ? '$label · ${entry.name}' : entry.name),
      trailing: Text(l10n.boardPoints(entry.score)),
    );
  }
}
