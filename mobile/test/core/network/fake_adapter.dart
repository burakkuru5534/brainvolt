import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';

/// A canned HTTP answer.
class FakeResponse {
  const FakeResponse(this.status, [this.body, this.headers = const {}]);

  final int status;
  final Object? body;
  final Map<String, List<String>> headers;
}

/// dio adapter that never touches the network: answers come from [respond]
/// and every request is recorded.
class FakeAdapter implements HttpClientAdapter {
  FakeAdapter(this.respond);

  /// Returns the answer for a request; throwing a [DioException] simulates
  /// timeouts or connection failures.
  final FakeResponse Function(RequestOptions options) respond;
  final requests = <RequestOptions>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    // Snapshot: dio reuses the same RequestOptions when it retries.
    requests.add(options.copyWith(headers: Map.of(options.headers)));
    final r = respond(options);
    final body = r.body == null ? '' : jsonEncode(r.body);
    return ResponseBody.fromString(
      body,
      r.status,
      headers: {
        if (r.body != null)
          Headers.contentTypeHeader: ['application/json; charset=utf-8'],
        ...r.headers,
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

/// The error envelope the backend sends (conventions.md §6).
Map<String, Object?> envelope(
  String code, {
  String message = 'server text',
  Map<String, Object?>? details,
}) => {
  'error': {
    'code': code,
    'message': message,
    'request_id': '0192c7e0-0000-7000-8000-000000000001',
    'details': ?details,
  },
};
