import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/match_type_name.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../../team/data/team_providers.dart';
import '../../team/domain/team_repository.dart';
import '../data/tournament_providers.dart';
import '../domain/tournament_repository.dart';

/// GET /v1/tournaments.
final tournamentsProvider = FutureProvider.autoDispose<List<Tournament>>(
  (ref) => ref.watch(tournamentRepositoryProvider).list(),
);

/// GET /v1/tournaments/{id}.
final tournamentProvider = FutureProvider.autoDispose
    .family<TournamentDetail, String>(
      (ref, id) => ref.watch(tournamentRepositoryProvider).get(id),
    );

String tournamentStatusText(AppLocalizations l, String s) => switch (s) {
  'registration' => l.tournamentStatusRegistration,
  'running' => l.tournamentStatusRunning,
  'finished' => l.tournamentStatusFinished,
  'cancelled' => l.tournamentStatusCancelled,
  _ => l.tournamentStatusDraft,
};

String entryStatusText(AppLocalizations l, TournamentEntry e) =>
    switch (e.status) {
      'checked_in' => l.tournamentCheckedIn,
      'active' => l.tournamentPlaying,
      'eliminated' => l.tournamentPlace(e.finalPlace ?? 0),
      'winner' => l.tournamentChampion,
      _ => l.tournamentRegistered,
    };

String _kind(AppLocalizations l, Tournament t) => switch (t.entityKind) {
  'couple' => l.tournamentKindCouple,
  'team' => l.tournamentKindTeam(t.format),
  _ => l.tournamentKindSolo,
};

String _when(BuildContext context, DateTime d) {
  final m = MaterialLocalizations.of(context);
  return '${m.formatShortDate(d)} ${m.formatTimeOfDay(TimeOfDay.fromDateTime(d))}';
}

bool _isAdmin(WidgetRef ref) =>
    switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.roles.contains('admin'),
      _ => false,
    };

/// Tournaments: open and running first, then the finished ones; admins
/// create new ones (D-090).
class TournamentsScreen extends ConsumerWidget {
  const TournamentsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.tournamentsTitle)),
      floatingActionButton: _isAdmin(ref)
          ? FloatingActionButton.extended(
              key: const Key('create-tournament'),
              icon: const Icon(Icons.add),
              label: Text(l10n.tournamentCreate),
              onPressed: () => context.push(Routes.tournamentNew),
            )
          : null,
      body: switch (ref.watch(tournamentsProvider)) {
        AsyncData(:final value) when value.isEmpty => EmptyState(
          icon: Icons.emoji_events_outlined,
          text: l10n.tournamentsEmpty,
        ),
        AsyncData(:final value) => RefreshIndicator(
          onRefresh: () => ref.refresh(tournamentsProvider.future),
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              for (final t in [
                ...value.where((t) => t.status == 'running'),
                ...value.where((t) => t.status == 'registration'),
                ...value.where(
                  (t) => !{'running', 'registration'}.contains(t.status),
                ),
              ])
                Padding(
                  padding: const EdgeInsets.only(bottom: 10),
                  child: _Card(t: t),
                ),
            ],
          ),
        ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.t});

  final Tournament t;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final c = BvColors.of(context);
    return BvCard(
      key: Key('tournament-${t.id}'),
      onTap: () => context.push(Routes.tournament(t.id)),
      child: Row(
        children: [
          const IconBadge(Icons.emoji_events),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(t.name, style: Theme.of(context).textTheme.titleMedium),
                Text(
                  '${_kind(l10n, t)} · ${matchTypeName(l10n, t.matchType)} · '
                  '${l10n.tournamentEntrants(t.entrants, t.maxEntrants)}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                Text(
                  _when(context, t.startsAt),
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                tournamentStatusText(l10n, t.status),
                style: TextStyle(
                  color: c.gradientStart,
                  fontWeight: FontWeight.w700,
                ),
              ),
              if (t.myEntry case final e?)
                Text(
                  entryStatusText(l10n, e),
                  style: Theme.of(context).textTheme.bodySmall,
                ),
            ],
          ),
        ],
      ),
    );
  }
}

/// A tournament: details, what the caller can do now (register, check in,
/// play), the bracket and the entries. Follows tournament.updated and
/// goes to the match on tournament.match_ready.
class TournamentScreen extends ConsumerStatefulWidget {
  const TournamentScreen({required this.id, super.key});

  final String id;

  @override
  ConsumerState<TournamentScreen> createState() => _TournamentScreenState();
}

class _TournamentScreenState extends ConsumerState<TournamentScreen> {
  StreamSubscription<Object>? _sub;
  Timer? _tick;

  @override
  void initState() {
    super.initState();
    final rt = ref.read(realtimeClientProvider)..start();
    _sub = rt.frames.listen((f) {
      if (f.data['tournament_id'] != widget.id) {
        return;
      }
      if (f.type == 'tournament.updated') {
        ref.invalidate(tournamentProvider(widget.id));
      } else if (f.type == 'tournament.match_ready' && mounted) {
        if (f.data['match_id'] case final String m) {
          context.go(Routes.match(m));
        }
      }
    });
    // The ready window's countdown.
    _tick = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted) {
        setState(() {});
      }
    });
  }

  @override
  void dispose() {
    unawaited(_sub?.cancel());
    _tick?.cancel();
    super.dispose();
  }

  Future<void> _run(Future<void> Function(TournamentRepository) action) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      await action(ref.read(tournamentRepositoryProvider));
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
    ref
      ..invalidate(tournamentProvider(widget.id))
      ..invalidate(tournamentsProvider);
  }

  Future<void> _register(Tournament t) async {
    if (t.entityKind != 'team') {
      return _run((r) => r.register(t.id));
    }
    final pick = await showModalBottomSheet<(String, List<String>)>(
      context: context,
      isScrollControlled: true,
      builder: (_) => _LineupSheet(size: int.parse(t.format[0])),
    );
    if (pick == null) {
      return;
    }
    await _run((r) => r.register(t.id, teamId: pick.$1, lineup: pick.$2));
  }

  Future<void> _play() async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final match = await ref
          .read(tournamentRepositoryProvider)
          .play(widget.id);
      if (match != null && mounted) {
        context.go(Routes.match(match));
        return;
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
    ref.invalidate(tournamentProvider(widget.id));
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final detail = ref.watch(tournamentProvider(widget.id));
    return Scaffold(
      appBar: AppBar(title: Text(detail.value?.tournament.name ?? '')),
      body: switch (detail) {
        AsyncData(:final value) => RefreshIndicator(
          onRefresh: () => ref.refresh(tournamentProvider(widget.id).future),
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              _Header(t: value.tournament),
              const SizedBox(height: 12),
              _actions(context, value, me),
              if (value.bracket.isNotEmpty) ...[
                SectionTitle(l10n.tournamentBracket),
                _Bracket(d: value),
              ],
              SectionTitle(
                l10n.tournamentEntrants(
                  value.tournament.entrants,
                  value.tournament.maxEntrants,
                ),
              ),
              for (final e in value.entries.where(
                (e) => e.status != 'withdrawn',
              ))
                ListTile(
                  key: Key('entry-${e.id}'),
                  leading: Text(
                    e.entityKind == 'user'
                        ? avatarFor(e.image)
                        : emblemEmoji[e.image] ?? '🏆',
                    style: const TextStyle(fontSize: 26),
                  ),
                  title: Text(e.name),
                  subtitle: e.seed == null ? null : Text('#${e.seed}'),
                  trailing: Text(entryStatusText(l10n, e)),
                ),
            ],
          ),
        ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }

  /// What the caller can do now.
  Widget _actions(BuildContext context, TournamentDetail d, String me) {
    final l10n = context.l10n;
    final t = d.tournament;
    final mine = t.myEntry;
    final now = DateTime.now();
    final children = <Widget>[];
    switch (t.status) {
      case 'registration':
        if (mine == null) {
          final open =
              !now.isBefore(t.registrationOpensAt) &&
              now.isBefore(t.registrationClosesAt);
          children.add(
            GradientButton(
              key: const Key('tournament-register'),
              label: l10n.tournamentRegister,
              icon: Icons.how_to_reg,
              onPressed: open && t.entrants < t.maxEntrants
                  ? () => _register(t)
                  : null,
            ),
          );
        } else {
          final checkIn = !now.isBefore(t.checkInOpensAt);
          children.addAll([
            Text(
              mine.status == 'checked_in'
                  ? l10n.tournamentCheckedInInfo
                  : checkIn
                  ? l10n.tournamentCheckInNow
                  : l10n.tournamentCheckInAt(_when(context, t.checkInOpensAt)),
              key: const Key('tournament-entry-info'),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            if (mine.status != 'checked_in')
              GradientButton(
                key: const Key('tournament-check-in'),
                label: l10n.tournamentCheckIn,
                icon: Icons.check_circle,
                onPressed: checkIn ? () => _run((r) => r.checkIn(t.id)) : null,
              ),
            TextButton(
              key: const Key('tournament-withdraw'),
              onPressed: () => _run((r) => r.withdraw(t.id)),
              child: Text(l10n.tournamentWithdraw),
            ),
          ]);
        }
      case 'running':
        if (d.myMatch case final m?) {
          final opp = d.entry(m.entryA == mine?.id ? m.entryB : m.entryA);
          final left = m.readyUntil?.difference(now).inSeconds ?? 0;
          final here = m.present.contains(me);
          children.addAll([
            Text(
              l10n.tournamentYourMatch(m.round, opp?.name ?? ''),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            if (m.status == 'ready')
              Text(
                here
                    ? l10n.tournamentWaitingOpponent
                    : l10n.tournamentReadyLeft(
                        '${(left ~/ 60).clamp(0, 99)}:${(left % 60).clamp(0, 59).toString().padLeft(2, '0')}',
                      ),
                key: const Key('tournament-ready-left'),
                textAlign: TextAlign.center,
              ),
            const SizedBox(height: 8),
            if (m.status == 'in_progress' && m.matchId != null)
              GradientButton(
                key: const Key('tournament-go-match'),
                label: l10n.tournamentGoToMatch,
                icon: Icons.play_arrow_rounded,
                onPressed: () => context.go(Routes.match(m.matchId!)),
              )
            else
              GradientButton(
                key: const Key('tournament-play'),
                label: l10n.tournamentPlay,
                icon: Icons.play_arrow_rounded,
                onPressed: here || left <= 0 ? null : _play,
              ),
          ]);
        } else if (mine != null) {
          children.add(
            Text(
              mine.status == 'active'
                  ? l10n.tournamentWaitingRound
                  : entryStatusText(l10n, mine),
              textAlign: TextAlign.center,
            ),
          );
        }
      case 'finished':
        if (mine != null) {
          children.add(
            Text(
              entryStatusText(l10n, mine),
              key: const Key('tournament-my-place'),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleLarge,
            ),
          );
        }
    }
    if (children.isEmpty) {
      return const SizedBox.shrink();
    }
    return BvCard(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: children,
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.t});

  final Tournament t;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    const white = TextStyle(color: Colors.white);
    return BvCard(
      gradient: BvColors.of(context).gradient,
      padding: const EdgeInsets.all(18),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.emoji_events, color: Color(0xFFFFD45C)),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  tournamentStatusText(l10n, t.status),
                  key: const Key('tournament-status'),
                  style: white.copyWith(fontWeight: FontWeight.w800),
                ),
              ),
              Text(
                l10n.tournamentEntrants(t.entrants, t.maxEntrants),
                style: white,
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            t.name,
            style: Theme.of(context).textTheme.headlineSmall
                ?.copyWith(color: Colors.white),
          ),
          if (t.description case final d? when d.isNotEmpty)
            Text(d, style: white),
          const SizedBox(height: 8),
          Text(
            '${_kind(l10n, t)} · ${matchTypeName(l10n, t.matchType)}',
            style: white,
          ),
          Text(l10n.tournamentStarts(_when(context, t.startsAt)), style: white),
          if (t.rewards.isNotEmpty)
            Text(
              [
                for (final (place, xp) in t.rewards)
                  l10n.tournamentRewardLine(place, xp),
              ].join(' · '),
              key: const Key('tournament-rewards'),
              style: white,
            ),
        ],
      ),
    );
  }
}

/// The bracket, one column per round.
class _Bracket extends StatelessWidget {
  const _Bracket({required this.d});

  final TournamentDetail d;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final c = BvColors.of(context);
    String roundName(int r) => switch (d.rounds - r) {
      0 => l10n.tournamentFinal,
      1 => l10n.tournamentSemiFinal,
      _ => l10n.tournamentRound(r),
    };
    Widget line(String? id, BracketSlot s) {
      final e = d.entry(id);
      final won = id != null && s.winner == id;
      return Text(
        e?.name ?? (s.round == 1 ? l10n.tournamentBye : '—'),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          fontWeight: won ? FontWeight.w800 : FontWeight.w500,
          color: s.winner != null && !won ? c.muted : null,
        ),
      );
    }

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (var r = 1; r <= d.rounds; r++)
            SizedBox(
              width: 170,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Padding(
                    padding: const EdgeInsets.all(4),
                    child: Text(
                      roundName(r),
                      style: Theme.of(context).textTheme.labelLarge,
                    ),
                  ),
                  for (final s in d.bracket.where((s) => s.round == r))
                    Padding(
                      padding: const EdgeInsets.all(4),
                      child: BvCard(
                        key: Key('slot-${s.round}-${s.slot}'),
                        padding: const EdgeInsets.all(10),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            line(s.entryA, s),
                            const Divider(height: 10),
                            line(s.entryB, s),
                            if (s.status == 'walkover' && s.entryB != null)
                              Text(
                                l10n.tournamentWalkover,
                                style: Theme.of(context).textTheme.bodySmall,
                              ),
                          ],
                        ),
                      ),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

/// A team tournament: the captain picks the team and its lineup.
class _LineupSheet extends ConsumerStatefulWidget {
  const _LineupSheet({required this.size});

  final int size;

  @override
  ConsumerState<_LineupSheet> createState() => _LineupSheetState();
}

class _LineupSheetState extends ConsumerState<_LineupSheet> {
  Team? _team;
  final _picked = <String>{};

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final teams = ref.watch(myTeamsForTournamentProvider).value ?? const [];
    final mine = teams.where((t) => t.captainUserId == me).toList();
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              l10n.tournamentPickLineup(widget.size),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            if (mine.isEmpty) Text(l10n.tournamentNoCaptainTeam),
            for (final t in mine)
              ListTile(
                key: Key('lineup-team-${t.id}'),
                selected: _team?.id == t.id,
                leading: Icon(
                  _team?.id == t.id
                      ? Icons.radio_button_checked
                      : Icons.radio_button_off,
                ),
                title: Text(t.name),
                onTap: () => setState(() {
                  _team = t;
                  _picked.clear();
                }),
              ),
            if (_team case final t?)
              for (final m in t.members)
                CheckboxListTile(
                  key: Key('lineup-${m.card.userId}'),
                  value: _picked.contains(m.card.userId),
                  title: Text(m.card.displayName),
                  onChanged: (v) => setState(
                    () => v ?? false
                        ? _picked.add(m.card.userId)
                        : _picked.remove(m.card.userId),
                  ),
                ),
            const SizedBox(height: 8),
            FilledButton(
              key: const Key('lineup-done'),
              onPressed: _team != null && _picked.length == widget.size
                  ? () => Navigator.pop(context, (_team!.id, _picked.toList()))
                  : null,
              child: Text(l10n.tournamentRegister),
            ),
          ],
        ),
      ),
    );
  }
}

/// The caller's teams for the lineup sheet.
final myTeamsForTournamentProvider = FutureProvider.autoDispose<List<Team>>(
  (ref) => ref.watch(teamRepositoryProvider).myTeams(),
);

/// Admin: a quick tournament form (D-090).
class CreateTournamentScreen extends ConsumerStatefulWidget {
  const CreateTournamentScreen({super.key});

  @override
  ConsumerState<CreateTournamentScreen> createState() =>
      _CreateTournamentScreenState();
}

class _CreateTournamentScreenState
    extends ConsumerState<CreateTournamentScreen> {
  final _name = TextEditingController();
  final _nameEn = TextEditingController();
  var _kind = 'user';
  var _teamFormat = '2v2';
  var _type = 'quick_duel';
  var _max = 8;
  var _startIn = 60;
  var _window = 5;
  var _busy = false;

  @override
  void dispose() {
    _name.dispose();
    _nameEn.dispose();
    super.dispose();
  }

  Future<void> _create() async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    setState(() => _busy = true);
    try {
      final id = await ref
          .read(tournamentRepositoryProvider)
          .create(
            TournamentDraft(
              nameTr: _name.text.trim(),
              nameEn: _nameEn.text.trim(),
              entityKind: _kind,
              format: switch (_kind) {
                'couple' => 'couple',
                'team' => _teamFormat,
                _ => '1v1',
              },
              matchType: _type,
              maxEntrants: _max,
              startsAt: DateTime.now().add(Duration(minutes: _startIn)),
              readyWindowS: _window * 60,
              rewards: const [300, 150, 75],
            ),
          );
      ref.invalidate(tournamentsProvider);
      if (mounted) {
        context.pushReplacement(Routes.tournament(id));
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    Widget choice<T>(
      String key,
      String label,
      T value,
      Map<T, String> options,
      void Function(T) set,
    ) => Padding(
      padding: const EdgeInsets.only(top: 12),
      child: DropdownButtonFormField<T>(
        key: Key(key),
        initialValue: value,
        decoration: InputDecoration(labelText: label),
        items: [
          for (final MapEntry(:key, value: text) in options.entries)
            DropdownMenuItem(value: key, child: Text(text)),
        ],
        onChanged: (v) => v == null ? null : setState(() => set(v)),
      ),
    );
    return Scaffold(
      appBar: AppBar(title: Text(l10n.tournamentCreate)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            key: const Key('tournament-name'),
            controller: _name,
            maxLength: 60,
            decoration: InputDecoration(labelText: l10n.tournamentNameTr),
            onChanged: (_) => setState(() {}),
          ),
          TextField(
            key: const Key('tournament-name-en'),
            controller: _nameEn,
            maxLength: 60,
            decoration: InputDecoration(labelText: l10n.tournamentNameEn),
          ),
          choice('tournament-kind', l10n.tournamentKind, _kind, {
            'user': l10n.tournamentKindSolo,
            'couple': l10n.tournamentKindCouple,
            'team': l10n.tournamentKindTeam(''),
          }, (v) => _kind = v),
          if (_kind == 'team')
            choice('tournament-format', l10n.tournamentFormat, _teamFormat, {
              for (final f in ['2v2', '3v3', '4v4']) f: f,
            }, (v) => _teamFormat = v),
          choice('tournament-type', l10n.tournamentMatchType, _type, {
            for (final t in ['quick_duel', 'sudden_death', 'marathon'])
              t: matchTypeName(l10n, t),
          }, (v) => _type = v),
          choice('tournament-max', l10n.tournamentMax, _max, {
            for (final n in [4, 8, 16, 32]) n: '$n',
          }, (v) => _max = v),
          choice('tournament-start', l10n.tournamentStartIn, _startIn, {
            for (final m in [15, 30, 60, 120, 1440])
              m: l10n.tournamentMinutes(m),
          }, (v) => _startIn = v),
          choice('tournament-window', l10n.tournamentWindow, _window, {
            for (final m in [3, 5, 10]) m: l10n.tournamentMinutes(m),
          }, (v) => _window = v),
          const SizedBox(height: 8),
          Text(l10n.tournamentCreateHint),
          const SizedBox(height: 16),
          GradientButton(
            key: const Key('tournament-submit'),
            label: l10n.tournamentCreate,
            icon: Icons.emoji_events,
            onPressed: _busy || _name.text.trim().length < 3 ? null : _create,
          ),
        ],
      ),
    );
  }
}
