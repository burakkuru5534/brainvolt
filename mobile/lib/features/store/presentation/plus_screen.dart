import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_client.dart';
import '../../../core/purchases/purchase_gateway.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../ads/presentation/ads.dart';
import '../../auth/domain/session.dart';
import '../../progress/presentation/quests_screen.dart';
import '../data/api_store_repository.dart';
import '../domain/store_repository.dart';

/// The store; main() installs [RevenueCatGateway], tests keep none.
final purchaseGatewayProvider = Provider<PurchaseGateway>(
  (ref) => const NoPurchaseGateway(),
);

/// The entitlement endpoints. Tests override it with a fake.
final storeRepositoryProvider = Provider<StoreRepository>(
  (ref) => ApiStoreRepository(ref.watch(apiClientProvider)),
);

/// How often the app re-reads entitlements after a purchase (tests: 0).
final plusPollEveryProvider = Provider<Duration>(
  (ref) => const Duration(seconds: 1),
);

String? _userId(Ref ref) => ref.watch(
  sessionControllerProvider.select(
    (s) => switch (s.value) {
      SignedIn(:final session) => session.user.id,
      _ => null,
    },
  ),
);

/// The signed-in user's entitlements (server truth, D-104).
final entitlementsProvider = FutureProvider<Entitlements>((ref) {
  if (_userId(ref) == null) {
    return const Entitlements();
  }
  return ref.watch(storeRepositoryProvider).entitlements();
});

/// The Plus plans with store prices.
final plusOptionsProvider = FutureProvider.autoDispose<List<PlusOption>>((ref) {
  final user = _userId(ref);
  final gateway = ref.watch(purchaseGatewayProvider);
  if (user == null || !gateway.available) {
    return const [];
  }
  return gateway.plusOptions(user);
});

/// Google Play's page for managing this app's subscription.
final _manageUrl = Uri.parse(
  'https://play.google.com/store/account/subscriptions'
  '?sku=brainvolt_plus&package=com.brainvolt.app',
);

/// BrainVolt Plus (D-104): what it gives, the plans, buy and restore.
class PlusScreen extends ConsumerStatefulWidget {
  const PlusScreen({super.key});

  @override
  ConsumerState<PlusScreen> createState() => _PlusScreenState();
}

class _PlusScreenState extends ConsumerState<PlusScreen> {
  String? _selected;
  var _busy = false;

  String get _me => switch (ref.read(sessionControllerProvider).value) {
    SignedIn(:final session) => session.user.id,
    _ => '',
  };

  /// The webhook reaches the server a moment after the store; wait for it.
  Future<bool> _awaitPlus() async {
    final every = ref.read(plusPollEveryProvider);
    final repo = ref.read(storeRepositoryProvider);
    for (var i = 0; i < 12; i++) {
      try {
        if ((await repo.entitlements()).plus) {
          return true;
        }
      } on Object {
        // try again
      }
      await Future<void>.delayed(every);
    }
    return false;
  }

  void _refresh() {
    ref
      ..invalidate(entitlementsProvider)
      ..invalidate(adsConfigProvider)
      ..invalidate(questsProvider);
  }

  Future<void> _run(Future<BuyResult> Function() action) async {
    final l10n = context.l10n;
    setState(() => _busy = true);
    try {
      final result = await action();
      if (!mounted) {
        return;
      }
      switch (result) {
        case BuyResult.purchased:
          final active = await _awaitPlus();
          if (mounted) {
            showSnack(context, active ? l10n.plusWelcome : l10n.plusActivating);
          }
        case BuyResult.pending:
          showSnack(context, l10n.plusPending);
        case BuyResult.failed:
          showSnack(context, l10n.plusFailed);
        case BuyResult.cancelled:
          break;
      }
    } on Object {
      if (mounted) {
        showSnack(context, l10n.plusFailed);
      }
    } finally {
      _refresh();
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _buy(PlusOption option) =>
      _run(() => ref.read(purchaseGatewayProvider).buy(_me, option));

  Future<void> _restore() => _run(() async {
    final active = await ref.read(purchaseGatewayProvider).restore(_me);
    if (!active) {
      if (mounted) {
        showSnack(context, context.l10n.plusNothingToRestore);
      }
      return BuyResult.cancelled;
    }
    return BuyResult.purchased;
  });

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final session = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session,
      _ => null,
    };
    final ent = ref.watch(entitlementsProvider).value ?? const Entitlements();
    final gateway = ref.watch(purchaseGatewayProvider);
    final perks = [
      (Icons.block_rounded, l10n.plusPerkNoAds),
      (Icons.ondemand_video_rounded, l10n.plusPerkRewards),
      (Icons.flag_rounded, l10n.plusPerkQuests),
      (Icons.workspace_premium_rounded, l10n.plusPerkBadge),
    ];
    return Scaffold(
      appBar: AppBar(title: Text(l10n.plusTitle)),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
        children: [
          BvCard(
            gradient: bv.heroGradient,
            padding: const EdgeInsets.all(18),
            child: Row(
              children: [
                Icon(Icons.workspace_premium_rounded, size: 52, color: bv.gold),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.plusTitle,
                        style: text.headlineSmall?.copyWith(
                          color: Colors.white,
                        ),
                      ),
                      Text(
                        l10n.plusTagline,
                        style: text.bodyMedium?.copyWith(color: Colors.white),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          BvCard(
            padding: const EdgeInsets.all(14),
            child: Column(
              children: [
                for (final (icon, label) in perks)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 6),
                    child: Row(
                      children: [
                        IconBadge(icon, size: 38),
                        const SizedBox(width: 12),
                        Expanded(child: Text(label, style: text.bodyLarge)),
                      ],
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          if (ent.plus) ...[
            NoticeBar(
              ent.plusUntil == null
                  ? l10n.plusActive
                  : l10n.plusActiveUntil(
                      DateFormat.yMMMd(
                        Localizations.localeOf(context).toLanguageTag(),
                      ).format(ent.plusUntil!),
                    ),
              key: const Key('plus-active'),
              icon: Icons.check_circle_rounded,
            ),
            const SizedBox(height: 10),
            OutlinedButton.icon(
              key: const Key('plus-manage'),
              icon: const Icon(Icons.open_in_new_rounded),
              label: Text(l10n.plusManage),
              onPressed: () => unawaited(
                launchUrl(_manageUrl, mode: LaunchMode.externalApplication),
              ),
            ),
          ] else if (session?.tier == Tier.guest) ...[
            NoticeBar(
              l10n.plusLinkFirst,
              key: const Key('plus-link-first'),
              icon: Icons.link_rounded,
            ),
            const SizedBox(height: 10),
            GradientButton(
              key: const Key('plus-link-account'),
              label: l10n.linkAccountAction,
              onPressed: () => context.push(Routes.account),
            ),
          ] else if (!gateway.available)
            NoticeBar(l10n.plusUnavailable, key: const Key('plus-unavailable'))
          else
            ..._plans(context),
        ],
      ),
    );
  }

  List<Widget> _plans(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final options = ref.watch(plusOptionsProvider);
    final list = options.value ?? const <PlusOption>[];
    if (options.isLoading) {
      return const [LoadingView()];
    }
    if (list.isEmpty) {
      return [NoticeBar(l10n.plusNoPlans, key: const Key('plus-no-plans'))];
    }
    final chosen = list.firstWhere(
      (o) => o.id == _selected,
      orElse: () => list.first,
    );
    return [
      for (final o in list)
        Padding(
          padding: const EdgeInsets.only(bottom: 10),
          child: BvCard(
            key: Key('plus-option-${o.id}'),
            onTap: () => setState(() => _selected = o.id),
            color: o == chosen ? bv.gradientStart.withAlpha(70) : null,
            padding: const EdgeInsets.all(14),
            child: Row(
              children: [
                Icon(
                  o == chosen
                      ? Icons.radio_button_checked_rounded
                      : Icons.radio_button_off_rounded,
                  color: bv.accent,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        o.id == 'annual' ? l10n.plusAnnual : l10n.plusMonthly,
                        style: text.titleMedium,
                      ),
                      if (o.trialDays > 0)
                        Text(
                          l10n.plusTrial(o.trialDays),
                          style: text.bodySmall?.copyWith(color: bv.success),
                        ),
                    ],
                  ),
                ),
                Text(
                  o.id == 'annual'
                      ? l10n.plusPerYear(o.priceString)
                      : l10n.plusPerMonth(o.priceString),
                  style: text.titleMedium,
                ),
              ],
            ),
          ),
        ),
      const SizedBox(height: 6),
      GradientButton(
        key: const Key('plus-buy'),
        icon: Icons.workspace_premium_rounded,
        label: chosen.trialDays > 0
            ? l10n.plusStartTrial(chosen.trialDays)
            : l10n.plusBuy,
        onPressed: _busy ? null : () => _buy(chosen),
      ),
      TextButton(
        key: const Key('plus-restore'),
        onPressed: _busy ? null : _restore,
        child: Text(l10n.plusRestore),
      ),
      Text(
        l10n.plusLegal,
        textAlign: TextAlign.center,
        style: text.bodySmall?.copyWith(color: bv.muted),
      ),
    ];
  }
}

/// Home: the Plus badge, or the way to it (D-104).
class PlusBadge extends ConsumerWidget {
  const PlusBadge({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final plus = ref.watch(entitlementsProvider).value?.plus ?? false;
    return InkWell(
      key: Key(plus ? 'plus-badge' : 'plus-cta'),
      borderRadius: BorderRadius.circular(20),
      onTap: () => context.push(Routes.plus),
      child: Pill(
        plus ? 'PLUS' : l10n.plusCta,
        icon: Icons.workspace_premium_rounded,
        gradient: plus,
      ),
    );
  }
}
