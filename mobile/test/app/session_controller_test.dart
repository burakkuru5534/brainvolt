import 'package:brainvolt/app/session_controller.dart';
import 'package:brainvolt/core/network/api_exception.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../support/fakes.dart';

void main() {
  Future<AppSession> start(TestApp t) =>
      t.container.read(sessionControllerProvider.future);

  Session session(TestApp t) =>
      (t.container.read(sessionControllerProvider).requireValue as SignedIn)
          .session;

  SessionController controller(TestApp t) =>
      t.container.read(sessionControllerProvider.notifier);

  test('first launch waits for the age gate and signs nobody in', () async {
    final t = await TestApp.create(ageConfirmed: false);
    expect(await start(t), isA<AgeGateRequired>());
    expect(t.auth.calls, isEmpty);
    expect(t.sessions.calls, isEmpty);
  });

  test(
    'confirming the age gate creates a guest and syncs the session',
    () async {
      final t = await TestApp.create(ageConfirmed: false);
      await start(t);
      await controller(t).confirmAge();

      expect(t.prefs.getBool(ageConfirmedKey), isTrue);
      expect(t.auth.calls, ['signInAnonymously']);
      expect(t.sessions.calls, ['sync', 'registerDevice']);
      expect(session(t).tier, Tier.guest);
    },
  );

  test('a returning user is not signed in again', () async {
    final t = await TestApp.create(
      user: const AuthUser(uid: 'u1', isAnonymous: true),
    );
    expect(await start(t), isA<SignedIn>());
    expect(t.auth.calls, isEmpty);
    expect(session(t).user.id, 'user-u1');
  });

  test('a failed sync is an error state; invalidating retries', () async {
    final t = await TestApp.create();
    t.sessions.syncError = const ApiException(code: ApiException.networkCode);
    await expectLater(start(t), throwsA(isA<ApiException>()));
    expect(t.container.read(sessionControllerProvider).hasError, isTrue);

    t.container.invalidate(sessionControllerProvider);
    expect(await start(t), isA<SignedIn>());
  });

  test(
    'guest upgrade with Google keeps the user and refreshes the token',
    () async {
      final t = await TestApp.create();
      await start(t);
      t.auth.calls.clear();

      expect(await controller(t).upgrade(AuthMethod.google), isA<Upgraded>());
      expect(t.auth.calls, ['obtain:google', 'link:google', 'idToken:refresh']);
      expect(t.auth.currentUser!.uid, 'guest-1');
      expect(session(t).tier, Tier.full);
    },
  );

  test('email upgrade sends the verification email (D-017)', () async {
    final t = await TestApp.create();
    await start(t);
    await controller(t)
        .upgrade(AuthMethod.email, email: 'a@b.co', password: 'secret1');

    expect(t.auth.calls, contains('sendEmailVerification'));
    expect(session(t).tier, Tier.unverified);
    expect(session(t).flags.emailVerificationRequired, isTrue);

    expect(await controller(t).checkEmailVerified(), isFalse);
    t.auth.verifyOnReload = true;
    expect(await controller(t).checkEmailVerified(), isTrue);
    expect(session(t).tier, Tier.full);
  });

  group('link conflict (D-015)', () {
    const existing = AuthUser(
      uid: 'existing',
      isAnonymous: false,
      email: 'me@example.com',
      emailVerified: true,
      providers: {AuthMethod.google},
    );

    test('reports the conflict without changing anything', () async {
      final t = await TestApp.create();
      await start(t);
      t.auth.existing[AuthMethod.google] = existing;
      t.sessions.calls.clear();

      final result = await controller(t).upgrade(AuthMethod.google);
      expect(result, isA<UpgradeConflict>());
      expect(t.auth.currentUser!.uid, 'guest-1');
      expect(t.sessions.calls, isEmpty);
    });

    test(
      'switching signs in first, then abandons the guest with its own token',
      () async {
        final t = await TestApp.create();
        await start(t);
        t.auth.existing[AuthMethod.google] = existing;
        final conflict =
            await controller(t).upgrade(AuthMethod.google) as UpgradeConflict;
        t.auth.calls.clear();
        t.sessions.calls.clear();

        await controller(t).switchAccount(conflict.credential);

        expect(t.auth.calls, [
          'idToken:refresh',
          'signIn:google',
          'idToken:refresh',
        ]);
        expect(t.sessions.calls, [
          'abandonGuest:token-of-guest-1',
          'sync',
          'registerDevice',
        ]);
        expect(session(t).user.id, 'user-existing');
      },
    );

    test('a failed sign-in leaves the guest untouched', () async {
      final t = await TestApp.create();
      await start(t);
      t.auth.signInFailure = const AuthFailure(
        AuthFailureKind.invalidCredential,
      );
      final credential = await controller(t)
          .credentialFor(AuthMethod.email, email: 'a@b.co', password: 'wrong');

      await expectLater(
        controller(t).switchAccount(credential),
        throwsA(isA<AuthFailure>()),
      );
      expect(t.auth.currentUser!.uid, 'guest-1');
      expect(t.sessions.calls, isNot(contains(startsWith('abandonGuest'))));
    });
  });

  test(
    'sign-out unregisters the device and continues as a new guest',
    () async {
      final t = await TestApp.create(
        user: const AuthUser(
          uid: 'full',
          isAnonymous: false,
          emailVerified: true,
        ),
      );
      await start(t);
      t.auth.calls.clear();
      t.sessions.calls.clear();

      await controller(t).signOut();
      expect(t.sessions.calls.first, 'unregisterDevice');
      expect(t.auth.calls, ['signOut', 'signInAnonymously']);
      expect(session(t).tier, Tier.guest);
    },
  );

  test('deletion request and cancel resync the session', () async {
    final t = await TestApp.create();
    await start(t);
    final purge = await controller(t).requestDeletion();
    expect(purge, DateTime.utc(2026, 11, 1));
    expect(session(t).flags.deletionPending, isTrue);

    await controller(t).cancelDeletion();
    expect(session(t).flags.deletionPending, isFalse);
  });
}
