import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/dialogs.dart';
import '../domain/auth_repository.dart';
import 'error_text.dart';

/// How an account flow ended.
enum FlowResult {
  /// The credential was linked to the current user (progress kept).
  linked,

  /// The app switched to another, existing account (D-015).
  switched,

  /// Cancelled or failed; the user was told why.
  none,
}

/// Guest upgrade (D-013a) with the link-conflict flow (D-015): link the
/// credential; if another account owns it, warn that guest progress is lost
/// and switch to that account only when the user agrees.
Future<FlowResult> runUpgrade(
  BuildContext context,
  WidgetRef ref,
  AuthMethod method, {
  String? email,
  String? password,
}) async {
  final l10n = context.l10n;
  final controller = ref.read(sessionControllerProvider.notifier);
  try {
    final result = await controller.upgrade(
      method,
      email: email,
      password: password,
    );
    switch (result) {
      case Upgraded():
        if (context.mounted) {
          showSnack(context, l10n.accountLinked);
        }
        return FlowResult.linked;
      case UpgradeConflict(:final credential):
        if (!context.mounted || !await _confirmSwitch(context)) {
          return FlowResult.none;
        }
        await controller.switchAccount(credential);
        return FlowResult.switched;
    }
  } on Object catch (e) {
    if (context.mounted) {
      _report(context, e);
    }
    return FlowResult.none;
  }
}

/// Signs a guest in to an existing email account: same warning as a link
/// conflict, because the guest's progress stays behind (D-015).
Future<FlowResult> runEmailSignIn(
  BuildContext context,
  WidgetRef ref, {
  required String email,
  required String password,
}) async {
  final controller = ref.read(sessionControllerProvider.notifier);
  if (!await _confirmSwitch(context)) {
    return FlowResult.none;
  }
  try {
    final credential = await controller.credentialFor(
      AuthMethod.email,
      email: email,
      password: password,
    );
    await controller.switchAccount(credential);
    return FlowResult.switched;
  } on Object catch (e) {
    if (context.mounted) {
      _report(context, e);
    }
    return FlowResult.none;
  }
}

Future<bool> _confirmSwitch(BuildContext context) {
  final l10n = context.l10n;
  return confirmDialog(
    context,
    title: l10n.linkConflictTitle,
    body: l10n.linkConflictBody,
    confirmLabel: l10n.continueButton,
    destructive: true,
  );
}

void _report(BuildContext context, Object error) {
  final text = errorText(context.l10n, error);
  if (text != null) {
    showSnack(context, text);
  }
}
