/// Supplies Firebase ID tokens to the API client. The Firebase-backed
/// implementation arrives with sign-in (roadmap 1.8).
abstract interface class TokenSource {
  /// The current ID token, or null when signed out. [forceRefresh] asks for
  /// a new token after the server answered TOKEN_EXPIRED.
  Future<String?> idToken({bool forceRefresh = false});
}

/// The state before sign-in: no token, requests go out unauthenticated.
class SignedOutTokenSource implements TokenSource {
  const SignedOutTokenSource();

  @override
  Future<String?> idToken({bool forceRefresh = false}) async => null;
}
