import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/notification_repository.dart';
import 'api_notification_repository.dart';

part 'notification_providers.g.dart';

/// The inbox and notification settings endpoints. Tests override it.
@Riverpod(keepAlive: true)
NotificationRepository notificationRepository(Ref ref) =>
    ApiNotificationRepository(ref.watch(apiClientProvider));
