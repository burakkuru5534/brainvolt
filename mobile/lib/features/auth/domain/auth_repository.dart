/// Sign-in with Firebase Auth (ADR-0003, D-013): the app always has a
/// Firebase user — an anonymous guest at first, upgraded later by linking a
/// Google, Apple or email credential to the same user (D-013a).
library;

/// The signed-in Firebase user, as far as the app needs it.
class AuthUser {
  const AuthUser({
    required this.uid,
    required this.isAnonymous,
    this.email,
    this.emailVerified = false,
    this.providers = const {},
  });

  final String uid;
  final bool isAnonymous;
  final String? email;
  final bool emailVerified;

  /// Linked sign-in methods.
  final Set<AuthMethod> providers;
}

/// Sign-in methods besides the anonymous guest (D-013).
enum AuthMethod { google, apple, email }

/// A credential obtained from a provider's sign-in UI (or an email and
/// password), used to link or sign in. Opaque outside the data layer.
class ProviderCredential {
  const ProviderCredential(this.method, this.raw);

  final AuthMethod method;

  /// The platform credential (a Firebase `AuthCredential`).
  final Object raw;
}

/// Why an auth call failed.
enum AuthFailureKind {
  /// The credential belongs to another account (link conflict, D-015).
  credentialAlreadyInUse,

  /// Wrong email/password or an invalid provider token.
  invalidCredential,
  weakPassword,
  invalidEmail,

  /// No account with this email (sign-in only).
  userNotFound,

  /// The user closed the provider's sign-in UI.
  cancelled,
  tooManyRequests,
  network,

  /// The sign-in method is not available on this device or build.
  unsupported,

  /// The signed-in Firebase user is gone or its refresh token is no longer
  /// valid (deleted or disabled account, or a restarted Auth emulator).
  sessionLost,
  unknown,
}

/// A failed auth call.
class AuthFailure implements Exception {
  const AuthFailure(this.kind, {this.conflict, this.cause});

  final AuthFailureKind kind;

  /// For [AuthFailureKind.credentialAlreadyInUse]: the credential to sign
  /// in to the existing account with.
  final ProviderCredential? conflict;

  /// The underlying error, for logs.
  final Object? cause;

  @override
  String toString() => 'AuthFailure($kind, $cause)';
}

/// Firebase Auth behind an interface so tests can replace it.
abstract interface class AuthRepository {
  /// The signed-in user, or null.
  AuthUser? get currentUser;

  /// Creates an anonymous guest user.
  Future<AuthUser> signInAnonymously();

  /// Opens the provider's sign-in UI (Google, Apple) or wraps [email] and
  /// [password]. Throws [AuthFailure] (`cancelled` when the user backs out).
  Future<ProviderCredential> obtainCredential(
    AuthMethod method, {
    String? email,
    String? password,
  });

  /// Links [credential] to the current user (guest upgrade, D-013a). Throws
  /// [AuthFailure] with `credentialAlreadyInUse` and [AuthFailure.conflict]
  /// when another account owns it.
  Future<AuthUser> link(ProviderCredential credential);

  /// Signs in with [credential], replacing the current user.
  Future<AuthUser> signIn(ProviderCredential credential);

  /// Sends the verification email to the current user.
  Future<void> sendEmailVerification();

  /// Reloads the current user from Firebase (after email verification).
  Future<AuthUser?> reload();

  /// The current ID token; null when signed out.
  Future<String?> idToken({bool forceRefresh = false});

  /// Signs out of Firebase and the provider SDKs.
  Future<void> signOut();
}
