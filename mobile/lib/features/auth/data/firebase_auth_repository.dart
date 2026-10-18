import 'dart:convert';
import 'dart:math';

import 'package:crypto/crypto.dart';
import 'package:firebase_auth/firebase_auth.dart' as fb;
import 'package:flutter/foundation.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:sign_in_with_apple/sign_in_with_apple.dart';

import '../domain/auth_repository.dart';

/// [AuthRepository] on Firebase Auth, Google Sign-In and Sign in with Apple.
class FirebaseAuthRepository implements AuthRepository {
  FirebaseAuthRepository({
    fb.FirebaseAuth? auth,
    GoogleSignIn? google,
    this.googleClientId,
  }) : _auth = auth ?? fb.FirebaseAuth.instance,
       _google = google ?? GoogleSignIn.instance;

  final fb.FirebaseAuth _auth;
  final GoogleSignIn _google;

  /// iOS OAuth client id (`DefaultFirebaseOptions.ios.iosClientId`). Android
  /// reads the web client from google-services.json.
  final String? googleClientId;

  Future<void>? _googleReady;

  @override
  AuthUser? get currentUser => _map(_auth.currentUser);

  @override
  Future<AuthUser> signInAnonymously() =>
      _guard(() async => _user((await _auth.signInAnonymously()).user));

  @override
  Future<ProviderCredential> obtainCredential(
    AuthMethod method, {
    String? email,
    String? password,
  }) => _guard(() async {
    final fb.AuthCredential credential = switch (method) {
      AuthMethod.google => await _googleCredential(),
      AuthMethod.apple => await _appleCredential(),
      AuthMethod.email => fb.EmailAuthProvider.credential(
        email: (email ?? '').trim(),
        password: password ?? '',
      ),
    };
    return ProviderCredential(method, credential);
  });

  Future<fb.AuthCredential> _googleCredential() async {
    await (_googleReady ??= _google.initialize(clientId: googleClientId));
    if (!_google.supportsAuthenticate()) {
      throw const AuthFailure(AuthFailureKind.unsupported);
    }
    final account = await _google.authenticate();
    final idToken = account.authentication.idToken;
    if (idToken == null) {
      throw const AuthFailure(AuthFailureKind.invalidCredential);
    }
    return fb.GoogleAuthProvider.credential(idToken: idToken);
  }

  /// Sign in with Apple with a nonce: Apple embeds sha256(rawNonce) in its
  /// token and Firebase checks it against the raw value.
  Future<fb.AuthCredential> _appleCredential() async {
    if (!await SignInWithApple.isAvailable()) {
      throw const AuthFailure(AuthFailureKind.unsupported);
    }
    final rawNonce = _nonce();
    final apple = await SignInWithApple.getAppleIDCredential(
      scopes: [AppleIDAuthorizationScopes.email],
      nonce: sha256.convert(utf8.encode(rawNonce)).toString(),
    );
    final idToken = apple.identityToken;
    if (idToken == null) {
      throw const AuthFailure(AuthFailureKind.invalidCredential);
    }
    return fb.OAuthProvider('apple.com').credential(
      idToken: idToken,
      rawNonce: rawNonce,
      accessToken: apple.authorizationCode,
    );
  }

  @override
  Future<AuthUser> link(ProviderCredential credential) => _guard(() async {
    final user = _auth.currentUser;
    if (user == null) {
      throw const AuthFailure(AuthFailureKind.unknown);
    }
    try {
      return _user((await user.linkWithCredential(_raw(credential))).user);
    } on fb.FirebaseAuthException catch (e) {
      if (const {
        'credential-already-in-use',
        'email-already-in-use',
        'account-exists-with-different-credential',
      }.contains(e.code)) {
        // Firebase hands back a usable credential for OAuth providers; an
        // email credential can simply be reused.
        final conflict = e.credential == null
            ? credential
            : ProviderCredential(credential.method, e.credential!);
        throw AuthFailure(
          AuthFailureKind.credentialAlreadyInUse,
          conflict: conflict,
          cause: e,
        );
      }
      rethrow;
    }
  });

  @override
  Future<AuthUser> signIn(ProviderCredential credential) => _guard(
    () async =>
        _user((await _auth.signInWithCredential(_raw(credential))).user),
  );

  @override
  Future<void> sendEmailVerification() => _guard(() async {
    await _auth.currentUser?.sendEmailVerification();
  });

  @override
  Future<AuthUser?> reload() => _guard(() async {
    await _auth.currentUser?.reload();
    return currentUser;
  });

  @override
  Future<String?> idToken({bool forceRefresh = false}) async =>
      _auth.currentUser?.getIdToken(forceRefresh);

  @override
  Future<void> signOut() async {
    if (_googleReady != null) {
      try {
        await _google.signOut();
      } on Object catch (e) {
        debugPrint('Google sign-out failed: $e');
      }
    }
    await _auth.signOut();
  }

  static fb.AuthCredential _raw(ProviderCredential c) =>
      c.raw as fb.AuthCredential;

  static AuthUser _user(fb.User? user) =>
      _map(user) ?? (throw const AuthFailure(AuthFailureKind.unknown));

  static AuthUser? _map(fb.User? user) {
    if (user == null) {
      return null;
    }
    return AuthUser(
      uid: user.uid,
      isAnonymous: user.isAnonymous,
      email: user.email,
      emailVerified: user.emailVerified,
      providers: {
        for (final p in user.providerData)
          ?switch (p.providerId) {
            'google.com' => AuthMethod.google,
            'apple.com' => AuthMethod.apple,
            'password' => AuthMethod.email,
            _ => null,
          },
      },
    );
  }

  static String _nonce([int length = 32]) {
    const chars =
        '0123456789ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz-._';
    final random = Random.secure();
    return List.generate(
      length,
      (_) => chars[random.nextInt(chars.length)],
    ).join();
  }

  /// Runs [call] and turns platform errors into [AuthFailure]s.
  static Future<T> _guard<T>(Future<T> Function() call) async {
    try {
      return await call();
    } on AuthFailure {
      rethrow;
    } on fb.FirebaseAuthException catch (e) {
      throw AuthFailure(firebaseFailureKind(e.code), cause: e);
    } on GoogleSignInException catch (e) {
      throw AuthFailure(switch (e.code) {
        GoogleSignInExceptionCode.canceled ||
        GoogleSignInExceptionCode.interrupted => AuthFailureKind.cancelled,
        GoogleSignInExceptionCode.uiUnavailable ||
        GoogleSignInExceptionCode.clientConfigurationError ||
        GoogleSignInExceptionCode.providerConfigurationError =>
          AuthFailureKind.unsupported,
        _ => AuthFailureKind.unknown,
      }, cause: e);
    } on SignInWithAppleAuthorizationException catch (e) {
      throw AuthFailure(
        e.code == AuthorizationErrorCode.canceled
            ? AuthFailureKind.cancelled
            : AuthFailureKind.unknown,
        cause: e,
      );
    } on SignInWithAppleNotSupportedException catch (e) {
      throw AuthFailure(AuthFailureKind.unsupported, cause: e);
    }
  }
}

/// Maps a FirebaseAuthException code (firebase_auth docs) to a failure kind.
@visibleForTesting
AuthFailureKind firebaseFailureKind(String code) => switch (code) {
  'credential-already-in-use' ||
  'email-already-in-use' ||
  'account-exists-with-different-credential' =>
    AuthFailureKind.credentialAlreadyInUse,
  'invalid-credential' ||
  'wrong-password' ||
  'invalid-verification-code' ||
  'user-mismatch' => AuthFailureKind.invalidCredential,
  'user-not-found' => AuthFailureKind.userNotFound,
  'weak-password' => AuthFailureKind.weakPassword,
  'invalid-email' => AuthFailureKind.invalidEmail,
  'too-many-requests' => AuthFailureKind.tooManyRequests,
  'network-request-failed' => AuthFailureKind.network,
  'operation-not-allowed' => AuthFailureKind.unsupported,
  _ => AuthFailureKind.unknown,
};
