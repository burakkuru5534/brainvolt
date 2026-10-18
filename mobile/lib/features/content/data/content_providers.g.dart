// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'content_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The content endpoints. Tests override it with a fake.

@ProviderFor(contentRepository)
final contentRepositoryProvider = ContentRepositoryProvider._();

/// The content endpoints. Tests override it with a fake.

final class ContentRepositoryProvider
    extends
        $FunctionalProvider<
          ContentRepository,
          ContentRepository,
          ContentRepository
        >
    with $Provider<ContentRepository> {
  /// The content endpoints. Tests override it with a fake.
  ContentRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'contentRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$contentRepositoryHash();

  @$internal
  @override
  $ProviderElement<ContentRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  ContentRepository create(Ref ref) {
    return contentRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ContentRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ContentRepository>(value),
    );
  }
}

String _$contentRepositoryHash() => r'a274d052b1720f15b765794f46f389addaee318f';
