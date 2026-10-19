// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'creator_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The creator endpoints. Tests override it with a fake.

@ProviderFor(creatorRepository)
final creatorRepositoryProvider = CreatorRepositoryProvider._();

/// The creator endpoints. Tests override it with a fake.

final class CreatorRepositoryProvider
    extends
        $FunctionalProvider<
          CreatorRepository,
          CreatorRepository,
          CreatorRepository
        >
    with $Provider<CreatorRepository> {
  /// The creator endpoints. Tests override it with a fake.
  CreatorRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'creatorRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$creatorRepositoryHash();

  @$internal
  @override
  $ProviderElement<CreatorRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  CreatorRepository create(Ref ref) {
    return creatorRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(CreatorRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<CreatorRepository>(value),
    );
  }
}

String _$creatorRepositoryHash() => r'1fb0afa8b48eee9161e297f2881f9e346faf7208';
