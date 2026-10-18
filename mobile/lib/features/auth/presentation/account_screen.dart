import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/dialogs.dart';
import '../data/auth_providers.dart';
import '../domain/auth_repository.dart';
import '../domain/session.dart';
import 'account_flows.dart';
import 'error_text.dart';

/// Sign-in methods and account actions (roadmap 1.8.4–1.8.5). Guests link
/// Google, Apple (iOS) or email; linked accounts can sign out or start
/// deletion.
class AccountScreen extends ConsumerStatefulWidget {
  const AccountScreen({super.key});

  @override
  ConsumerState<AccountScreen> createState() => _AccountScreenState();
}

class _AccountScreenState extends ConsumerState<AccountScreen> {
  bool _busy = false;

  Future<void> _run(Future<void> Function() action) async {
    setState(() => _busy = true);
    try {
      await action();
    } on Object catch (e) {
      if (mounted) {
        final text = errorText(context.l10n, e);
        if (text != null) {
          showSnack(context, text);
        }
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
    final session = ref.watch(sessionControllerProvider).value;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.accountTitle)),
      body: switch (session) {
        SignedIn(:final session) => AbsorbPointer(
          absorbing: _busy,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              if (_busy) const LinearProgressIndicator(),
              if (session.tier == Tier.guest)
                ..._guest(context)
              else
                ..._linked(context, session),
            ],
          ),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }

  List<Widget> _guest(BuildContext context) {
    final l10n = context.l10n;
    Future<void> upgrade(AuthMethod method) =>
        _run(() => runUpgrade(context, ref, method));
    return [
      Text(l10n.accountGuestInfo),
      const SizedBox(height: 24),
      FilledButton.icon(
        key: const Key('sign-in-google'),
        onPressed: () => upgrade(AuthMethod.google),
        icon: const Icon(Icons.g_mobiledata),
        label: Text(l10n.signInWithGoogle),
      ),
      // Sign in with Apple sits next to Google on iOS (D-013).
      if (defaultTargetPlatform == TargetPlatform.iOS) ...[
        const SizedBox(height: 12),
        FilledButton.icon(
          key: const Key('sign-in-apple'),
          onPressed: () => upgrade(AuthMethod.apple),
          icon: const Icon(Icons.apple),
          label: Text(l10n.signInWithApple),
        ),
      ],
      const SizedBox(height: 12),
      OutlinedButton.icon(
        key: const Key('sign-up-email'),
        onPressed: () => context.push(Routes.emailSignUp),
        icon: const Icon(Icons.email_outlined),
        label: Text(l10n.signUpWithEmail),
      ),
      const SizedBox(height: 12),
      TextButton(
        key: const Key('sign-in-email'),
        onPressed: () => context.push(Routes.emailSignIn),
        child: Text(l10n.signInWithEmail),
      ),
    ];
  }

  List<Widget> _linked(BuildContext context, Session session) {
    final l10n = context.l10n;
    final controller = ref.read(sessionControllerProvider.notifier);
    final email = ref.read(authRepositoryProvider).currentUser?.email;
    return [
      ListTile(
        leading: const Icon(Icons.verified_user_outlined),
        title: Text(
          email == null ? l10n.accountSignedIn : l10n.accountSignedInAs(email),
        ),
      ),
      if (session.flags.emailVerificationRequired)
        ListTile(
          key: const Key('open-verify'),
          leading: const Icon(Icons.mark_email_unread_outlined),
          title: Text(l10n.verifyEmailTitle),
          trailing: const Icon(Icons.chevron_right),
          onTap: () => context.push(Routes.verifyEmail),
        ),
      const Divider(),
      ListTile(
        key: const Key('sign-out'),
        leading: const Icon(Icons.logout),
        title: Text(l10n.signOut),
        onTap: () => _run(() async {
          if (await confirmDialog(
            context,
            title: l10n.signOut,
            body: l10n.signOutConfirm,
            confirmLabel: l10n.signOut,
          )) {
            await controller.signOut();
          }
        }),
      ),
      ListTile(
        key: const Key('delete-account'),
        leading: Icon(
          Icons.delete_forever_outlined,
          color: Theme.of(context).colorScheme.error,
        ),
        title: Text(l10n.deleteAccount),
        onTap: () => _run(() async {
          if (await confirmDialog(
            context,
            title: l10n.deleteAccount,
            body: l10n.deleteAccountConfirm,
            confirmLabel: l10n.deleteAccount,
            destructive: true,
          )) {
            await controller.requestDeletion();
          }
        }),
      ),
    ];
  }
}
