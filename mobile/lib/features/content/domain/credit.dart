import 'package:freezed_annotation/freezed_annotation.dart';

part 'credit.freezed.dart';
part 'credit.g.dart';

/// A question source and its license (`GET /v1/credits`, D-006).
@freezed
abstract class Credit with _$Credit {
  const factory Credit({
    required String name,
    required String licenseCode,
    required String attributionText,
    required bool shareAlike,
    String? url,
    String? licenseUrl,
  }) = _Credit;

  factory Credit.fromJson(Map<String, dynamic> json) => _$CreditFromJson(json);
}
