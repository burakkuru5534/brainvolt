// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'session_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The app's sign-in state (roadmap 1.8.3), app-wide like the router so
/// every feature can read it: age gate → Firebase user (an
/// anonymous guest on first launch) → `POST /v1/session`. Every account
/// action ends with a fresh session sync so tier and flags stay current.

@ProviderFor(SessionController)
final sessionControllerProvider = SessionControllerProvider._();

/// The app's sign-in state (roadmap 1.8.3), app-wide like the router so
/// every feature can read it: age gate → Firebase user (an
/// anonymous guest on first launch) → `POST /v1/session`. Every account
/// action ends with a fresh session sync so tier and flags stay current.
final class SessionControllerProvider
    extends $AsyncNotifierProvider<SessionController, AppSession> {
  /// The app's sign-in state (roadmap 1.8.3), app-wide like the router so
  /// every feature can read it: age gate → Firebase user (an
  /// anonymous guest on first launch) → `POST /v1/session`. Every account
  /// action ends with a fresh session sync so tier and flags stay current.
  SessionControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: _noRetry,
        name: r'sessionControllerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sessionControllerHash();

  @$internal
  @override
  SessionController create() => SessionController();
}

String _$sessionControllerHash() => r'9b904d45614091dd9c7e1a3b10f9194aad87346d';

/// The app's sign-in state (roadmap 1.8.3), app-wide like the router so
/// every feature can read it: age gate → Firebase user (an
/// anonymous guest on first launch) → `POST /v1/session`. Every account
/// action ends with a fresh session sync so tier and flags stay current.

abstract class _$SessionController extends $AsyncNotifier<AppSession> {
  FutureOr<AppSession> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<AppSession>, AppSession>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<AppSession>, AppSession>,
              AsyncValue<AppSession>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
