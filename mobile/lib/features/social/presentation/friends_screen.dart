import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../lobby/data/lobby_providers.dart';
import '../../lobby/domain/lobby_repository.dart';
import '../../lobby/presentation/challenge_dialog.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../data/social_providers.dart';
import '../domain/social_repository.dart';

/// The caller's friends with presence.
final friendsProvider = FutureProvider.autoDispose<List<Friend>>(
  (ref) => ref.watch(socialRepositoryProvider).friends(),
);

/// Pending friend requests: incoming first, then sent ones.
final friendRequestsProvider = FutureProvider.autoDispose<List<FriendRequest>>((
  ref,
) async {
  final repo = ref.watch(socialRepositoryProvider);
  return [
    ...await repo.requests(incoming: true),
    ...await repo.requests(incoming: false),
  ];
});

/// The caller's friend invite code.
final friendCodeProvider = FutureProvider.autoDispose<FriendCode>(
  (ref) => ref.watch(lobbyRepositoryProvider).friendCode(),
);

bool _needsUpgrade(Object? e) =>
    e is ApiException && e.code == 'ACCOUNT_UPGRADE_REQUIRED';

/// Friends (with presence, challenge, unfriend, block), friend requests
/// and adding friends by username or friend code (D-016: full accounts).
class FriendsScreen extends ConsumerWidget {
  const FriendsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final friends = ref.watch(friendsProvider);
    if (_needsUpgrade(friends.error)) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.homeFriends)),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const IconBadge(Icons.people_alt_rounded, size: 88),
                const SizedBox(height: 16),
                Text(
                  l10n.friendsUpgrade,
                  key: const Key('friends-upgrade'),
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => context.push(Routes.account),
                  child: Text(l10n.linkAccountAction),
                ),
              ],
            ),
          ),
        ),
      );
    }
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: Text(l10n.homeFriends),
          actions: [
            IconButton(
              key: const Key('add-friend'),
              tooltip: l10n.addFriend,
              icon: const Icon(Icons.person_add),
              onPressed: () => unawaited(_addFriend(context, ref)),
            ),
          ],
          bottom: TabBar(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
            tabs: [
              Tab(text: l10n.friendsTab),
              Tab(text: l10n.requestsTab),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            _FriendsTab(friends: friends),
            const _RequestsTab(),
          ],
        ),
      ),
    );
  }
}

/// Runs [action], refreshes the lists and reports errors in a snack bar;
/// [done] is the success text (computed after the action).
Future<void> _run(
  BuildContext context,
  WidgetRef ref,
  Future<void> Function() action, {
  String Function()? done,
}) async {
  final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
  final l10n = context.l10n;
  try {
    await action();
    if (done != null) {
      messenger.showSnackBar(SnackBar(content: Text(done())));
    }
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
  ref
    ..invalidate(friendsProvider)
    ..invalidate(friendRequestsProvider);
}

Future<void> _addFriend(BuildContext context, WidgetRef ref) async {
  final l10n = context.l10n;
  final input = await showDialog<(bool, String)>(
    context: context,
    builder: (_) => const _AddFriendDialog(),
  );
  if (input == null || !context.mounted) {
    return;
  }
  final (byCode, value) = input;
  final social = ref.read(socialRepositoryProvider);
  final lobby = ref.read(lobbyRepositoryProvider);
  var accepted = false;
  await _run(context, ref, () async {
    if (byCode) {
      final code = await lobby.resolveCode(value);
      accepted = await social.sendRequest(
        userId: code.user?.userId,
        via: 'invite_code',
      );
    } else {
      accepted = await social.sendRequest(username: value);
    }
  }, done: () => accepted ? l10n.friendAdded : l10n.friendRequestSent);
}

class _AddFriendDialog extends StatefulWidget {
  const _AddFriendDialog();

  @override
  State<_AddFriendDialog> createState() => _AddFriendDialogState();
}

class _AddFriendDialogState extends State<_AddFriendDialog> {
  final _text = TextEditingController();
  bool _byCode = false;

  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AlertDialog(
      title: Text(l10n.addFriend),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          SegmentedButton<bool>(
            segments: [
              ButtonSegment(value: false, label: Text(l10n.addByUsername)),
              ButtonSegment(value: true, label: Text(l10n.addByCode)),
            ],
            selected: {_byCode},
            onSelectionChanged: (s) => setState(() => _byCode = s.first),
          ),
          const SizedBox(height: 12),
          TextField(
            key: const Key('add-friend-input'),
            controller: _text,
            autofocus: true,
            textCapitalization: _byCode
                ? TextCapitalization.characters
                : TextCapitalization.none,
            decoration: InputDecoration(
              labelText: _byCode ? l10n.addByCode : l10n.addByUsername,
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('add-friend-send'),
          onPressed: () {
            final v = _text.text.trim();
            if (v.isNotEmpty) {
              Navigator.pop(context, (_byCode, v));
            }
          },
          child: Text(l10n.send),
        ),
      ],
    );
  }
}

class _FriendsTab extends ConsumerWidget {
  const _FriendsTab({required this.friends});

  final AsyncValue<List<Friend>> friends;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final code = ref.watch(friendCodeProvider).value;
    return RefreshIndicator(
      onRefresh: () => ref.refresh(friendsProvider.future),
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          if (code != null)
            TileCard(
              highlight: BvColors.of(context).gradientStart,
              child: ListTile(
                key: const Key('friend-code'),
                leading: const IconBadge(Icons.qr_code_2_rounded, size: 40),
                title: Text(l10n.friendCode(code.code)),
                trailing: IconButton(
                  icon: const Icon(Icons.copy_rounded),
                  onPressed: () async {
                    await Clipboard.setData(ClipboardData(text: code.code));
                    if (context.mounted) {
                      ScaffoldMessenger.of(context)
                          .showSnackBar(SnackBar(content: Text(l10n.copied)));
                    }
                  },
                ),
              ),
            ),
          const SizedBox(height: 4),
          ...switch (friends) {
            AsyncData(:final value) when value.isEmpty => [
              EmptyState(icon: Icons.people_alt, text: l10n.friendsEmpty),
            ],
            AsyncData(:final value) => [
              for (final f in value) _FriendTile(friend: f),
            ],
            AsyncError(:final error) => [
              ErrorView(message: playErrorText(l10n, error)),
            ],
            _ => [
              const Padding(padding: EdgeInsets.all(24), child: LoadingView()),
            ],
          },
        ],
      ),
    );
  }
}

enum _FriendMenu { challenge, unfriend, block }

class _FriendTile extends ConsumerWidget {
  const _FriendTile({required this.friend});

  final Friend friend;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final card = friend.card;
    return TileCard(
      child: ListTile(
        key: Key('friend-${card.userId}'),
        leading: AvatarBubble(
          avatarFor(card.avatarId),
          size: 44,
          online: friend.online ?? false,
        ),
        title: Text(card.displayName),
        subtitle: Text(
          friend.online ?? false
              ? '@${card.username} · ${l10n.online}'
              : '@${card.username}',
        ),
        trailing: PopupMenuButton<_FriendMenu>(
          key: Key('friend-menu-${card.userId}'),
          onSelected: (m) => unawaited(switch (m) {
            _FriendMenu.challenge => showChallengeDialog(context, ref, card),
            _FriendMenu.unfriend => _run(
              context,
              ref,
              () => ref.read(socialRepositoryProvider).unfriend(card.userId),
            ),
            _FriendMenu.block => _block(context, ref, card),
          }),
          itemBuilder: (_) => [
            PopupMenuItem(
              value: _FriendMenu.challenge,
              child: Text(l10n.challengeAction),
            ),
            PopupMenuItem(
              value: _FriendMenu.unfriend,
              child: Text(l10n.unfriend),
            ),
            PopupMenuItem(
              value: _FriendMenu.block,
              child: Text(l10n.blockUser),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _block(
    BuildContext context,
    WidgetRef ref,
    UserCard card,
  ) async {
    final l10n = context.l10n;
    final ok = await confirmDialog(
      context,
      title: l10n.blockUser,
      body: l10n.blockConfirm(card.displayName),
      confirmLabel: l10n.blockUser,
      destructive: true,
    );
    if (ok && context.mounted) {
      await _run(
        context,
        ref,
        () => ref.read(socialRepositoryProvider).block(card.userId),
      );
    }
  }
}

class _RequestsTab extends ConsumerWidget {
  const _RequestsTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final repo = ref.read(socialRepositoryProvider);
    return switch (ref.watch(friendRequestsProvider)) {
      AsyncData(:final value) when value.isEmpty => EmptyState(
        icon: Icons.mark_email_read_rounded,
        text: l10n.requestsEmpty,
      ),
      AsyncData(:final value) => ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          for (final r in value)
            TileCard(
              highlight: r.incoming ? BvColors.of(context).accent : null,
              child: ListTile(
                key: Key('request-${r.id}'),
                leading: AvatarBubble(avatarFor(r.other.avatarId), size: 44),
                title: Text(r.other.displayName),
                subtitle: Text(
                  r.incoming ? l10n.incomingLabel : l10n.outgoingLabel,
                ),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: r.incoming
                      ? [
                          IconButton(
                            key: Key('accept-${r.id}'),
                            tooltip: l10n.accept,
                            icon: Icon(
                              Icons.check_circle_rounded,
                              color: BvColors.of(context).success,
                            ),
                            onPressed: () => _run(
                              context,
                              ref,
                              () => repo.respond(r.id, RequestAction.accept),
                            ),
                          ),
                          IconButton(
                            tooltip: l10n.decline,
                            icon: const Icon(Icons.close),
                            onPressed: () => _run(
                              context,
                              ref,
                              () => repo.respond(r.id, RequestAction.decline),
                            ),
                          ),
                        ]
                      : [
                          TextButton(
                            onPressed: () => _run(
                              context,
                              ref,
                              () => repo.respond(r.id, RequestAction.cancel),
                            ),
                            child: Text(l10n.withdraw),
                          ),
                        ],
                ),
              ),
            ),
        ],
      ),
      AsyncError(:final error) => ErrorView(
        message: playErrorText(l10n, error),
      ),
      _ => const LoadingView(),
    };
  }
}
