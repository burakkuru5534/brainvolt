import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';

/// First-launch age confirmation (D-020, roadmap 1.8.2): nothing is signed
/// in or sent to the backend before the user confirms being 13 or older.
class AgeGateScreen extends ConsumerStatefulWidget {
  const AgeGateScreen({super.key});

  @override
  ConsumerState<AgeGateScreen> createState() => _AgeGateScreenState();
}

class _AgeGateScreenState extends ConsumerState<AgeGateScreen> {
  bool _confirmed = false;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final busy = ref.watch(sessionControllerProvider).isLoading;
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Spacer(),
              BrandHero(tagline: l10n.brandTagline),
              const Spacer(),
              BvCard(
                padding: const EdgeInsets.all(20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      l10n.ageGateTitle,
                      style: Theme.of(context).textTheme.headlineSmall,
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 10),
                    Text(
                      l10n.ageGateBody,
                      textAlign: TextAlign.center,
                      style: TextStyle(color: BvColors.of(context).muted),
                    ),
                    const SizedBox(height: 12),
                    CheckboxListTile(
                      key: const Key('age-confirm'),
                      value: _confirmed,
                      contentPadding: EdgeInsets.zero,
                      onChanged: busy
                          ? null
                          : (v) => setState(() => _confirmed = v ?? false),
                      title: Text(l10n.ageGateConfirm),
                      controlAffinity: ListTileControlAffinity.leading,
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 20),
              GradientButton(
                key: const Key('age-continue'),
                icon: Icons.arrow_forward_rounded,
                label: l10n.continueButton,
                onPressed: _confirmed && !busy
                    ? () => ref
                          .read(sessionControllerProvider.notifier)
                          .confirmAge()
                    : null,
              ),
              const SizedBox(height: 12),
            ],
          ),
        ),
      ),
    );
  }
}
