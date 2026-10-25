import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/cosmetics.dart';
import '../../home/presentation/main_nav.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../../social/presentation/player_profile.dart';
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
    Widget chips<T extends Enum>(
      List<T> values,
      T selected,
      String Function(T) name,
      void Function(T) pick,
      String key,
    ) => SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          for (final v in values)
            Padding(
              padding: const EdgeInsets.only(right: 6),
              child: ChoiceChip(
                key: Key('$key-${v.name}'),
                label: Text(name(v)),
                showCheckmark: false,
                visualDensity: VisualDensity.compact,
                selected: v == selected,
                onSelected: (_) => setState(() => pick(v)),
              ),
            ),
        ],
      ),
    );
    return Scaffold(
      bottomNavigationBar: const MainNavBar(current: MainTab.leaderboards),
      appBar: AppBar(title: Text(l10n.leaderboardsTitle)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
            child: WcSegmented<BoardScope>(
              keyPrefix: 'scope',
              selected: _scope,
              onChanged: (s) => setState(() => _scope = s),
              segments: [
                for (final s in [
                  BoardScope.global,
                  BoardScope.country,
                  if (_entity == BoardEntity.user) BoardScope.friends,
                ])
                  (s, scopeName(l10n, s)),
              ],
            ),
          ),
          chips(
            BoardPeriod.values,
            _period,
            (p) => periodName(l10n, p),
            (p) => _period = p,
            'period',
          ),
          const SizedBox(height: 6),
          chips(BoardEntity.values, _entity, (e) => entityName(l10n, e), (e) {
            _entity = e;
            if (e != BoardEntity.user && _scope == BoardScope.friends) {
              _scope = BoardScope.global;
            }
          }, 'entity'),
          const SizedBox(height: 6),
          Expanded(
            child: switch (board) {
              AsyncData(:final value) when value.entries.isEmpty => EmptyState(
                icon: Icons.leaderboard_rounded,
                text: l10n.leaderboardEmpty,
                textKey: const Key('board-empty'),
              ),
              AsyncData(:final value) => ListView(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                children: [
                  _Podium(
                    entries: value.entries.take(3).toList(),
                    me: value.me?.entityId,
                  ),
                  const SizedBox(height: 12),
                  for (final e in value.entries.skip(3))
                    Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: _Row(
                        entry: e,
                        mine: e.entityId == value.me?.entityId,
                      ),
                    ),
                ],
              ),
              AsyncError(:final error) => ErrorView(
                message: playErrorText(l10n, error),
                retryLabel: l10n.retry,
                onRetry: () => ref.invalidate(boardProvider(filter)),
              ),
              _ => const LoadingView(),
            },
          ),
          if (board.value?.me case final me?)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
              child: _Row(entry: me, mine: true, label: l10n.leaderboardMe),
            ),
        ],
      ),
    );
  }
}

String _leading(BoardEntry entry) => entry.emblemId.isNotEmpty
    ? emblemEmoji[entry.emblemId] ?? '🛡️'
    : entry.avatarId.isNotEmpty
    ? avatarFor(entry.avatarId)
    : '👥';

/// Opens a player's card (D-101); teams, couples and the caller have none.
VoidCallback? _openPlayer(BuildContext context, BoardEntry e, String? me) {
  final user = e.user;
  if (user == null || e.entityId == me) {
    return null;
  }
  return () => showPlayerSheet(context, user);
}

const _medals = [Color(0xFFFBBF24), Color(0xFFCBD5E1), Color(0xFFF59E5B)];

/// The top three on steps: second, first (with the crown), third.
class _Podium extends StatelessWidget {
  const _Podium({required this.entries, this.me});

  final List<BoardEntry> entries;
  final String? me;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    Widget place(int i) {
      if (i >= entries.length) {
        return const Expanded(child: SizedBox());
      }
      final e = entries[i];
      final first = i == 0;
      final medal = _medals[i];
      return Expanded(
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: _openPlayer(context, e, me),
          child: Column(
            key: Key('board-row-${e.entityId}'),
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              if (first)
                Icon(Icons.workspace_premium_rounded, color: bv.gold, size: 30),
              Container(
                padding: const EdgeInsets.all(3),
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: LinearGradient(
                    colors: [medal, medal.withAlpha(140)],
                  ),
                  boxShadow: [
                    BoxShadow(color: medal.withAlpha(90), blurRadius: 14),
                  ],
                ),
                child: CircleAvatar(
                  radius: first ? 34 : 27,
                  backgroundColor: bv.surface,
                  child: Text(
                    _leading(e),
                    style: TextStyle(fontSize: first ? 32 : 26),
                  ),
                ),
              ),
              const SizedBox(height: 6),
              CosmeticName(
                e.name,
                nameColor: e.user?.nameColor ?? '',
                style: text.titleSmall?.copyWith(
                  color: e.entityId == me ? bv.accent : bv.ink,
                ),
              ),
              Text(
                l10n.boardPoints(e.score),
                style: text.labelMedium?.copyWith(color: bv.muted),
              ),
              const SizedBox(height: 6),
              Container(
                height: first ? 64 : (i == 1 ? 46 : 34),
                width: double.infinity,
                margin: const EdgeInsets.symmetric(horizontal: 6),
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(14),
                  ),
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [medal.withAlpha(150), medal.withAlpha(25)],
                  ),
                ),
                child: Text(
                  '${e.rank}',
                  style: text.headlineSmall?.copyWith(color: Colors.white),
                ),
              ),
            ],
          ),
        ),
      );
    }

    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [place(1), place(0), place(2)],
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
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      key: Key('board-row-${entry.entityId}'),
      onTap: mine ? null : _openPlayer(context, entry, null),
      radius: 18,
      color: mine ? bv.gradientStart.withAlpha(70) : null,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Row(
        children: [
          SizedBox(
            width: 40,
            child: Text(
              '#${entry.rank}',
              style: text.titleSmall?.copyWith(
                color: entry.rank <= 3 ? bv.gold : bv.muted,
              ),
            ),
          ),
          if (entry.user?.frame case final frame? when frame.isNotEmpty)
            AvatarBubble(_leading(entry), size: 40, frame: frame)
          else
            CircleAvatar(
              radius: 20,
              backgroundColor: bv.badge,
              child: Text(
                _leading(entry),
                style: const TextStyle(fontSize: 20),
              ),
            ),
          const SizedBox(width: 12),
          Expanded(
            child: CosmeticName(
              label != null ? '$label · ${entry.name}' : entry.name,
              nameColor: entry.user?.nameColor ?? '',
              style: text.titleSmall,
            ),
          ),
          Text(
            l10n.boardPoints(entry.score),
            style: text.labelLarge?.copyWith(color: bv.accent),
          ),
        ],
      ),
    );
  }
}
