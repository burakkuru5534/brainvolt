import 'package:brainvolt/core/l10n/generated/app_localizations_en.dart';
import 'package:brainvolt/core/l10n/generated/app_localizations_tr.dart';
import 'package:brainvolt/core/network/api_client.dart';
import 'package:brainvolt/core/network/api_exception.dart';
import 'package:brainvolt/core/network/token_source.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fake_adapter.dart';

/// Hands out numbered tokens and counts forced refreshes.
class CountingTokens implements TokenSource {
  int issued = 0;
  int refreshes = 0;

  @override
  Future<String?> idToken({bool forceRefresh = false}) async {
    if (forceRefresh) {
      refreshes++;
    }
    return 'token-${++issued}';
  }
}

ApiClient client(
  FakeAdapter adapter, {
  TokenSource? tokens,
  String locale = 'tr',
}) => ApiClient(
  buildDio(
    baseUrl: Uri.parse('http://api.test'),
    appVersion: '1.2.3+45',
    platform: 'android',
    locale: () => locale,
    tokens: tokens ?? CountingTokens(),
    adapter: adapter,
  ),
);

void main() {
  test('sends the standard headers and the Bearer token', () async {
    final adapter = FakeAdapter((_) => const FakeResponse(200, {'ok': true}));
    final body = await client(adapter, locale: 'en').getJson('/v1/me');

    expect(body, {'ok': true});
    final h = adapter.requests.single.headers;
    expect(adapter.requests.single.uri.toString(), 'http://api.test/v1/me');
    expect(h[ApiHeaders.acceptLanguage], 'en');
    expect(h[ApiHeaders.appVersion], '1.2.3+45');
    expect(h[ApiHeaders.platform], 'android');
    expect(h[ApiHeaders.authorization], 'Bearer token-1');
  });

  test('signed out: no Authorization header', () async {
    final adapter = FakeAdapter((_) => const FakeResponse(200, {}));
    await client(
      adapter,
      tokens: const SignedOutTokenSource(),
    ).getJson('/healthz');
    expect(
      adapter.requests.single.headers.containsKey(ApiHeaders.authorization),
      isFalse,
    );
  });

  test('TOKEN_EXPIRED: refreshes the token once and retries', () async {
    final tokens = CountingTokens();
    final adapter = FakeAdapter(
      (o) => o.headers[ApiHeaders.authorization] == 'Bearer token-1'
          ? FakeResponse(401, envelope('TOKEN_EXPIRED'))
          : const FakeResponse(200, {'ok': true}),
    );
    expect(await client(adapter, tokens: tokens).getJson('/v1/me'), {
      'ok': true,
    });
    expect(tokens.refreshes, 1);
    expect(adapter.requests.map((r) => r.headers[ApiHeaders.authorization]), [
      'Bearer token-1',
      'Bearer token-2',
    ]);
  });

  test('TOKEN_EXPIRED twice: gives up after one retry', () async {
    final tokens = CountingTokens();
    final adapter = FakeAdapter(
      (_) => FakeResponse(401, envelope('TOKEN_EXPIRED')),
    );
    await expectLater(
      client(adapter, tokens: tokens).getJson('/v1/me'),
      throwsA(
        isA<ApiException>().having((e) => e.code, 'code', 'TOKEN_EXPIRED'),
      ),
    );
    expect(adapter.requests, hasLength(2));
  });

  test('other 401s are not retried', () async {
    final adapter = FakeAdapter(
      (_) => FakeResponse(401, envelope('UNAUTHENTICATED')),
    );
    await expectLater(
      client(adapter).getJson('/v1/me'),
      throwsA(isA<ApiException>()),
    );
    expect(adapter.requests, hasLength(1));
  });

  test(
    'error envelope becomes ApiException with details and Retry-After',
    () async {
      final adapter = FakeAdapter(
        (_) => FakeResponse(
          429,
          envelope('RATE_LIMITED', details: {'retry_after_s': 7}),
          {
            'retry-after': ['7'],
          },
        ),
      );
      try {
        await client(adapter)
            .postJson('/v1/session', body: {'age_confirmed': true});
        fail('expected ApiException');
      } on ApiException catch (e) {
        expect(e.code, 'RATE_LIMITED');
        expect(e.status, 429);
        expect(e.details['retry_after_s'], 7);
        expect(e.retryAfter, const Duration(seconds: 7));
        expect(e.requestId, '0192c7e0-0000-7000-8000-000000000001');
        expect(e.serverMessage, 'server text');
      }
    },
  );

  test('Idempotency-Key is sent when given; 204 gives null', () async {
    final adapter = FakeAdapter((_) => const FakeResponse(204));
    expect(
      await client(adapter)
          .postJson('/v1/session/abandon-guest', idempotencyKey: 'k-1'),
      isNull,
    );
    expect(adapter.requests.single.headers[ApiHeaders.idempotencyKey], 'k-1');

    await client(adapter).postJson('/v1/me/devices', body: {});
    expect(
      adapter.requests.last.headers.containsKey(ApiHeaders.idempotencyKey),
      isFalse,
    );
  });

  test('network failures map to client-side codes', () async {
    for (final (type, code) in [
      (DioExceptionType.connectionTimeout, ApiException.timeoutCode),
      (DioExceptionType.receiveTimeout, ApiException.timeoutCode),
      (DioExceptionType.connectionError, ApiException.networkCode),
    ]) {
      final adapter = FakeAdapter(
        (o) => throw DioException(requestOptions: o, type: type),
      );
      await expectLater(
        client(adapter).getJson('/v1/me'),
        throwsA(isA<ApiException>().having((e) => e.code, 'code', code)),
        reason: '$type',
      );
    }
    final garbage = FakeAdapter(
      (_) => const FakeResponse(502, {'not': 'an envelope'}),
    );
    await expectLater(
      client(garbage).getJson('/v1/me'),
      throwsA(
        isA<ApiException>()
            .having((e) => e.code, 'code', ApiException.unknownCode)
            .having((e) => e.status, 'status', 502),
      ),
    );
  });

  test(
    'localized messages: app translation, then server text, then generic',
    () {
      final tr = AppLocalizationsTr();
      final en = AppLocalizationsEn();
      const upgrade = ApiException(
        code: 'ACCOUNT_UPGRADE_REQUIRED',
        serverMessage: 'ignored',
      );
      expect(
        upgrade.localizedMessage(tr),
        'Bu işlem için hesabını yükseltmelisin.',
      );
      expect(upgrade.localizedMessage(en), 'Upgrade your account to do this.');
      expect(
        const ApiException(code: ApiException.networkCode).localizedMessage(en),
        en.errorNetwork,
      );
      expect(
        const ApiException(
          code: 'BRAND_NEW_CODE',
          serverMessage: 'Yeni',
        ).localizedMessage(tr),
        'Yeni',
      );
      expect(
        const ApiException(code: 'BRAND_NEW_CODE').localizedMessage(en),
        en.errorUnknown,
      );
    },
  );
}
