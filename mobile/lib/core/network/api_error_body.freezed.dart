// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'api_error_body.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$ApiErrorBody {

 ApiErrorDetail get error;
/// Create a copy of ApiErrorBody
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ApiErrorBodyCopyWith<ApiErrorBody> get copyWith => _$ApiErrorBodyCopyWithImpl<ApiErrorBody>(this as ApiErrorBody, _$identity);

  /// Serializes this ApiErrorBody to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as ApiErrorBody;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ApiErrorBody&&(identical(other.error, _this.error) || other.error == _this.error));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as ApiErrorBody;
  return Object.hash(runtimeType,_this.error);
}

@override
String toString() {
  final _this = this as ApiErrorBody;
  return 'ApiErrorBody(error: ${_this.error})';
}


}

/// @nodoc
abstract mixin class $ApiErrorBodyCopyWith<$Res>  {
  factory $ApiErrorBodyCopyWith(ApiErrorBody value, $Res Function(ApiErrorBody) _then) = _$ApiErrorBodyCopyWithImpl;
@useResult
$Res call({
 ApiErrorDetail error
});


$ApiErrorDetailCopyWith<$Res> get error;

}
/// @nodoc
class _$ApiErrorBodyCopyWithImpl<$Res>
    implements $ApiErrorBodyCopyWith<$Res> {
  _$ApiErrorBodyCopyWithImpl(this._self, this._then);

  final ApiErrorBody _self;
  final $Res Function(ApiErrorBody) _then;

/// Create a copy of ApiErrorBody
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? error = null,}) {
  return _then(ApiErrorBody(
error: null == error ? _self.error : error // ignore: cast_nullable_to_non_nullable
as ApiErrorDetail,
  ));
}
/// Create a copy of ApiErrorBody
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ApiErrorDetailCopyWith<$Res> get error {
  
  return $ApiErrorDetailCopyWith<$Res>(_self.error, (value) {
    return _then(_self.copyWith(error: value));
  });
}
}


/// Adds pattern-matching-related methods to [ApiErrorBody].
extension ApiErrorBodyPatterns on ApiErrorBody {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ApiErrorBody value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ApiErrorBody() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ApiErrorBody value)  $default,){
final _that = this;
switch (_that) {
case _ApiErrorBody():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ApiErrorBody value)?  $default,){
final _that = this;
switch (_that) {
case _ApiErrorBody() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( ApiErrorDetail error)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ApiErrorBody() when $default != null:
return $default(_that.error);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( ApiErrorDetail error)  $default,) {final _that = this;
switch (_that) {
case _ApiErrorBody():
return $default(_that.error);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( ApiErrorDetail error)?  $default,) {final _that = this;
switch (_that) {
case _ApiErrorBody() when $default != null:
return $default(_that.error);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ApiErrorBody implements ApiErrorBody {
  const _ApiErrorBody({required this.error});
  factory _ApiErrorBody.fromJson(Map<String, dynamic> json) => _$ApiErrorBodyFromJson(json);

@override final  ApiErrorDetail error;

/// Create a copy of ApiErrorBody
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ApiErrorBodyCopyWith<_ApiErrorBody> get copyWith => __$ApiErrorBodyCopyWithImpl<_ApiErrorBody>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ApiErrorBodyToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _ApiErrorBody&&(identical(other.error, error) || other.error == error));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,error);
}

@override
String toString() {
    return 'ApiErrorBody(error: $error)';
}


}

/// @nodoc
abstract mixin class _$ApiErrorBodyCopyWith<$Res> implements $ApiErrorBodyCopyWith<$Res> {
  factory _$ApiErrorBodyCopyWith(_ApiErrorBody value, $Res Function(_ApiErrorBody) _then) = __$ApiErrorBodyCopyWithImpl;
@override @useResult
$Res call({
 ApiErrorDetail error
});


@override $ApiErrorDetailCopyWith<$Res> get error;

}
/// @nodoc
class __$ApiErrorBodyCopyWithImpl<$Res>
    implements _$ApiErrorBodyCopyWith<$Res> {
  __$ApiErrorBodyCopyWithImpl(this._self, this._then);

  final _ApiErrorBody _self;
  final $Res Function(_ApiErrorBody) _then;

/// Create a copy of ApiErrorBody
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? error = null,}) {
  return _then(_ApiErrorBody(
error: null == error ? _self.error : error // ignore: cast_nullable_to_non_nullable
as ApiErrorDetail,
  ));
}

/// Create a copy of ApiErrorBody
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ApiErrorDetailCopyWith<$Res> get error {
  
  return $ApiErrorDetailCopyWith<$Res>(_self.error, (value) {
    return _then(_self.copyWith(error: value));
  });
}
}


/// @nodoc
mixin _$ApiErrorDetail {

 String get code; String get message;@JsonKey(name: 'request_id') String get requestId; Map<String, dynamic>? get details;
/// Create a copy of ApiErrorDetail
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ApiErrorDetailCopyWith<ApiErrorDetail> get copyWith => _$ApiErrorDetailCopyWithImpl<ApiErrorDetail>(this as ApiErrorDetail, _$identity);

  /// Serializes this ApiErrorDetail to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as ApiErrorDetail;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ApiErrorDetail&&(identical(other.code, _this.code) || other.code == _this.code)&&(identical(other.message, _this.message) || other.message == _this.message)&&(identical(other.requestId, _this.requestId) || other.requestId == _this.requestId)&&const DeepCollectionEquality().equals(other.details, _this.details));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as ApiErrorDetail;
  return Object.hash(runtimeType,_this.code,_this.message,_this.requestId,const DeepCollectionEquality().hash(_this.details));
}

@override
String toString() {
  final _this = this as ApiErrorDetail;
  return 'ApiErrorDetail(code: ${_this.code}, message: ${_this.message}, requestId: ${_this.requestId}, details: ${_this.details})';
}


}

/// @nodoc
abstract mixin class $ApiErrorDetailCopyWith<$Res>  {
  factory $ApiErrorDetailCopyWith(ApiErrorDetail value, $Res Function(ApiErrorDetail) _then) = _$ApiErrorDetailCopyWithImpl;
@useResult
$Res call({
 String code, String message,@JsonKey(name: 'request_id') String requestId, Map<String, dynamic>? details
});




}
/// @nodoc
class _$ApiErrorDetailCopyWithImpl<$Res>
    implements $ApiErrorDetailCopyWith<$Res> {
  _$ApiErrorDetailCopyWithImpl(this._self, this._then);

  final ApiErrorDetail _self;
  final $Res Function(ApiErrorDetail) _then;

/// Create a copy of ApiErrorDetail
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? code = null,Object? message = null,Object? requestId = null,Object? details = freezed,}) {
  return _then(ApiErrorDetail(
code: null == code ? _self.code : code // ignore: cast_nullable_to_non_nullable
as String,message: null == message ? _self.message : message // ignore: cast_nullable_to_non_nullable
as String,requestId: null == requestId ? _self.requestId : requestId // ignore: cast_nullable_to_non_nullable
as String,details: freezed == details ? _self.details : details // ignore: cast_nullable_to_non_nullable
as Map<String, dynamic>?,
  ));
}

}


/// Adds pattern-matching-related methods to [ApiErrorDetail].
extension ApiErrorDetailPatterns on ApiErrorDetail {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ApiErrorDetail value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ApiErrorDetail() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ApiErrorDetail value)  $default,){
final _that = this;
switch (_that) {
case _ApiErrorDetail():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ApiErrorDetail value)?  $default,){
final _that = this;
switch (_that) {
case _ApiErrorDetail() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String code,  String message, @JsonKey(name: 'request_id')  String requestId,  Map<String, dynamic>? details)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ApiErrorDetail() when $default != null:
return $default(_that.code,_that.message,_that.requestId,_that.details);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String code,  String message, @JsonKey(name: 'request_id')  String requestId,  Map<String, dynamic>? details)  $default,) {final _that = this;
switch (_that) {
case _ApiErrorDetail():
return $default(_that.code,_that.message,_that.requestId,_that.details);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String code,  String message, @JsonKey(name: 'request_id')  String requestId,  Map<String, dynamic>? details)?  $default,) {final _that = this;
switch (_that) {
case _ApiErrorDetail() when $default != null:
return $default(_that.code,_that.message,_that.requestId,_that.details);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ApiErrorDetail implements ApiErrorDetail {
  const _ApiErrorDetail({required this.code, required this.message, @JsonKey(name: 'request_id') required this.requestId,  Map<String, dynamic>? details}): _details = details;
  factory _ApiErrorDetail.fromJson(Map<String, dynamic> json) => _$ApiErrorDetailFromJson(json);

@override final  String code;
@override final  String message;
@override@JsonKey(name: 'request_id') final  String requestId;
 final  Map<String, dynamic>? _details;
@override Map<String, dynamic>? get details {
  final value = _details;
  if (value == null) return null;
  if (_details is EqualUnmodifiableMapView) return _details;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableMapView(value);
}


/// Create a copy of ApiErrorDetail
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ApiErrorDetailCopyWith<_ApiErrorDetail> get copyWith => __$ApiErrorDetailCopyWithImpl<_ApiErrorDetail>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ApiErrorDetailToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _ApiErrorDetail&&(identical(other.code, code) || other.code == code)&&(identical(other.message, message) || other.message == message)&&(identical(other.requestId, requestId) || other.requestId == requestId)&&const DeepCollectionEquality().equals(other.details, _details));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,code,message,requestId,const DeepCollectionEquality().hash(_details));
}

@override
String toString() {
    return 'ApiErrorDetail(code: $code, message: $message, requestId: $requestId, details: $details)';
}


}

/// @nodoc
abstract mixin class _$ApiErrorDetailCopyWith<$Res> implements $ApiErrorDetailCopyWith<$Res> {
  factory _$ApiErrorDetailCopyWith(_ApiErrorDetail value, $Res Function(_ApiErrorDetail) _then) = __$ApiErrorDetailCopyWithImpl;
@override @useResult
$Res call({
 String code, String message,@JsonKey(name: 'request_id') String requestId, Map<String, dynamic>? details
});




}
/// @nodoc
class __$ApiErrorDetailCopyWithImpl<$Res>
    implements _$ApiErrorDetailCopyWith<$Res> {
  __$ApiErrorDetailCopyWithImpl(this._self, this._then);

  final _ApiErrorDetail _self;
  final $Res Function(_ApiErrorDetail) _then;

/// Create a copy of ApiErrorDetail
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? code = null,Object? message = null,Object? requestId = null,Object? details = freezed,}) {
  return _then(_ApiErrorDetail(
code: null == code ? _self.code : code // ignore: cast_nullable_to_non_nullable
as String,message: null == message ? _self.message : message // ignore: cast_nullable_to_non_nullable
as String,requestId: null == requestId ? _self.requestId : requestId // ignore: cast_nullable_to_non_nullable
as String,details: freezed == details ? _self._details : details // ignore: cast_nullable_to_non_nullable
as Map<String, dynamic>?,
  ));
}


}

// dart format on
