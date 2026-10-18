// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'api_error_body.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_ApiErrorBody _$ApiErrorBodyFromJson(Map<String, dynamic> json) =>
    _ApiErrorBody(
      error: ApiErrorDetail.fromJson(json['error'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$ApiErrorBodyToJson(_ApiErrorBody instance) =>
    <String, dynamic>{'error': instance.error};

_ApiErrorDetail _$ApiErrorDetailFromJson(Map<String, dynamic> json) =>
    _ApiErrorDetail(
      code: json['code'] as String,
      message: json['message'] as String,
      requestId: json['request_id'] as String,
      details: json['details'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$ApiErrorDetailToJson(_ApiErrorDetail instance) =>
    <String, dynamic>{
      'code': instance.code,
      'message': instance.message,
      'request_id': instance.requestId,
      'details': instance.details,
    };
