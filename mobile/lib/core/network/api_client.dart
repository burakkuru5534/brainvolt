import 'dart:io' show Platform;

import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../app/providers.dart';
import '../l10n/locale_controller.dart';
import 'api_exception.dart';
import 'token_source.dart';

part 'api_client.g.dart';

/// Timeouts matching the server (limits.md §3: write timeout 15 s).
const connectTimeout = Duration(seconds: 10);
const receiveTimeout = Duration(seconds: 15);

/// Header names (conventions.md §2–§5, §9).
abstract final class ApiHeaders {
  static const acceptLanguage = 'Accept-Language';
  static const appVersion = 'X-App-Version';
  static const platform = 'X-Platform';
  static const authorization = 'Authorization';
  static const idempotencyKey = 'Idempotency-Key';
}

const _retriedKey = 'brainvolt_auth_retried';

/// Builds the dio instance with BrainVolt's interceptors:
/// standard headers, the Firebase Bearer token, and one retry with a
/// refreshed token when the server answers 401 TOKEN_EXPIRED. The plain
/// (not queued) wrapper is deliberate: retrying from inside a queued error
/// handler deadlocks dio; Firebase itself serialises token refreshes.
Dio buildDio({
  required Uri baseUrl,
  required String appVersion,
  required String platform,
  required String Function() locale,
  required TokenSource tokens,
  HttpClientAdapter? adapter,
}) {
  final dio = Dio(
    BaseOptions(
      baseUrl: baseUrl.toString(),
      connectTimeout: connectTimeout,
      receiveTimeout: receiveTimeout,
      sendTimeout: receiveTimeout,
      contentType: 'application/json',
      responseType: ResponseType.json,
    ),
  );
  if (adapter != null) {
    dio.httpClientAdapter = adapter;
  }
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) async {
        options.headers[ApiHeaders.acceptLanguage] = locale();
        options.headers[ApiHeaders.appVersion] = appVersion;
        options.headers[ApiHeaders.platform] = platform;
        final token = await tokens.idToken(
          forceRefresh: options.extra[_retriedKey] == true,
        );
        if (token != null) {
          options.headers[ApiHeaders.authorization] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        final options = error.requestOptions;
        final expired =
            error.response?.statusCode == 401 &&
            ApiException.fromDio(error).code == 'TOKEN_EXPIRED';
        if (!expired || options.extra[_retriedKey] == true) {
          handler.next(error);
          return;
        }
        options.extra[_retriedKey] = true;
        try {
          handler.resolve(await dio.fetch<dynamic>(options));
        } on DioException catch (retryError) {
          handler.next(retryError);
        }
      },
    ),
  );
  return dio;
}

/// Thin JSON client over dio; every failure is an [ApiException].
class ApiClient {
  ApiClient(this._dio);

  final Dio _dio;

  /// GET returning the decoded JSON object.
  Future<Map<String, dynamic>> getJson(String path) =>
      _send(() => _dio.get<dynamic>(path));

  /// POST; [idempotencyKey] for creating/spending calls (conventions.md §9).
  /// Returns null for 204 No Content.
  Future<Map<String, dynamic>?> postJson(
    String path, {
    Object? body,
    String? idempotencyKey,
  }) => _sendNullable(
    () => _dio.post<dynamic>(
      path,
      data: body,
      options: Options(headers: {ApiHeaders.idempotencyKey: ?idempotencyKey}),
    ),
  );

  /// PATCH returning the decoded JSON object.
  Future<Map<String, dynamic>> patchJson(String path, {required Object body}) =>
      _send(() => _dio.patch<dynamic>(path, data: body));

  /// DELETE; null for 204 No Content.
  Future<Map<String, dynamic>?> delete(String path) =>
      _sendNullable(() => _dio.delete<dynamic>(path));

  Future<Map<String, dynamic>> _send(
    Future<Response<dynamic>> Function() call,
  ) async {
    final data = await _sendNullable(call);
    if (data == null) {
      throw const ApiException(code: ApiException.unknownCode);
    }
    return data;
  }

  Future<Map<String, dynamic>?> _sendNullable(
    Future<Response<dynamic>> Function() call,
  ) async {
    try {
      final response = await call();
      final data = response.data;
      if (data == null || data == '') {
        return null;
      }
      if (data is Map<String, dynamic>) {
        return data;
      }
      throw const ApiException(code: ApiException.unknownCode);
    } on DioException catch (e) {
      throw ApiException.fromDio(e);
    }
  }
}

/// Firebase ID tokens; signed out until roadmap 1.8 replaces it.
@Riverpod(keepAlive: true)
TokenSource tokenSource(Ref ref) => const SignedOutTokenSource();

/// The API client of the running app.
@Riverpod(keepAlive: true)
ApiClient apiClient(Ref ref) => ApiClient(
  buildDio(
    baseUrl: ref.watch(appConfigProvider).apiBaseUrl,
    appVersion: ref.watch(appVersionProvider),
    platform: Platform.isIOS ? 'ios' : 'android',
    locale: () =>
        effectiveLocale(ref.read(localeOverrideProvider)).languageCode,
    tokens: ref.watch(tokenSourceProvider),
  ),
);
