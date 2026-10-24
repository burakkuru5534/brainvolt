import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../profile/domain/avatars.dart';
import '../domain/match_repository.dart';
import 'match_type_name.dart';
import 'play_controllers.dart';
import 'play_error_text.dart';

/// Looking for an opponent (or starting a practice match), then straight
/// to the match screen.
class QueueScreen extends ConsumerStatefulWidget {
  const QueueScreen({
    this.practice = false,
    this.mode = const QueueMode(),
    super.key,
  });

  final bool practice;
  final QueueMode mode;

  @override
  ConsumerState<QueueScreen> createState() => _QueueScreenState();
}

class _QueueScreenState extends ConsumerState<QueueScreen> {
  Timer? _tick;

  @override
  void initState() {
    super.initState();
    _tick = Timer.periodic(const Duration(seconds: 1), (_) => setState(() {}));
    unawaited(Future.microtask(_search));
  }

  Future<void> _search() => ref
      .read(queueControllerProvider.notifier)
      .search(practice: widget.practice, mode: widget.mode);

  @override
  void dispose() {
    _tick?.cancel();
    super.dispose();
  }

  Future<void> _cancel() async {
    await ref
        .read(queueControllerProvider.notifier)
        .cancel()
        .catchError((_) {});
    if (mounted) {
      context.go(Routes.home);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    ref.listen(queueControllerProvider, (_, next) {
      if (next is QueueFound) {
        context.go(Routes.match(next.matchId));
      } else if (next is QueueLobby) {
        context.go(Routes.room(next.roomId));
      }
    });
    final state = ref.watch(queueControllerProvider);
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) {
          unawaited(_cancel());
        }
      },
      child: Scaffold(
        appBar: AppBar(title: Text(matchTypeName(l10n, widget.mode.matchType))),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: switch (state) {
              QueueFailed(:final error) => Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(playErrorText(l10n, error), textAlign: TextAlign.center),
                  const SizedBox(height: 16),
                  FilledButton(onPressed: _search, child: Text(l10n.retry)),
                  TextButton(
                    onPressed: () => context.go(Routes.home),
                    child: Text(l10n.backHome),
                  ),
                ],
              ),
              _ => Column(
                key: const Key('queue-searching'),
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Radar(),
                  const SizedBox(height: 28),
                  Text(
                    l10n.searchingTitle,
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 20),
                  if (state case QueueSearching(:final since))
                    _Elapsed(DateTime.now().difference(since).inSeconds),
                  const SizedBox(height: 8),
                  Text(
                    l10n.queueFinding,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 28),
                  SizedBox(
                    width: 260,
                    child: OutlinedButton.icon(
                      key: const Key('queue-cancel'),
                      onPressed: _cancel,
                      icon: const Icon(Icons.close),
                      label: Text(l10n.searchingCancel),
                    ),
                  ),
                ],
              ),
            },
          ),
        ),
      ),
    );
  }
}

/// The elapsed seconds in a segmented ring (one lit segment a second).
class _Elapsed extends StatelessWidget {
  const _Elapsed(this.seconds);

  final int seconds;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Column(
      children: [
        SizedBox(
          width: 96,
          height: 96,
          child: CustomPaint(
            painter: _RingPainter(seconds % 10, bv.gradientStart, bv.badge),
            child: Center(
              child: Icon(Icons.bolt, size: 40, color: bv.gradientStart),
            ),
          ),
        ),
        const SizedBox(height: 8),
        Text(
          context.l10n.searchingElapsed(seconds),
          style: Theme.of(context).textTheme.headlineSmall
              ?.copyWith(color: bv.muted),
        ),
      ],
    );
  }
}

class _RingPainter extends CustomPainter {
  _RingPainter(this.lit, this.on, this.off);

  final int lit;
  final Color on;
  final Color off;

  @override
  void paint(Canvas canvas, Size size) {
    const segments = 10;
    final rect = Offset.zero & size;
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 10
      ..strokeCap = StrokeCap.round;
    for (var i = 0; i < segments; i++) {
      paint.color = i <= lit ? on : off;
      final start = -math.pi / 2 + i * 2 * math.pi / segments + 0.12;
      canvas.drawArc(
        rect.deflate(6),
        start,
        2 * math.pi / segments - 0.24,
        false,
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(_RingPainter old) => old.lit != lit;
}

/// The searching radar (D-085): a sweeping beam between the player and
/// the unknown opponent.
class Radar extends ConsumerStatefulWidget {
  const Radar({super.key});

  @override
  ConsumerState<Radar> createState() => _RadarState();
}

class _RadarState extends ConsumerState<Radar>
    with SingleTickerProviderStateMixin {
  late final _spin = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 3),
  )..repeat();

  @override
  void dispose() {
    _spin.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final avatar = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.profile.avatarId,
      _ => '',
    };
    return SizedBox(
      width: 320,
      height: 260,
      child: Stack(
        alignment: Alignment.center,
        children: [
          SizedBox(
            width: 250,
            height: 250,
            child: AnimatedBuilder(
              animation: _spin,
              builder: (context, _) => CustomPaint(
                painter: _RadarPainter(
                  _spin.value,
                  bv.gradientEnd,
                  bv.gradientStart,
                ),
              ),
            ),
          ),
          const IconBadge(Icons.bolt, size: 64, onGradient: true),
          Positioned(left: 0, child: AvatarBubble(avatarFor(avatar), size: 84)),
          const Positioned(right: 0, child: AvatarBubble('❔', size: 84)),
        ],
      ),
    );
  }
}

class _RadarPainter extends CustomPainter {
  _RadarPainter(this.turn, this.deep, this.light);

  final double turn;
  final Color deep;
  final Color light;

  @override
  void paint(Canvas canvas, Size size) {
    final c = size.center(Offset.zero);
    final r = size.shortestSide / 2;
    canvas.drawCircle(c, r, Paint()..color = deep);
    final grid = Paint()
      ..style = PaintingStyle.stroke
      ..color = Colors.white.withAlpha(40)
      ..strokeWidth = 1;
    for (final f in [0.33, 0.66, 1.0]) {
      canvas.drawCircle(c, r * f, grid);
    }
    canvas
      ..drawLine(c - Offset(r, 0), c + Offset(r, 0), grid)
      ..drawLine(c - Offset(0, r), c + Offset(0, r), grid);
    final angle = turn * 2 * math.pi;
    final sweep = Paint()
      ..shader = SweepGradient(
        startAngle: 0,
        endAngle: math.pi / 2,
        colors: [light.withAlpha(0), light.withAlpha(200)],
        transform: GradientRotation(angle - math.pi / 2),
      ).createShader(Rect.fromCircle(center: c, radius: r));
    canvas.drawArc(
      Rect.fromCircle(center: c, radius: r),
      angle - math.pi / 2,
      math.pi / 2,
      true,
      sweep,
    );
    canvas.drawCircle(
      c,
      r,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 4
        ..color = light.withAlpha(160),
    );
  }

  @override
  bool shouldRepaint(_RadarPainter old) => old.turn != turn;
}
