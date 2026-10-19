import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/match_repository.dart';

int _int(Object? v) => (v as num?)?.toInt() ?? 0;
List<Map<String, dynamic>> _maps(Object? v) => [
  for (final x in (v as List?) ?? const []) (x as Map).cast<String, dynamic>(),
];

/// [MatchRepository] over the REST API.
class ApiMatchRepository implements MatchRepository {
  const ApiMatchRepository(this.api);

  final ApiClient api;

  @override
  Future<QueueTicket> enqueue({
    bool practice = false,
    QueueMode mode = const QueueMode(),
  }) async {
    final j = await api.postJson(
      '/v1/matchmaking/tickets',
      body: {
        'match_type': mode.matchType,
        'format': mode.format,
        if (mode.partyUserIds.isNotEmpty) 'party_user_ids': mode.partyUserIds,
        'team_id': ?mode.teamId,
        'category_mode': ?mode.categoryMode,
        'category_id': ?mode.categoryId,
        if (practice) 'practice': true,
      },
      idempotencyKey: const Uuid().v7(),
    );
    return QueueTicket(
      id: j!['id'] as String,
      state: j['state'] as String,
      matchId: j['match_id'] as String?,
    );
  }

  @override
  Future<void> leaveQueue() => api.delete('/v1/matchmaking/tickets/current');

  @override
  Future<String?> currentMatch() async {
    final j = await api.getJson('/v1/matches/current');
    return (j['match'] as Map?)?['id'] as String?;
  }

  @override
  Future<MatchResult> result(String matchId) async {
    final j = await api.getJson('/v1/matches/$matchId');
    final sides = _maps(j['sides']);
    return MatchResult(
      id: j['id'] as String,
      status: j['status'] as String,
      origin: j['origin'] as String? ?? '',
      endedAt: DateTime.tryParse(j['ended_at'] as String? ?? ''),
      winnerSide: (j['winner_side_no'] as num?)?.toInt(),
      isDraw: j['is_draw'] == true,
      endReason: j['end_reason'] as String?,
      cancelReason: j['cancel_reason'] as String?,
      scores: [
        for (var side = 1; side <= 2; side++)
          _int(
            sides
                .where((s) => _int(s['side_no']) == side)
                .firstOrNull?['score'],
          ),
      ],
      players: [
        for (final p in _maps(j['participants']))
          ResultPlayer(
            userId: p['user_id'] as String,
            side: _int(p['side_no']),
            displayName: p['display_name'] as String? ?? '',
            avatarId: p['avatar_id'] as String? ?? '',
            correct: _int(p['correct_count']),
            status: p['status'] as String? ?? '',
          ),
      ],
      rounds: [
        for (final r in _maps(j['rounds']))
          ResultRound(
            roundNo: _int(r['round_no']),
            prompt: r['prompt'] as String? ?? '',
            options: {
              for (final o in _maps(r['options']))
                o['id'] as String: o['text'] as String? ?? '',
            },
            correctText:
                _maps(r['options'])
                        .where((o) => o['id'] == r['correct_option_id'])
                        .firstOrNull?['text']
                    as String? ??
                '',
            answers: [
              for (final a in _maps(r['answers']))
                ResultAnswer(
                  userId: a['user_id'] as String,
                  optionId: a['option_id'] as String?,
                  correct: a['correct'] == true,
                  points: _int(a['points']),
                ),
            ],
          ),
      ],
    );
  }
}
