/// Build-time configuration (roadmap 1.7.4, ADR-0003): one app id
/// (`com.brainvolt.app`), two flavors chosen with
/// `--dart-define-from-file=env/<flavor>.json`.
library;

/// The deployment the app talks to.
enum Flavor {
  /// Local backend + Firebase Auth emulator.
  dev,

  /// Deployed servers + the real Firebase project (`brainvolt-a9caa`).
  prod,
}

/// Values from `env/<flavor>.json`.
class AppConfig {
  const AppConfig({
    required this.flavor,
    required this.apiBaseUrl,
    required this.wsUrl,
    this.authEmulator,
  });

  /// Reads the `--dart-define-from-file` values and fails fast with a
  /// readable message when the app was started without them.
  factory AppConfig.fromEnvironment() => AppConfig.parse(
    flavor: const String.fromEnvironment('FLAVOR'),
    apiBaseUrl: const String.fromEnvironment('API_BASE_URL'),
    wsUrl: const String.fromEnvironment('WS_URL'),
    authEmulatorHost: const String.fromEnvironment('AUTH_EMULATOR_HOST'),
  );

  /// Validates raw values (separate from [AppConfig.fromEnvironment] so it
  /// can be tested).
  factory AppConfig.parse({
    required String flavor,
    required String apiBaseUrl,
    required String wsUrl,
    String authEmulatorHost = '',
  }) {
    final problems = <String>[];
    final parsedFlavor = Flavor.values
        .where((f) => f.name == flavor)
        .firstOrNull;
    if (parsedFlavor == null) {
      problems.add('FLAVOR must be dev or prod, got "$flavor"');
    }
    final api = Uri.tryParse(apiBaseUrl);
    if (api == null ||
        !api.hasAuthority ||
        !{'http', 'https'}.contains(api.scheme)) {
      problems.add('API_BASE_URL must be an http(s) URL, got "$apiBaseUrl"');
    }
    final ws = Uri.tryParse(wsUrl);
    if (ws == null || !ws.hasAuthority || !{'ws', 'wss'}.contains(ws.scheme)) {
      problems.add('WS_URL must be a ws(s) URL, got "$wsUrl"');
    }
    if (parsedFlavor == Flavor.prod &&
        (api?.scheme != 'https' || ws?.scheme != 'wss')) {
      problems.add('prod must use https and wss (conventions.md §1)');
    }
    AuthEmulator? emulator;
    if (authEmulatorHost.isNotEmpty) {
      emulator = AuthEmulator.tryParse(authEmulatorHost);
      if (emulator == null) {
        problems.add(
          'AUTH_EMULATOR_HOST must be host:port, got "$authEmulatorHost"',
        );
      } else if (parsedFlavor == Flavor.prod) {
        problems.add('prod must not use the Firebase Auth emulator');
      }
    }
    if (problems.isNotEmpty) {
      throw StateError(
        'Invalid app configuration — start the app with '
        '--dart-define-from-file=env/dev.json (or env/prod.json):\n'
        '${problems.join('\n')}',
      );
    }
    return AppConfig(
      flavor: parsedFlavor!,
      apiBaseUrl: api!,
      wsUrl: ws!,
      authEmulator: emulator,
    );
  }

  final Flavor flavor;
  final Uri apiBaseUrl;
  final Uri wsUrl;

  /// The Firebase Auth emulator the app signs in against (dev only); null
  /// means the real Firebase project.
  final AuthEmulator? authEmulator;
}

/// Address of the Firebase Auth emulator (`AUTH_EMULATOR_HOST`).
class AuthEmulator {
  const AuthEmulator(this.host, this.port);

  /// Parses `host:port`; null when malformed.
  static AuthEmulator? tryParse(String value) {
    final i = value.lastIndexOf(':');
    if (i <= 0) {
      return null;
    }
    final port = int.tryParse(value.substring(i + 1));
    if (port == null || port <= 0 || port > 65535) {
      return null;
    }
    return AuthEmulator(value.substring(0, i), port);
  }

  final String host;
  final int port;
}
