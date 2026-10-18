// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'api_client.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Firebase ID tokens. `main.dart` overrides it with the auth feature's
/// source (core never imports features, ADR-0003); unauthenticated until then.

@ProviderFor(tokenSource)
final tokenSourceProvider = TokenSourceProvider._();

/// Firebase ID tokens. `main.dart` overrides it with the auth feature's
/// source (core never imports features, ADR-0003); unauthenticated until then.

final class TokenSourceProvider
    extends $FunctionalProvider<TokenSource, TokenSource, TokenSource>
    with $Provider<TokenSource> {
  /// Firebase ID tokens. `main.dart` overrides it with the auth feature's
  /// source (core never imports features, ADR-0003); unauthenticated until then.
  TokenSourceProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tokenSourceProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tokenSourceHash();

  @$internal
  @override
  $ProviderElement<TokenSource> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  TokenSource create(Ref ref) {
    return tokenSource(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(TokenSource value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<TokenSource>(value),
    );
  }
}

String _$tokenSourceHash() => r'38d11e44159c450e797e1a6b2e7a634628eab2f6';

/// The API client of the running app.

@ProviderFor(apiClient)
final apiClientProvider = ApiClientProvider._();

/// The API client of the running app.

final class ApiClientProvider
    extends $FunctionalProvider<ApiClient, ApiClient, ApiClient>
    with $Provider<ApiClient> {
  /// The API client of the running app.
  ApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'apiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$apiClientHash();

  @$internal
  @override
  $ProviderElement<ApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ApiClient create(Ref ref) {
    return apiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ApiClient>(value),
    );
  }
}

String _$apiClientHash() => r'ca9e071a8bc1174505eba7735c97b4067167e72a';
