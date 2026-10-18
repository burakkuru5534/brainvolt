// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'match_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The game and matchmaking endpoints. Tests override it with a fake.

@ProviderFor(matchRepository)
final matchRepositoryProvider = MatchRepositoryProvider._();

/// The game and matchmaking endpoints. Tests override it with a fake.

final class MatchRepositoryProvider
    extends
        $FunctionalProvider<MatchRepository, MatchRepository, MatchRepository>
    with $Provider<MatchRepository> {
  /// The game and matchmaking endpoints. Tests override it with a fake.
  MatchRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'matchRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$matchRepositoryHash();

  @$internal
  @override
  $ProviderElement<MatchRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  MatchRepository create(Ref ref) {
    return matchRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(MatchRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<MatchRepository>(value),
    );
  }
}

String _$matchRepositoryHash() => r'6e2e4db6fdf7fcfccc5f9e9535a1960a8fac7745';
