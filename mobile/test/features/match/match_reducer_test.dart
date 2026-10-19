import 'package:brainvolt/core/realtime/realtime_client.dart';
import 'package:brainvolt/features/match/domain/match_reducer.dart';
import 'package:brainvolt/features/match/domain/match_state.dart';
import 'package:flutter_test/flutter_test.dart';

ServerFrame f(String type, Map<String, dynamic> d) =>
    ServerFrame(type: type, data: d, stream: 'match:m1');

void main() {
  const start = MatchState(matchId: 'm1', myUserId: 'me');

  test('a snapshot rebuilds an open round with the own answer', () {
    final s = reduce(
      start,
      f('match.snapshot', {
        'status': 'in_progress',
        'phase': 'question',
        'question_count': 7,
        'participants': [
          {'user_id': 'me', 'side_no': 2, 'status': 'playing'},
          {'user_id': 'op', 'side_no': 1, 'status': 'disconnected'},
        ],
        'sides': [
          {'side_no': 1, 'score': 300},
          {'side_no': 2, 'score': 120},
        ],
        'round': {
          'round_no': 3,
          'question': {'id': 'q', 'prompt': 'P'},
          'options': [
            {'id': 'a', 'text': 'A'},
          ],
          'show_at': 10,
          'deadline_at': 20,
          'own_option_id': 'a',
        },
      }),
    );
    expect(s.phase, MatchPhase.question);
    expect(s.mySide, 2);
    expect(s.myScore, 120);
    expect(s.opponentScore, 300);
    expect(s.opponent?.status, 'disconnected');
    expect(s.round?.myOptionId, 'a');
    expect(s.round?.roundNo, 3);
  });

  test('answers of others never reveal the option; cancel ends it', () {
    var s = reduce(
      start,
      f('round.started', {
        'round_no': 1,
        'question': {'id': 'q', 'prompt': 'P'},
        'options': <Object>[],
        'show_at': 1,
        'deadline_at': 2,
      }),
    );
    s = reduce(s, f('round.answer_received', {'round_no': 1, 'user_id': 'op'}));
    expect(s.round?.answered, {'op'});
    expect(s.round?.myOptionId, isNull);
    s = reduce(s, f('match.cancelled', {'reason': 'not_all_ready'}));
    expect(s.over, isTrue);
    expect(s.cancelReason, 'not_all_ready');
  });

  test('the match type comes from match.created and snapshots', () {
    var s = reduce(
      start,
      f('match.created', {'match_type': 'sudden_death', 'question_count': 0}),
    );
    expect(s.matchType, 'sudden_death');
    s = reduce(
      s,
      f('match.snapshot', {'status': 'waiting', 'match_type': 'marathon'}),
    );
    expect(s.matchType, 'marathon');
  });
}
