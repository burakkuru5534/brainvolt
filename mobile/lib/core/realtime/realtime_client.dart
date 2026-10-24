import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:logging/logging.dart';
import 'package:web_socket_channel/io.dart';

import '../network/token_source.dart';

final _log = Logger('RealtimeClient');

/// A text WebSocket as [RealtimeClient] needs it; tests use a fake.
abstract interface class SocketConnection {
  Stream<dynamic> get stream;
  void add(String data);
  Future<void> close([int? code, String? reason]);

  /// The close code once the server closed the socket.
  int? get closeCode;
}

/// Opens a connection to [url] with the handshake [headers].
typedef SocketConnector = Future<SocketConnection> Function(
  Uri url,
  Map<String, String> headers,
);

/// The real connector (web_socket_channel, ADR-0003).
Future<SocketConnection> ioConnector(
  Uri url,
  Map<String, String> headers,
) async {
  final channel = IOWebSocketChannel.connect(
    url,
    headers: headers,
    connectTimeout: const Duration(seconds: 10),
  );
  await channel.ready;
  return _ChannelConnection(channel);
}

class _ChannelConnection implements SocketConnection {
  _ChannelConnection(this._channel);

  final IOWebSocketChannel _channel;

  @override
  Stream<dynamic> get stream => _channel.stream;

  @override
  void add(String data) => _channel.sink.add(data);

  @override
  Future<void> close([int? code, String? reason]) =>
      _channel.sink.close(code, reason);

  @override
  int? get closeCode => _channel.closeCode;
}

/// A server → client message (websocket.md §3).
class ServerFrame {
  const ServerFrame({
    required this.type,
    required this.data,
    this.stream,
    this.seq,
    this.ts = 0,
    this.re,
  });

  factory ServerFrame.fromJson(Map<String, dynamic> j) => ServerFrame(
    type: j['t'] as String? ?? '',
    stream: j['s'] as String?,
    seq: (j['seq'] as num?)?.toInt(),
    ts: (j['ts'] as num?)?.toInt() ?? 0,
    re: j['re'] as String?,
    data: (j['d'] as Map?)?.cast<String, dynamic>() ?? const {},
  );

  final String type;
  final String? stream;
  final int? seq;
  final int ts;
  final String? re;
  final Map<String, dynamic> data;
}

/// A rejected intent (`sys.error`) or a send without connection.
class RealtimeError implements Exception {
  const RealtimeError(this.code, [this.message = '']);

  /// No connection when the intent was sent, or no answer in time.
  static const disconnected = 'DISCONNECTED';

  final String code;
  final String message;

  @override
  String toString() => 'RealtimeError($code: $message)';
}

/// Connection state shown by the UI (reconnect banner).
enum RealtimeStatus { idle, connecting, connected, reconnecting, stopped }

/// A screen's claim on the WebSocket (D-097): the socket is open only while
/// at least one lease is held and the app is in the foreground. [release]
/// is idempotent.
class RealtimeLease {
  RealtimeLease._(this._onRelease);

  final void Function() _onRelease;
  var _released = false;

  void release() {
    if (_released) {
      return;
    }
    _released = true;
    _onRelease();
  }
}

/// The app's single WebSocket (websocket.md, ADR-0003): authentication,
/// heartbeat and clock offset, intent acknowledgements, per-stream
/// sequence numbers with duplicate dropping, and reconnect with backoff and
/// resume.
///
/// The socket is open only while it is needed (D-097, cost: an open
/// WebSocket keeps the server billing): screens that need live events hold
/// a [RealtimeLease]; with none left it closes after [idleGrace], and it
/// closes at once when the app leaves the foreground. A later lease or the
/// return to the foreground reconnects and resumes the session.
class RealtimeClient {
  RealtimeClient({
    required this.url,
    required this.tokens,
    required this.installationId,
    this.headers = const {},
    this.connector = ioConnector,
    this.idleGrace = defaultIdleGrace,
    DateTime Function()? now,
    Random? random,
  }) : _now = now ?? DateTime.now,
       _random = random ?? Random();

  final Uri url;
  final TokenSource tokens;
  final String installationId;
  final Map<String, String> headers;
  final SocketConnector connector;
  final DateTime Function() _now;
  final Random _random;

  /// Backoff steps (ADR-0003); the last one repeats.
  static const backoff = [500, 1000, 2000, 4000, 8000, 10000];

  /// Close codes after which the client does not reconnect by itself.
  static const _final = {4004, 4009, 4026};

  /// Re-sends the token well before Firebase's one-hour expiry.
  static const reauthEvery = Duration(minutes: 45);

  /// How long an intent waits for its sys.ack / sys.error.
  static const ackTimeout = Duration(seconds: 10);

  /// How long the socket stays open after the last lease is released, so
  /// going from one screen to the next does not reconnect.
  static const defaultIdleGrace = Duration(seconds: 5);

  /// [defaultIdleGrace], shorter in tests.
  final Duration idleGrace;

  /// The server closed an idle session (D-097): reconnect only on demand.
  static const closeIdle = 4010;

  final _frames = StreamController<ServerFrame>.broadcast();
  final _status = StreamController<RealtimeStatus>.broadcast();
  final _pending = <String, Completer<ServerFrame>>{};
  final _lastSeq = <String, int>{};
  final _offsets = <int>[];

  SocketConnection? _conn;
  StreamSubscription<dynamic>? _sub;
  Timer? _retry;
  Timer? _reauth;
  int _attempt = 0;
  int _nextId = 0;
  bool _running = false;
  bool _refreshToken = false;
  int _leases = 0;
  bool _pinned = false;
  bool _foreground = true;
  Timer? _idle;
  String? _sessionId;
  String? _resumeToken;

  RealtimeStatus _state = RealtimeStatus.idle;

  /// Events from the server except protocol messages (`sys.*`).
  Stream<ServerFrame> get frames => _frames.stream;

  /// Connection state changes.
  Stream<RealtimeStatus> get statusChanges => _status.stream;

  RealtimeStatus get status => _state;

  /// What the user is part of, from the last sys.welcome (`match_id`, …).
  Map<String, dynamic> live = const {};

  /// Estimated server time (timing.md §6): used only to show questions at
  /// `show_at` and to draw countdowns.
  int serverNowMs() => _now().millisecondsSinceEpoch + clockOffsetMs;

  /// Server clock minus local clock, from the least-delayed ping of the
  /// last ten.
  int get clockOffsetMs => _offsets.isEmpty ? 0 : _offsets.reduce(max);

  /// Takes a lease (D-097): connects if needed and keeps the socket open
  /// until every lease is released; [statusChanges] tells when it is
  /// connected. [owner] is for logs.
  RealtimeLease acquire([String owner = '']) {
    _leases++;
    _log.fine('lease +$owner ($_leases)');
    _update();
    return RealtimeLease._(() {
      _leases--;
      _log.fine('lease -$owner ($_leases)');
      _update();
    });
  }

  /// Keeps the socket wanted with no lease (tests and tools); [stop] ends it.
  void start() {
    _pinned = true;
    _update();
  }

  /// Number of leases held (tests).
  int get leases => _leases;

  /// The app went to the background (false) or came back (true): the
  /// socket closes at once in the background and reconnects on return if
  /// a lease is still held.
  void setForeground({required bool foreground}) {
    if (_foreground == foreground) {
      return;
    }
    _foreground = foreground;
    _update();
  }

  bool get _wanted => (_pinned || _leases > 0) && _foreground;

  void _update() {
    if (_wanted) {
      _idle?.cancel();
      _idle = null;
      if (!_running) {
        _running = true;
        _attempt = 0;
        unawaited(_connect());
      }
      return;
    }
    if (!_running) {
      return;
    }
    if (!_foreground) {
      unawaited(_suspend());
      return;
    }
    _idle ??= Timer(idleGrace, () {
      _idle = null;
      if (!_wanted) {
        unawaited(_suspend());
      }
    });
  }

  /// Closes the socket while nothing needs it; the session id, resume token
  /// and stream positions stay, so the next connect resumes.
  Future<void> _suspend() async {
    _running = false;
    _idle?.cancel();
    _idle = null;
    _retry?.cancel();
    _reauth?.cancel();
    _failPending();
    final conn = _conn;
    final sub = _sub;
    _conn = null;
    _sub = null;
    _setStatus(RealtimeStatus.idle);
    unawaited(conn?.close(1000));
    await sub?.cancel();
  }

  /// Closes the socket for good (sign-out).
  Future<void> stop() async {
    _pinned = false;
    _idle?.cancel();
    _idle = null;
    _running = false;
    _retry?.cancel();
    _reauth?.cancel();
    _setStatus(RealtimeStatus.stopped);
    _failPending();
    await _sub?.cancel();
    await _conn?.close(1000);
    _conn = null;
  }

  /// Sends an intent and waits for its acknowledgement; a `sys.error`
  /// throws [RealtimeError] with the server's code.
  Future<ServerFrame> send(String type, Map<String, dynamic> data) {
    final conn = _conn;
    if (conn == null || _state != RealtimeStatus.connected) {
      return Future.error(const RealtimeError(RealtimeError.disconnected));
    }
    final id = 'c${++_nextId}';
    final done = Completer<ServerFrame>();
    _pending[id] = done;
    conn.add(jsonEncode({'t': type, 'id': id, 'd': data}));
    return done.future.timeout(
      ackTimeout,
      onTimeout: () {
        _pending.remove(id);
        throw const RealtimeError(RealtimeError.disconnected);
      },
    );
  }

  /// Forgets a stream's position (the match is over).
  void forget(String stream) => _lastSeq.remove(stream);

  Future<void> _connect() async {
    _setStatus(
      _attempt == 0 ? RealtimeStatus.connecting : RealtimeStatus.reconnecting,
    );
    try {
      final token = await tokens.idToken(forceRefresh: _refreshToken);
      _refreshToken = false;
      if (token == null) {
        throw StateError('not signed in');
      }
      final conn = await connector(url, headers);
      if (!_running) {
        await conn.close(1000);
        return;
      }
      _conn = conn;
      _sub = conn.stream.listen(
        _onMessage,
        onDone: () => _onClosed(conn),
        onError: (Object e) => _onClosed(conn),
      );
      final auth = <String, dynamic>{
        'token': token,
        'installation_id': installationId,
      };
      if (_sessionId != null && _resumeToken != null) {
        auth['resume'] = {
          'session_id': _sessionId,
          'resume_token': _resumeToken,
          'streams': Map.of(_lastSeq),
        };
      }
      conn.add(jsonEncode({'t': 'sys.auth', 'id': 'auth', 'd': auth}));
    } on Object catch (e) {
      _log.info('connect failed: $e');
      _scheduleRetry();
    }
  }

  void _onMessage(dynamic raw) {
    if (raw is! String) {
      return;
    }
    final Map<String, dynamic> json;
    try {
      json = jsonDecode(raw) as Map<String, dynamic>;
    } on FormatException {
      return;
    }
    final f = ServerFrame.fromJson(json);
    switch (f.type) {
      case 'sys.welcome':
        _sessionId = f.data['session_id'] as String?;
        _resumeToken = f.data['resume_token'] as String?;
        live = (f.data['live'] as Map?)?.cast<String, dynamic>() ?? const {};
        _attempt = 0;
        _setStatus(RealtimeStatus.connected);
        _reauth?.cancel();
        _reauth = Timer.periodic(reauthEvery, (_) => _sendReauth());
        _frames.add(f);
        return;
      case 'sys.ping':
        final ts = (f.data['ts'] as num?)?.toInt() ?? f.ts;
        final local = _now().millisecondsSinceEpoch;
        _offsets.add(ts - local);
        if (_offsets.length > 10) {
          _offsets.removeAt(0);
        }
        _conn?.add(
          jsonEncode({
            't': 'sys.pong',
            'd': {'ts': ts, 'client_ts': local},
          }),
        );
        return;
      case 'sys.ack' || 'sys.error':
        final done = _pending.remove(f.re);
        if (done == null) {
          return;
        }
        if (f.type == 'sys.ack') {
          done.complete(f);
        } else {
          done.completeError(
            RealtimeError(
              f.data['code'] as String? ?? 'INTERNAL',
              f.data['message'] as String? ?? '',
            ),
          );
        }
        return;
    }
    final stream = f.stream;
    final seq = f.seq;
    if (stream != null && seq != null) {
      final last = _lastSeq[stream] ?? 0;
      // Snapshots replace the screen state: match.snapshot always passes;
      // room.state (a full room) passes again at the same seq, because a
      // screen opened later asks for it with room.join.
      final stale = switch (f.type) {
        'match.snapshot' => false,
        'room.state' => seq < last,
        _ => seq <= last,
      };
      if (stale) {
        return; // duplicate (websocket.md §6)
      }
      _lastSeq[stream] = seq;
    }
    _frames.add(f);
  }

  Future<void> _sendReauth() async {
    final token = await tokens.idToken(forceRefresh: true);
    if (token == null) {
      return;
    }
    try {
      await send('sys.reauth', {'token': token});
    } on RealtimeError catch (e) {
      _log.info('reauth failed: $e');
    }
  }

  void _onClosed(SocketConnection conn) {
    if (!identical(conn, _conn)) {
      return;
    }
    _conn = null;
    _reauth?.cancel();
    _failPending();
    final code = conn.closeCode;
    if (!_running) {
      return;
    }
    if (code != null && _final.contains(code)) {
      _running = false;
      _setStatus(RealtimeStatus.stopped);
      return;
    }
    if (code == closeIdle) {
      // Nothing was going on: reconnect when a screen asks again.
      _running = false;
      _setStatus(RealtimeStatus.idle);
      return;
    }
    if (code == 4002 || code == 4003) {
      _refreshToken = true;
    }
    _scheduleRetry();
  }

  void _scheduleRetry() {
    if (!_running) {
      return;
    }
    _setStatus(RealtimeStatus.reconnecting);
    final base = backoff[min(_attempt, backoff.length - 1)];
    _attempt++;
    final jitter = (base * 0.2 * _random.nextDouble()).round();
    _retry?.cancel();
    _retry = Timer(Duration(milliseconds: base + jitter), () {
      if (_running) {
        unawaited(_connect());
      }
    });
  }

  void _failPending() {
    for (final c in _pending.values) {
      c.completeError(const RealtimeError(RealtimeError.disconnected));
    }
    _pending.clear();
  }

  void _setStatus(RealtimeStatus s) {
    if (_state != s) {
      _state = s;
      _status.add(s);
    }
  }

  /// Releases the client (provider disposal).
  Future<void> dispose() async {
    await stop();
    await _frames.close();
    await _status.close();
  }
}
