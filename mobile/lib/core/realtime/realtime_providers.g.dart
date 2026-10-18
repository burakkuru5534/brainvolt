// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'realtime_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The app's WebSocket. Features call `start()` when they need it; it stays
/// connected (with reconnects) from then on.

@ProviderFor(realtimeClient)
final realtimeClientProvider = RealtimeClientProvider._();

/// The app's WebSocket. Features call `start()` when they need it; it stays
/// connected (with reconnects) from then on.

final class RealtimeClientProvider
    extends $FunctionalProvider<RealtimeClient, RealtimeClient, RealtimeClient>
    with $Provider<RealtimeClient> {
  /// The app's WebSocket. Features call `start()` when they need it; it stays
  /// connected (with reconnects) from then on.
  RealtimeClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'realtimeClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$realtimeClientHash();

  @$internal
  @override
  $ProviderElement<RealtimeClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  RealtimeClient create(Ref ref) {
    return realtimeClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RealtimeClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RealtimeClient>(value),
    );
  }
}

String _$realtimeClientHash() => r'be68915b7c8eb1f887a5d3a9e3a120f8ac5abfff';
