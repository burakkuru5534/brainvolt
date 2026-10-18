import 'package:brainvolt/app/config.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('dev config from env/dev.json values', () {
    final c = AppConfig.parse(
      flavor: 'dev',
      apiBaseUrl: 'http://10.0.2.2:8080',
      wsUrl: 'ws://10.0.2.2:8080/v1/ws',
    );
    expect(c.flavor, Flavor.dev);
    expect(c.apiBaseUrl.port, 8080);
  });

  test('prod requires https and wss', () {
    expect(
      () => AppConfig.parse(
        flavor: 'prod',
        apiBaseUrl: 'http://api.brainvolt.app',
        wsUrl: 'wss://api.brainvolt.app/v1/ws',
      ),
      throwsA(
        isA<StateError>().having(
          (e) => e.message,
          'message',
          contains('https and wss'),
        ),
      ),
    );
    expect(
      AppConfig.parse(
        flavor: 'prod',
        apiBaseUrl: 'https://api.brainvolt.app',
        wsUrl: 'wss://api.brainvolt.app/v1/ws',
      ).flavor,
      Flavor.prod,
    );
  });

  test('missing --dart-define-from-file lists every problem', () {
    expect(
      () => AppConfig.parse(flavor: '', apiBaseUrl: '', wsUrl: ''),
      throwsA(
        isA<StateError>().having(
          (e) => e.message,
          'message',
          allOf(
            contains('--dart-define-from-file=env/dev.json'),
            contains('FLAVOR'),
            contains('API_BASE_URL'),
            contains('WS_URL'),
          ),
        ),
      ),
    );
  });

  test('AUTH_EMULATOR_HOST points dev at the Auth emulator', () {
    final c = AppConfig.parse(
      flavor: 'dev',
      apiBaseUrl: 'http://10.0.2.2:8080',
      wsUrl: 'ws://10.0.2.2:8080/v1/ws',
      authEmulatorHost: '10.0.2.2:9099',
    );
    expect(c.authEmulator?.host, '10.0.2.2');
    expect(c.authEmulator?.port, 9099);
    expect(
      AppConfig.parse(
        flavor: 'dev',
        apiBaseUrl: 'http://10.0.2.2:8080',
        wsUrl: 'ws://10.0.2.2:8080/v1/ws',
      ).authEmulator,
      isNull,
    );
  });

  test('a malformed emulator host, or one in prod, is rejected', () {
    Matcher fails(String text) => throwsA(
      isA<StateError>().having((e) => e.message, 'message', contains(text)),
    );
    expect(
      () => AppConfig.parse(
        flavor: 'dev',
        apiBaseUrl: 'http://10.0.2.2:8080',
        wsUrl: 'ws://10.0.2.2:8080/v1/ws',
        authEmulatorHost: '10.0.2.2',
      ),
      fails('host:port'),
    );
    expect(
      () => AppConfig.parse(
        flavor: 'prod',
        apiBaseUrl: 'https://api.brainvolt.app',
        wsUrl: 'wss://api.brainvolt.app/v1/ws',
        authEmulatorHost: 'localhost:9099',
      ),
      fails('must not use the Firebase Auth emulator'),
    );
  });
}
