import 'dart:ui' as ui;

import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';

/// Shares a rendered widget as a PNG with a line of text (result cards).
abstract interface class ShareService {
  /// Captures the RepaintBoundary under [boundary] and opens the system
  /// share sheet with it and [text].
  Future<void> shareImage(GlobalKey boundary, String text);
}

/// The system share sheet (share_plus).
class SystemShareService implements ShareService {
  const SystemShareService();

  @override
  Future<void> shareImage(GlobalKey boundary, String text) async {
    final ro = boundary.currentContext?.findRenderObject();
    if (ro is! RenderRepaintBoundary) {
      return;
    }
    final image = await ro.toImage(pixelRatio: 3);
    final data = await image.toByteData(format: ui.ImageByteFormat.png);
    image.dispose();
    if (data == null) {
      return;
    }
    await SharePlus.instance.share(
      ShareParams(
        text: text,
        files: [
          XFile.fromData(
            data.buffer.asUint8List(),
            mimeType: 'image/png',
            name: 'brainvolt.png',
          ),
        ],
      ),
    );
  }
}

/// The app's share service; tests record calls instead.
final shareServiceProvider = Provider<ShareService>(
  (ref) => const SystemShareService(),
);
