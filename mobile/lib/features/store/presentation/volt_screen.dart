import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../ads/domain/ads_repository.dart';
import '../../ads/presentation/ads.dart';
import '../../match/presentation/play_error_text.dart';
import '../../progress/presentation/progress_widgets.dart';
import '../../progress/presentation/quests_screen.dart';
import '../domain/store_repository.dart';
import 'cosmetics_tab.dart';
import 'plus_screen.dart';
import 'volt_packs.dart';

/// Volt's icon everywhere (D-105).
const voltIcon = Icons.offline_bolt_rounded;

/// The signed-in user's Volt (server truth, D-105).
final walletProvider = FutureProvider.autoDispose<Wallet>((ref) {
  final user = ref.watch(
    sessionControllerProvider.select(
      (s) => switch (s.value) {
        SignedIn(:final session) => session.user.id,
        _ => null,
      },
    ),
  );
  if (user == null) {
    return const Wallet();
  }
  return ref.watch(storeRepositoryProvider).wallet();
});

/// Home: the Volt balance; opens the store.
class VoltChip extends ConsumerWidget {
  const VoltChip({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final balance = ref.watch(walletProvider).value?.balance ?? 0;
    return InkWell(
      key: const Key('volt-chip'),
      borderRadius: BorderRadius.circular(20),
      onTap: () => context.push(Routes.store),
      child: Pill(
        NumberFormat.decimalPattern(
          Localizations.localeOf(context).toLanguageTag(),
        ).format(balance),
        icon: voltIcon,
        color: BvColors.of(context).accent,
      ),
    );
  }
}

/// Item name for messages.
String voltItemName(AppLocalizations l10n, String code) => switch (code) {
  StoreItemCode.streakShield => l10n.voltItemShield,
  StoreItemCode.questReroll => l10n.voltItemReroll,
  _ => code,
};

/// Buys [code] after asking; refreshes everything it touches. Returns
/// true when bought.
Future<bool> buyVoltItem(
  BuildContext context,
  WidgetRef ref,
  String code,
  int price,
) async {
  final l10n = context.l10n;
  final name = voltItemName(l10n, code);
  final ok = await confirmDialog(
    context,
    title: l10n.voltBuyTitle(name),
    body: l10n.voltBuyBody(price),
    confirmLabel: l10n.voltBuy,
  );
  if (!ok || !context.mounted) {
    return false;
  }
  try {
    await ref.read(storeRepositoryProvider).buy(code);
    if (context.mounted) {
      showSnack(context, l10n.voltBought(name));
    }
    return true;
  } on Object catch (e) {
    if (context.mounted) {
      showSnack(context, playErrorText(l10n, e));
    }
    return false;
  } finally {
    ref
      ..invalidate(walletProvider)
      ..invalidate(myProgressProvider)
      ..invalidate(questsProvider);
  }
}

/// The Volt store (D-105): balance, the daily Volt ad, items, how Volt is
/// earned and the latest movements.
class VoltScreen extends ConsumerStatefulWidget {
  const VoltScreen({this.cosmetics = false, super.key});

  /// Opens on the cosmetics tab.
  final bool cosmetics;

  @override
  ConsumerState<VoltScreen> createState() => _VoltScreenState();
}

class _VoltScreenState extends ConsumerState<VoltScreen> {
  var _busy = false;

  Future<void> _watchAd(int amount) async {
    final l10n = context.l10n;
    setState(() => _busy = true);
    try {
      final (outcome, _) = await ref
          .read(rewardedAdRunnerProvider)
          .watch(AdPlacement.volt);
      if (mounted) {
        showSnack(
          context,
          adOutcomeText(l10n, outcome) ?? l10n.voltAdDone(amount),
        );
      }
    } on Object catch (e) {
      if (mounted) {
        showSnack(context, playErrorText(l10n, e));
      }
    } finally {
      ref.invalidate(walletProvider);
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _buy(String code, int price) async {
    setState(() => _busy = true);
    await buyVoltItem(context, ref, code, price);
    if (mounted) {
      setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final balance = ref.watch(walletProvider).value?.balance;
    return DefaultTabController(
      length: 2,
      initialIndex: widget.cosmetics ? 1 : 0,
      child: Scaffold(
        appBar: AppBar(
          title: Text(l10n.voltTitle),
          actions: [
            if (balance != null)
              Padding(
                padding: const EdgeInsets.only(right: 12),
                child: Pill(
                  '$balance',
                  key: const Key('store-balance'),
                  icon: voltIcon,
                  color: BvColors.of(context).accent,
                ),
              ),
          ],
          bottom: TabBar(
            tabs: [
              Tab(key: const Key('tab-volt'), text: l10n.storeTabVolt),
              Tab(
                key: const Key('tab-cosmetics'),
                text: l10n.storeTabCosmetics,
              ),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            switch (ref.watch(walletProvider)) {
              AsyncData(:final value) => RefreshIndicator(
                onRefresh: () => ref.refresh(walletProvider.future),
                child: _body(context, value),
              ),
              AsyncError(:final error) => ErrorView(
                message: playErrorText(l10n, error),
                retryLabel: l10n.retry,
                onRetry: () => ref.invalidate(walletProvider),
              ),
              _ => const LoadingView(),
            },
            const CosmeticsTab(),
          ],
        ),
      ),
    );
  }

  Widget _body(BuildContext context, Wallet w) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final locale = Localizations.localeOf(context).toLanguageTag();
    final progress = ref.watch(myProgressProvider).value;
    final shields = progress?.shields ?? 0;
    final maxShields = progress?.maxShields ?? 2;
    Widget item({
      required String code,
      required IconData icon,
      required String title,
      required String sub,
      bool enabled = true,
    }) {
      final price = w.priceOf(code);
      return Padding(
        padding: const EdgeInsets.only(bottom: 10),
        child: BvCard(
          key: Key('volt-item-$code'),
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              IconBadge(icon, size: 44),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title, style: text.titleSmall),
                    const SizedBox(height: 2),
                    Text(sub, style: text.bodySmall?.copyWith(color: bv.muted)),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              FilledButton.icon(
                key: Key('volt-buy-$code'),
                icon: const Icon(voltIcon, size: 18),
                onPressed: _busy || !enabled || w.balance < price
                    ? null
                    : () => _buy(code, price),
                label: Text('$price'),
              ),
            ],
          ),
        ),
      );
    }

    String reason(VoltEntry e) => switch (e.reason) {
      'quest' => l10n.voltReasonQuest,
      'daily_challenge' => l10n.voltReasonDaily,
      'level_up' => l10n.voltReasonLevel(e.ref),
      'tournament' => l10n.voltReasonTournament,
      'season' => l10n.voltReasonSeason,
      'ad' => l10n.voltReasonAd,
      'plus_monthly' => l10n.voltReasonPlus,
      'plus_refund' => l10n.voltReasonPlusRefund,
      'item' => l10n.voltReasonItem(voltItemName(l10n, e.ref)),
      'cosmetic' => l10n.voltReasonCosmetic(cosmeticName(l10n, e.ref)),
      'purchase' => l10n.voltReasonPurchase,
      'purchase_refund' => l10n.voltReasonPurchaseRefund,
      'season_pass' => l10n.voltReasonSeasonPass,
      'pass_reward' => l10n.voltReasonPassReward,
      'friend_tournament' => l10n.voltReasonFriendTournament,
      'friend_tournament_refund' => l10n.voltReasonFriendTournamentRefund,
      _ => l10n.voltReasonOther,
    };

    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
      children: [
        BvCard(
          gradient: bv.heroGradient,
          padding: const EdgeInsets.all(18),
          child: Row(
            children: [
              Icon(voltIcon, size: 56, color: bv.gold),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.voltBalanceLabel,
                      style: text.labelLarge?.copyWith(color: Colors.white70),
                    ),
                    Text(
                      NumberFormat.decimalPattern(locale).format(w.balance),
                      key: const Key('volt-balance'),
                      style: text.displaySmall?.copyWith(
                        color: Colors.white,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    Text(
                      l10n.voltTagline,
                      style: text.bodySmall?.copyWith(color: Colors.white),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        BvCard(
          key: const Key('volt-ad'),
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              const IconBadge(Icons.ondemand_video_rounded, size: 44),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(l10n.voltAdTitle(w.adVolt), style: text.titleSmall),
                    Text(
                      w.adVoltLeft > 0
                          ? l10n.voltAdLeft(w.adVoltLeft)
                          : l10n.voltAdNone,
                      style: text.bodySmall?.copyWith(color: bv.muted),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              FilledButton(
                key: const Key('volt-ad-watch'),
                onPressed: _busy || w.adVoltLeft == 0
                    ? null
                    : () => _watchAd(w.adVolt),
                child: Text(l10n.adWatch),
              ),
            ],
          ),
        ),
        VoltPacksSection(wallet: w),
        SectionTitle(l10n.voltShop),
        item(
          code: StoreItemCode.streakShield,
          icon: Icons.shield_rounded,
          title: l10n.voltItemShield,
          sub: l10n.voltItemShieldSub(shields, maxShields),
          enabled: shields < maxShields,
        ),
        item(
          code: StoreItemCode.questReroll,
          icon: Icons.autorenew_rounded,
          title: l10n.voltItemReroll,
          sub: l10n.voltItemRerollSub,
        ),
        SectionTitle(l10n.voltEarnTitle),
        BvCard(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          child: Column(
            children: [
              for (final (icon, label) in [
                (Icons.flag_rounded, l10n.voltEarnQuests),
                (Icons.today_rounded, l10n.voltEarnDaily),
                (Icons.trending_up_rounded, l10n.voltEarnLevel),
                (Icons.emoji_events_rounded, l10n.voltEarnTournament),
                (Icons.military_tech_rounded, l10n.voltEarnSeason),
                (Icons.workspace_premium_rounded, l10n.voltEarnPlus),
              ])
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 6),
                  child: Row(
                    children: [
                      Icon(icon, size: 20, color: bv.accent),
                      const SizedBox(width: 10),
                      Expanded(child: Text(label, style: text.bodyMedium)),
                    ],
                  ),
                ),
            ],
          ),
        ),
        SectionTitle(l10n.voltHistory),
        if (w.history.isEmpty)
          Text(
            l10n.voltHistoryEmpty,
            key: const Key('volt-history-empty'),
            style: text.bodyMedium?.copyWith(color: bv.muted),
          )
        else
          BvCard(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 4),
            child: Column(
              children: [
                for (final e in w.history)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Row(
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(reason(e), style: text.bodyMedium),
                              Text(
                                DateFormat.MMMd(locale)
                                    .add_Hm()
                                    .format(e.createdAt),
                                style: text.bodySmall?.copyWith(
                                  color: bv.muted,
                                ),
                              ),
                            ],
                          ),
                        ),
                        Text(
                          e.amount > 0 ? '+${e.amount}' : '${e.amount}',
                          style: text.titleSmall?.copyWith(
                            color: e.amount > 0 ? bv.success : bv.danger,
                          ),
                        ),
                      ],
                    ),
                  ),
              ],
            ),
          ),
        const SizedBox(height: 12),
        if (ref.watch(entitlementsProvider).value?.plus != true)
          TextButton.icon(
            key: const Key('volt-plus'),
            icon: const Icon(Icons.workspace_premium_rounded),
            onPressed: () => context.push(Routes.plus),
            label: Text(l10n.voltEarnPlus),
          ),
      ],
    );
  }
}
