import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/cosmetics.dart';
import '../../../core/widgets/dialogs.dart';
import '../../auth/domain/session.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/data/profile_providers.dart';
import '../../profile/domain/avatars.dart';
import '../../profile/presentation/profile_controllers.dart';
import '../domain/store_repository.dart';
import 'plus_screen.dart';
import 'volt_screen.dart';

/// The signed-in user's cosmetics (D-106).
final cosmeticsProvider = FutureProvider.autoDispose<MyCosmetics>((ref) {
  final user = ref.watch(
    sessionControllerProvider.select(
      (s) => switch (s.value) {
        SignedIn(:final session) => session.user.id,
        _ => null,
      },
    ),
  );
  if (user == null) {
    return const MyCosmetics();
  }
  return ref.watch(storeRepositoryProvider).cosmetics();
});

/// What the player wears of [kind] ('' for nothing).
String myCosmetic(WidgetRef ref, String kind) =>
    ref.watch(cosmeticsProvider).value?.worn(kind) ?? '';

/// A cosmetic's name.
String cosmeticName(AppLocalizations l10n, String code) => switch (code) {
  'frame_bronze' => l10n.cosFrameBronze,
  'frame_neon' => l10n.cosFrameNeon,
  'frame_flame' => l10n.cosFrameFlame,
  'frame_crown' => l10n.cosFrameCrown,
  'frame_plus' => l10n.cosFramePlus,
  'avatar_octopus' => l10n.cosAvatarOctopus,
  'avatar_eagle' => l10n.cosAvatarEagle,
  'avatar_robot' => l10n.cosAvatarRobot,
  'avatar_alien' => l10n.cosAvatarAlien,
  'avatar_dragon' => l10n.cosAvatarDragon,
  'avatar_unicorn' => l10n.cosAvatarUnicorn,
  'name_gold' => l10n.cosNameGold,
  'name_neon' => l10n.cosNameNeon,
  'name_pink' => l10n.cosNamePink,
  'name_ice' => l10n.cosNameIce,
  'name_rainbow' => l10n.cosNameRainbow,
  'victory_confetti' => l10n.cosVictoryConfetti,
  'victory_fireworks' => l10n.cosVictoryFireworks,
  'victory_lightning' => l10n.cosVictoryLightning,
  'theme_midnight' => l10n.cosThemeMidnight,
  'theme_sunset' => l10n.cosThemeSunset,
  'theme_emerald' => l10n.cosThemeEmerald,
  _ => code,
};

/// The store's cosmetics tab: every kind with a preview, buy, wear and
/// take off.
class CosmeticsTab extends ConsumerStatefulWidget {
  const CosmeticsTab({super.key});

  @override
  ConsumerState<CosmeticsTab> createState() => _CosmeticsTabState();
}

class _CosmeticsTabState extends ConsumerState<CosmeticsTab> {
  var _busy = false;

  ProfileSummary? get _me =>
      switch (ref.read(sessionControllerProvider).value) {
        SignedIn(:final session) => session.profile,
        _ => null,
      };

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
        ..invalidate(cosmeticsProvider)
        ..invalidate(walletProvider);
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _buy(CosmeticItem item) async {
    final l10n = context.l10n;
    final name = cosmeticName(l10n, item.code);
    final ok = await confirmDialog(
      context,
      title: l10n.voltBuyTitle(name),
      body: l10n.voltBuyBody(item.price),
      confirmLabel: l10n.voltBuy,
    );
    if (!ok || !mounted) {
      return;
    }
    await _act(() async {
      await ref.read(storeRepositoryProvider).buyCosmetic(item.code);
      return l10n.cosmeticBought(name);
    });
  }

  Future<void> _wear(CosmeticItem item, {bool off = false}) {
    final l10n = context.l10n;
    return _act(() async {
      if (item.kind == CosmeticKind.avatar) {
        // Special avatars are the profile's avatar (D-106).
        await ref
            .read(profileRepositoryProvider)
            .updateProfile(avatarId: item.avatarId);
        ref.invalidate(meProvider);
        await ref.read(sessionControllerProvider.notifier).refresh();
      } else {
        await ref
            .read(storeRepositoryProvider)
            .equip(item.kind, off ? null : item.code);
      }
      return off ? null : l10n.cosmeticEquipped(cosmeticName(l10n, item.code));
    });
  }

  Future<void> _preview(String code) => showDialog<void>(
    context: context,
    builder: (context) => Dialog(
      backgroundColor: Colors.transparent,
      child: SizedBox(
        height: 360,
        child: Stack(
          children: [
            Positioned.fill(
              child: BvCard(
                child: Center(
                  child: Text(
                    cosmeticName(context.l10n, code),
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                ),
              ),
            ),
            Positioned.fill(child: VictoryCelebration(code: code)),
          ],
        ),
      ),
    ),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return switch (ref.watch(cosmeticsProvider)) {
      AsyncData(:final value) => _body(context, value),
      AsyncError(:final error) => ErrorView(
        message: playErrorText(l10n, error),
        retryLabel: l10n.retry,
        onRetry: () => ref.invalidate(cosmeticsProvider),
      ),
      _ => const LoadingView(),
    };
  }

  Widget _body(BuildContext context, MyCosmetics c) {
    final l10n = context.l10n;
    final me = _me;
    final sections = [
      (CosmeticKind.frame, l10n.cosmeticFrames, l10n.cosmeticFramesSub),
      (CosmeticKind.avatar, l10n.cosmeticAvatars, l10n.cosmeticAvatarsSub),
      (
        CosmeticKind.nameColor,
        l10n.cosmeticNameColors,
        l10n.cosmeticNameColorsSub,
      ),
      (CosmeticKind.victory, l10n.cosmeticVictory, l10n.cosmeticVictorySub),
      (CosmeticKind.matchTheme, l10n.cosmeticThemes, l10n.cosmeticThemesSub),
    ];
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
      children: [
        for (final (kind, title, sub) in sections) ...[
          SectionTitle(title),
          Padding(
            padding: const EdgeInsets.only(bottom: 10),
            child: Text(
              sub,
              style: Theme.of(context).textTheme.bodySmall
                  ?.copyWith(color: BvColors.of(context).muted),
            ),
          ),
          LayoutBuilder(
            builder: (context, box) {
              final w = (box.maxWidth - 10) / 2;
              return Wrap(
                spacing: 10,
                runSpacing: 10,
                children: [
                  for (final item in c.ofKind(kind))
                    SizedBox(width: w, child: _tile(context, item, c, me)),
                ],
              );
            },
          ),
        ],
      ],
    );
  }

  Widget _tile(
    BuildContext context,
    CosmeticItem item,
    MyCosmetics c,
    ProfileSummary? me,
  ) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final wearing = item.kind == CosmeticKind.avatar
        ? me?.avatarId == item.avatarId
        : item.equipped;
    final Widget action;
    if (item.plusOnly && !c.plus) {
      action = OutlinedButton.icon(
        key: Key('cos-plus-${item.code}'),
        icon: const Icon(Icons.workspace_premium_rounded, size: 16),
        onPressed: () => context.push(Routes.plus),
        label: Text(l10n.cosmeticPlusOnly),
      );
    } else if (!item.owned) {
      final balance = ref.watch(walletProvider).value?.balance ?? 0;
      action = FilledButton.icon(
        key: Key('cos-buy-${item.code}'),
        icon: const Icon(voltIcon, size: 16),
        onPressed: _busy || balance < item.price ? null : () => _buy(item),
        label: Text('${item.price}'),
      );
    } else if (wearing) {
      action = item.kind == CosmeticKind.avatar
          ? Pill(
              l10n.cosmeticWorn,
              icon: Icons.check_rounded,
              color: bv.success,
            )
          : OutlinedButton(
              key: Key('cos-off-${item.code}'),
              onPressed: _busy ? null : () => _wear(item, off: true),
              child: Text(l10n.cosmeticRemove),
            );
    } else {
      action = FilledButton.tonal(
        key: Key('cos-use-${item.code}'),
        onPressed: _busy ? null : () => _wear(item),
        child: Text(l10n.cosmeticUse),
      );
    }
    return BvCard(
      key: Key('cos-${item.code}'),
      color: wearing ? bv.success.withAlpha(30) : null,
      padding: const EdgeInsets.all(12),
      child: Column(
        children: [
          SizedBox(
            height: 76,
            child: Center(child: _preview2(context, item, me)),
          ),
          const SizedBox(height: 6),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Flexible(
                child: Text(
                  cosmeticName(l10n, item.code),
                  style: text.titleSmall,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (wearing) ...[
                const SizedBox(width: 4),
                Icon(Icons.check_circle_rounded, size: 16, color: bv.success),
              ],
            ],
          ),
          const SizedBox(height: 8),
          SizedBox(width: double.infinity, child: action),
        ],
      ),
    );
  }

  Widget _preview2(
    BuildContext context,
    CosmeticItem item,
    ProfileSummary? me,
  ) {
    final text = Theme.of(context).textTheme;
    final avatar = avatarFor(me?.avatarId ?? '');
    return switch (item.kind) {
      CosmeticKind.frame => AvatarBubble(avatar, size: 58, frame: item.code),
      CosmeticKind.avatar => AvatarBubble(avatarFor(item.avatarId), size: 58),
      CosmeticKind.nameColor => CosmeticName(
        me?.displayName ?? 'BrainVolt',
        nameColor: item.code,
        style: text.titleLarge,
        textAlign: TextAlign.center,
      ),
      CosmeticKind.victory => TextButton.icon(
        key: Key('cos-preview-${item.code}'),
        icon: Icon(switch (victoryKind(item.code)) {
          VictoryKind.fireworks => Icons.celebration_rounded,
          VictoryKind.lightning => Icons.bolt_rounded,
          _ => Icons.auto_awesome_rounded,
        }, size: 30),
        onPressed: () => _preview(item.code),
        label: Text(context.l10n.cosmeticPreview),
      ),
      CosmeticKind.matchTheme => _ThemeSwatch(matchThemeStyle(item.code)),
      _ => const SizedBox.shrink(),
    };
  }
}

/// A small picture of a match theme: background with two answer buttons.
class _ThemeSwatch extends StatelessWidget {
  const _ThemeSwatch(this.style);

  final MatchThemeStyle? style;

  @override
  Widget build(BuildContext context) {
    final s = style;
    if (s == null) {
      return const SizedBox.shrink();
    }
    Widget bar(Decoration d) => Container(
      height: 16,
      margin: const EdgeInsets.symmetric(vertical: 3),
      decoration: d,
    );
    return Container(
      width: 120,
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: s.background,
        ),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          bar(
            BoxDecoration(
              color: s.option,
              borderRadius: BorderRadius.circular(8),
            ),
          ),
          bar(
            BoxDecoration(
              gradient: LinearGradient(colors: s.chosen),
              borderRadius: BorderRadius.circular(8),
            ),
          ),
        ],
      ),
    );
  }
}
