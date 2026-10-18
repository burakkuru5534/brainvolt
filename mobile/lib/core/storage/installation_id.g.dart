// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'installation_id.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// A random id for this app install (UUIDv7), created on first use and kept
/// until the app is uninstalled. Sent with the session and device calls; the
/// backend rate-limits sessions per installation (D-059).

@ProviderFor(installationId)
final installationIdProvider = InstallationIdProvider._();

/// A random id for this app install (UUIDv7), created on first use and kept
/// until the app is uninstalled. Sent with the session and device calls; the
/// backend rate-limits sessions per installation (D-059).

final class InstallationIdProvider
    extends $FunctionalProvider<String, String, String>
    with $Provider<String> {
  /// A random id for this app install (UUIDv7), created on first use and kept
  /// until the app is uninstalled. Sent with the session and device calls; the
  /// backend rate-limits sessions per installation (D-059).
  InstallationIdProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'installationIdProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$installationIdHash();

  @$internal
  @override
  $ProviderElement<String> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  String create(Ref ref) {
    return installationId(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String>(value),
    );
  }
}

String _$installationIdHash() => r'b8b7e0f3cb6399bd0a53f21b7c4620f2154e5ea1';
