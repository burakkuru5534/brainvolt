import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../app/providers.dart';
import '../l10n/locale_controller.dart';
import '../network/api_client.dart';
import '../storage/installation_id.dart';
import 'realtime_client.dart';

part 'realtime_providers.g.dart';

/// The app's WebSocket. Features call `start()` when they need it; it stays
/// connected (with reconnects) from then on.
@Riverpod(keepAlive: true)
RealtimeClient realtimeClient(Ref ref) {
  final client = RealtimeClient(
    url: ref.watch(appConfigProvider).wsUrl,
    tokens: ref.watch(tokenSourceProvider),
    installationId: ref.watch(installationIdProvider),
    headers: {
      ApiHeaders.appVersion: ref.watch(appVersionProvider),
      ApiHeaders.platform: ref.watch(devicePlatformProvider),
      ApiHeaders.acceptLanguage: effectiveLocale(
        ref.read(localeOverrideProvider),
      ).languageCode,
    },
  );
  ref.onDispose(client.dispose);
  return client;
}
