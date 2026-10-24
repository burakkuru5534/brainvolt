import 'dart:async';

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../data/api_lobby_repository.dart';
import '../domain/lobby_repository.dart';

part 'room_controller.g.dart';

/// The room the player is in (for "back to the room" after a match).
@Riverpod(keepAlive: true)
class CurrentRoom extends _$CurrentRoom {
  @override
  String? build() => null;

  // A setter would read as a field; the room changes on purpose.
  // ignore: use_setters_to_change_properties
  void set(String? id) => state = id;
}

/// What the room screen shows.
class RoomView {
  const RoomView({this.room, this.closedReason, this.startingMatchId});

  final Room? room;

  /// host_closed / all_left / idle once the room is gone.
  final String? closedReason;

  /// Set by room.match_starting: the screen opens the match.
  final String? startingMatchId;
}

/// One private room: asks for its state on every (re)connect (room.join)
/// and follows the room stream (room.state, room.match_starting,
/// room.closed).
@riverpod
class RoomController extends _$RoomController {
  @override
  RoomView build(String roomId) {
    final rt = ref.watch(realtimeClientProvider)..start();
    final stream = 'room:$roomId';
    final frames = rt.frames.listen((f) {
      if (f.stream != stream) {
        return;
      }
      switch (f.type) {
        case 'room.state':
          final room = roomFromJson(
            (f.data['room'] as Map).cast<String, dynamic>(),
          );
          state = RoomView(room: room);
        case 'room.match_starting':
          final id = f.data['match_id'] as String;
          // Private rooms ready everyone at once (match-lifecycle §4).
          unawaited(
            rt
                .send('match.ready', {'match_id': id})
                .then<void>((_) {}, onError: (Object _) {}),
          );
          state = RoomView(room: state.room, startingMatchId: id);
        case 'room.closed':
          rt.forget(stream);
          // A matchmaking lobby closes once its match starts (D-094): the
          // screen is already on its way to the match.
          if (f.data['reason'] == 'match_started' &&
              state.startingMatchId != null) {
            return;
          }
          state = RoomView(closedReason: f.data['reason'] as String? ?? '');
      }
    });
    final status = rt.statusChanges.listen((s) {
      if (s == RealtimeStatus.connected) {
        unawaited(_join());
      }
    });
    ref.onDispose(() {
      unawaited(frames.cancel());
      unawaited(status.cancel());
    });
    if (rt.status == RealtimeStatus.connected) {
      unawaited(Future.microtask(_join));
    }
    return const RoomView();
  }

  RealtimeClient get _rt => ref.read(realtimeClientProvider);

  Future<void> _join() async {
    try {
      await _rt.send('room.join', {'room_id': roomId});
    } on RealtimeError catch (e) {
      if (e.code == 'ROOM_CLOSED' || e.code == 'NOT_FOUND') {
        state = RoomView(closedReason: e.code);
      }
    }
  }

  Future<void> setReady({required bool ready}) =>
      _rt.send(ready ? 'room.ready' : 'room.unready', {'room_id': roomId});

  Future<void> start() => _rt.send('room.start', {'room_id': roomId});

  Future<void> moveTo(int side, int seat) => _rt.send('room.update_seat', {
    'room_id': roomId,
    'side_no': side,
    'seat_no': seat,
  });
}
