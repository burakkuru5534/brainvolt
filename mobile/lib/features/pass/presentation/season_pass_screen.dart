import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_client.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../../progress/presentation/progress_widgets.dart';
import '../../store/presentation/cosmetics_tab.dart';
import '../../store/presentation/volt_screen.dart';
import '../data/api_season_pass_repository.dart';
import '../domain/season_pass.dart';

/// The Season Pass endpoints. Tests override it with a fake.
final seasonPassRepositoryProvider = Provider<SeasonPassRepository>(
  (ref) => ApiSeasonPassRepository(ref.watch(apiClientProvider)),
);

/// The signed-in user's Season Pass (D-108).
final seasonPassProvider = FutureProvider.autoDispose<SeasonPass?>((ref) {
  final user = ref.watch(
    sessionControllerProvider.select(
      (s) => switch (s.value) {
        SignedIn(:final session) => session.user.id,
        _ => null,
      },
    ),
  );
  if (user == null) {
    return null;
  }
  return ref.watch(seasonPassRepositoryProvider).pass();
});

/// Home: the season's tier and rewards waiting; opens the pass.
class SeasonPassCard extends ConsumerWidget {
  const SeasonPassCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final p = ref.watch(seasonPassProvider).value;
    if (p == null) {
      return const SizedBox.shrink();
    }
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.only(top: 14),
      child: BvCard(
        key: const Key('season-pass-card'),
        onTap: () => context.push(Routes.seasonPass),
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            const IconBadge(Icons.military_tech_rounded, size: 44),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(l10n.passTitle, style: text.titleSmall),
                  const SizedBox(height: 4),
                  GradientProgressBar(value: p.tier / p.maxTier),
                  const SizedBox(height: 4),
                  Text(
                    l10n.passTier(p.tier, p.maxTier),
                    style: text.labelMedium?.copyWith(color: bv.muted),
                  ),
                ],
              ),
            ),
            if (p.claimable > 0) ...[
              const SizedBox(width: 10),
              Badge(
                key: const Key('season-pass-claimable'),
                label: Text('${p.claimable}'),
                child: Icon(Icons.card_giftcard_rounded, color: bv.gold),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// The Season Pass: progress, the premium track, claim and all 40 tiers.
class SeasonPassScreen extends ConsumerStatefulWidget {
  const SeasonPassScreen({super.key});

  @override
  ConsumerState<SeasonPassScreen> createState() => _SeasonPassScreenState();
}

class _SeasonPassScreenState extends ConsumerState<SeasonPassScreen> {
  var _busy = false;

  Future<void> _act(Future<String?> Function() action) async {
    final l10n = context.l10n;
    setState(() => _busy = true);
    try {
      final msg = await action();
      if (msg != null && mounted) {
        showSnack(context, msg);
      }
    } on Object catch (e) {
      if (mounted) {
        showSnack(context, playErrorText(l10n, e));
      }
    } finally {
      ref
        ..invalidate(seasonPassProvider)
        ..invalidate(walletProvider)
        ..invalidate(cosmeticsProvider)
        ..invalidate(myProgressProvider);
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _unlock(SeasonPass p) async {
    final l10n = context.l10n;
    final ok = await confirmDialog(
      context,
      title: l10n.passUnlockTitle,
      body: l10n.voltBuyBody(p.price),
      confirmLabel: l10n.voltBuy,
    );
    if (!ok || !mounted) {
      return;
    }
    await _act(() async {
      await ref.read(seasonPassRepositoryProvider).buyPremium();
      return l10n.passUnlocked;
    });
  }

  Future<void> _claim() {
    final l10n = context.l10n;
    return _act(() async {
      final c = await ref.read(seasonPassRepositoryProvider).claim();
      final volt = l10n.passClaimedVolt(c.volt);
      return c.shields == 0 && c.cosmetics.isEmpty
          ? volt
          : l10n.passClaimedMore(volt, c.shields, c.cosmetics.length);
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.passTitle)),
      body: switch (ref.watch(seasonPassProvider)) {
        AsyncData(:final value?) => _body(context, value),
        AsyncError(:final error) => ErrorView(
          message: playErrorText(l10n, error),
          retryLabel: l10n.retry,
          onRetry: () => ref.invalidate(seasonPassProvider),
        ),
        _ => const LoadingView(),
      },
    );
  }

  Widget _body(BuildContext context, SeasonPass p) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final days = p.endsAt.difference(DateTime.now()).inDays.clamp(0, 999);
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
      children: [
        BvCard(
          gradient: bv.heroGradient,
          padding: const EdgeInsets.all(18),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.passSeason(p.seasonNumber, p.seasonName),
                key: const Key('pass-season'),
                style: text.titleLarge?.copyWith(color: Colors.white),
              ),
              Text(
                l10n.passEndsIn(days),
                style: text.bodySmall?.copyWith(color: Colors.white70),
              ),
              const SizedBox(height: 14),
              Row(
                children: [
                  Text(
                    l10n.passTier(p.tier, p.maxTier),
                    key: const Key('pass-tier'),
                    style: text.headlineSmall?.copyWith(color: Colors.white),
                  ),
                  const Spacer(),
                  if (p.premium)
                    Pill(
                      l10n.passPremium,
                      icon: Icons.workspace_premium_rounded,
                      gradient: true,
                    ),
                ],
              ),
              const SizedBox(height: 8),
              GradientProgressBar(
                value: p.tier >= p.maxTier ? 1 : p.tierXp / p.tierSize,
              ),
              const SizedBox(height: 6),
              Text(
                p.tier >= p.maxTier
                    ? l10n.passAllDone
                    : l10n.passXpToNext(p.tierSize - p.tierXp),
                style: text.bodySmall?.copyWith(color: Colors.white),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        if (!p.premium)
          BvCard(
            key: const Key('pass-unlock-card'),
            padding: const EdgeInsets.all(14),
            child: Row(
              children: [
                Icon(Icons.workspace_premium_rounded, size: 40, color: bv.gold),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(l10n.passUnlock, style: text.titleSmall),
                      Text(
                        l10n.passUnlockSub,
                        style: text.bodySmall?.copyWith(color: bv.muted),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                FilledButton.icon(
                  key: const Key('pass-unlock'),
                  icon: const Icon(voltIcon, size: 18),
                  onPressed: _busy ? null : () => _unlock(p),
                  label: Text('${p.price}'),
                ),
              ],
            ),
          ),
        if (p.claimable > 0) ...[
          const SizedBox(height: 12),
          GradientButton(
            key: const Key('pass-claim'),
            icon: Icons.card_giftcard_rounded,
            label: l10n.passClaimAll(p.claimable),
            onPressed: _busy ? null : _claim,
          ),
        ],
        const SizedBox(height: 8),
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Row(
            children: [
              const SizedBox(width: 52),
              Expanded(
                child: Text(
                  l10n.passFree,
                  textAlign: TextAlign.center,
                  style: text.labelLarge?.copyWith(color: bv.muted),
                ),
              ),
              Expanded(
                child: Text(
                  l10n.passPremium,
                  textAlign: TextAlign.center,
                  style: text.labelLarge?.copyWith(color: bv.gold),
                ),
              ),
            ],
          ),
        ),
        for (final t in p.tiers) _tierRow(context, p, t),
      ],
    );
  }

  Widget _tierRow(BuildContext context, SeasonPass p, PassTier t) {
    final bv = BvColors.of(context);
    final reached = t.tier <= p.tier;
    return Padding(
      key: Key('pass-tier-${t.tier}'),
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              gradient: reached ? bv.gradient : null,
              color: reached ? null : bv.badge,
            ),
            child: Text(
              '${t.tier}',
              style: Theme.of(context).textTheme.titleSmall
                  ?.copyWith(color: reached ? Colors.white : bv.muted),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _RewardCell(
              reward: t.free,
              reached: reached,
              claimed: t.freeClaimed,
              locked: false,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _RewardCell(
              reward: t.premium,
              reached: reached,
              claimed: t.premiumClaimed,
              locked: !p.premium,
            ),
          ),
        ],
      ),
    );
  }
}

class _RewardCell extends StatelessWidget {
  const _RewardCell({
    required this.reward,
    required this.reached,
    required this.claimed,
    required this.locked,
  });

  final PassReward? reward;
  final bool reached;
  final bool claimed;
  final bool locked;

  @override
  Widget build(BuildContext context) {
    final r = reward;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    if (r == null) {
      return const SizedBox(height: 56);
    }
    final l10n = context.l10n;
    final parts = <(Widget, String)>[
      if (r.cosmetic.isNotEmpty)
        (_preview(r.cosmetic), cosmeticName(l10n, r.cosmetic)),
      if (r.shield)
        (Icon(Icons.shield_rounded, color: bv.accent), l10n.passRewardShield),
      if (r.volt > 0)
        (Icon(voltIcon, color: bv.gold), l10n.passClaimedVolt(r.volt)),
    ];
    return Opacity(
      opacity: reached ? 1 : 0.55,
      child: BvCard(
        radius: 14,
        color: claimed ? bv.success.withAlpha(35) : null,
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
        child: SizedBox(
          height: 40,
          child: Row(
            children: [
              parts.first.$1,
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  parts.map((p) => p.$2).join(' + '),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: text.labelMedium,
                ),
              ),
              if (claimed)
                Icon(Icons.check_circle_rounded, size: 18, color: bv.success)
              else if (locked)
                Icon(Icons.lock_rounded, size: 16, color: bv.muted),
            ],
          ),
        ),
      ),
    );
  }

  Widget _preview(String code) {
    if (code.startsWith('frame_')) {
      return AvatarBubble('⚡', size: 30, frame: code);
    }
    if (code.startsWith('avatar_')) {
      return AvatarBubble(avatarFor(code.substring(7)), size: 30, ring: false);
    }
    return Icon(switch (code) {
      _ when code.startsWith('name_') => Icons.text_fields_rounded,
      _ when code.startsWith('theme_') => Icons.palette_rounded,
      _ => Icons.thunderstorm_rounded,
    }, color: const Color(0xFFB9A8FF));
  }
}
