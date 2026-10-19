// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'lobby_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The lobby endpoints. Tests override it with a fake.

@ProviderFor(lobbyRepository)
final lobbyRepositoryProvider = LobbyRepositoryProvider._();

/// The lobby endpoints. Tests override it with a fake.

final class LobbyRepositoryProvider
    extends
        $FunctionalProvider<LobbyRepository, LobbyRepository, LobbyRepository>
    with $Provider<LobbyRepository> {
  /// The lobby endpoints. Tests override it with a fake.
  LobbyRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'lobbyRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$lobbyRepositoryHash();

  @$internal
  @override
  $ProviderElement<LobbyRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  LobbyRepository create(Ref ref) {
    return lobbyRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(LobbyRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<LobbyRepository>(value),
    );
  }
}

String _$lobbyRepositoryHash() => r'1672cbde025c3f99f2ad7ba13f3386382c3bb8e1';
