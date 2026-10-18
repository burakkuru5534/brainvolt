import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../domain/auth_repository.dart';
import 'account_flows.dart';

/// Email/password sign-up (links the email to the guest, D-013a) or
/// sign-in to an existing email account (D-015 warning first).
class EmailAuthScreen extends ConsumerStatefulWidget {
  const EmailAuthScreen({required this.signUp, super.key});

  /// true: create the account; false: sign in.
  final bool signUp;

  @override
  ConsumerState<EmailAuthScreen> createState() => _EmailAuthScreenState();
}

class _EmailAuthScreenState extends ConsumerState<EmailAuthScreen> {
  final _form = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();
  bool _busy = false;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_form.currentState!.validate()) {
      return;
    }
    setState(() => _busy = true);
    final email = _email.text.trim();
    final password = _password.text;
    final result = widget.signUp
        ? await runUpgrade(
            context,
            ref,
            AuthMethod.email,
            email: email,
            password: password,
          )
        : await runEmailSignIn(context, ref, email: email, password: password);
    if (!mounted) {
      return;
    }
    setState(() => _busy = false);
    switch (result) {
      case FlowResult.linked:
        context.pushReplacement(Routes.verifyEmail);
      case FlowResult.switched:
        context.pop();
      case FlowResult.none:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.signUp ? l10n.signUpWithEmail : l10n.emailSignInTitle,
        ),
      ),
      body: Form(
        key: _form,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              key: const Key('email-field'),
              controller: _email,
              enabled: !_busy,
              keyboardType: TextInputType.emailAddress,
              autofillHints: const [AutofillHints.email],
              decoration: InputDecoration(labelText: l10n.emailLabel),
              validator: (v) => v != null && v.trim().contains('@')
                  ? null
                  : l10n.authErrorInvalidEmail,
            ),
            const SizedBox(height: 12),
            TextFormField(
              key: const Key('password-field'),
              controller: _password,
              enabled: !_busy,
              obscureText: true,
              autofillHints: [
                if (widget.signUp)
                  AutofillHints.newPassword
                else
                  AutofillHints.password,
              ],
              decoration: InputDecoration(
                labelText: l10n.passwordLabel,
                helperText: widget.signUp ? l10n.passwordHint : null,
              ),
              // Firebase's minimum; the server enforces it too.
              validator: (v) =>
                  (v?.length ?? 0) >= 6 ? null : l10n.authErrorWeakPassword,
            ),
            const SizedBox(height: 24),
            FilledButton(
              key: const Key('email-submit'),
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox.square(
                      dimension: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(widget.signUp ? l10n.signUpButton : l10n.signInButton),
            ),
          ],
        ),
      ),
    );
  }
}
