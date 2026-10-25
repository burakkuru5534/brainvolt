import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/purchases/purchase_gateway.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../auth/domain/session.dart';
import '../domain/store_repository.dart';
import 'cosmetics_tab.dart';
import 'plus_screen.dart';
import 'volt_screen.dart';

/// The Volt packs with store prices (D-107).
final voltOffersProvider = FutureProvider.autoDispose<List<StoreOffer>>((ref) {
  final user = ref.watch(
    sessionControllerProvider.select(
      (s) => switch (s.value) {
        SignedIn(:final session) => session.user.id,
        _ => null,
      },
    ),
  );
  final gateway = ref.watch(purchaseGatewayProvider);
  if (user == null || !gateway.available) {
    return const [];
  }
  return gateway.voltOffers(user);
});

/// The Volt tab's "Buy Volt" section: the starter pack once, then the
/// packs, each with the store's price. Guests link their account first.
class VoltPacksSection extends ConsumerStatefulWidget {
  const VoltPacksSection({required this.wallet, super.key});

  final Wallet wallet;

  @override
  ConsumerState<VoltPacksSection> createState() => _VoltPacksSectionState();
}

class _VoltPacksSectionState extends ConsumerState<VoltPacksSection> {
  var _busy = false;

  /// The webhook reaches the server a moment after the store.
  Future<bool> _awaitVolt(int before) async {
    final every = ref.read(plusPollEveryProvider);
    final repo = ref.read(storeRepositoryProvider);
    for (var i = 0; i < 12; i++) {
      try {
        if ((await repo.wallet()).balance > before) {
          return true;
        }
      } on Object {
        // try again
      }
      await Future<void>.delayed(every);
    }
    return false;
  }

  Future<void> _buy(VoltPack pack, StoreOffer offer) async {
    final l10n = context.l10n;
    final user = switch (ref.read(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    setState(() => _busy = true);
    try {
      final before = widget.wallet.balance;
      final result = await ref
          .read(purchaseGatewayProvider)
          .buyVolt(user, offer);
      if (!mounted) {
        return;
      }
      switch (result) {
        case BuyResult.purchased:
          final arrived = await _awaitVolt(before);
          if (mounted) {
            showSnack(
              context,
              arrived ? l10n.voltPackDone(pack.volt) : l10n.voltPackProcessing,
            );
          }
        case BuyResult.pending:
          showSnack(context, l10n.voltPackProcessing);
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
      ref
        ..invalidate(walletProvider)
        ..invalidate(cosmeticsProvider);
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final gateway = ref.watch(purchaseGatewayProvider);
    final tier = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.tier,
      _ => Tier.guest,
    };
    if (!gateway.available || widget.wallet.packs.isEmpty) {
      return const SizedBox.shrink();
    }
    final children = <Widget>[SectionTitle(l10n.voltPacksTitle)];
    if (tier == Tier.guest) {
      children.addAll([
        NoticeBar(
          l10n.voltPackLinkFirst,
          key: const Key('volt-link-first'),
          icon: Icons.link_rounded,
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          key: const Key('volt-link-account'),
          onPressed: () => context.push(Routes.account),
          child: Text(l10n.linkAccountAction),
        ),
      ]);
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: children,
      );
    }
    final offers = ref.watch(voltOffersProvider);
    if (offers.isLoading) {
      children.add(const LoadingView());
    } else {
      final byId = {
        for (final o in offers.value ?? const <StoreOffer>[]) o.id: o,
      };
      final rows = [
        for (final p in widget.wallet.packs)
          if (byId[p.productId] case final o?) _tile(context, p, o),
      ];
      children.addAll(
        rows.isEmpty
            ? [NoticeBar(l10n.voltPacksNone, key: const Key('volt-packs-none'))]
            : rows,
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: children,
    );
  }

  Widget _tile(BuildContext context, VoltPack pack, StoreOffer offer) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final starter = pack.once;
    final light = starter ? Colors.white : null;
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: BvCard(
        key: Key('volt-pack-${pack.productId}'),
        gradient: starter ? bv.heroGradient : null,
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            if (starter)
              const AvatarBubble('🎁', size: 44, frame: 'frame_starter')
            else
              Icon(voltIcon, size: 40, color: bv.gold),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    starter
                        ? l10n.voltStarterTitle
                        : l10n.voltPackAmount(pack.volt),
                    style: text.titleMedium?.copyWith(color: light),
                  ),
                  if (starter)
                    Text(
                      l10n.voltStarterSub(pack.volt),
                      style: text.bodySmall?.copyWith(color: light),
                    )
                  else if (pack.bonus > 0)
                    Padding(
                      padding: const EdgeInsets.only(top: 4),
                      child: Pill(
                        l10n.voltPackBonus(pack.bonus),
                        icon: Icons.card_giftcard_rounded,
                        color: bv.success,
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            FilledButton(
              key: Key('volt-pack-buy-${pack.productId}'),
              onPressed: _busy ? null : () => _buy(pack, offer),
              child: Text(offer.priceString),
            ),
          ],
        ),
      ),
    );
  }
}
