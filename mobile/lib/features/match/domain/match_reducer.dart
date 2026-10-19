import '../../../core/realtime/realtime_client.dart';
import 'match_state.dart';

int _int(Object? v) => (v as num?)?.toInt() ?? 0;
String? _str(Object? v) => v as String?;
List<Map<String, dynamic>> _maps(Object? v) => [
  for (final x in (v as List?) ?? const []) (x as Map).cast<String, dynamic>(),
];

List<Player> _players(Object? list, List<Player> known) => [
  for (final p in _maps(list))
    Player(
      userId: p['user_id'] as String,
      side: _int(p['side_no']),
      displayName: _str(p['display_name']) ?? '',
      avatarId: _str(p['avatar_id']) ?? '',
      status:
          _str(p['status']) ??
          known.where((k) => k.userId == p['user_id']).firstOrNull?.status ??
          'joined',
      ready:
          _str(p['status']) == 'ready' ||
          (known.where((k) => k.userId == p['user_id']).firstOrNull?.ready ??
              false),
    ),
];

RoundState _round(Map<String, dynamic> d) {
  final q = (d['question'] as Map?)?.cast<String, dynamic>() ?? const {};
  final cat = (q['category'] as Map?)?.cast<String, dynamic>();
  return RoundState(
    roundNo: _int(d['round_no']),
    prompt: _str(q['prompt']) ?? '',
    category: _str(cat?['name']),
    options: [
      for (final o in _maps(d['options']))
        AnswerOption(o['id'] as String, _str(o['text']) ?? ''),
    ],
    showAt: _int(d['show_at']),
    deadlineAt: _int(d['deadline_at']),
    myOptionId: _str(d['own_option_id']),
  );
}

MatchPhase _phaseOf(String? status, String? phase) => switch (status) {
  'waiting' => MatchPhase.waiting,
  'ready' => MatchPhase.countdown,
  'completed' => MatchPhase.completed,
  'cancelled' => MatchPhase.cancelled,
  _ => switch (phase) {
    'reveal' => MatchPhase.reveal,
    'pre_round' => MatchPhase.preRound,
    _ => MatchPhase.question,
  },
};

PreRound _preRound(Map<String, dynamic> d) => PreRound(
  roundNo: _int(d['round_no']),
  pickerUserId: _str(d['picker_user_id']),
  categories: [
    for (final c in _maps(d['categories']))
      (c['id'] as String, _str(c['name']) ?? ''),
  ],
  allowed: {
    for (final id in (d['allowed_category_ids'] as List?) ?? const [])
      id as String,
  },
  deadlineAt: _int(d['pick_deadline_at']),
);

/// Applies one match-stream event to the state (websocket.md §5.2).
/// `match.snapshot` replaces everything (client rule: discard local
/// assumptions, websocket.md §6).
MatchState reduce(MatchState s, ServerFrame f) {
  final d = f.data;
  switch (f.type) {
    case 'match.created':
      return s.copyWith(
        matchType: _str(d['match_type']),
        phase: MatchPhase.waiting,
        questionCount: _int(d['question_count']),
        readyDeadlineAt: _int(d['ready_deadline_at']),
        players: _players(d['participants'], s.players),
      );
    case 'match.snapshot':
      final sides = _maps(d['sides']);
      final result = (d['result'] as Map?)?.cast<String, dynamic>();
      final round = d['round'] is Map
          ? _round((d['round'] as Map).cast<String, dynamic>())
          : null;
      return MatchState(
        matchId: s.matchId,
        myUserId: s.myUserId,
        matchType: _str(d['match_type']) ?? s.matchType,
        phase: _phaseOf(_str(d['status']), _str(d['phase'])),
        questionCount: _int(d['question_count']),
        readyDeadlineAt: _int(d['ready_deadline_at']),
        startsAt: _int(d['starts_at']),
        players: _players(d['participants'], s.players),
        scores: [
          for (var side = 1; side <= 2; side++)
            _int(
              sides
                  .where((x) => _int(x['side_no']) == side)
                  .firstOrNull?['score'],
            ),
        ],
        round: round,
        preRound: d['pre_round'] is Map
            ? _preRound((d['pre_round'] as Map).cast<String, dynamic>())
            : null,
        endReason: _str(result?['end_reason']),
        cancelReason: _str(result?['cancel_reason']),
        winnerSide: result == null ? null : _int(result['winner_side_no']),
        isDraw: result?['is_draw'] == true,
      );
    case 'match.participant_ready':
      return s.copyWith(
        players: [
          for (final p in s.players)
            p.userId == d['user_id'] ? p.copyWith(ready: true) : p,
        ],
      );
    case 'match.starting':
      return s.copyWith(
        phase: MatchPhase.countdown,
        startsAt: _int(d['starts_at']),
      );
    case 'match.started':
      return s.copyWith(
        players: [for (final p in s.players) p.copyWith(status: 'playing')],
      );
    case 'round.pre_round':
      return s.copyWith(
        phase: MatchPhase.preRound,
        preRound: _preRound(d),
        clearRound: true,
      );
    case 'round.category_picked':
      return s.copyWith(clearPreRound: true);
    case 'round.started':
      return s.copyWith(
        phase: MatchPhase.question,
        round: _round(d),
        clearPreRound: true,
      );
    case 'round.answer_received':
      final r = s.round;
      if (r == null || r.roundNo != _int(d['round_no'])) {
        return s;
      }
      final user = d['user_id'] as String;
      return s.copyWith(
        round: r.copyWith(
          answered: {...r.answered, user},
          myOptionId: user == s.myUserId ? _str(d['option_id']) : null,
        ),
      );
    case 'round.locked':
      final r = s.round;
      return r == null ? s : s.copyWith(round: r.copyWith(locked: true));
    case 'round.revealed':
      final r = s.round;
      if (r == null) {
        return s;
      }
      final scores = List.of(s.scores);
      for (final side in _maps(d['sides'])) {
        scores[_int(side['side_no']) - 1] = _int(side['score']);
      }
      final cat = (d['category'] as Map?)?.cast<String, dynamic>();
      return s.copyWith(
        phase: MatchPhase.reveal,
        scores: scores,
        round: r.copyWith(
          locked: true,
          correctOptionId: _str(d['correct_option_id']),
          explanation: _str(d['explanation']),
          category: _str(cat?['name']),
          revealed: {
            for (final a in _maps(d['answers']))
              a['user_id'] as String: RevealedAnswer(
                optionId: _str(a['option_id']),
                correct: a['correct'] == true,
                points: _int((a['points'] as Map?)?['total']),
                responseMs: (a['response_ms'] as num?)?.toInt(),
              ),
          },
        ),
      );
    case 'match.participant_status':
      return s.copyWith(
        players: [
          for (final p in s.players)
            p.userId == d['user_id']
                ? p.copyWith(status: _str(d['status']))
                : p,
        ],
      );
    case 'match.completed':
      final scores = List.of(s.scores);
      for (final side in _maps(d['sides'])) {
        scores[_int(side['side_no']) - 1] = _int(side['score']);
      }
      return s.copyWith(
        phase: MatchPhase.completed,
        scores: scores,
        endReason: _str(d['end_reason']),
        winnerSide: _int(d['winner_side_no']),
        isDraw: d['is_draw'] == true,
      );
    case 'match.cancelled':
      return s.copyWith(
        phase: MatchPhase.cancelled,
        cancelReason: _str(d['reason']),
      );
    case 'match.seat_moved':
      return s.copyWith(seatMoved: true);
    case 'match.resumed':
      return s.copyWith(resumed: true);
  }
  return s;
}
