// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'credit.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Credit _$CreditFromJson(Map<String, dynamic> json) => _Credit(
  name: json['name'] as String,
  licenseCode: json['license_code'] as String,
  attributionText: json['attribution_text'] as String,
  shareAlike: json['share_alike'] as bool,
  url: json['url'] as String?,
  licenseUrl: json['license_url'] as String?,
);

Map<String, dynamic> _$CreditToJson(_Credit instance) => <String, dynamic>{
  'name': instance.name,
  'license_code': instance.licenseCode,
  'attribution_text': instance.attributionText,
  'share_alike': instance.shareAlike,
  'url': instance.url,
  'license_url': instance.licenseUrl,
};
