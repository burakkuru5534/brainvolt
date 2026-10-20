/// The in-app inbox and notification settings (schema-platform §1, D-086).
library;

/// An inbox entry, already in the app's language.
class InboxItem {
  const InboxItem({
    required this.id,
    required this.category,
    required this.kind,
    required this.title,
    required this.body,
    required this.createdAt,
    required this.read,
    this.deepLink,
  });

  factory InboxItem.fromJson(Map<String, dynamic> j) => InboxItem(
    id: j['id'] as String,
    category: j['category'] as String? ?? '',
    kind: j['kind'] as String? ?? '',
    title: j['title'] as String? ?? '',
    body: j['body'] as String? ?? '',
    deepLink: j['deep_link'] as String?,
    createdAt: DateTime.parse(j['created_at'] as String).toLocal(),
    read: j['read'] == true,
  );

  final String id;
  final String category;
  final String kind;
  final String title;
  final String body;

  /// The app route to open, e.g. `/friends`.
  final String? deepLink;
  final DateTime createdAt;
  final bool read;
}

/// A page of the inbox with the unread count.
class InboxPage {
  const InboxPage({required this.items, required this.unread, this.next});

  final List<InboxItem> items;
  final int unread;
  final String? next;
}

/// A category's channels.
class CategoryPref {
  const CategoryPref({
    required this.category,
    required this.push,
    required this.inApp,
    this.locked = false,
  });

  final String category;
  final bool push;
  final bool inApp;

  /// Account notices cannot be turned off.
  final bool locked;

  Map<String, dynamic> toJson() => {
    'category': category,
    'push': push,
    'in_app': inApp,
  };
}

/// Quiet hours in local "HH:MM".
class QuietHours {
  const QuietHours({
    required this.enabled,
    required this.start,
    required this.end,
  });

  final bool enabled;
  final String start;
  final String end;
}

class NotificationPrefs {
  const NotificationPrefs({required this.categories, required this.quiet});

  final List<CategoryPref> categories;
  final QuietHours quiet;
}

abstract interface class NotificationRepository {
  Future<InboxPage> inbox({String? cursor});

  /// Marks [ids] read, or everything with [all].
  Future<void> markRead({List<String> ids = const [], bool all = false});

  Future<NotificationPrefs> preferences();

  /// Saves categories and/or quiet hours; quiet hours carry the device's
  /// current UTC offset.
  Future<NotificationPrefs> update({
    List<CategoryPref>? categories,
    QuietHours? quiet,
  });
}
