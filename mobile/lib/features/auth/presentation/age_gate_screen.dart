import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';

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
              Text(
                l10n.ageGateTitle,
                style: Theme.of(context).textTheme.headlineMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              Text(l10n.ageGateBody, textAlign: TextAlign.center),
              const SizedBox(height: 24),
              CheckboxListTile(
                key: const Key('age-confirm'),
                value: _confirmed,
                onChanged: busy
                    ? null
                    : (v) => setState(() => _confirmed = v ?? false),
                title: Text(l10n.ageGateConfirm),
                controlAffinity: ListTileControlAffinity.leading,
              ),
              const SizedBox(height: 16),
              FilledButton(
                key: const Key('age-continue'),
                onPressed: _confirmed && !busy
                    ? () => ref
                          .read(sessionControllerProvider.notifier)
                          .confirmAge()
                    : null,
                child: Text(l10n.continueButton),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
