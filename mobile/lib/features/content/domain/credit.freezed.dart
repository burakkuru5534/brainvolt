// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'credit.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Credit {

 String get name; String get licenseCode; String get attributionText; bool get shareAlike; String? get url; String? get licenseUrl;
/// Create a copy of Credit
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$CreditCopyWith<Credit> get copyWith => _$CreditCopyWithImpl<Credit>(this as Credit, _$identity);

  /// Serializes this Credit to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as Credit;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Credit&&(identical(other.name, _this.name) || other.name == _this.name)&&(identical(other.licenseCode, _this.licenseCode) || other.licenseCode == _this.licenseCode)&&(identical(other.attributionText, _this.attributionText) || other.attributionText == _this.attributionText)&&(identical(other.shareAlike, _this.shareAlike) || other.shareAlike == _this.shareAlike)&&(identical(other.url, _this.url) || other.url == _this.url)&&(identical(other.licenseUrl, _this.licenseUrl) || other.licenseUrl == _this.licenseUrl));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as Credit;
  return Object.hash(runtimeType,_this.name,_this.licenseCode,_this.attributionText,_this.shareAlike,_this.url,_this.licenseUrl);
}

@override
String toString() {
  final _this = this as Credit;
  return 'Credit(name: ${_this.name}, licenseCode: ${_this.licenseCode}, attributionText: ${_this.attributionText}, shareAlike: ${_this.shareAlike}, url: ${_this.url}, licenseUrl: ${_this.licenseUrl})';
}


}

/// @nodoc
abstract mixin class $CreditCopyWith<$Res>  {
  factory $CreditCopyWith(Credit value, $Res Function(Credit) _then) = _$CreditCopyWithImpl;
@useResult
$Res call({
 String name, String licenseCode, String attributionText, bool shareAlike, String? url, String? licenseUrl
});




}
/// @nodoc
class _$CreditCopyWithImpl<$Res>
    implements $CreditCopyWith<$Res> {
  _$CreditCopyWithImpl(this._self, this._then);

  final Credit _self;
  final $Res Function(Credit) _then;

/// Create a copy of Credit
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? name = null,Object? licenseCode = null,Object? attributionText = null,Object? shareAlike = null,Object? url = freezed,Object? licenseUrl = freezed,}) {
  return _then(Credit(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,licenseCode: null == licenseCode ? _self.licenseCode : licenseCode // ignore: cast_nullable_to_non_nullable
as String,attributionText: null == attributionText ? _self.attributionText : attributionText // ignore: cast_nullable_to_non_nullable
as String,shareAlike: null == shareAlike ? _self.shareAlike : shareAlike // ignore: cast_nullable_to_non_nullable
as bool,url: freezed == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String?,licenseUrl: freezed == licenseUrl ? _self.licenseUrl : licenseUrl // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [Credit].
extension CreditPatterns on Credit {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Credit value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Credit() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Credit value)  $default,){
final _that = this;
switch (_that) {
case _Credit():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Credit value)?  $default,){
final _that = this;
switch (_that) {
case _Credit() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String name,  String licenseCode,  String attributionText,  bool shareAlike,  String? url,  String? licenseUrl)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Credit() when $default != null:
return $default(_that.name,_that.licenseCode,_that.attributionText,_that.shareAlike,_that.url,_that.licenseUrl);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String name,  String licenseCode,  String attributionText,  bool shareAlike,  String? url,  String? licenseUrl)  $default,) {final _that = this;
switch (_that) {
case _Credit():
return $default(_that.name,_that.licenseCode,_that.attributionText,_that.shareAlike,_that.url,_that.licenseUrl);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String name,  String licenseCode,  String attributionText,  bool shareAlike,  String? url,  String? licenseUrl)?  $default,) {final _that = this;
switch (_that) {
case _Credit() when $default != null:
return $default(_that.name,_that.licenseCode,_that.attributionText,_that.shareAlike,_that.url,_that.licenseUrl);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Credit implements Credit {
  const _Credit({required this.name, required this.licenseCode, required this.attributionText, required this.shareAlike, this.url, this.licenseUrl});
  factory _Credit.fromJson(Map<String, dynamic> json) => _$CreditFromJson(json);

@override final  String name;
@override final  String licenseCode;
@override final  String attributionText;
@override final  bool shareAlike;
@override final  String? url;
@override final  String? licenseUrl;

/// Create a copy of Credit
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$CreditCopyWith<_Credit> get copyWith => __$CreditCopyWithImpl<_Credit>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$CreditToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _Credit&&(identical(other.name, name) || other.name == name)&&(identical(other.licenseCode, licenseCode) || other.licenseCode == licenseCode)&&(identical(other.attributionText, attributionText) || other.attributionText == attributionText)&&(identical(other.shareAlike, shareAlike) || other.shareAlike == shareAlike)&&(identical(other.url, url) || other.url == url)&&(identical(other.licenseUrl, licenseUrl) || other.licenseUrl == licenseUrl));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,name,licenseCode,attributionText,shareAlike,url,licenseUrl);
}

@override
String toString() {
    return 'Credit(name: $name, licenseCode: $licenseCode, attributionText: $attributionText, shareAlike: $shareAlike, url: $url, licenseUrl: $licenseUrl)';
}


}

/// @nodoc
abstract mixin class _$CreditCopyWith<$Res> implements $CreditCopyWith<$Res> {
  factory _$CreditCopyWith(_Credit value, $Res Function(_Credit) _then) = __$CreditCopyWithImpl;
@override @useResult
$Res call({
 String name, String licenseCode, String attributionText, bool shareAlike, String? url, String? licenseUrl
});




}
/// @nodoc
class __$CreditCopyWithImpl<$Res>
    implements _$CreditCopyWith<$Res> {
  __$CreditCopyWithImpl(this._self, this._then);

  final _Credit _self;
  final $Res Function(_Credit) _then;

/// Create a copy of Credit
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? name = null,Object? licenseCode = null,Object? attributionText = null,Object? shareAlike = null,Object? url = freezed,Object? licenseUrl = freezed,}) {
  return _then(_Credit(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,licenseCode: null == licenseCode ? _self.licenseCode : licenseCode // ignore: cast_nullable_to_non_nullable
as String,attributionText: null == attributionText ? _self.attributionText : attributionText // ignore: cast_nullable_to_non_nullable
as String,shareAlike: null == shareAlike ? _self.shareAlike : shareAlike // ignore: cast_nullable_to_non_nullable
as bool,url: freezed == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String?,licenseUrl: freezed == licenseUrl ? _self.licenseUrl : licenseUrl // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
