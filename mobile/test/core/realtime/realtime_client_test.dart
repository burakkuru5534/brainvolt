import 'dart:math';

import 'package:brainvolt/core/network/token_source.dart';
import 'package:brainvolt/core/realtime/realtime_client.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/fakes.dart';

class _Tokens implements TokenSource {
  int refreshes = 0;

  @override
  Future<String?> idToken({bool forceRefresh = false}) async {
    if (forceRefresh) {
      refreshes++;
    }
    return 'token-$refreshes';
  }
}

void main() {
  late List<FakeSocket> sockets;
  late _Tokens tokens;
  late RealtimeClient client;
  final now = DateTime.fromMillisecondsSinceEpoch(1000000);

  setUp(() {
    sockets = [];
    tokens = _Tokens();
    client = RealtimeClient(
      url: Uri.parse('ws://test/v1/ws'),
      tokens: tokens,
      installationId: 'inst-1',
      now: () => now,
      random: Random(1),
      connector: (url, headers) async {
        final s = FakeSocket();
        sockets.add(s);
        return s;
      },
    );
  });

  tearDown(() => client.dispose());

  Future<void> settle() => Future<void>.delayed(Duration.zero);

  Future<FakeSocket> connect({String session = 's1'}) async {
    client.start();
    await settle();
    final s = sockets.last;
    s.serverSends({
      't': 'sys.welcome',
      'd': {
        'session_id': session,
        'resume_token': 'rt-$session',
        'live': {'match_id': 'm1'},
      },
    });
    await settle();
    return s;
  }

  test('authenticates first and reports the welcome', () async {
    final s = await connect();
    final auth = s.sent.first;
    expect(auth['t'], 'sys.auth');
    expect(auth['d'], {'token': 'token-0', 'installation_id': 'inst-1'});
    expect(client.status, RealtimeStatus.connected);
    expect(client.live['match_id'], 'm1');
  });

  test('intents resolve with sys.ack and fail with sys.error', () async {
    final s = await connect();
    final ok = client.send('match.ready', {'match_id': 'm1'});
    await settle();
    final id = s.lastSent('match.ready')['id'];
    s.serverSends({'t': 'sys.ack', 're': id});
    expect((await ok).type, 'sys.ack');

    final bad = client.send('match.answer', {'match_id': 'm1'});
    await settle();
    s.serverSends({
      't': 'sys.error',
      're': s.lastSent('match.answer')['id'],
      'd': {'code': 'ANSWER_LATE', 'message': 'late'},
    });
    await expectLater(
      bad,
      throwsA(
        isA<RealtimeError>().having((e) => e.code, 'code', 'ANSWER_LATE'),
      ),
    );
  });

  test('answers pings and estimates the server clock', () async {
    final s = await connect();
    s.serverSends({
      't': 'sys.ping',
      'd': {'ts': 1000500},
    });
    await settle();
    expect(s.lastSent('sys.pong')['d'], {'ts': 1000500, 'client_ts': 1000000});
    expect(client.clockOffsetMs, 500);
    expect(client.serverNowMs(), 1000500);
  });

  test('drops duplicate stream events, keeps snapshots', () async {
    final s = await connect();
    final got = <String>[];
    client.frames.listen((f) => got.add('${f.type}:${f.seq}'));
    for (final f in [
      {'t': 'round.started', 's': 'match:m1', 'seq': 5},
      {'t': 'round.started', 's': 'match:m1', 'seq': 5},
      {'t': 'round.answer_received', 's': 'match:m1', 'seq': 6},
      {'t': 'match.snapshot', 's': 'match:m1', 'seq': 6},
    ]) {
      s.serverSends(f);
    }
    await settle();
    expect(got, [
      'round.started:5',
      'round.answer_received:6',
      'match.snapshot:6',
    ]);
  });

  test('reconnects with resume after a drop; 4009 stops', () async {
    final s = await connect();
    s.serverSends({'t': 'round.started', 's': 'match:m1', 'seq': 9});
    await settle();
    final pending = client.send('match.ready', {'match_id': 'm1'});
    await s.serverCloses(1006);
    await expectLater(pending, throwsA(isA<RealtimeError>()));
    expect(client.status, RealtimeStatus.reconnecting);

    await Future<void>.delayed(const Duration(milliseconds: 700));
    final s2 = sockets.last;
    expect(s2, isNot(same(s)));
    expect((s2.sent.first['d'] as Map)['resume'], {
      'session_id': 's1',
      'resume_token': 'rt-s1',
      'streams': {'match:m1': 9},
    });

    // A 4003 refreshes the token before the next attempt.
    await s2.serverCloses(4003);
    await Future<void>.delayed(const Duration(milliseconds: 1400));
    expect(tokens.refreshes, 1);
    final s3 = sockets.last;
    s3.serverSends({
      't': 'sys.welcome',
      'd': {'session_id': 's1', 'resume_token': 'rt2'},
    });
    await settle();
    expect(client.status, RealtimeStatus.connected);

    await s3.serverCloses(4009);
    await settle();
    expect(client.status, RealtimeStatus.stopped);
    final count = sockets.length;
    await Future<void>.delayed(const Duration(milliseconds: 700));
    expect(sockets.length, count);
  });
}
