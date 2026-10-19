// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'room_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The room the player is in (for "back to the room" after a match).

@ProviderFor(CurrentRoom)
final currentRoomProvider = CurrentRoomProvider._();

/// The room the player is in (for "back to the room" after a match).
final class CurrentRoomProvider
    extends $NotifierProvider<CurrentRoom, String?> {
  /// The room the player is in (for "back to the room" after a match).
  CurrentRoomProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'currentRoomProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$currentRoomHash();

  @$internal
  @override
  CurrentRoom create() => CurrentRoom();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String?>(value),
    );
  }
}

String _$currentRoomHash() => r'486a5470b84505396e2a307f770f7c1e1e8e9c63';

/// The room the player is in (for "back to the room" after a match).

abstract class _$CurrentRoom extends $Notifier<String?> {
  String? build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<String?, String?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<String?, String?>,
              String?,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// One private room: asks for its state on every (re)connect (room.join)
/// and follows the room stream (room.state, room.match_starting,
/// room.closed).

@ProviderFor(RoomController)
final roomControllerProvider = RoomControllerFamily._();

/// One private room: asks for its state on every (re)connect (room.join)
/// and follows the room stream (room.state, room.match_starting,
/// room.closed).
final class RoomControllerProvider
    extends $NotifierProvider<RoomController, RoomView> {
  /// One private room: asks for its state on every (re)connect (room.join)
  /// and follows the room stream (room.state, room.match_starting,
  /// room.closed).
  RoomControllerProvider._({
    required RoomControllerFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'roomControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$roomControllerHash();

  @override
  String toString() {
    return r'roomControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  RoomController create() => RoomController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RoomView value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RoomView>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is RoomControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$roomControllerHash() => r'cc5091517b59b93b46b4d672f0113ebf9b235610';

/// One private room: asks for its state on every (re)connect (room.join)
/// and follows the room stream (room.state, room.match_starting,
/// room.closed).

final class RoomControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          RoomController,
          RoomView,
          RoomView,
          RoomView,
          String
        > {
  RoomControllerFamily._()
    : super(
        retry: null,
        name: r'roomControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// One private room: asks for its state on every (re)connect (room.join)
  /// and follows the room stream (room.state, room.match_starting,
  /// room.closed).

  RoomControllerProvider call(String roomId) =>
      RoomControllerProvider._(argument: roomId, from: this);

  @override
  String toString() => r'roomControllerProvider';
}

/// One private room: asks for its state on every (re)connect (room.join)
/// and follows the room stream (room.state, room.match_starting,
/// room.closed).

abstract class _$RoomController extends $Notifier<RoomView> {
  late final _$args = ref.$arg as String;
  String get roomId => _$args;

  RoomView build(String roomId);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<RoomView, RoomView>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<RoomView, RoomView>,
              RoomView,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
