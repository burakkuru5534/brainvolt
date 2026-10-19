/// The BrainVolt component kit (D-085): cards, icon badges, gradient
/// buttons and tiles shared by every screen.
library;

import 'package:flutter/material.dart';

import '../../app/theme.dart';

/// A rounded card with a soft border; [onTap] makes it a button.
class BvCard extends StatelessWidget {
  const BvCard({
    required this.child,
    this.onTap,
    this.padding = const EdgeInsets.all(16),
    this.color,
    this.gradient,
    this.radius = 20,
    super.key,
  });

  final Widget child;
  final VoidCallback? onTap;
  final EdgeInsetsGeometry padding;
  final Color? color;
  final Gradient? gradient;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final shape = BorderRadius.circular(radius);
    return DecoratedBox(
      decoration: BoxDecoration(
        color: gradient == null ? (color ?? bv.card) : null,
        gradient: gradient,
        borderRadius: shape,
        border: gradient == null && color == null
            ? Border.all(color: bv.cardBorder)
            : null,
        boxShadow: [
          BoxShadow(
            color: gradient != null ? bv.glow : bv.cardBorder.withAlpha(90),
            blurRadius: gradient != null ? 18 : 12,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Material(
        type: MaterialType.transparency,
        child: InkWell(
          borderRadius: shape,
          onTap: onTap,
          child: Padding(padding: padding, child: child),
        ),
      ),
    );
  }
}

/// A round badge holding an icon.
class IconBadge extends StatelessWidget {
  const IconBadge(
    this.icon, {
    this.size = 48,
    this.onGradient = false,
    super.key,
  });

  final IconData icon;
  final double size;

  /// White on translucent white, for gradient backgrounds.
  final bool onGradient;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: onGradient ? Colors.white.withAlpha(46) : bv.badge,
      ),
      child: Icon(
        icon,
        size: size * 0.52,
        color: onGradient ? Colors.white : bv.badgeInk,
      ),
    );
  }
}

/// The primary call to action: a gradient pill with a glow.
class GradientButton extends StatelessWidget {
  const GradientButton({
    required this.label,
    required this.onPressed,
    this.icon,
    this.height = 60,
    super.key,
  });

  final String label;
  final IconData? icon;
  final VoidCallback? onPressed;
  final double height;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final enabled = onPressed != null;
    return Opacity(
      opacity: enabled ? 1 : 0.5,
      child: DecoratedBox(
        decoration: BoxDecoration(
          gradient: bv.gradient,
          borderRadius: BorderRadius.circular(height),
          boxShadow: [
            if (enabled)
              BoxShadow(
                color: bv.glow,
                blurRadius: 20,
                offset: const Offset(0, 8),
              ),
          ],
        ),
        child: Material(
          type: MaterialType.transparency,
          child: InkWell(
            borderRadius: BorderRadius.circular(height),
            onTap: onPressed,
            child: SizedBox(
              height: height,
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  if (icon != null) ...[
                    Icon(icon, color: Colors.white, size: 26),
                    const SizedBox(width: 10),
                  ],
                  Flexible(
                    child: Text(
                      label,
                      overflow: TextOverflow.ellipsis,
                      style: weighted(
                        const TextStyle(
                          fontSize: 20,
                          color: Colors.white,
                          fontFamily: brandFont,
                        ),
                        FontWeight.w900,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// A row card: badge, title, subtitle and a chevron (sheets, menus).
class ActionCard extends StatelessWidget {
  const ActionCard({
    required this.icon,
    required this.title,
    required this.onTap,
    this.subtitle,
    this.trailing,
    super.key,
  });

  final IconData icon;
  final String title;
  final String? subtitle;
  final VoidCallback? onTap;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      onTap: onTap,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      child: Row(
        children: [
          IconBadge(icon, size: 56),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: text.titleMedium),
                if (subtitle != null)
                  Text(
                    subtitle!,
                    style: text.bodyMedium?.copyWith(color: bv.muted),
                  ),
              ],
            ),
          ),
          trailing ?? Icon(Icons.chevron_right, color: bv.ink),
        ],
      ),
    );
  }
}

/// A square choice tile for grids (modes, formats): filled when chosen.
class ChoiceTile extends StatelessWidget {
  const ChoiceTile({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onTap,
    this.subtitle,
    this.vertical = false,
    super.key,
  });

  final IconData icon;
  final String label;
  final String? subtitle;
  final bool selected;
  final VoidCallback? onTap;

  /// Icon above the label (small tiles) instead of beside it.
  final bool vertical;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final ink = selected ? Colors.white : bv.ink;
    final style = weighted(
      TextStyle(
        fontSize: vertical ? 14 : 16,
        height: 1.1,
        color: ink,
        fontFamily: brandFont,
      ),
      FontWeight.w800,
    );
    final iconWidget = Icon(
      icon,
      color: selected ? Colors.white : bv.badgeInk,
      size: 28,
    );
    final body = vertical
        ? Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              iconWidget,
              const SizedBox(height: 6),
              Text(
                label,
                style: style,
                textAlign: TextAlign.center,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              if (subtitle != null)
                Text(
                  subtitle!,
                  textAlign: TextAlign.center,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall
                      ?.copyWith(color: selected ? Colors.white70 : bv.muted),
                ),
            ],
          )
        : Row(
            children: [
              iconWidget,
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  label,
                  style: style,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          );
    return Stack(
      children: [
        Positioned.fill(
          child: BvCard(
            onTap: onTap,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            gradient: selected
                ? LinearGradient(colors: [bv.selected, bv.gradientEnd])
                : null,
            radius: 18,
            child: Center(child: body),
          ),
        ),
        if (selected)
          const Positioned(
            top: 6,
            right: 6,
            child: Icon(Icons.check_circle, color: Colors.white, size: 22),
          ),
      ],
    );
  }
}

/// A heading above a group of tiles.
class SectionTitle extends StatelessWidget {
  const SectionTitle(this.text, {super.key});

  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 20, bottom: 10),
    child: Text(text, style: Theme.of(context).textTheme.titleMedium),
  );
}

/// The word mark: "Brain" in ink, "Volt" in brand purple with a bolt.
class BrandLogo extends StatelessWidget {
  const BrandLogo({this.size = 30, super.key});

  final double size;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final base = weighted(
      TextStyle(fontSize: size, fontFamily: brandFont, height: 1),
      FontWeight.w900,
    );
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          'Brain',
          style: base.copyWith(color: bv.ink, fontStyle: FontStyle.italic),
        ),
        ShaderMask(
          shaderCallback: (r) => bv.gradient.createShader(r),
          child: Text(
            'Volt',
            style: base.copyWith(
              color: Colors.white,
              fontStyle: FontStyle.italic,
            ),
          ),
        ),
        Icon(Icons.bolt, size: size * 0.9, color: bv.gradientStart),
      ],
    );
  }
}

/// A round avatar: the user's artwork (an emoji until it arrives) on a
/// tinted circle, optionally with an online dot.
class AvatarBubble extends StatelessWidget {
  const AvatarBubble(
    this.emoji, {
    this.size = 56,
    this.online = false,
    this.ring = true,
    super.key,
  });

  final String emoji;
  final double size;
  final bool online;
  final bool ring;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return SizedBox(
      width: size,
      height: size,
      child: Stack(
        children: [
          Container(
            width: size,
            height: size,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: bv.badge,
              border: ring
                  ? Border.all(color: bv.gradientStart, width: 2.5)
                  : null,
            ),
            child: Text(emoji, style: TextStyle(fontSize: size * 0.55)),
          ),
          if (online)
            Positioned(
              right: 1,
              bottom: 1,
              child: Container(
                width: size * 0.24,
                height: size * 0.24,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: bv.success,
                  border: Border.all(color: bv.card, width: 2),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
