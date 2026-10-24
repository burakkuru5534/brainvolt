import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../app/theme.dart';
import '../l10n/l10n.dart';
import '../widgets/bv.dart';
import 'share_service.dart';

/// A shareable result (D-087): the brand, a headline, a big line (score
/// or correct answers) and a detail line, on the WitClash gradient.
class ResultCard extends StatelessWidget {
  const ResultCard({
    required this.headline,
    required this.big,
    required this.detail,
    this.icon = Icons.emoji_events,
    super.key,
  });

  final String headline;
  final String big;
  final String detail;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    TextStyle s(double size, FontWeight w, [Color c = Colors.white]) =>
        weighted(TextStyle(fontSize: size, color: c, fontFamily: brandFont), w);
    return Container(
      width: 320,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [bv.backgroundTop, bv.gradientStart, bv.gradientEnd],
        ),
        borderRadius: BorderRadius.circular(28),
        border: Border.all(color: Colors.white.withAlpha(60)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const BrandMark(size: 30),
              const SizedBox(width: 8),
              Text(
                'WitClash',
                style: s(
                  22,
                  FontWeight.w900,
                ).copyWith(fontStyle: FontStyle.italic),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Icon(icon, size: 56, color: bv.gold),
          const SizedBox(height: 8),
          Text(
            headline,
            textAlign: TextAlign.center,
            style: s(24, FontWeight.w900),
          ),
          Text(big, textAlign: TextAlign.center, style: s(44, FontWeight.w900)),
          const SizedBox(height: 4),
          Text(
            detail,
            textAlign: TextAlign.center,
            style: s(16, FontWeight.w700, Colors.white70),
          ),
        ],
      ),
    );
  }
}

/// Shows [card] in a dialog with a share button that sends it as an
/// image with [text].
Future<void> showShareCard(
  BuildContext context,
  WidgetRef ref, {
  required ResultCard card,
  required String text,
}) {
  final boundary = GlobalKey();
  final l10n = context.l10n;
  return showDialog<void>(
    context: context,
    builder: (context) => Dialog(
      backgroundColor: Colors.transparent,
      elevation: 0,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          RepaintBoundary(key: boundary, child: card),
          const SizedBox(height: 16),
          SizedBox(
            width: 320,
            child: GradientButton(
              key: const Key('share-send'),
              label: l10n.shareAction,
              icon: Icons.ios_share,
              onPressed: () async {
                await ref.read(shareServiceProvider).shareImage(boundary, text);
                if (context.mounted) {
                  Navigator.of(context).pop();
                }
              },
            ),
          ),
        ],
      ),
    ),
  );
}
