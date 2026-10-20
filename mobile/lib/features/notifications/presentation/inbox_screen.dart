import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/notification_providers.dart';
import '../domain/notification_repository.dart';

/// The first page of the inbox (unread count for the bell).
final inboxProvider = FutureProvider.autoDispose<InboxPage>(
  (ref) => ref.watch(notificationRepositoryProvider).inbox(),
);

/// `inbox.new` from the server: the inbox changed.
final inboxEventsProvider = StreamProvider.autoDispose<ServerFrame>((ref) {
  final rt = ref.watch(realtimeClientProvider)..start();
  return rt.frames.where((f) => f.type == 'inbox.new');
});

/// A category's name in settings.
String notificationCategoryName(AppLocalizations l10n, String c) => switch (c) {
  'challenge' => l10n.notifCatChallenge,
  'friend' => l10n.notifCatFriend,
  'match_found' => l10n.notifCatMatchFound,
  'team' => l10n.notifCatTeam,
  'chat' => l10n.notifCatChat,
  'season' => l10n.notifCatSeason,
  'quest' => l10n.notifCatQuest,
  'tournament' => l10n.notifCatTournament,
  'moderation' => l10n.notifCatModeration,
  _ => l10n.notifCatProductNews,
};

IconData _icon(String category) => switch (category) {
  'challenge' => Icons.sports_kabaddi,
  'friend' => Icons.person_add_alt_1,
  'team' => Icons.groups,
  'chat' => Icons.chat_bubble_rounded,
  'moderation' => Icons.gavel,
  'quest' => Icons.flag,
  'season' => Icons.emoji_events,
  _ => Icons.notifications,
};

/// Opens an inbox item's route; matches replace the stack.
void openRoute(BuildContext context, String route) {
  if (route.startsWith('/match/')) {
    context.go(route);
  } else {
    unawaited(context.push(route));
  }
}

/// The in-app notifications, newest first (D-086).
class InboxScreen extends ConsumerStatefulWidget {
  const InboxScreen({super.key});

  @override
  ConsumerState<InboxScreen> createState() => _InboxScreenState();
}

class _InboxScreenState extends ConsumerState<InboxScreen> {
  @override
  void initState() {
    super.initState();
    // The bell keeps the first page alive; opening the inbox reloads it.
    unawaited(Future.microtask(() => ref.invalidate(inboxProvider)));
  }

  Future<void> _markAll(BuildContext context, WidgetRef ref) async {
    final messenger = ScaffoldMessenger.of(context);
    final l10n = context.l10n;
    try {
      await ref.read(notificationRepositoryProvider).markRead(all: true);
      ref.invalidate(inboxProvider);
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    ref.listen(inboxEventsProvider, (_, _) => ref.invalidate(inboxProvider));
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.inboxTitle),
        actions: [
          IconButton(
            key: const Key('inbox-mark-all'),
            tooltip: l10n.inboxMarkAll,
            icon: const Icon(Icons.done_all),
            onPressed: () => _markAll(context, ref),
          ),
          IconButton(
            key: const Key('open-notification-settings'),
            tooltip: l10n.notificationSettingsTitle,
            icon: const Icon(Icons.tune),
            onPressed: () => context.push(Routes.notificationSettings),
          ),
        ],
      ),
      body: switch (ref.watch(inboxProvider)) {
        AsyncData(:final value) when value.items.isEmpty => EmptyState(
          icon: Icons.notifications_none,
          text: l10n.inboxEmpty,
        ),
        AsyncData(:final value) => RefreshIndicator(
          onRefresh: () => ref.refresh(inboxProvider.future),
          child: ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: value.items.length,
            separatorBuilder: (_, _) => const SizedBox(height: 10),
            itemBuilder: (context, i) => _Item(value.items[i]),
          ),
        ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _Item extends ConsumerWidget {
  const _Item(this.item);

  final InboxItem item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final loc = MaterialLocalizations.of(context);
    return BvCard(
      key: Key('inbox-${item.id}'),
      padding: const EdgeInsets.all(14),
      onTap: () async {
        if (!item.read) {
          unawaited(
            ref
                .read(notificationRepositoryProvider)
                .markRead(ids: [item.id])
                .then((_) => ref.invalidate(inboxProvider))
                .catchError((Object _) {}),
          );
        }
        if (item.deepLink case final route?) {
          openRoute(context, route);
        }
      },
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          IconBadge(_icon(item.category)),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(item.title, style: text.titleMedium),
                Text(item.body),
                const SizedBox(height: 4),
                Text(
                  '${loc.formatShortDate(item.createdAt)} '
                  '${loc.formatTimeOfDay(TimeOfDay.fromDateTime(item.createdAt))}',
                  style: text.bodySmall,
                ),
              ],
            ),
          ),
          if (!item.read)
            Container(
              key: Key('inbox-unread-${item.id}'),
              width: 10,
              height: 10,
              margin: const EdgeInsets.only(top: 6),
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: bv.gradientStart,
              ),
            ),
        ],
      ),
    );
  }
}
