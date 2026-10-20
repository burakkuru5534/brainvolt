import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/notification_providers.dart';
import '../domain/notification_repository.dart';
import 'inbox_screen.dart';

/// Per-category push / in-app switches and quiet hours (D-031, D-086).
class NotificationSettingsScreen extends ConsumerStatefulWidget {
  const NotificationSettingsScreen({super.key});

  @override
  ConsumerState<NotificationSettingsScreen> createState() =>
      _NotificationSettingsScreenState();
}

class _NotificationSettingsScreenState
    extends ConsumerState<NotificationSettingsScreen> {
  NotificationPrefs? _prefs;
  Object? _error;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final p = await ref.read(notificationRepositoryProvider).preferences();
      if (mounted) {
        setState(() => _prefs = p);
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() => _error = e);
      }
    }
  }

  Future<void> _save({
    List<CategoryPref>? categories,
    QuietHours? quiet,
  }) async {
    final messenger = ScaffoldMessenger.of(context);
    final l10n = context.l10n;
    try {
      final p = await ref
          .read(notificationRepositoryProvider)
          .update(categories: categories, quiet: quiet);
      if (mounted) {
        setState(() => _prefs = p);
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  Future<void> _pickTime(QuietHours q, {required bool start}) async {
    final parts = (start ? q.start : q.end).split(':');
    final picked = await showTimePicker(
      context: context,
      initialTime: TimeOfDay(
        hour: int.tryParse(parts.first) ?? 23,
        minute: int.tryParse(parts.last) ?? 0,
      ),
    );
    if (picked == null) {
      return;
    }
    final v =
        '${picked.hour.toString().padLeft(2, '0')}:'
        '${picked.minute.toString().padLeft(2, '0')}';
    await _save(
      quiet: QuietHours(
        enabled: q.enabled,
        start: start ? v : q.start,
        end: start ? q.end : v,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final p = _prefs;
    final text = Theme.of(context).textTheme;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.notificationSettingsTitle)),
      body: p == null
          ? Center(
              child: _error == null
                  ? const CircularProgressIndicator()
                  : Text(playErrorText(l10n, _error!)),
            )
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                BvCard(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Column(
                    children: [
                      SwitchListTile(
                        key: const Key('quiet-enabled'),
                        title: Text(l10n.notifQuietHours),
                        subtitle: Text(l10n.notifQuietHoursHint),
                        value: p.quiet.enabled,
                        onChanged: (v) => _save(
                          quiet: QuietHours(
                            enabled: v,
                            start: p.quiet.start,
                            end: p.quiet.end,
                          ),
                        ),
                      ),
                      ListTile(
                        key: const Key('quiet-start'),
                        title: Text(l10n.notifQuietStart),
                        trailing: Text(p.quiet.start, style: text.titleMedium),
                        onTap: () => _pickTime(p.quiet, start: true),
                      ),
                      ListTile(
                        key: const Key('quiet-end'),
                        title: Text(l10n.notifQuietEnd),
                        trailing: Text(p.quiet.end, style: text.titleMedium),
                        onTap: () => _pickTime(p.quiet, start: false),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 16),
                for (final c in p.categories) ...[
                  BvCard(
                    key: Key('pref-${c.category}'),
                    padding: const EdgeInsets.fromLTRB(16, 10, 8, 10),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          notificationCategoryName(l10n, c.category),
                          style: text.titleMedium,
                        ),
                        if (c.locked)
                          Text(l10n.notifLocked, style: text.bodySmall)
                        else
                          Row(
                            children: [
                              Expanded(child: Text(l10n.notifPush)),
                              Switch(
                                key: Key('pref-${c.category}-push'),
                                value: c.push,
                                onChanged: (v) => _save(
                                  categories: [
                                    CategoryPref(
                                      category: c.category,
                                      push: v,
                                      inApp: c.inApp,
                                    ),
                                  ],
                                ),
                              ),
                              const SizedBox(width: 8),
                              Expanded(child: Text(l10n.notifInApp)),
                              Switch(
                                key: Key('pref-${c.category}-in-app'),
                                value: c.inApp,
                                onChanged: (v) => _save(
                                  categories: [
                                    CategoryPref(
                                      category: c.category,
                                      push: c.push,
                                      inApp: v,
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 10),
                ],
              ],
            ),
    );
  }
}
