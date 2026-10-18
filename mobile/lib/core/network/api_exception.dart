import 'package:dio/dio.dart';

import '../l10n/l10n.dart';
import 'api_error_body.dart';
import 'error_messages.g.dart';

/// A failed API call. [code] is a server code from docs/api/errors.md, or
/// one of the client-side codes below when no server answer arrived.
class ApiException implements Exception {
  const ApiException({
    required this.code,
    this.status,
    this.serverMessage,
    this.details = const {},
    this.requestId,
    this.retryAfter,
  });

  /// Builds an [ApiException] from a dio failure.
  factory ApiException.fromDio(DioException e) {
    final response = e.response;
    final data = response?.data;
    if (response != null &&
        data is Map<String, dynamic> &&
        data['error'] is Map<String, dynamic>) {
      try {
        final body = ApiErrorBody.fromJson(data).error;
        return ApiException(
          code: body.code,
          status: response.statusCode,
          serverMessage: body.message,
          details: body.details ?? const {},
          requestId: body.requestId,
          retryAfter: _retryAfter(response.headers.value('retry-after')),
        );
      } on Object {
        // A malformed envelope is reported as UNKNOWN below.
      }
    }
    final code = switch (e.type) {
      DioExceptionType.connectionTimeout ||
      DioExceptionType.sendTimeout ||
      DioExceptionType.receiveTimeout => timeoutCode,
      DioExceptionType.connectionError => networkCode,
      _ => unknownCode,
    };
    return ApiException(code: code, status: response?.statusCode);
  }

  /// The server could not be reached.
  static const networkCode = 'NETWORK_ERROR';

  /// The server did not answer in time.
  static const timeoutCode = 'TIMEOUT';

  /// Anything else without a usable error envelope.
  static const unknownCode = 'UNKNOWN';

  final String code;
  final int? status;

  /// The server's localized fallback text (conventions.md §6).
  final String? serverMessage;
  final Map<String, dynamic> details;
  final String? requestId;
  final Duration? retryAfter;

  /// Text to show the user: the app's own translation for known codes, else
  /// the server's message, else a generic text.
  String localizedMessage(AppLocalizations l) => switch (code) {
    networkCode => l.errorNetwork,
    timeoutCode => l.errorTimeout,
    _ => serverErrorMessage(l, code) ?? serverMessage ?? l.errorUnknown,
  };

  static Duration? _retryAfter(String? header) {
    final seconds = int.tryParse(header ?? '');
    return seconds == null ? null : Duration(seconds: seconds);
  }

  @override
  String toString() =>
      'ApiException($code, status: $status, request: $requestId)';
}
