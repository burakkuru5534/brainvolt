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
import 'package:brainvolt/features/daily/data/daily_providers.dart';
import 'package:brainvolt/features/daily/domain/daily_repository.dart';
import 'package:brainvolt/features/lobby/data/lobby_providers.dart';
import 'package:brainvolt/features/lobby/domain/lobby_repository.dart';
import 'package:brainvolt/features/match/data/match_providers.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/moderation/data/moderation_providers.dart';
import 'package:brainvolt/features/moderation/domain/moderation_repository.dart';
import 'package:brainvolt/features/profile/data/profile_providers.dart';
import 'package:brainvolt/features/profile/domain/me.dart';
import 'package:brainvolt/features/profile/domain/profile_repository.dart';
import 'package:brainvolt/features/progress/data/progress_providers.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
import 'package:brainvolt/features/social/data/social_providers.dart';
import 'package:brainvolt/features/social/domain/social_repository.dart';
import 'package:brainvolt/features/team/data/team_providers.dart';
import 'package:brainvolt/features/team/domain/team_repository.dart';
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

  /// The next [idToken] fails as if Firebase forgot the user.
  bool loseSession = false;

  @override
  Future<String?> idToken({bool forceRefresh = false}) async {
    calls.add(forceRefresh ? 'idToken:refresh' : 'idToken');
    if (loseSession) {
      loseSession = false;
      throw const AuthFailure(AuthFailureKind.sessionLost);
    }
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

  @override
  Future<List<CategoryInfo>> categories() async => const [
    CategoryInfo('cat-history', 'Tarih'),
    CategoryInfo('cat-science', 'Bilim'),
  ];
}

/// The game and matchmaking endpoints in memory.
class FakeMatchRepository implements MatchRepository {
  final calls = <String>[];

  /// What enqueue answers; a practice ticket carries a match id.
  QueueTicket ticket = const QueueTicket(id: 't1', state: 'searching');
  String? current;
  MatchResult? matchResult;

  @override
  Future<QueueTicket> enqueue({
    bool practice = false,
    QueueMode mode = const QueueMode(),
  }) async {
    calls.add(
      practice
          ? 'enqueue:practice'
          : mode.isDefault
          ? 'enqueue'
          : 'enqueue:${mode.matchType}:${mode.format}:${mode.categoryMode}:'
                '${mode.categoryId}:${mode.partyUserIds.join(',')}:${mode.teamId}',
    );
    return ticket;
  }

  @override
  Future<void> leaveQueue() async => calls.add('leaveQueue');

  @override
  Future<String?> currentMatch() async => current;

  @override
  Future<MatchResult> result(String matchId) async => matchResult!;
}

/// Friends and requests in memory; [failure] is thrown by every call.
class FakeSocialRepository implements SocialRepository {
  final calls = <String>[];
  List<Friend> friendList = [];
  List<FriendRequest> incoming = [];
  List<FriendRequest> outgoing = [];
  Object? failure;

  void _check() {
    if (failure != null) {
      throw failure!;
    }
  }

  @override
  Future<List<Friend>> friends() async {
    _check();
    return friendList;
  }

  @override
  Future<List<FriendRequest>> requests({required bool incoming}) async {
    _check();
    return incoming ? this.incoming : outgoing;
  }

  @override
  Future<bool> sendRequest({
    String? userId,
    String? username,
    String via = 'search',
  }) async {
    _check();
    calls.add('send:${userId ?? username}:$via');
    return false;
  }

  @override
  Future<void> respond(String requestId, RequestAction action) async =>
      calls.add('${action.name}:$requestId');

  @override
  Future<void> unfriend(String userId) async => calls.add('unfriend:$userId');

  @override
  Future<void> block(String userId) async => calls.add('block:$userId');
}

/// Codes and challenges in memory.
class FakeLobbyRepository implements LobbyRepository {
  final calls = <String>[];
  List<Challenge> incoming = [];
  List<Challenge> outgoing = [];

  /// What resolving any code gives.
  ResolvedCode resolved = const ResolvedCode(kind: 'friend');

  /// What responding to a challenge gives.
  Challenge? responded;

  @override
  Future<FriendCode> friendCode() async => FriendCode('ABC234', DateTime(2030));

  @override
  Future<ResolvedCode> resolveCode(String code) async {
    calls.add('resolve:$code');
    return resolved;
  }

  @override
  Future<Challenge> challenge(String targetUserId, MatchType type) async {
    calls.add('challenge:$targetUserId:${type.wire}');
    return Challenge(
      id: 'c-new',
      status: 'pending',
      challenger: const UserCard(userId: 'me', displayName: 'Ben'),
      target: UserCard(userId: targetUserId, displayName: ''),
      matchType: type.wire,
      expiresAt: DateTime(2030),
    );
  }

  @override
  Future<List<Challenge>> challenges({required bool incoming}) async =>
      incoming ? this.incoming : outgoing;

  @override
  Future<Challenge> respondChallenge(String id, ChallengeAction action) async {
    calls.add('${action.name}:$id');
    return responded!;
  }

  @override
  Future<Challenge> revenge(String matchId, String homeCategoryId) async {
    calls.add('revenge:$matchId:$homeCategoryId');
    return responded!;
  }

  /// What creating or joining a room gives.
  Room room1 = const Room(
    id: 'r1',
    status: 'open',
    version: 1,
    hostUserId: 'user-guest-1',
    code: 'ROOM23',
    matchType: 'quick_duel',
    capacity: 2,
    members: [],
  );

  @override
  Future<Room> createRoom(
    MatchType type, {
    String? categoryMode,
    String format = '1v1',
    String? teamMode,
  }) async {
    calls.add(
      format == '1v1'
          ? 'createRoom:${type.wire}'
          : 'createRoom:${type.wire}:$format:$teamMode',
    );
    return room1;
  }

  @override
  Future<Room> joinRoom(String code) async {
    calls.add('joinRoom:$code');
    return room1;
  }

  @override
  Future<Room> room(String id) async => room1;

  @override
  Future<void> leaveRoom(String id) async => calls.add('leaveRoom:$id');
}

/// Teams and couples in memory.
class FakeTeamRepository implements TeamRepository {
  final calls = <String>[];
  List<Team> teams = [];
  List<TeamInvite> pendingInvites = [];
  Couple? couple;

  @override
  Future<List<Team>> myTeams() async => teams;

  @override
  Future<Team> team(String id) async => teams.firstWhere((t) => t.id == id);

  @override
  Future<Team> createTeam({
    required String name,
    required String tag,
    required String emblemId,
  }) async {
    calls.add('create:$name:$tag:$emblemId');
    final t = Team(
      id: 't${teams.length + 1}',
      name: name,
      tag: tag,
      emblemId: emblemId,
      captainUserId: 'user-guest-1',
      members: const [],
    );
    teams = [...teams, t];
    return t;
  }

  @override
  Future<void> invite(String teamId, String userId) async =>
      calls.add('invite:$teamId:$userId');

  @override
  Future<List<TeamInvite>> invites() async => pendingInvites;

  @override
  Future<void> respondInvite(String inviteId, {required bool accept}) async {
    calls.add('${accept ? 'accept' : 'decline'}:$inviteId');
    pendingInvites = [];
  }

  @override
  Future<void> teamAction(String teamId, TeamAction action) async =>
      calls.add('${action.name}:$teamId');

  @override
  Future<void> kick(String teamId, String userId) async =>
      calls.add('kick:$teamId:$userId');

  @override
  Future<void> makeCaptain(String teamId, String userId) async =>
      calls.add('captain:$teamId:$userId');

  @override
  Future<Couple?> myCouple() async => couple;

  @override
  Future<Couple> requestCouple(String partnerUserId) async {
    calls.add('couple:$partnerUserId');
    return couple = Couple(
      id: 'cp1',
      status: 'pending',
      requestedBy: 'user-guest-1',
      partners: [
        const UserCard(userId: 'user-guest-1', displayName: 'Ben'),
        UserCard(userId: partnerUserId, displayName: 'Ali'),
      ],
    );
  }

  @override
  Future<Couple> respondCouple(String coupleId, String action) async {
    calls.add('$action:$coupleId');
    return couple!;
  }
}

/// A WebSocket whose server side the test drives.
class FakeProgressRepository implements ProgressRepository {
  Progress current = const Progress(
    xpTotal: 120,
    level: 2,
    levelXp: 20,
    levelSize: 125,
    streak: 3,
    longestStreak: 3,
    playedToday: true,
    badges: 2,
  );
  Standing myStanding = const Standing(
    seasonPoints: 130,
    league: 'bronze',
    division: 2,
    matches: 5,
    wins: 4,
    nextAt: 200,
  );
  QuestList questList = const QuestList(daily: [], weekly: [], rerollsLeft: 1);
  List<BadgeInfo> catalogue = const [];
  final boards = <String, Board>{};
  final boardCalls = <String>[];
  final claimed = <String>[];
  final rerolled = <String>[];
  MatchAwards matchAwards = const MatchAwards();

  @override
  Future<Progress> progress() async => current;

  @override
  Future<Standing> standing(String kind, String id) async => myStanding;

  @override
  Future<QuestList> quests() async => questList;

  @override
  Future<int> claim(String questId) async {
    claimed.add(questId);
    return 50;
  }

  @override
  Future<QuestList> reroll(String questId) async {
    rerolled.add(questId);
    return questList;
  }

  @override
  Future<List<BadgeInfo>> badges() async => catalogue;

  @override
  Future<Board> leaderboard({
    required BoardPeriod period,
    required BoardEntity entity,
    required BoardScope scope,
    String? categoryId,
    String? cursor,
  }) async {
    final key = '${period.wire}/${entity.name}/${scope.name}';
    boardCalls.add(key);
    return boards[key] ?? const Board(entries: []);
  }

  @override
  Future<MatchAwards> awards(String matchId) async => matchAwards;
}

class FakeDailyRepository implements DailyRepository {
  DailyToday todayValue = const DailyToday(
    day: '2026-10-20',
    available: true,
    practiceAllowed: false,
  );
  DailyBoard boardValue = const DailyBoard(entries: []);
  final started = <String>[];

  @override
  Future<DailyToday> today() async => todayValue;

  @override
  Future<DailyRun> start(String kind) async {
    started.add(kind);
    return DailyRun(id: 'run-$kind', kind: kind, status: 'in_progress');
  }

  @override
  Future<DailyBoard> board(String day, String scope) async => boardValue;
}

class FakeModerationRepository implements ModerationRepository {
  final reports = <String>[];
  final appeals = <String>[];
  List<Sanction> sanctions = const [];

  @override
  Future<void> reportQuestion({
    required String questionId,
    required String reason,
    String? matchId,
    String? comment,
  }) async => reports.add('$questionId:$reason:$matchId:${comment ?? ''}');

  @override
  Future<List<Sanction>> mySanctions() async => sanctions;

  @override
  Future<void> appeal(String sanctionId, String text) async =>
      appeals.add('$sanctionId:$text');
}

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
    this.social,
    this.lobby,
    this.teams,
    this.progress,
    this.daily,
    this.moderation,
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
    final social = FakeSocialRepository();
    final lobby = FakeLobbyRepository();
    final teams = FakeTeamRepository();
    final progress = FakeProgressRepository();
    final daily = FakeDailyRepository();
    final moderation = FakeModerationRepository();
    final sockets = <FakeSocket>[];
    final container = ProviderContainer(
      retry: noRetry,
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
        socialRepositoryProvider.overrideWithValue(social),
        lobbyRepositoryProvider.overrideWithValue(lobby),
        teamRepositoryProvider.overrideWithValue(teams),
        progressRepositoryProvider.overrideWithValue(progress),
        dailyRepositoryProvider.overrideWithValue(daily),
        moderationRepositoryProvider.overrideWithValue(moderation),
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
    return TestApp._(
      container,
      p,
      auth,
      sessions,
      profiles,
      matches,
      sockets,
      social,
      lobby,
      teams,
      progress,
      daily,
      moderation,
    );
  }

  final ProviderContainer container;
  final SharedPreferences prefs;
  final FakeMatchRepository matches;
  final FakeSocialRepository social;
  final FakeLobbyRepository lobby;
  final FakeTeamRepository teams;
  final FakeProgressRepository progress;
  final FakeDailyRepository daily;
  final FakeModerationRepository moderation;

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
