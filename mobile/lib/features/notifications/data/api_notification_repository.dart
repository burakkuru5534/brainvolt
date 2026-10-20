import '../../../core/network/api_client.dart';
import '../domain/notification_repository.dart';

class ApiNotificationRepository implements NotificationRepository {
  const ApiNotificationRepository(this.api);

  final ApiClient api;

  @override
  Future<InboxPage> inbox({String? cursor}) async {
    final j = await api.getJson(
      Uri(
        path: '/v1/me/inbox',
        queryParameters: {'cursor': ?cursor},
      ).toString(),
    );
    return InboxPage(
      items: [
        for (final i in (j['items'] as List? ?? const []))
          InboxItem.fromJson((i as Map).cast<String, dynamic>()),
      ],
      unread: (j['unread'] as num?)?.toInt() ?? 0,
      next: j['next_cursor'] as String?,
    );
  }

  @override
  Future<void> markRead({List<String> ids = const [], bool all = false}) =>
      api.postJson(
        '/v1/me/inbox/read',
        body: {if (all) 'all': true else 'ids': ids},
      );

  NotificationPrefs _prefs(Map<String, dynamic> j) {
    final q = (j['quiet_hours'] as Map?)?.cast<String, dynamic>() ?? const {};
    return NotificationPrefs(
      categories: [
        for (final c in (j['categories'] as List? ?? const []))
          CategoryPref(
            category: (c as Map)['category'] as String,
            push: c['push'] == true,
            inApp: c['in_app'] == true,
            locked: c['locked'] == true,
          ),
      ],
      quiet: QuietHours(
        enabled: q['enabled'] == true,
        start: q['start'] as String? ?? '23:00',
        end: q['end'] as String? ?? '08:00',
      ),
    );
  }

  @override
  Future<NotificationPrefs> preferences() async =>
      _prefs(await api.getJson('/v1/me/notification-preferences'));

  @override
  Future<NotificationPrefs> update({
    List<CategoryPref>? categories,
    QuietHours? quiet,
  }) async => _prefs(
    await api.patchJson(
      '/v1/me/notification-preferences',
      body: {
        if (categories != null)
          'categories': [for (final c in categories) c.toJson()],
        if (quiet != null)
          'quiet_hours': {
            'enabled': quiet.enabled,
            'start': quiet.start,
            'end': quiet.end,
            'utc_offset_minutes': DateTime.now().timeZoneOffset.inMinutes,
          },
      },
    ),
  );
}
