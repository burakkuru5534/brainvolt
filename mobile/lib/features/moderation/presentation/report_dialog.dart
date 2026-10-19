import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/moderation_providers.dart';
import '../domain/moderation_repository.dart';

String reportReasonName(AppLocalizations l10n, String reason) =>
    switch (reason) {
      'wrong_answer' => l10n.reportReasonWrongAnswer,
      'ambiguous' => l10n.reportReasonAmbiguous,
      'outdated' => l10n.reportReasonOutdated,
      'typo_or_translation' => l10n.reportReasonTypo,
      'offensive_content' => l10n.reportReasonOffensive,
      _ => l10n.reportReasonOther,
    };

/// Asks for a reason and an optional note, then reports the question.
Future<void> reportQuestion(
  BuildContext context,
  WidgetRef ref, {
  required String questionId,
  String? matchId,
}) async {
  final l10n = context.l10n;
  final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
  final picked = await showDialog<(String, String)>(
    context: context,
    builder: (_) => const _ReportDialog(),
  );
  if (picked == null) {
    return;
  }
  try {
    await ref
        .read(moderationRepositoryProvider)
        .reportQuestion(
          questionId: questionId,
          reason: picked.$1,
          matchId: matchId,
          comment: picked.$2,
        );
    messenger.showSnackBar(SnackBar(content: Text(l10n.reportSent)));
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
}

class _ReportDialog extends StatefulWidget {
  const _ReportDialog();

  @override
  State<_ReportDialog> createState() => _ReportDialogState();
}

class _ReportDialogState extends State<_ReportDialog> {
  final _note = TextEditingController();
  String? _reason;

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AlertDialog(
      title: Text(l10n.reportReasonTitle),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            RadioGroup<String>(
              groupValue: _reason,
              onChanged: (v) => setState(() => _reason = v),
              child: Column(
                children: [
                  for (final r in questionReportReasons)
                    RadioListTile<String>(
                      key: Key('report-reason-$r'),
                      value: r,
                      title: Text(reportReasonName(l10n, r)),
                    ),
                ],
              ),
            ),
            TextField(
              key: const Key('report-comment'),
              controller: _note,
              maxLength: 500,
              decoration: InputDecoration(labelText: l10n.reportComment),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('report-send'),
          onPressed: _reason == null
              ? null
              : () => Navigator.pop(context, (_reason!, _note.text.trim())),
          child: Text(l10n.reportSend),
        ),
      ],
    );
  }
}
