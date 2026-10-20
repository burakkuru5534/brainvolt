// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tournament_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The tournament endpoints. Tests override it.

@ProviderFor(tournamentRepository)
final tournamentRepositoryProvider = TournamentRepositoryProvider._();

/// The tournament endpoints. Tests override it.

final class TournamentRepositoryProvider
    extends
        $FunctionalProvider<
          TournamentRepository,
          TournamentRepository,
          TournamentRepository
        >
    with $Provider<TournamentRepository> {
  /// The tournament endpoints. Tests override it.
  TournamentRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tournamentRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tournamentRepositoryHash();

  @$internal
  @override
  $ProviderElement<TournamentRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  TournamentRepository create(Ref ref) {
    return tournamentRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(TournamentRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<TournamentRepository>(value),
    );
  }
}

String _$tournamentRepositoryHash() =>
    r'2afc22eed948810a920a065639e480ff00584887';
