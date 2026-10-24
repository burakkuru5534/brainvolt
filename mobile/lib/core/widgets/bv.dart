/// The BrainVolt component kit (D-085, restyled in D-099): glass cards,
/// icon badges, gradient buttons, tiles and the brand mark shared by every
/// screen.
library;

import 'package:flutter/material.dart';

import '../../app/theme.dart';

/// A frosted glass card with a hairline border; [onTap] makes it a
/// button. [color] or [gradient] replace the glass fill.
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
    final glass = gradient == null && color == null;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: glass ? null : color,
        gradient:
            gradient ??
            (glass
                ? const LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [Color(0x24FFFFFF), Color(0x0FFFFFFF)],
                  )
                : null),
        borderRadius: shape,
        border: Border.all(
          color: gradient != null ? Colors.white.withAlpha(50) : bv.cardBorder,
        ),
        boxShadow: [
          BoxShadow(
            color: gradient != null ? bv.glow : const Color(0x40000000),
            blurRadius: gradient != null ? 22 : 18,
            offset: const Offset(0, 6),
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
        border: Border.all(
          color: onGradient
              ? Colors.white.withAlpha(60)
              : bv.badgeInk.withAlpha(50),
        ),
      ),
      child: Icon(
        icon,
        size: size * 0.52,
        color: onGradient ? Colors.white : bv.badgeInk,
      ),
    );
  }
}

/// The primary call to action: a purple → orange pill with a glow.
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
            gradient: selected ? bv.heroGradient : null,
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

/// A heading above a group of tiles; [action] adds a link on the right.
class SectionTitle extends StatelessWidget {
  const SectionTitle(this.text, {this.action, this.onAction, super.key});

  final String text;
  final String? action;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    final title = Text(text, style: Theme.of(context).textTheme.titleMedium);
    return Padding(
      padding: const EdgeInsets.only(top: 20, bottom: 10),
      child: action == null
          ? title
          : Row(
              children: [
                Expanded(child: title),
                InkWell(
                  borderRadius: BorderRadius.circular(12),
                  onTap: onAction,
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 6,
                      vertical: 2,
                    ),
                    child: Text(
                      action!,
                      style: Theme.of(context).textTheme.labelLarge
                          ?.copyWith(color: BvColors.of(context).accent),
                    ),
                  ),
                ),
              ],
            ),
    );
  }
}

/// A rounded progress bar filled with the brand gradient (XP, timers).
class GradientProgressBar extends StatelessWidget {
  const GradientProgressBar({
    required this.value,
    this.height = 8,
    this.gradient,
    super.key,
  });

  /// 0–1.
  final double value;
  final double height;
  final Gradient? gradient;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final v = value.isNaN ? 0.0 : value.clamp(0.0, 1.0);
    return Semantics(
      value: '${(v * 100).round()}%',
      child: Container(
        height: height,
        decoration: BoxDecoration(
          color: Colors.white.withAlpha(28),
          borderRadius: BorderRadius.circular(height),
        ),
        alignment: Alignment.centerLeft,
        child: FractionallySizedBox(
          widthFactor: v,
          heightFactor: 1,
          child: DecoratedBox(
            decoration: BoxDecoration(
              gradient: gradient ?? bv.gradient,
              borderRadius: BorderRadius.circular(height),
              boxShadow: [
                BoxShadow(color: bv.gradientEnd.withAlpha(90), blurRadius: 8),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// A small pill: a label with an optional icon (levels, tags, states).
class Pill extends StatelessWidget {
  const Pill(
    this.label, {
    this.icon,
    this.gradient = false,
    this.color,
    super.key,
  });

  final String label;
  final IconData? icon;

  /// Brand gradient fill instead of the tinted glass.
  final bool gradient;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final ink = gradient ? Colors.white : (color ?? bv.badgeInk);
    final style = Theme.of(context).textTheme.labelMedium?.copyWith(color: ink);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        gradient: gradient ? bv.gradient : null,
        color: gradient ? null : (color ?? bv.badgeInk).withAlpha(40),
        borderRadius: BorderRadius.circular(20),
        border: gradient ? null : Border.all(color: ink.withAlpha(70)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 15, color: ink),
            const SizedBox(width: 3),
          ],
          Flexible(
            child: Text(
              label,
              style: style,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

/// The BrainVolt mark: "BV" on the brand gradient, split by a bolt. The
/// same drawing as `assets/brand/brainvolt_logo.svg` (100 × 100 units).
class BrandMark extends StatelessWidget {
  const BrandMark({this.size = 40, super.key});

  final double size;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(size * 0.28),
        boxShadow: [BoxShadow(color: bv.glow, blurRadius: size * 0.4)],
      ),
      child: CustomPaint(
        size: Size.square(size),
        painter: _BrandMarkPainter(bv),
      ),
    );
  }
}

class _BrandMarkPainter extends CustomPainter {
  _BrandMarkPainter(this.bv);

  final BvColors bv;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.scale(size.width / 100, size.height / 100);
    const box = Rect.fromLTWH(0, 0, 100, 100);
    final tile = RRect.fromRectAndRadius(box, const Radius.circular(28));
    canvas
      ..drawRRect(
        tile,
        Paint()
          ..shader = LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [bv.gradientStart, bv.gradientMid, bv.gradientEnd],
            stops: const [0, 0.55, 1],
          ).createShader(box),
      )
      ..drawRRect(
        tile,
        Paint()
          ..shader = const LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [Color(0x47FFFFFF), Color(0x00FFFFFF)],
            stops: [0, 0.5],
          ).createShader(box),
      );

    // "BV", slanted like the word mark.
    final letters = Path()
      ..moveTo(20, 27)
      ..lineTo(20, 73)
      ..moveTo(20, 27)
      ..lineTo(36, 27)
      ..cubicTo(44.5, 27, 49, 31.5, 49, 38.25)
      ..cubicTo(49, 45, 44.5, 50, 36, 50)
      ..lineTo(20, 50)
      ..moveTo(20, 50)
      ..lineTo(38, 50)
      ..cubicTo(47, 50, 52, 55, 52, 61.5)
      ..cubicTo(52, 68, 47, 73, 38, 73)
      ..lineTo(20, 73)
      ..moveTo(60, 27)
      ..lineTo(71, 73)
      ..lineTo(83, 27);
    canvas
      ..save()
      ..translate(50, 50)
      ..transform(Matrix4.skewX(-0.13963).storage)
      ..translate(-50, -50)
      ..drawPath(
        letters,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = 9
          ..strokeCap = StrokeCap.round
          ..strokeJoin = StrokeJoin.round
          ..color = Colors.white,
      )
      ..restore();

    final bolt = Path()
      ..moveTo(60, 4)
      ..lineTo(49, 30)
      ..lineTo(56, 30)
      ..lineTo(51, 52)
      ..lineTo(67, 24)
      ..lineTo(60, 24)
      ..lineTo(65, 4)
      ..close();
    canvas
      ..drawPath(
        bolt,
        Paint()
          ..shader = const LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [Color(0xFFFFE27A), Color(0xFFFBBF24)],
          ).createShader(const Rect.fromLTWH(49, 4, 18, 48)),
      )
      ..drawPath(
        bolt,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2.2
          ..strokeJoin = StrokeJoin.round
          ..color = const Color(0xFF3B1A6B),
      );
  }

  @override
  bool shouldRepaint(_BrandMarkPainter old) => old.bv != bv;
}

/// The word mark: "Brain" in white, "Volt" in the purple → orange
/// gradient, heavy italic; [mark] puts the [BrandMark] in front.
class BrandLogo extends StatelessWidget {
  const BrandLogo({this.size = 30, this.mark = true, super.key});

  final double size;
  final bool mark;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final base = weighted(
      TextStyle(
        fontSize: size,
        fontFamily: brandFont,
        height: 1.15,
        fontStyle: FontStyle.italic,
        letterSpacing: -0.5,
      ),
      FontWeight.w900,
    );
    return Semantics(
      label: 'BrainVolt',
      excludeSemantics: true,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (mark) ...[
            BrandMark(size: size * 1.25),
            SizedBox(width: size * 0.3),
          ],
          Text('Brain', style: base.copyWith(color: bv.ink)),
          ShaderMask(
            shaderCallback: (r) =>
                LinearGradient(colors: [bv.gradientMid, bv.gradientEnd])
                    .createShader(r),
            child: Text(
              // Room for the slanted last letter.
              'Volt ',
              style: base.copyWith(color: Colors.white),
            ),
          ),
        ],
      ),
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
            padding: EdgeInsets.all(ring ? 2.5 : 0),
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              gradient: ring ? bv.gradient : null,
            ),
            child: Container(
              alignment: Alignment.center,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: ring ? bv.surface : bv.badge,
              ),
              child: Text(
                emoji,
                style: TextStyle(fontSize: size * (ring ? 0.5 : 0.55)),
              ),
            ),
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
                  border: Border.all(color: bv.surface, width: 2),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// How an answer option looks.
enum OptionLook { idle, chosen, proposed, correct, wrong }

/// An answer option: a letter badge, the text and, when chosen or
/// revealed, a filled colour with a mark (D-085).
class OptionTile extends StatelessWidget {
  const OptionTile({
    required this.letter,
    required this.text,
    required this.onPressed,
    this.look = OptionLook.idle,
    this.footer,
    super.key,
  });

  final String letter;
  final String text;
  final OptionLook look;
  final VoidCallback? onPressed;

  /// Extra lines under the text (teammates' proposals).
  final Widget? footer;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final filled = look != OptionLook.idle && look != OptionLook.proposed;
    final ink = filled ? Colors.white : bv.ink;
    final gradient = switch (look) {
      OptionLook.chosen => bv.gradient,
      OptionLook.correct => LinearGradient(
        colors: [bv.success, bv.success.withAlpha(220)],
      ),
      OptionLook.wrong => LinearGradient(
        colors: [bv.danger, bv.danger.withAlpha(220)],
      ),
      _ => null,
    };
    final mark = switch (look) {
      OptionLook.chosen || OptionLook.correct => Icons.check_circle,
      OptionLook.wrong => Icons.cancel,
      _ => null,
    };
    return Opacity(
      opacity: onPressed == null && look == OptionLook.idle ? 0.75 : 1,
      child: BvCard(
        onTap: onPressed,
        gradient: gradient,
        color: look == OptionLook.proposed
            ? bv.gradientStart.withAlpha(60)
            : null,
        radius: 18,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
        child: Row(
          children: [
            Container(
              width: 40,
              height: 40,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: filled ? Colors.white.withAlpha(50) : bv.badge,
              ),
              child: Text(
                letter,
                style: weighted(
                  TextStyle(
                    fontSize: 18,
                    color: filled ? Colors.white : bv.badgeInk,
                    fontFamily: brandFont,
                  ),
                  FontWeight.w900,
                ),
              ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    text,
                    style: weighted(
                      TextStyle(
                        fontSize: 18,
                        color: ink,
                        fontFamily: brandFont,
                      ),
                      FontWeight.w800,
                    ),
                  ),
                  ?footer,
                ],
              ),
            ),
            if (mark != null) Icon(mark, color: Colors.white, size: 30),
          ],
        ),
      ),
    );
  }
}

/// An empty list: a large icon badge over a short line.
class EmptyState extends StatelessWidget {
  const EmptyState({
    required this.icon,
    required this.text,
    this.textKey,
    super.key,
  });

  final IconData icon;
  final String text;
  final Key? textKey;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconBadge(icon, size: 88),
          const SizedBox(height: 16),
          Text(
            text,
            key: textKey,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.titleMedium
                ?.copyWith(color: BvColors.of(context).muted),
          ),
        ],
      ),
    ),
  );
}

/// A titled group of rows (settings, menus): a glass card holding
/// [children] separated by hairlines; [title] sits above it.
class GlassSection extends StatelessWidget {
  const GlassSection({required this.children, this.title, super.key});

  final String? title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (title != null)
          Padding(
            padding: const EdgeInsets.fromLTRB(6, 14, 6, 8),
            child: Text(
              title!.toUpperCase(),
              style: Theme.of(context).textTheme.labelMedium
                  ?.copyWith(color: bv.muted, letterSpacing: 1.1),
            ),
          ),
        BvCard(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              for (final (i, c) in children.indexed) ...[
                if (i > 0)
                  Divider(
                    height: 1,
                    indent: 16,
                    endIndent: 16,
                    color: bv.cardBorder,
                  ),
                c,
              ],
            ],
          ),
        ),
      ],
    );
  }
}

/// A pill-shaped segmented control: the chosen segment is filled with the
/// brand gradient. Each segment gets the key `'$keyPrefix-${value.name}'`
/// when [keyPrefix] is set and values are enums.
class WcSegmented<T> extends StatelessWidget {
  const WcSegmented({
    required this.segments,
    required this.selected,
    required this.onChanged,
    this.keyPrefix,
    super.key,
  });

  final List<(T, String)> segments;
  final T selected;
  final ValueChanged<T> onChanged;
  final String? keyPrefix;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final style = Theme.of(context).textTheme.labelLarge;
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: Colors.white.withAlpha(18),
        borderRadius: BorderRadius.circular(30),
        border: Border.all(color: bv.cardBorder),
      ),
      child: Row(
        children: [
          for (final (value, label) in segments)
            Expanded(
              child: GestureDetector(
                key: keyPrefix == null
                    ? null
                    : Key('$keyPrefix-${value is Enum ? value.name : value}'),
                behavior: HitTestBehavior.opaque,
                onTap: value == selected ? null : () => onChanged(value),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  decoration: BoxDecoration(
                    gradient: value == selected ? bv.gradient : null,
                    borderRadius: BorderRadius.circular(26),
                    boxShadow: [
                      if (value == selected)
                        BoxShadow(color: bv.glow, blurRadius: 10),
                    ],
                  ),
                  child: Text(
                    label,
                    textAlign: TextAlign.center,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: style?.copyWith(
                      color: value == selected ? Colors.white : bv.muted,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// A stat in a glass box: a tinted icon, a big value and a small label.
class StatBox extends StatelessWidget {
  const StatBox({
    required this.icon,
    required this.value,
    required this.label,
    this.color,
    this.valueKey,
    super.key,
  });

  final IconData icon;
  final String value;
  final String label;
  final Color? color;
  final Key? valueKey;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final tint = color ?? bv.accent;
    return BvCard(
      radius: 18,
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 12),
      child: Column(
        children: [
          Icon(icon, color: tint, size: 24),
          const SizedBox(height: 4),
          FittedBox(
            fit: BoxFit.scaleDown,
            child: Text(value, key: valueKey, style: text.titleLarge),
          ),
          Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
            style: text.labelSmall,
          ),
        ],
      ),
    );
  }
}

/// Async states in one place: a spinner, or an error line with a retry.
class LoadingView extends StatelessWidget {
  const LoadingView({super.key});

  @override
  Widget build(BuildContext context) =>
      const Center(child: CircularProgressIndicator());
}

class ErrorView extends StatelessWidget {
  const ErrorView({
    required this.message,
    this.onRetry,
    this.retryLabel,
    super.key,
  });

  final String message;
  final VoidCallback? onRetry;
  final String? retryLabel;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: bv.danger.withAlpha(40),
                border: Border.all(color: bv.danger.withAlpha(90)),
              ),
              child: Icon(
                Icons.error_outline_rounded,
                color: bv.danger,
                size: 34,
              ),
            ),
            const SizedBox(height: 14),
            Text(
              message,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyLarge,
            ),
            if (onRetry != null && retryLabel != null) ...[
              const SizedBox(height: 12),
              OutlinedButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh_rounded),
                label: Text(retryLabel!),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// A countdown ring: a gradient arc for the time left ([value] 0–1) around
/// [child] (usually the seconds).
class CountdownRing extends StatelessWidget {
  const CountdownRing({
    required this.value,
    required this.child,
    this.size = 64,
    this.stroke = 6,
    super.key,
  });

  final double value;
  final Widget child;
  final double size;
  final double stroke;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final v = value.isNaN ? 0.0 : value.clamp(0.0, 1.0);
    return Semantics(
      value: '${(v * 100).round()}%',
      child: SizedBox(
        width: size,
        height: size,
        child: CustomPaint(
          painter: _RingPainter(
            value: v,
            stroke: stroke,
            track: Colors.white.withAlpha(30),
            colors: [bv.gradientStart, bv.gradientMid, bv.gradientEnd],
            fill: bv.surface,
            urgent: v < 0.25 ? bv.danger : null,
          ),
          child: Center(child: child),
        ),
      ),
    );
  }
}

class _RingPainter extends CustomPainter {
  _RingPainter({
    required this.value,
    required this.stroke,
    required this.track,
    required this.colors,
    required this.fill,
    this.urgent,
  });

  final double value;
  final double stroke;
  final Color track;
  final List<Color> colors;
  final Color fill;
  final Color? urgent;

  @override
  void paint(Canvas canvas, Size size) {
    final c = size.center(Offset.zero);
    final r = size.shortestSide / 2 - stroke / 2;
    final rect = Rect.fromCircle(center: c, radius: r);
    canvas
      ..drawCircle(c, r, Paint()..color = fill)
      ..drawCircle(
        c,
        r,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = stroke
          ..color = track,
      );
    if (value <= 0) {
      return;
    }
    final arc = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = stroke
      ..strokeCap = StrokeCap.round;
    if (urgent != null) {
      arc.color = urgent!;
    } else {
      arc.shader = SweepGradient(
        startAngle: -1.5708,
        endAngle: 4.7124,
        colors: colors,
        transform: const GradientRotation(-1.5708),
      ).createShader(rect);
    }
    canvas.drawArc(rect, -1.5708, 6.2832 * value, false, arc);
  }

  @override
  bool shouldRepaint(_RingPainter old) =>
      old.value != value || old.urgent != urgent;
}

/// A one-line notice in a tinted pill (reconnecting, opponent left …).
class NoticeBar extends StatelessWidget {
  const NoticeBar(
    this.text, {
    this.icon = Icons.info_outline_rounded,
    super.key,
  });

  final String text;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(16, 6, 16, 0),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 9),
      decoration: BoxDecoration(
        color: bv.accent.withAlpha(36),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: bv.accent.withAlpha(90)),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 18, color: bv.accent),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              text,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.labelLarge,
            ),
          ),
        ],
      ),
    );
  }
}

/// The big brand block for splash and welcome: the mark, the word mark
/// and the tagline.
class BrandHero extends StatelessWidget {
  const BrandHero({this.tagline, super.key});

  final String? tagline;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        const BrandMark(size: 96),
        const SizedBox(height: 18),
        const FittedBox(child: BrandLogo(size: 44, mark: false)),
        if (tagline != null) ...[
          const SizedBox(height: 8),
          Text(
            tagline!,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.labelLarge
                ?.copyWith(color: bv.muted, letterSpacing: 2),
          ),
        ],
      ],
    );
  }
}

/// A list row ([child], usually a ListTile) on a glass card, with the gap
/// below it; [highlight] tints it (selected, mine, unread).
class TileCard extends StatelessWidget {
  const TileCard({required this.child, this.highlight, super.key});

  final Widget child;
  final Color? highlight;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 8),
    child: BvCard(
      radius: 18,
      padding: const EdgeInsets.symmetric(vertical: 2),
      color: highlight?.withAlpha(40),
      child: child,
    ),
  );
}
