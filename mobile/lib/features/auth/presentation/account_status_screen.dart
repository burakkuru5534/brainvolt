import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../domain/session.dart';
import 'error_text.dart';

/// Shown instead of the game while the account is suspended or waiting for
/// deletion (D-022, D-059): the only actions are cancelling the deletion
/// and signing out.
class AccountStatusScreen extends ConsumerStatefulWidget {
  const AccountStatusScreen({super.key});

  @override
  ConsumerState<AccountStatusScreen> createState() =>
      _AccountStatusScreenState();
}

class _AccountStatusScreenState extends ConsumerState<AccountStatusScreen> {
  bool _busy = false;

  Future<void> _run(Future<void> Function() action) async {
    setState(() => _busy = true);
    try {
      await action();
    } on Object catch (e) {
      if (mounted) {
        showSnack(
          context,
          errorText(context.l10n, e) ?? context.l10n.errorUnknown,
        );
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final controller = ref.read(sessionControllerProvider.notifier);
    final session = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session,
      _ => null,
    };
    if (session == null) {
      return const Scaffold(body: LoadingView());
    }
    final requestedAt = session.user.deletionRequestedAt;
    final pending = session.flags.deletionPending && requestedAt != null;
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Center(
                child: IconBadge(
                  pending ? Icons.delete_outline_rounded : Icons.block_rounded,
                  size: 88,
                ),
              ),
              const SizedBox(height: 20),
              Text(
                pending
                    ? l10n.deletionPendingTitle
                    : l10n.accountSuspendedTitle,
                style: Theme.of(context).textTheme.headlineSmall,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              Text(
                pending
                    ? l10n.deletionPendingBody(
                        requestedAt.add(deletionGracePeriod).toLocal(),
                      )
                    : l10n.accountSuspendedBody,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              if (pending)
                FilledButton(
                  key: const Key('cancel-deletion'),
                  onPressed: _busy
                      ? null
                      : () => _run(controller.cancelDeletion),
                  child: Text(l10n.cancelDeletion),
                ),
              const SizedBox(height: 12),
              OutlinedButton(
                key: const Key('status-sign-out'),
                onPressed: _busy ? null : () => _run(controller.signOut),
                child: Text(l10n.signOut),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
