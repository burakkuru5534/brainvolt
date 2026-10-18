import 'dart:async';
import 'dart:convert';

import 'package:brainvolt/app/config.dart';
import 'package:brainvolt/app/providers.dart';
import 'package:brainvolt/core/network/token_source.dart';
import 'package:brainvolt/core/realtime/realtime_client.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/features/auth/data/auth_providers.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:brainvolt/features/auth/domain/session_repository.dart';
import 'package:brainvolt/features/content/data/content_providers.dart';
import 'package:brainvolt/features/content/domain/content_repository.dart';
import 'package:brainvolt/features/content/domain/credit.dart';
import 'package:brainvolt/features/match/data/match_providers.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/profile/data/profile_providers.dart';
import 'package:brainvolt/features/profile/domain/me.dart';
import 'package:brainvolt/features/profile/domain/profile_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Firebase Auth in memory. Every call is appended to [calls].
class FakeAuthRepository implements AuthRepository {
  FakeAuthRepository({this._user});

  AuthUser? _user;
  final calls = <String>[];
  int _nextUid = 1;

  /// Accounts that already own a credential: linking it conflicts.
  final existing = <AuthMethod, AuthUser>{};

  /// Thrown by the next [obtainCredential].
  AuthFailure? obtainFailure;

  /// Thrown by the next [signIn].
  AuthFailure? signInFailure;

  /// Email is verified after the next [reload].
  bool verifyOnReload = false;

  @override
  AuthUser? get currentUser => _user;

  @override
  Future<AuthUser> signInAnonymously() async {
    calls.add('signInAnonymously');
    return _user = AuthUser(uid: 'guest-${_nextUid++}', isAnonymous: true);
  }

  @override
  Future<ProviderCredential> obtainCredential(
    AuthMethod method, {
    String? email,
    String? password,
  }) async {
    calls.add('obtain:${method.name}');
    final failure = obtainFailure;
    if (failure != null) {
      obtainFailure = null;
      throw failure;
    }
    return ProviderCredential(method, '${method.name}:$email');
  }

  @override
  Future<AuthUser> link(ProviderCredential credential) async {
    calls.add('link:${credential.method.name}');
    if (existing.containsKey(credential.method)) {
      throw AuthFailure(
        AuthFailureKind.credentialAlreadyInUse,
        conflict: credential,
      );
    }
    final email = credential.method == AuthMethod.email
        ? (credential.raw as String).split(':').last
        : '${credential.method.name}@example.com';
    return _user = AuthUser(
      uid: _user!.uid,
      isAnonymous: false,
      email: email,
      emailVerified: credential.method != AuthMethod.email,
      providers: {credential.method},
    );
  }

  @override
  Future<AuthUser> signIn(ProviderCredential credential) async {
    calls.add('signIn:${credential.method.name}');
    final failure = signInFailure;
    if (failure != null) {
      signInFailure = null;
      throw failure;
    }
    return _user = existing[credential.method]!;
  }

  @override
  Future<void> sendEmailVerification() async =>
      calls.add('sendEmailVerification');

  @override
  Future<AuthUser?> reload() async {
    calls.add('reload');
    final u = _user;
    if (u != null && verifyOnReload) {
      _user = AuthUser(
        uid: u.uid,
        isAnonymous: false,
        email: u.email,
        emailVerified: true,
        providers: u.providers,
      );
    }
    return _user;
  }

  @override
  Future<String?> idToken({bool forceRefresh = false}) async {
    calls.add(forceRefresh ? 'idToken:refresh' : 'idToken');
    final u = _user;
    return u == null ? null : 'token-of-${u.uid}';
  }

  @override
  Future<void> signOut() async {
    calls.add('signOut');
    _user = null;
  }
}

/// The identity endpoints in memory: the session mirrors the fake auth user.
class FakeSessionRepository implements SessionRepository {
  FakeSessionRepository(this.auth);

  final FakeAuthRepository auth;
  final calls = <String>[];

  /// Thrown by the next [sync].
  Object? syncError;
  DateTime? deletionRequestedAt;
  AccountStatus status = AccountStatus.active;

  @override
  Future<Session> sync() async {
    calls.add('sync');
    final error = syncError;
    if (error != null) {
      syncError = null;
      throw error;
    }
    final u = auth.currentUser!;
    final tier = u.isAnonymous
        ? Tier.guest
        : u.emailVerified
        ? Tier.full
        : Tier.unverified;
    return testSession(
      uid: u.uid,
      tier: tier,
      status: status,
      deletionRequestedAt: deletionRequestedAt,
    );
  }

  @override
  Future<void> registerDevice() async => calls.add('registerDevice');

  @override
  Future<void> unregisterDevice() async => calls.add('unregisterDevice');

  @override
  Future<void> abandonGuest({required String guestToken}) async =>
      calls.add('abandonGuest:$guestToken');

  @override
  Future<DateTime> requestDeletion() async {
    calls.add('requestDeletion');
    deletionRequestedAt = DateTime.utc(2026, 10, 18);
    return deletionRequestedAt!.add(deletionGracePeriod);
  }

  @override
  Future<void> cancelDeletion() async {
    calls.add('cancelDeletion');
    deletionRequestedAt = null;
  }
}

/// A session as the backend would answer it.
Session testSession({
  String uid = 'guest-1',
  Tier tier = Tier.guest,
  AccountStatus status = AccountStatus.active,
  DateTime? deletionRequestedAt,
  String displayName = 'Misafir 1234',
}) => Session(
  user: SessionUser(id: 'user-$uid', deletionRequestedAt: deletionRequestedAt),
  profile: ProfileSummary(
    username: 'oyuncu12345',
    displayName: displayName,
    avatarId: 'fox',
    locale: 'tr',
  ),
  tier: tier,
  status: status,
  flags: SessionFlags(
    created: false,
    restricted: tier != Tier.full,
    emailVerificationRequired: tier == Tier.unverified,
    deletionPending: deletionRequestedAt != null,
    suspended: status == AccountStatus.suspended,
  ),
);

/// The profile endpoints in memory.
class FakeProfileRepository implements ProfileRepository {
  FakeProfileRepository({this.tier = Tier.guest});

  Tier tier;
  final calls = <String>[];

  /// Thrown by the next [updateSettings].
  Object? settingsError;

  OwnProfile profile = const OwnProfile(
    username: 'oyuncu12345',
    displayName: 'Misafir 1234',
    avatarId: 'fox',
    locale: 'tr',
  );

  UserSettings userSettings = const UserSettings(
    challengesFrom: ChallengesFrom.everyone,
    showOnlineStatus: true,
    allowFriendRequests: true,
    soundEnabled: true,
    musicEnabled: true,
    hapticsEnabled: true,
  );

  @override
  Future<Me> me() async {
    calls.add('me');
    return Me(
      user: MeUser(id: 'user-1', tier: tier),
      profile: profile,
      settings: userSettings,
    );
  }

  @override
  Future<OwnProfile> updateProfile({
    String? displayName,
    String? username,
    String? avatarId,
    String? locale,
  }) async {
    calls.add(
      'updateProfile:${{'display_name': ?displayName, 'username': ?username, 'avatar_id': ?avatarId, 'locale': ?locale}}',
    );
    return profile = profile.copyWith(
      displayName: displayName ?? profile.displayName,
      username: username ?? profile.username,
      avatarId: avatarId ?? profile.avatarId,
      locale: locale ?? profile.locale,
    );
  }

  @override
  Future<UserSettings> settings() async => userSettings;

  @override
  Future<UserSettings> updateSettings(UserSettings settings) async {
    calls.add('updateSettings');
    final error = settingsError;
    if (error != null) {
      settingsError = null;
      throw error;
    }
    return userSettings = settings;
  }
}

/// The content endpoints in memory.
class FakeContentRepository implements ContentRepository {
  List<Credit> sources = const [
    Credit(
      name: 'BrainVolt',
      licenseCode: 'LicenseRef-BrainVolt',
      attributionText: 'BrainVolt team',
      shareAlike: false,
    ),
    Credit(
      name: 'Open Trivia Database',
      licenseCode: 'CC-BY-SA-4.0',
      attributionText: 'Open Trivia Database (opentdb.com), CC BY-SA 4.0',
      shareAlike: true,
      url: 'https://opentdb.com',
    ),
  ];

  @override
  Future<List<Credit>> credits() async => sources;
}

/// The game and matchmaking endpoints in memory.
class FakeMatchRepository implements MatchRepository {
  final calls = <String>[];

  /// What enqueue answers; a practice ticket carries a match id.
  QueueTicket ticket = const QueueTicket(id: 't1', state: 'searching');
  String? current;
  MatchResult? matchResult;

  @override
  Future<QueueTicket> enqueue({bool practice = false}) async {
    calls.add(practice ? 'enqueue:practice' : 'enqueue');
    return ticket;
  }

  @override
  Future<void> leaveQueue() async => calls.add('leaveQueue');

  @override
  Future<String?> currentMatch() async => current;

  @override
  Future<MatchResult> result(String matchId) async => matchResult!;
}

/// A WebSocket whose server side the test drives.
class FakeSocket implements SocketConnection {
  final _in = StreamController<dynamic>();
  final sent = <Map<String, dynamic>>[];

  @override
  int? closeCode;

  @override
  Stream<dynamic> get stream => _in.stream;

  @override
  void add(String data) => sent.add(jsonDecode(data) as Map<String, dynamic>);

  @override
  Future<void> close([int? code, String? reason]) async {
    closeCode ??= code;
    await _in.close();
  }

  /// Sends a frame from the server.
  void serverSends(Map<String, dynamic> frame) => _in.add(jsonEncode(frame));

  /// Closes from the server side with [code].
  Future<void> serverCloses(int code) async {
    closeCode = code;
    await _in.close();
  }

  /// The last client message of [type].
  Map<String, dynamic> lastSent(String type) =>
      sent.lastWhere((m) => m['t'] == type);

  /// Acknowledges the last client message of [type].
  void ack(String type) =>
      serverSends({'t': 'sys.ack', 're': lastSent(type)['id']});
}

/// Everything a test app needs, with fakes behind the repositories.
class TestApp {
  TestApp._(
    this.container,
    this.prefs,
    this.auth,
    this.sessions,
    this.profiles,
    this.matches,
    this.sockets,
  );

  /// [ageConfirmed] skips the age gate; [user] is an already signed-in user.
  static Future<TestApp> create({
    bool ageConfirmed = true,
    AuthUser? user,
    Map<String, Object> prefs = const {},
  }) async {
    SharedPreferences.setMockInitialValues({
      'age_confirmed': ?(ageConfirmed ? true : null),
      ...prefs,
    });
    final p = await SharedPreferences.getInstance();
    final auth = FakeAuthRepository(user: user);
    final sessions = FakeSessionRepository(auth);
    final profiles = FakeProfileRepository();
    final matches = FakeMatchRepository();
    final sockets = <FakeSocket>[];
    final container = ProviderContainer(
      overrides: [
        appConfigProvider.overrideWithValue(
          AppConfig.parse(
            flavor: 'dev',
            apiBaseUrl: 'http://10.0.2.2:8080',
            wsUrl: 'ws://10.0.2.2:8080/v1/ws',
          ),
        ),
        sharedPreferencesProvider.overrideWithValue(p),
        appVersionProvider.overrideWithValue('1.0.0+1'),
        authRepositoryProvider.overrideWithValue(auth),
        sessionRepositoryProvider.overrideWithValue(sessions),
        profileRepositoryProvider.overrideWithValue(profiles),
        contentRepositoryProvider.overrideWithValue(FakeContentRepository()),
        matchRepositoryProvider.overrideWithValue(matches),
        realtimeClientProvider.overrideWith((ref) {
          final client = RealtimeClient(
            url: Uri.parse('ws://test/v1/ws'),
            tokens: const _StaticTokens(),
            installationId: 'test-installation',
            connector: (url, headers) async {
              final s = FakeSocket();
              sockets.add(s);
              return s;
            },
          );
          ref.onDispose(client.dispose);
          return client;
        }),
      ],
    );
    addTearDown(container.dispose);
    return TestApp._(container, p, auth, sessions, profiles, matches, sockets);
  }

  final ProviderContainer container;
  final SharedPreferences prefs;
  final FakeMatchRepository matches;

  /// Every WebSocket the app opened, newest last.
  final List<FakeSocket> sockets;
  final FakeAuthRepository auth;
  final FakeSessionRepository sessions;
  final FakeProfileRepository profiles;
}

class _StaticTokens implements TokenSource {
  const _StaticTokens();

  @override
  Future<String?> idToken({bool forceRefresh = false}) async => 'test-token';
}
