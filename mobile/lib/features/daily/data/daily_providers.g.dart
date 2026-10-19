// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'daily_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The Daily Challenge endpoints. Tests override it with a fake.

@ProviderFor(dailyRepository)
final dailyRepositoryProvider = DailyRepositoryProvider._();

/// The Daily Challenge endpoints. Tests override it with a fake.

final class DailyRepositoryProvider
    extends
        $FunctionalProvider<DailyRepository, DailyRepository, DailyRepository>
    with $Provider<DailyRepository> {
  /// The Daily Challenge endpoints. Tests override it with a fake.
  DailyRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'dailyRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$dailyRepositoryHash();

  @$internal
  @override
  $ProviderElement<DailyRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  DailyRepository create(Ref ref) {
    return dailyRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(DailyRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<DailyRepository>(value),
    );
  }
}

String _$dailyRepositoryHash() => r'135c9df61fcc403ab5325777da5da37a99cf6102';
