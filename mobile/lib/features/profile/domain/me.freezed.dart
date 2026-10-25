// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'me.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Me {

 MeUser get user; OwnProfile get profile; UserSettings get settings;
/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MeCopyWith<Me> get copyWith => _$MeCopyWithImpl<Me>(this as Me, _$identity);

  /// Serializes this Me to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as Me;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Me&&(identical(other.user, _this.user) || other.user == _this.user)&&(identical(other.profile, _this.profile) || other.profile == _this.profile)&&(identical(other.settings, _this.settings) || other.settings == _this.settings));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as Me;
  return Object.hash(runtimeType,_this.user,_this.profile,_this.settings);
}

@override
String toString() {
  final _this = this as Me;
  return 'Me(user: ${_this.user}, profile: ${_this.profile}, settings: ${_this.settings})';
}


}

/// @nodoc
abstract mixin class $MeCopyWith<$Res>  {
  factory $MeCopyWith(Me value, $Res Function(Me) _then) = _$MeCopyWithImpl;
@useResult
$Res call({
 MeUser user, OwnProfile profile, UserSettings settings
});


$MeUserCopyWith<$Res> get user;$OwnProfileCopyWith<$Res> get profile;$UserSettingsCopyWith<$Res> get settings;

}
/// @nodoc
class _$MeCopyWithImpl<$Res>
    implements $MeCopyWith<$Res> {
  _$MeCopyWithImpl(this._self, this._then);

  final Me _self;
  final $Res Function(Me) _then;

/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? user = null,Object? profile = null,Object? settings = null,}) {
  return _then(Me(
user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as MeUser,profile: null == profile ? _self.profile : profile // ignore: cast_nullable_to_non_nullable
as OwnProfile,settings: null == settings ? _self.settings : settings // ignore: cast_nullable_to_non_nullable
as UserSettings,
  ));
}
/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$MeUserCopyWith<$Res> get user {
  
  return $MeUserCopyWith<$Res>(_self.user, (value) {
    return _then(_self.copyWith(user: value));
  });
}/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$OwnProfileCopyWith<$Res> get profile {
  
  return $OwnProfileCopyWith<$Res>(_self.profile, (value) {
    return _then(_self.copyWith(profile: value));
  });
}/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$UserSettingsCopyWith<$Res> get settings {
  
  return $UserSettingsCopyWith<$Res>(_self.settings, (value) {
    return _then(_self.copyWith(settings: value));
  });
}
}


/// Adds pattern-matching-related methods to [Me].
extension MePatterns on Me {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Me value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Me() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Me value)  $default,){
final _that = this;
switch (_that) {
case _Me():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Me value)?  $default,){
final _that = this;
switch (_that) {
case _Me() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( MeUser user,  OwnProfile profile,  UserSettings settings)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Me() when $default != null:
return $default(_that.user,_that.profile,_that.settings);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( MeUser user,  OwnProfile profile,  UserSettings settings)  $default,) {final _that = this;
switch (_that) {
case _Me():
return $default(_that.user,_that.profile,_that.settings);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( MeUser user,  OwnProfile profile,  UserSettings settings)?  $default,) {final _that = this;
switch (_that) {
case _Me() when $default != null:
return $default(_that.user,_that.profile,_that.settings);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Me implements Me {
  const _Me({required this.user, required this.profile, required this.settings});
  factory _Me.fromJson(Map<String, dynamic> json) => _$MeFromJson(json);

@override final  MeUser user;
@override final  OwnProfile profile;
@override final  UserSettings settings;

/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MeCopyWith<_Me> get copyWith => __$MeCopyWithImpl<_Me>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MeToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _Me&&(identical(other.user, user) || other.user == user)&&(identical(other.profile, profile) || other.profile == profile)&&(identical(other.settings, settings) || other.settings == settings));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,user,profile,settings);
}

@override
String toString() {
    return 'Me(user: $user, profile: $profile, settings: $settings)';
}


}

/// @nodoc
abstract mixin class _$MeCopyWith<$Res> implements $MeCopyWith<$Res> {
  factory _$MeCopyWith(_Me value, $Res Function(_Me) _then) = __$MeCopyWithImpl;
@override @useResult
$Res call({
 MeUser user, OwnProfile profile, UserSettings settings
});


@override $MeUserCopyWith<$Res> get user;@override $OwnProfileCopyWith<$Res> get profile;@override $UserSettingsCopyWith<$Res> get settings;

}
/// @nodoc
class __$MeCopyWithImpl<$Res>
    implements _$MeCopyWith<$Res> {
  __$MeCopyWithImpl(this._self, this._then);

  final _Me _self;
  final $Res Function(_Me) _then;

/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? user = null,Object? profile = null,Object? settings = null,}) {
  return _then(_Me(
user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as MeUser,profile: null == profile ? _self.profile : profile // ignore: cast_nullable_to_non_nullable
as OwnProfile,settings: null == settings ? _self.settings : settings // ignore: cast_nullable_to_non_nullable
as UserSettings,
  ));
}

/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$MeUserCopyWith<$Res> get user {
  
  return $MeUserCopyWith<$Res>(_self.user, (value) {
    return _then(_self.copyWith(user: value));
  });
}/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$OwnProfileCopyWith<$Res> get profile {
  
  return $OwnProfileCopyWith<$Res>(_self.profile, (value) {
    return _then(_self.copyWith(profile: value));
  });
}/// Create a copy of Me
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$UserSettingsCopyWith<$Res> get settings {
  
  return $UserSettingsCopyWith<$Res>(_self.settings, (value) {
    return _then(_self.copyWith(settings: value));
  });
}
}


/// @nodoc
mixin _$MeUser {

 String get id; Tier get tier;
/// Create a copy of MeUser
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MeUserCopyWith<MeUser> get copyWith => _$MeUserCopyWithImpl<MeUser>(this as MeUser, _$identity);

  /// Serializes this MeUser to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as MeUser;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is MeUser&&(identical(other.id, _this.id) || other.id == _this.id)&&(identical(other.tier, _this.tier) || other.tier == _this.tier));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as MeUser;
  return Object.hash(runtimeType,_this.id,_this.tier);
}

@override
String toString() {
  final _this = this as MeUser;
  return 'MeUser(id: ${_this.id}, tier: ${_this.tier})';
}


}

/// @nodoc
abstract mixin class $MeUserCopyWith<$Res>  {
  factory $MeUserCopyWith(MeUser value, $Res Function(MeUser) _then) = _$MeUserCopyWithImpl;
@useResult
$Res call({
 String id, Tier tier
});




}
/// @nodoc
class _$MeUserCopyWithImpl<$Res>
    implements $MeUserCopyWith<$Res> {
  _$MeUserCopyWithImpl(this._self, this._then);

  final MeUser _self;
  final $Res Function(MeUser) _then;

/// Create a copy of MeUser
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? tier = null,}) {
  return _then(MeUser(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,tier: null == tier ? _self.tier : tier // ignore: cast_nullable_to_non_nullable
as Tier,
  ));
}

}


/// Adds pattern-matching-related methods to [MeUser].
extension MeUserPatterns on MeUser {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _MeUser value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _MeUser() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _MeUser value)  $default,){
final _that = this;
switch (_that) {
case _MeUser():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _MeUser value)?  $default,){
final _that = this;
switch (_that) {
case _MeUser() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  Tier tier)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _MeUser() when $default != null:
return $default(_that.id,_that.tier);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  Tier tier)  $default,) {final _that = this;
switch (_that) {
case _MeUser():
return $default(_that.id,_that.tier);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  Tier tier)?  $default,) {final _that = this;
switch (_that) {
case _MeUser() when $default != null:
return $default(_that.id,_that.tier);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _MeUser implements MeUser {
  const _MeUser({required this.id, required this.tier});
  factory _MeUser.fromJson(Map<String, dynamic> json) => _$MeUserFromJson(json);

@override final  String id;
@override final  Tier tier;

/// Create a copy of MeUser
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MeUserCopyWith<_MeUser> get copyWith => __$MeUserCopyWithImpl<_MeUser>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MeUserToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _MeUser&&(identical(other.id, id) || other.id == id)&&(identical(other.tier, tier) || other.tier == tier));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,id,tier);
}

@override
String toString() {
    return 'MeUser(id: $id, tier: $tier)';
}


}

/// @nodoc
abstract mixin class _$MeUserCopyWith<$Res> implements $MeUserCopyWith<$Res> {
  factory _$MeUserCopyWith(_MeUser value, $Res Function(_MeUser) _then) = __$MeUserCopyWithImpl;
@override @useResult
$Res call({
 String id, Tier tier
});




}
/// @nodoc
class __$MeUserCopyWithImpl<$Res>
    implements _$MeUserCopyWith<$Res> {
  __$MeUserCopyWithImpl(this._self, this._then);

  final _MeUser _self;
  final $Res Function(_MeUser) _then;

/// Create a copy of MeUser
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? tier = null,}) {
  return _then(_MeUser(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,tier: null == tier ? _self.tier : tier // ignore: cast_nullable_to_non_nullable
as Tier,
  ));
}


}


/// @nodoc
mixin _$OwnProfile {

 String get username; String get displayName; String get avatarId; String get locale; String? get countryCode;/// null: the username may be changed now (full accounts only).
 DateTime? get usernameChangeAvailableAt; DateTime? get countryChangeAvailableAt;
/// Create a copy of OwnProfile
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$OwnProfileCopyWith<OwnProfile> get copyWith => _$OwnProfileCopyWithImpl<OwnProfile>(this as OwnProfile, _$identity);

  /// Serializes this OwnProfile to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as OwnProfile;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is OwnProfile&&(identical(other.username, _this.username) || other.username == _this.username)&&(identical(other.displayName, _this.displayName) || other.displayName == _this.displayName)&&(identical(other.avatarId, _this.avatarId) || other.avatarId == _this.avatarId)&&(identical(other.locale, _this.locale) || other.locale == _this.locale)&&(identical(other.countryCode, _this.countryCode) || other.countryCode == _this.countryCode)&&(identical(other.usernameChangeAvailableAt, _this.usernameChangeAvailableAt) || other.usernameChangeAvailableAt == _this.usernameChangeAvailableAt)&&(identical(other.countryChangeAvailableAt, _this.countryChangeAvailableAt) || other.countryChangeAvailableAt == _this.countryChangeAvailableAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as OwnProfile;
  return Object.hash(runtimeType,_this.username,_this.displayName,_this.avatarId,_this.locale,_this.countryCode,_this.usernameChangeAvailableAt,_this.countryChangeAvailableAt);
}

@override
String toString() {
  final _this = this as OwnProfile;
  return 'OwnProfile(username: ${_this.username}, displayName: ${_this.displayName}, avatarId: ${_this.avatarId}, locale: ${_this.locale}, countryCode: ${_this.countryCode}, usernameChangeAvailableAt: ${_this.usernameChangeAvailableAt}, countryChangeAvailableAt: ${_this.countryChangeAvailableAt})';
}


}

/// @nodoc
abstract mixin class $OwnProfileCopyWith<$Res>  {
  factory $OwnProfileCopyWith(OwnProfile value, $Res Function(OwnProfile) _then) = _$OwnProfileCopyWithImpl;
@useResult
$Res call({
 String username, String displayName, String avatarId, String locale, String? countryCode, DateTime? usernameChangeAvailableAt, DateTime? countryChangeAvailableAt
});




}
/// @nodoc
class _$OwnProfileCopyWithImpl<$Res>
    implements $OwnProfileCopyWith<$Res> {
  _$OwnProfileCopyWithImpl(this._self, this._then);

  final OwnProfile _self;
  final $Res Function(OwnProfile) _then;

/// Create a copy of OwnProfile
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? username = null,Object? displayName = null,Object? avatarId = null,Object? locale = null,Object? countryCode = freezed,Object? usernameChangeAvailableAt = freezed,Object? countryChangeAvailableAt = freezed,}) {
  return _then(OwnProfile(
username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,displayName: null == displayName ? _self.displayName : displayName // ignore: cast_nullable_to_non_nullable
as String,avatarId: null == avatarId ? _self.avatarId : avatarId // ignore: cast_nullable_to_non_nullable
as String,locale: null == locale ? _self.locale : locale // ignore: cast_nullable_to_non_nullable
as String,countryCode: freezed == countryCode ? _self.countryCode : countryCode // ignore: cast_nullable_to_non_nullable
as String?,usernameChangeAvailableAt: freezed == usernameChangeAvailableAt ? _self.usernameChangeAvailableAt : usernameChangeAvailableAt // ignore: cast_nullable_to_non_nullable
as DateTime?,countryChangeAvailableAt: freezed == countryChangeAvailableAt ? _self.countryChangeAvailableAt : countryChangeAvailableAt // ignore: cast_nullable_to_non_nullable
as DateTime?,
  ));
}

}


/// Adds pattern-matching-related methods to [OwnProfile].
extension OwnProfilePatterns on OwnProfile {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _OwnProfile value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _OwnProfile() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _OwnProfile value)  $default,){
final _that = this;
switch (_that) {
case _OwnProfile():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _OwnProfile value)?  $default,){
final _that = this;
switch (_that) {
case _OwnProfile() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String username,  String displayName,  String avatarId,  String locale,  String? countryCode,  DateTime? usernameChangeAvailableAt,  DateTime? countryChangeAvailableAt)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _OwnProfile() when $default != null:
return $default(_that.username,_that.displayName,_that.avatarId,_that.locale,_that.countryCode,_that.usernameChangeAvailableAt,_that.countryChangeAvailableAt);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String username,  String displayName,  String avatarId,  String locale,  String? countryCode,  DateTime? usernameChangeAvailableAt,  DateTime? countryChangeAvailableAt)  $default,) {final _that = this;
switch (_that) {
case _OwnProfile():
return $default(_that.username,_that.displayName,_that.avatarId,_that.locale,_that.countryCode,_that.usernameChangeAvailableAt,_that.countryChangeAvailableAt);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String username,  String displayName,  String avatarId,  String locale,  String? countryCode,  DateTime? usernameChangeAvailableAt,  DateTime? countryChangeAvailableAt)?  $default,) {final _that = this;
switch (_that) {
case _OwnProfile() when $default != null:
return $default(_that.username,_that.displayName,_that.avatarId,_that.locale,_that.countryCode,_that.usernameChangeAvailableAt,_that.countryChangeAvailableAt);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _OwnProfile implements OwnProfile {
  const _OwnProfile({required this.username, required this.displayName, required this.avatarId, required this.locale, this.countryCode, this.usernameChangeAvailableAt, this.countryChangeAvailableAt});
  factory _OwnProfile.fromJson(Map<String, dynamic> json) => _$OwnProfileFromJson(json);

@override final  String username;
@override final  String displayName;
@override final  String avatarId;
@override final  String locale;
@override final  String? countryCode;
/// null: the username may be changed now (full accounts only).
@override final  DateTime? usernameChangeAvailableAt;
@override final  DateTime? countryChangeAvailableAt;

/// Create a copy of OwnProfile
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$OwnProfileCopyWith<_OwnProfile> get copyWith => __$OwnProfileCopyWithImpl<_OwnProfile>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$OwnProfileToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _OwnProfile&&(identical(other.username, username) || other.username == username)&&(identical(other.displayName, displayName) || other.displayName == displayName)&&(identical(other.avatarId, avatarId) || other.avatarId == avatarId)&&(identical(other.locale, locale) || other.locale == locale)&&(identical(other.countryCode, countryCode) || other.countryCode == countryCode)&&(identical(other.usernameChangeAvailableAt, usernameChangeAvailableAt) || other.usernameChangeAvailableAt == usernameChangeAvailableAt)&&(identical(other.countryChangeAvailableAt, countryChangeAvailableAt) || other.countryChangeAvailableAt == countryChangeAvailableAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,username,displayName,avatarId,locale,countryCode,usernameChangeAvailableAt,countryChangeAvailableAt);
}

@override
String toString() {
    return 'OwnProfile(username: $username, displayName: $displayName, avatarId: $avatarId, locale: $locale, countryCode: $countryCode, usernameChangeAvailableAt: $usernameChangeAvailableAt, countryChangeAvailableAt: $countryChangeAvailableAt)';
}


}

/// @nodoc
abstract mixin class _$OwnProfileCopyWith<$Res> implements $OwnProfileCopyWith<$Res> {
  factory _$OwnProfileCopyWith(_OwnProfile value, $Res Function(_OwnProfile) _then) = __$OwnProfileCopyWithImpl;
@override @useResult
$Res call({
 String username, String displayName, String avatarId, String locale, String? countryCode, DateTime? usernameChangeAvailableAt, DateTime? countryChangeAvailableAt
});




}
/// @nodoc
class __$OwnProfileCopyWithImpl<$Res>
    implements _$OwnProfileCopyWith<$Res> {
  __$OwnProfileCopyWithImpl(this._self, this._then);

  final _OwnProfile _self;
  final $Res Function(_OwnProfile) _then;

/// Create a copy of OwnProfile
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? username = null,Object? displayName = null,Object? avatarId = null,Object? locale = null,Object? countryCode = freezed,Object? usernameChangeAvailableAt = freezed,Object? countryChangeAvailableAt = freezed,}) {
  return _then(_OwnProfile(
username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,displayName: null == displayName ? _self.displayName : displayName // ignore: cast_nullable_to_non_nullable
as String,avatarId: null == avatarId ? _self.avatarId : avatarId // ignore: cast_nullable_to_non_nullable
as String,locale: null == locale ? _self.locale : locale // ignore: cast_nullable_to_non_nullable
as String,countryCode: freezed == countryCode ? _self.countryCode : countryCode // ignore: cast_nullable_to_non_nullable
as String?,usernameChangeAvailableAt: freezed == usernameChangeAvailableAt ? _self.usernameChangeAvailableAt : usernameChangeAvailableAt // ignore: cast_nullable_to_non_nullable
as DateTime?,countryChangeAvailableAt: freezed == countryChangeAvailableAt ? _self.countryChangeAvailableAt : countryChangeAvailableAt // ignore: cast_nullable_to_non_nullable
as DateTime?,
  ));
}


}


/// @nodoc
mixin _$UserSettings {

 ChallengesFrom get challengesFrom; bool get showOnlineStatus; bool get allowFriendRequests; bool get soundEnabled; bool get musicEnabled; bool get hapticsEnabled; bool get profilePrivate;
/// Create a copy of UserSettings
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$UserSettingsCopyWith<UserSettings> get copyWith => _$UserSettingsCopyWithImpl<UserSettings>(this as UserSettings, _$identity);

  /// Serializes this UserSettings to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as UserSettings;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is UserSettings&&(identical(other.challengesFrom, _this.challengesFrom) || other.challengesFrom == _this.challengesFrom)&&(identical(other.showOnlineStatus, _this.showOnlineStatus) || other.showOnlineStatus == _this.showOnlineStatus)&&(identical(other.allowFriendRequests, _this.allowFriendRequests) || other.allowFriendRequests == _this.allowFriendRequests)&&(identical(other.soundEnabled, _this.soundEnabled) || other.soundEnabled == _this.soundEnabled)&&(identical(other.musicEnabled, _this.musicEnabled) || other.musicEnabled == _this.musicEnabled)&&(identical(other.hapticsEnabled, _this.hapticsEnabled) || other.hapticsEnabled == _this.hapticsEnabled)&&(identical(other.profilePrivate, _this.profilePrivate) || other.profilePrivate == _this.profilePrivate));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as UserSettings;
  return Object.hash(runtimeType,_this.challengesFrom,_this.showOnlineStatus,_this.allowFriendRequests,_this.soundEnabled,_this.musicEnabled,_this.hapticsEnabled,_this.profilePrivate);
}

@override
String toString() {
  final _this = this as UserSettings;
  return 'UserSettings(challengesFrom: ${_this.challengesFrom}, showOnlineStatus: ${_this.showOnlineStatus}, allowFriendRequests: ${_this.allowFriendRequests}, soundEnabled: ${_this.soundEnabled}, musicEnabled: ${_this.musicEnabled}, hapticsEnabled: ${_this.hapticsEnabled}, profilePrivate: ${_this.profilePrivate})';
}


}

/// @nodoc
abstract mixin class $UserSettingsCopyWith<$Res>  {
  factory $UserSettingsCopyWith(UserSettings value, $Res Function(UserSettings) _then) = _$UserSettingsCopyWithImpl;
@useResult
$Res call({
 ChallengesFrom challengesFrom, bool showOnlineStatus, bool allowFriendRequests, bool soundEnabled, bool musicEnabled, bool hapticsEnabled, bool profilePrivate
});




}
/// @nodoc
class _$UserSettingsCopyWithImpl<$Res>
    implements $UserSettingsCopyWith<$Res> {
  _$UserSettingsCopyWithImpl(this._self, this._then);

  final UserSettings _self;
  final $Res Function(UserSettings) _then;

/// Create a copy of UserSettings
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? challengesFrom = null,Object? showOnlineStatus = null,Object? allowFriendRequests = null,Object? soundEnabled = null,Object? musicEnabled = null,Object? hapticsEnabled = null,Object? profilePrivate = null,}) {
  return _then(UserSettings(
challengesFrom: null == challengesFrom ? _self.challengesFrom : challengesFrom // ignore: cast_nullable_to_non_nullable
as ChallengesFrom,showOnlineStatus: null == showOnlineStatus ? _self.showOnlineStatus : showOnlineStatus // ignore: cast_nullable_to_non_nullable
as bool,allowFriendRequests: null == allowFriendRequests ? _self.allowFriendRequests : allowFriendRequests // ignore: cast_nullable_to_non_nullable
as bool,soundEnabled: null == soundEnabled ? _self.soundEnabled : soundEnabled // ignore: cast_nullable_to_non_nullable
as bool,musicEnabled: null == musicEnabled ? _self.musicEnabled : musicEnabled // ignore: cast_nullable_to_non_nullable
as bool,hapticsEnabled: null == hapticsEnabled ? _self.hapticsEnabled : hapticsEnabled // ignore: cast_nullable_to_non_nullable
as bool,profilePrivate: null == profilePrivate ? _self.profilePrivate : profilePrivate // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [UserSettings].
extension UserSettingsPatterns on UserSettings {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _UserSettings value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _UserSettings() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _UserSettings value)  $default,){
final _that = this;
switch (_that) {
case _UserSettings():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _UserSettings value)?  $default,){
final _that = this;
switch (_that) {
case _UserSettings() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( ChallengesFrom challengesFrom,  bool showOnlineStatus,  bool allowFriendRequests,  bool soundEnabled,  bool musicEnabled,  bool hapticsEnabled,  bool profilePrivate)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _UserSettings() when $default != null:
return $default(_that.challengesFrom,_that.showOnlineStatus,_that.allowFriendRequests,_that.soundEnabled,_that.musicEnabled,_that.hapticsEnabled,_that.profilePrivate);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( ChallengesFrom challengesFrom,  bool showOnlineStatus,  bool allowFriendRequests,  bool soundEnabled,  bool musicEnabled,  bool hapticsEnabled,  bool profilePrivate)  $default,) {final _that = this;
switch (_that) {
case _UserSettings():
return $default(_that.challengesFrom,_that.showOnlineStatus,_that.allowFriendRequests,_that.soundEnabled,_that.musicEnabled,_that.hapticsEnabled,_that.profilePrivate);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( ChallengesFrom challengesFrom,  bool showOnlineStatus,  bool allowFriendRequests,  bool soundEnabled,  bool musicEnabled,  bool hapticsEnabled,  bool profilePrivate)?  $default,) {final _that = this;
switch (_that) {
case _UserSettings() when $default != null:
return $default(_that.challengesFrom,_that.showOnlineStatus,_that.allowFriendRequests,_that.soundEnabled,_that.musicEnabled,_that.hapticsEnabled,_that.profilePrivate);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _UserSettings implements UserSettings {
  const _UserSettings({required this.challengesFrom, required this.showOnlineStatus, required this.allowFriendRequests, required this.soundEnabled, required this.musicEnabled, required this.hapticsEnabled, this.profilePrivate = false});
  factory _UserSettings.fromJson(Map<String, dynamic> json) => _$UserSettingsFromJson(json);

@override final  ChallengesFrom challengesFrom;
@override final  bool showOnlineStatus;
@override final  bool allowFriendRequests;
@override final  bool soundEnabled;
@override final  bool musicEnabled;
@override final  bool hapticsEnabled;
@override@JsonKey() final  bool profilePrivate;

/// Create a copy of UserSettings
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$UserSettingsCopyWith<_UserSettings> get copyWith => __$UserSettingsCopyWithImpl<_UserSettings>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$UserSettingsToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _UserSettings&&(identical(other.challengesFrom, challengesFrom) || other.challengesFrom == challengesFrom)&&(identical(other.showOnlineStatus, showOnlineStatus) || other.showOnlineStatus == showOnlineStatus)&&(identical(other.allowFriendRequests, allowFriendRequests) || other.allowFriendRequests == allowFriendRequests)&&(identical(other.soundEnabled, soundEnabled) || other.soundEnabled == soundEnabled)&&(identical(other.musicEnabled, musicEnabled) || other.musicEnabled == musicEnabled)&&(identical(other.hapticsEnabled, hapticsEnabled) || other.hapticsEnabled == hapticsEnabled)&&(identical(other.profilePrivate, profilePrivate) || other.profilePrivate == profilePrivate));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,challengesFrom,showOnlineStatus,allowFriendRequests,soundEnabled,musicEnabled,hapticsEnabled,profilePrivate);
}

@override
String toString() {
    return 'UserSettings(challengesFrom: $challengesFrom, showOnlineStatus: $showOnlineStatus, allowFriendRequests: $allowFriendRequests, soundEnabled: $soundEnabled, musicEnabled: $musicEnabled, hapticsEnabled: $hapticsEnabled, profilePrivate: $profilePrivate)';
}


}

/// @nodoc
abstract mixin class _$UserSettingsCopyWith<$Res> implements $UserSettingsCopyWith<$Res> {
  factory _$UserSettingsCopyWith(_UserSettings value, $Res Function(_UserSettings) _then) = __$UserSettingsCopyWithImpl;
@override @useResult
$Res call({
 ChallengesFrom challengesFrom, bool showOnlineStatus, bool allowFriendRequests, bool soundEnabled, bool musicEnabled, bool hapticsEnabled, bool profilePrivate
});




}
/// @nodoc
class __$UserSettingsCopyWithImpl<$Res>
    implements _$UserSettingsCopyWith<$Res> {
  __$UserSettingsCopyWithImpl(this._self, this._then);

  final _UserSettings _self;
  final $Res Function(_UserSettings) _then;

/// Create a copy of UserSettings
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? challengesFrom = null,Object? showOnlineStatus = null,Object? allowFriendRequests = null,Object? soundEnabled = null,Object? musicEnabled = null,Object? hapticsEnabled = null,Object? profilePrivate = null,}) {
  return _then(_UserSettings(
challengesFrom: null == challengesFrom ? _self.challengesFrom : challengesFrom // ignore: cast_nullable_to_non_nullable
as ChallengesFrom,showOnlineStatus: null == showOnlineStatus ? _self.showOnlineStatus : showOnlineStatus // ignore: cast_nullable_to_non_nullable
as bool,allowFriendRequests: null == allowFriendRequests ? _self.allowFriendRequests : allowFriendRequests // ignore: cast_nullable_to_non_nullable
as bool,soundEnabled: null == soundEnabled ? _self.soundEnabled : soundEnabled // ignore: cast_nullable_to_non_nullable
as bool,musicEnabled: null == musicEnabled ? _self.musicEnabled : musicEnabled // ignore: cast_nullable_to_non_nullable
as bool,hapticsEnabled: null == hapticsEnabled ? _self.hapticsEnabled : hapticsEnabled // ignore: cast_nullable_to_non_nullable
as bool,profilePrivate: null == profilePrivate ? _self.profilePrivate : profilePrivate // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

// dart format on
