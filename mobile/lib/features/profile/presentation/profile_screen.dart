import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/dialogs.dart';
import '../../auth/domain/session.dart';
import '../data/profile_providers.dart';
import '../domain/avatars.dart';
import '../domain/me.dart';
import 'profile_controllers.dart';

/// Edit display name, username and avatar (roadmap 1.8.6). The server
/// validates and filters every field; its errors are shown as they come.
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final me = ref.watch(meProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.profileTitle)),
      body: switch (me) {
        AsyncValue(value: final me?) => _ProfileForm(me: me),
        AsyncValue(:final error?) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                error is ApiException
                    ? error.localizedMessage(l10n)
                    : l10n.errorUnknown,
              ),
              TextButton(
                onPressed: () => ref.invalidate(meProvider),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _ProfileForm extends ConsumerStatefulWidget {
  const _ProfileForm({required this.me});

  final Me me;

  @override
  ConsumerState<_ProfileForm> createState() => _ProfileFormState();
}

class _ProfileFormState extends ConsumerState<_ProfileForm> {
  late final _displayName = TextEditingController(
    text: widget.me.profile.displayName,
  );
  late final _username = TextEditingController(
    text: widget.me.profile.username,
  );
  late String _avatar = widget.me.profile.avatarId;
  bool _busy = false;

  @override
  void dispose() {
    _displayName.dispose();
    _username.dispose();
    super.dispose();
  }

  OwnProfile get _profile => widget.me.profile;

  /// Restricted accounts (guests, unverified) keep the generated username
  /// (D-016); full accounts change it once per 30 days.
  bool get _usernameEditable {
    final at = _profile.usernameChangeAvailableAt;
    return widget.me.user.tier == Tier.full &&
        (at == null || !at.isAfter(DateTime.now()));
  }

  Future<void> _save() async {
    final l10n = context.l10n;
    final displayName = _displayName.text.trim();
    final username = _username.text.trim();
    setState(() => _busy = true);
    try {
      await ref
          .read(profileRepositoryProvider)
          .updateProfile(
            displayName: displayName == _profile.displayName
                ? null
                : displayName,
            username: username == _profile.username ? null : username,
            avatarId: _avatar == _profile.avatarId ? null : _avatar,
          );
      ref.invalidate(meProvider);
      // The home screen shows name and avatar from the session.
      await ref.read(sessionControllerProvider.notifier).refresh();
      if (mounted) {
        showSnack(context, l10n.profileSaved);
      }
    } on ApiException catch (e) {
      if (mounted) {
        showSnack(context, e.localizedMessage(l10n));
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
    final available = _profile.usernameChangeAvailableAt;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(l10n.avatarLabel, style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final MapEntry(key: id, value: emoji) in avatarEmoji.entries)
              ChoiceChip(
                key: Key('avatar-$id'),
                label: Text(emoji, style: const TextStyle(fontSize: 24)),
                selected: _avatar == id,
                onSelected: _busy ? null : (_) => setState(() => _avatar = id),
              ),
          ],
        ),
        const SizedBox(height: 24),
        TextField(
          key: const Key('display-name-field'),
          controller: _displayName,
          enabled: !_busy,
          maxLength: 24,
          decoration: InputDecoration(
            labelText: l10n.displayNameLabel,
            helperText: l10n.displayNameHint,
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          key: const Key('username-field'),
          controller: _username,
          enabled: !_busy && _usernameEditable,
          maxLength: 20,
          decoration: InputDecoration(
            labelText: l10n.usernameLabel,
            helperText: widget.me.user.tier != Tier.full
                ? l10n.usernameGuestHint
                : !_usernameEditable && available != null
                ? l10n.usernameAvailableAt(available.toLocal())
                : l10n.usernameHint,
            helperMaxLines: 2,
          ),
        ),
        const SizedBox(height: 24),
        FilledButton(
          key: const Key('profile-save'),
          onPressed: _busy ? null : _save,
          child: Text(l10n.save),
        ),
      ],
    );
  }
}
