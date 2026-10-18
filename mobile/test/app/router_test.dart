import 'package:brainvolt/app/router.dart';
import 'package:brainvolt/app/session_controller.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../support/fakes.dart';

void main() {
  AsyncValue<AppSession> signedIn(Session s) => AsyncData(SignedIn(s));

  test('starting or failed sessions show the splash screen', () {
    const loading = AsyncLoading<AppSession>();
    expect(sessionRedirect(loading, Routes.home), Routes.splash);
    expect(sessionRedirect(loading, Routes.splash), isNull);
    final failed = AsyncError<AppSession>(Exception('x'), StackTrace.empty);
    expect(sessionRedirect(failed, Routes.settings), Routes.splash);
  });

  test('the age gate comes first (D-020)', () {
    const gate = AsyncData<AppSession>(AgeGateRequired());
    expect(sessionRedirect(gate, Routes.splash), Routes.ageGate);
    expect(sessionRedirect(gate, Routes.home), Routes.ageGate);
    expect(sessionRedirect(gate, Routes.ageGate), isNull);
  });

  test('signed-in users leave the gates and may go anywhere else', () {
    final s = signedIn(testSession());
    expect(sessionRedirect(s, Routes.splash), Routes.home);
    expect(sessionRedirect(s, Routes.ageGate), Routes.home);
    expect(sessionRedirect(s, Routes.accountStatus), Routes.home);
    expect(sessionRedirect(s, Routes.settings), isNull);
    expect(sessionRedirect(s, Routes.home), isNull);
  });

  test(
    'suspended or deletion-pending accounts are held on the status screen',
    () {
      for (final s in [
        testSession(status: AccountStatus.suspended),
        testSession(tier: Tier.full, deletionRequestedAt: DateTime.utc(2026)),
      ]) {
        expect(sessionRedirect(signedIn(s), Routes.home), Routes.accountStatus);
        expect(sessionRedirect(signedIn(s), Routes.accountStatus), isNull);
      }
    },
  );
}
