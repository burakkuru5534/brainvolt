// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'theme.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The former light / dark choice (D-085). BrainVolt is dark only
/// (D-099); the stored value is kept but no longer applied.

@ProviderFor(ThemeModeSetting)
final themeModeSettingProvider = ThemeModeSettingProvider._();

/// The former light / dark choice (D-085). BrainVolt is dark only
/// (D-099); the stored value is kept but no longer applied.
final class ThemeModeSettingProvider
    extends $NotifierProvider<ThemeModeSetting, ThemeMode> {
  /// The former light / dark choice (D-085). BrainVolt is dark only
  /// (D-099); the stored value is kept but no longer applied.
  ThemeModeSettingProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'themeModeSettingProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$themeModeSettingHash();

  @$internal
  @override
  ThemeModeSetting create() => ThemeModeSetting();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ThemeMode value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ThemeMode>(value),
    );
  }
}

String _$themeModeSettingHash() => r'b52a351bb2291cdec71ce71bc941fa6057f7d8a4';

/// The former light / dark choice (D-085). BrainVolt is dark only
/// (D-099); the stored value is kept but no longer applied.

abstract class _$ThemeModeSetting extends $Notifier<ThemeMode> {
  ThemeMode build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<ThemeMode, ThemeMode>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<ThemeMode, ThemeMode>,
              ThemeMode,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
