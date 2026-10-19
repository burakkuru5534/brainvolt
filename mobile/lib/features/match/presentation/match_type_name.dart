import '../../../core/l10n/l10n.dart';

/// The localized name of a match type (`match_type` on the wire).
String matchTypeName(AppLocalizations l, String wire) => switch (wire) {
  'ranked' => l.matchTypeRanked,
  'sudden_death' => l.matchTypeSuddenDeath,
  'marathon' => l.matchTypeMarathon,
  _ => l.matchTypeQuickDuel,
};
