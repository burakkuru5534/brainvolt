import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/cosmetics.dart';
import '../../../core/widgets/dialogs.dart';
import '../../auth/domain/session.dart';
import '../../home/presentation/main_nav.dart';
import '../../progress/domain/progress_repository.dart';
import '../../progress/presentation/badges_screen.dart';
import '../../progress/presentation/progress_widgets.dart';
import '../../store/domain/store_repository.dart';
import '../../store/presentation/cosmetics_tab.dart';
import '../data/profile_providers.dart';
import '../domain/avatars.dart';
import '../domain/me.dart';
import 'profile_controllers.dart';

/// The player's profile (BrainVolt, D-099): header with avatar, level and
/// league, stat boxes, earned badges, then editing of display name,
/// username and avatar (roadmap 1.8.6). The server validates and filters
/// every field; its errors are shown as they come.
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final me = ref.watch(meProvider);
    return Scaffold(
      bottomNavigationBar: const MainNavBar(current: MainTab.profile),
      appBar: AppBar(title: Text(l10n.profileTitle)),
      body: switch (me) {
        AsyncValue(value: final me?) => _ProfileForm(me: me),
        AsyncValue(:final error?) => ErrorView(
          message: error is ApiException
              ? error.localizedMessage(l10n)
              : l10n.errorUnknown,
          retryLabel: l10n.retry,
          onRetry: () => ref.invalidate(meProvider),
        ),
        _ => const LoadingView(),
      },
    );
  }
}

class _ProfileForm extends ConsumerStatefulWidget {
  const _ProfileForm({required this.me});

  final Me me;

  @override
  ConsumerState<_ProfileForm> createState() => _ProfileFormState();
}

class _ProfileFormState extends ConsumerState<_ProfileForm> {
  late final _displayName = TextEditingController(
    text: widget.me.profile.displayName,
  );
  late final _username = TextEditingController(
    text: widget.me.profile.username,
  );
  late String _avatar = widget.me.profile.avatarId;
  bool _busy = false;

  @override
  void dispose() {
    _displayName.dispose();
    _username.dispose();
    super.dispose();
  }

  OwnProfile get _profile => widget.me.profile;

  /// Restricted accounts (guests, unverified) keep the generated username
  /// (D-016); full accounts change it once per 30 days.
  bool get _usernameEditable {
    final at = _profile.usernameChangeAvailableAt;
    return widget.me.user.tier == Tier.full &&
        (at == null || !at.isAfter(DateTime.now()));
  }

  Future<void> _save() async {
    final l10n = context.l10n;
    final displayName = _displayName.text.trim();
    final username = _username.text.trim();
    setState(() => _busy = true);
    try {
      await ref
          .read(profileRepositoryProvider)
          .updateProfile(
            displayName: displayName == _profile.displayName
                ? null
                : displayName,
            username: username == _profile.username ? null : username,
            avatarId: _avatar == _profile.avatarId ? null : _avatar,
          );
      ref.invalidate(meProvider);
      // The home screen shows name and avatar from the session.
      await ref.read(sessionControllerProvider.notifier).refresh();
      if (mounted) {
        showSnack(context, l10n.profileSaved);
      }
    } on ApiException catch (e) {
      if (mounted) {
        showSnack(context, e.localizedMessage(l10n));
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
    final text = Theme.of(context).textTheme;
    final available = _profile.usernameChangeAvailableAt;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 24),
      children: [
        _Header(profile: _profile, avatar: _avatar),
        const SizedBox(height: 14),
        const _Stats(),
        const _Achievements(),
        SectionTitle(l10n.profileEdit),
        BvCard(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(l10n.avatarLabel, style: text.titleSmall),
              const SizedBox(height: 10),
              Wrap(
                spacing: 10,
                runSpacing: 10,
                children: [
                  for (final MapEntry(key: id, value: emoji) in {
                    ...avatarEmoji,
                    // Special avatars bought in the store (D-106).
                    for (final a
                        in ref.watch(cosmeticsProvider).value?.ownedAvatars ??
                            const <String>[])
                      a: avatarFor(a),
                  }.entries)
                    Semantics(
                      selected: _avatar == id,
                      button: true,
                      child: InkWell(
                        key: Key('avatar-$id'),
                        customBorder: const CircleBorder(),
                        onTap: _busy
                            ? null
                            : () => setState(() => _avatar = id),
                        child: AnimatedContainer(
                          duration: const Duration(milliseconds: 150),
                          width: 52,
                          height: 52,
                          padding: const EdgeInsets.all(2.5),
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            gradient: _avatar == id ? bv.gradient : null,
                            border: _avatar == id
                                ? null
                                : Border.all(color: bv.cardBorder),
                          ),
                          child: CircleAvatar(
                            backgroundColor: _avatar == id
                                ? bv.surface
                                : Colors.white.withAlpha(14),
                            child: Text(
                              emoji,
                              style: const TextStyle(fontSize: 24),
                            ),
                          ),
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 20),
              TextField(
                key: const Key('display-name-field'),
                controller: _displayName,
                enabled: !_busy,
                maxLength: 24,
                decoration: InputDecoration(
                  labelText: l10n.displayNameLabel,
                  helperText: l10n.displayNameHint,
                  prefixIcon: const Icon(Icons.badge_outlined),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                key: const Key('username-field'),
                controller: _username,
                enabled: !_busy && _usernameEditable,
                maxLength: 20,
                decoration: InputDecoration(
                  labelText: l10n.usernameLabel,
                  prefixIcon: const Icon(Icons.alternate_email),
                  helperText: widget.me.user.tier != Tier.full
                      ? l10n.usernameGuestHint
                      : !_usernameEditable && available != null
                      ? l10n.usernameAvailableAt(available.toLocal())
                      : l10n.usernameHint,
                  helperMaxLines: 2,
                ),
              ),
              const SizedBox(height: 18),
              GradientButton(
                key: const Key('profile-save'),
                height: 52,
                icon: Icons.check_rounded,
                label: l10n.save,
                onPressed: _busy ? null : _save,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// Banner, avatar with the brand ring, name, @username, level and league.
class _Header extends ConsumerWidget {
  const _Header({required this.profile, required this.avatar});

  final OwnProfile profile;
  final String avatar;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final progress = ref.watch(myProgressProvider).value;
    final standing = ref.watch(myStandingProvider).value;
    return BvCard(
      padding: EdgeInsets.zero,
      child: Column(
        children: [
          SizedBox(
            height: 150,
            child: Stack(
              clipBehavior: Clip.none,
              alignment: Alignment.topCenter,
              children: [
                Container(
                  height: 96,
                  decoration: BoxDecoration(
                    gradient: bv.heroGradient,
                    borderRadius: const BorderRadius.vertical(
                      top: Radius.circular(20),
                    ),
                  ),
                  child: Stack(
                    children: [
                      Positioned(
                        right: 12,
                        top: 8,
                        child: Icon(
                          Icons.bolt,
                          size: 80,
                          color: Colors.white.withAlpha(36),
                        ),
                      ),
                      Positioned(
                        left: 14,
                        top: 10,
                        child: Icon(
                          Icons.auto_awesome,
                          size: 26,
                          color: Colors.white.withAlpha(60),
                        ),
                      ),
                    ],
                  ),
                ),
                Positioned(
                  top: 44,
                  child: AvatarBubble(
                    avatarFor(avatar),
                    size: 100,
                    frame: myCosmetic(ref, CosmeticKind.frame),
                  ),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: Column(
              children: [
                CosmeticName(
                  profile.displayName,
                  nameColor: myCosmetic(ref, CosmeticKind.nameColor),
                  textAlign: TextAlign.center,
                  style: text.headlineSmall,
                ),
                Text(
                  '@${profile.username}',
                  style: text.bodyMedium?.copyWith(color: bv.muted),
                ),
                const SizedBox(height: 10),
                Wrap(
                  spacing: 8,
                  runSpacing: 6,
                  alignment: WrapAlignment.center,
                  children: [
                    if (progress != null)
                      Pill(
                        l10n.levelLabel(progress.level),
                        icon: Icons.bolt,
                        gradient: true,
                      ),
                    Pill(
                      standing == null
                          ? l10n.homeNoRank
                          : placementName(
                              l10n,
                              standing.league,
                              standing.division,
                            ),
                      icon: Icons.shield_rounded,
                      color: bv.gold,
                    ),
                  ],
                ),
                if (progress != null) ...[
                  const SizedBox(height: 12),
                  GradientProgressBar(
                    value: progress.levelSize == 0
                        ? 1
                        : progress.levelXp / progress.levelSize,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    l10n.xpProgress(progress.levelXp, progress.levelSize),
                    style: text.labelSmall,
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Matches, wins, win rate and the longest streak.
class _Stats extends ConsumerWidget {
  const _Stats();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final p = ref.watch(myProgressProvider).value;
    final rate = NumberFormat.percentPattern(
      Localizations.localeOf(context).toLanguageTag(),
    ).format(p == null || p.matches == 0 ? 0 : p.wins / p.matches);
    final boxes = [
      StatBox(
        icon: Icons.sports_esports_rounded,
        value: '${p?.matches ?? 0}',
        label: l10n.profileMatches,
        color: bv.badgeInk,
      ),
      StatBox(
        icon: Icons.emoji_events_rounded,
        value: '${p?.wins ?? 0}',
        label: l10n.profileWins,
        color: bv.gold,
      ),
      StatBox(
        icon: Icons.percent_rounded,
        value: rate,
        label: l10n.profileWinRate,
        color: bv.success,
      ),
      StatBox(
        icon: Icons.local_fire_department_rounded,
        value: '${p?.longestStreak ?? 0}',
        label: l10n.profileBestStreak,
        color: bv.accent,
      ),
    ];
    return Row(
      children: [
        for (final (i, b) in boxes.indexed) ...[
          if (i > 0) const SizedBox(width: 8),
          Expanded(child: b),
        ],
      ],
    );
  }
}

/// Earned badges in a strip, with a link to all of them.
class _Achievements extends ConsumerWidget {
  const _Achievements();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final earned = [
      for (final b in ref.watch(badgesProvider).value ?? const <BadgeInfo>[])
        if (b.earnedAt != null) b,
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SectionTitle(
          l10n.profileAchievements,
          action: l10n.homeSeeAll,
          onAction: () => context.push(Routes.badges),
        ),
        if (earned.isEmpty)
          BvCard(
            child: Row(
              children: [
                const IconBadge(Icons.workspace_premium_rounded, size: 44),
                const SizedBox(width: 12),
                Expanded(child: Text(l10n.profileNoBadges)),
              ],
            ),
          )
        else
          SizedBox(
            height: 124,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: earned.length,
              separatorBuilder: (_, _) => const SizedBox(width: 10),
              itemBuilder: (_, i) => SizedBox(
                width: 96,
                child: BadgeTile(badge: earned[i], compact: true),
              ),
            ),
          ),
      ],
    );
  }
}
