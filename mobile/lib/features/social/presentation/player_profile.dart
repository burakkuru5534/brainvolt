import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../profile/domain/avatars.dart';
import '../data/social_providers.dart';
import '../domain/social_repository.dart';
import 'friends_screen.dart';

/// A player's profile as the signed-in user sees it (D-101).
final playerProfileProvider = FutureProvider.autoDispose
    .family<PlayerProfile, String>(
      (ref, id) => ref.watch(socialRepositoryProvider).profile(id),
    );

/// The quick card of a player (leaderboards, D-101): who they are, add
/// friend or the request's state, and a way to the full profile.
Future<void> showPlayerSheet(BuildContext context, UserCard card) =>
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) => _PlayerSheet(card),
    );

class _PlayerSheet extends ConsumerWidget {
  const _PlayerSheet(this.card);

  final UserCard card;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final profile = ref.watch(playerProfileProvider(card.userId)).value;
    return SafeArea(
      child: Padding(
        key: const Key('player-sheet'),
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _Header(card: profile?.card ?? card, profile: profile),
            const SizedBox(height: 14),
            if (profile == null)
              const Padding(
                padding: EdgeInsets.all(12),
                child: Center(child: CircularProgressIndicator()),
              )
            else ...[
              if (profile.stats case final s?)
                _StatsRow(s)
              else if (profile.private)
                NoticeBar(
                  l10n.playerPrivate,
                  key: const Key('player-private'),
                  icon: Icons.lock_rounded,
                ),
              const SizedBox(height: 14),
              _RelationAction(profile),
              const SizedBox(height: 8),
              OutlinedButton.icon(
                key: const Key('player-view-profile'),
                icon: const Icon(Icons.person_rounded),
                label: Text(l10n.playerViewProfile),
                onPressed: () {
                  Navigator.of(context).pop();
                  unawaited(context.push(Routes.player(card.userId)));
                },
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// The full profile page of a player (D-101).
class PlayerProfileScreen extends ConsumerWidget {
  const PlayerProfileScreen({required this.userId, super.key});

  final String userId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.playerProfileTitle)),
      body: switch (ref.watch(playerProfileProvider(userId))) {
        AsyncData(:final value) => ListView(
          padding: const EdgeInsets.all(16),
          children: [
            BvCard(
              padding: const EdgeInsets.all(18),
              child: _Header(card: value.card, profile: value, large: true),
            ),
            const SizedBox(height: 16),
            if (value.stats case final s?) ...[
              SectionTitle(l10n.playerStats),
              _StatsGrid(s),
            ] else
              EmptyState(
                icon: Icons.lock_rounded,
                text: value.relation == Relation.blocked
                    ? l10n.playerBlocked
                    : l10n.playerPrivate,
                textKey: const Key('player-private'),
              ),
            const SizedBox(height: 16),
            _RelationAction(value),
          ],
        ),
        AsyncError(:final error) => ErrorView(
          message: error is ApiException
              ? error.localizedMessage(l10n)
              : l10n.errorUnknown,
          retryLabel: l10n.retry,
          onRetry: () => ref.invalidate(playerProfileProvider(userId)),
        ),
        _ => const LoadingView(),
      },
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.card, this.profile, this.large = false});

  final UserCard card;
  final PlayerProfile? profile;
  final bool large;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return Row(
      children: [
        AvatarBubble(avatarFor(card.avatarId), size: large ? 72 : 60),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                card.displayName,
                key: const Key('player-name'),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: large ? text.headlineSmall : text.titleLarge,
              ),
              if (card.username.isNotEmpty)
                Text(
                  '@${card.username}',
                  style: text.bodyMedium?.copyWith(color: bv.muted),
                ),
              if (profile case final p?) ...[
                const SizedBox(height: 6),
                Wrap(
                  spacing: 6,
                  runSpacing: 6,
                  children: [
                    if (p.stats case final s?)
                      Pill(
                        l10n.playerLevel(s.level),
                        icon: Icons.bolt_rounded,
                        gradient: true,
                      ),
                    if (p.countryCode case final c? when c.isNotEmpty)
                      Pill(_flag(c)),
                    if (p.private)
                      Pill(l10n.playerPrivateTag, icon: Icons.lock_rounded),
                  ],
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

/// The flag emoji of an ISO country code.
String _flag(String code) => code.length != 2
    ? code
    : String.fromCharCodes([
        for (final c in code.toUpperCase().codeUnits) 0x1F1E6 + c - 65,
      ]);

class _StatsRow extends StatelessWidget {
  const _StatsRow(this.s);

  final PlayerStats s;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Row(
      children: [
        for (final (icon, value, label) in [
          (Icons.sports_esports_rounded, '${s.matches}', l10n.playerMatches),
          (Icons.emoji_events_rounded, '${s.wins}', l10n.playerWins),
          (Icons.percent_rounded, '%${s.winRate}', l10n.playerWinRate),
        ]) ...[
          Expanded(
            child: StatBox(icon: icon, value: value, label: label),
          ),
          if (label != l10n.playerWinRate) const SizedBox(width: 8),
        ],
      ],
    );
  }
}

class _StatsGrid extends StatelessWidget {
  const _StatsGrid(this.s);

  final PlayerStats s;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final boxes = [
      StatBox(
        icon: Icons.bolt_rounded,
        value: '${s.level}',
        label: l10n.playerLevelLabel,
        valueKey: const Key('player-level'),
      ),
      StatBox(
        icon: Icons.sports_esports_rounded,
        value: '${s.matches}',
        label: l10n.playerMatches,
      ),
      StatBox(
        icon: Icons.emoji_events_rounded,
        value: '${s.wins}',
        label: l10n.playerWins,
        color: bv.gold,
      ),
      StatBox(
        icon: Icons.percent_rounded,
        value: '%${s.winRate}',
        label: l10n.playerWinRate,
      ),
      StatBox(
        icon: Icons.military_tech_rounded,
        value: '${s.badges}',
        label: l10n.playerBadges,
      ),
      StatBox(
        icon: Icons.local_fire_department_rounded,
        value: '${s.streak}',
        label: l10n.playerStreak,
        color: bv.gradientEnd,
      ),
    ];
    return Column(
      children: [
        for (var i = 0; i < boxes.length; i += 3) ...[
          Row(
            children: [
              for (var j = i; j < i + 3; j++) ...[
                Expanded(child: boxes[j]),
                if (j < i + 2) const SizedBox(width: 8),
              ],
            ],
          ),
          const SizedBox(height: 8),
        ],
      ],
    );
  }
}

/// Add friend, accept their request, or the current state.
class _RelationAction extends ConsumerStatefulWidget {
  const _RelationAction(this.profile);

  final PlayerProfile profile;

  @override
  ConsumerState<_RelationAction> createState() => _RelationActionState();
}

class _RelationActionState extends ConsumerState<_RelationAction> {
  var _busy = false;

  Future<void> _send() async {
    final l10n = context.l10n;
    final id = widget.profile.card.userId;
    setState(() => _busy = true);
    try {
      final accepted = await ref
          .read(socialRepositoryProvider)
          .sendRequest(userId: id, via: 'profile');
      if (!mounted) {
        return;
      }
      showSnack(context, accepted ? l10n.friendAdded : l10n.friendRequestSent);
      ref
        ..invalidate(playerProfileProvider(id))
        ..invalidate(friendsProvider)
        ..invalidate(friendRequestsProvider);
    } on Object catch (e) {
      if (mounted) {
        showSnack(
          context,
          e is ApiException ? e.localizedMessage(l10n) : l10n.errorUnknown,
        );
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    Widget state(IconData icon, String text, Color color) =>
        Pill(text, key: const Key('player-relation'), icon: icon, color: color);
    return switch (widget.profile.relation) {
      Relation.self || Relation.blocked => const SizedBox.shrink(),
      Relation.friend => Center(
        child: state(Icons.people_rounded, l10n.playerFriend, bv.success),
      ),
      Relation.requestSent => Center(
        child: state(Icons.schedule_rounded, l10n.playerRequestSent, bv.muted),
      ),
      Relation.requestReceived => GradientButton(
        key: const Key('player-accept'),
        icon: Icons.how_to_reg_rounded,
        label: l10n.playerAccept,
        onPressed: _busy ? null : _send,
      ),
      Relation.none => GradientButton(
        key: const Key('player-add-friend'),
        icon: Icons.person_add_rounded,
        label: l10n.playerAddFriend,
        onPressed: _busy ? null : _send,
      ),
    };
  }
}
