import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../match/presentation/format_picker.dart';
import '../../match/presentation/match_type_name.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../data/lobby_providers.dart';
import '../domain/lobby_repository.dart';
import 'room_controller.dart';

/// Creates a room or joins one by code, then opens it.
Future<void> showRoomSheet(BuildContext context, WidgetRef ref) async {
  final l10n = context.l10n;
  final choice = await showModalBottomSheet<String>(
    context: context,
    builder: (context) => SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final t in MatchType.values)
            ListTile(
              key: Key('create-room-${t.wire}'),
              leading: const Icon(Icons.add_home),
              title: Text(
                '${l10n.createRoom} · ${matchTypeName(l10n, t.wire)}',
              ),
              onTap: () => Navigator.pop(context, t.wire),
            ),
          ListTile(
            key: const Key('join-room'),
            leading: const Icon(Icons.login),
            title: Text(l10n.joinRoom),
            onTap: () => Navigator.pop(context, 'join'),
          ),
        ],
      ),
    ),
  );
  if (choice == null || !context.mounted) {
    return;
  }
  final repo = ref.read(lobbyRepositoryProvider);
  final messenger = ScaffoldMessenger.of(context);
  try {
    final Room room;
    if (choice == 'join') {
      final code = await _askCode(context);
      if (code == null) {
        return;
      }
      room = await repo.joinRoom(code);
    } else {
      final format = await pickFormat(context);
      if (format == null) {
        return;
      }
      room = await repo.createRoom(
        MatchType.parse(choice)!,
        format: format.$1,
        teamMode: format.$2,
      );
    }
    if (context.mounted) {
      unawaited(context.push(Routes.room(room.id)));
    }
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
}

Future<String?> _askCode(BuildContext context) =>
    showDialog<String>(context: context, builder: (_) => const _CodeDialog());

/// Asks for a room code; owns its text controller so it outlives the
/// closing animation.
class _CodeDialog extends StatefulWidget {
  const _CodeDialog();

  @override
  State<_CodeDialog> createState() => _CodeDialogState();
}

class _CodeDialogState extends State<_CodeDialog> {
  final _text = TextEditingController();

  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AlertDialog(
      title: Text(l10n.joinRoom),
      content: TextField(
        key: const Key('room-code-input'),
        controller: _text,
        autofocus: true,
        textCapitalization: TextCapitalization.characters,
        decoration: InputDecoration(labelText: l10n.roomCodeLabel),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('room-code-join'),
          onPressed: () => Navigator.pop(context, _text.text.trim()),
          child: Text(l10n.joinRoom),
        ),
      ],
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
        title: Text(l10n.privateRoom),
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
        RoomView(closedReason: final _?) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(l10n.roomClosed, key: const Key('room-closed')),
              TextButton(
                onPressed: () => context.go(Routes.home),
                child: Text(l10n.backHome),
              ),
            ],
          ),
        ),
        RoomView(room: null) => const Center(
          child: CircularProgressIndicator(),
        ),
        _ => _body(context, room!, me),
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
        ListTile(
          key: const Key('room-code'),
          leading: const Icon(Icons.vpn_key),
          title: Text(l10n.roomCode(room.code)),
          subtitle: Text(
            [
              matchTypeName(l10n, room.matchType),
              formatName(l10n, room.format),
              if (room.teamMode case final m?) teamModeName(l10n, m),
            ].join(' · '),
          ),
          trailing: IconButton(
            icon: const Icon(Icons.copy),
            onPressed: () async {
              await Clipboard.setData(ClipboardData(text: room.code));
              if (context.mounted) {
                ScaffoldMessenger.of(context)
                    .showSnackBar(SnackBar(content: Text(l10n.copied)));
              }
            },
          ),
        ),
        for (var side = 1; side <= 2; side++) ...[
          Padding(
            padding: const EdgeInsets.only(top: 16, bottom: 4),
            child: Text(
              l10n.sideLabel(side),
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),
          for (var seat = 1; seat <= perSide; seat++)
            _seat(context, room, side, seat, mine),
        ],
        const SizedBox(height: 24),
        if (mine != null && !host)
          FilledButton.tonal(
            key: const Key('room-ready'),
            onPressed: room.status == 'open'
                ? () => _run(() => _ctrl.setReady(ready: !mine.ready))
                : null,
            child: Text(mine.ready ? l10n.notReady : l10n.readyLabel),
          ),
        if (host)
          FilledButton(
            key: const Key('room-start'),
            onPressed: room.status == 'open' && room.full && othersReady
                ? () => _run(_ctrl.start)
                : null,
            child: Text(room.full ? l10n.startMatch : l10n.roomWaitingPlayers),
          ),
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
      return ListTile(
        key: Key('seat-$side-$seat'),
        leading: const Icon(Icons.event_seat_outlined),
        title: Text(l10n.emptySeat),
        onTap: mine != null && room.status == 'open'
            ? () => _run(() => _ctrl.moveTo(side, seat))
            : null,
      );
    }
    return ListTile(
      key: Key('seat-$side-$seat'),
      leading: Text(
        avatarFor(m.card.avatarId),
        style: const TextStyle(fontSize: 28),
      ),
      title: Text(m.card.displayName),
      subtitle: Text(
        m.isHost ? l10n.host : (m.ready ? l10n.readyLabel : l10n.notReady),
      ),
      trailing: m.ready || m.isHost
          ? const Icon(Icons.check_circle, color: Colors.green)
          : null,
    );
  }
}
