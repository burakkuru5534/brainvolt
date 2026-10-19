/// Reports, sanctions and appeals (rest.md §11, schema-moderation).
library;

/// Reasons a player can give for a question (schema-moderation §2).
const questionReportReasons = [
  'wrong_answer',
  'ambiguous',
  'outdated',
  'typo_or_translation',
  'offensive_content',
  'other',
];

/// A sanction or warning on the player's account.
class Sanction {
  const Sanction({
    required this.id,
    required this.kind,
    required this.reasonCode,
    required this.active,
    this.endsAt,
    this.appealStatus,
  });

  factory Sanction.fromJson(Map<String, dynamic> j) => Sanction(
    id: j['id'] as String,
    kind: j['kind'] as String,
    reasonCode: j['reason_code'] as String? ?? '',
    active: j['active'] == true,
    endsAt: DateTime.tryParse(j['ends_at'] as String? ?? '')?.toLocal(),
    appealStatus: j['appeal_status'] as String?,
  );

  final String id;

  /// warning / chat_mute / ranked_ban / matchmaking_ban / social_ban /
  /// suspension.
  final String kind;
  final String reasonCode;
  final bool active;

  /// null = permanent (or a warning).
  final DateTime? endsAt;

  /// pending / accepted / rejected, null when not appealed.
  final String? appealStatus;
}

abstract interface class ModerationRepository {
  Future<void> reportQuestion({
    required String questionId,
    required String reason,
    String? matchId,
    String? comment,
  });

  Future<List<Sanction>> mySanctions();

  Future<void> appeal(String sanctionId, String text);
}
