/// Cosmetics drawn in code (D-106): avatar frames, name colours, victory
/// animations and match themes. Codes come from the server's catalogue;
/// an unknown code (a newer server) draws nothing special.
library;

import 'dart:math' as math;

import 'package:flutter/material.dart';

/// Off in widget tests, where an endless animation never settles.
bool cosmeticAnimations = true;

/// Special avatars sold as cosmetics (`avatar_<id>`).
const specialAvatarEmoji = <String, String>{
  'octopus': '🐙',
  'eagle': '🦅',
  'robot': '🤖',
  'alien': '👽',
  'dragon': '🐉',
  'stormcloud': '🌩️',
  'unicorn': '🦄',
};

/// How an avatar frame looks.
class FrameStyle {
  const FrameStyle({
    required this.gradient,
    this.glow,
    this.width = 3.5,
    this.crown = false,
  });

  final Gradient gradient;
  final Color? glow;
  final double width;
  final bool crown;
}

/// The frame of [code], or null for none.
FrameStyle? frameStyle(String code) => switch (code) {
  'frame_bronze' => const FrameStyle(
    gradient: LinearGradient(
      colors: [Color(0xFFE2A46B), Color(0xFF9C5B2E), Color(0xFFD08A4E)],
    ),
  ),
  'frame_neon' => const FrameStyle(
    gradient: LinearGradient(colors: [Color(0xFF3AF6FF), Color(0xFF00B7FF)]),
    glow: Color(0xFF26E6FF),
  ),
  'frame_flame' => const FrameStyle(
    gradient: SweepGradient(
      colors: [
        Color(0xFFFFD34E),
        Color(0xFFFF7A1A),
        Color(0xFFE5242B),
        Color(0xFFFF7A1A),
        Color(0xFFFFD34E),
      ],
    ),
    glow: Color(0xFFFF6A1A),
    width: 4,
  ),
  'frame_crown' => const FrameStyle(
    gradient: LinearGradient(
      begin: Alignment.topLeft,
      end: Alignment.bottomRight,
      colors: [Color(0xFFFFF1A8), Color(0xFFF5B700), Color(0xFFB07A00)],
    ),
    glow: Color(0xFFF5C400),
    width: 4,
    crown: true,
  ),
  'frame_s1_storm' => const FrameStyle(
    gradient: SweepGradient(
      colors: [
        Color(0xFF5B8CFF),
        Color(0xFFB36BFF),
        Color(0xFFFFF04D),
        Color(0xFF5B8CFF),
      ],
    ),
    glow: Color(0xFF7C9BFF),
    width: 4.5,
  ),
  'frame_starter' => const FrameStyle(
    gradient: LinearGradient(
      begin: Alignment.topLeft,
      end: Alignment.bottomRight,
      colors: [Color(0xFF7CFFCB), Color(0xFF1FB8A6), Color(0xFF7C5CFF)],
    ),
    glow: Color(0xFF3FE0C0),
    width: 4,
  ),
  'frame_plus' => const FrameStyle(
    gradient: SweepGradient(
      colors: [
        Color(0xFF8E44FF),
        Color(0xFFF5C400),
        Color(0xFFFF4FA8),
        Color(0xFF8E44FF),
      ],
    ),
    glow: Color(0xFFB36BFF),
    width: 4,
  ),
  _ => null,
};

/// A name colour: one colour or a gradient.
class NameStyle {
  const NameStyle({this.color, this.gradient, this.animated = false});

  final Color? color;
  final Gradient? gradient;
  final bool animated;
}

const _rainbow = [
  Color(0xFFFF4D4D),
  Color(0xFFFFB13D),
  Color(0xFFFFF04D),
  Color(0xFF4DFF88),
  Color(0xFF4DC3FF),
  Color(0xFFB36BFF),
  Color(0xFFFF4D4D),
];

/// The name colour of [code], or null for the default.
NameStyle? nameStyle(String code) => switch (code) {
  'name_gold' => const NameStyle(
    gradient: LinearGradient(colors: [Color(0xFFFFE27A), Color(0xFFF5B700)]),
  ),
  'name_neon' => const NameStyle(color: Color(0xFF39FF88)),
  'name_pink' => const NameStyle(color: Color(0xFFFF5FB8)),
  'name_ice' => const NameStyle(color: Color(0xFF8FE3FF)),
  'name_s1_electric' => const NameStyle(
    gradient: LinearGradient(
      colors: [Color(0xFF6FD3FF), Color(0xFF8E6BFF), Color(0xFF6FD3FF)],
    ),
  ),
  'name_rainbow' => const NameStyle(
    gradient: LinearGradient(colors: _rainbow),
    animated: true,
  ),
  _ => null,
};

/// A player's name in their name colour (D-106).
class CosmeticName extends StatefulWidget {
  const CosmeticName(
    this.name, {
    required this.nameColor,
    this.style,
    this.maxLines = 1,
    this.textAlign,
    super.key,
  });

  final String name;
  final String nameColor;
  final TextStyle? style;
  final int maxLines;
  final TextAlign? textAlign;

  @override
  State<CosmeticName> createState() => _CosmeticNameState();
}

class _CosmeticNameState extends State<CosmeticName>
    with SingleTickerProviderStateMixin {
  AnimationController? _shift;

  @override
  void initState() {
    super.initState();
    _sync();
  }

  @override
  void didUpdateWidget(CosmeticName old) {
    super.didUpdateWidget(old);
    if (old.nameColor != widget.nameColor) {
      _sync();
    }
  }

  void _sync() {
    final animate =
        cosmeticAnimations && (nameStyle(widget.nameColor)?.animated ?? false);
    if (animate && _shift == null) {
      _shift = AnimationController(
        vsync: this,
        duration: const Duration(seconds: 4),
      )..repeat();
    } else if (!animate && _shift != null) {
      _shift!.dispose();
      _shift = null;
    }
  }

  @override
  void dispose() {
    _shift?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final ns = nameStyle(widget.nameColor);
    final base = widget.style ?? DefaultTextStyle.of(context).style;
    Widget text(Color? color) => Text(
      widget.name,
      maxLines: widget.maxLines,
      overflow: TextOverflow.ellipsis,
      textAlign: widget.textAlign,
      style: color == null ? base : base.copyWith(color: color),
    );
    if (ns == null) {
      return text(null);
    }
    if (ns.gradient == null) {
      return text(ns.color);
    }
    Widget masked(double shift) => ShaderMask(
      blendMode: BlendMode.srcIn,
      shaderCallback: (rect) => ns.gradient!.createShader(
        Rect.fromLTWH(-rect.width * shift, 0, rect.width * 2, rect.height),
      ),
      child: text(Colors.white),
    );
    final shift = _shift;
    if (shift == null) {
      return masked(0);
    }
    return AnimatedBuilder(
      animation: shift,
      builder: (context, _) => masked(shift.value),
    );
  }
}

/// A match theme (D-106): the background and the answer buttons.
class MatchThemeStyle {
  const MatchThemeStyle({
    required this.background,
    required this.option,
    required this.chosen,
  });

  final List<Color> background;
  final Color option;
  final List<Color> chosen;
}

/// The match theme of [code], or null for the default.
MatchThemeStyle? matchThemeStyle(String code) => switch (code) {
  'theme_midnight' => const MatchThemeStyle(
    background: [Color(0xFF0B1A3A), Color(0xFF0A2350), Color(0xFF04101F)],
    option: Color(0xFF16386B),
    chosen: [Color(0xFF2F80ED), Color(0xFF56CCF2)],
  ),
  'theme_sunset' => const MatchThemeStyle(
    background: [Color(0xFF3A1030), Color(0xFF6B1E3A), Color(0xFF2A0A1E)],
    option: Color(0xFF6A2848),
    chosen: [Color(0xFFFF7E5F), Color(0xFFFEB47B)],
  ),
  'theme_s1_storm' => const MatchThemeStyle(
    background: [Color(0xFF0D0B2E), Color(0xFF1F1A5C), Color(0xFF080618)],
    option: Color(0xFF2A2770),
    chosen: [Color(0xFF6F5BFF), Color(0xFFFFE04D)],
  ),
  'theme_emerald' => const MatchThemeStyle(
    background: [Color(0xFF062A22), Color(0xFF0B3D2E), Color(0xFF031A14)],
    option: Color(0xFF145A45),
    chosen: [Color(0xFF11998E), Color(0xFF38EF7D)],
  ),
  _ => null,
};

/// Gives the match screen's answer buttons their theme.
class MatchThemeScope extends InheritedWidget {
  const MatchThemeScope({required this.style, required super.child, super.key});

  final MatchThemeStyle? style;

  static MatchThemeStyle? of(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<MatchThemeScope>()?.style;

  @override
  bool updateShouldNotify(MatchThemeScope old) => old.style != style;
}

/// The victory animations (D-106).
enum VictoryKind { confetti, fireworks, lightning, thunder }

VictoryKind? victoryKind(String code) => switch (code) {
  'victory_confetti' => VictoryKind.confetti,
  'victory_fireworks' => VictoryKind.fireworks,
  'victory_lightning' => VictoryKind.lightning,
  'victory_s1_thunder' => VictoryKind.thunder,
  _ => null,
};

/// Plays a victory animation once over its area; it ignores touches.
class VictoryCelebration extends StatefulWidget {
  const VictoryCelebration({
    required this.code,
    this.duration = const Duration(milliseconds: 2800),
    super.key,
  });

  final String code;
  final Duration duration;

  @override
  State<VictoryCelebration> createState() => _VictoryCelebrationState();
}

class _VictoryCelebrationState extends State<VictoryCelebration>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: widget.duration,
  )..forward();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final kind = victoryKind(widget.code);
    if (kind == null) {
      return const SizedBox.shrink();
    }
    return IgnorePointer(
      child: AnimatedBuilder(
        animation: _c,
        builder: (context, _) => _c.isCompleted
            ? const SizedBox.shrink()
            : CustomPaint(
                painter: _VictoryPainter(kind, _c.value),
                size: Size.infinite,
              ),
      ),
    );
  }
}

class _VictoryPainter extends CustomPainter {
  _VictoryPainter(this.kind, this.t);

  final VictoryKind kind;
  final double t;

  static const _colors = [
    Color(0xFFFF4D8D),
    Color(0xFFFFC83D),
    Color(0xFF4DE1FF),
    Color(0xFF8E5BFF),
    Color(0xFF39FF88),
    Color(0xFFFF7A1A),
  ];

  @override
  void paint(Canvas canvas, Size size) {
    final rnd = math.Random(7);
    final fade = t < 0.8 ? 1.0 : (1 - t) / 0.2;
    switch (kind) {
      case VictoryKind.confetti:
        for (var i = 0; i < 90; i++) {
          final x = rnd.nextDouble() * size.width;
          final speed = 0.6 + rnd.nextDouble() * 0.8;
          final y =
              -20 +
              (size.height + 40) * ((t * speed + rnd.nextDouble() * 0.3) % 1.2);
          final sway = math.sin(t * 12 + i) * 18;
          canvas
            ..save()
            ..translate(x + sway, y)
            ..rotate(t * 10 + i)
            ..drawRect(
              const Rect.fromLTWH(-5, -3, 10, 6),
              Paint()
                ..color = _colors[i % _colors.length].withValues(alpha: fade),
            )
            ..restore();
        }
      case VictoryKind.fireworks:
        for (var b = 0; b < 5; b++) {
          final start = b * 0.14;
          final local = ((t - start) / 0.45).clamp(0.0, 1.0);
          if (local <= 0 || local >= 1) {
            continue;
          }
          final center = Offset(
            size.width * (0.15 + rnd.nextDouble() * 0.7),
            size.height * (0.12 + rnd.nextDouble() * 0.4),
          );
          final color = _colors[b % _colors.length];
          final radius = 30 + 90 * Curves.easeOut.transform(local);
          for (var s = 0; s < 18; s++) {
            final a = s / 18 * math.pi * 2;
            final p = center + Offset(math.cos(a), math.sin(a)) * radius;
            canvas.drawCircle(
              p,
              3.2 * (1 - local) + 1,
              Paint()..color = color.withValues(alpha: (1 - local) * fade),
            );
          }
        }
      case VictoryKind.lightning || VictoryKind.thunder:
        final thunder = kind == VictoryKind.thunder;
        final flash =
            (math.sin(t * math.pi * (thunder ? 10 : 6))).abs() *
            (1 - t) *
            (thunder ? 0.45 : 0.25);
        canvas.drawRect(
          Offset.zero & size,
          Paint()..color = const Color(0xFFFFF6B0).withValues(alpha: flash),
        );
        final bolt = Paint()
          ..color =
              (thunder ? const Color(0xFFB9A8FF) : const Color(0xFFFFE04D))
                  .withValues(alpha: fade)
          ..strokeWidth = thunder ? 6 : 4
          ..style = PaintingStyle.stroke
          ..strokeJoin = StrokeJoin.round;
        for (var i = 0; i < (thunder ? 12 : 7); i++) {
          final x = size.width * (0.08 + rnd.nextDouble() * 0.84);
          final y0 =
              size.height *
                  ((t * (0.9 + rnd.nextDouble()) + rnd.nextDouble()) % 1.1) -
              60;
          final path = Path()
            ..moveTo(x, y0)
            ..lineTo(x - 12, y0 + 28)
            ..lineTo(x + 4, y0 + 28)
            ..lineTo(x - 10, y0 + 60);
          canvas.drawPath(path, bolt);
        }
    }
  }

  @override
  bool shouldRepaint(_VictoryPainter old) => old.t != t;
}
