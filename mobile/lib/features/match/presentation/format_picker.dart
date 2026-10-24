import 'package:flutter/material.dart';

import '../../../core/l10n/l10n.dart';
import '../../lobby/domain/lobby_repository.dart';
import 'match_type_name.dart';

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

/// Players a format seats in a room (both sides).
int formatCapacity(String format) => switch (format) {
  '2v2' || 'couple' => 4,
  '3v3' => 6,
  '4v4' => 8,
  _ => 2,
};

/// Asks for every room mode (D-093): match type, format and, for team
/// formats, the team answer mode. [initial] preselects the current
/// settings; [quizLocked] keeps Quick Duel (a Creator Quiz room, R07);
/// [members] greys out formats with too few seats for the room.
Future<RoomSettings?> pickRoomSettings(
  BuildContext context, {
  RoomSettings initial = const RoomSettings(),
  bool quizLocked = false,
  int members = 0,
  String? title,
}) => showDialog<RoomSettings>(
  context: context,
  builder: (_) => _SettingsDialog(
    initial: initial,
    quizLocked: quizLocked,
    members: members,
    title: title,
  ),
);

class _SettingsDialog extends StatefulWidget {
  const _SettingsDialog({
    required this.initial,
    required this.quizLocked,
    required this.members,
    this.title,
  });

  final RoomSettings initial;
  final bool quizLocked;
  final int members;
  final String? title;

  @override
  State<_SettingsDialog> createState() => _SettingsDialogState();
}

class _SettingsDialogState extends State<_SettingsDialog> {
  late MatchType _type = widget.initial.matchType;
  late String _format = widget.initial.format;
  late String _mode = widget.initial.teamMode ?? teamModes.first;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final text = Theme.of(context).textTheme;
    Widget label(String s) => Padding(
      padding: const EdgeInsets.only(top: 14, bottom: 6),
      child: Text(s, style: text.titleSmall),
    );
    return AlertDialog(
      title: Text(widget.title ?? l10n.roomSettingsTitle),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            label(l10n.matchTypeLabel),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                for (final t in MatchType.values)
                  ChoiceChip(
                    key: Key('match-type-${t.wire}'),
                    label: Text(matchTypeName(l10n, t.wire)),
                    selected: _type == t,
                    onSelected: widget.quizLocked && t != MatchType.quickDuel
                        ? null
                        : (_) => setState(() => _type = t),
                  ),
              ],
            ),
            label(l10n.formatLabel),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                for (final f in formats)
                  ChoiceChip(
                    key: Key('format-$f'),
                    label: Text(formatName(l10n, f)),
                    selected: _format == f,
                    onSelected: formatCapacity(f) < widget.members
                        ? null
                        : (_) => setState(() => _format = f),
                  ),
              ],
            ),
            if (_format != '1v1') ...[
              label(l10n.teamModeLabel),
              Wrap(
                spacing: 6,
                runSpacing: 6,
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
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('format-ok'),
          onPressed: () => Navigator.pop(
            context,
            RoomSettings(
              matchType: widget.quizLocked ? MatchType.quickDuel : _type,
              format: _format,
              teamMode: _format == '1v1' ? null : _mode,
              categoryMode: widget.initial.categoryMode,
            ),
          ),
          child: Text(l10n.continueButton),
        ),
      ],
    );
  }
}
