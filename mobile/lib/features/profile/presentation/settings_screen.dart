import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../../core/l10n/locale_controller.dart';

/// App settings. Phase 1 has the language choice (D-002); profile and
/// privacy settings are wired to the API in roadmap 1.8.6.
class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final selected = ref.watch(localeOverrideProvider)?.languageCode;
    final options = <(String?, String)>[
      (null, l10n.languageSystem),
      ('tr', l10n.languageTurkish),
      ('en', l10n.languageEnglish),
    ];
    return Scaffold(
      appBar: AppBar(title: Text(l10n.settingsTitle)),
      body: ListView(
        children: [
          ListTile(
            title: Text(
              l10n.settingsLanguage,
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),
          RadioGroup<String?>(
            groupValue: selected,
            onChanged: (code) => ref
                .read(localeOverrideProvider.notifier)
                .set(code == null ? null : Locale(code)),
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
      ),
    );
  }
}
