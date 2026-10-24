import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/config.dart';
import '../../../app/providers.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import 'error_text.dart';

/// Shown while the session starts, and when it could not start.
class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final session = ref.watch(sessionControllerProvider);
    final error = session.hasError && !session.isLoading ? session.error : null;
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              BrandHero(tagline: l10n.brandTagline),
              const SizedBox(height: 40),
              if (error == null)
                const CircularProgressIndicator()
              else ...[
                Text(
                  l10n.splashError,
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 8),
                Text(
                  errorText(l10n, error) ?? l10n.errorUnknown,
                  textAlign: TextAlign.center,
                  style: TextStyle(color: BvColors.of(context).muted),
                ),
                // Development builds show the technical cause as well.
                if (ref.watch(appConfigProvider).flavor == Flavor.dev) ...[
                  const SizedBox(height: 8),
                  SelectableText(
                    '$error',
                    key: const Key('splash-error-detail'),
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
                const SizedBox(height: 16),
                SizedBox(
                  width: 240,
                  child: GradientButton(
                    key: const Key('splash-retry'),
                    height: 52,
                    icon: Icons.refresh_rounded,
                    label: l10n.retry,
                    onPressed: () => ref.invalidate(sessionControllerProvider),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
