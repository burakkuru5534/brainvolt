// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'play_controllers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Looking for an opponent (rest.md §5, websocket.md mm.*). The match is
/// announced by mm.match_found; /v1/matches/current is polled as well in
/// case the push was missed while the socket reconnected.

@ProviderFor(QueueController)
final queueControllerProvider = QueueControllerProvider._();

/// Looking for an opponent (rest.md §5, websocket.md mm.*). The match is
/// announced by mm.match_found; /v1/matches/current is polled as well in
/// case the push was missed while the socket reconnected.
final class QueueControllerProvider
    extends $NotifierProvider<QueueController, QueueState> {
  /// Looking for an opponent (rest.md §5, websocket.md mm.*). The match is
  /// announced by mm.match_found; /v1/matches/current is polled as well in
  /// case the push was missed while the socket reconnected.
  QueueControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'queueControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$queueControllerHash();

  @$internal
  @override
  QueueController create() => QueueController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(QueueState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<QueueState>(value),
    );
  }
}

String _$queueControllerHash() => r'6735a9f7e5a31b8840d0b62bb0742c6bf1be1349';

/// Looking for an opponent (rest.md §5, websocket.md mm.*). The match is
/// announced by mm.match_found; /v1/matches/current is polled as well in
/// case the push was missed while the socket reconnected.

abstract class _$QueueController extends $Notifier<QueueState> {
  QueueState build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<QueueState, QueueState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<QueueState, QueueState>,
              QueueState,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// One live match: joins it on every (re)connect, folds the match stream
/// into [MatchState] and sends the player's intents.

@ProviderFor(MatchController)
final matchControllerProvider = MatchControllerFamily._();

/// One live match: joins it on every (re)connect, folds the match stream
/// into [MatchState] and sends the player's intents.
final class MatchControllerProvider
    extends $NotifierProvider<MatchController, MatchState> {
  /// One live match: joins it on every (re)connect, folds the match stream
  /// into [MatchState] and sends the player's intents.
  MatchControllerProvider._({
    required MatchControllerFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'matchControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$matchControllerHash();

  @override
  String toString() {
    return r'matchControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  MatchController create() => MatchController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(MatchState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<MatchState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is MatchControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$matchControllerHash() => r'0ad54fa2cb171435d333f268876a655dc5f30968';

/// One live match: joins it on every (re)connect, folds the match stream
/// into [MatchState] and sends the player's intents.

final class MatchControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          MatchController,
          MatchState,
          MatchState,
          MatchState,
          String
        > {
  MatchControllerFamily._()
    : super(
        retry: null,
        name: r'matchControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// One live match: joins it on every (re)connect, folds the match stream
  /// into [MatchState] and sends the player's intents.

  MatchControllerProvider call(String matchId) =>
      MatchControllerProvider._(argument: matchId, from: this);

  @override
  String toString() => r'matchControllerProvider';
}

/// One live match: joins it on every (re)connect, folds the match stream
/// into [MatchState] and sends the player's intents.

abstract class _$MatchController extends $Notifier<MatchState> {
  late final _$args = ref.$arg as String;
  String get matchId => _$args;

  MatchState build(String matchId);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<MatchState, MatchState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<MatchState, MatchState>,
              MatchState,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}

/// "Play again" on the result screen: everyone accepts within 30 s of the
/// end → game creates the rematch (rematch.result carries its id).

@ProviderFor(RematchController)
final rematchControllerProvider = RematchControllerFamily._();

/// "Play again" on the result screen: everyone accepts within 30 s of the
/// end → game creates the rematch (rematch.result carries its id).
final class RematchControllerProvider
    extends $NotifierProvider<RematchController, RematchState> {
  /// "Play again" on the result screen: everyone accepts within 30 s of the
  /// end → game creates the rematch (rematch.result carries its id).
  RematchControllerProvider._({
    required RematchControllerFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'rematchControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$rematchControllerHash();

  @override
  String toString() {
    return r'rematchControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  RematchController create() => RematchController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RematchState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RematchState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is RematchControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$rematchControllerHash() => r'7b5b6454b4afe7e090a7cbbe5930f2fd79018a6f';

/// "Play again" on the result screen: everyone accepts within 30 s of the
/// end → game creates the rematch (rematch.result carries its id).

final class RematchControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          RematchController,
          RematchState,
          RematchState,
          RematchState,
          String
        > {
  RematchControllerFamily._()
    : super(
        retry: null,
        name: r'rematchControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// "Play again" on the result screen: everyone accepts within 30 s of the
  /// end → game creates the rematch (rematch.result carries its id).

  RematchControllerProvider call(String matchId) =>
      RematchControllerProvider._(argument: matchId, from: this);

  @override
  String toString() => r'rematchControllerProvider';
}

/// "Play again" on the result screen: everyone accepts within 30 s of the
/// end → game creates the rematch (rematch.result carries its id).

abstract class _$RematchController extends $Notifier<RematchState> {
  late final _$args = ref.$arg as String;
  String get matchId => _$args;

  RematchState build(String matchId);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<RematchState, RematchState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<RematchState, RematchState>,
              RematchState,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
