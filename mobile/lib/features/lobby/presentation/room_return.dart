import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/result_screen.dart';
import 'room_controller.dart';

/// How long the end screen of a room match shows before the players are
/// taken back to the room (D-092).
const roomReturnDelay = Duration(seconds: 5);

/// The end of a private room match (D-092): the score stays on screen for
/// [roomReturnDelay], then everyone goes back to the room lobby, where
/// they ready up again and the host starts the next match. "Back to the
/// room" goes at once. A player who opened the details is not pulled
/// away; the result screen has its own "Back to the room" button.
class RoomReturn extends ConsumerStatefulWidget {
  const RoomReturn({required this.matchId, super.key});

  final String matchId;

  @override
  ConsumerState<RoomReturn> createState() => _RoomReturnState();
}

class _RoomReturnState extends ConsumerState<RoomReturn> {
  Timer? _timer;
  late int _left = roomReturnDelay.inSeconds;

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  void _start(String room) {
    _timer ??= Timer.periodic(const Duration(seconds: 1), (t) {
      if (!mounted) {
        t.cancel();
        return;
      }
      setState(() => _left--);
      if (_left > 0) {
        return;
      }
      t.cancel();
      if (ModalRoute.of(context)?.isCurrent ?? true) {
        _go(room);
      }
    });
  }

  void _go(String room) {
    _timer?.cancel();
    context.go(Routes.room(room));
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final room = ref.watch(currentRoomProvider);
    final result = ref.watch(matchResultProvider(widget.matchId)).value;
    if (room == null || result == null || result.origin != 'private_room') {
      return const SizedBox.shrink();
    }
    _start(room);
    return Padding(
      key: const Key('room-return'),
      padding: const EdgeInsets.only(top: 12),
      child: BvCard(
        padding: const EdgeInsets.all(14),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              l10n.roomReturning(_left.clamp(0, 99)),
              key: const Key('room-returning'),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 10),
            FilledButton.icon(
              key: const Key('back-to-room'),
              icon: const Icon(Icons.meeting_room),
              onPressed: () => _go(room),
              label: Text(l10n.backToRoom),
            ),
          ],
        ),
      ),
    );
  }
}
