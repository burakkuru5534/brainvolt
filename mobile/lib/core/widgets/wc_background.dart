import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../app/theme.dart';

/// The WitClash page background (D-099): a deep navy gradient, a purple
/// glow at the top, an orange glow at the bottom and a still star field.
/// Every routed page gets it through the theme's page transitions; use it
/// directly only for surfaces outside the router (tests, overlays).
class WcBackground extends StatelessWidget {
  const WcBackground({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [bv.backgroundTop, bv.background, bv.backgroundBottom],
          stops: const [0, 0.55, 1],
        ),
      ),
      child: Stack(
        fit: StackFit.expand,
        children: [
          RepaintBoundary(
            child: CustomPaint(
              painter: _StarFieldPainter(bv),
              child: const SizedBox.expand(),
            ),
          ),
          child,
        ],
      ),
    );
  }
}

class _StarFieldPainter extends CustomPainter {
  _StarFieldPainter(this.bv);

  final BvColors bv;

  @override
  void paint(Canvas canvas, Size size) {
    if (size.isEmpty) {
      return;
    }
    final rect = Offset.zero & size;
    // Glows: purple from the top left, orange from the bottom right.
    canvas
      ..drawRect(
        rect,
        Paint()
          ..shader = RadialGradient(
            center: const Alignment(-0.9, -1.05),
            radius: 1.1,
            colors: [
              bv.gradientStart.withAlpha(110),
              bv.gradientStart.withAlpha(0),
            ],
          ).createShader(rect),
      )
      ..drawRect(
        rect,
        Paint()
          ..shader = RadialGradient(
            center: const Alignment(1.1, 1.05),
            radius: 1.0,
            colors: [bv.gradientEnd.withAlpha(70), bv.gradientEnd.withAlpha(0)],
          ).createShader(rect),
      )
      ..drawRect(
        rect,
        Paint()
          ..shader = RadialGradient(
            center: const Alignment(1, -0.2),
            radius: 0.7,
            colors: [bv.gradientMid.withAlpha(40), bv.gradientMid.withAlpha(0)],
          ).createShader(rect),
      );

    // Stars: the same field on every page (a fixed seed), density by area.
    final random = math.Random(7);
    final count = (size.width * size.height / 2600).clamp(60, 420).round();
    final dot = Paint();
    for (var i = 0; i < count; i++) {
      final p = Offset(
        random.nextDouble() * size.width,
        random.nextDouble() * size.height,
      );
      final r = 0.4 + random.nextDouble() * random.nextDouble() * 1.4;
      final tint = switch (random.nextInt(9)) {
        0 => bv.badgeInk,
        1 => const Color(0xFFFFD6B8),
        _ => Colors.white,
      };
      dot.color = tint.withAlpha(50 + random.nextInt(150));
      canvas.drawCircle(p, r, dot);
    }

    // A few bright sparkles with a soft halo.
    final halo = Paint()
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 4);
    final ray = Paint()
      ..strokeWidth = 1
      ..strokeCap = StrokeCap.round;
    for (var i = 0; i < 5; i++) {
      final p = Offset(
        random.nextDouble() * size.width,
        random.nextDouble() * size.height,
      );
      final l = 2 + random.nextDouble() * 2.5;
      halo.color = Colors.white.withAlpha(70);
      canvas.drawCircle(p, l * 0.7, halo);
      ray.color = Colors.white.withAlpha(150);
      canvas
        ..drawLine(p.translate(-l, 0), p.translate(l, 0), ray)
        ..drawLine(p.translate(0, -l), p.translate(0, l), ray)
        ..drawCircle(p, 1.1, Paint()..color = Colors.white);
    }
  }

  @override
  bool shouldRepaint(_StarFieldPainter old) => old.bv != bv;
}
