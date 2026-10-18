import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:logging/logging.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/l10n/locale_controller.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/dialogs.dart';
import '../data/profile_providers.dart';
import '../domain/me.dart';
import 'profile_controllers.dart';

final _log = Logger('profile');

/// App settings: language (D-002), links to profile and account, and the
/// game and privacy settings stored on the server (roadmap 1.8.6).
class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final settings = ref.watch(settingsControllerProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.settingsTitle)),
      body: ListView(
        children: [
          ListTile(
            key: const Key('open-profile'),
            leading: const Icon(Icons.person_outline),
            title: Text(l10n.profileTitle),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push(Routes.profile),
          ),
          ListTile(
            key: const Key('open-account'),
            leading: const Icon(Icons.account_circle_outlined),
            title: Text(l10n.accountTitle),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push(Routes.account),
          ),
          ListTile(
            key: const Key('open-credits'),
            leading: const Icon(Icons.info_outline),
            title: Text(l10n.creditsTitle),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push(Routes.credits),
          ),
          const Divider(),
          const _LanguageSection(),
          const Divider(),
          switch (settings) {
            AsyncValue(value: final s?) => _ServerSettings(settings: s),
            AsyncValue(:final error?) => ListTile(
              title: Text(
                error is ApiException
                    ? error.localizedMessage(l10n)
                    : l10n.errorUnknown,
              ),
              trailing: TextButton(
                onPressed: () => ref.invalidate(settingsControllerProvider),
                child: Text(l10n.retry),
              ),
            ),
            _ => const Padding(
              padding: EdgeInsets.all(16),
              child: Center(child: CircularProgressIndicator()),
            ),
          },
        ],
      ),
    );
  }
}

class _LanguageSection extends ConsumerWidget {
  const _LanguageSection();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final selected = ref.watch(localeOverrideProvider)?.languageCode;
    final options = <(String?, String)>[
      (null, l10n.languageSystem),
      ('tr', l10n.languageTurkish),
      ('en', l10n.languageEnglish),
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Header(l10n.settingsLanguage),
        RadioGroup<String?>(
          groupValue: selected,
          onChanged: (code) async {
            await ref
                .read(localeOverrideProvider.notifier)
                .set(code == null ? null : Locale(code));
            // The profile locale picks the language of server texts such as
            // push messages; best effort, the app works without it.
            final locale = effectiveLocale(ref.read(localeOverrideProvider));
            try {
              await ref
                  .read(profileRepositoryProvider)
                  .updateProfile(locale: locale.languageCode);
            } on Object catch (e) {
              _log.warning('profile locale update failed', e);
            }
          },
          child: Column(
            children: [
              for (final (code, label) in options)
                RadioListTile<String?>(
                  key: Key('locale-${code ?? 'system'}'),
                  value: code,
                  title: Text(label),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ServerSettings extends ConsumerWidget {
  const _ServerSettings({required this.settings});

  final UserSettings settings;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    Future<void> change(UserSettings next) async {
      try {
        await ref.read(settingsControllerProvider.notifier).change(next);
      } on Object catch (e) {
        if (context.mounted) {
          showSnack(
            context,
            e is ApiException ? e.localizedMessage(l10n) : l10n.errorUnknown,
          );
        }
      }
    }

    Widget toggle(String key, String title, bool value, UserSettings next) =>
        SwitchListTile(
          key: Key('setting-$key'),
          title: Text(title),
          value: value,
          onChanged: (_) => change(next),
        );

    final s = settings;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Header(l10n.settingsGame),
        toggle(
          'sound',
          l10n.settingsSound,
          s.soundEnabled,
          s.copyWith(soundEnabled: !s.soundEnabled),
        ),
        toggle(
          'music',
          l10n.settingsMusic,
          s.musicEnabled,
          s.copyWith(musicEnabled: !s.musicEnabled),
        ),
        toggle(
          'haptics',
          l10n.settingsHaptics,
          s.hapticsEnabled,
          s.copyWith(hapticsEnabled: !s.hapticsEnabled),
        ),
        const Divider(),
        _Header(l10n.settingsPrivacy),
        toggle(
          'online',
          l10n.settingsShowOnline,
          s.showOnlineStatus,
          s.copyWith(showOnlineStatus: !s.showOnlineStatus),
        ),
        toggle(
          'friend-requests',
          l10n.settingsAllowFriendRequests,
          s.allowFriendRequests,
          s.copyWith(allowFriendRequests: !s.allowFriendRequests),
        ),
        ListTile(
          title: Text(l10n.settingsChallengesFrom),
          trailing: DropdownButton<ChallengesFrom>(
            key: const Key('setting-challenges'),
            value: s.challengesFrom,
            onChanged: (v) =>
                v == null ? null : change(s.copyWith(challengesFrom: v)),
            items: [
              for (final (v, label) in [
                (ChallengesFrom.everyone, l10n.challengesEveryone),
                (ChallengesFrom.friends, l10n.challengesFriends),
                (ChallengesFrom.nobody, l10n.challengesNobody),
              ])
                DropdownMenuItem(value: v, child: Text(label)),
            ],
          ),
        ),
      ],
    );
  }
}

class _Header extends StatelessWidget {
  const _Header(this.text);

  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
    child: Text(text, style: Theme.of(context).textTheme.titleMedium),
  );
}
