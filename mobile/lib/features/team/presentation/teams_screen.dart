import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/dialogs.dart';
import '../../auth/domain/session.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../../social/domain/social_repository.dart';
import '../../social/presentation/friends_screen.dart';
import '../data/team_providers.dart';
import '../domain/team_repository.dart';

/// The caller's teams, team invites and couple.
final teamsOverviewProvider =
    FutureProvider.autoDispose<(List<Team>, List<TeamInvite>, Couple?)>((
      ref,
    ) async {
      final repo = ref.watch(teamRepositoryProvider);
      return (
        await repo.myTeams(),
        await repo.invites(),
        await repo.myCouple(),
      );
    });

/// One team.
final teamProvider = FutureProvider.autoDispose.family<Team, String>(
  (ref, id) => ref.watch(teamRepositoryProvider).team(id),
);

/// Runs [action], reports errors and refreshes the team screens.
Future<void> _run(
  BuildContext context,
  WidgetRef ref,
  Future<void> Function() action,
) async {
  final l10n = context.l10n;
  final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
  try {
    await action();
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
  ref
    ..invalidate(teamsOverviewProvider)
    ..invalidate(teamProvider);
}

/// Picks one of the caller's friends.
Future<UserCard?> pickFriend(BuildContext context, WidgetRef ref) async {
  final l10n = context.l10n;
  final friends = await ref
      .read(friendsProvider.future)
      .catchError((Object _) => <Friend>[]);
  if (!context.mounted) {
    return null;
  }
  return showDialog<UserCard>(
    context: context,
    builder: (context) => SimpleDialog(
      title: Text(l10n.pickFriend),
      children: [
        if (friends.isEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(l10n.friendsEmpty),
          ),
        for (final f in friends)
          SimpleDialogOption(
            key: Key('pick-friend-${f.card.userId}'),
            onPressed: () => Navigator.pop(context, f.card),
            child: Text('${avatarFor(f.card.avatarId)}  ${f.card.displayName}'),
          ),
      ],
    ),
  );
}

/// Teams (2–4 friends with full accounts) and the couple profile.
class TeamsScreen extends ConsumerWidget {
  const TeamsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final full = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.tier == Tier.full,
      _ => false,
    };
    if (!full) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.homeTeams)),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  l10n.teamsUpgrade,
                  key: const Key('teams-upgrade'),
                  textAlign: TextAlign.center,
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
    final repo = ref.read(teamRepositoryProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.homeTeams)),
      floatingActionButton: FloatingActionButton.extended(
        key: const Key('create-team'),
        icon: const Icon(Icons.add),
        label: Text(l10n.createTeam),
        onPressed: () => unawaited(_createTeam(context, ref)),
      ),
      body: switch (ref.watch(teamsOverviewProvider)) {
        AsyncData(value: (final teams, final invites, final couple)) =>
          ListView(
            padding: const EdgeInsets.only(bottom: 96),
            children: [
              _Header(l10n.coupleSection),
              _CoupleTile(couple: couple),
              if (invites.isNotEmpty) ...[
                _Header(l10n.teamInvites),
                for (final i in invites)
                  ListTile(
                    key: Key('team-invite-${i.id}'),
                    leading: Text(
                      emblemEmoji[i.emblemId] ?? '🏳️',
                      style: const TextStyle(fontSize: 28),
                    ),
                    title: Text('[${i.teamTag}] ${i.teamName}'),
                    subtitle: Text(i.invitedBy.displayName),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          key: Key('accept-invite-${i.id}'),
                          tooltip: l10n.accept,
                          icon: const Icon(Icons.check),
                          onPressed: () => _run(
                            context,
                            ref,
                            () => repo.respondInvite(i.id, accept: true),
                          ),
                        ),
                        IconButton(
                          tooltip: l10n.decline,
                          icon: const Icon(Icons.close),
                          onPressed: () => _run(
                            context,
                            ref,
                            () => repo.respondInvite(i.id, accept: false),
                          ),
                        ),
                      ],
                    ),
                  ),
              ],
              _Header(l10n.myTeams),
              if (teams.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(l10n.teamsEmpty),
                ),
              for (final t in teams)
                ListTile(
                  key: Key('team-${t.id}'),
                  leading: Text(
                    emblemEmoji[t.emblemId] ?? '🏳️',
                    style: const TextStyle(fontSize: 28),
                  ),
                  title: Text('[${t.tag}] ${t.name}'),
                  subtitle: Text('${t.members.length}/4'),
                  onTap: () => context.push(Routes.team(t.id)),
                ),
            ],
          ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }

  Future<void> _createTeam(BuildContext context, WidgetRef ref) async {
    final input = await showDialog<(String, String, String)>(
      context: context,
      builder: (_) => const _CreateTeamDialog(),
    );
    if (input == null || !context.mounted) {
      return;
    }
    final (name, tag, emblem) = input;
    await _run(
      context,
      ref,
      () => ref
          .read(teamRepositoryProvider)
          .createTeam(name: name, tag: tag, emblemId: emblem),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header(this.text);
  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
    child: Text(text, style: Theme.of(context).textTheme.titleMedium),
  );
}

class _CoupleTile extends ConsumerWidget {
  const _CoupleTile({required this.couple});

  final Couple? couple;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final repo = ref.read(teamRepositoryProvider);
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final c = couple;
    if (c == null) {
      return ListTile(
        key: const Key('no-couple'),
        title: Text(l10n.noCouple),
        trailing: TextButton(
          key: const Key('request-couple'),
          onPressed: () async {
            final friend = await pickFriend(context, ref);
            if (friend != null && context.mounted) {
              await _run(context, ref, () => repo.requestCouple(friend.userId));
            }
          },
          child: Text(l10n.requestCouple),
        ),
      );
    }
    final partner =
        c.partners.where((p) => p.userId != me).firstOrNull ?? c.partners.first;
    final pending = c.status == 'pending';
    return ListTile(
      key: Key('couple-${c.id}'),
      leading: Text(
        avatarFor(partner.avatarId),
        style: const TextStyle(fontSize: 28),
      ),
      title: Text(c.name ?? l10n.coupleWith(partner.displayName)),
      subtitle: Text(pending ? l10n.couplePending : l10n.coupleActive),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (pending && c.requestedBy != me)
            IconButton(
              key: const Key('accept-couple'),
              tooltip: l10n.accept,
              icon: const Icon(Icons.check),
              onPressed: () =>
                  _run(context, ref, () => repo.respondCouple(c.id, 'accept')),
            ),
          if (!pending)
            IconButton(
              key: const Key('couple-chat'),
              tooltip: l10n.chatOpen,
              icon: const Icon(Icons.chat_bubble_outline),
              onPressed: () => context.push(
                Routes.coupleChat(
                  c.id,
                  c.name ?? l10n.coupleWith(partner.displayName),
                ),
              ),
            ),
          if (pending)
            IconButton(
              tooltip: l10n.decline,
              icon: const Icon(Icons.close),
              onPressed: () =>
                  _run(context, ref, () => repo.respondCouple(c.id, 'decline')),
            )
          else
            TextButton(
              onPressed: () => _run(
                context,
                ref,
                () => repo.respondCouple(c.id, 'dissolve'),
              ),
              child: Text(l10n.dissolveCouple),
            ),
        ],
      ),
    );
  }
}

class _CreateTeamDialog extends StatefulWidget {
  const _CreateTeamDialog();

  @override
  State<_CreateTeamDialog> createState() => _CreateTeamDialogState();
}

class _CreateTeamDialogState extends State<_CreateTeamDialog> {
  final _name = TextEditingController();
  final _tag = TextEditingController();
  String _emblem = emblemEmoji.keys.first;

  @override
  void dispose() {
    _name.dispose();
    _tag.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AlertDialog(
      title: Text(l10n.createTeam),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              key: const Key('team-name'),
              controller: _name,
              maxLength: 24,
              decoration: InputDecoration(labelText: l10n.teamName),
            ),
            TextField(
              key: const Key('team-tag'),
              controller: _tag,
              maxLength: 4,
              textCapitalization: TextCapitalization.characters,
              decoration: InputDecoration(labelText: l10n.teamTag),
            ),
            Wrap(
              spacing: 4,
              children: [
                for (final e in emblemEmoji.entries)
                  ChoiceChip(
                    label: Text(e.value),
                    selected: _emblem == e.key,
                    onSelected: (_) => setState(() => _emblem = e.key),
                  ),
              ],
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('team-create'),
          onPressed: () => Navigator.pop(context, (
            _name.text.trim(),
            _tag.text.trim(),
            _emblem,
          )),
          child: Text(l10n.createTeam),
        ),
      ],
    );
  }
}

/// One team: members, invites (any member), captain tools.
class TeamScreen extends ConsumerWidget {
  const TeamScreen({required this.teamId, super.key});

  final String teamId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final repo = ref.read(teamRepositoryProvider);
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    return switch (ref.watch(teamProvider(teamId))) {
      AsyncData(value: final t) => Scaffold(
        appBar: AppBar(
          title: Text('[${t.tag}] ${t.name}'),
          actions: [
            IconButton(
              key: const Key('team-chat'),
              tooltip: l10n.chatOpen,
              icon: const Icon(Icons.chat_bubble_outline),
              onPressed: () => context.push(Routes.teamChat(t.id, t.name)),
            ),
          ],
        ),
        body: ListView(
          children: [
            for (final m in t.members)
              ListTile(
                key: Key('member-${m.card.userId}'),
                leading: Text(
                  avatarFor(m.card.avatarId),
                  style: const TextStyle(fontSize: 28),
                ),
                title: Text(m.card.displayName),
                subtitle: m.captain ? Text(l10n.captain) : null,
                trailing: t.captainUserId == me && m.card.userId != me
                    ? PopupMenuButton<String>(
                        key: Key('member-menu-${m.card.userId}'),
                        onSelected: (v) => _run(
                          context,
                          ref,
                          () => v == 'captain'
                              ? repo.makeCaptain(teamId, m.card.userId)
                              : repo.kick(teamId, m.card.userId),
                        ),
                        itemBuilder: (_) => [
                          PopupMenuItem(
                            value: 'captain',
                            child: Text(l10n.makeCaptain),
                          ),
                          PopupMenuItem(value: 'kick', child: Text(l10n.kick)),
                        ],
                      )
                    : null,
              ),
            const Divider(),
            if (t.members.length < 4)
              ListTile(
                key: const Key('invite-to-team'),
                leading: const Icon(Icons.person_add),
                title: Text(l10n.inviteToTeam),
                onTap: () async {
                  final friend = await pickFriend(context, ref);
                  if (friend != null && context.mounted) {
                    final messenger = ScaffoldMessenger.of(context);
                    await _run(context, ref, () async {
                      await repo.invite(teamId, friend.userId);
                      messenger.showSnackBar(
                        SnackBar(content: Text(l10n.teamInviteSent)),
                      );
                    });
                  }
                },
              ),
            ListTile(
              key: const Key('leave-team'),
              leading: const Icon(Icons.logout),
              title: Text(l10n.leaveTeam),
              onTap: () async {
                await _run(
                  context,
                  ref,
                  () => repo.teamAction(teamId, TeamAction.leave),
                );
                if (context.mounted) {
                  context.pop();
                }
              },
            ),
            if (t.captainUserId == me)
              ListTile(
                key: const Key('disband-team'),
                leading: const Icon(Icons.delete_forever),
                title: Text(l10n.disbandTeam),
                onTap: () async {
                  final ok = await confirmDialog(
                    context,
                    title: l10n.disbandTeam,
                    body: l10n.disbandConfirm,
                    confirmLabel: l10n.disbandTeam,
                    destructive: true,
                  );
                  if (ok && context.mounted) {
                    await _run(
                      context,
                      ref,
                      () => repo.teamAction(teamId, TeamAction.disband),
                    );
                    if (context.mounted) {
                      context.pop();
                    }
                  }
                },
              ),
          ],
        ),
      ),
      AsyncError(:final error) => Scaffold(
        appBar: AppBar(),
        body: Center(child: Text(playErrorText(l10n, error))),
      ),
      _ => Scaffold(
        appBar: AppBar(),
        body: const Center(child: CircularProgressIndicator()),
      ),
    };
  }
}
