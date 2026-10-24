import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';

/// The four main destinations of the bottom bar (D-099).
enum MainTab { home, tournaments, leaderboards, profile }

/// The WitClash bottom bar: Home, Tournaments, Leaderboards, Profile. Each
/// destination screen shows it; a tap goes to that screen as the new root.
/// On a tab other than Home with nothing to pop, Back returns to Home.
class MainNavBar extends StatelessWidget {
  const MainNavBar({required this.current, super.key});

  final MainTab current;

  static const _routes = {
    MainTab.home: Routes.home,
    MainTab.tournaments: Routes.tournaments,
    MainTab.leaderboards: Routes.leaderboards,
    MainTab.profile: Routes.profile,
  };

  // The tournament and leaderboard keys are the ones Home's shortcuts had.
  static const _keys = {
    MainTab.home: Key('nav-home'),
    MainTab.tournaments: Key('open-tournaments'),
    MainTab.leaderboards: Key('open-leaderboards'),
    MainTab.profile: Key('nav-profile'),
  };

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final items = [
      (MainTab.home, Icons.home_rounded, l10n.navHome),
      (MainTab.tournaments, Icons.emoji_events_rounded, l10n.tournamentsTitle),
      (MainTab.leaderboards, Icons.leaderboard_rounded, l10n.navLeaderboards),
      (MainTab.profile, Icons.person_rounded, l10n.profileTitle),
    ];
    final bar = SafeArea(
      top: false,
      minimum: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 4, 12, 0),
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: const Color(0xF0140F38),
            borderRadius: BorderRadius.circular(28),
            border: Border.all(color: bv.cardBorder),
            boxShadow: const [
              BoxShadow(
                color: Color(0x66000000),
                blurRadius: 24,
                offset: Offset(0, 8),
              ),
            ],
          ),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 8),
            child: Row(
              children: [
                for (final (tab, icon, label) in items)
                  Expanded(
                    child: _NavItem(
                      key: _keys[tab],
                      icon: icon,
                      label: label,
                      selected: tab == current,
                      onTap: tab == current
                          ? null
                          : () => context.go(_routes[tab]!),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
    if (current == MainTab.home) {
      return bar;
    }
    return PopScope(
      canPop: Navigator.of(context).canPop(),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) {
          context.go(Routes.home);
        }
      },
      child: bar,
    );
  }
}

class _NavItem extends StatelessWidget {
  const _NavItem({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onTap,
    super.key,
  });

  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Semantics(
      selected: selected,
      button: true,
      child: InkWell(
        borderRadius: BorderRadius.circular(20),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 2),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                width: 52,
                height: 32,
                decoration: BoxDecoration(
                  gradient: selected ? bv.gradient : null,
                  borderRadius: BorderRadius.circular(16),
                  boxShadow: [
                    if (selected) BoxShadow(color: bv.glow, blurRadius: 12),
                  ],
                ),
                child: Icon(
                  icon,
                  size: 22,
                  color: selected ? Colors.white : bv.muted,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: weighted(
                  TextStyle(
                    fontSize: 11.5,
                    fontFamily: brandFont,
                    color: selected ? Colors.white : bv.muted,
                  ),
                  FontWeight.w800,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
