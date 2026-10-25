import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/config.dart';
import '../../../app/providers.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/ads/ad_gateway.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_client.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../match/domain/match_repository.dart';
import '../../match/presentation/play_error_text.dart';
import '../../progress/presentation/progress_widgets.dart';
import '../data/api_ads_repository.dart';
import '../domain/ads_repository.dart';

/// The ad network; main() installs [GoogleAdGateway], tests keep none.
final adGatewayProvider = Provider<AdGateway>((ref) => const NoAdGateway());

/// The ad endpoints. Tests override it with a fake.
final adsRepositoryProvider = Provider<AdsRepository>(
  (ref) => ApiAdsRepository(ref.watch(apiClientProvider)),
);

/// The signed-in user's ad settings; refreshed after every reward.
final adsConfigProvider = FutureProvider<AdsConfig>((ref) {
  final user = ref.watch(
    sessionControllerProvider.select(
      (s) => switch (s.value) {
        SignedIn(:final session) => session.user.id,
        _ => null,
      },
    ),
  );
  if (user == null) {
    return const AdsConfig();
  }
  return ref.watch(adsRepositoryProvider).config();
});

/// How a rewarded ad ended for the player.
enum AdOutcome {
  /// The reward was verified (and is being applied).
  granted,

  /// Verified late: the reward arrives in a moment.
  pending,

  /// The ad was closed before the end.
  notEarned,

  /// No ad could be shown.
  unavailable,
}

/// Runs a rewarded ad end to end (D-103): reserve on the server, show the
/// ad with the reward id as SSV custom data, then wait for AdMob's
/// verified callback to grant it. Development builds grant directly,
/// because test ads cannot call a local server.
class RewardedAdRunner {
  RewardedAdRunner(this.ref, {this.pollEvery = const Duration(seconds: 1)});

  final Ref ref;
  final Duration pollEvery;

  /// Throws [ApiException] when the server refuses the reservation.
  Future<(AdOutcome, AdReward?)> watch(String placement, {String? ref}) async {
    final repo = this.ref.read(adsRepositoryProvider);
    final user = switch (this.ref.read(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final reward = await repo.reserve(placement, ref: ref);
    if (reward.granted) {
      // Plus earns the reward without the ad (D-104).
      await Future<void>.delayed(pollEvery);
      this.ref.invalidate(adsConfigProvider);
      return (AdOutcome.granted, reward);
    }
    final shown = await this.ref
        .read(adGatewayProvider)
        .showRewarded(userId: user, customData: reward.id);
    if (shown != AdShowResult.earned) {
      return (
        shown == AdShowResult.dismissed
            ? AdOutcome.notEarned
            : AdOutcome.unavailable,
        reward,
      );
    }
    if (this.ref.read(appConfigProvider).flavor == Flavor.dev) {
      await repo.devGrant(reward.id);
    }
    for (var i = 0; i < 15; i++) {
      final now = await repo.get(reward.id);
      if (now.granted) {
        // Progression applies the reward from the outbox right after.
        await Future<void>.delayed(pollEvery);
        this.ref.invalidate(adsConfigProvider);
        return (AdOutcome.granted, reward);
      }
      await Future<void>.delayed(pollEvery);
    }
    this.ref.invalidate(adsConfigProvider);
    return (AdOutcome.pending, reward);
  }
}

final rewardedAdRunnerProvider = Provider<RewardedAdRunner>(
  RewardedAdRunner.new,
);

/// Snack text for an outcome that did not give the reward now.
String? adOutcomeText(AppLocalizations l10n, AdOutcome o) => switch (o) {
  AdOutcome.granted => null,
  AdOutcome.pending => l10n.adPending,
  AdOutcome.notEarned => l10n.adNotEarned,
  AdOutcome.unavailable => l10n.adNoFill,
};

/// One interstitial every [AdsConfig.interstitialEvery] finished matches
/// (D-103), counted on the device: never after Ranked or tournament
/// matches, never for new players or Plus (the server's config says).
class InterstitialPolicy {
  InterstitialPolicy(this.ref);

  final Ref ref;

  static const _countKey = 'ads_matches_since_interstitial';
  static const _lastKey = 'ads_last_counted_match';

  /// Counts [r] once and shows an interstitial when its turn has come.
  Future<bool> afterMatch(MatchResult r) async {
    if (r.status != 'completed' ||
        r.matchType == 'ranked' ||
        r.origin == 'tournament') {
      return false;
    }
    final prefs = ref.read(sharedPreferencesProvider);
    if (prefs.getString(_lastKey) == r.id) {
      return false;
    }
    await prefs.setString(_lastKey, r.id);
    final AdsConfig config;
    try {
      config = await ref.read(adsConfigProvider.future);
    } on Object {
      return false;
    }
    if (!config.interstitialEnabled) {
      return false;
    }
    final count = (prefs.getInt(_countKey) ?? 0) + 1;
    if (count < config.interstitialEvery) {
      await prefs.setInt(_countKey, count);
      return false;
    }
    final shown = await ref.read(adGatewayProvider).showInterstitial();
    await prefs.setInt(_countKey, shown ? 0 : count);
    return shown;
  }
}

final interstitialPolicyProvider = Provider<InterstitialPolicy>(
  InterstitialPolicy.new,
);

/// Result screen: "Watch an ad: +50% XP" (D-103).
class BonusXpAdButton extends ConsumerStatefulWidget {
  const BonusXpAdButton({required this.matchId, super.key});

  final String matchId;

  @override
  ConsumerState<BonusXpAdButton> createState() => _BonusXpAdButtonState();
}

class _BonusXpAdButtonState extends ConsumerState<BonusXpAdButton> {
  var _busy = false;
  int? _gained;
  var _gone = false;

  Future<void> _watch() async {
    final l10n = context.l10n;
    setState(() => _busy = true);
    try {
      final (outcome, reward) = await ref
          .read(rewardedAdRunnerProvider)
          .watch(AdPlacement.bonusXp, ref: widget.matchId);
      if (!mounted) {
        return;
      }
      if (outcome == AdOutcome.granted || outcome == AdOutcome.pending) {
        setState(() => _gained = reward?.xp ?? 0);
      }
      if (adOutcomeText(l10n, outcome) case final text?) {
        showSnack(context, text);
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() => _gone = true);
        showSnack(context, playErrorText(l10n, e));
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
    final config = ref.watch(adsConfigProvider).value;
    if (_gained case final xp?) {
      return Padding(
        padding: const EdgeInsets.only(top: 10),
        child: NoticeBar(
          l10n.adBonusXpDone(xp),
          key: const Key('ad-bonus-done'),
          icon: Icons.bolt_rounded,
        ),
      );
    }
    if (_gone || config == null || config.bonusXpLeft <= 0) {
      return const SizedBox.shrink();
    }
    return Padding(
      padding: const EdgeInsets.only(top: 10),
      child: OutlinedButton.icon(
        key: const Key('ad-bonus-xp'),
        icon: _busy
            ? const SizedBox.square(
                dimension: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Icon(Icons.ondemand_video_rounded),
        label: Text(l10n.adBonusXpButton),
        onPressed: _busy ? null : _watch,
      ),
    );
  }
}

/// Home: a streak lost by missing only yesterday can be saved with an ad
/// (once a week).
class StreakRepairBanner extends ConsumerStatefulWidget {
  const StreakRepairBanner({super.key});

  @override
  ConsumerState<StreakRepairBanner> createState() => _StreakRepairBannerState();
}

class _StreakRepairBannerState extends ConsumerState<StreakRepairBanner> {
  var _busy = false;

  Future<void> _watch(int days) async {
    final l10n = context.l10n;
    setState(() => _busy = true);
    try {
      final (outcome, _) = await ref
          .read(rewardedAdRunnerProvider)
          .watch(AdPlacement.streakRepair);
      if (!mounted) {
        return;
      }
      showSnack(
        context,
        adOutcomeText(l10n, outcome) ?? l10n.adStreakDone(days),
      );
      ref.invalidate(myProgressProvider);
    } on Object catch (e) {
      if (mounted) {
        showSnack(context, playErrorText(l10n, e));
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final config = ref.watch(adsConfigProvider).value;
    if (config == null || !config.streakRepairAvailable) {
      return const SizedBox.shrink();
    }
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final days = config.streakRepairDays;
    return Padding(
      padding: const EdgeInsets.only(top: 14),
      child: BvCard(
        key: const Key('streak-repair'),
        color: bv.gradientEnd.withAlpha(40),
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Icon(Icons.local_fire_department_rounded, color: bv.gradientEnd),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.adStreakTitle(days),
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                  Text(
                    l10n.adStreakSub,
                    style: Theme.of(context).textTheme.bodySmall
                        ?.copyWith(color: bv.muted),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            FilledButton(
              key: const Key('streak-repair-watch'),
              onPressed: _busy ? null : () => _watch(days),
              child: Text(l10n.adWatch),
            ),
          ],
        ),
      ),
    );
  }
}
