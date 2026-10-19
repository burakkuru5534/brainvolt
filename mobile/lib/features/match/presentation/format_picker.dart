import 'package:flutter/material.dart';

import '../../../core/l10n/l10n.dart';

/// The formats the app offers (mode-matrix §1).
const formats = ['1v1', '2v2', '3v3', '4v4', 'couple'];

/// The team answer modes (team-modes.md).
const teamModes = ['fast_team_round', 'shared_answer', 'individual_turns'];

String formatName(AppLocalizations l, String f) =>
    f == 'couple' ? l.formatCouple : f;

String teamModeName(AppLocalizations l, String m) => switch (m) {
  'shared_answer' => l.teamModeShared,
  'individual_turns' => l.teamModeTurns,
  _ => l.teamModeFast,
};

/// Asks for a format and, for team formats, a team answer mode.
Future<(String, String?)?> pickFormat(BuildContext context) =>
    showDialog<(String, String?)>(
      context: context,
      builder: (_) => const _FormatDialog(),
    );

class _FormatDialog extends StatefulWidget {
  const _FormatDialog();

  @override
  State<_FormatDialog> createState() => _FormatDialogState();
}

class _FormatDialogState extends State<_FormatDialog> {
  String _format = '1v1';
  String _mode = teamModes.first;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AlertDialog(
      title: Text(l10n.formatLabel),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            spacing: 6,
            children: [
              for (final f in formats)
                ChoiceChip(
                  key: Key('format-$f'),
                  label: Text(formatName(l10n, f)),
                  selected: _format == f,
                  onSelected: (_) => setState(() => _format = f),
                ),
            ],
          ),
          if (_format != '1v1') ...[
            const SizedBox(height: 16),
            Text(l10n.teamModeLabel),
            Wrap(
              spacing: 6,
              children: [
                for (final m in teamModes)
                  ChoiceChip(
                    key: Key('team-mode-$m'),
                    label: Text(teamModeName(l10n, m)),
                    selected: _mode == m,
                    onSelected: (_) => setState(() => _mode = m),
                  ),
              ],
            ),
          ],
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('format-ok'),
          onPressed: () => Navigator.pop(context, (
            _format,
            _format == '1v1' ? null : _mode,
          )),
          child: Text(l10n.continueButton),
        ),
      ],
    );
  }
}
