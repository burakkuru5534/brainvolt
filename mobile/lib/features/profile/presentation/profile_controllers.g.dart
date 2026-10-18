// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_controllers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// `GET /v1/me` for the profile screen; reloaded every time it opens.

@ProviderFor(me)
final meProvider = MeProvider._();

/// `GET /v1/me` for the profile screen; reloaded every time it opens.

final class MeProvider
    extends $FunctionalProvider<AsyncValue<Me>, Me, FutureOr<Me>>
    with $FutureModifier<Me>, $FutureProvider<Me> {
  /// `GET /v1/me` for the profile screen; reloaded every time it opens.
  MeProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'meProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$meHash();

  @$internal
  @override
  $FutureProviderElement<Me> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<Me> create(Ref ref) {
    return me(ref);
  }
}

String _$meHash() => r'22fdff6ef41191e4c7381561c4def0428e008c5e';

/// Privacy and app settings, saved on every change (roadmap 1.8.6).

@ProviderFor(SettingsController)
final settingsControllerProvider = SettingsControllerProvider._();

/// Privacy and app settings, saved on every change (roadmap 1.8.6).
final class SettingsControllerProvider
    extends $AsyncNotifierProvider<SettingsController, UserSettings> {
  /// Privacy and app settings, saved on every change (roadmap 1.8.6).
  SettingsControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'settingsControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$settingsControllerHash();

  @$internal
  @override
  SettingsController create() => SettingsController();
}

String _$settingsControllerHash() =>
    r'19e3a4f2aad2cd0e811168c208caf2ee7296144b';

/// Privacy and app settings, saved on every change (roadmap 1.8.6).

abstract class _$SettingsController extends $AsyncNotifier<UserSettings> {
  FutureOr<UserSettings> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<UserSettings>, UserSettings>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<UserSettings>, UserSettings>,
              AsyncValue<UserSettings>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
