import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../data/auth_providers.dart';
import 'error_text.dart';

/// Email verification (D-017): unverified email accounts keep guest limits
/// until the link in the email is opened.
class VerifyEmailScreen extends ConsumerStatefulWidget {
  const VerifyEmailScreen({super.key});

  @override
  ConsumerState<VerifyEmailScreen> createState() => _VerifyEmailScreenState();
}

class _VerifyEmailScreenState extends ConsumerState<VerifyEmailScreen> {
  bool _busy = false;

  Future<void> _run(Future<String?> Function() action) async {
    setState(() => _busy = true);
    String? message;
    try {
      message = await action();
    } on Object catch (e) {
      if (mounted) {
        message = errorText(context.l10n, e);
      }
    }
    if (!mounted) {
      return;
    }
    setState(() => _busy = false);
    if (message != null) {
      showSnack(context, message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final controller = ref.read(sessionControllerProvider.notifier);
    final email = ref.read(authRepositoryProvider).currentUser?.email ?? '';
    return Scaffold(
      appBar: AppBar(title: Text(l10n.verifyEmailTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Center(
            child: IconBadge(Icons.mark_email_unread_rounded, size: 88),
          ),
          const SizedBox(height: 20),
          BvCard(child: Text(l10n.verifyEmailBody(email))),
          const SizedBox(height: 24),
          FilledButton(
            key: const Key('verify-check'),
            onPressed: _busy
                ? null
                : () => _run(() async {
                    if (!await controller.checkEmailVerified()) {
                      return l10n.verifyEmailNotYet;
                    }
                    if (context.mounted && context.canPop()) {
                      context.pop();
                    }
                    return l10n.verifyEmailVerified;
                  }),
            child: Text(l10n.verifyEmailDone),
          ),
          const SizedBox(height: 12),
          TextButton(
            key: const Key('verify-resend'),
            onPressed: _busy
                ? null
                : () => _run(() async {
                    await controller.resendVerification();
                    return l10n.verifyEmailSent;
                  }),
            child: Text(l10n.verifyEmailResend),
          ),
        ],
      ),
    );
  }
}
