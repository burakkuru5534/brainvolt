import 'package:freezed_annotation/freezed_annotation.dart';

part 'api_error_body.freezed.dart';
part 'api_error_body.g.dart';

/// The error envelope of every non-2xx response (conventions.md §6).
@freezed
abstract class ApiErrorBody with _$ApiErrorBody {
  const factory ApiErrorBody({required ApiErrorDetail error}) = _ApiErrorBody;

  factory ApiErrorBody.fromJson(Map<String, dynamic> json) =>
      _$ApiErrorBodyFromJson(json);
}

/// `error` object of the envelope.
@freezed
abstract class ApiErrorDetail with _$ApiErrorDetail {
  const factory ApiErrorDetail({
    required String code,
    required String message,
    @JsonKey(name: 'request_id') required String requestId,
    Map<String, dynamic>? details,
  }) = _ApiErrorDetail;

  factory ApiErrorDetail.fromJson(Map<String, dynamic> json) =>
      _$ApiErrorDetailFromJson(json);
}
