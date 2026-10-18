import 'package:brainvolt/core/network/api_client.dart';
import 'package:brainvolt/core/network/token_source.dart';
import 'package:brainvolt/features/auth/data/api_session_repository.dart';
import 'package:brainvolt/features/auth/data/firebase_auth_repository.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../core/network/fake_adapter.dart';

class _Tokens implements TokenSource {
  final calls = <bool>[];

  @override
  Future<String?> idToken({bool forceRefresh = false}) async {
    calls.add(forceRefresh);
    return 'current-user-token';
  }
}

const _sessionJson = {
  'user': {
    'id': '0192c7e0-0000-7000-8000-000000000001',
    'created_at': '2026-10-18T10:00:00Z',
    'deletion_requested_at': null,
    'roles': <String>[],
    'status': 'active',
    'tier': 'guest',
    'upgraded_at': null,
  },
  'profile': {
    'username': 'oyuncu12345',
    'display_name': 'Misafir 1234',
    'avatar_id': 'owl',
    'locale': 'tr',
    'country_code': 'TR',
  },
  'tier': 'guest',
  'status': 'active',
  'flags': {
    'created': true,
    'restricted': true,
    'email_verification_required': false,
    'deletion_pending': false,
    'suspended': false,
  },
};

void main() {
  late FakeAdapter adapter;
  late _Tokens tokens;

  ApiSessionRepository repo({String? country = 'tr'}) => ApiSessionRepository(
    api: ApiClient(
      buildDio(
        baseUrl: Uri.parse('http://api.test'),
        appVersion: '1.0.0+1',
        platform: 'android',
        locale: () => 'tr',
        tokens: tokens,
        adapter: adapter,
      ),
    ),
    installationId: '0192c7e0-0000-7000-8000-0000000000aa',
    platform: 'android',
    appVersion: '1.0.0+1',
    locale: () => 'en',
    countryHint: () => country,
  );

  Map<String, dynamic> body(int i) =>
      Map<String, dynamic>.from(adapter.requests[i].data as Map);

  setUp(() {
    tokens = _Tokens();
    adapter = FakeAdapter((o) {
      if (o.path == '/v1/session') {
        return const FakeResponse(200, _sessionJson);
      }
      if (o.path == '/v1/me/deletion' && o.method == 'POST') {
        return const FakeResponse(200, {
          'requested_at': '2026-10-18T10:00:00Z',
          'purge_after': '2026-11-01T10:00:00Z',
        });
      }
      return const FakeResponse(204);
    });
  });

  test(
    'session sync sends the age confirmation and parses the answer',
    () async {
      final s = await repo().sync();
      expect(adapter.requests.single.method, 'POST');
      expect(body(0), {
        'age_confirmed': true,
        'installation_id': '0192c7e0-0000-7000-8000-0000000000aa',
        'platform': 'android',
        'app_version': '1.0.0+1',
        'locale': 'en',
        'country_hint': 'TR',
      });
      expect(s.tier, Tier.guest);
      expect(s.profile.displayName, 'Misafir 1234');
      expect(s.profile.avatarId, 'owl');
      expect(s.flags.created, isTrue);
    },
  );

  test('regions that are not countries are left out', () async {
    await repo(country: '419').sync();
    expect(body(0).containsKey('country_hint'), isFalse);
    await repo(country: null).sync();
    expect(body(1).containsKey('country_hint'), isFalse);
  });

  test('device registration and sign-out', () async {
    await repo().registerDevice();
    expect(adapter.requests[0].path, '/v1/me/devices');
    expect(body(0), {
      'installation_id': '0192c7e0-0000-7000-8000-0000000000aa',
      'platform': 'android',
      'app_version': '1.0.0+1',
      'locale': 'en',
      'fcm_token': null,
    });
    await repo().unregisterDevice();
    expect(adapter.requests[1].method, 'DELETE');
    expect(
      adapter.requests[1].path,
      '/v1/me/devices/0192c7e0-0000-7000-8000-0000000000aa',
    );
  });

  test('abandon-guest uses the guest token and an idempotency key', () async {
    await repo().abandonGuest(guestToken: 'guest-token');
    final r = adapter.requests.single;
    expect(r.path, '/v1/session/abandon-guest');
    expect(r.headers[ApiHeaders.authorization], 'Bearer guest-token');
    expect(r.headers[ApiHeaders.idempotencyKey], isA<String>());
    expect(tokens.calls, isEmpty, reason: 'current user token not used');
  });

  test('an explicit token is never refreshed on TOKEN_EXPIRED', () async {
    adapter = FakeAdapter((_) => FakeResponse(401, envelope('TOKEN_EXPIRED')));
    await expectLater(
      repo().abandonGuest(guestToken: 'old'),
      throwsA(anything),
    );
    expect(adapter.requests, hasLength(1));
    expect(tokens.calls, isEmpty);
  });

  test('deletion returns the purge date', () async {
    expect(await repo().requestDeletion(), DateTime.utc(2026, 11, 1, 10));
    expect(
      adapter.requests.single.headers[ApiHeaders.idempotencyKey],
      isNotNull,
    );
    await repo().cancelDeletion();
    expect(adapter.requests[1].method, 'DELETE');
  });

  test('Firebase error codes map to failure kinds', () {
    expect(
      firebaseFailureKind('credential-already-in-use'),
      AuthFailureKind.credentialAlreadyInUse,
    );
    expect(
      firebaseFailureKind('email-already-in-use'),
      AuthFailureKind.credentialAlreadyInUse,
    );
    expect(
      firebaseFailureKind('wrong-password'),
      AuthFailureKind.invalidCredential,
    );
    expect(firebaseFailureKind('weak-password'), AuthFailureKind.weakPassword);
    expect(
      firebaseFailureKind('network-request-failed'),
      AuthFailureKind.network,
    );
    expect(firebaseFailureKind('something-new'), AuthFailureKind.unknown);
  });
}
