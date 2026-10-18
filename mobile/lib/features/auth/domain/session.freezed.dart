// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'session.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Session {

 SessionUser get user; ProfileSummary get profile; Tier get tier; AccountStatus get status; SessionFlags get flags;
/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SessionCopyWith<Session> get copyWith => _$SessionCopyWithImpl<Session>(this as Session, _$identity);

  /// Serializes this Session to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as Session;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Session&&(identical(other.user, _this.user) || other.user == _this.user)&&(identical(other.profile, _this.profile) || other.profile == _this.profile)&&(identical(other.tier, _this.tier) || other.tier == _this.tier)&&(identical(other.status, _this.status) || other.status == _this.status)&&(identical(other.flags, _this.flags) || other.flags == _this.flags));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as Session;
  return Object.hash(runtimeType,_this.user,_this.profile,_this.tier,_this.status,_this.flags);
}

@override
String toString() {
  final _this = this as Session;
  return 'Session(user: ${_this.user}, profile: ${_this.profile}, tier: ${_this.tier}, status: ${_this.status}, flags: ${_this.flags})';
}


}

/// @nodoc
abstract mixin class $SessionCopyWith<$Res>  {
  factory $SessionCopyWith(Session value, $Res Function(Session) _then) = _$SessionCopyWithImpl;
@useResult
$Res call({
 SessionUser user, ProfileSummary profile, Tier tier, AccountStatus status, SessionFlags flags
});


$SessionUserCopyWith<$Res> get user;$ProfileSummaryCopyWith<$Res> get profile;$SessionFlagsCopyWith<$Res> get flags;

}
/// @nodoc
class _$SessionCopyWithImpl<$Res>
    implements $SessionCopyWith<$Res> {
  _$SessionCopyWithImpl(this._self, this._then);

  final Session _self;
  final $Res Function(Session) _then;

/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? user = null,Object? profile = null,Object? tier = null,Object? status = null,Object? flags = null,}) {
  return _then(Session(
user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as SessionUser,profile: null == profile ? _self.profile : profile // ignore: cast_nullable_to_non_nullable
as ProfileSummary,tier: null == tier ? _self.tier : tier // ignore: cast_nullable_to_non_nullable
as Tier,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as AccountStatus,flags: null == flags ? _self.flags : flags // ignore: cast_nullable_to_non_nullable
as SessionFlags,
  ));
}
/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SessionUserCopyWith<$Res> get user {
  
  return $SessionUserCopyWith<$Res>(_self.user, (value) {
    return _then(_self.copyWith(user: value));
  });
}/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ProfileSummaryCopyWith<$Res> get profile {
  
  return $ProfileSummaryCopyWith<$Res>(_self.profile, (value) {
    return _then(_self.copyWith(profile: value));
  });
}/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SessionFlagsCopyWith<$Res> get flags {
  
  return $SessionFlagsCopyWith<$Res>(_self.flags, (value) {
    return _then(_self.copyWith(flags: value));
  });
}
}


/// Adds pattern-matching-related methods to [Session].
extension SessionPatterns on Session {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Session value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Session() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Session value)  $default,){
final _that = this;
switch (_that) {
case _Session():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Session value)?  $default,){
final _that = this;
switch (_that) {
case _Session() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( SessionUser user,  ProfileSummary profile,  Tier tier,  AccountStatus status,  SessionFlags flags)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Session() when $default != null:
return $default(_that.user,_that.profile,_that.tier,_that.status,_that.flags);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( SessionUser user,  ProfileSummary profile,  Tier tier,  AccountStatus status,  SessionFlags flags)  $default,) {final _that = this;
switch (_that) {
case _Session():
return $default(_that.user,_that.profile,_that.tier,_that.status,_that.flags);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( SessionUser user,  ProfileSummary profile,  Tier tier,  AccountStatus status,  SessionFlags flags)?  $default,) {final _that = this;
switch (_that) {
case _Session() when $default != null:
return $default(_that.user,_that.profile,_that.tier,_that.status,_that.flags);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Session implements Session {
  const _Session({required this.user, required this.profile, required this.tier, required this.status, required this.flags});
  factory _Session.fromJson(Map<String, dynamic> json) => _$SessionFromJson(json);

@override final  SessionUser user;
@override final  ProfileSummary profile;
@override final  Tier tier;
@override final  AccountStatus status;
@override final  SessionFlags flags;

/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SessionCopyWith<_Session> get copyWith => __$SessionCopyWithImpl<_Session>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SessionToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _Session&&(identical(other.user, user) || other.user == user)&&(identical(other.profile, profile) || other.profile == profile)&&(identical(other.tier, tier) || other.tier == tier)&&(identical(other.status, status) || other.status == status)&&(identical(other.flags, flags) || other.flags == flags));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,user,profile,tier,status,flags);
}

@override
String toString() {
    return 'Session(user: $user, profile: $profile, tier: $tier, status: $status, flags: $flags)';
}


}

/// @nodoc
abstract mixin class _$SessionCopyWith<$Res> implements $SessionCopyWith<$Res> {
  factory _$SessionCopyWith(_Session value, $Res Function(_Session) _then) = __$SessionCopyWithImpl;
@override @useResult
$Res call({
 SessionUser user, ProfileSummary profile, Tier tier, AccountStatus status, SessionFlags flags
});


@override $SessionUserCopyWith<$Res> get user;@override $ProfileSummaryCopyWith<$Res> get profile;@override $SessionFlagsCopyWith<$Res> get flags;

}
/// @nodoc
class __$SessionCopyWithImpl<$Res>
    implements _$SessionCopyWith<$Res> {
  __$SessionCopyWithImpl(this._self, this._then);

  final _Session _self;
  final $Res Function(_Session) _then;

/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? user = null,Object? profile = null,Object? tier = null,Object? status = null,Object? flags = null,}) {
  return _then(_Session(
user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as SessionUser,profile: null == profile ? _self.profile : profile // ignore: cast_nullable_to_non_nullable
as ProfileSummary,tier: null == tier ? _self.tier : tier // ignore: cast_nullable_to_non_nullable
as Tier,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as AccountStatus,flags: null == flags ? _self.flags : flags // ignore: cast_nullable_to_non_nullable
as SessionFlags,
  ));
}

/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SessionUserCopyWith<$Res> get user {
  
  return $SessionUserCopyWith<$Res>(_self.user, (value) {
    return _then(_self.copyWith(user: value));
  });
}/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ProfileSummaryCopyWith<$Res> get profile {
  
  return $ProfileSummaryCopyWith<$Res>(_self.profile, (value) {
    return _then(_self.copyWith(profile: value));
  });
}/// Create a copy of Session
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SessionFlagsCopyWith<$Res> get flags {
  
  return $SessionFlagsCopyWith<$Res>(_self.flags, (value) {
    return _then(_self.copyWith(flags: value));
  });
}
}


/// @nodoc
mixin _$SessionUser {

 String get id; List<String> get roles; DateTime? get deletionRequestedAt;
/// Create a copy of SessionUser
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SessionUserCopyWith<SessionUser> get copyWith => _$SessionUserCopyWithImpl<SessionUser>(this as SessionUser, _$identity);

  /// Serializes this SessionUser to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as SessionUser;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SessionUser&&(identical(other.id, _this.id) || other.id == _this.id)&&const DeepCollectionEquality().equals(other.roles, _this.roles)&&(identical(other.deletionRequestedAt, _this.deletionRequestedAt) || other.deletionRequestedAt == _this.deletionRequestedAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as SessionUser;
  return Object.hash(runtimeType,_this.id,const DeepCollectionEquality().hash(_this.roles),_this.deletionRequestedAt);
}

@override
String toString() {
  final _this = this as SessionUser;
  return 'SessionUser(id: ${_this.id}, roles: ${_this.roles}, deletionRequestedAt: ${_this.deletionRequestedAt})';
}


}

/// @nodoc
abstract mixin class $SessionUserCopyWith<$Res>  {
  factory $SessionUserCopyWith(SessionUser value, $Res Function(SessionUser) _then) = _$SessionUserCopyWithImpl;
@useResult
$Res call({
 String id, List<String> roles, DateTime? deletionRequestedAt
});




}
/// @nodoc
class _$SessionUserCopyWithImpl<$Res>
    implements $SessionUserCopyWith<$Res> {
  _$SessionUserCopyWithImpl(this._self, this._then);

  final SessionUser _self;
  final $Res Function(SessionUser) _then;

/// Create a copy of SessionUser
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? roles = null,Object? deletionRequestedAt = freezed,}) {
  return _then(SessionUser(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,roles: null == roles ? _self.roles : roles // ignore: cast_nullable_to_non_nullable
as List<String>,deletionRequestedAt: freezed == deletionRequestedAt ? _self.deletionRequestedAt : deletionRequestedAt // ignore: cast_nullable_to_non_nullable
as DateTime?,
  ));
}

}


/// Adds pattern-matching-related methods to [SessionUser].
extension SessionUserPatterns on SessionUser {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SessionUser value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SessionUser() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SessionUser value)  $default,){
final _that = this;
switch (_that) {
case _SessionUser():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SessionUser value)?  $default,){
final _that = this;
switch (_that) {
case _SessionUser() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  List<String> roles,  DateTime? deletionRequestedAt)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SessionUser() when $default != null:
return $default(_that.id,_that.roles,_that.deletionRequestedAt);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  List<String> roles,  DateTime? deletionRequestedAt)  $default,) {final _that = this;
switch (_that) {
case _SessionUser():
return $default(_that.id,_that.roles,_that.deletionRequestedAt);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  List<String> roles,  DateTime? deletionRequestedAt)?  $default,) {final _that = this;
switch (_that) {
case _SessionUser() when $default != null:
return $default(_that.id,_that.roles,_that.deletionRequestedAt);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SessionUser implements SessionUser {
  const _SessionUser({required this.id,  List<String> roles = const <String>[], this.deletionRequestedAt}): _roles = roles;
  factory _SessionUser.fromJson(Map<String, dynamic> json) => _$SessionUserFromJson(json);

@override final  String id;
 final  List<String> _roles;
@override@JsonKey() List<String> get roles {
  if (_roles is EqualUnmodifiableListView) return _roles;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_roles);
}

@override final  DateTime? deletionRequestedAt;

/// Create a copy of SessionUser
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SessionUserCopyWith<_SessionUser> get copyWith => __$SessionUserCopyWithImpl<_SessionUser>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SessionUserToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _SessionUser&&(identical(other.id, id) || other.id == id)&&const DeepCollectionEquality().equals(other.roles, _roles)&&(identical(other.deletionRequestedAt, deletionRequestedAt) || other.deletionRequestedAt == deletionRequestedAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,id,const DeepCollectionEquality().hash(_roles),deletionRequestedAt);
}

@override
String toString() {
    return 'SessionUser(id: $id, roles: $roles, deletionRequestedAt: $deletionRequestedAt)';
}


}

/// @nodoc
abstract mixin class _$SessionUserCopyWith<$Res> implements $SessionUserCopyWith<$Res> {
  factory _$SessionUserCopyWith(_SessionUser value, $Res Function(_SessionUser) _then) = __$SessionUserCopyWithImpl;
@override @useResult
$Res call({
 String id, List<String> roles, DateTime? deletionRequestedAt
});




}
/// @nodoc
class __$SessionUserCopyWithImpl<$Res>
    implements _$SessionUserCopyWith<$Res> {
  __$SessionUserCopyWithImpl(this._self, this._then);

  final _SessionUser _self;
  final $Res Function(_SessionUser) _then;

/// Create a copy of SessionUser
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? roles = null,Object? deletionRequestedAt = freezed,}) {
  return _then(_SessionUser(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,roles: null == roles ? _self._roles : roles // ignore: cast_nullable_to_non_nullable
as List<String>,deletionRequestedAt: freezed == deletionRequestedAt ? _self.deletionRequestedAt : deletionRequestedAt // ignore: cast_nullable_to_non_nullable
as DateTime?,
  ));
}


}


/// @nodoc
mixin _$ProfileSummary {

 String get username; String get displayName; String get avatarId; String get locale; String? get countryCode;
/// Create a copy of ProfileSummary
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ProfileSummaryCopyWith<ProfileSummary> get copyWith => _$ProfileSummaryCopyWithImpl<ProfileSummary>(this as ProfileSummary, _$identity);

  /// Serializes this ProfileSummary to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as ProfileSummary;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ProfileSummary&&(identical(other.username, _this.username) || other.username == _this.username)&&(identical(other.displayName, _this.displayName) || other.displayName == _this.displayName)&&(identical(other.avatarId, _this.avatarId) || other.avatarId == _this.avatarId)&&(identical(other.locale, _this.locale) || other.locale == _this.locale)&&(identical(other.countryCode, _this.countryCode) || other.countryCode == _this.countryCode));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as ProfileSummary;
  return Object.hash(runtimeType,_this.username,_this.displayName,_this.avatarId,_this.locale,_this.countryCode);
}

@override
String toString() {
  final _this = this as ProfileSummary;
  return 'ProfileSummary(username: ${_this.username}, displayName: ${_this.displayName}, avatarId: ${_this.avatarId}, locale: ${_this.locale}, countryCode: ${_this.countryCode})';
}


}

/// @nodoc
abstract mixin class $ProfileSummaryCopyWith<$Res>  {
  factory $ProfileSummaryCopyWith(ProfileSummary value, $Res Function(ProfileSummary) _then) = _$ProfileSummaryCopyWithImpl;
@useResult
$Res call({
 String username, String displayName, String avatarId, String locale, String? countryCode
});




}
/// @nodoc
class _$ProfileSummaryCopyWithImpl<$Res>
    implements $ProfileSummaryCopyWith<$Res> {
  _$ProfileSummaryCopyWithImpl(this._self, this._then);

  final ProfileSummary _self;
  final $Res Function(ProfileSummary) _then;

/// Create a copy of ProfileSummary
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? username = null,Object? displayName = null,Object? avatarId = null,Object? locale = null,Object? countryCode = freezed,}) {
  return _then(ProfileSummary(
username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,displayName: null == displayName ? _self.displayName : displayName // ignore: cast_nullable_to_non_nullable
as String,avatarId: null == avatarId ? _self.avatarId : avatarId // ignore: cast_nullable_to_non_nullable
as String,locale: null == locale ? _self.locale : locale // ignore: cast_nullable_to_non_nullable
as String,countryCode: freezed == countryCode ? _self.countryCode : countryCode // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [ProfileSummary].
extension ProfileSummaryPatterns on ProfileSummary {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ProfileSummary value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ProfileSummary() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ProfileSummary value)  $default,){
final _that = this;
switch (_that) {
case _ProfileSummary():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ProfileSummary value)?  $default,){
final _that = this;
switch (_that) {
case _ProfileSummary() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String username,  String displayName,  String avatarId,  String locale,  String? countryCode)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ProfileSummary() when $default != null:
return $default(_that.username,_that.displayName,_that.avatarId,_that.locale,_that.countryCode);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String username,  String displayName,  String avatarId,  String locale,  String? countryCode)  $default,) {final _that = this;
switch (_that) {
case _ProfileSummary():
return $default(_that.username,_that.displayName,_that.avatarId,_that.locale,_that.countryCode);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String username,  String displayName,  String avatarId,  String locale,  String? countryCode)?  $default,) {final _that = this;
switch (_that) {
case _ProfileSummary() when $default != null:
return $default(_that.username,_that.displayName,_that.avatarId,_that.locale,_that.countryCode);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ProfileSummary implements ProfileSummary {
  const _ProfileSummary({required this.username, required this.displayName, required this.avatarId, required this.locale, this.countryCode});
  factory _ProfileSummary.fromJson(Map<String, dynamic> json) => _$ProfileSummaryFromJson(json);

@override final  String username;
@override final  String displayName;
@override final  String avatarId;
@override final  String locale;
@override final  String? countryCode;

/// Create a copy of ProfileSummary
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ProfileSummaryCopyWith<_ProfileSummary> get copyWith => __$ProfileSummaryCopyWithImpl<_ProfileSummary>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ProfileSummaryToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _ProfileSummary&&(identical(other.username, username) || other.username == username)&&(identical(other.displayName, displayName) || other.displayName == displayName)&&(identical(other.avatarId, avatarId) || other.avatarId == avatarId)&&(identical(other.locale, locale) || other.locale == locale)&&(identical(other.countryCode, countryCode) || other.countryCode == countryCode));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,username,displayName,avatarId,locale,countryCode);
}

@override
String toString() {
    return 'ProfileSummary(username: $username, displayName: $displayName, avatarId: $avatarId, locale: $locale, countryCode: $countryCode)';
}


}

/// @nodoc
abstract mixin class _$ProfileSummaryCopyWith<$Res> implements $ProfileSummaryCopyWith<$Res> {
  factory _$ProfileSummaryCopyWith(_ProfileSummary value, $Res Function(_ProfileSummary) _then) = __$ProfileSummaryCopyWithImpl;
@override @useResult
$Res call({
 String username, String displayName, String avatarId, String locale, String? countryCode
});




}
/// @nodoc
class __$ProfileSummaryCopyWithImpl<$Res>
    implements _$ProfileSummaryCopyWith<$Res> {
  __$ProfileSummaryCopyWithImpl(this._self, this._then);

  final _ProfileSummary _self;
  final $Res Function(_ProfileSummary) _then;

/// Create a copy of ProfileSummary
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? username = null,Object? displayName = null,Object? avatarId = null,Object? locale = null,Object? countryCode = freezed,}) {
  return _then(_ProfileSummary(
username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,displayName: null == displayName ? _self.displayName : displayName // ignore: cast_nullable_to_non_nullable
as String,avatarId: null == avatarId ? _self.avatarId : avatarId // ignore: cast_nullable_to_non_nullable
as String,locale: null == locale ? _self.locale : locale // ignore: cast_nullable_to_non_nullable
as String,countryCode: freezed == countryCode ? _self.countryCode : countryCode // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}


/// @nodoc
mixin _$SessionFlags {

 bool get created; bool get restricted; bool get emailVerificationRequired; bool get deletionPending; bool get suspended;
/// Create a copy of SessionFlags
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SessionFlagsCopyWith<SessionFlags> get copyWith => _$SessionFlagsCopyWithImpl<SessionFlags>(this as SessionFlags, _$identity);

  /// Serializes this SessionFlags to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  final _this = this as SessionFlags;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SessionFlags&&(identical(other.created, _this.created) || other.created == _this.created)&&(identical(other.restricted, _this.restricted) || other.restricted == _this.restricted)&&(identical(other.emailVerificationRequired, _this.emailVerificationRequired) || other.emailVerificationRequired == _this.emailVerificationRequired)&&(identical(other.deletionPending, _this.deletionPending) || other.deletionPending == _this.deletionPending)&&(identical(other.suspended, _this.suspended) || other.suspended == _this.suspended));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
  final _this = this as SessionFlags;
  return Object.hash(runtimeType,_this.created,_this.restricted,_this.emailVerificationRequired,_this.deletionPending,_this.suspended);
}

@override
String toString() {
  final _this = this as SessionFlags;
  return 'SessionFlags(created: ${_this.created}, restricted: ${_this.restricted}, emailVerificationRequired: ${_this.emailVerificationRequired}, deletionPending: ${_this.deletionPending}, suspended: ${_this.suspended})';
}


}

/// @nodoc
abstract mixin class $SessionFlagsCopyWith<$Res>  {
  factory $SessionFlagsCopyWith(SessionFlags value, $Res Function(SessionFlags) _then) = _$SessionFlagsCopyWithImpl;
@useResult
$Res call({
 bool created, bool restricted, bool emailVerificationRequired, bool deletionPending, bool suspended
});




}
/// @nodoc
class _$SessionFlagsCopyWithImpl<$Res>
    implements $SessionFlagsCopyWith<$Res> {
  _$SessionFlagsCopyWithImpl(this._self, this._then);

  final SessionFlags _self;
  final $Res Function(SessionFlags) _then;

/// Create a copy of SessionFlags
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? created = null,Object? restricted = null,Object? emailVerificationRequired = null,Object? deletionPending = null,Object? suspended = null,}) {
  return _then(SessionFlags(
created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as bool,restricted: null == restricted ? _self.restricted : restricted // ignore: cast_nullable_to_non_nullable
as bool,emailVerificationRequired: null == emailVerificationRequired ? _self.emailVerificationRequired : emailVerificationRequired // ignore: cast_nullable_to_non_nullable
as bool,deletionPending: null == deletionPending ? _self.deletionPending : deletionPending // ignore: cast_nullable_to_non_nullable
as bool,suspended: null == suspended ? _self.suspended : suspended // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [SessionFlags].
extension SessionFlagsPatterns on SessionFlags {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SessionFlags value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SessionFlags() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SessionFlags value)  $default,){
final _that = this;
switch (_that) {
case _SessionFlags():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SessionFlags value)?  $default,){
final _that = this;
switch (_that) {
case _SessionFlags() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( bool created,  bool restricted,  bool emailVerificationRequired,  bool deletionPending,  bool suspended)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SessionFlags() when $default != null:
return $default(_that.created,_that.restricted,_that.emailVerificationRequired,_that.deletionPending,_that.suspended);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( bool created,  bool restricted,  bool emailVerificationRequired,  bool deletionPending,  bool suspended)  $default,) {final _that = this;
switch (_that) {
case _SessionFlags():
return $default(_that.created,_that.restricted,_that.emailVerificationRequired,_that.deletionPending,_that.suspended);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( bool created,  bool restricted,  bool emailVerificationRequired,  bool deletionPending,  bool suspended)?  $default,) {final _that = this;
switch (_that) {
case _SessionFlags() when $default != null:
return $default(_that.created,_that.restricted,_that.emailVerificationRequired,_that.deletionPending,_that.suspended);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SessionFlags implements SessionFlags {
  const _SessionFlags({required this.created, required this.restricted, required this.emailVerificationRequired, required this.deletionPending, required this.suspended});
  factory _SessionFlags.fromJson(Map<String, dynamic> json) => _$SessionFlagsFromJson(json);

@override final  bool created;
@override final  bool restricted;
@override final  bool emailVerificationRequired;
@override final  bool deletionPending;
@override final  bool suspended;

/// Create a copy of SessionFlags
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SessionFlagsCopyWith<_SessionFlags> get copyWith => __$SessionFlagsCopyWithImpl<_SessionFlags>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SessionFlagsToJson(this, );
}

@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _SessionFlags&&(identical(other.created, created) || other.created == created)&&(identical(other.restricted, restricted) || other.restricted == restricted)&&(identical(other.emailVerificationRequired, emailVerificationRequired) || other.emailVerificationRequired == emailVerificationRequired)&&(identical(other.deletionPending, deletionPending) || other.deletionPending == deletionPending)&&(identical(other.suspended, suspended) || other.suspended == suspended));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode {
    return Object.hash(runtimeType,created,restricted,emailVerificationRequired,deletionPending,suspended);
}

@override
String toString() {
    return 'SessionFlags(created: $created, restricted: $restricted, emailVerificationRequired: $emailVerificationRequired, deletionPending: $deletionPending, suspended: $suspended)';
}


}

/// @nodoc
abstract mixin class _$SessionFlagsCopyWith<$Res> implements $SessionFlagsCopyWith<$Res> {
  factory _$SessionFlagsCopyWith(_SessionFlags value, $Res Function(_SessionFlags) _then) = __$SessionFlagsCopyWithImpl;
@override @useResult
$Res call({
 bool created, bool restricted, bool emailVerificationRequired, bool deletionPending, bool suspended
});




}
/// @nodoc
class __$SessionFlagsCopyWithImpl<$Res>
    implements _$SessionFlagsCopyWith<$Res> {
  __$SessionFlagsCopyWithImpl(this._self, this._then);

  final _SessionFlags _self;
  final $Res Function(_SessionFlags) _then;

/// Create a copy of SessionFlags
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? created = null,Object? restricted = null,Object? emailVerificationRequired = null,Object? deletionPending = null,Object? suspended = null,}) {
  return _then(_SessionFlags(
created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as bool,restricted: null == restricted ? _self.restricted : restricted // ignore: cast_nullable_to_non_nullable
as bool,emailVerificationRequired: null == emailVerificationRequired ? _self.emailVerificationRequired : emailVerificationRequired // ignore: cast_nullable_to_non_nullable
as bool,deletionPending: null == deletionPending ? _self.deletionPending : deletionPending // ignore: cast_nullable_to_non_nullable
as bool,suspended: null == suspended ? _self.suspended : suspended // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

// dart format on
