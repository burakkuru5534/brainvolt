import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../content/domain/content_repository.dart';
import '../../match/presentation/format_picker.dart';
import '../../match/presentation/match_type_name.dart';
import '../../match/presentation/mode_picker_screen.dart';
import '../../match/presentation/play_error_text.dart';
import '../../match/presentation/topic_picker.dart';
import '../../profile/domain/avatars.dart';
import '../data/lobby_providers.dart';
import '../domain/lobby_repository.dart';
import 'room_controller.dart';

/// The Private Room sheet (D-085): a card per match type creates a room,
/// the code field joins one; either way the room opens.
Future<void> showRoomSheet(BuildContext context, WidgetRef ref) async {
  final l10n = context.l10n;
  final choice = await showModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    builder: (context) => const _RoomSheet(),
  );
  if (choice == null || !context.mounted) {
    return;
  }
  final repo = ref.read(lobbyRepositoryProvider);
  final messenger = ScaffoldMessenger.of(context);
  try {
    final Room room;
    if (choice.startsWith('code:')) {
      room = await repo.joinRoom(choice.substring(5));
    } else {
      final categories = await ref
          .read(categoriesProvider.future)
          .catchError((Object _) => <CategoryInfo>[]);
      if (!context.mounted) {
        return;
      }
      final settings = await pickRoomSettings(
        context,
        initial: RoomSettings(matchType: MatchType.parse(choice)!),
        title: context.l10n.createRoom,
        categories: categories,
      );
      if (settings == null) {
        return;
      }
      room = await repo.createRoom(
        settings.matchType,
        format: settings.format,
        teamMode: settings.teamMode,
        categoryMode: settings.categoryMode,
        categoryIds: settings.categoryIds,
      );
    }
    if (context.mounted) {
      unawaited(context.push(Routes.room(room.id)));
    }
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
}

/// Pops a match type's wire name, or `code:<code>` to join.
class _RoomSheet extends StatefulWidget {
  const _RoomSheet();

  @override
  State<_RoomSheet> createState() => _RoomSheetState();
}

class _RoomSheetState extends State<_RoomSheet> {
  final _code = TextEditingController();

  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  void _join() {
    final code = _code.text.trim();
    if (code.isNotEmpty) {
      Navigator.pop(context, 'code:$code');
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final text = Theme.of(context).textTheme;
    final bv = BvColors.of(context);
    final options = {
      MatchType.quickDuel: (Icons.add_home_rounded, l10n.roomQuickDuelSub),
      MatchType.suddenDeath: (Icons.bolt, l10n.roomSuddenDeathSub),
      MatchType.marathon: (Icons.emoji_events_outlined, l10n.roomMarathonSub),
    };
    return SafeArea(
      child: Padding(
        padding: EdgeInsets.fromLTRB(
          16,
          0,
          16,
          16 + MediaQuery.viewInsetsOf(context).bottom,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l10n.roomSheetTitle,
                textAlign: TextAlign.center,
                style: text.headlineMedium,
              ),
              Text(
                l10n.roomSheetSub,
                textAlign: TextAlign.center,
                style: text.bodyLarge?.copyWith(color: bv.muted),
              ),
              const SizedBox(height: 16),
              for (final t in MatchType.values) ...[
                ActionCard(
                  key: Key('create-room-${t.wire}'),
                  icon: options[t]?.$1 ?? Icons.add_home_rounded,
                  title: '${l10n.createRoom} — ${matchTypeName(l10n, t.wire)}',
                  subtitle: options[t]?.$2,
                  onTap: () => Navigator.pop(context, t.wire),
                ),
                const SizedBox(height: 12),
              ],
              const SizedBox(height: 8),
              Text(l10n.roomJoinWithCode, style: text.titleLarge),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      key: const Key('room-code-input'),
                      controller: _code,
                      textCapitalization: TextCapitalization.characters,
                      onSubmitted: (_) => _join(),
                      decoration: InputDecoration(
                        hintText: l10n.roomCodeHint,
                        prefixIcon: Icon(Icons.key_rounded, color: bv.badgeInk),
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  SizedBox(
                    width: 120,
                    child: GradientButton(
                      key: const Key('room-code-join'),
                      label: l10n.roomJoinAction,
                      height: 56,
                      onPressed: _join,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// A private room: the code to share, the two sides with their seats,
/// ready flags and, for the host, the start button.
class RoomScreen extends ConsumerStatefulWidget {
  const RoomScreen({required this.roomId, super.key});

  final String roomId;

  @override
  ConsumerState<RoomScreen> createState() => _RoomScreenState();
}

class _RoomScreenState extends ConsumerState<RoomScreen> {
  @override
  void initState() {
    super.initState();
    unawaited(
      Future.microtask(
        () => ref.read(currentRoomProvider.notifier).set(widget.roomId),
      ),
    );
  }

  RoomController get _ctrl =>
      ref.read(roomControllerProvider(widget.roomId).notifier);

  Future<void> _run(Future<void> Function() action) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      await action();
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  /// The host changes any room mode between matches (D-093).
  Future<void> _settings(Room room) async {
    final l10n = context.l10n;
    final categories = await ref
        .read(categoriesProvider.future)
        .catchError((Object _) => <CategoryInfo>[]);
    if (!mounted) {
      return;
    }
    final s = await pickRoomSettings(
      context,
      initial: RoomSettings.of(room),
      quizLocked: room.creatorQuizId != null,
      members: room.members.length,
      categories: categories,
    );
    if (s == null || !mounted) {
      return;
    }
    await _run(() async {
      await ref
          .read(lobbyRepositoryProvider)
          .updateRoom(
            room.id,
            s,
            creatorQuizId: room.creatorQuizId,
            version: room.version,
          );
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(l10n.roomSettingsSaved)));
      }
    });
  }

  Future<void> _leave() async {
    await ref
        .read(lobbyRepositoryProvider)
        .leaveRoom(widget.roomId)
        .catchError((Object _) {});
    ref.read(currentRoomProvider.notifier).set(null);
    if (mounted) {
      context.go(Routes.home);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    ref.listen(roomControllerProvider(widget.roomId), (prev, next) {
      if (next.startingMatchId case final id?
          when prev?.startingMatchId != id) {
        context.go(Routes.match(id));
      }
    });
    final view = ref.watch(roomControllerProvider(widget.roomId));
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final room = view.room;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          view.room?.isMatchmaking ?? false
              ? l10n.mmLobbyTitle
              : l10n.privateRoom,
        ),
        actions: [
          IconButton(
            key: const Key('room-leave'),
            tooltip: l10n.leaveRoom,
            icon: const Icon(Icons.logout),
            onPressed: _leave,
          ),
        ],
      ),
      body: switch (view) {
        RoomView(closedReason: final reason?) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const IconBadge(Icons.meeting_room_outlined, size: 88),
              const SizedBox(height: 16),
              Text(
                switch (reason) {
                  'not_ready' => l10n.mmLobbyDropped,
                  'lobby_expired' => l10n.mmLobbyExpired,
                  _ => l10n.roomClosed,
                },
                key: const Key('room-closed'),
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () => context.go(Routes.home),
                child: Text(l10n.backHome),
              ),
            ],
          ),
        ),
        RoomView(room: null) => const LoadingView(),
        _ when room!.isMatchmaking => _lobbyBody(context, room, me),
        _ => _body(context, room, me),
      },
    );
  }

  Widget _body(BuildContext context, Room room, String me) {
    final l10n = context.l10n;
    final mine = room.members.where((m) => m.card.userId == me).firstOrNull;
    final host = room.hostUserId == me;
    final othersReady = room.members.every((m) => m.ready || m.isHost);
    final perSide = room.capacity ~/ 2;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        BvCard(
          key: const Key('room-code'),
          gradient: BvColors.of(context).heroGradient,
          padding: const EdgeInsets.fromLTRB(16, 14, 8, 14),
          child: Row(
            children: [
              const IconBadge(
                Icons.vpn_key_rounded,
                size: 48,
                onGradient: true,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.roomCode(room.code),
                      style: Theme.of(context).textTheme.titleLarge
                          ?.copyWith(color: Colors.white, letterSpacing: 1.2),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      [
                        matchTypeName(l10n, room.matchType),
                        formatName(l10n, room.format),
                        if (room.teamMode case final m?) teamModeName(l10n, m),
                        if (room.quizLength case final n?
                            when room.creatorQuizId != null)
                          l10n.roomQuiz(n)
                        else
                          l10n.topicLine(
                            topicName(
                              l10n,
                              RoomSettings.of(room).topic,
                              ref.watch(categoriesProvider).value ?? const [],
                            ),
                          ),
                      ].join(' · '),
                      style: Theme.of(context).textTheme.bodySmall
                          ?.copyWith(color: Colors.white.withAlpha(220)),
                    ),
                  ],
                ),
              ),
              IconButton(
                tooltip: l10n.copied,
                icon: const Icon(Icons.copy_rounded, color: Colors.white),
                onPressed: () async {
                  await Clipboard.setData(ClipboardData(text: room.code));
                  if (context.mounted) {
                    ScaffoldMessenger.of(context)
                        .showSnackBar(SnackBar(content: Text(l10n.copied)));
                  }
                },
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        if (host)
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton.icon(
              key: const Key('room-settings'),
              icon: const Icon(Icons.tune),
              onPressed: room.status == 'open' ? () => _settings(room) : null,
              label: Text(l10n.roomSettingsTitle),
            ),
          ),
        if (room.friendsOnly)
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: BvCard(
              padding: EdgeInsets.zero,
              child: ListTile(
                key: const Key('room-friends-only'),
                leading: const Icon(Icons.lock_outline),
                title: Text(l10n.roomFriendsOnly),
                subtitle: Text(l10n.quizUnpublishedNote),
              ),
            ),
          ),
        for (var side = 1; side <= 2; side++) ...[
          _SideTitle(side: side, label: l10n.sideLabel(side)),
          for (var seat = 1; seat <= perSide; seat++)
            _seat(context, room, side, seat, mine),
        ],
        const SizedBox(height: 24),
        if (mine != null && !host)
          OutlinedButton.icon(
            key: const Key('room-ready'),
            icon: Icon(
              mine.ready ? Icons.close_rounded : Icons.check_circle_rounded,
            ),
            onPressed: room.status == 'open'
                ? () => _run(() => _ctrl.setReady(ready: !mine.ready))
                : null,
            label: Text(mine.ready ? l10n.notReady : l10n.readyLabel),
          ),
        if (host) ...[
          const SizedBox(height: 8),
          FilledButton(
            key: const Key('room-start'),
            onPressed: room.status == 'open' && room.full && othersReady
                ? () => _run(_ctrl.start)
                : null,
            child: Text(room.full ? l10n.startMatch : l10n.roomWaitingPlayers),
          ),
        ],
        if (room.matchId case final id?) ...[
          const SizedBox(height: 12),
          OutlinedButton(
            onPressed: () => context.go(Routes.match(id)),
            child: Text(l10n.resumeMatch),
          ),
        ],
      ],
    );
  }

  /// The waiting lobby of a matchmade team match (D-094): seats fill as
  /// players are found; once full a random leader is drawn; everyone
  /// readies up and the leader starts.
  Widget _lobbyBody(BuildContext context, Room room, String me) {
    final l10n = context.l10n;
    final text = Theme.of(context).textTheme;
    final mine = room.members.where((m) => m.card.userId == me).firstOrNull;
    final leader = room.leaderUserId;
    final allReady = room.members.every((m) => m.ready);
    final perSide = room.capacity ~/ 2;
    final leaderName = room.members
        .where((m) => m.card.userId == leader)
        .firstOrNull
        ?.card
        .displayName;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        BvCard(
          key: const Key('mm-lobby-status'),
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                [
                  matchTypeName(l10n, room.matchType),
                  formatName(l10n, room.format),
                  if (room.teamMode case final m?) teamModeName(l10n, m),
                  l10n.topicLine(
                    topicName(
                      l10n,
                      RoomSettings.of(room).topic,
                      ref.watch(categoriesProvider).value ?? const [],
                    ),
                  ),
                ].join(' · '),
                style: text.titleMedium,
              ),
              const SizedBox(height: 6),
              if (!room.full)
                Row(
                  children: [
                    const SizedBox.square(
                      dimension: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        l10n.mmLobbyWaiting(room.members.length, room.capacity),
                        key: const Key('mm-lobby-waiting'),
                      ),
                    ),
                  ],
                )
              else if (leader == me)
                Text(l10n.mmLobbyYouLead, key: const Key('mm-lobby-you-lead'))
              else
                Text(
                  l10n.mmLobbyLeaderIs(leaderName ?? ''),
                  key: const Key('mm-lobby-leader'),
                ),
              if (!room.full) ...[
                const SizedBox(height: 4),
                Text(
                  l10n.mmLobbyLeaderSoon,
                  style: text.bodySmall?.copyWith(
                    color: BvColors.of(context).muted,
                  ),
                ),
              ],
            ],
          ),
        ),
        for (var side = 1; side <= 2; side++) ...[
          _SideTitle(side: side, label: l10n.sideLabel(side)),
          for (var seat = 1; seat <= perSide; seat++)
            _lobbySeat(context, room, side, seat),
        ],
        const SizedBox(height: 24),
        if (mine != null)
          OutlinedButton.icon(
            key: const Key('room-ready'),
            icon: Icon(
              mine.ready ? Icons.close_rounded : Icons.check_circle_rounded,
            ),
            onPressed: room.status == 'open'
                ? () => _run(() => _ctrl.setReady(ready: !mine.ready))
                : null,
            label: Text(mine.ready ? l10n.notReady : l10n.readyLabel),
          ),
        if (leader == me) ...[
          const SizedBox(height: 8),
          FilledButton(
            key: const Key('room-start'),
            onPressed: room.status == 'open' && room.full && allReady
                ? () => _run(_ctrl.start)
                : null,
            child: Text(allReady ? l10n.startMatch : l10n.mmLobbyWaitReady),
          ),
        ],
      ],
    );
  }

  Widget _lobbySeat(BuildContext context, Room room, int side, int seat) {
    final l10n = context.l10n;
    final m = room.members
        .where((m) => m.side == side && m.seat == seat)
        .firstOrNull;
    if (m == null) {
      return _SeatCard(
        child: ListTile(
          key: Key('seat-$side-$seat'),
          leading: const Icon(Icons.hourglass_empty_rounded),
          title: Text(
            l10n.mmLobbySearchingSeat,
            style: TextStyle(color: BvColors.of(context).muted),
          ),
        ),
      );
    }
    final isLeader = m.card.userId == room.leaderUserId;
    return _SeatCard(
      ready: m.ready,
      child: ListTile(
        key: Key('seat-$side-$seat'),
        leading: AvatarBubble(avatarFor(m.card.avatarId), size: 42),
        title: Text(isLeader ? '👑 ${m.card.displayName}' : m.card.displayName),
        subtitle: Text(m.ready ? l10n.readyLabel : l10n.notReady),
        trailing: m.ready
            ? Icon(
                Icons.check_circle_rounded,
                color: BvColors.of(context).success,
              )
            : null,
      ),
    );
  }

  Widget _seat(
    BuildContext context,
    Room room,
    int side,
    int seat,
    RoomMember? mine,
  ) {
    final l10n = context.l10n;
    final m = room.members
        .where((m) => m.side == side && m.seat == seat)
        .firstOrNull;
    if (m == null) {
      return _SeatCard(
        child: ListTile(
          key: Key('seat-$side-$seat'),
          leading: const Icon(Icons.event_seat_outlined),
          title: Text(
            l10n.emptySeat,
            style: TextStyle(color: BvColors.of(context).muted),
          ),
          onTap: mine != null && room.status == 'open'
              ? () => _run(() => _ctrl.moveTo(side, seat))
              : null,
        ),
      );
    }
    return _SeatCard(
      ready: m.ready || m.isHost,
      child: ListTile(
        key: Key('seat-$side-$seat'),
        leading: AvatarBubble(avatarFor(m.card.avatarId), size: 42),
        title: Text(m.card.displayName),
        subtitle: Text(
          m.isHost ? l10n.host : (m.ready ? l10n.readyLabel : l10n.notReady),
        ),
        trailing: m.ready || m.isHost
            ? Icon(
                Icons.check_circle_rounded,
                color: BvColors.of(context).success,
              )
            : null,
      ),
    );
  }
}

/// A side's heading: purple for side 1, orange for side 2.
class _SideTitle extends StatelessWidget {
  const _SideTitle({required this.side, required this.label});

  final int side;
  final String label;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final tint = side == 1 ? bv.gradientStart : bv.gradientEnd;
    return Padding(
      padding: const EdgeInsets.only(top: 18, bottom: 8),
      child: Row(
        children: [
          Container(
            width: 6,
            height: 20,
            decoration: BoxDecoration(
              color: tint,
              borderRadius: BorderRadius.circular(3),
            ),
          ),
          const SizedBox(width: 8),
          Text(label, style: Theme.of(context).textTheme.titleMedium),
        ],
      ),
    );
  }
}

/// A seat row in a glass card; a ready seat gets a green edge.
class _SeatCard extends StatelessWidget {
  const _SeatCard({required this.child, this.ready = false});

  final Widget child;
  final bool ready;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: BvCard(
        radius: 18,
        padding: EdgeInsets.zero,
        color: ready ? bv.success.withAlpha(28) : null,
        child: child,
      ),
    );
  }
}
