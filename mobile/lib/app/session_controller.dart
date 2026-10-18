import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../core/network/api_exception.dart';
import '../features/auth/data/auth_providers.dart';
import '../features/auth/domain/auth_repository.dart';
import '../features/auth/domain/session.dart';
import 'providers.dart';

part 'session_controller.g.dart';

final _log = Logger('auth');

/// Preferences key of the age gate answer (D-020). Only the fact is kept on
/// the device; the backend stores `age_verified_at`.
const ageConfirmedKey = 'age_confirmed';

/// Server code for a token whose user the server cannot accept.
const unauthenticatedCode = 'UNAUTHENTICATED';

/// Where the app stands before showing game screens.
sealed class AppSession {
  const AppSession();
}

/// First launch: the age gate must be confirmed before any sign-in.
final class AgeGateRequired extends AppSession {
  const AgeGateRequired();
}

/// A Firebase user exists and the backend session is synced.
final class SignedIn extends AppSession {
  const SignedIn(this.session);

  final Session session;
}

/// The outcome of a guest upgrade.
sealed class UpgradeResult {
  const UpgradeResult();
}

/// The credential is now linked to the guest: same user, progress kept.
final class Upgraded extends UpgradeResult {
  const Upgraded();
}

/// The credential belongs to another account (D-015): ask the user, then
/// call [SessionController.switchAccount] with [credential].
final class UpgradeConflict extends UpgradeResult {
  const UpgradeConflict(this.credential);

  final ProviderCredential credential;
}

// Errors are shown on the splash screen with a retry button instead of being
// retried silently.
Duration? _noRetry(int retryCount, Object error) => null;

/// The app's sign-in state (roadmap 1.8.3), app-wide like the router so
/// every feature can read it: age gate → Firebase user (an
/// anonymous guest on first launch) → `POST /v1/session`. Every account
/// action ends with a fresh session sync so tier and flags stay current.
@Riverpod(keepAlive: true, retry: _noRetry)
class SessionController extends _$SessionController {
  AuthRepository get _auth => ref.read(authRepositoryProvider);

  @override
  Future<AppSession> build() async {
    final prefs = ref.watch(sharedPreferencesProvider);
    if (prefs.getBool(ageConfirmedKey) != true) {
      return const AgeGateRequired();
    }
    return SignedIn(await _sync());
  }

  Future<Session> _sync({bool forceTokenRefresh = false}) async {
    if (_auth.currentUser == null) {
      await _auth.signInAnonymously();
    } else if (forceTokenRefresh) {
      // A new token carries the providers and email_verified the backend
      // derives the tier from (D-059).
      await _auth.idToken(forceRefresh: true);
    }
    final sessions = ref.read(sessionRepositoryProvider);
    Session session;
    try {
      session = await sessions.sync();
    } on ApiException catch (e) {
      if (e.code != unauthenticatedCode) {
        rethrow;
      }
      // The server no longer knows this Firebase user (deleted account, or
      // a restarted Auth emulator in development): start over as a new
      // guest instead of failing on every launch. TOKEN_EXPIRED never gets
      // here; the API client refreshes and retries it.
      _log.warning(
        'Firebase user unknown to the server; continuing as a new guest',
      );
      await _auth.signOut();
      await _auth.signInAnonymously();
      session = await sessions.sync();
    }
    // Not needed to play; a failure is retried on the next sync.
    sessions.registerDevice().catchError((Object e) {
      _log.warning('device registration failed', e);
    }).ignore();
    return session;
  }

  Session? get _session => switch (state.value) {
    SignedIn(:final session) => session,
    _ => null,
  };

  /// The user confirmed the age gate (D-020).
  Future<void> confirmAge() async {
    await ref.read(sharedPreferencesProvider).setBool(ageConfirmedKey, true);
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async => SignedIn(await _sync()));
  }

  /// Re-syncs the session (after a link, email verification or an error).
  Future<void> refresh() async {
    state = await AsyncValue.guard(
      () async => SignedIn(await _sync(forceTokenRefresh: true)),
    );
  }

  /// Guest upgrade (D-013a): opens the provider UI (or uses [email] and
  /// [password]) and links the credential to the current user. Throws
  /// [AuthFailure].
  Future<UpgradeResult> upgrade(
    AuthMethod method, {
    String? email,
    String? password,
  }) async {
    final credential = await _auth.obtainCredential(
      method,
      email: email,
      password: password,
    );
    try {
      await _auth.link(credential);
    } on AuthFailure catch (e) {
      if (e.kind == AuthFailureKind.credentialAlreadyInUse &&
          e.conflict != null) {
        return UpgradeConflict(e.conflict!);
      }
      rethrow;
    }
    if (method == AuthMethod.email) {
      await _sendVerification();
    }
    await refresh();
    return const Upgraded();
  }

  /// Obtains a credential for signing in to an existing account (email
  /// sign-in from a guest); pass it to [switchAccount] after the D-015
  /// warning.
  Future<ProviderCredential> credentialFor(
    AuthMethod method, {
    String? email,
    String? password,
  }) => _auth.obtainCredential(method, email: email, password: password);

  /// Link conflict, confirmed by the user (D-015): signs in to the existing
  /// account, then marks the guest abandoned with the guest's own token.
  /// Signing in first means a wrong password or a cancelled sign-in leaves
  /// the guest untouched (D-061).
  Future<void> switchAccount(ProviderCredential credential) async {
    final wasGuest = _session?.tier == Tier.guest;
    final guestToken = wasGuest
        ? await _auth.idToken(forceRefresh: true)
        : null;
    await _auth.signIn(credential);
    if (guestToken != null) {
      try {
        await ref
            .read(sessionRepositoryProvider)
            .abandonGuest(guestToken: guestToken);
      } on Object catch (e) {
        // The guest stays an orphan that guest cleanup removes (D-016).
        _log.warning('abandon-guest failed', e);
      }
    }
    await refresh();
  }

  /// Sends the verification email again (D-017).
  Future<void> resendVerification() => _auth.sendEmailVerification();

  Future<void> _sendVerification() async {
    try {
      await _auth.sendEmailVerification();
    } on AuthFailure catch (e) {
      // The verify screen offers "send again".
      _log.warning('verification email failed', e);
    }
  }

  /// After the user opened the verification link: reloads the Firebase
  /// user and, when verified, syncs the new tier. Returns whether the email
  /// is verified.
  Future<bool> checkEmailVerified() async {
    final user = await _auth.reload();
    if (user?.emailVerified != true) {
      return false;
    }
    await refresh();
    return true;
  }

  /// Signs out on this device; the app continues as a new guest.
  Future<void> signOut() async {
    try {
      await ref.read(sessionRepositoryProvider).unregisterDevice();
    } on Object catch (e) {
      _log.warning('device sign-out failed', e);
    }
    await _auth.signOut();
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async => SignedIn(await _sync()));
  }

  /// Starts account deletion (D-022); returns the purge date.
  Future<DateTime> requestDeletion() async {
    final purgeAfter = await ref
        .read(sessionRepositoryProvider)
        .requestDeletion();
    await refresh();
    return purgeAfter;
  }

  /// Cancels a pending deletion.
  Future<void> cancelDeletion() async {
    await ref.read(sessionRepositoryProvider).cancelDeletion();
    await refresh();
  }
}
