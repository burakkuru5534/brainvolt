// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'locale_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The user's language choice; `null` follows the device (roadmap 1.7.5).
/// Persisted in shared preferences.

@ProviderFor(LocaleOverride)
final localeOverrideProvider = LocaleOverrideProvider._();

/// The user's language choice; `null` follows the device (roadmap 1.7.5).
/// Persisted in shared preferences.
final class LocaleOverrideProvider
    extends $NotifierProvider<LocaleOverride, Locale?> {
  /// The user's language choice; `null` follows the device (roadmap 1.7.5).
  /// Persisted in shared preferences.
  LocaleOverrideProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'localeOverrideProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$localeOverrideHash();

  @$internal
  @override
  LocaleOverride create() => LocaleOverride();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Locale? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Locale?>(value),
    );
  }
}

String _$localeOverrideHash() => r'dc8022c4d1104e263aa9ceeb9369220140d25373';

/// The user's language choice; `null` follows the device (roadmap 1.7.5).
/// Persisted in shared preferences.

abstract class _$LocaleOverride extends $Notifier<Locale?> {
  Locale? build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<Locale?, Locale?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<Locale?, Locale?>,
              Locale?,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
